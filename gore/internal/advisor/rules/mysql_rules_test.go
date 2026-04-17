package rules

import (
	"testing"

	"gore/internal/advisor"
)

func TestInvisibleIndexRule(t *testing.T) {
	rule := NewInvisibleIndexRule()

	if rule.ID() != "IDX-MYSQL-001" {
		t.Fatalf("expected ID IDX-MYSQL-001, got %s", rule.ID())
	}

	if rule.Name() != "Invisible Index Detection" {
		t.Fatalf("unexpected name: %s", rule.Name())
	}

	if rule.Severity() != advisor.SeverityInfo {
		t.Fatalf("expected SeverityInfo, got %v", rule.Severity())
	}
}

func TestInvisibleIndexRuleCheck(t *testing.T) {
	rule := NewInvisibleIndexRule()

	// Query that uses 'email' column, but invisible index is on 'name'
	query := &advisor.QueryMetadata{
		TableName: "users",
		Conditions: []advisor.Condition{
			{Field: "email", Operator: "="},
		},
	}

	schema := &advisor.TableSchema{
		TableName: "users",
		Indexes: []advisor.IndexInfo{
			{
				Name:    "idx_name",
				Columns: []string{"name"},
				Metadata: map[string]any{
					"is_visible": false, // invisible
				},
			},
		},
	}

	suggestions := rule.Check(query, schema)
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}

	if suggestions[0].RuleID != "IDX-MYSQL-001" {
		t.Fatalf("expected rule ID IDX-MYSQL-001, got %s", suggestions[0].RuleID)
	}
}

func TestInvisibleIndexRuleNoInvisible(t *testing.T) {
	rule := NewInvisibleIndexRule()

	query := &advisor.QueryMetadata{
		TableName: "users",
		Conditions: []advisor.Condition{
			{Field: "name", Operator: "="},
		},
	}

	schema := &advisor.TableSchema{
		TableName: "users",
		Indexes: []advisor.IndexInfo{
			{
				Name:    "idx_name",
				Columns: []string{"name"},
				Metadata: map[string]any{
					"is_visible": true, // visible
				},
			},
		},
	}

	suggestions := rule.Check(query, schema)
	if len(suggestions) != 0 {
		t.Fatalf("expected 0 suggestions for visible index, got %d", len(suggestions))
	}
}

func TestInvisibleIndexRuleNil(t *testing.T) {
	rule := NewInvisibleIndexRule()

	if rule.Check(nil, nil) != nil {
		t.Fatal("expected nil for nil inputs")
	}

	if rule.Check(&advisor.QueryMetadata{}, nil) != nil {
		t.Fatal("expected nil for nil schema")
	}
}

func TestDescendingIndexRule(t *testing.T) {
	rule := NewDescendingIndexRule()

	if rule.ID() != "IDX-MYSQL-002" {
		t.Fatalf("expected ID IDX-MYSQL-002, got %s", rule.ID())
	}

	if rule.Name() != "Descending Index Validation" {
		t.Fatalf("unexpected name: %s", rule.Name())
	}

	if rule.Severity() != advisor.SeverityWarn {
		t.Fatalf("expected SeverityWarn, got %v", rule.Severity())
	}
}

func TestDescendingIndexRuleCheck(t *testing.T) {
	rule := NewDescendingIndexRule()

	// Query ORDER BY name DESC, but index is ASC
	query := &advisor.QueryMetadata{
		TableName: "users",
		OrderBy: []advisor.OrderField{
			{Field: "name", Direction: "DESC"},
		},
	}

	schema := &advisor.TableSchema{
		TableName: "users",
		Indexes: []advisor.IndexInfo{
			{
				Name:    "idx_name",
				Columns: []string{"name"},
				Metadata: map[string]any{
					"collation": "A", // ascending
				},
			},
		},
	}

	suggestions := rule.Check(query, schema)
	if len(suggestions) == 0 {
		t.Fatal("expected suggestions for direction mismatch")
	}

	if suggestions[0].RuleID != "IDX-MYSQL-002" {
		t.Fatalf("expected rule ID IDX-MYSQL-002, got %s", suggestions[0].RuleID)
	}
}

func TestDescendingIndexRuleMatch(t *testing.T) {
	rule := NewDescendingIndexRule()

	// Query ORDER BY name DESC, and index is DESC
	query := &advisor.QueryMetadata{
		TableName: "users",
		OrderBy: []advisor.OrderField{
			{Field: "name", Direction: "DESC"},
		},
	}

	schema := &advisor.TableSchema{
		TableName: "users",
		Indexes: []advisor.IndexInfo{
			{
				Name:    "idx_name",
				Columns: []string{"name"},
				Metadata: map[string]any{
					"collation": "D", // descending, matches query
				},
			},
		},
	}

	suggestions := rule.Check(query, schema)
	if len(suggestions) != 0 {
		t.Fatalf("expected 0 suggestions for matching directions, got %d", len(suggestions))
	}
}

