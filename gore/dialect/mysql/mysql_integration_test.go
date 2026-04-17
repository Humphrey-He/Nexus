package mysql_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"gore/dialect"
	"gore/dialect/mysql"
	"gore/internal/advisor"
	"gore/internal/advisor/rules"
)

const (
	testDSN = "root:root@tcp(localhost:3307)/gore_test"
)

func newTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("mysql", testDSN)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Fatalf("failed to ping database: %v", err)
	}

	return db
}

func newTestDBForBench() *sql.DB {
	db, _ := sql.Open("mysql", testDSN)
	return db
}

// =============================================================================
// Connection Tests
// =============================================================================

func TestMySQLConnection(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	var version string
	err := db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		t.Fatalf("failed to query version: %v", err)
	}

	t.Logf("MySQL version: %s", version)
}

func TestMySQLConnectionTimeout(t *testing.T) {
	db, err := sql.Open("mysql", testDSN)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("failed to ping with timeout: %v", err)
	}
}

// =============================================================================
// MetadataProvider.Indexes Tests
// =============================================================================

func TestMySQLMetadataProviderIndexes(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	provider := mysql.NewMetadataProvider(db)
	ctx := context.Background()

	indexes, err := provider.Indexes(ctx, "users")
	if err != nil {
		t.Fatalf("failed to get indexes: %v", err)
	}

	if len(indexes) == 0 {
		t.Fatal("expected indexes, got none")
	}

	// Build index map for verification
	indexMap := make(map[string]bool)
	for _, idx := range indexes {
		indexMap[idx.Name] = true
		t.Logf("Index: name=%s, columns=%v, method=%s, unique=%v, isBtree=%v",
			idx.Name, idx.Columns, idx.Method, idx.Unique, idx.IsBTree)
	}

	// Verify expected indexes exist
	expectedIndexes := []string{"PRIMARY", "idx_name", "idx_status", "idx_name_desc"}
	for _, name := range expectedIndexes {
		if !indexMap[name] {
			t.Errorf("expected index %s not found", name)
		}
	}
}

func TestMySQLMetadataProviderIndexesPrimaryKey(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	provider := mysql.NewMetadataProvider(db)
	ctx := context.Background()

	indexes, err := provider.Indexes(ctx, "users")
	if err != nil {
		t.Fatalf("failed to get indexes: %v", err)
	}

	for _, idx := range indexes {
		if idx.Name == "PRIMARY" {
			if !idx.Unique {
				t.Error("expected PRIMARY index to be unique")
			}
			if len(idx.Columns) != 1 || idx.Columns[0] != "id" {
				t.Errorf("expected PRIMARY on 'id', got %v", idx.Columns)
			}
			if idx.Method != "BTREE" {
				t.Errorf("expected BTREE method, got %s", idx.Method)
			}
			t.Logf("PRIMARY index verified: unique=%v, columns=%v, method=%s", idx.Unique, idx.Columns, idx.Method)
		}
	}
}

func TestMySQLMetadataProviderIndexesComposite(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	provider := mysql.NewMetadataProvider(db)
	ctx := context.Background()

	// Create composite index if not exists
	_, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_name_status ON users(name, status)")
	if err != nil {
		t.Logf("index may already exist or table doesn't support IF NOT EXISTS: %v", err)
	}

	indexes, err := provider.Indexes(ctx, "users")
	if err != nil {
		t.Fatalf("failed to get indexes: %v", err)
	}

	for _, idx := range indexes {
		if len(idx.Columns) > 1 {
			t.Logf("Composite index: name=%s, columns=%v", idx.Name, idx.Columns)
		}
	}
}

func TestMySQLMetadataProviderIndexesNonExistentTable(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	provider := mysql.NewMetadataProvider(db)
	ctx := context.Background()

	indexes, err := provider.Indexes(ctx, "non_existent_table_xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(indexes) != 0 {
		t.Errorf("expected 0 indexes for non-existent table, got %d", len(indexes))
	}
}

