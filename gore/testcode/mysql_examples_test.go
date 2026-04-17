package testcode

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// ===============================================================================
// MySQL Integration Test Examples
// ===============================================================================

// TestMySQLConnection demonstrates MySQL database connection
func TestMySQLConnection(t *testing.T) {
	dsn := "root:root@tcp(localhost:3307)/gore_test"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	var version string
	err = db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		t.Fatalf("Failed to query version: %v", err)
	}

	t.Logf("MySQL version: %s", version)
}

// TestMySQLCRUD demonstrates Create, Read, Update, Delete operations
func TestMySQLCRUD(t *testing.T) {
	dsn := "root:root@tcp(localhost:3307)/gore_test"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Clean up first
	db.Exec("DELETE FROM users WHERE email = ?", "test@example.com")

	// CREATE - Insert a new user
	result, err := db.Exec(
		"INSERT INTO users (name, email, status) VALUES (?, ?, ?)",
		"Test User", "test@example.com", 1,
	)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	id, _ := result.LastInsertId()
	t.Logf("Inserted user with ID: %d", id)

	// READ - Query the user back
	var name, email string
	var status int
	err = db.QueryRow(
		"SELECT name, email, status FROM users WHERE id = ?",
		id,
	).Scan(&name, &email, &status)
	if err != nil {
		t.Fatalf("Failed to query user: %v", err)
	}

	if name != "Test User" {
		t.Errorf("Expected name 'Test User', got '%s'", name)
	}
	t.Logf("Read user: name=%s, email=%s, status=%d", name, email, status)

	// UPDATE - Modify the user
	_, err = db.Exec(
		"UPDATE users SET name = ?, status = ? WHERE id = ?",
		"Updated User", 2, id,
	)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	err = db.QueryRow(
		"SELECT name, status FROM users WHERE id = ?",
		id,
	).Scan(&name, &status)
	if err != nil {
		t.Fatalf("Failed to query updated user: %v", err)
	}

	if name != "Updated User" {
		t.Errorf("Expected name 'Updated User', got '%s'", name)
	}
	t.Logf("Updated user: name=%s, status=%d", name, status)

	// DELETE - Remove the user
	_, err = db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	var count int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM users WHERE id = ?",
		id,
	).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 users after delete, got %d", count)
	}
	t.Log("User deleted successfully")
}

// TestMySQLQueryPagination demonstrates LIMIT OFFSET pagination
func TestMySQLQueryPagination(t *testing.T) {
	dsn := "root:root@tcp(localhost:3307)/gore_test"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Page 1: 10 records per page
	rows, err := db.Query(
		"SELECT id, name FROM users LIMIT 10 OFFSET 0",
	)
	if err != nil {
		t.Fatalf("Failed to query page 1: %v", err)
	}

	page1Count := 0
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		page1Count++
	}
	rows.Close()
	t.Logf("Page 1: %d records", page1Count)

	// Page 2
	rows, err = db.Query(
		"SELECT id, name FROM users LIMIT 10 OFFSET 10",
	)
	if err != nil {
		t.Fatalf("Failed to query page 2: %v", err)
	}

	page2Count := 0
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		page2Count++
	}
	rows.Close()
	t.Logf("Page 2: %d records", page2Count)
}

// TestMySQLIndexInvisible demonstrates invisible index behavior
func TestMySQLIndexInvisible(t *testing.T) {
	dsn := "root:root@tcp(localhost:3307)/gore_test"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Check if idx_status is invisible
	var isVisible string
	err = db.QueryRow(`
		SELECT IS_VISIBLE
		FROM INFORMATION_SCHEMA.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'users'
		  AND INDEX_NAME = 'idx_status'
	`).Scan(&isVisible)
	if err != nil {
		t.Fatalf("Failed to query index visibility: %v", err)
	}

	t.Logf("idx_status visibility: %s", isVisible)

	if isVisible != "NO" {
		t.Errorf("Expected idx_status to be INVISIBLE (NO), got %s", isVisible)
	}
}

// TestMySQLDescendingIndex demonstrates descending index
func TestMySQLDescendingIndex(t *testing.T) {
	dsn := "root:root@tcp(localhost:3307)/gore_test"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Check idx_name_desc collation
	var collation string
	err = db.QueryRow(`
		SELECT COLLATION
		FROM INFORMATION_SCHEMA.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'users'
		  AND INDEX_NAME = 'idx_name_desc'
		  AND SEQ_IN_INDEX = 1
	`).Scan(&collation)
	if err != nil {
		t.Fatalf("Failed to query index collation: %v", err)
	}

	t.Logf("idx_name_desc collation: %s", collation)

	if collation != "D" {
		t.Errorf("Expected idx_name_desc collation to be D, got %s", collation)
	}
}

// TestMySQLInformationSchema demonstrates INFORMATION_SCHEMA queries
func TestMySQLInformationSchema(t *testing.T) {
	dsn := "root:root@tcp(localhost:3307)/gore_test"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Query all tables
	rows, err := db.Query(`
		SELECT TABLE_NAME
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME
	`)
	if err != nil {
		t.Fatalf("Failed to query tables: %v", err)
	}

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			t.Fatalf("Failed to scan table name: %v", err)
		}
		tables = append(tables, tableName)
	}
	rows.Close()

	t.Logf("Tables: %v", tables)

	// Query all columns for users table
	rows, err = db.Query(`
		SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, EXTRA
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'users'
		ORDER BY ORDINAL_POSITION
	`)
	if err != nil {
		t.Fatalf("Failed to query columns: %v", err)
	}

	for rows.Next() {
		var colName, dataType, nullable, extra string
		if err := rows.Scan(&colName, &dataType, &nullable, &extra); err != nil {
			t.Fatalf("Failed to scan column: %v", err)
		}
		t.Logf("Column: name=%s, type=%s, nullable=%s, extra=%s",
			colName, dataType, nullable, extra)
	}
	rows.Close()

	// Query all indexes for users table
	rows, err = db.Query(`
		SELECT INDEX_NAME, COLUMN_NAME, SEQ_IN_INDEX, COLLATION, IS_VISIBLE
		FROM INFORMATION_SCHEMA.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'users'
		ORDER BY INDEX_NAME, SEQ_IN_INDEX
	`)
	if err != nil {
		t.Fatalf("Failed to query indexes: %v", err)
	}

	for rows.Next() {
		var idxName, colName, collation, visible string
		var seq int
		if err := rows.Scan(&idxName, &colName, &seq, &collation, &visible); err != nil {
			t.Fatalf("Failed to scan index: %v", err)
		}
		t.Logf("Index: name=%s, column=%s, seq=%d, collation=%s, visible=%s",
			idxName, colName, seq, collation, visible)
	}
	rows.Close()
}
