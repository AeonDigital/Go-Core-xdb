package xdb_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"testing"

	"github.com/AeonDigital/Go-Core-xdb/pkg/xdb"
	_ "modernc.org/sqlite"
)

// ============================================================================
// Mock Structures & Helpers
// ============================================================================

// mockResultRowsError simulates a database result that returns an error
// when retrieving the number of affected rows.
type mockResultRowsError struct{}

func (m mockResultRowsError) LastInsertId() (int64, error) { return 0, nil }
func (m mockResultRowsError) RowsAffected() (int64, error) {
	return 0, fmt.Errorf("simulated driver failure on rows affected")
}

// mockDriverConnector implements the driver.Connector interface to initialize OpenDB.
type mockDriverConnector struct{}

func (mockDriverConnector) Connect(context.Context) (driver.Conn, error) {
	return mockDriverConn{}, nil
}
func (mockDriverConnector) Driver() driver.Driver { return nil }

// mockExecutorRowsError simulates a database executor that triggers errors
// during query context execution based on specific arguments.
type mockExecutorRowsError struct{}

func (m mockExecutorRowsError) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return mockResultRowsError{}, nil
}

func (m mockExecutorRowsError) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if len(args) > 0 && (args[0] == "FAIL_GETWHERE" || args[0] == "FAIL_GETBYFIELD") {
		fakeDb := sql.OpenDB(mockDriverConnector{})
		return fakeDb.QueryContext(ctx, query, args...)
	}

	return nil, nil
}

func (m mockExecutorRowsError) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return nil
}

// mockRowsWithError implements the driver.Rows interface to force an unexpected EOF error.
type mockRowsWithError struct{}

func (m *mockRowsWithError) Columns() []string { return []string{"id", "uuid", "name", "email"} }
func (m *mockRowsWithError) Close() error      { return nil }
func (m *mockRowsWithError) Next(dest []driver.Value) error {
	return io.ErrUnexpectedEOF
}

// mockDriverConn represents a fake database connection required to deliver mock rows.
type mockDriverConn struct{ driver.Conn }

func (m mockDriverConn) Prepare(query string) (driver.Stmt, error) { return mockDriverStmt{}, nil }
func (m mockDriverConn) Close() error                              { return nil }
func (m mockDriverConn) Begin() (driver.Tx, error)                 { return nil, nil }

// mockDriverStmt represents a fake database statement for executing mock queries.
type mockDriverStmt struct{ driver.Stmt }

func (m mockDriverStmt) Close() error                                    { return nil }
func (m mockDriverStmt) NumInput() int                                   { return -1 }
func (m mockDriverStmt) Exec(args []driver.Value) (driver.Result, error) { return nil, nil }
func (m mockDriverStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &mockRowsWithError{}, nil
}

// mockExecutorGetAllIterError simulates an executor designed to fail during iteration tests.
type mockExecutorGetAllIterError struct{}

func (m mockExecutorGetAllIterError) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}

func (m mockExecutorGetAllIterError) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	fakeDb := sql.OpenDB(mockDriverConnector{})
	return fakeDb.QueryContext(ctx, query, args...)
}

func (m mockExecutorGetAllIterError) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return nil
}

type mockCountScanConnector struct{}

func (mockCountScanConnector) Connect(context.Context) (driver.Conn, error) {
	return mockCountScanConn{}, nil
}
func (mockCountScanConnector) Driver() driver.Driver { return nil }

type mockCountScanConn struct{}

func (mockCountScanConn) Prepare(query string) (driver.Stmt, error) {
	return mockCountScanStmt{}, nil
}
func (mockCountScanConn) Close() error              { return nil }
func (mockCountScanConn) Begin() (driver.Tx, error) { return nil, nil }

type mockCountScanStmt struct{}

func (mockCountScanStmt) Close() error                                    { return nil }
func (mockCountScanStmt) NumInput() int                                   { return -1 }
func (mockCountScanStmt) Exec(args []driver.Value) (driver.Result, error) { return nil, nil }
func (mockCountScanStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &mockCountScanRows{}, nil
}

type mockCountScanRows struct {
	read bool
}

func (m *mockCountScanRows) Columns() []string { return []string{"count"} }
func (m *mockCountScanRows) Close() error      { return nil }
func (m *mockCountScanRows) Next(dest []driver.Value) error {
	if m.read {
		return io.EOF
	}
	m.read = true
	dest[0] = "not-a-number"
	return nil
}

type mockExecutorCountScanError struct{}

func (mockExecutorCountScanError) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}

func (mockExecutorCountScanError) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	fakeDb := sql.OpenDB(mockCountScanConnector{})
	return fakeDb.QueryContext(ctx, query, args...)
}

func (mockExecutorCountScanError) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return nil
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Abre a conexão explicitamente usando o driver "sqlite" da modernc.org
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Forçamos o PRAGMA de chaves estrangeiras apenas para alinhar com o comportamento real
	_, _ = db.Exec("PRAGMA foreign_keys = ON;")

	ddl := `
	CREATE TABLE mock_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid TEXT,
		name TEXT,
		email TEXT
	);`

	if _, err := db.Exec(ddl); err != nil {
		db.Close()
		t.Fatalf("failed to create mock table: %v", err)
	}

	return db
}