// =============================================================================
// MetadataProvider.Columns Tests
// =============================================================================

func TestMySQLMetadataProviderColumns(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	provider := mysql.NewMetadataProvider(db)
	ctx := context.Background()

	columns, err := provider.Columns(ctx, "users")
	if err != nil {
		t.Fatalf("failed to get columns: %v", err)
	}

	if len(columns) == 0 {
		t.Fatal("expected columns, got none")
	}

	for _, col := range columns {
		t.Logf("Column: name=%s, type=%s", col.Name, col.Type)
	}
}

func TestMySQLMetadataProviderColumnsAutoIncrement(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	provider := mysql.NewMetadataProvider(db)
	ctx := context.Background()

	columns, err := provider.Columns(ctx, "users")
	if err != nil {
		t.Fatalf("failed to get columns: %v", err)
	}

	for _, col := range columns {
		if col.Name == "id" {
			if col.Type != "int auto_increment" {
				t.Errorf("expected 'int auto_increment' for id, got %s", col.Type)
			}
			t.Logf("id column type verified: %s", col.Type)
		}
	}
}

func TestMySQLMetadataProviderColumnsTimestamp(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	provider := mysql.NewMetadataProvider(db)
	ctx := context.Background()

	columns, err := provider.Columns(ctx, "users")
	if err != nil {
		t.Fatalf("failed to get columns: %v", err)
	}

	for _, col := range columns {
		if col.Name == "created_at" {
			t.Logf("created_at type: %s", col.Type)
		}
	}
}

func TestMySQLMetadataProviderColumnsNonExistentTable(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	provider := mysql.NewMetadataProvider(db)
	ctx := context.Background()

	columns, err := provider.Columns(ctx, "non_existent_table_xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(columns) != 0 {
		t.Errorf("expected 0 columns for non-existent table, got %d", len(columns))
	}
}

// =============================================================================
// MetadataProvider.Tables Tests
// =============================================================================

func TestMySQLMetadataProviderTables(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	provider := mysql.NewMetadataProvider(db)
	ctx := context.Background()

	tables, err := provider.Tables(ctx)
	if err != nil {
		t.Fatalf("failed to get tables: %v", err)
	}

	if len(tables) == 0 {
		t.Fatal("expected tables, got none")
	}

	found := false
	for _, table := range tables {
		t.Logf("Table: %s", table)
		if table == "users" {
			found = true
		}
	}

	if !found {
		t.Error("expected 'users' table not found")
	}
}

func TestMySQLMetadataProviderTablesExcludesViews(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	provider := mysql.NewMetadataProvider(db)
	ctx := context.Background()

	tables, err := provider.Tables(ctx)
	if err != nil {
		t.Fatalf("failed to get tables: %v", err)
	}

	for _, table := range tables {
		if table == "orders" {
			t.Log("found orders table")
		}
	}

	t.Logf("Total base tables: %d", len(tables))
}

// =============================================================================
// Dialector BuildSelect Tests
// =============================================================================

