package xdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/AeonDigital/Go-Core-xerrors/pkg/xerrors"
	_ "modernc.org/sqlite"
)

// DBConfig manages database driver options, connection pool life cycles, and engine-specific options.
type DBConfig struct {
	Driver                string        `json:"driver"`
	DSN                   string        `json:"dsn"`
	MigrationsDirPath     string        `json:"migrationsDirPath"`
	MaxOpenConnections    int           `json:"maxOpenConnections"`
	MaxIdleConnections    int           `json:"maxIdleConnections"`
	ConnectionMaxLifetime time.Duration `json:"connectionMaxLifetime"`
	SQLite                SQLiteConfig  `json:"sqlite"`

	DB *sql.DB
}

// SQLiteConfig aggregates dedicated attributes and behavior modifiers needed to shape SQLite behavior.
type SQLiteConfig struct {
	Mode        string            `json:"mode"`
	Dir         string            `json:"dir"`
	FileName    string            `json:"fileName"`
	QueryString string            `json:"querystring"`
	Pragma      map[string]string `json:"pragma"`
}

// NewDBConfig instantiates a baseline operational configuration setup with optimized defaults for embedded deployments.
func NewDBConfig(
	driver string,
	dsn string,
	migrationsDirPath string,
	sqliteMode string,
	sqliteDir string,
	sqliteFileName string,
	sqliteQuerystring string,
) DBConfig {
	sqlite := SQLiteConfig{
		Mode:        sqliteMode,
		Dir:         sqliteDir,
		FileName:    sqliteFileName,
		QueryString: sqliteQuerystring,

		Pragma: map[string]string{
			"journal_mode": "WAL",
			"synchronous":  "NORMAL",
			"busy_timeout": "5000",
			"foreign_keys": "ON",
		},
	}

	dbCfg := DBConfig{
		Driver:                driver,
		DSN:                   dsn,
		MigrationsDirPath:     migrationsDirPath,
		MaxOpenConnections:    1,
		MaxIdleConnections:    1,
		ConnectionMaxLifetime: 30 * time.Minute,
		SQLite:                sqlite,
	}

	dbCfg.buildDSN()

	return dbCfg
}

// buildDSN synthesizes target variables and environments to compile a clean connection URI sequence.
func (o *DBConfig) buildDSN() {
	if o.DSN != "" {
		o.SQLite.Mode = ""
		o.SQLite.Dir = ""
		o.SQLite.FileName = ""
		o.SQLite.QueryString = ""
		return
	}

	o.Driver = "sqlite"

	// Compiles and forces critical SQLite tuning parameters directly into o.SQLite.QueryString
	o.compileSQLitePragmas()

	mode := strings.TrimSpace(o.SQLite.Mode)
	if mode == "" {
		mode = "file:"
	}

	if mode == ":memory:" || mode == "file::memory:" {
		if o.SQLite.QueryString != "" {
			if mode == "file::memory:" {
				o.DSN = mode + "?" + o.SQLite.QueryString
				return
			}
			o.DSN = "file::memory:?" + o.SQLite.QueryString
			return
		}
		o.DSN = mode
		return
	}

	if o.SQLite.Dir == "" || o.SQLite.Dir == "." || o.SQLite.FileName == "" {
		return
	}

	fullPath := filepath.Join(o.SQLite.Dir, o.SQLite.FileName)
	dsn := fmt.Sprintf("file:%s", filepath.ToSlash(fullPath))

	if o.SQLite.QueryString != "" {
		dsn = fmt.Sprintf("%s?%s", dsn, o.SQLite.QueryString)
	}

	o.DSN = dsn
}