// ============================================================================
// Tests
// ============================================================================

func TestDBGeneric_Transaction_Lifecycle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	globalRepo := xdb.NewDBGeneric[MockUser](db)

	t.Run("Success insert utilizing active transaction and forced rollback", func(t *testing.T) {
		// 1. Iniciamos uma transação real no banco de dados
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("failed to begin transaction: %v", err)
		}

		// 2. Criamos o clone contextual do repositório usando o WithTx
		// Isso força a propriedade r.tx a ficar preenchida, ativando o fluxo "return r.tx"
		txRepo := globalRepo.WithTx(tx)

		user := &MockUser{
			Name:  "Tx User",
			Email: "tx@example.com",
		}

		// 3. Executamos o Insert através do repositório transacional
		errCode := txRepo.Insert(ctx, user)
		if errCode != xdb.XERR_NONE {
			tx.Rollback()
			t.Fatalf("expected XERR_NONE, got %s", errCode)
		}

		// 4. Cancelamos a transação explicitamente (Rollback)
		if err := tx.Rollback(); err != nil {
			t.Fatalf("failed to rollback transaction: %v", err)
		}

		// 5. PROVA DE ISOLAMENTO: Buscamos o ID no repositório global (fora da transação)
		// Como houve Rollback, o registro DEVE ter sumido e retornado RecordNotFound (E0020)
		_, fetchErr := globalRepo.GetByID(ctx, user.ID)
		if fetchErr != xdb.XERR_REPO_GET_BY_ID_RECORD_NOT_FOUND {
			t.Errorf("expected record to be rolled back, but found code: %s", fetchErr)
		}
	})
}

func TestQueryRaw(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Inserimos dados de teste usando SQL puro
	_, _ = db.Exec("INSERT INTO mock_users (name, email) VALUES (?, ?);", "Raw User", "raw@example.com")

	// Criamos uma struct local simples apenas para provar que o QueryRaw aceita qualquer tipo (R)
	type CustomResult struct {
		NameSnapshot string
	}

	// Scanner de sucesso
	successScanner := func(rows *sql.Rows) (CustomResult, error) {
		var res CustomResult
		var dummyID int64
		var dummyUUID, dummyEmail any
		err := rows.Scan(&dummyID, &dummyUUID, &res.NameSnapshot, &dummyEmail)
		return res, err
	}

	t.Run("Success raw query execution and custom mapping", func(t *testing.T) {
		cq := xdb.CustomQuery[CustomResult]{
			SQL:     "SELECT * FROM mock_users WHERE name = ?;",
			Args:    []any{"Raw User"},
			Scanner: successScanner,
		}

		result, errCode := xdb.QueryRaw(ctx, db, cq)
		if errCode != xdb.XERR_NONE {
			t.Errorf("expected XERR_NONE, got %s", errCode)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 record, got %d", len(result))
		}
		if result[0].NameSnapshot != "Raw User" {
			t.Errorf("expected 'Raw User', got %s", result[0].NameSnapshot)
		}
	})

	t.Run("Fail when raw database execution crashes", func(t *testing.T) {
		deadDb, _ := sql.Open("sqlite", ":memory:")
		deadDb.Close()

		cq := xdb.CustomQuery[CustomResult]{
			SQL:     "SELECT * FROM mock_users;",
			Scanner: successScanner,
		}

		_, errCode := xdb.QueryRaw(ctx, deadDb, cq)
		if errCode != xdb.XERR_REPO_QUERY_RAW_EXEC_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_QUERY_RAW_EXEC_FAILED, errCode)
		}
	})

	t.Run("Fail when custom row scanner returns an error", func(t *testing.T) {
		// Criamos um scanner que força um erro de mapeamento de propósito
		failScanner := func(rows *sql.Rows) (CustomResult, error) {
			return CustomResult{}, fmt.Errorf("simulated custom scanner failure")
		}

		cq := xdb.CustomQuery[CustomResult]{
			SQL:     "SELECT * FROM mock_users;",
			Scanner: failScanner,
		}

		_, errCode := xdb.QueryRaw(ctx, db, cq)
		if errCode != xdb.XERR_REPO_QUERY_RAW_SCAN_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_QUERY_RAW_SCAN_FAILED, errCode)
		}
	})

	t.Run("Fail when raw query cursor iteration fails", func(t *testing.T) {
		// Fabricamos uma conexão *sql.DB legítima usando o nosso driver falso sabotador
		fakeDb := sql.OpenDB(mockDriverConnector{})
		defer fakeDb.Close()

		cq := xdb.CustomQuery[CustomResult]{
			SQL:     "SELECT * FROM mock_users;",
			Scanner: successScanner,
		}

		_, errCode := xdb.QueryRaw(ctx, fakeDb, cq)
		if errCode != xdb.XERR_REPO_GET_ALL_ITERATION_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_GET_ALL_ITERATION_FAILED, errCode)
		}
	})
}