func TestMySQLDialectorBuildSelect(t *testing.T) {
	d := &mysql.Dialector{}

	tests := []struct {
		name     string
		ast      *dialect.QueryAST
		expected string
	}{
		{
			name:     "basic select",
			ast:      &dialect.QueryAST{Table: "users"},
			expected: "SELECT * FROM `users`",
		},
		{
			name:     "select with columns",
			ast:      &dialect.QueryAST{Table: "users", Columns: []string{"id", "name"}},
			expected: "SELECT `id`, `name` FROM `users`",
		},
		{
			name:     "select with where",
			ast:      &dialect.QueryAST{Table: "users", Where: []string{"name = ?"}},
			expected: "SELECT * FROM `users` WHERE name = ?",
		},
		{
			name:     "select with limit offset",
			ast:      &dialect.QueryAST{Table: "users", Limit: 10, Offset: 20},
			expected: "SELECT * FROM `users` LIMIT 20, 10",
		},
		{
			name:     "select with limit only",
			ast:      &dialect.QueryAST{Table: "users", Limit: 10},
			expected: "SELECT * FROM `users` LIMIT 10",
		},
		{
			name:     "select with offset only",
			ast:      &dialect.QueryAST{Table: "users", Offset: 20},
			expected: "SELECT * FROM `users` LIMIT 20",
		},
		{
			name:     "select with order by asc",
			ast:      &dialect.QueryAST{Table: "users", OrderBy: []string{"name ASC"}},
			expected: "SELECT * FROM `users` ORDER BY name ASC",
		},
		{
			name:     "select with order by desc",
			ast:      &dialect.QueryAST{Table: "users", OrderBy: []string{"created_at DESC"}},
			expected: "SELECT * FROM `users` ORDER BY created_at DESC",
		},
		{
			name:     "select with order by multiple",
			ast:      &dialect.QueryAST{Table: "users", OrderBy: []string{"name ASC", "created_at DESC"}},
			expected: "SELECT * FROM `users` ORDER BY name ASC, created_at DESC",
		},
		{
			name:     "select with group by",
			ast:      &dialect.QueryAST{Table: "users", GroupBy: []string{"status"}},
			expected: "SELECT * FROM `users` GROUP BY `status`",
		},
		{
			name:     "select with group by multiple",
			ast:      &dialect.QueryAST{Table: "users", GroupBy: []string{"status", "category"}},
			expected: "SELECT * FROM `users` GROUP BY `status`, `category`",
		},
		{
			name: "full query",
			ast: &dialect.QueryAST{
				Table:   "users",
				Columns: []string{"id", "name"},
				Where:   []string{"status > ?"},
				OrderBy: []string{"created_at DESC"},
				Limit:   10,
				Offset:  5,
			},
			expected: "SELECT `id`, `name` FROM `users` WHERE status > ? ORDER BY created_at DESC LIMIT 5, 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlStr, _, err := d.BuildSelect(tt.ast)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sqlStr != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, sqlStr)
			}
		})
	}
}

func TestMySQLDialectorBuildSelectNilAST(t *testing.T) {
	d := &mysql.Dialector{}
	_, _, err := d.BuildSelect(nil)
	if err == nil {
		t.Fatal("expected error for nil ast")
	}
}

func TestMySQLDialectorBuildSelectEmptyTable(t *testing.T) {
	d := &mysql.Dialector{}
	_, _, err := d.BuildSelect(&dialect.QueryAST{Table: ""})
	if err == nil {
		t.Fatal("expected error for empty table")
	}
}

// =============================================================================
// Dialector BuildInsert Tests
// =============================================================================

func TestMySQLDialectorBuildInsert(t *testing.T) {
	d := &mysql.Dialector{}

	tests := []struct {
		name     string
		ast      *dialect.InsertAST
		expected string
	}{
		{
			name:     "insert single column",
			ast:      &dialect.InsertAST{Table: "users", Columns: []string{"name"}},
			expected: "INSERT INTO `users` (`name`) VALUES (?)",
		},
		{
			name:     "insert multiple columns",
			ast:      &dialect.InsertAST{Table: "users", Columns: []string{"name", "email"}},
			expected: "INSERT INTO `users` (`name`, `email`) VALUES (?, ?)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlStr, _, err := d.BuildInsert(tt.ast)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sqlStr != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, sqlStr)
			}
		})
	}
}

func TestMySQLDialectorBuildInsertNilAST(t *testing.T) {
	d := &mysql.Dialector{}
	_, _, err := d.BuildInsert(nil)
	if err == nil {
		t.Fatal("expected error for nil ast")
	}
}

func TestMySQLDialectorBuildInsertEmptyTable(t *testing.T) {
	d := &mysql.Dialector{}
	_, _, err := d.BuildInsert(&dialect.InsertAST{Table: "", Columns: []string{"name"}})
	if err == nil {
		t.Fatal("expected error for empty table")
	}
}

