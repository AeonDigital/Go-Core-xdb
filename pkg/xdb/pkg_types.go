package xdb

import (
	"context"
	"database/sql"

	"github.com/AeonDigital/Go-Core-xerrors/pkg/xerrors"
)

// ctxIdempotencyKey defines a custom type for context-based idempotency flags.
type ctxIdempotencyKey string

const (
	// forceIdempotencyKey bypasses strict missing record checks for updates and deletes.
	forceIdempotencyKey ctxIdempotencyKey = "force_idempotency"
	// prohibitIdempotencyKey enforces record existence validation even if the instance defaults to idempotent operations.
	prohibitIdempotencyKey ctxIdempotencyKey = "prohibit_idempotency"
)

// SQLExecutor unifies common database operations available on both *sql.DB and *sql.Tx connections.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Entity establishes the mandatory domain lifecycle methods required for automatic CRUD operations.
type Entity interface {
	// Normalize cleanses and standardizes internal field values before processing (e.g., trimming whitespace or altering casing).
	Normalize()

	// Validate performs fail-fast domain business rules checking, returning false and a distinct status code upon failure.
	Validate() (bool, xerrors.ErrorCode)

	// TableName returns the exact database table identifier linked to this entity.
	TableName() string

	// Columns yields the sequence of table columns targeted for INSERT/UPDATE actions, strictly excluding database-managed values.
	Columns() []string

	// Values yields the field records mapped in the exact corresponding sequence order specified by Columns().
	Values() []any

	// TablePK returns the physical database primary key column identifier (e.g., "id" or "key").
	TablePK() string

	// BindPK allows the repository engine to inject database-generated or application-generated identifiers back into the instance memory pointer.
	BindPK(id any)

	// PKValue returns the current snapshot value of the primary key field (e.g., an int64 ID or a string UUID).
	PKValue() any

	// ScanRow hydrats the entire entity fields from an active database query cursor row result, mapping all table columns sequentially.
	ScanRow(rows *sql.Rows) error

	// GeneratePK creates an application-side unique identifier, returning nil if the key lifecycle is delegated to the database engine.
	GeneratePK() any

	// IsNaturalPK indicates whether the entity primary key is a natural identifier supplied from an external context outside the system.
	IsNaturalPK() bool
}

// RowScanner defines the function signature required to map database columns into a structured type.
type RowScanner[R any] func(rows *sql.Rows) (R, error)

// CustomQuery decouples raw SQL execution from rigid models by bundling the query statement, its runtime parameters, and its mapping logic.
type CustomQuery[R any] struct {
	SQL     string
	Args    []any
	Scanner RowScanner[R]
}