func TestDBGeneric_Insert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := xdb.NewDBGeneric[MockUser](db)

	t.Run("Success with DB auto-increment ID", func(t *testing.T) {
		user := &MockUser{
			Name:  "John Doe",
			Email: "john@example.com",
		}

		errCode := repo.Insert(ctx, user)
		if errCode != xdb.XERR_NONE {
			t.Errorf("expected XERR_NONE, got %s", errCode)
		}

		if user.ID != 1 {
			t.Errorf("expected injected ID to be 1, got %d", user.ID)
		}
	})

	t.Run("Validation barrier failure", func(t *testing.T) {
		invalidUser := &MockUser{
			Name:  "INVALID",
			Email: "invalid@example.com",
		}

		errCode := repo.Insert(ctx, invalidUser)
		if errCode != "E9999" {
			t.Errorf("expected validation code E9999, got %s", errCode)
		}
	})

	t.Run("Normalization triggered before insert", func(t *testing.T) {
		user := &MockUser{
			Name:  "  Trim Me  ",
			Email: "trim@example.com",
		}

		errCode := repo.Insert(ctx, user)
		if errCode != xdb.XERR_NONE {
			t.Errorf("expected XERR_NONE, got %s", errCode)
		}

		if user.Name != "Trim Me" {
			t.Errorf("expected name to be normalized to 'Trim Me', got '%s'", user.Name)
		}
	})

	t.Run("Fail when entity has an existing numerical PK", func(t *testing.T) {
		user := &MockUser{
			ID:    99,
			Name:  "Jane Doe",
			Email: "jane@example.com",
		}

		errCode := repo.Insert(ctx, user)
		if errCode != xdb.XERR_REPO_INSERT_WITH_UNEXPECTED_NUMERIC_PK {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_INSERT_WITH_UNEXPECTED_NUMERIC_PK, errCode)
		}
	})

	t.Run("Fail when surrogate string entity already has a key", func(t *testing.T) {
		user := &MockUser{
			UUID:      "existing-uuid-abc",
			IsNatural: false,
			Name:      "Alice",
			Email:     "alice@example.com",
		}

		errCode := repo.Insert(ctx, user)
		if errCode != xdb.XERR_REPO_INSERT_WITH_UNEXPECTED_STRING_PK {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_INSERT_WITH_UNEXPECTED_STRING_PK, errCode)
		}
	})

	t.Run("Fail when natural PK is an empty string", func(t *testing.T) {
		user := &MockUser{
			UUID:      "   ",
			IsNatural: true,
			Name:      "Bob",
			Email:     "bob@example.com",
		}

		errCode := repo.Insert(ctx, user)
		if errCode != xdb.XERR_REPO_INSERT_WITH_UNEXPECTED_EMPTY_PK {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_INSERT_WITH_UNEXPECTED_EMPTY_PK, errCode)
		}
	})

	t.Run("Fail when natural PK is nil", func(t *testing.T) {
		user := &MockUser{
			UUID:      "",
			IsNatural: true,
			Name:      "Charlie",
			Email:     "charlie@example.com",
		}

		errCode := repo.Insert(ctx, user)
		if errCode != xdb.XERR_REPO_INSERT_WITH_UNEXPECTED_NIL_PK {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_INSERT_WITH_UNEXPECTED_NIL_PK, errCode)
		}
	})

	t.Run("Success with application-side generated PK", func(t *testing.T) {
		user := &MockUser{
			UUID:  "",
			Name:  "App Generated User",
			Email: "app@example.com",
		}

		errCode := repo.Insert(ctx, user)
		if errCode != xdb.XERR_NONE {
			t.Fatalf("expected XERR_NONE, got %s", errCode)
		}

		if user.UUID != "GENERATED-UUID-123" {
			t.Errorf("expected application to bind generated PK, got %s", user.UUID)
		}
	})

	t.Run("Fail when database execution crashes", func(t *testing.T) {
		deadDb, _ := sql.Open("sqlite", ":memory:")
		deadDb.Close()
		deadRepo := xdb.NewDBGeneric[MockUser](deadDb)

		user := &MockUser{
			Name:  "Ghost User",
			Email: "ghost@example.com",
		}

		errCode := deadRepo.Insert(ctx, user)
		if errCode != xdb.XERR_REPO_INSERT_EXECUTION_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_INSERT_EXECUTION_FAILED, errCode)
		}
	})

	t.Run("Success when numerical PK is explicitly zero type-asserted", func(t *testing.T) {
		user := &MockUser{
			ID:    0,
			Name:  "Explicit Zero User",
			Email: "zero@example.com",
		}

		errCode := repo.Insert(ctx, user)
		if errCode != xdb.XERR_NONE {
			t.Errorf("expected XERR_NONE, got %s", errCode)
		}

		if user.ID != 2 {
			if user.ID == 0 {
				t.Errorf("expected ID to be updated from database, got 0")
			}
		}
	})
}

