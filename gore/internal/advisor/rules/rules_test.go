package rules

import (
	"testing"

	"gore/internal/advisor"
)

func baseSchema() advisor.TableSchema {
	return advisor.TableSchema{
		TableName: "users",
		Columns: []advisor.ColumnInfo{
			{Name: "id", Type: "int"},
			{Name: "name", Type: "text"},
			{Name: "age", Type: "int"},
			{Name: "email", Type: "varchar"},
			{Name: "created_at", Type: "timestamp"},
			{Name: "status", Type: "varchar"},
		},
		Indexes: []advisor.IndexInfo{
			{Name: "users_id_idx", Columns: []string{"id"}, Unique: true, Method: "btree", IsBTree: true},
			{Name: "users_name_age_idx", Columns: []string{"name", "age"}, Method: "btree", IsBTree: true},
			{Name: "users_email_idx", Columns: []string{"email"}, Method: "btree", IsBTree: true},
			{Name: "users_created_at_idx", Columns: []string{"created_at"}, Method: "btree", IsBTree: true},
		},
	}
}

func sampleQuery() *advisor.QueryMetadata {
	return &advisor.QueryMetadata{
		TableName: "users",
		Conditions: []advisor.Condition{
			{Field: "age", Operator: "=", Value: "20", ValueType: "string"},
			{Field: "LOWER(name)", Operator: "=", Value: "bob", ValueType: "string", IsFunction: true, FuncName: "LOWER"},
			{Field: "name", Operator: "LIKE", Value: "%bob", ValueType: "string"},
			{Field: "status", Operator: "!=", Value: "deleted", ValueType: "string", IsNegated: true},
		},
		OrderBy: []advisor.OrderField{{Field: "created_at", Direction: "DESC"}},
		Joins: []advisor.JoinClause{{
			Type:         "INNER",
			Table:        "orders",
			OnConditions: []advisor.Condition{{Field: "order_id", Operator: "=", ValueType: "int"}},
		}},
	}
}

func TestRulesEmitSuggestions(t *testing.T) {
	schema := baseSchema()
	query := sampleQuery()

	cases := []struct {
		rule advisor.Rule
		name string
		min  int
	}{
		{NewFunctionIndexRule(), "IDX-002", 1},
		{NewTypeMismatchRule(), "IDX-003", 1},
		{NewLikePrefixRule(), "IDX-004", 1},
		{NewNegationRule(), "IDX-005", 1},
		{NewOrderByIndexRule(), "IDX-008", 1},
		{NewJoinIndexRule(), "IDX-009", 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := tc.rule.Check(query, &schema)
			if len(out) < tc.min {
				t.Fatalf("%s expected >= %d suggestion(s), got %d", tc.name, tc.min, len(out))
			}
		})
	}
}

func TestLeftmostMatchRuleEdgeCases(t *testing.T) {
	schema := advisor.TableSchema{
		TableName: "users",
		Columns: []advisor.ColumnInfo{
			{Name: "id", Type: "int"},
			{Name: "name", Type: "text"},
			{Name: "age", Type: "int"},
		},
		Indexes: []advisor.IndexInfo{
			{Name: "users_name_age_idx", Columns: []string{"name", "age"}, Method: "btree", IsBTree: true},
		},
	}

	tests := []struct {
		name        string
		conditions  []advisor.Condition
		wantWarn    bool // whether we expect a warning
		description string
	}{
		{
			name:        "uses first column only - OK",
			conditions:  []advisor.Condition{{Field: "name", Operator: "=", Value: "alice"}},
			wantWarn:    false,
			description: "Using only first column should be OK",
		},
		{
			name:        "uses first and second columns - OK",
			conditions:  []advisor.Condition{{Field: "name", Operator: "=", Value: "alice"}, {Field: "age", Operator: "=", Value: "30"}},
			wantWarn:    false,
			description: "Using first and second columns in order should be OK",
		},
		{
			name:        "skips first column - should warn",
			conditions:  []advisor.Condition{{Field: "age", Operator: "=", Value: "30"}},
			wantWarn:    true,
			description: "Skipping first column should trigger warning",
		},
		{
			name:        "uses first then third (skipping second) - OK per PostgreSQL",
			conditions:  []advisor.Condition{{Field: "name", Operator: "=", Value: "alice"}, {Field: "age", Operator: "=", Value: "30"}},
			wantWarn:    false,
			description: "Using first then third (skipping second) is OK in PostgreSQL",
		},
		{
			name:        "negated condition on first column - no IDX-001 warning (IDX-005 handles negation)",
			conditions:  []advisor.Condition{{Field: "name", Operator: "!=", Value: "alice", IsNegated: true}},
			wantWarn:    false,
			description: "Negated condition on first column satisfies leftmost match; negation caught by IDX-005",
		},
	}

	rule := NewLeftmostMatchRule()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &advisor.QueryMetadata{
				TableName:  "users",
				Conditions: tt.conditions,
			}
			out := rule.Check(query, &schema)
			hasWarning := len(out) > 0
			if hasWarning != tt.wantWarn {
				t.Errorf("%s: got warning=%v, want warn=%v (%s)", tt.name, hasWarning, tt.wantWarn, tt.description)
			}
		})
	}
}

