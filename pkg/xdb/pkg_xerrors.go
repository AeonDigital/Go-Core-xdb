package xdb

import (
	"github.com/AeonDigital/Go-Core-xerrors/pkg/xerrors"
)

const (
	XERR_NONE   xerrors.ErrorCode = ""
	XERR_PKGCTX xerrors.ErrorCode = "ERR_XDB"

	// ============================================================================
	// GROUP 1: DATABASE INFRASTRUCTURE AND ENVIRONMENT ERRORS
	// Shared Layout Structure Token: [CTX: %v][ERR: %v][COMPONENT: %v][MSG: %v][TARGET: %v][DETAILS: %v] :: [ERR: %w]
	// ============================================================================

	// XERR_CONNECTION_FAILED targets failures when establishing physical channel lines or initial driver handshakes.
	// Format expects: TARGET, DETAILS
	XERR_CONNECTION_FAILED xerrors.ErrorCode = "E1001"

	// XERR_ENGINE_CONFIG_FAILED targets syntax or logic failures when applying internal engine runtime configurations (e.g. PRAGMAs).
	// Format expects: TARGET, DETAILS
	XERR_ENGINE_CONFIG_FAILED xerrors.ErrorCode = "E1002"

	// XERR_MIGRATION_EXECUTION_FAILED targets processing sequences where valid SQL structures fail state transmission or command limits.
	// Format expects: TARGET, DETAILS
	XERR_MIGRATION_EXECUTION_FAILED xerrors.ErrorCode = "E1003"

	XERR_REPO_INSERT_WITH_UNEXPECTED_NUMERIC_PK xerrors.ErrorCode = "E2001"
	XERR_REPO_INSERT_WITH_UNEXPECTED_STRING_PK  xerrors.ErrorCode = "E2002"
	XERR_REPO_INSERT_WITH_UNEXPECTED_EMPTY_PK   xerrors.ErrorCode = "E2003"
	XERR_REPO_INSERT_WITH_UNEXPECTED_NIL_PK     xerrors.ErrorCode = "E2004"
	XERR_REPO_INSERT_EXECUTION_FAILED           xerrors.ErrorCode = "E2005"
	XERR_REPO_INSERT_FETCH_ID_FAILED            xerrors.ErrorCode = "E2006"

	XERR_REPO_UPDATE_INVALID_NUMERICAL_PK xerrors.ErrorCode = "E3001"
	XERR_REPO_UPDATE_EMPTY_STRING_PK      xerrors.ErrorCode = "E3002"
	XERR_REPO_UPDATE_PK_NIL               xerrors.ErrorCode = "E3003"
	XERR_REPO_UPDATE_UNKNOWN_PK_TYPE      xerrors.ErrorCode = "E3004"
	XERR_REPO_UPDATE_NO_COLUMNS_DEFINED   xerrors.ErrorCode = "E3005"
	XERR_REPO_UPDATE_EXEC_FAILED          xerrors.ErrorCode = "E3006"
	XERR_REPO_UPDATE_VERIFY_ROWS_FAILED   xerrors.ErrorCode = "E3007"
	XERR_REPO_UPDATE_RECORD_NOT_FOUND     xerrors.ErrorCode = "E3008"

	XERR_REPO_DELETE_EXEC_FAILED        xerrors.ErrorCode = "E4001"
	XERR_REPO_DELETE_VERIFY_ROWS_FAILED xerrors.ErrorCode = "E4002"
	XERR_REPO_DELETE_RECORD_NOT_FOUND   xerrors.ErrorCode = "E4003"

	XERR_REPO_GET_BY_ID_EXEC_FAILED      xerrors.ErrorCode = "E5001"
	XERR_REPO_GET_BY_ID_RECORD_NOT_FOUND xerrors.ErrorCode = "E5002"
	XERR_REPO_GET_BY_ID_SCAN_FAILED      xerrors.ErrorCode = "E5003"
	XERR_REPO_GET_ALL_EXEC_FAILED        xerrors.ErrorCode = "E5004"
	XERR_REPO_GET_ALL_SCAN_FAILED        xerrors.ErrorCode = "E5005"
	XERR_REPO_GET_ALL_ITERATION_FAILED   xerrors.ErrorCode = "E5006"
	XERR_REPO_GET_BY_FIELD_EXEC_FAILED   xerrors.ErrorCode = "E5007"
	XERR_REPO_GET_BY_FIELD_SCAN_FAILED   xerrors.ErrorCode = "E5008"
	XERR_REPO_GET_WHERE_ARGS_MISMATCH    xerrors.ErrorCode = "E5009"
	XERR_REPO_GET_WHERE_EXEC_FAILED      xerrors.ErrorCode = "E5010"
	XERR_REPO_GET_WHERE_SCAN_FAILED      xerrors.ErrorCode = "E5011"

	XERR_REPO_QUERY_RAW_EXEC_FAILED xerrors.ErrorCode = "E6001"
	XERR_REPO_QUERY_RAW_SCAN_FAILED xerrors.ErrorCode = "E6002"
)