func TestDBGeneric_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := xdb.NewDBGeneric[MockUser](db)

	existingUser := &MockUser{Name: "Original Name", Email: "orig@example.com"}
	_ = repo.Insert(ctx, existingUser)

	t.Run("Success standard update", func(t *testing.T) {
		existingUser.Name = "Updated Name"

		errCode := repo.Update(ctx, existingUser)
		if errCode != xdb.XERR_NONE {
			t.Errorf("expected XERR_NONE, got %s", errCode)
		}

		fetched, _ := repo.GetByID(ctx, existingUser.ID)
		if fetched.Name != "Updated Name" {
			t.Errorf("expected name to be updated in database, got %s", fetched.Name)
		}
	})

	t.Run("Fail when numerical PK is invalid", func(t *testing.T) {
		invalidUser := &MockUser{
			ID:    0,
			Name:  "No ID",
			Email: "noid@example.com",
		}

		errCode := repo.Update(ctx, invalidUser)
		if errCode != xdb.XERR_REPO_UPDATE_INVALID_NUMERICAL_PK {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_UPDATE_INVALID_NUMERICAL_PK, errCode)
		}
	})

	t.Run("Fail when string PK is empty", func(t *testing.T) {
		invalidUser := &MockUser{
			UUID:  "   ",
			Name:  "Empty String PK",
			Email: "empty@example.com",
		}

		errCode := repo.Update(ctx, invalidUser)
		if errCode != xdb.XERR_REPO_UPDATE_EMPTY_STRING_PK {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_UPDATE_EMPTY_STRING_PK, errCode)
		}
	})

	t.Run("Fail when PK is nil", func(t *testing.T) {
		invalidUser := &MockUser{
			Name:  "Nil PK User",
			Email: "nilpk@example.com",
		}

		errCode := repo.Update(ctx, invalidUser)
		if errCode != xdb.XERR_REPO_UPDATE_PK_NIL {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_UPDATE_PK_NIL, errCode)
		}
	})

	t.Run("Fail when PK type is unknown", func(t *testing.T) {
		invalidUser := &MockUser{
			Name:  "Unknown PK Type",
			Email: "unknown@example.com",
		}
		invalidUser.Name = "FORCE_UNKNOWN_PK_TYPE"

		errCode := repo.Update(ctx, invalidUser)
		if errCode != xdb.XERR_REPO_UPDATE_UNKNOWN_PK_TYPE {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_UPDATE_UNKNOWN_PK_TYPE, errCode)
		}
	})

	t.Run("Fail when validation barrier fails", func(t *testing.T) {
		existingUser.Name = "INVALID"

		errCode := repo.Update(ctx, existingUser)
		if errCode != "E9999" {
			t.Errorf("expected E9999, got %s", errCode)
		}

		existingUser.Name = "Updated Name"
	})

	t.Run("Fail when columns defined is empty", func(t *testing.T) {
		user := &MockUser{
			ID:    1,
			Name:  "FORCE_EMPTY_COLUMNS",
			Email: "emptycols@example.com",
		}

		errCode := repo.Update(ctx, user)
		if errCode != xdb.XERR_REPO_UPDATE_NO_COLUMNS_DEFINED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_UPDATE_NO_COLUMNS_DEFINED, errCode)
		}
	})

	t.Run("Fail when database execution crashes", func(t *testing.T) {
		deadDb, _ := sql.Open("sqlite", ":memory:")
		deadDb.Close()
		deadRepo := xdb.NewDBGeneric[MockUser](deadDb)

		user := &MockUser{
			ID:    1,
			Name:  "Ghost User",
			Email: "ghost@example.com",
		}

		errCode := deadRepo.Update(ctx, user)
		if errCode != xdb.XERR_REPO_UPDATE_EXEC_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_UPDATE_EXEC_FAILED, errCode)
		}
	})

	t.Run("Fail record not found without idempotency", func(t *testing.T) {
		notFoundUser := &MockUser{
			ID:    999,
			Name:  "Ghost",
			Email: "ghost@example.com",
		}

		errCode := repo.Update(ctx, notFoundUser)
		if errCode != xdb.XERR_REPO_UPDATE_RECORD_NOT_FOUND {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_UPDATE_RECORD_NOT_FOUND, errCode)
		}
	})

	t.Run("Success record not found with instance idempotency", func(t *testing.T) {
		notFoundUser := &MockUser{
			ID:    888,
			Name:  "Idempotent Ghost",
			Email: "ghost@example.com",
		}

		repo.SetIdempotentUpdate(true)
		defer repo.SetIdempotentUpdate(false)

		errCode := repo.Update(ctx, notFoundUser)
		if errCode != xdb.XERR_NONE {
			t.Errorf("expected XERR_NONE due to idempotency, got %s", errCode)
		}
	})

	t.Run("Success record not found with forced context idempotency", func(t *testing.T) {
		notFoundUser := &MockUser{
			ID:    777,
			Name:  "Context Ghost",
			Email: "ghost@example.com",
		}

		forcedCtx := xdb.ContextWithForcedIdempotency(ctx)

		errCode := repo.Update(forcedCtx, notFoundUser)
		if errCode != xdb.XERR_NONE {
			t.Errorf("expected XERR_NONE due to forced context, got %s", errCode)
		}
	})

	t.Run("Fail record not found when context prohibits idempotency", func(t *testing.T) {
		notFoundUser := &MockUser{
			ID:    666,
			Name:  "Prohibited Ghost",
			Email: "ghost@example.com",
		}

		repo.SetIdempotentUpdate(true)
		defer repo.SetIdempotentUpdate(false)

		prohibitedCtx := xdb.ContextWithProhibitedIdempotency(ctx)

		errCode := repo.Update(prohibitedCtx, notFoundUser)
		if errCode != xdb.XERR_REPO_UPDATE_RECORD_NOT_FOUND {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_UPDATE_RECORD_NOT_FOUND, errCode)
		}
	})

	t.Run("Fail when rows affected verification fails", func(t *testing.T) {
		repoRowsFail := xdb.NewDBGeneric[MockUser](db)

		repoRowsFail.SetExecutorForTest(mockExecutorRowsError{})

		user := &MockUser{
			ID:    1,
			Name:  "Test Rows Fail",
			Email: "rows@example.com",
		}

		errCode := repoRowsFail.Update(ctx, user)
		if errCode != xdb.XERR_REPO_UPDATE_VERIFY_ROWS_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_UPDATE_VERIFY_ROWS_FAILED, errCode)
		}
	})
}

