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
		},
		Indexes: []advisor.IndexInfo{
			{Name: "users_id_idx", Columns: []string{"id"}, Unique: true, Method: "btree", IsBTree: true},
			{Name: "users_name_age_idx", Columns: []string{"name", "age"}, Method: "btree", IsBTree: true},
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
		{NewOrderByIndexRule(), "IDX-008", 1},
		{NewJoinIndexRule(), "IDX-009", 1},
	}

	for _, tc := range cases {
		out := tc.rule.Check(query, &schema)
		if len(out) < tc.min {
			t.Fatalf("%s expected >= %d suggestion(s), got %d", tc.name, tc.min, len(out))
		}
	}
}