func TestDescendingIndexRuleNil(t *testing.T) {
	rule := NewDescendingIndexRule()

	if rule.Check(nil, nil) != nil {
		t.Fatal("expected nil for nil inputs")
	}
}

func TestHashIndexRangeRule(t *testing.T) {
	rule := NewHashIndexRangeRule()

	if rule.ID() != "IDX-MYSQL-003" {
		t.Fatalf("expected ID IDX-MYSQL-003, got %s", rule.ID())
	}

	if rule.Name() != "Hash Index Range Query" {
		t.Fatalf("unexpected name: %s", rule.Name())
	}

	if rule.Severity() != advisor.SeverityWarn {
		t.Fatalf("expected SeverityWarn, got %v", rule.Severity())
	}
}

func TestHashIndexRangeRuleCheck(t *testing.T) {
	rule := NewHashIndexRangeRule()

	// Query with range operator on HASH indexed column
	query := &advisor.QueryMetadata{
		TableName: "users",
		Conditions: []advisor.Condition{
			{Field: "id", Operator: ">", Value: 10},
		},
	}

	schema := &advisor.TableSchema{
		TableName: "users",
		Indexes: []advisor.IndexInfo{
			{
				Name:    "idx_id",
				Columns: []string{"id"},
				Method:  "HASH",
				IsBTree: false,
			},
		},
	}

	suggestions := rule.Check(query, schema)
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}

	if suggestions[0].RuleID != "IDX-MYSQL-003" {
		t.Fatalf("expected rule ID IDX-MYSQL-003, got %s", suggestions[0].RuleID)
	}
}

func TestHashIndexRangeRuleEqual(t *testing.T) {
	rule := NewHashIndexRangeRule()

	// Query with equality operator on HASH indexed column - should be OK
	query := &advisor.QueryMetadata{
		TableName: "users",
		Conditions: []advisor.Condition{
			{Field: "id", Operator: "=", Value: 10},
		},
	}

	schema := &advisor.TableSchema{
		TableName: "users",
		Indexes: []advisor.IndexInfo{
			{
				Name:    "idx_id",
				Columns: []string{"id"},
				Method:  "HASH",
				IsBTree: false,
			},
		},
	}

	suggestions := rule.Check(query, schema)
	if len(suggestions) != 0 {
		t.Fatalf("expected 0 suggestions for equality, got %d", len(suggestions))
	}
}

func TestHashIndexRangeRuleIN(t *testing.T) {
	rule := NewHashIndexRangeRule()

	// Query with IN operator on HASH indexed column - should be OK
	query := &advisor.QueryMetadata{
		TableName: "users",
		Conditions: []advisor.Condition{
			{Field: "id", Operator: "IN", Value: []int{1, 2, 3}},
		},
	}

	schema := &advisor.TableSchema{
		TableName: "users",
		Indexes: []advisor.IndexInfo{
			{
				Name:    "idx_id",
				Columns: []string{"id"},
				Method:  "HASH",
				IsBTree: false,
			},
		},
	}

	suggestions := rule.Check(query, schema)
	if len(suggestions) != 0 {
		t.Fatalf("expected 0 suggestions for IN, got %d", len(suggestions))
	}
}

func TestHashIndexRangeRuleBTREE(t *testing.T) {
	rule := NewHashIndexRangeRule()

	// Query with range operator on BTREE indexed column - should be OK
	query := &advisor.QueryMetadata{
		TableName: "users",
		Conditions: []advisor.Condition{
			{Field: "id", Operator: ">", Value: 10},
		},
	}

	schema := &advisor.TableSchema{
		TableName: "users",
		Indexes: []advisor.IndexInfo{
			{
				Name:    "idx_id",
				Columns: []string{"id"},
				Method:  "BTREE",
				IsBTree: true,
			},
		},
	}

	suggestions := rule.Check(query, schema)
	if len(suggestions) != 0 {
		t.Fatalf("expected 0 suggestions for BTREE range, got %d", len(suggestions))
	}
}

func TestHashIndexRangeRuleNil(t *testing.T) {
	rule := NewHashIndexRangeRule()

	if rule.Check(nil, nil) != nil {
		t.Fatal("expected nil for nil inputs")
	}
}

func TestOppositeDirection(t *testing.T) {
	if oppositeDirection("A") != "DESC" {
		t.Fatal("expected DESC for A")
	}
	if oppositeDirection("D") != "ASC" {
		t.Fatal("expected ASC for D")
	}
}