func TestDBGeneric_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := xdb.NewDBGeneric[MockUser](db)

	t.Run("Success standard delete", func(t *testing.T) {
		user := &MockUser{Name: "To Delete", Email: "delete@example.com"}
		_ = repo.Insert(ctx, user)

		errCode := repo.Delete(ctx, user)
		if errCode != xdb.XERR_NONE {
			t.Errorf("expected XERR_NONE, got %s", errCode)
		}

		_, fetchErr := repo.GetByID(ctx, user.ID)
		if fetchErr != xdb.XERR_REPO_GET_BY_ID_RECORD_NOT_FOUND {
			t.Errorf("expected record to be gone, got %s", fetchErr)
		}
	})

	t.Run("Fail when database execution crashes", func(t *testing.T) {
		deadDb, _ := sql.Open("sqlite", ":memory:")
		deadDb.Close()
		deadRepo := xdb.NewDBGeneric[MockUser](deadDb)

		user := &MockUser{ID: 1}

		errCode := deadRepo.Delete(ctx, user)
		if errCode != xdb.XERR_REPO_DELETE_EXEC_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_DELETE_EXEC_FAILED, errCode)
		}
	})

	t.Run("Fail when rows affected verification fails", func(t *testing.T) {
		repoRowsFail := xdb.NewDBGeneric[MockUser](db)
		repoRowsFail.SetExecutorForTest(mockExecutorRowsError{})

		user := &MockUser{ID: 1}

		errCode := repoRowsFail.Delete(ctx, user)
		if errCode != xdb.XERR_REPO_DELETE_VERIFY_ROWS_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_DELETE_VERIFY_ROWS_FAILED, errCode)
		}
	})

	t.Run("Fail record not found without idempotency", func(t *testing.T) {
		user := &MockUser{ID: 999}

		errCode := repo.Delete(ctx, user)
		if errCode != xdb.XERR_REPO_DELETE_RECORD_NOT_FOUND {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_DELETE_RECORD_NOT_FOUND, errCode)
		}
	})

	t.Run("Success record not found with instance idempotency", func(t *testing.T) {
		user := &MockUser{ID: 888}

		repo.SetIdempotentDelete(true)
		defer repo.SetIdempotentDelete(false)

		errCode := repo.Delete(ctx, user)
		if errCode != xdb.XERR_NONE {
			t.Errorf("expected XERR_NONE due to idempotency, got %s", errCode)
		}
	})

	t.Run("Success record not found with forced context idempotency", func(t *testing.T) {
		user := &MockUser{ID: 777}

		forcedCtx := xdb.ContextWithForcedIdempotency(ctx)

		errCode := repo.Delete(forcedCtx, user)
		if errCode != xdb.XERR_NONE {
			t.Errorf("expected XERR_NONE due to forced context, got %s", errCode)
		}
	})

	t.Run("Fail record not found when context prohibits idempotency", func(t *testing.T) {
		user := &MockUser{ID: 666}

		repo.SetIdempotentDelete(true)
		defer repo.SetIdempotentDelete(false)

		prohibitedCtx := xdb.ContextWithProhibitedIdempotency(ctx)

		errCode := repo.Delete(prohibitedCtx, user)
		if errCode != xdb.XERR_REPO_DELETE_RECORD_NOT_FOUND {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_DELETE_RECORD_NOT_FOUND, errCode)
		}
	})
}

