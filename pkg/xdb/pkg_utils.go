package xdb

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

// ContextWithForcedIdempotency wraps the provided context to guarantee that down-stream update and delete actions run idempotently.
func ContextWithForcedIdempotency(ctx context.Context) context.Context {
	return context.WithValue(ctx, forceIdempotencyKey, true)
}

// ContextWithProhibitedIdempotency wraps the provided context to strictly block idempotent behavior on down-stream data modifications.
func ContextWithProhibitedIdempotency(ctx context.Context) context.Context {
	return context.WithValue(ctx, prohibitIdempotencyKey, true)
}

// retrieveDbType inspects the underlying driver of an active sql.DB instance safely to identify the target dialect.
func RetrieveDbType(db *sql.DB) string {
	resource := "DB"

	if db != nil {
		if drv := db.Driver(); drv != nil {
			drvType := strings.ToLower(reflect.TypeOf(drv).String())
			switch {
			case strings.Contains(drvType, "sqlite"):
				resource = "sqlite"
			case strings.Contains(drvType, "mysql"):
				resource = "mysql"
			case strings.Contains(drvType, "postgres") || strings.Contains(drvType, "pq"):
				resource = "postgres"
			}
		}
	}

	return resource
}

// BuildTruncateQuery assembles the correct schema-clearing statement for the given database dialect.
// SQLite has no TRUNCATE statement, so it falls back to an unconditional DELETE.
func BuildTruncateQuery(dbType string, tableName string) string {
	switch dbType {
	case "postgres", "mysql":
		return fmt.Sprintf("TRUNCATE TABLE %s;", tableName)
	default:
		return fmt.Sprintf("DELETE FROM %s;", tableName)
	}
}

// GetAllColumnNames yields the complete sequence of database column names for the entity.
// It places the Primary Key column identifier (if present) as the very first element,
// followed by the sequence of data columns targeted for operations.
func GetAllColumnNames(entity Entity) []string {
	if entity == nil {
		return nil
	}

	cols := entity.Columns()
	pkCol := entity.PKColumnName()

	if pkCol == "" {
		return cols
	}

	// Pre-allocate the slice with exact required capacity
	allCols := make([]string, 0, len(cols)+1)
	allCols = append(allCols, pkCol)
	allCols = append(allCols, cols...)

	return allCols
}

// GetAllColumnValues yields the complete sequence of data values for the entity.
// It places the Primary Key value (if present) as the very first element,
// followed by the sequence of field records targeted for operations.
func GetAllColumnValues(entity Entity) []any {
	if entity == nil {
		return nil
	}

	vals := entity.Values(true)
	pkCol := entity.PKColumnName()

	if pkCol == "" {
		return vals
	}

	// Pre-allocate the slice with exact required capacity
	allVals := make([]any, 0, len(vals)+1)
	allVals = append(allVals, entity.PKGetValue())
	allVals = append(allVals, vals...)

	return allVals
}

// ConvertEntityAsMap transforms an Entity implementation into a key-value map representation.
//
// This function combines the entity's complete schema columns and data values into a unified
// map[string]any by leveraging GetAllColumnNames and GetAllColumnValues.
//
// Behavior/Constraints:
//   - Guarantees that the primary key is handled consistently at index 0 across both layers.
//   - Aligns columns with their respective values sequentially, safely preventing out-of-bounds
//     runtime panics if slice lengths differ by applying a defensive upper limit.
func ConvertEntityAsMap(entity Entity) map[string]any {
	if entity == nil {
		return nil
	}

	allCols := GetAllColumnNames(entity)
	allVals := GetAllColumnValues(entity)

	if len(allCols) == 0 {
		return make(map[string]any)
	}

	// Initialize map with exact capacity to minimize allocations
	result := make(map[string]any, len(allCols))

	// Defensive check: ensure slices match to prevent out-of-bounds runtime panics
	limit := min(len(allVals), len(allCols))

	for i := range limit {
		result[allCols[i]] = allVals[i]
	}

	return result
}