// xerrorDomainMapRegistry centralizes the core validation error metadata block and default
// corporate layout mapping definitions specific to the framework's own runtime context.
var xerrorDomainMapRegistry = map[xerrors.ErrorCode]xerrors.MetaMessage{
	XERR_CONNECTION_FAILED: xerrors.NewMetaMessage(
		"database baseline operational connection channel failed",
		"",
		[]string{"TARGET", "DETAILS"},
	),
	XERR_ENGINE_CONFIG_FAILED: xerrors.NewMetaMessage(
		"failed to optimize database structural runtime settings or options",
		"",
		[]string{"TARGET", "DETAILS"},
	),
	XERR_MIGRATION_EXECUTION_FAILED: xerrors.NewMetaMessage(
		"failed to process chronological structural data migration script statements",
		"",
		[]string{"TARGET", "DETAILS"},
	),

	//
	//

	XERR_REPO_INSERT_WITH_UNEXPECTED_NUMERIC_PK: xerrors.NewMetaMessage(
		"cannot insert entity with an existing numerical ID",
		"",
		[]string{},
	),
	XERR_REPO_INSERT_WITH_UNEXPECTED_STRING_PK: xerrors.NewMetaMessage(
		"cannot insert entity with an existing primary key string",
		"",
		[]string{},
	),
	XERR_REPO_INSERT_WITH_UNEXPECTED_EMPTY_PK: xerrors.NewMetaMessage(
		"cannot insert entity: natural primary key string cannot be empty",
		"",
		[]string{},
	),
	XERR_REPO_INSERT_WITH_UNEXPECTED_NIL_PK: xerrors.NewMetaMessage(
		"cannot insert entity: natural primary key cannot be nil",
		"",
		[]string{},
	),
	XERR_REPO_INSERT_EXECUTION_FAILED: xerrors.NewMetaMessage(
		"failed to execute insert statement in the database",
		"",
		[]string{},
	),
	XERR_REPO_INSERT_FETCH_ID_FAILED: xerrors.NewMetaMessage(
		"failed to retrieve last inserted numerical ID from database",
		"",
		[]string{},
	),

	//
	//

	XERR_REPO_UPDATE_INVALID_NUMERICAL_PK: xerrors.NewMetaMessage(
		"cannot update entity with invalid numerical ID (must be > 0)",
		"",
		[]string{},
	),
	XERR_REPO_UPDATE_EMPTY_STRING_PK: xerrors.NewMetaMessage(
		"cannot update entity with empty string key",
		"",
		[]string{},
	),
	XERR_REPO_UPDATE_PK_NIL: xerrors.NewMetaMessage(
		"cannot update entity without a primary key",
		"",
		[]string{},
	),
	XERR_REPO_UPDATE_UNKNOWN_PK_TYPE: xerrors.NewMetaMessage(
		"unknown primary key type format",
		"",
		[]string{},
	),
	XERR_REPO_UPDATE_NO_COLUMNS_DEFINED: xerrors.NewMetaMessage(
		"no columns defined for update operation in this table",
		"",
		[]string{},
	),
	XERR_REPO_UPDATE_EXEC_FAILED: xerrors.NewMetaMessage(
		"failed to execute update statement in the database",
		"",
		[]string{},
	),
	XERR_REPO_UPDATE_VERIFY_ROWS_FAILED: xerrors.NewMetaMessage(
		"failed to verify affected rows during update operation",
		"",
		[]string{},
	),
	XERR_REPO_UPDATE_RECORD_NOT_FOUND: xerrors.NewMetaMessage(
		"target record not found in database for update operation",
		"",
		[]string{},
	),

	//
	//

	XERR_REPO_DELETE_EXEC_FAILED: xerrors.NewMetaMessage(
		"failed to execute delete statement in the database",
		"",
		[]string{},
	),
	XERR_REPO_DELETE_VERIFY_ROWS_FAILED: xerrors.NewMetaMessage(
		"failed to verify affected rows during delete operation",
		"",
		[]string{},
	),
	XERR_REPO_DELETE_RECORD_NOT_FOUND: xerrors.NewMetaMessage(
		"target record not found in database for delete operation",
		"",
		[]string{},
	),

	//
	//

	XERR_REPO_GET_BY_ID_EXEC_FAILED: xerrors.NewMetaMessage(
		"failed to execute select single record query",
		"",
		[]string{},
	),
	XERR_REPO_GET_BY_ID_RECORD_NOT_FOUND: xerrors.NewMetaMessage(
		"requested record not found in the database table",
		"",
		[]string{},
	),
	XERR_REPO_GET_BY_ID_SCAN_FAILED: xerrors.NewMetaMessage(
		"failed to scan database columns into entity memory pointers",
		"",
		[]string{},
	),
	XERR_REPO_GET_ALL_EXEC_FAILED: xerrors.NewMetaMessage(
		"failed to execute select collection list query",
		"",
		[]string{},
	),
	XERR_REPO_GET_ALL_SCAN_FAILED: xerrors.NewMetaMessage(
		"failed to scan row iteration into entity memory pointers",
		"",
		[]string{},
	),
	XERR_REPO_GET_ALL_ITERATION_FAILED: xerrors.NewMetaMessage(
		"database cursor failure during rows iteration process",
		"",
		[]string{},
	),
	XERR_REPO_GET_BY_FIELD_EXEC_FAILED: xerrors.NewMetaMessage(
		"failed to execute field query statement in the database",
		"",
		[]string{},
	),
	XERR_REPO_GET_BY_FIELD_SCAN_FAILED: xerrors.NewMetaMessage(
		"failed to scan database row into the entity fields",
		"",
		[]string{},
	),
	XERR_REPO_GET_WHERE_ARGS_MISMATCH: xerrors.NewMetaMessage(
		"query parameters count does not match the provided arguments length",
		"",
		[]string{},
	),
	XERR_REPO_GET_WHERE_EXEC_FAILED: xerrors.NewMetaMessage(
		"failed to execute conditional query statement in the database",
		"",
		[]string{},
	),
	XERR_REPO_GET_WHERE_SCAN_FAILED: xerrors.NewMetaMessage(
		"failed to scan conditional database row into the entity fields",
		"",
		[]string{},
	),

	//
	//

	XERR_REPO_QUERY_RAW_EXEC_FAILED: xerrors.NewMetaMessage(
		"failed to execute raw sql query statement in the database",
		"",
		[]string{},
	),
	XERR_REPO_QUERY_RAW_SCAN_FAILED: xerrors.NewMetaMessage(
		"failed to scan database row into the raw query destination structure",
		"",
		[]string{},
	),
}

func init() {
	xerrors.RegisterDomainErrors(XERR_PKGCTX, xerrorDomainMapRegistry)
}
