package xdb

import (
	"context"
	"database/sql"

	"github.com/AeonDigital/Go-Core-xerrors/pkg/xerrors"
)

// ctxIdempotencyKey defines a custom type for context-based idempotency flags.
type ctxIdempotencyKey string

const (
	// forceIdempotencyKey bypasses strict missing record checks for updates and
	// deletes.
	forceIdempotencyKey ctxIdempotencyKey = "force_idempotency"
	// prohibitIdempotencyKey enforces record existence validation even if the
	// instance defaults to idempotent operations.
	prohibitIdempotencyKey ctxIdempotencyKey = "prohibit_idempotency"
)

// SQLExecutor unifies common database operations available on both *sql.DB
// and *sql.Tx connections.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Entity establishes the mandatory domain lifecycle methods required for
// automatic CRUD operations.
type Entity interface {
	// TableName returns the exact database table identifier linked to this entity.
	TableName() string

	//
	// PK Definitions

	// PKColumnName returns the physical database primary key column
	// identifier (e.g., "id" or "key").
	PKColumnName() string

	// PKSetValue allows the repository engine to inject database-generated
	// or application-generated identifiers back into the instance memory
	// pointer.
	PKSetValue(id any)

	// PKGetValue returns the current snapshot value of the primary key
	// field (e.g., an int64 ID or a string UUID).
	PKGetValue() any

	// PKGenerateValue creates an application-side unique identifier,
	// returning nil if the key lifecycle is delegated to the database
	// engine.
	PKGenerateValue() any

	// PKExternal indicates whether the entity's primary key comes from
	// an external identifier provided by the system context, rather
	// than from the database itself (This value is commonly 'false').
	PKExternal() bool

	//
	// General Values

	// Columns yields the sequence of table columns targeted for
	// INSERT/UPDATE actions, strictly excluding database-managed values.
	Columns() []string

	// Values yields the field records mapped in the exact corresponding
	// sequence order specified by Columns().
	// - flat: if true returns all values in a flat object.
	Values(flat bool) []any

	// ScanRow hydrats the entire entity fields from an active database
	// query cursor row result, mapping all table columns sequentially.
	ScanRow(rows *sql.Rows) error

	//
	// Normalization and Validation

	// Normalize cleanses and standardizes internal field values before
	// processing (e.g., trimming whitespace or altering casing).
	Normalize()

	// Validate performs fail-fast domain business rules checking,
	// returning false and a distinct status code upon failure.
	Validate() (bool, xerrors.ErrorCode)
}

// RowScanner defines the function signature required to map database columns into a structured type.
type RowScanner[R any] func(rows *sql.Rows) (R, error)

// CustomQuery decouples raw SQL execution from rigid models by bundling the query statement, its runtime parameters, and its mapping logic.
type CustomQuery[R any] struct {
	SQL     string
	Args    []any
	Scanner RowScanner[R]
}

//
// Expand sql.Null<type> objects
//

// nullableType defines a generic type constraint for the local wrappers.
// It maps the internal default structure of each sql.Null* type.
type nullableType interface {
	NullBool | NullByte | NullInt16 | NullInt32 | NullInt64 | NullFloat64 | NullTime | NullString
}

// evalNull centralizes validation logic for nullable types. If the struct is invalid, it returns nil;
// if valid, it returns the actual value extracted from the correct field.
func evalNull[T nullableType](n T) any {
	switch v := any(n).(type) {
	case NullBool:
		if !v.Valid {
			return nil
		}
		return v.Bool

	case NullByte:
		if !v.Valid {
			return nil
		}
		return v.Byte

	case NullInt16:
		if !v.Valid {
			return nil
		}
		return v.Int16

	case NullInt32:
		if !v.Valid {
			return nil
		}
		return v.Int32

	case NullInt64:
		if !v.Valid {
			return nil
		}
		return v.Int64

	case NullFloat64:
		if !v.Valid {
			return nil
		}
		return v.Float64

	case NullTime:
		if !v.Valid {
			return nil
		}
		return v.Time

	case NullString:
		if !v.Valid {
			return nil
		}
		return v.String

	default:
		return nil
	}
}

// NullBool represents a bool that may be null.
type NullBool struct{ sql.NullBool }

// Val return nil or bool acording to its real value
func (n NullBool) Val() any { return evalNull(n) }

// NullByte represents a byte that may be null.
type NullByte struct{ sql.NullByte }

// Val return nil or byte acording to its real value
func (n NullByte) Val() any { return evalNull(n) }

// NullInt16 represents a int16 that may be null.
type NullInt16 struct{ sql.NullInt16 }

// Val return nil or int16 acording to its real value
func (n NullInt16) Val() any { return evalNull(n) }

// NullInt32 represents a int32 that may be null.
type NullInt32 struct{ sql.NullInt32 }

// Val return nil or int32 acording to its real value
func (n NullInt32) Val() any { return evalNull(n) }

// NullInt64 represents a int64 that may be null.
type NullInt64 struct{ sql.NullInt64 }

// Val return nil or int64 acording to its real value
func (n NullInt64) Val() any { return evalNull(n) }

// NullFloat64 represents a float64 that may be null.
type NullFloat64 struct{ sql.NullFloat64 }

// Val return nil or float64 acording to its real value
func (n NullFloat64) Val() any { return evalNull(n) }

// NullTime represents a time.Time that may be null.
type NullTime struct{ sql.NullTime }

// Val return nil or time.Time acording to its real value
func (n NullTime) Val() any { return evalNull(n) }

// NullString represents a string that may be null.
type NullString struct{ sql.NullString }

// Val return nil or string acording to its real value
func (n NullString) Val() any { return evalNull(n) }