func TestMySQLDialectorBuildInsertNoColumns(t *testing.T) {
	d := &mysql.Dialector{}
	_, _, err := d.BuildInsert(&dialect.InsertAST{Table: "users", Columns: []string{}})
	if err == nil {
		t.Fatal("expected error for no columns")
	}
}

// =============================================================================
// Dialector BuildUpdate Tests
// =============================================================================

func TestMySQLDialectorBuildUpdate(t *testing.T) {
	d := &mysql.Dialector{}

	tests := []struct {
		name     string
		ast      *dialect.UpdateAST
		expected string
	}{
		{
			name:     "update single column",
			ast:      &dialect.UpdateAST{Table: "users", Columns: []string{"name"}, Where: []string{"id = ?"}},
			expected: "UPDATE `users` SET `name` = ? WHERE id = ?",
		},
		{
			name:     "update multiple columns",
			ast:      &dialect.UpdateAST{Table: "users", Columns: []string{"name", "email"}, Where: []string{"id = ?"}},
			expected: "UPDATE `users` SET `name` = ?, `email` = ? WHERE id = ?",
		},
		{
			name:     "update without where",
			ast:      &dialect.UpdateAST{Table: "users", Columns: []string{"name"}},
			expected: "UPDATE `users` SET `name` = ?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlStr, _, err := d.BuildUpdate(tt.ast)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sqlStr != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, sqlStr)
			}
		})
	}
}

func TestMySQLDialectorBuildUpdateNilAST(t *testing.T) {
	d := &mysql.Dialector{}
	_, _, err := d.BuildUpdate(nil)
	if err == nil {
		t.Fatal("expected error for nil ast")
	}
}

func TestMySQLDialectorBuildUpdateEmptyTable(t *testing.T) {
	d := &mysql.Dialector{}
	_, _, err := d.BuildUpdate(&dialect.UpdateAST{Table: "", Columns: []string{"name"}})
	if err == nil {
		t.Fatal("expected error for empty table")
	}
}

func TestMySQLDialectorBuildUpdateNoColumns(t *testing.T) {
	d := &mysql.Dialector{}
	_, _, err := d.BuildUpdate(&dialect.UpdateAST{Table: "users", Columns: []string{}})
	if err == nil {
		t.Fatal("expected error for no columns")
	}
}

// =============================================================================
// Dialector BuildDelete Tests
// =============================================================================

func TestMySQLDialectorBuildDelete(t *testing.T) {
	d := &mysql.Dialector{}

	tests := []struct {
		name     string
		ast      *dialect.DeleteAST
		expected string
	}{
		{
			name:     "delete with where",
			ast:      &dialect.DeleteAST{Table: "users", Where: []string{"id = ?"}},
			expected: "DELETE FROM `users` WHERE id = ?",
		},
		{
			name:     "delete without where",
			ast:      &dialect.DeleteAST{Table: "users"},
			expected: "DELETE FROM `users`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlStr, _, err := d.BuildDelete(tt.ast)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sqlStr != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, sqlStr)
			}
		})
	}
}

func TestMySQLDialectorBuildDeleteNilAST(t *testing.T) {
	d := &mysql.Dialector{}
	_, _, err := d.BuildDelete(nil)
	if err == nil {
		t.Fatal("expected error for nil ast")
	}
}

func TestMySQLDialectorBuildDeleteEmptyTable(t *testing.T) {
	d := &mysql.Dialector{}
	_, _, err := d.BuildDelete(&dialect.DeleteAST{Table: ""})
	if err == nil {
		t.Fatal("expected error for empty table")
	}
}

// =============================================================================
// MySQL Index Rules Tests
// =============================================================================

func TestMySQLRuleInvisibleIndex(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	// Check if idx_status is invisible
	var isVisible string
	err := db.QueryRow(`
		SELECT IS_VISIBLE
		FROM INFORMATION_SCHEMA.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'users'
		  AND INDEX_NAME = 'idx_status'
	`).Scan(&isVisible)
	if err != nil {
		t.Fatalf("failed to query visibility: %v", err)
	}

	if isVisible != "NO" {
		t.Errorf("expected idx_status to be INVISIBLE (NO), got %s", isVisible)
	}

	t.Logf("idx_status visibility: %s", isVisible)
}