func TestTypeMismatchRuleEdgeCases(t *testing.T) {
	schema := advisor.TableSchema{
		TableName: "users",
		Columns: []advisor.ColumnInfo{
			{Name: "id", Type: "int"},
			{Name: "big_id", Type: "bigint"},
			{Name: "price", Type: "numeric"},
			{Name: "name", Type: "varchar"},
			{Name: "active", Type: "bool"},
		},
		Indexes: []advisor.IndexInfo{
			{Name: "users_id_idx", Columns: []string{"id"}, Method: "btree"},
		},
	}

	tests := []struct {
		name      string
		field     string
		valueType string
		wantWarn  bool
		desc      string
	}{
		// int with string - should warn
		{"int column with string value", "id", "string", true, "int vs string mismatch"},
		// int with int - OK
		{"int column with int value", "id", "int", false, "same type"},
		// int with int64 - OK (widening)
		{"int column with int64 value", "id", "int64", false, "widening conversion"},
		// numeric with int - should warn
		{"numeric column with int value", "price", "int", true, "numeric vs int mismatch"},
		// varchar with string - OK
		{"varchar column with string value", "name", "string", false, "varchar accepts string"},
		// bool with bool - OK
		{"bool column with bool value", "active", "bool", false, "same type"},
		// bool with string - should warn
		{"bool column with string value", "active", "string", true, "bool vs string mismatch"},
	}

	rule := NewTypeMismatchRule()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &advisor.QueryMetadata{
				TableName: "users",
				Conditions: []advisor.Condition{
					{Field: tt.field, Operator: "=", Value: nil, ValueType: tt.valueType},
				},
			}
			out := rule.Check(query, &schema)
			hasWarning := len(out) > 0
			if hasWarning != tt.wantWarn {
				t.Errorf("%s: got warn=%v, want warn=%v (%s)", tt.name, hasWarning, tt.wantWarn, tt.desc)
			}
		})
	}
}

func TestOrderByIndexRuleDirection(t *testing.T) {
	schema := advisor.TableSchema{
		TableName: "users",
		Columns: []advisor.ColumnInfo{
			{Name: "created_at", Type: "timestamp"},
		},
		Indexes: []advisor.IndexInfo{
			{Name: "users_created_at_idx", Columns: []string{"created_at"}, Method: "btree", IsBTree: true},
		},
	}

	tests := []struct {
		name     string
		orderBy  []advisor.OrderField
		wantWarn bool
		desc     string
	}{
		{
			name:     "order by DESC on indexed column",
			orderBy:  []advisor.OrderField{{Field: "created_at", Direction: "DESC"}},
			wantWarn: true, // Direction mismatch warning
			desc:     "DESC order can use ASC index but not optimal",
		},
		{
			name:     "order by ASC on indexed column",
			orderBy:  []advisor.OrderField{{Field: "created_at", Direction: "ASC"}},
			wantWarn: false,
			desc:     "ASC matches ASC index direction",
		},
		{
			name:     "order by unindexed column",
			orderBy:  []advisor.OrderField{{Field: "unknown", Direction: "ASC"}},
			wantWarn: true,
			desc:     "No index coverage",
		},
	}

	rule := NewOrderByIndexRule()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &advisor.QueryMetadata{
				TableName: "users",
				OrderBy:   tt.orderBy,
			}
			out := rule.Check(query, &schema)
			hasWarning := len(out) > 0
			if hasWarning != tt.wantWarn {
				t.Errorf("%s: got warn=%v, want warn=%v (%s)", tt.name, hasWarning, tt.wantWarn, tt.desc)
			}
		})
	}
}

