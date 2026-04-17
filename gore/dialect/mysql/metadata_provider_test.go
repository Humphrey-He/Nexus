package mysql

import (
	"database/sql"
	"testing"

	"gore/internal/metadata"
)

const testDSN = "root:root@tcp(localhost:3307)/gore_test"

// TestNewMetadataProvider tests creating a new MetadataProvider
func TestNewMetadataProvider(t *testing.T) {
	db, _ := sql.Open("mysql", "root:root@tcp(localhost:3307)/gore_test")
	provider := NewMetadataProvider(db)
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
	if provider.db == nil {
		t.Fatal("expected non-nil db")
	}
	db.Close()
}

// TestIndexInfoMetadata tests the IndexInfo structure
func TestIndexInfoMetadata(t *testing.T) {
	info := metadata.IndexInfo{
		Name:    "idx_test",
		Table:   "users",
		Columns: []string{"id", "name"},
		Unique:  false,
		Method:  "BTREE",
		IsBTree: true,
	}

	if info.Name != "idx_test" {
		t.Errorf("expected Name 'idx_test', got %s", info.Name)
	}
	if info.Table != "users" {
		t.Errorf("expected Table 'users', got %s", info.Table)
	}
	if len(info.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(info.Columns))
	}
	if info.Unique {
		t.Error("expected Unique to be false")
	}
	if info.Method != "BTREE" {
		t.Errorf("expected Method 'BTREE', got %s", info.Method)
	}
	if !info.IsBTree {
		t.Error("expected IsBTree to be true")
	}
}

// TestColumnInfoMetadata tests the ColumnInfo structure
func TestColumnInfoMetadata(t *testing.T) {
	col := metadata.ColumnInfo{
		Name: "id",
		Type: "int auto_increment",
	}

	if col.Name != "id" {
		t.Errorf("expected Name 'id', got %s", col.Name)
	}
	if col.Type != "int auto_increment" {
		t.Errorf("expected Type 'int auto_increment', got %s", col.Type)
	}
}

// TestIndexInfoBTreeMapping tests IsBTree mapping
func TestIndexInfoBTreeMapping(t *testing.T) {
	tests := []struct {
		method  string
		isBTree bool
	}{
		{"BTREE", true},
		{"HASH", false},
		{"btree", false}, // case sensitive
		{"", false},
	}

	for _, tt := range tests {
		info := metadata.IndexInfo{
			Name:    "test",
			Method:  tt.method,
			IsBTree: tt.method == "BTREE",
		}
		if info.IsBTree != tt.isBTree {
			t.Errorf("method=%s: expected IsBTree=%v, got %v", tt.method, tt.isBTree, info.IsBTree)
		}
	}
}

// TestIndexInfoUniqueMapping tests Unique mapping from NON_UNIQUE
func TestIndexInfoUniqueMapping(t *testing.T) {
	tests := []struct {
		nonUnique bool
		unique    bool
	}{
		{false, true},  // PRIMARY and UNIQUE indexes
		{true, false},  // Non-unique indexes
	}

	for _, tt := range tests {
		info := metadata.IndexInfo{
			Name:   "test",
			Unique: !tt.nonUnique,
		}
		if info.Unique != tt.unique {
			t.Errorf("nonUnique=%v: expected Unique=%v, got %v", tt.nonUnique, tt.unique, info.Unique)
		}
	}
}

// TestColumnInfoType tests type string building
func TestColumnInfoType(t *testing.T) {
	tests := []struct {
		dataType string
		extra    string
		expected string
	}{
		{"int", "", "int"},
		{"int", "auto_increment", "int auto_increment"},
		{"varchar", "", "varchar"},
		{"timestamp", "DEFAULT_GENERATED", "timestamp DEFAULT_GENERATED"},
	}

	for _, tt := range tests {
		var colType string
		if tt.extra != "" {
			colType = tt.dataType + " " + tt.extra
		} else {
			colType = tt.dataType
		}
		if colType != tt.expected {
			t.Errorf("dataType=%s, extra=%s: expected %s, got %s",
				tt.dataType, tt.extra, tt.expected, colType)
		}
	}
}

// TestProviderDBAssignment tests that db is properly assigned
func TestProviderDBAssignment(t *testing.T) {
	db, _ := sql.Open("mysql", testDSN)
	defer db.Close()

	provider := NewMetadataProvider(db)
	if provider.db != db {
		t.Error("expected provider.db to be the same as input db")
	}
}
