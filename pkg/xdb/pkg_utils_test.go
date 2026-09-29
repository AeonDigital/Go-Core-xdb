package xdb_test

import (
	"database/sql"
	"testing"

	"github.com/AeonDigital/Go-Core-xdb/pkg/xdb"
	"github.com/AeonDigital/Go-Core-xerrors/pkg/xerrors"
)

func TestRetrieveDbType(t *testing.T) {
	t.Run("Retorna DB se o ponteiro for nil", func(t *testing.T) {
		res := xdb.RetrieveDbType(nil)
		if res != "DB" {
			t.Errorf("esperava 'DB', recebeu '%s'", res)
		}
	})

	t.Run("Detecta driver SQLite", func(t *testing.T) {
		db, _ := sql.Open("mock_sqlite", "dsn")
		defer db.Close()

		res := xdb.RetrieveDbType(db)
		if res != "sqlite" {
			t.Errorf("esperava 'sqlite', recebeu '%s'", res)
		}
	})

	t.Run("Detecta driver Postgres", func(t *testing.T) {
		db, _ := sql.Open("mock_postgres", "dsn")
		defer db.Close()

		res := xdb.RetrieveDbType(db)
		if res != "postgres" {
			t.Errorf("esperava 'postgres', recebeu '%s'", res)
		}
	})

	t.Run("Detecta driver MySQL", func(t *testing.T) {
		db, _ := sql.Open("mock_mysql", "dsn")
		defer db.Close()

		res := xdb.RetrieveDbType(db)
		if res != "mysql" {
			t.Errorf("esperava 'mysql', recebeu '%s'", res)
		}
	})
}

func TestBuildTruncateQuery(t *testing.T) {
	t.Run("Usa DELETE sem WHERE para sqlite", func(t *testing.T) {
		res := xdb.BuildTruncateQuery("sqlite", "users")
		if res != "DELETE FROM users;" {
			t.Errorf("esperava 'DELETE FROM users;', recebeu '%s'", res)
		}
	})

	t.Run("Usa TRUNCATE TABLE para postgres", func(t *testing.T) {
		res := xdb.BuildTruncateQuery("postgres", "users")
		if res != "TRUNCATE TABLE users;" {
			t.Errorf("esperava 'TRUNCATE TABLE users;', recebeu '%s'", res)
		}
	})

	t.Run("Usa TRUNCATE TABLE para mysql", func(t *testing.T) {
		res := xdb.BuildTruncateQuery("mysql", "users")
		if res != "TRUNCATE TABLE users;" {
			t.Errorf("esperava 'TRUNCATE TABLE users;', recebeu '%s'", res)
		}
	})

	t.Run("Usa DELETE sem WHERE como padrao para dialeto desconhecido", func(t *testing.T) {
		res := xdb.BuildTruncateQuery("DB", "users")
		if res != "DELETE FROM users;" {
			t.Errorf("esperava 'DELETE FROM users;', recebeu '%s'", res)
		}
	})
}

// mockEntity implements the Entity interface for testing purposes
type mockEntity struct {
	pkCol string
	pkVal any
	cols  []string
	vals  []any
}

func (m *mockEntity) TableName() string                   { return "mock_table" }
func (m *mockEntity) PKColumnName() string                { return m.pkCol }
func (m *mockEntity) PKSetValue(id any)                   {}
func (m *mockEntity) PKGetValue() any                     { return m.pkVal }
func (m *mockEntity) PKGenerateValue() any                { return nil }
func (m *mockEntity) PKExternal() bool                    { return false }
func (m *mockEntity) Columns() []string                   { return m.cols }
func (m *mockEntity) Values() []any                       { return m.vals }
func (m *mockEntity) ScanRow(rows *sql.Rows) error        { return nil }
func (m *mockEntity) Normalize()                          {}
func (m *mockEntity) Validate() (bool, xerrors.ErrorCode) { return true, "" }

func TestConvertToMap(t *testing.T) {
	t.Run("Success with valid entity", func(t *testing.T) {
		entity := &mockEntity{
			pkCol: "id",
			pkVal: int64(42),
			cols:  []string{"name", "api_type"},
			vals:  []any{"OpenAI", "rest"},
		}

		res := xdb.ConvertToMap(entity)

		if res == nil {
			t.Fatalf("expected map, got nil")
		}
		if len(res) != 3 {
			t.Errorf("expected map length 3, got %d", len(res))
		}
		if res["id"] != int64(42) {
			t.Errorf("expected id to be 42, got %v", res["id"])
		}
		if res["name"] != "OpenAI" {
			t.Errorf("expected name to be 'OpenAI', got %v", res["name"])
		}
		if res["api_type"] != "rest" {
			t.Errorf("expected api_type to be 'rest', got %v", res["api_type"])
		}
	})

	t.Run("Nil entity handling", func(t *testing.T) {
		res := xdb.ConvertToMap(nil)
		if res != nil {
			t.Errorf("expected nil result when passing nil entity, got %v", res)
		}
	})

	t.Run("Missing PK column name", func(t *testing.T) {
		entity := &mockEntity{
			pkCol: "", // No PK column
			pkVal: nil,
			cols:  []string{"name"},
			vals:  []any{"OnlyName"},
		}

		res := xdb.ConvertToMap(entity)

		if len(res) != 1 {
			t.Errorf("expected map length 1, got %d", len(res))
		}
		if _, exists := res[""]; exists {
			t.Errorf("did not expect empty string key to be injected")
		}
		if res["name"] != "OnlyName" {
			t.Errorf("expected name to be 'OnlyName', got %v", res["name"])
		}
	})

	t.Run("Defensive check for mismatched slice sizes (fewer values than columns)", func(t *testing.T) {
		entity := &mockEntity{
			pkCol: "id",
			pkVal: int64(10),
			cols:  []string{"col1", "col2", "col3"}, // 3 columns
			vals:  []any{"val1"},                    // only 1 value
		}

		res := xdb.ConvertToMap(entity)

		// Expecting PK + 1 value = 2 items total
		if len(res) != 2 {
			t.Errorf("expected map length 2, got %d", len(res))
		}
		if res["col1"] != "val1" {
			t.Errorf("expected col1 to be 'val1', got %v", res["col1"])
		}
		if _, exists := res["col2"]; exists {
			t.Errorf("did not expect col2 to be present due to missing matching value")
		}
	})
}