func TestDBGeneric_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := xdb.NewDBGeneric[MockUser](db)

	user := &MockUser{Name: "Get Tester", Email: "get@example.com"}
	_ = repo.Insert(ctx, user)

	t.Run("Success get record by id", func(t *testing.T) {
		fetched, errCode := repo.GetByID(ctx, user.ID)
		if errCode != xdb.XERR_NONE {
			t.Errorf("expected XERR_NONE, got %s", errCode)
		}
		if fetched == nil || fetched.Name != "Get Tester" {
			t.Errorf("failed to fetch correct user data")
		}
	})

	t.Run("Fail when record is not found", func(t *testing.T) {
		fetched, errCode := repo.GetByID(ctx, 9999)
		if errCode != xdb.XERR_REPO_GET_BY_ID_RECORD_NOT_FOUND {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_GET_BY_ID_RECORD_NOT_FOUND, errCode)
		}
		if fetched != nil {
			t.Errorf("expected nil instance on failure")
		}
	})

	t.Run("Fail when database execution crashes", func(t *testing.T) {
		deadDb, _ := sql.Open("sqlite", ":memory:")
		deadDb.Close()
		deadRepo := xdb.NewDBGeneric[MockUser](deadDb)

		_, errCode := deadRepo.GetByID(ctx, 1)
		if errCode != xdb.XERR_REPO_GET_BY_ID_EXEC_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_GET_BY_ID_EXEC_FAILED, errCode)
		}
	})

	t.Run("Fail when rows scan operation fails", func(t *testing.T) {
		dbScanFail := setupTestDB(t)
		defer dbScanFail.Close()
		repoScanFail := xdb.NewDBGeneric[MockUser](dbScanFail)

		_, err := dbScanFail.Exec("INSERT INTO mock_users (id, uuid, name, email) VALUES (?, ?, ?, ?);", 1, nil, "FORCE_SCAN_FAILURE", "fail@example.com")
		if err != nil {
			t.Fatalf("failed to prepare setup table record: %v", err)
		}

		fetched, errCode := repoScanFail.GetByID(ctx, 1)
		if errCode != xdb.XERR_REPO_GET_BY_ID_SCAN_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_GET_BY_ID_SCAN_FAILED, errCode)
		}
		if fetched != nil {
			t.Errorf("expected nil instance on scan failure")
		}
	})
}

func TestDBGeneric_GetAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := xdb.NewDBGeneric[MockUser](db)

	_ = repo.Insert(ctx, &MockUser{Name: "User One", Email: "one@example.com"})
	_ = repo.Insert(ctx, &MockUser{Name: "User Two", Email: "two@example.com"})

	t.Run("Success get all records", func(t *testing.T) {
		list, errCode := repo.GetAll(ctx)
		if errCode != xdb.XERR_NONE {
			t.Errorf("expected XERR_NONE, got %s", errCode)
		}
		if len(list) != 2 {
			t.Errorf("expected 2 records, got %d", len(list))
		}
	})

	t.Run("Fail when database execution crashes", func(t *testing.T) {
		deadDb, _ := sql.Open("sqlite", ":memory:")
		deadDb.Close()
		deadRepo := xdb.NewDBGeneric[MockUser](deadDb)

		_, errCode := deadRepo.GetAll(ctx)
		if errCode != xdb.XERR_REPO_GET_ALL_EXEC_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_GET_ALL_EXEC_FAILED, errCode)
		}
	})

	t.Run("Fail when row scan operation fails", func(t *testing.T) {
		dbScanFail := setupTestDB(t)
		defer dbScanFail.Close()
		repoScanFail := xdb.NewDBGeneric[MockUser](dbScanFail)

		_, _ = dbScanFail.Exec("INSERT INTO mock_users (id, uuid, name, email) VALUES (?, ?, ?, ?);", 1, nil, "FORCE_SCAN_FAILURE", "fail@example.com")

		_, errCode := repoScanFail.GetAll(ctx)
		if errCode != xdb.XERR_REPO_GET_ALL_SCAN_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_GET_ALL_SCAN_FAILED, errCode)
		}
	})

	t.Run("Fail when cursor iteration fails", func(t *testing.T) {
		repoRowsFail := xdb.NewDBGeneric[MockUser](db)

		repoRowsFail.SetExecutorForTest(mockExecutorGetAllIterError{})

		_, errCode := repoRowsFail.GetAll(ctx)
		if errCode != xdb.XERR_REPO_GET_ALL_ITERATION_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_GET_ALL_ITERATION_FAILED, errCode)
		}
	})
}

