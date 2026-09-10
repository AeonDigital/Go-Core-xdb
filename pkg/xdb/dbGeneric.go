package xdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/AeonDigital/Go-Core-xerrors/pkg/xerrors"
)

// DBGeneric implements a type-safe, generic repository pattern dedicated to a specific domain entity model.
type DBGeneric[T any, PT interface {
	*T
	Entity
}] struct {
	db                     *sql.DB
	tx                     *sql.Tx
	executor               SQLExecutor
	idempotentUpdateActive bool
	idempotentDeleteActive bool
}

// NewDBGeneric initializes and yields a new operational instance of the generic repository interface.
func NewDBGeneric[T any, PT interface {
	*T
	Entity
}](db *sql.DB) *DBGeneric[T, PT] {
	return &DBGeneric[T, PT]{
		db:       db,
		executor: db,
	}
}

// WithTx returns a contextual shallow clone of the repository bound to an active database transaction lifecycle.
func (r *DBGeneric[T, PT]) WithTx(tx *sql.Tx) *DBGeneric[T, PT] {
	return &DBGeneric[T, PT]{
		db:                     r.db,
		tx:                     tx,
		executor:               tx,
		idempotentUpdateActive: r.idempotentUpdateActive,
		idempotentDeleteActive: r.idempotentDeleteActive,
	}
}

// GetSQLExecutor evaluates whether to pipeline execution states through an active transaction isolation or the global connection pool.
func (r *DBGeneric[T, PT]) GetSQLExecutor() SQLExecutor {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// QueryRaw coordinates the isolation, manual execution, and custom collection scan mapping of arbitrary database commands.
func QueryRaw[R any](
	ctx context.Context,
	db *sql.DB,
	cq CustomQuery[R],
) ([]R, xerrors.ErrorCode) {
	rows, err := db.QueryContext(ctx, cq.SQL, cq.Args...)
	if err != nil {
		logRepoError(ctx, db, XERR_REPO_QUERY_RAW_EXEC_FAILED, err, cq.SQL, cq.Args)
		return nil, XERR_REPO_QUERY_RAW_EXEC_FAILED
	}
	defer rows.Close()

	var result []R
	for rows.Next() {
		item, err := cq.Scanner(rows)
		if err != nil {
			logRepoError(ctx, db, XERR_REPO_QUERY_RAW_SCAN_FAILED, err, cq.SQL, cq.Args)
			return nil, XERR_REPO_QUERY_RAW_SCAN_FAILED
		}
		result = append(result, item)
	}

	if err = rows.Err(); err != nil {
		logRepoError(ctx, db, XERR_REPO_GET_ALL_ITERATION_FAILED, err, cq.SQL, cq.Args)
		return nil, XERR_REPO_GET_ALL_ITERATION_FAILED
	}

	return result, XERR_NONE
}

// SetIdempotentUpdate overrides instance update settings to prevent throwing missing record validation errors on missing datasets.
func (r *DBGeneric[T, PT]) SetIdempotentUpdate(enabled bool) *DBGeneric[T, PT] {
	r.idempotentUpdateActive = enabled
	return r
}

// SetIdempotentDelete overrides instance delete settings to silently tolerate non-existent entities during removal attempts.
func (r *DBGeneric[T, PT]) SetIdempotentDelete(enabled bool) *DBGeneric[T, PT] {
	r.idempotentDeleteActive = enabled
	return r
}

// shouldBeIdempotent calculates the situational hierarchy between active context variables and historical instance flags.
func (r *DBGeneric[T, PT]) shouldBeIdempotent(ctx context.Context, isUpdate bool) bool {
	if prohibit, _ := ctx.Value(prohibitIdempotencyKey).(bool); prohibit {
		return false
	}

	if force, _ := ctx.Value(forceIdempotencyKey).(bool); force {
		return true
	}

	if isUpdate {
		return r.idempotentUpdateActive
	}
	return r.idempotentDeleteActive
}

// Insert validates model states, produces identifiers when applicable, and stores records securely.
func (r *DBGeneric[T, PT]) Insert(ctx context.Context, entity PT) xerrors.ErrorCode {
	currentPK := entity.PKGetValue()

	switch v := currentPK.(type) {
	case int64:
		if v > 0 {
			logRepoError(ctx, r.db, XERR_REPO_INSERT_WITH_UNEXPECTED_NUMERIC_PK, nil, "", nil)
			return XERR_REPO_INSERT_WITH_UNEXPECTED_NUMERIC_PK
		}
	case string:
		if !entity.PKExternal() && strings.TrimSpace(v) != "" {
			logRepoError(ctx, r.db, XERR_REPO_INSERT_WITH_UNEXPECTED_STRING_PK, nil, "", nil)
			return XERR_REPO_INSERT_WITH_UNEXPECTED_STRING_PK
		}
		if entity.PKExternal() && strings.TrimSpace(v) == "" {
			logRepoError(ctx, r.db, XERR_REPO_INSERT_WITH_UNEXPECTED_EMPTY_PK, nil, "", nil)
			return XERR_REPO_INSERT_WITH_UNEXPECTED_EMPTY_PK
		}
	case nil:
		if entity.PKExternal() {
			logRepoError(ctx, r.db, XERR_REPO_INSERT_WITH_UNEXPECTED_NIL_PK, nil, "", nil)
			return XERR_REPO_INSERT_WITH_UNEXPECTED_NIL_PK
		}
	}

	generated := entity.PKGenerateValue()
	if generated != nil {
		entity.PKSetValue(generated)
	}

	entity.Normalize()
	if valid, errCode := entity.Validate(); !valid {
		logRepoError(ctx, r.db, errCode, nil, "", nil)
		return errCode
	}

	cols := entity.Columns()
	values := entity.Values()

	placeholders := make([]string, len(cols))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s);",
		entity.TableName(),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	executor := r.GetSQLExecutor()
	result, err := executor.ExecContext(ctx, query, values...)
	if err != nil {
		logRepoError(ctx, r.db, XERR_REPO_INSERT_EXECUTION_FAILED, err, query, values)
		return XERR_REPO_INSERT_EXECUTION_FAILED
	}

	if entity.PKColumnName() == "id" {
		currentVal := entity.PKGetValue()

		isUnset := currentVal == nil
		if !isUnset {
			pkInt, ok := currentVal.(int64)
			if pkInt == 0 && ok {
				isUnset = true
			}
		}

		if isUnset {
			id, err := result.LastInsertId()
			if id > 0 && err == nil {
				entity.PKSetValue(id)
			} else {
				logRepoError(ctx, r.db, XERR_REPO_INSERT_FETCH_ID_FAILED, err, query, values)
			}
		}
	}

	return XERR_NONE
}