// compileSQLitePragmas normalizes defaults and appends SQLite custom configuration settings directly into the QueryString.
func (o *DBConfig) compileSQLitePragmas() {
	requiredDefaults := map[string]string{
		"journal_mode": "WAL",
		"synchronous":  "NORMAL",
		"busy_timeout": "5000",
		"foreign_keys": "ON",
	}

	if o.SQLite.Pragma == nil {
		o.SQLite.Pragma = make(map[string]string)
	}

	for key, defaultValue := range requiredDefaults {
		if _, exists := o.SQLite.Pragma[key]; !exists {
			o.SQLite.Pragma[key] = defaultValue
		}
	}

	// Order keys alphabetically to guarantee a deterministic DSN generation order
	var keys []string
	for key := range o.SQLite.Pragma {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var pragmaParams []string
	for _, key := range keys {
		// modernc.org/sqlite driver format syntax requirement: _pragma=key(value)
		pragmaParams = append(pragmaParams, fmt.Sprintf("_pragma=%s(%s)", key, o.SQLite.Pragma[key]))
	}

	compiledPragmas := strings.Join(pragmaParams, "&")
	if o.SQLite.QueryString == "" {
		o.SQLite.QueryString = compiledPragmas
		return
	}

	// Integrates existing custom user queries while safely compounding the calculated pragmas
	if !strings.Contains(o.SQLite.QueryString, "_pragma=") {
		o.SQLite.QueryString = o.SQLite.QueryString + "&" + compiledPragmas
	}
}

// CheckConfiguration verifies structural parameters to maintain setup layout consistency before initializing adapters.
func (o *DBConfig) CheckConfiguration() error {
	o.buildDSN()
	return nil
}

// InitDataBaseConnection activates database interface structures and applies custom engine tuning options safely.
func (o *DBConfig) InitDataBaseConnection(ctx context.Context) error {
	db, err := sql.Open(o.Driver, o.DSN)
	if err != nil {
		return xerrors.NewError500(
			XERR_PKGCTX,
			XERR_CONNECTION_FAILED,
			err,
			"",
			"",
		).WithArgs(
			o.Driver,
			"failed to open channel wrapper",
		)
	}

	db.SetMaxOpenConns(o.MaxOpenConnections)
	db.SetMaxIdleConns(o.MaxIdleConnections)
	db.SetConnMaxLifetime(o.ConnectionMaxLifetime)

	if strings.Contains(o.Driver, "sqlite") {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := db.PingContext(pingCtx); err != nil {
			db.Close()
			return xerrors.NewError500(
				XERR_PKGCTX,
				XERR_CONNECTION_FAILED,
				err,
				"",
				"",
			).WithArgs(
				o.DSN,
				"liveness ping execution timeout boundary breached",
			)
		}
	}

	o.DB = db
	return err
}

// RunMigrations processes organized script collections chronologically against target database environments.
func (o *DBConfig) RunMigrations(ctx context.Context) error {
	if o.DB == nil {
		return xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_NIL_NOT_ALLOWED,
			nil,
			"",
			"",
		).WithArgs(
			"DB",
		)
	}

	dirPath := strings.TrimSpace(o.MigrationsDirPath)
	if dirPath == "" {
		return xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_EMPTY_NOT_ALLOWED,
			nil,
			"",
			"",
		).WithArgs(
			"MigrationsDirPath",
		)
	}

	dirInfo, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_RESOURCE_NOT_FOUND,
			err,
			"",
			"",
		).WithArgs(
			"MigrationsDirPath",
			dirPath,
		)
	}
	if !dirInfo.IsDir() {
		return xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_INVALID_TYPE,
			nil,
			"",
			"",
		).WithArgs(
			"MigrationsDirPath",
			fmt.Sprintf("path points to structural file instead of directory: %s", dirPath),
		)
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_RESOURCE_UNAVAILABLE,
			err,
			"",
			"",
		).WithArgs(
			dirPath,
			"unreadable system directory descriptors descriptor slice",
		)
	}

	var migrationFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".sql") {
			migrationFiles = append(migrationFiles, entry.Name())
		}
	}

	sort.Strings(migrationFiles)

	if len(migrationFiles) == 0 {
		return nil
	}

	for _, fileName := range migrationFiles {
		fullFilePath := filepath.Join(dirPath, fileName)

		scriptBytes, err := os.ReadFile(fullFilePath)
		if err != nil {
			return xerrors.NewError500(
				XERR_PKGCTX,
				xerrors.XERR_RESOURCE_CORRUPTED,
				err,
				"",
				"",
			).WithArgs(
				fileName,
				"failed to stream linear script payload stream bytes",
			)
		}

		scriptSQL := string(scriptBytes)
		if strings.TrimSpace(scriptSQL) == "" {
			continue
		}

		tx, err := o.DB.BeginTx(ctx, nil)
		if err != nil {
			return xerrors.NewError500(
				XERR_PKGCTX,
				XERR_MIGRATION_EXECUTION_FAILED,
				err,
				"",
				"",
			).WithArgs(
				fileName,
				"failed to begin database transaction block",
			)
		}

		if _, err = tx.ExecContext(ctx, scriptSQL); err != nil {
			tx.Rollback()
			return xerrors.NewError500(
				XERR_PKGCTX,
				XERR_MIGRATION_EXECUTION_FAILED,
				err,
				"",
				"",
			).WithArgs(
				fileName,
				"failed transaction command context lifecycle validation execution block",
			)
		}

		if err = tx.Commit(); err != nil {
			return xerrors.NewError500(
				XERR_PKGCTX,
				XERR_MIGRATION_EXECUTION_FAILED,
				err,
				"",
				"",
			).WithArgs(
				fileName,
				"failed to commit finalized transaction statements",
			)
		}
	}

	return nil
}