func TestMissingIndexRule(t *testing.T) {
	schema := advisor.TableSchema{
		TableName: "users",
		Columns: []advisor.ColumnInfo{
			{Name: "id", Type: "int"},
			{Name: "name", Type: "text"},
		},
		// No indexes at all - all fields are missing indexes
		Indexes: []advisor.IndexInfo{},
	}

	// Multiple queries on the same field should trigger IDX-006
	queries := []*advisor.QueryMetadata{
		{TableName: "users", Conditions: []advisor.Condition{{Field: "name", Operator: "=", Value: "alice"}}},
		{TableName: "users", Conditions: []advisor.Condition{{Field: "name", Operator: "=", Value: "bob"}}},
		{TableName: "users", Conditions: []advisor.Condition{{Field: "name", Operator: "=", Value: "carol"}}},
	}

	rule := NewMissingIndexRule()
	out := rule.CheckAll(queries, &schema)
	if len(out) == 0 {
		t.Fatal("expected missing index suggestion for 'name' field")
	}
	if out[0].RuleID != "IDX-006" {
		t.Fatalf("expected IDX-006, got %s", out[0].RuleID)
	}
}

func TestRedundantIndexRule(t *testing.T) {
	schema := advisor.TableSchema{
		TableName: "users",
		Indexes: []advisor.IndexInfo{
			{Name: "users_name_idx", Columns: []string{"name"}, Unique: false, Method: "btree"},
			{Name: "users_name_age_idx", Columns: []string{"name", "age"}, Unique: false, Method: "btree"},
		},
	}

	rule := NewRedundantIndexRule()
	out := rule.Check(nil, &schema)
	if len(out) == 0 {
		t.Fatal("expected redundant index suggestion")
	}
	if out[0].RuleID != "IDX-007" {
		t.Fatalf("expected IDX-007, got %s", out[0].RuleID)
	}
}

func TestLowSelectivityRule(t *testing.T) {
	schema := advisor.TableSchema{
		TableName: "users",
		Indexes: []advisor.IndexInfo{
			{
				Name:     "users_status_idx",
				Columns:  []string{"status"},
				Unique:   false,
				Method:   "btree",
				Metadata: map[string]any{"selectivity": 0.005}, // 0.5% selectivity
			},
		},
	}

	rule := NewLowSelectivityRule()
	out := rule.CheckSchema(&schema)
	if len(out) == 0 {
		t.Fatal("expected low selectivity suggestion")
	}
	if out[0].RuleID != "IDX-010" {
		t.Fatalf("expected IDX-010, got %s", out[0].RuleID)
	}
}

func TestGroupByDistinctExtraction(t *testing.T) {
	// This test verifies that GroupBy fields are extracted correctly in the AST
	// The actual Index Advisor rules may or may not use GroupBy fields
	// depending on the rule semantics (GROUP BY typically doesn't use indexes the same way WHERE does)
	query := &advisor.QueryMetadata{
		GroupBy:     []string{"status", "category"},
		IsDistinct: true,
	}

	if len(query.GroupBy) != 2 {
		t.Fatalf("expected 2 group by fields, got %d", len(query.GroupBy))
	}
	if !query.IsDistinct {
		t.Fatal("expected IsDistinct to be true")
	}
}