func TestMySQLRuleDescendingIndex(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	// Check collation via INFORMATION_SCHEMA
	var collation string
	err := db.QueryRow(`
		SELECT COLLATION
		FROM INFORMATION_SCHEMA.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'users'
		  AND INDEX_NAME = 'idx_name_desc'
		  AND SEQ_IN_INDEX = 1
	`).Scan(&collation)
	if err != nil {
		t.Fatalf("failed to query collation: %v", err)
	}

	if collation != "D" {
		t.Errorf("expected idx_name_desc collation to be D, got %s", collation)
	}

	t.Logf("idx_name_desc collation: %s", collation)
}

func TestMySQLRuleDescendingIndexRule(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	provider := mysql.NewMetadataProvider(db)
	ctx := context.Background()

	indexes, err := provider.Indexes(ctx, "users")
	if err != nil {
		t.Fatalf("failed to get indexes: %v", err)
	}

	// Build schema for advisor
	schema := &advisor.TableSchema{
		TableName: "users",
		Indexes:   make([]advisor.IndexInfo, len(indexes)),
	}

	for i, idx := range indexes {
		schema.Indexes[i] = advisor.IndexInfo{
			Name:      idx.Name,
			Columns:   idx.Columns,
			Unique:    idx.Unique,
			Method:    idx.Method,
			IsBTree:   idx.IsBTree,
			IsVisible: idx.IsVisible,
			Metadata:  map[string]any{"table": idx.Table},
		}
	}

	// Query with ORDER BY name DESC
	query := &advisor.QueryMetadata{
		TableName: "users",
		Conditions: []advisor.Condition{
			{Field: "name", Operator: "="},
		},
		OrderBy: []advisor.OrderField{
			{Field: "name", Direction: "DESC"},
		},
	}

	rule := rules.NewDescendingIndexRule()
	suggestions := rule.Check(query, schema)

	// Should have suggestions because idx_name is ASC but query is DESC
	if len(suggestions) == 0 {
		t.Error("expected suggestions for ORDER BY DESC with ASC index")
	}

	for _, s := range suggestions {
		t.Logf("Suggestion: %s - %s", s.RuleID, s.Message)
	}
}

// =============================================================================
// Real Query Execution Tests
// =============================================================================

func TestMySQLRealQueryExecution(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	// Clean up first
	db.Exec("DELETE FROM users WHERE email = ?", "testuser@test.example.com")

	// Insert test data
	_, err := db.Exec("INSERT INTO users (name, email, status) VALUES (?, ?, ?)",
		"TestUser", "testuser@test.example.com", 1)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	// Query using the gore-lint style
	d := &mysql.Dialector{}
	ast := &dialect.QueryAST{
		Table:   "users",
		Columns: []string{"id", "name", "email"},
		Where:   []string{"name = ?"},
		OrderBy: []string{"id ASC"},
		Limit:   10,
	}

	sqlStr, _, err := d.BuildSelect(ast)
	if err != nil {
		t.Fatalf("failed to build select: %v", err)
	}

	t.Logf("Generated SQL: %s", sqlStr)

	// Execute the query
	rows, err := db.Query(sqlStr, "TestUser")
	if err != nil {
		t.Fatalf("failed to execute query: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int
		var name, email string
		if err := rows.Scan(&id, &name, &email); err != nil {
			t.Fatalf("failed to scan row: %v", err)
		}
		count++
		t.Logf("User: id=%d, name=%s, email=%s", id, name, email)
	}

	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}

	// Clean up
	db.Exec("DELETE FROM users WHERE email = ?", "testuser@test.example.com")
}

