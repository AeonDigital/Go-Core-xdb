// setup_mock_test.go
package xdb_test

import (
	"database/sql"
	"database/sql/driver"
)

type sqliteDriver struct{}

func (s *sqliteDriver) Open(name string) (driver.Conn, error) { return &fakeConn{}, nil }

type postgresDriver struct{}

func (p *postgresDriver) Open(name string) (driver.Conn, error) { return &fakeConn{}, nil }

type mysqlDriver struct{}

func (m *mysqlDriver) Open(name string) (driver.Conn, error) { return &fakeConn{}, nil }

type fakeConn struct{}

func (f *fakeConn) Prepare(query string) (driver.Stmt, error) { return nil, nil }
func (f *fakeConn) Close() error                              { return nil }
func (f *fakeConn) Begin() (driver.Tx, error)                 { return nil, nil }

func init() {
	sql.Register("mock_sqlite", &sqliteDriver{})
	sql.Register("mock_postgres", &postgresDriver{})
	sql.Register("mock_mysql", &mysqlDriver{})
}
