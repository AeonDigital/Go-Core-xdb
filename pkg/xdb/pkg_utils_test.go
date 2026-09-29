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

func TestGetAllColumnNames(t *testing.T) {
	t.Run("Success with normal entity", func(t *testing.T) {
		entity := &mockEntity{
			pkCol: "id",
			cols:  []string{"name", "api_type"},
		}

		res := xdb.GetAllColumnNames(entity)
		expected := []string{"id", "name", "api_type"}

		if len(res) != 3 {
			t.Fatalf("expected length 3, got %d", len(res))
		}
		for i, v := range res {
			if v != expected[i] {
				t.Errorf("at index %d: expected %s, got %s", i, expected[i], v)
			}
		}
	})

	t.Run("Nil entity handling", func(t *testing.T) {
		res := xdb.GetAllColumnNames(nil)
		if res != nil {
			t.Errorf("expected nil result, got %v", res)
		}
	})

	t.Run("Missing PK column", func(t *testing.T) {
		entity := &mockEntity{
			pkCol: "",
			cols:  []string{"name"},
		}

		res := xdb.GetAllColumnNames(entity)
		if len(res) != 1 || res[0] != "name" {
			t.Errorf("expected ['name'], got %v", res)
		}
	})
}

func TestGetAllColumnValues(t *testing.T) {
	t.Run("Success with normal entity", func(t *testing.T) {
		entity := &mockEntity{
			pkCol: "id",
			pkVal: int64(99),
			vals:  []any{"OpenAI", "rest"},
		}

		res := xdb.GetAllColumnValues(entity)
		if len(res) != 3 {
			t.Fatalf("expected length 3, got %d", len(res))
		}
		if res[0] != int64(99) || res[1] != "OpenAI" || res[2] != "rest" {
			t.Errorf("unexpected dynamic array build: %v", res)
		}
	})

	t.Run("Nil entity handling", func(t *testing.T) {
		res := xdb.GetAllColumnValues(nil)
		if res != nil {
			t.Errorf("expected nil result, got %v", res)
		}
	})

	t.Run("Missing PK column values fallback", func(t *testing.T) {
		entity := &mockEntity{
			pkCol: "",
			vals:  []any{"only_value"},
		}

		res := xdb.GetAllColumnValues(entity)
		if len(res) != 1 || res[0] != "only_value" {
			t.Errorf("expected ['only_value'], got %v", res)
		}
	})
}

func TestConvertEntityAsMap(t *testing.T) {
	t.Run("Success with valid entity matching schema", func(t *testing.T) {
		entity := &mockEntity{
			pkCol: "id",
			pkVal: int64(42),
			cols:  []string{"name"},
			vals:  []any{"OpenAI"},
		}

		res := xdb.ConvertEntityAsMap(entity)

		if res == nil || len(res) != 2 {
			t.Fatalf("expected map length 2, got %v", res)
		}
		if res["id"] != int64(42) || res["name"] != "OpenAI" {
			t.Errorf("map values mismatched: %v", res)
		}
	})

	t.Run("Nil entity handling", func(t *testing.T) {
		res := xdb.ConvertEntityAsMap(nil)
		if res != nil {
			t.Errorf("expected nil result when passing nil entity")
		}
	})

	t.Run("Empty entity schema", func(t *testing.T) {
		entity := &mockEntity{pkCol: "", cols: []string{}}
		res := xdb.ConvertEntityAsMap(entity)
		if res == nil || len(res) != 0 {
			t.Errorf("expected initialized empty map, got %v", res)
		}
	})

	t.Run("Defensive check for mismatched length slices", func(t *testing.T) {
		entity := &mockEntity{
			pkCol: "id",
			pkVal: int64(10),
			cols:  []string{"col1", "col2"}, // 2 extra columns
			vals:  []any{},                  // 0 extra values
		}

		res := xdb.ConvertEntityAsMap(entity)

		// limit rules it to match len(allVals) which is 1 (the PK val)
		if len(res) != 1 {
			t.Errorf("expected map length 1 due to mismatch, got %d", len(res))
		}
		if res["id"] != int64(10) {
			t.Errorf("expected 'id' to track 10, got %v", res["id"])
		}
	})
}