func TestDBGeneric_GetByField(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := xdb.NewDBGeneric[MockUser](db)

	_ = repo.Insert(ctx, &MockUser{Name: "Target User", Email: "target@example.com"})
	_ = repo.Insert(ctx, &MockUser{Name: "Other User", Email: "other@example.com"})

	t.Run("Success get records by field", func(t *testing.T) {
		list, errCode := repo.GetByField(ctx, "email", "target@example.com")
		if errCode != xdb.XERR_NONE {
			t.Errorf("expected XERR_NONE, got %s", errCode)
		}
		if len(list) != 1 {
			t.Errorf("expected 1 record, got %d", len(list))
		}
		if list[0].Name != "Target User" {
			t.Errorf("expected name 'Target User', got %s", list[0].Name)
		}
	})

	t.Run("Fail when database execution crashes", func(t *testing.T) {
		deadDb, _ := sql.Open("sqlite", ":memory:")
		deadDb.Close()
		deadRepo := xdb.NewDBGeneric[MockUser](deadDb)

		_, errCode := deadRepo.GetByField(ctx, "email", "target@example.com")
		if errCode != xdb.XERR_REPO_GET_BY_FIELD_EXEC_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_GET_BY_FIELD_EXEC_FAILED, errCode)
		}
	})

	t.Run("Fail when row scan operation fails", func(t *testing.T) {
		dbScanFail := setupTestDB(t)
		defer dbScanFail.Close()
		repoScanFail := xdb.NewDBGeneric[MockUser](dbScanFail)

		_, _ = dbScanFail.Exec("INSERT INTO mock_users (id, uuid, name, email) VALUES (?, ?, ?, ?);", 1, nil, "FORCE_SCAN_FAILURE", "fail@example.com")

		_, errCode := repoScanFail.GetByField(ctx, "email", "fail@example.com")
		if errCode != xdb.XERR_REPO_GET_BY_FIELD_SCAN_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_GET_BY_FIELD_SCAN_FAILED, errCode)
		}
	})

	t.Run("Fail when cursor iteration fails", func(t *testing.T) {
		repoRowsFail := xdb.NewDBGeneric[MockUser](db)

		repoRowsFail.SetExecutorForTest(mockExecutorRowsError{})

		_, errCode := repoRowsFail.GetByField(ctx, "email", "FAIL_GETBYFIELD")
		if errCode != xdb.XERR_REPO_GET_ALL_ITERATION_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_GET_ALL_ITERATION_FAILED, errCode)
		}
	})
}

func TestDBGeneric_GetWhere(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := xdb.NewDBGeneric[MockUser](db)

	_ = repo.Insert(ctx, &MockUser{Name: "Active User One", Email: "active1@example.com"})
	_ = repo.Insert(ctx, &MockUser{Name: "Active User Two", Email: "active2@example.com"})

	t.Run("Success get records with custom where fragment", func(t *testing.T) {
		list, errCode := repo.GetWhere(ctx, "name LIKE ? AND email = ?", "Active User%", "active1@example.com")
		if errCode != xdb.XERR_NONE {
			t.Errorf("expected XERR_NONE, got %s", errCode)
		}
		if len(list) != 1 {
			t.Errorf("expected 1 record, got %d", len(list))
		}
		if list[0].Name != "Active User One" {
			t.Errorf("expected name 'Active User One', got %s", list[0].Name)
		}
	})

	t.Run("Fail when placeholders count and arguments length mismatch", func(t *testing.T) {
		// Passamos 2 placeholders mas apenas 1 argumento de propósito
		_, errCode := repo.GetWhere(ctx, "name = ? AND email = ?", "Active User One")
		if errCode != xdb.XERR_REPO_GET_WHERE_ARGS_MISMATCH {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_GET_WHERE_ARGS_MISMATCH, errCode)
		}
	})

	t.Run("Fail when database execution crashes", func(t *testing.T) {
		deadDb, _ := sql.Open("sqlite", ":memory:")
		deadDb.Close()
		deadRepo := xdb.NewDBGeneric[MockUser](deadDb)

		_, errCode := deadRepo.GetWhere(ctx, "name = ?", "Active User One")
		if errCode != xdb.XERR_REPO_GET_WHERE_EXEC_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_GET_WHERE_EXEC_FAILED, errCode)
		}
	})

	t.Run("Fail when row scan operation fails", func(t *testing.T) {
		dbScanFail := setupTestDB(t)
		defer dbScanFail.Close()
		repoScanFail := xdb.NewDBGeneric[MockUser](dbScanFail)

		_, err := dbScanFail.Exec(
			"INSERT INTO mock_users (id, uuid, name, email) VALUES (?, ?, ?, ?);",
			1, nil, "FORCE_SCAN_FAILURE", "fail@example.com",
		)
		if err != nil {
			t.Fatalf("failed to prepare setup table record: %v", err)
		}

		_, errCode := repoScanFail.GetWhere(ctx, "email = ?", "fail@example.com")
		if errCode != xdb.XERR_REPO_GET_WHERE_SCAN_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_GET_WHERE_SCAN_FAILED, errCode)
		}
	})

	t.Run("Fail when cursor iteration fails", func(t *testing.T) {
		repoRowsFail := xdb.NewDBGeneric[MockUser](db)
		repoRowsFail.SetExecutorForTest(mockExecutorRowsError{})

		_, errCode := repoRowsFail.GetWhere(ctx, "name = ?", "FAIL_GETWHERE")
		if errCode != xdb.XERR_REPO_GET_ALL_ITERATION_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_GET_ALL_ITERATION_FAILED, errCode)
		}
	})
}

