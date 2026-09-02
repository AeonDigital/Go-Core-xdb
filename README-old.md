xdb
================================

> Type-safe generic repository pattern for database CRUD operations with transaction and idempotency support.

&nbsp;

This package provides a `DBGeneric` generic repository abstraction that encapsulates
CRUD operations for strongly-typed domain entities. It enforces domain validation,
supports database transactions, and enables configurable idempotent behavior for
updates and deletes.


________________________________________________________________________________

## Purpose

`xdb` centralizes database operations behind a type-safe generic repository interface.
It simplifies common data persistence tasks such as:

* creating, reading, updating, and deleting domain entities,
* binding database-managed and application-generated identifiers,
* executing custom raw SQL queries with type-safe row scanning,
* managing transaction lifecycles with shallow-copy context propagation,
* enforcing domain validation through the `Entity` interface,
* controlling idempotent behavior per-operation via context flags,
* centralizing database configuration for embedded deployments.

The package is designed to minimize boilerplate repository code while maintaining
strong type safety and predictable error handling through domain-specific error codes.


________________________________________________________________________________

## Installation

Use `go get` to add the package to your module:

```shell
  go get github.com/AeonDigital/Go-Core/xdb@latest
```

Import it in your code:

```go
import "github.com/AeonDigital/Go-Core/xdb"
```

If you also need structured error handling from the repository, import:

```go
import "github.com/AeonDigital/Go-Core-xerrors/pkg/xerrors"
```


________________________________________________________________________________

## Core Abstractions

### Entity Interface

Every domain model must implement the `Entity` interface to work with `DBGeneric`:

```go
type MyEntity struct {
    ID    int64
    Name  string
}

func (e *MyEntity) Normalize()                     { /* clean fields */ }
func (e *MyEntity) Validate() (bool, xerrors.ErrorCode) { /* validate */ }
func (e *MyEntity) TableName() string              { return "my_table" }
func (e *MyEntity) Columns() []string              { return []string{"name"} }
func (e *MyEntity) Values() []any                  { return []any{e.Name} }
func (e *MyEntity) TablePK() string                { return "id" }
func (e *MyEntity) BindPK(id any)                  { e.ID = id.(int64) }
func (e *MyEntity) PKValue() any                   { return e.ID }
func (e *MyEntity) ScanRow(rows *sql.Rows) error  { /* hydrate */ }
func (e *MyEntity) GeneratePK() any                { return nil }
func (e *MyEntity) IsNaturalPK() bool              { return false }
```

### Generic Repository

Once your entity implements `Entity`, instantiate a repository:

```go
repo := xdb.NewDBGeneric[MyEntity, *MyEntity](db)
```

### Idempotency Control

Control whether update/delete operations should fail if the target record is missing:

```go
// Force idempotent behavior (no error if record missing)
ctx := xdb.ContextWithForcedIdempotency(context.Background())
errCode := repo.Update(ctx, entity)

// Prohibit idempotent behavior (error if record missing)
ctx := xdb.ContextWithProhibitedIdempotency(context.Background())
errCode := repo.Update(ctx, entity)
```

### Transaction Support

Bind the repository to an active transaction:

```go
tx, _ := db.BeginTx(ctx, nil)
defer tx.Rollback()

txRepo := repo.WithTx(tx)
errCode := txRepo.Insert(ctx, entity)

tx.Commit()
```


________________________________________________________________________________

## External dependencies

This package depends on:

* `database/sql` — Go standard library database abstraction.
* `context` — Go standard library context propagation.

Optional dependencies for specific use cases:

* `github.com/AeonDigital/Go-Core/xerrors` — structured error integration.
* `modernc.org/sqlite` — SQLite driver for embedded deployments.