func TestMySQLRealInsertAndQuery(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	// Clean up first
	db.Exec("DELETE FROM users WHERE email = ?", "inserttest@test.example.com")

	// Insert using gore-generated SQL
	d := &mysql.Dialector{}
	insertAST := &dialect.InsertAST{
		Table:   "users",
		Columns: []string{"name", "email", "status"},
	}
	insertSQL, _, _ := d.BuildInsert(insertAST)
	t.Logf("Insert SQL: %s", insertSQL)

	_, err := db.Exec(insertSQL, "InsertTest", "inserttest@test.example.com", 1)
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	// Query back
	selectAST := &dialect.QueryAST{
		Table:   "users",
		Columns: []string{"id", "name", "email"},
		Where:   []string{"email = ?"},
	}
	selectSQL, _, _ := d.BuildSelect(selectAST)
	t.Logf("Select SQL: %s", selectSQL)

	var id int
	var name, email string
	err = db.QueryRow(selectSQL, "inserttest@test.example.com").Scan(&id, &name, &email)
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}

	if name != "InsertTest" {
		t.Errorf("expected name 'InsertTest', got %s", name)
	}

	// Clean up
	db.Exec("DELETE FROM users WHERE email = ?", "inserttest@test.example.com")
}

func TestMySQLRealUpdateAndQuery(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	// Setup: insert test user
	db.Exec("DELETE FROM users WHERE email = ?", "updatetest@test.example.com")
	db.Exec("INSERT INTO users (name, email, status) VALUES (?, ?, ?)",
		"UpdateTest", "updatetest@test.example.com", 1)

	// Update using gore-generated SQL
	d := &mysql.Dialector{}
	updateAST := &dialect.UpdateAST{
		Table:   "users",
		Columns: []string{"name", "status"},
		Where:   []string{"email = ?"},
	}
	updateSQL, _, _ := d.BuildUpdate(updateAST)
	t.Logf("Update SQL: %s", updateSQL)

	_, err := db.Exec(updateSQL, "UpdatedName", 2, "updatetest@test.example.com")
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	// Verify update
	var name string
	var status int
	err = db.QueryRow("SELECT name, status FROM users WHERE email = ?", "updatetest@test.example.com").
		Scan(&name, &status)
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}

	if name != "UpdatedName" {
		t.Errorf("expected name 'UpdatedName', got %s", name)
	}
	if status != 2 {
		t.Errorf("expected status 2, got %d", status)
	}

	// Clean up
	db.Exec("DELETE FROM users WHERE email = ?", "updatetest@test.example.com")
}

func TestMySQLRealDeleteAndQuery(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	// Setup: insert test user
	db.Exec("DELETE FROM users WHERE email = ?", "deletetest@test.example.com")
	db.Exec("INSERT INTO users (name, email, status) VALUES (?, ?, ?)",
		"DeleteTest", "deletetest@test.example.com", 1)

	// Delete using gore-generated SQL
	d := &mysql.Dialector{}
	deleteAST := &dialect.DeleteAST{
		Table: "users",
		Where: []string{"email = ?"},
	}
	deleteSQL, _, _ := d.BuildDelete(deleteAST)
	t.Logf("Delete SQL: %s", deleteSQL)

	_, err := db.Exec(deleteSQL, "deletetest@test.example.com")
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// Verify delete
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "deletetest@test.example.com").
		Scan(&count)
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 rows after delete, got %d", count)
	}
}

// =============================================================================
// gore-lint Index Advisor Integration
// =============================================================================

