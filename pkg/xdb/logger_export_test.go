package xdb

import (
	"context"
	"database/sql"

	"github.com/AeonDigital/Go-Core-xerrors/pkg/xerrors"
)

// Expõe a função privada logRepoError para o pacote de testes externos.
func LogRepoError(
	ctx context.Context,
	db *sql.DB,
	errorCode xerrors.ErrorCode,
	originalErr error,
	query string,
	args []any,
) {
	logRepoError(ctx, db, errorCode, originalErr, query, args)
}
