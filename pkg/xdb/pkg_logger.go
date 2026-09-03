package xdb

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/AeonDigital/Go-Core-xerrors/pkg/xerrors"
)

// logRepoError normalizes engine metadata and delegates structured repository failure telemetry to the global xlog framework.
func logRepoError(
	ctx context.Context,
	db *sql.DB,
	errorCode xerrors.ErrorCode,
	originalErr error,
	query string,
	args []any,
) {
	resource := RetrieveDbType(db)

	attrs := []slog.Attr{
		slog.String("resource", resource),
	}

	if query != "" {
		attrs = append(attrs, slog.String("sql_query", query))
	}

	if len(args) > 0 {
		attrs = append(attrs, slog.Any("args", args))
	}

	attrs = append(attrs, slog.String("error_cod", string(errorCode)))
	attrs = append(attrs, slog.String("error_str", string(errorCode)))

	// If the original error implements DetailedError, adjust its caller skip
	// so the reported component points to the real origin.
	if de, ok := originalErr.(xerrors.IError500); ok {
		de = de.WithCallerSkip(1)
		attrs = append(attrs, slog.String("component", de.Component()))
	}

	if originalErr != nil {
		attrs = append(attrs, slog.String("error", originalErr.Error()))
	}

	// Emit structured error via slog so that consumer projects can configure
	// `slog` with an `xlog.LogHandler` to control formatting/outputs.
	slog.LogAttrs(ctx, slog.LevelError, "operation failure detected", attrs...)
}