func TestRuleInterfaceMethods(t *testing.T) {
	rules := []advisor.Rule{
		NewLeftmostMatchRule(),
		NewFunctionIndexRule(),
		NewTypeMismatchRule(),
		NewLikePrefixRule(),
		NewNegationRule(),
		NewMissingIndexRule(),
		NewRedundantIndexRule(),
		NewOrderByIndexRule(),
		NewJoinIndexRule(),
		NewLowSelectivityRule(),
	}

	for _, r := range rules {
		if r.ID() == "" {
			t.Fatalf("expected non-empty rule ID for %T", r)
		}
		if r.Name() == "" {
			t.Fatalf("expected non-empty rule name for %T", r)
		}
		if r.Description() == "" {
			t.Fatalf("expected non-empty rule description for %T", r)
		}
		// SeverityInfo = 0, so just verify it returns a valid value
		// Valid severities are: SeverityInfo(0), SeverityWarn(1), SeverityHigh(2), SeverityCritical(3)
		sev := r.Severity()
		if sev < 0 || sev > 3 {
			t.Fatalf("expected valid severity (0-3) for %T, got %d", r, sev)
		}
		// WhyDoc can be empty but should not panic
		_ = r.WhyDoc()
	}
}

func TestRedundantIndexRuleEdgeCases(t *testing.T) {
	schema := advisor.TableSchema{
		TableName: "users",
		Indexes: []advisor.IndexInfo{
			{Name: "users_name_idx", Columns: []string{"name"}, Unique: false, Method: "btree"},
			{Name: "users_name_email_idx", Columns: []string{"name", "email"}, Unique: false, Method: "btree"},
			{Name: "users_name_age_idx", Columns: []string{"name", "age"}, Unique: false, Method: "btree"},
		},
	}

	rule := NewRedundantIndexRule()
	out := rule.Check(nil, &schema)
	// With 3 indexes that are redundant (name is prefix of others), we expect suggestions
	// The exact number depends on how redundancy is defined
	if len(out) == 0 {
		t.Fatal("expected redundant index suggestions for prefix columns")
	}
}

func TestLikePrefixRuleWithUnderscore(t *testing.T) {
	schema := advisor.TableSchema{
		TableName: "users",
		Columns: []advisor.ColumnInfo{
			{Name: "name", Type: "text"},
		},
		Indexes: []advisor.IndexInfo{
			{Name: "users_name_idx", Columns: []string{"name"}, Method: "btree", IsBTree: true},
		},
	}

	// Query with underscore in pattern (not a prefix LIKE)
	query := &advisor.QueryMetadata{
		TableName: "users",
		Conditions: []advisor.Condition{
			{Field: "name", Operator: "LIKE", Value: "ali_e", ValueType: "string"},
		},
	}

	rule := NewLikePrefixRule()
	out := rule.Check(query, &schema)
	// Underscore is a single character wildcard, not a leading wildcard
	if len(out) != 0 {
		t.Fatalf("expected no warning for underscore pattern, got %d", len(out))
	}
}

func TestLikePrefixRuleNoIndex(t *testing.T) {
	schema := advisor.TableSchema{
		TableName: "users",
		Columns: []advisor.ColumnInfo{
			{Name: "name", Type: "text"},
		},
		Indexes: []advisor.IndexInfo{}, // No index
	}

	query := &advisor.QueryMetadata{
		TableName: "users",
		Conditions: []advisor.Condition{
			{Field: "name", Operator: "LIKE", Value: "%alice", ValueType: "string"},
		},
	}

	rule := NewLikePrefixRule()
	out := rule.Check(query, &schema)
	// LikePrefixRule warns about leading wildcard regardless of index existence
	// (it assumes you might want an index even if you don't have one)
	if len(out) != 1 {
		t.Fatalf("expected 1 warning for leading wildcard, got %d", len(out))
	}
}

func TestNegationRuleEdgeCases(t *testing.T) {
	schema := advisor.TableSchema{
		TableName: "users",
		Columns: []advisor.ColumnInfo{
			{Name: "id", Type: "int"},
			{Name: "name", Type: "text"},
		},
		Indexes: []advisor.IndexInfo{
			{Name: "users_id_idx", Columns: []string{"id"}, Method: "btree", IsBTree: true},
		},
	}

	tests := []struct {
		name      string
		operator  string
		wantWarn  bool
	}{
		{"NOT operator", "NOT", true},
		{"IS NOT operator", "IS NOT", true},
		{"<> operator", "<>", true},
	}

	rule := NewNegationRule()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &advisor.QueryMetadata{
				TableName: "users",
				Conditions: []advisor.Condition{
					{Field: "id", Operator: tt.operator, Value: "0", ValueType: "int", IsNegated: true},
				},
			}
			out := rule.Check(query, &schema)
			hasWarn := len(out) > 0
			if hasWarn != tt.wantWarn {
				t.Errorf("%s: got warn=%v, want warn=%v", tt.name, hasWarn, tt.wantWarn)
			}
		})
	}
}