// Update coordinates column updates while enforcing primary key validations and idempotency rules.
func (r *DBGeneric[T, PT]) Update(ctx context.Context, entity PT) xerrors.ErrorCode {
	currentPK := entity.PKGetValue()

	switch v := currentPK.(type) {
	case int64:
		if v <= 0 {
			logRepoError(ctx, r.db, XERR_REPO_UPDATE_INVALID_NUMERICAL_PK, nil, "", nil)
			return XERR_REPO_UPDATE_INVALID_NUMERICAL_PK
		}
	case string:
		if strings.TrimSpace(v) == "" {
			logRepoError(ctx, r.db, XERR_REPO_UPDATE_EMPTY_STRING_PK, nil, "", nil)
			return XERR_REPO_UPDATE_EMPTY_STRING_PK
		}
	case nil:
		logRepoError(ctx, r.db, XERR_REPO_UPDATE_PK_NIL, nil, "", nil)
		return XERR_REPO_UPDATE_PK_NIL
	default:
		logRepoError(ctx, r.db, XERR_REPO_UPDATE_UNKNOWN_PK_TYPE, nil, "", nil)
		return XERR_REPO_UPDATE_UNKNOWN_PK_TYPE
	}

	entity.Normalize()

	if valid, errCode := entity.Validate(); !valid {
		logRepoError(ctx, r.db, errCode, nil, "", nil)
		return errCode
	}

	cols := entity.Columns()
	values := entity.Values()

	if len(cols) == 0 {
		logRepoError(ctx, r.db, XERR_REPO_UPDATE_NO_COLUMNS_DEFINED, nil, "", nil)
		return XERR_REPO_UPDATE_NO_COLUMNS_DEFINED
	}

	setFragments := make([]string, len(cols))
	for i, col := range cols {
		setFragments[i] = fmt.Sprintf("%s = ?", col)
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = ?;",
		entity.TableName(),
		strings.Join(setFragments, ", "),
		entity.PKColumnName(),
	)

	args := append(values, entity.PKGetValue())

	result, err := r.executor.ExecContext(ctx, query, args...)
	if err != nil {
		logRepoError(ctx, r.db, XERR_REPO_UPDATE_EXEC_FAILED, err, query, args)
		return XERR_REPO_UPDATE_EXEC_FAILED
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logRepoError(ctx, r.db, XERR_REPO_UPDATE_VERIFY_ROWS_FAILED, err, query, args)
		return XERR_REPO_UPDATE_VERIFY_ROWS_FAILED
	}

	if rowsAffected == 0 {
		if r.shouldBeIdempotent(ctx, true) {
			return XERR_NONE
		}

		logRepoError(ctx, r.db, XERR_REPO_UPDATE_RECORD_NOT_FOUND, nil, query, args)
		return XERR_REPO_UPDATE_RECORD_NOT_FOUND
	}

	return XERR_NONE
}