func TestDBGeneric_Count(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := xdb.NewDBGeneric[MockUser](db)
	user := &MockUser{Name: "Count User", Email: "count@example.com"}
	if errCode := repo.Insert(ctx, user); errCode != xdb.XERR_NONE {
		t.Fatalf("failed to insert count fixture: %s", errCode)
	}

	t.Run("Success counting matching primary key", func(t *testing.T) {
		count, errCode := repo.Count(ctx, user.ID)
		if errCode != xdb.XERR_NONE {
			t.Fatalf("expected %s, got %s", xdb.XERR_NONE, errCode)
		}
		if count != 1 {
			t.Errorf("expected count 1, got %d", count)
		}
	})

	t.Run("Success returning zero for missing primary key", func(t *testing.T) {
		count, errCode := repo.Count(ctx, 9999)
		if errCode != xdb.XERR_NONE {
			t.Fatalf("expected %s, got %s", xdb.XERR_NONE, errCode)
		}
		if count != 0 {
			t.Errorf("expected count 0, got %d", count)
		}
	})

	t.Run("Fail when database execution crashes", func(t *testing.T) {
		deadDb, _ := sql.Open("sqlite", ":memory:")
		deadDb.Close()
		deadRepo := xdb.NewDBGeneric[MockUser](deadDb)

		_, errCode := deadRepo.Count(ctx, user.ID)
		if errCode != xdb.XERR_REPO_COUNT_EXEC_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_COUNT_EXEC_FAILED, errCode)
		}
	})

	t.Run("Fail when cursor iteration crashes", func(t *testing.T) {
		repoRowsFail := xdb.NewDBGeneric[MockUser](db)
		repoRowsFail.SetExecutorForTest(mockExecutorGetAllIterError{})

		_, errCode := repoRowsFail.Count(ctx, user.ID)
		if errCode != xdb.XERR_REPO_COUNT_SCAN_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_COUNT_SCAN_FAILED, errCode)
		}
	})

	t.Run("Fail when count scan crashes", func(t *testing.T) {
		repoScanFail := xdb.NewDBGeneric[MockUser](db)
		repoScanFail.SetExecutorForTest(mockExecutorCountScanError{})

		_, errCode := repoScanFail.Count(ctx, user.ID)
		if errCode != xdb.XERR_REPO_COUNT_SCAN_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_COUNT_SCAN_FAILED, errCode)
		}
	})

	t.Run("Success counting records with condition", func(t *testing.T) {
		count, errCode := repo.CountWhere(ctx, "email = ?", "count@example.com")
		if errCode != xdb.XERR_NONE {
			t.Fatalf("expected %s, got %s", xdb.XERR_NONE, errCode)
		}
		if count != 1 {
			t.Errorf("expected count 1, got %d", count)
		}
	})

	t.Run("Fail when placeholders and arguments mismatch", func(t *testing.T) {
		_, errCode := repo.CountWhere(ctx, "email = ? AND name = ?", "count@example.com")
		if errCode != xdb.XERR_REPO_COUNT_WHERE_ARGS_MISMATCH {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_COUNT_WHERE_ARGS_MISMATCH, errCode)
		}
	})

	t.Run("Fail conditional count when database execution crashes", func(t *testing.T) {
		deadDb, _ := sql.Open("sqlite", ":memory:")
		deadDb.Close()
		deadRepo := xdb.NewDBGeneric[MockUser](deadDb)

		_, errCode := deadRepo.CountWhere(ctx, "email = ?", "count@example.com")
		if errCode != xdb.XERR_REPO_COUNT_WHERE_EXEC_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_COUNT_WHERE_EXEC_FAILED, errCode)
		}
	})

	t.Run("Fail conditional count when cursor iteration crashes", func(t *testing.T) {
		repoRowsFail := xdb.NewDBGeneric[MockUser](db)
		repoRowsFail.SetExecutorForTest(mockExecutorGetAllIterError{})

		_, errCode := repoRowsFail.CountWhere(ctx, "email = ?", "count@example.com")
		if errCode != xdb.XERR_REPO_COUNT_WHERE_SCAN_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_COUNT_WHERE_SCAN_FAILED, errCode)
		}
	})

	t.Run("Fail conditional count when scan crashes", func(t *testing.T) {
		repoScanFail := xdb.NewDBGeneric[MockUser](db)
		repoScanFail.SetExecutorForTest(mockExecutorCountScanError{})

		_, errCode := repoScanFail.CountWhere(ctx, "email = ?", "count@example.com")
		if errCode != xdb.XERR_REPO_COUNT_WHERE_SCAN_FAILED {
			t.Errorf("expected %s, got %s", xdb.XERR_REPO_COUNT_WHERE_SCAN_FAILED, errCode)
		}
	})
}
