package xdb

import (
	"context"
	"database/sql"
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