// Delete drops target records based on explicit key mapping evaluations.
func (r *DBGeneric[T, PT]) Delete(ctx context.Context, entity PT) xerrors.ErrorCode {
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s = ?;",
		entity.TableName(),
		entity.PKColumnName(),
	)

	pkValue := entity.PKGetValue()
	args := []any{pkValue}

	result, err := r.executor.ExecContext(ctx, query, pkValue)
	if err != nil {
		logRepoError(ctx, r.db, XERR_REPO_DELETE_EXEC_FAILED, err, query, args)
		return XERR_REPO_DELETE_EXEC_FAILED
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logRepoError(ctx, r.db, XERR_REPO_DELETE_VERIFY_ROWS_FAILED, err, query, args)
		return XERR_REPO_DELETE_VERIFY_ROWS_FAILED
	}

	if rowsAffected == 0 {
		if r.shouldBeIdempotent(ctx, false) {
			return XERR_NONE
		}

		logRepoError(ctx, r.db, XERR_REPO_DELETE_RECORD_NOT_FOUND, nil, query, args)
		return XERR_REPO_DELETE_RECORD_NOT_FOUND
	}

	return XERR_NONE
}

// GetByID performs a target row execution based on primary key mappings to fetch a singular type-safe entry instance.
func (r *DBGeneric[T, PT]) GetByID(ctx context.Context, id any) (*T, xerrors.ErrorCode) {
	var instance PT = new(T)

	query := fmt.Sprintf(
		"SELECT * FROM %s WHERE %s = ? LIMIT 1;",
		instance.TableName(),
		instance.PKColumnName(),
	)

	args := []any{id}

	rows, err := r.executor.QueryContext(ctx, query, id)
	if err != nil {
		logRepoError(ctx, r.db, XERR_REPO_GET_BY_ID_EXEC_FAILED, err, query, args)
		return nil, XERR_REPO_GET_BY_ID_EXEC_FAILED
	}
	defer rows.Close()

	if !rows.Next() {
		logRepoError(ctx, r.db, XERR_REPO_GET_BY_ID_RECORD_NOT_FOUND, nil, query, args)
		return nil, XERR_REPO_GET_BY_ID_RECORD_NOT_FOUND
	}

	if err := instance.ScanRow(rows); err != nil {
		logRepoError(ctx, r.db, XERR_REPO_GET_BY_ID_SCAN_FAILED, err, query, args)
		return nil, XERR_REPO_GET_BY_ID_SCAN_FAILED
	}

	return instance, XERR_NONE
}

// GetAll extracts every existing collection sequence context from the entity schema targets.
func (r *DBGeneric[T, PT]) GetAll(ctx context.Context) ([]*T, xerrors.ErrorCode) {
	var meta PT = new(T)
	query := fmt.Sprintf(
		"SELECT * FROM %s;",
		meta.TableName(),
	)

	rows, err := r.executor.QueryContext(ctx, query)
	if err != nil {
		logRepoError(ctx, r.db, XERR_REPO_GET_ALL_EXEC_FAILED, err, query, nil)
		return nil, XERR_REPO_GET_ALL_EXEC_FAILED
	}
	defer rows.Close()

	var list []*T
	for rows.Next() {
		var item PT = new(T)

		if err := item.ScanRow(rows); err != nil {
			logRepoError(ctx, r.db, XERR_REPO_GET_ALL_SCAN_FAILED, err, query, nil)
			return nil, XERR_REPO_GET_ALL_SCAN_FAILED
		}

		list = append(list, item)
	}

	if err = rows.Err(); err != nil {
		logRepoError(ctx, r.db, XERR_REPO_GET_ALL_ITERATION_FAILED, err, query, nil)
		return nil, XERR_REPO_GET_ALL_ITERATION_FAILED
	}

	return list, XERR_NONE
}

// GetByField searches for matching dataset groups filtered by a specific column variable signature.
func (r *DBGeneric[T, PT]) GetByField(ctx context.Context, field string, value any) ([]*T, xerrors.ErrorCode) {
	var meta PT = new(T)

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?;", meta.TableName(), field)
	args := []any{value}

	rows, err := r.executor.QueryContext(ctx, query, value)
	if err != nil {
		logRepoError(ctx, r.db, XERR_REPO_GET_BY_FIELD_EXEC_FAILED, err, query, args)
		return nil, XERR_REPO_GET_BY_FIELD_EXEC_FAILED
	}
	defer rows.Close()

	var list []*T
	for rows.Next() {
		var item PT = new(T)
		if err := item.ScanRow(rows); err != nil {
			logRepoError(ctx, r.db, XERR_REPO_GET_BY_FIELD_SCAN_FAILED, err, query, args)
			return nil, XERR_REPO_GET_BY_FIELD_SCAN_FAILED
		}
		list = append(list, item)
	}

	if err = rows.Err(); err != nil {
		logRepoError(ctx, r.db, XERR_REPO_GET_ALL_ITERATION_FAILED, err, query, args)
		return nil, XERR_REPO_GET_ALL_ITERATION_FAILED
	}

	return list, XERR_NONE
}

