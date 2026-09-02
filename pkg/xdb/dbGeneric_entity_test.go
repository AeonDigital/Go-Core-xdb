package xdb_test

import (
	"database/sql"
	"errors"

	"github.com/AeonDigital/Go-Core-xdb/pkg/xdb"
	"github.com/AeonDigital/Go-Core-xerrors/pkg/xerrors"
)

// MockUser represents a dummy entity used to validate all repository logical paths.
type MockUser struct {
	ID        int64
	UUID      string
	Name      string
	Email     string
	IsNatural bool
}

// Normalize satisfies the Entity interface by cleaning up internal fields.
func (m *MockUser) Normalize() {
	// Simple normalization simulation
	if m.Name == "  Trim Me  " {
		m.Name = "Trim Me"
	}
}

// Validate satisfies the Entity interface by triggering dummy validation rules.
func (m *MockUser) Validate() (bool, xerrors.ErrorCode) {
	if m.Name == "INVALID" {
		return false, "E9999" // Custom simulation error
	}
	return true, xdb.XERR_NONE
}

// TableName returns the target dummy table name.
func (m *MockUser) TableName() string {
	return "mock_users"
}

// Columns returns the table columns excluding database auto-generated ones.
func (m *MockUser) Columns() []string {
	if m.Name == "FORCE_EMPTY_COLUMNS" {
		return []string{}
	}
	if m.UUID != "" || m.IsNatural {
		if m.UUID != "" {
			return []string{"uuid", "name", "email"}
		}
		return []string{"id", "name", "email"}
	}
	return []string{"name", "email"}
}

// Values returns the field values matching the strict order of Columns().
func (m *MockUser) Values() []any {
	if m.Name == "FORCE_EMPTY_COLUMNS" {
		return []any{}
	}
	if m.UUID != "" || m.IsNatural {
		return []any{m.PKValue(), m.Name, m.Email}
	}
	return []any{m.Name, m.Email}
}

// TablePK identifies the target primary key column name.
func (m *MockUser) TablePK() string {
	if m.UUID != "" || m.Name == "App Generated User" {
		return "uuid"
	}
	return "id"
}

// BindPK injects runtime generated keys back into the entity structure.
func (m *MockUser) BindPK(id any) {
	switch v := id.(type) {
	case int64:
		m.ID = v
	case string:
		m.UUID = v
	}
}

// PKValue yields the current primary key snapshot value.
func (m *MockUser) PKValue() any {
	if m.Name == "FORCE_UNKNOWN_PK_TYPE" {
		return float64(1.23)
	}
	if m.UUID == "TRIGGER_GENERATION" {
		return nil
	}
	if m.UUID != "" {
		return m.UUID
	}

	if m.Name == "Explicit Zero User" || m.Name == "No ID" {
		return int64(0)
	}

	if m.ID != 0 {
		return m.ID
	}
	return nil
}

// ScanRow hydrats the entity boundaries from active database cursor rows.
func (m *MockUser) ScanRow(rows *sql.Rows) error {
	var dummyUUID any
	if m.UUID != "" {
		var dummyID int64
		return rows.Scan(&dummyID, &m.UUID, &m.Name, &m.Email)
	}

	err := rows.Scan(&m.ID, &dummyUUID, &m.Name, &m.Email)
	if err != nil {
		return err
	}

	if m.Name == "FORCE_SCAN_FAILURE" {
		return errors.New("simulated database scan error")
	}

	return nil
}

// GeneratePK simulates application-side key generation (e.g., UUIDv7).
func (m *MockUser) GeneratePK() any {
	if m.UUID == "" && m.Name == "App Generated User" {
		return "GENERATED-UUID-123"
	}
	return nil
}

// IsNaturalPK informs the repository if the key configuration relies on external values.
func (m *MockUser) IsNaturalPK() bool {
	return m.IsNatural
}