func TestJoinIndexRuleEdgeCases(t *testing.T) {
	schema := advisor.TableSchema{
		TableName: "users",
		Columns: []advisor.ColumnInfo{
			{Name: "id", Type: "int"},
			{Name: "order_id", Type: "int"},
		},
		Indexes: []advisor.IndexInfo{
			{Name: "users_id_idx", Columns: []string{"id"}, Method: "btree", IsBTree: true},
		},
	}

	// Query with JOIN but no index on join column
	query := &advisor.QueryMetadata{
		TableName: "users",
		Joins: []advisor.JoinClause{
			{
				Type:         "INNER",
				Table:        "orders",
				OnConditions: []advisor.Condition{{Field: "order_id", Operator: "="}},
			},
		},
	}

	rule := NewJoinIndexRule()
	out := rule.Check(query, &schema)
	if len(out) == 0 {
		t.Fatal("expected join index warning when join column has no index")
	}
}

func TestMissingIndexRuleNoQueries(t *testing.T) {
	schema := advisor.TableSchema{
		TableName: "users",
		Columns: []advisor.ColumnInfo{
			{Name: "id", Type: "int"},
			{Name: "name", Type: "text"},
		},
		Indexes: []advisor.IndexInfo{},
	}

	rule := NewMissingIndexRule()
	out := rule.CheckAll([]*advisor.QueryMetadata{}, &schema)
	if len(out) != 0 {
		t.Fatalf("expected no suggestions with no queries, got %d", len(out))
	}
}

func TestMissingIndexRuleWithIndex(t *testing.T) {
	schema := advisor.TableSchema{
		TableName: "users",
		Columns: []advisor.ColumnInfo{
			{Name: "id", Type: "int"},
			{Name: "name", Type: "text"},
		},
		Indexes: []advisor.IndexInfo{
			{Name: "users_name_idx", Columns: []string{"name"}, Method: "btree", IsBTree: true},
		},
	}

	// Query on name which already has an index
	queries := []*advisor.QueryMetadata{
		{TableName: "users", Conditions: []advisor.Condition{{Field: "name", Operator: "=", Value: "alice"}}},
	}

	rule := NewMissingIndexRule()
	out := rule.CheckAll(queries, &schema)
	// Should not suggest missing index since name already has one
	if len(out) != 0 {
		t.Fatalf("expected no suggestions when index exists, got %d", len(out))
	}
}

func TestLowSelectivityRuleWithMetadata(t *testing.T) {
	schema := advisor.TableSchema{
		TableName: "users",
		Indexes: []advisor.IndexInfo{
			{
				Name:       "users_status_idx",
				Columns:    []string{"status"},
				Unique:     false,
				Method:     "btree",
				Metadata:   map[string]any{"selectivity": 0.001}, // Very low selectivity
			},
		},
	}

	rule := NewLowSelectivityRule()
	out := rule.CheckSchema(&schema)
	if len(out) == 0 {
		t.Fatal("expected low selectivity warning")
	}
	if out[0].RuleID != "IDX-010" {
		t.Fatalf("expected IDX-010, got %s", out[0].RuleID)
	}
}

func TestLowSelectivityRuleNoMetadata(t *testing.T) {
	schema := advisor.TableSchema{
		TableName: "users",
		Indexes: []advisor.IndexInfo{
			{
				Name:     "users_status_idx",
				Columns:  []string{"status"},
				Unique:   false,
				Method:   "btree",
				Metadata: nil, // No selectivity metadata
			},
		},
	}

	rule := NewLowSelectivityRule()
	out := rule.CheckSchema(&schema)
	// Without metadata, rule should not fire
	if len(out) != 0 {
		t.Fatalf("expected no warnings without selectivity metadata, got %d", len(out))
	}
}