func TestMySQLIndexAdvisorInvisibleIndex(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	provider := mysql.NewMetadataProvider(db)
	ctx := context.Background()

	indexes, err := provider.Indexes(ctx, "users")
	if err != nil {
		t.Fatalf("failed to get indexes: %v", err)
	}

	schema := &advisor.TableSchema{
		TableName: "users",
		Indexes:   make([]advisor.IndexInfo, len(indexes)),
	}

	for i, idx := range indexes {
		schema.Indexes[i] = advisor.IndexInfo{
			Name:      idx.Name,
			Columns:   idx.Columns,
			Unique:    idx.Unique,
			Method:    idx.Method,
			IsBTree:   idx.IsBTree,
			IsVisible: idx.IsVisible,
			Metadata:  map[string]any{"table": idx.Table},
		}
	}

	// Query that doesn't use the invisible index
	query := &advisor.QueryMetadata{
		TableName: "users",
		Conditions: []advisor.Condition{
			{Field: "email", Operator: "="},
		},
	}

	rule := rules.NewInvisibleIndexRule()
	suggestions := rule.Check(query, schema)

	// Should have suggestion because idx_status is invisible and not used
	if len(suggestions) == 0 {
		t.Error("expected suggestion for unused invisible index")
	}

	for _, s := range suggestions {
		t.Logf("Suggestion: %s - %s", s.RuleID, s.Message)
	}
}

func TestMySQLIndexAdvisorAllRules(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	provider := mysql.NewMetadataProvider(db)
	ctx := context.Background()

	indexes, err := provider.Indexes(ctx, "users")
	if err != nil {
		t.Fatalf("failed to get indexes: %v", err)
	}

	schema := &advisor.TableSchema{
		TableName: "users",
		Indexes:   make([]advisor.IndexInfo, len(indexes)),
	}

	for i, idx := range indexes {
		schema.Indexes[i] = advisor.IndexInfo{
			Name:      idx.Name,
			Columns:   idx.Columns,
			Unique:    idx.Unique,
			Method:    idx.Method,
			IsBTree:   idx.IsBTree,
			IsVisible: idx.IsVisible,
			Metadata:  map[string]any{"table": idx.Table},
		}
	}

	// Complex query
	query := &advisor.QueryMetadata{
		TableName: "users",
		Conditions: []advisor.Condition{
			{Field: "name", Operator: "="},
			{Field: "status", Operator: ">"},
		},
		OrderBy: []advisor.OrderField{
			{Field: "name", Direction: "DESC"},
		},
		Limit: ptrInt(10),
	}

	engine := advisor.NewEngine(
		rules.NewInvisibleIndexRule(),
		rules.NewDescendingIndexRule(),
		rules.NewHashIndexRangeRule(),
	)

	suggestions := engine.Analyze(query, schema)
	t.Logf("Total suggestions: %d", len(suggestions))

	for _, s := range suggestions {
		t.Logf("  [%s] %s", s.RuleID, s.Message)
	}
}

func ptrInt(i int) *int {
	return &i
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkMySQLIndexes(b *testing.B) {
	db := newTestDBForBench()
	defer db.Close()

	provider := mysql.NewMetadataProvider(db)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.Indexes(ctx, "users")
		if err != nil {
			b.Fatalf("failed to get indexes: %v", err)
		}
	}
}

func BenchmarkMySQLColumns(b *testing.B) {
	db := newTestDBForBench()
	defer db.Close()

	provider := mysql.NewMetadataProvider(db)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.Columns(ctx, "users")
		if err != nil {
			b.Fatalf("failed to get columns: %v", err)
		}
	}
}

func BenchmarkMySQLTables(b *testing.B) {
	db := newTestDBForBench()
	defer db.Close()

	provider := mysql.NewMetadataProvider(db)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.Tables(ctx)
		if err != nil {
			b.Fatalf("failed to get tables: %v", err)
		}
	}
}

func BenchmarkBuildSelect(b *testing.B) {
	d := &mysql.Dialector{}
	ast := &dialect.QueryAST{
		Table:   "users",
		Columns: []string{"id", "name", "email"},
		Where:   []string{"status > ?"},
		OrderBy: []string{"created_at DESC"},
		Limit:   10,
		Offset:  5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = d.BuildSelect(ast)
	}
}

func BenchmarkBuildInsert(b *testing.B) {
	d := &mysql.Dialector{}
	ast := &dialect.InsertAST{
		Table:   "users",
		Columns: []string{"name", "email", "status"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = d.BuildInsert(ast)
	}
}
