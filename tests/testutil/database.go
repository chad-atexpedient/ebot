// Package testutil provides shared testing utilities
package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3" // SQLite driver for testing
)

// TestDB provides a test database instance
type TestDB struct {
	DB       *sql.DB
	Name     string
	mu       sync.Mutex
	cleanups []func()
}

// NewTestDB creates a new test database (SQLite in-memory by default)
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	testDB := &TestDB{
		DB:       db,
		Name:     "test_db",
		cleanups: []func(){},
	}

	// Register cleanup
	t.Cleanup(func() {
		testDB.Close()
	})

	return testDB
}

// NewTestDBWithFile creates a test database backed by a file
func NewTestDBWithFile(t *testing.T, filename string) *TestDB {
	t.Helper()

	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	testDB := &TestDB{
		DB:   db,
		Name: filename,
		cleanups: []func(){
			func() { os.Remove(filename) },
		},
	}

	t.Cleanup(func() {
		testDB.Close()
	})

	return testDB
}

// Close closes the database and runs cleanups
func (db *TestDB) Close() {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.DB != nil {
		db.DB.Close()
		db.DB = nil
	}

	for _, cleanup := range db.cleanups {
		cleanup()
	}
}

// Exec executes a query without returning results
func (db *TestDB) Exec(t *testing.T, query string, args ...interface{}) sql.Result {
	t.Helper()

	result, err := db.DB.Exec(query, args...)
	if err != nil {
		t.Fatalf("Failed to execute query: %v\nQuery: %s", err, query)
	}
	return result
}

// Query executes a query and returns rows
func (db *TestDB) Query(t *testing.T, query string, args ...interface{}) *sql.Rows {
	t.Helper()

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		t.Fatalf("Failed to execute query: %v\nQuery: %s", err, query)
	}
	return rows
}

// QueryRow executes a query expecting a single row
func (db *TestDB) QueryRow(t *testing.T, query string, args ...interface{}) *sql.Row {
	t.Helper()
	return db.DB.QueryRow(query, args...)
}

// CreateTable creates a table from schema
func (db *TestDB) CreateTable(t *testing.T, schema string) {
	t.Helper()
	db.Exec(t, schema)
}

// DropTable drops a table
func (db *TestDB) DropTable(t *testing.T, tableName string) {
	t.Helper()
	db.Exec(t, fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
}

// TruncateTable truncates a table
func (db *TestDB) TruncateTable(t *testing.T, tableName string) {
	t.Helper()
	db.Exec(t, fmt.Sprintf("DELETE FROM %s", tableName))
}

// Count returns the number of rows in a table
func (db *TestDB) Count(t *testing.T, tableName string) int {
	t.Helper()

	var count int
	row := db.QueryRow(t, fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName))
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}
	return count
}

// InsertRow inserts a row and returns the ID
func (db *TestDB) InsertRow(t *testing.T, query string, args ...interface{}) int64 {
	t.Helper()

	result := db.Exec(t, query, args...)
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last insert ID: %v", err)
	}
	return id
}

// RunInTransaction runs a function in a transaction
func (db *TestDB) RunInTransaction(t *testing.T, fn func(tx *sql.Tx) error) {
	t.Helper()

	tx, err := db.DB.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		t.Fatalf("Transaction failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}
}

// StandardSchemas provides common table schemas for testing

// UserTableSchema returns the schema for a users table
func UserTableSchema() string {
	return `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			name TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
}

// WorkspaceTableSchema returns the schema for a workspaces table
func WorkspaceTableSchema() string {
	return `
		CREATE TABLE IF NOT EXISTS workspaces (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			owner_id INTEGER REFERENCES users(id),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
}

// AuditLogTableSchema returns the schema for an audit log table
func AuditLogTableSchema() string {
	return `
		CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			action TEXT NOT NULL,
			resource_type TEXT,
			resource_id TEXT,
			details TEXT,
			ip_address TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
}

// PolicyTableSchema returns the schema for a policies table
func PolicyTableSchema() string {
	return `
		CREATE TABLE IF NOT EXISTS policies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			effect TEXT NOT NULL,
			condition TEXT,
			enabled INTEGER DEFAULT 1,
			priority INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
}

// TokenTableSchema returns the schema for a token vault table
func TokenTableSchema() string {
	return `
		CREATE TABLE IF NOT EXISTS token_vault (
			token TEXT PRIMARY KEY,
			encrypted_value BLOB NOT NULL,
			key_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME,
			access_count INTEGER DEFAULT 0
		)
	`
}

// SetupTestTables creates all standard test tables
func (db *TestDB) SetupTestTables(t *testing.T) {
	t.Helper()

	schemas := []string{
		UserTableSchema(),
		WorkspaceTableSchema(),
		AuditLogTableSchema(),
		PolicyTableSchema(),
		TokenTableSchema(),
	}

	for _, schema := range schemas {
		db.CreateTable(t, schema)
	}
}

// SeedTestData populates tables with sample data
func (db *TestDB) SeedTestData(t *testing.T) {
	t.Helper()

	// Seed users
	db.Exec(t, `INSERT INTO users (email, name) VALUES ('admin@test.com', 'Admin User')`)
	db.Exec(t, `INSERT INTO users (email, name) VALUES ('user@test.com', 'Regular User')`)

	// Seed workspaces
	db.Exec(t, `INSERT INTO workspaces (name, owner_id) VALUES ('Default Workspace', 1)`)
	db.Exec(t, `INSERT INTO workspaces (name, owner_id) VALUES ('Team Workspace', 2)`)
}

// AssertRowExists checks if a row exists in a table
func (db *TestDB) AssertRowExists(t *testing.T, table, whereClause string, args ...interface{}) {
	t.Helper()

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, whereClause)
	var count int
	if err := db.QueryRow(t, query, args...).Scan(&count); err != nil {
		t.Fatalf("Failed to check row existence: %v", err)
	}

	if count == 0 {
		t.Errorf("Expected row to exist in %s WHERE %s", table, whereClause)
	}
}

// AssertRowNotExists checks if a row does not exist
func (db *TestDB) AssertRowNotExists(t *testing.T, table, whereClause string, args ...interface{}) {
	t.Helper()

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, whereClause)
	var count int
	if err := db.QueryRow(t, query, args...).Scan(&count); err != nil {
		t.Fatalf("Failed to check row existence: %v", err)
	}

	if count > 0 {
		t.Errorf("Expected no row in %s WHERE %s, but found %d", table, whereClause, count)
	}
}

// AssertTableCount checks the number of rows in a table
func (db *TestDB) AssertTableCount(t *testing.T, table string, expected int) {
	t.Helper()

	actual := db.Count(t, table)
	if actual != expected {
		t.Errorf("Expected %d rows in %s, got %d", expected, table, actual)
	}
}
