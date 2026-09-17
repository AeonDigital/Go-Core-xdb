package xdb_test

import (
	"database/sql"
	"testing"

	"github.com/AeonDigital/Go-Core-xdb/pkg/xdb"
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