// GetWhere parses complex conditional dynamic parameters to retrieve subset target collections.
func (r *DBGeneric[T, PT]) GetWhere(ctx context.Context, queryFragment string, args ...any) ([]*T, xerrors.ErrorCode) {
	placeholderCount := strings.Count(queryFragment, "?")
	argCount := len(args)

	if placeholderCount != argCount {
		logRepoError(ctx, r.db, XERR_REPO_GET_WHERE_ARGS_MISMATCH, nil, queryFragment, args)
		return nil, XERR_REPO_GET_WHERE_ARGS_MISMATCH
	}

	var meta PT = new(T)
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s;", meta.TableName(), queryFragment)

	rows, err := r.executor.QueryContext(ctx, query, args...)
	if err != nil {
		logRepoError(ctx, r.db, XERR_REPO_GET_WHERE_EXEC_FAILED, err, query, args)
		return nil, XERR_REPO_GET_WHERE_EXEC_FAILED
	}
	defer rows.Close()

	var list []*T
	for rows.Next() {
		var item PT = new(T)
		if err := item.ScanRow(rows); err != nil {
			logRepoError(ctx, r.db, XERR_REPO_GET_WHERE_SCAN_FAILED, err, query, args)
			return nil, XERR_REPO_GET_WHERE_SCAN_FAILED
		}
		list = append(list, item)
	}

	if err = rows.Err(); err != nil {
		logRepoError(ctx, r.db, XERR_REPO_GET_ALL_ITERATION_FAILED, err, query, args)
		return nil, XERR_REPO_GET_ALL_ITERATION_FAILED
	}

	return list, XERR_NONE
}

// Count returns the number of records matching the provided primary key.
func (r *DBGeneric[T, PT]) Count(ctx context.Context, id any) (int, xerrors.ErrorCode) {
	var instance PT = new(T)

	query := fmt.Sprintf(
		"SELECT COUNT(*) FROM %s WHERE %s = ?;",
		instance.TableName(),
		instance.PKColumnName(),
	)

	args := []any{id}
	rows, err := r.executor.QueryContext(ctx, query, id)
	if err != nil {
		logRepoError(ctx, r.db, XERR_REPO_COUNT_EXEC_FAILED, err, query, args)
		return 0, XERR_REPO_COUNT_EXEC_FAILED
	}
	defer rows.Close()

	if !rows.Next() {
		logRepoError(ctx, r.db, XERR_REPO_COUNT_SCAN_FAILED, rows.Err(), query, args)
		return 0, XERR_REPO_COUNT_SCAN_FAILED
	}

	var count int
	if err := rows.Scan(&count); err != nil {
		logRepoError(ctx, r.db, XERR_REPO_COUNT_SCAN_FAILED, err, query, args)
		return 0, XERR_REPO_COUNT_SCAN_FAILED
	}

	return count, XERR_NONE
}

// CountWhere returns the number of records matching the provided condition.
func (r *DBGeneric[T, PT]) CountWhere(ctx context.Context, queryFragment string, args ...any) (int, xerrors.ErrorCode) {
	placeholderCount := strings.Count(queryFragment, "?")
	argCount := len(args)

	if placeholderCount != argCount {
		logRepoError(ctx, r.db, XERR_REPO_COUNT_WHERE_ARGS_MISMATCH, nil, queryFragment, args)
		return 0, XERR_REPO_COUNT_WHERE_ARGS_MISMATCH
	}

	var meta PT = new(T)
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s;", meta.TableName(), queryFragment)

	rows, err := r.executor.QueryContext(ctx, query, args...)
	if err != nil {
		logRepoError(ctx, r.db, XERR_REPO_COUNT_WHERE_EXEC_FAILED, err, query, args)
		return 0, XERR_REPO_COUNT_WHERE_EXEC_FAILED
	}
	defer rows.Close()

	if !rows.Next() {
		logRepoError(ctx, r.db, XERR_REPO_COUNT_WHERE_SCAN_FAILED, rows.Err(), query, args)
		return 0, XERR_REPO_COUNT_WHERE_SCAN_FAILED
	}

	var count int
	if err := rows.Scan(&count); err != nil {
		logRepoError(ctx, r.db, XERR_REPO_COUNT_WHERE_SCAN_FAILED, err, query, args)
		return 0, XERR_REPO_COUNT_WHERE_SCAN_FAILED
	}

	return count, XERR_NONE
}
