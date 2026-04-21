package advisor

import (
	"testing"
)

func TestSeverityValues(t *testing.T) {
	tests := []struct {
		severity Severity
		expected int
	}{
		{SeverityInfo, 0},
		{SeverityWarn, 1},
		{SeverityHigh, 2},
		{SeverityCritical, 3},
	}

	for _, tt := range tests {
		if int(tt.severity) != tt.expected {
			t.Errorf("expected %d, got %d", tt.expected, tt.severity)
		}
	}
}

func TestSuggestionStruct(t *testing.T) {
	s := Suggestion{
		RuleID:         "TEST-001",
		Severity:       SeverityWarn,
		Message:        "test message",
		Reason:         "test reason",
		Evidence:       []string{"evidence1", "evidence2"},
		Recommendation: "test recommendation",
		Confidence:     0.95,
		SourceFile:    "test.go",
		LineNumber:    42,
		Tags:          []string{"tag1", "tag2"},
	}

	if s.RuleID != "TEST-001" {
		t.Errorf("expected RuleID=TEST-001, got %s", s.RuleID)
	}
	if s.Severity != SeverityWarn {
		t.Errorf("expected Severity=Warn, got %v", s.Severity)
	}
	if s.Message != "test message" {
		t.Errorf("expected Message=test message, got %s", s.Message)
	}
	if len(s.Evidence) != 2 {
		t.Errorf("expected 2 evidence items, got %d", len(s.Evidence))
	}
	if s.Confidence != 0.95 {
		t.Errorf("expected Confidence=0.95, got %f", s.Confidence)
	}
	if s.SourceFile != "test.go" {
		t.Errorf("expected SourceFile=test.go, got %s", s.SourceFile)
	}
	if s.LineNumber != 42 {
		t.Errorf("expected LineNumber=42, got %d", s.LineNumber)
	}
	if len(s.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(s.Tags))
	}
}

func TestQueryMetadata(t *testing.T) {
	limit := 10
	offset := 20
	qm := QueryMetadata{
		TableName:  "users",
		Columns:   []string{"id", "name", "email"},
		Conditions: []Condition{
			{Field: "status", Operator: "=", Value: "active"},
			{Field: "age", Operator: ">", Value: 18},
		},
		OrderBy: []OrderField{
			{Field: "name", Direction: "ASC"},
			{Field: "created_at", Direction: "DESC"},
		},
		GroupBy:    []string{"status"},
		Joins:      []JoinClause{},
		Limit:      &limit,
		Offset:     &offset,
		IsDistinct: true,
		SourceFile: "query.go",
		LineNumber: 100,
	}

	if qm.TableName != "users" {
		t.Errorf("expected TableName=users, got %s", qm.TableName)
	}
	if len(qm.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(qm.Columns))
	}
	if len(qm.Conditions) != 2 {
		t.Errorf("expected 2 conditions, got %d", len(qm.Conditions))
	}
	if len(qm.OrderBy) != 2 {
		t.Errorf("expected 2 order fields, got %d", len(qm.OrderBy))
	}
	if *qm.Limit != 10 {
		t.Errorf("expected Limit=10, got %d", *qm.Limit)
	}
	if *qm.Offset != 20 {
		t.Errorf("expected Offset=20, got %d", *qm.Offset)
	}
	if !qm.IsDistinct {
		t.Error("expected IsDistinct=true")
	}
}

func TestCondition(t *testing.T) {
	c := Condition{
		Field:      "email",
		Operator:   "LIKE",
		Value:      "%@example.com",
		ValueType:  "string",
		IsFunction: true,
		FuncName:   "LOWER",
		IsNegated:  false,
	}

	if c.Field != "email" {
		t.Errorf("expected Field=email, got %s", c.Field)
	}
	if c.Operator != "LIKE" {
		t.Errorf("expected Operator=LIKE, got %s", c.Operator)
	}
	if c.Value != "%@example.com" {
		t.Errorf("expected Value=%s, got %s", "%@example.com", c.Value)
	}
	if !c.IsFunction {
		t.Error("expected IsFunction=true")
	}
	if c.FuncName != "LOWER" {
		t.Errorf("expected FuncName=LOWER, got %s", c.FuncName)
	}
}

func TestOrderField(t *testing.T) {
	of := OrderField{
		Field:     "created_at",
		Direction: "DESC",
	}

	if of.Field != "created_at" {
		t.Errorf("expected Field=created_at, got %s", of.Field)
	}
	if of.Direction != "DESC" {
		t.Errorf("expected Direction=DESC, got %s", of.Direction)
	}
}

func TestJoinClause(t *testing.T) {
	jc := JoinClause{
		Type:  "INNER",
		Table: "orders",
		OnConditions: []Condition{
			{Field: "user_id", Operator: "="},
		},
	}

	if jc.Type != "INNER" {
		t.Errorf("expected Type=INNER, got %s", jc.Type)
	}
	if jc.Table != "orders" {
		t.Errorf("expected Table=orders, got %s", jc.Table)
	}
	if len(jc.OnConditions) != 1 {
		t.Errorf("expected 1 on condition, got %d", len(jc.OnConditions))
	}
}

func TestTableSchema(t *testing.T) {
	schema := TableSchema{
		TableName: "users",
		Columns: []ColumnInfo{
			{Name: "id", Type: "int"},
			{Name: "name", Type: "varchar(255)"},
		},
		Indexes: []IndexInfo{
			{Name: "idx_name", Columns: []string{"name"}, Unique: false, Method: "btree", IsBTree: true},
		},
	}

	if schema.TableName != "users" {
		t.Errorf("expected TableName=users, got %s", schema.TableName)
	}
	if len(schema.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(schema.Columns))
	}
	if len(schema.Indexes) != 1 {
		t.Errorf("expected 1 index, got %d", len(schema.Indexes))
	}
}

func TestColumnInfo(t *testing.T) {
	ci := ColumnInfo{
		Name: "email",
		Type: "varchar(255)",
	}

	if ci.Name != "email" {
		t.Errorf("expected Name=email, got %s", ci.Name)
	}
	if ci.Type != "varchar(255)" {
		t.Errorf("expected Type=varchar(255), got %s", ci.Type)
	}
}

func TestIndexInfo(t *testing.T) {
	ii := IndexInfo{
		Name:      "idx_email",
		Columns:   []string{"email"},
		Unique:    true,
		Method:    "btree",
		IsBTree:   true,
		IsVisible: true,
		Metadata:  map[string]any{"table": "users"},
	}

	if ii.Name != "idx_email" {
		t.Errorf("expected Name=idx_email, got %s", ii.Name)
	}
	if !ii.Unique {
		t.Error("expected Unique=true")
	}
	if ii.Method != "btree" {
		t.Errorf("expected Method=btree, got %s", ii.Method)
	}
	if !ii.IsBTree {
		t.Error("expected IsBTree=true")
	}
	if !ii.IsVisible {
		t.Error("expected IsVisible=true")
	}
	if ii.Metadata["table"] != "users" {
		t.Errorf("expected Metadata[table]=users, got %v", ii.Metadata["table"])
	}
}

// ===============================================================================
// Engine Tests
// ===============================================================================

type testRule struct {
	id   string
	name string
}

func (r *testRule) ID() string          { return r.id }
func (r *testRule) Name() string        { return r.name }
func (r *testRule) Description() string { return "test rule" }
func (r *testRule) Severity() Severity { return SeverityWarn }
func (r *testRule) Check(query *QueryMetadata, schema *TableSchema) []Suggestion {
	if query == nil || schema == nil {
		return nil
	}
	return []Suggestion{
		{RuleID: r.id, Severity: SeverityWarn, Message: "test suggestion"},
	}
}
func (r *testRule) WhyDoc() string { return "test why" }

func TestNewEngine(t *testing.T) {
	rule1 := &testRule{id: "RULE-001", name: "rule1"}
	rule2 := &testRule{id: "RULE-002", name: "rule2"}

	engine := NewEngine(rule1, rule2)
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
	if len(engine.rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(engine.rules))
	}
}

func TestEngineAnalyzeNilQuery(t *testing.T) {
	engine := NewEngine(&testRule{id: "RULE-001", name: "rule1"})
	schema := &TableSchema{TableName: "users"}

	suggestions := engine.Analyze(nil, schema)
	if suggestions != nil {
		t.Errorf("expected nil suggestions for nil query, got %d", len(suggestions))
	}
}

func TestEngineAnalyzeNilSchema(t *testing.T) {
	engine := NewEngine(&testRule{id: "RULE-001", name: "rule1"})
	query := &QueryMetadata{TableName: "users"}

	suggestions := engine.Analyze(query, nil)
	if suggestions != nil {
		t.Errorf("expected nil suggestions for nil schema, got %d", len(suggestions))
	}
}

func TestEngineAnalyze(t *testing.T) {
	engine := NewEngine(
		&testRule{id: "RULE-001", name: "rule1"},
		&testRule{id: "RULE-002", name: "rule2"},
	)
	query := &QueryMetadata{TableName: "users"}
	schema := &TableSchema{TableName: "users"}

	suggestions := engine.Analyze(query, schema)
	if len(suggestions) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(suggestions))
	}

	if suggestions[0].RuleID != "RULE-001" {
		t.Errorf("expected RuleID=RULE-001, got %s", suggestions[0].RuleID)
	}
	if suggestions[1].RuleID != "RULE-002" {
		t.Errorf("expected RuleID=RULE-002, got %s", suggestions[1].RuleID)
	}
}

func TestEngineAnalyzeNoRules(t *testing.T) {
	engine := NewEngine()
	query := &QueryMetadata{TableName: "users"}
	schema := &TableSchema{TableName: "users"}

	suggestions := engine.Analyze(query, schema)
	if suggestions != nil {
		t.Errorf("expected nil suggestions for no rules, got %d", len(suggestions))
	}
}

func TestEngineAnalyzeRuleReturnsNil(t *testing.T) {
	engine := NewEngine(&testRule{id: "RULE-001", name: "rule1"})
	query := &QueryMetadata{TableName: "users"}
	schema := &TableSchema{TableName: "users"}

	suggestions := engine.Analyze(query, schema)
	if suggestions == nil {
		t.Error("expected suggestions, got nil")
	}
}

// ===============================================================================
// Mock Rule for Testing
// ===============================================================================

type mockRule struct {
	id          string
	name        string
	description string
	severity    Severity
	checkFunc   func(*QueryMetadata, *TableSchema) []Suggestion
	whyDoc      string
}

func (r *mockRule) ID() string          { return r.id }
func (r *mockRule) Name() string       { return r.name }
func (r *mockRule) Description() string { return r.description }
func (r *mockRule) Severity() Severity { return r.severity }
func (r *mockRule) Check(query *QueryMetadata, schema *TableSchema) []Suggestion {
	if r.checkFunc != nil {
		return r.checkFunc(query, schema)
	}
	return nil
}
func (r *mockRule) WhyDoc() string { return r.whyDoc }

func TestMockRule(t *testing.T) {
	rule := &mockRule{
		id:          "MOCK-001",
		name:        "mock rule",
		description: "a mock rule for testing",
		severity:    SeverityHigh,
		whyDoc:      "mock why doc",
	}

	if rule.ID() != "MOCK-001" {
		t.Errorf("expected ID=MOCK-001, got %s", rule.ID())
	}
	if rule.Name() != "mock rule" {
		t.Errorf("expected Name=mock rule, got %s", rule.Name())
	}
	if rule.Description() != "a mock rule for testing" {
		t.Errorf("expected Description=a mock rule for testing, got %s", rule.Description())
	}
	if rule.Severity() != SeverityHigh {
		t.Errorf("expected Severity=High, got %v", rule.Severity())
	}
	if rule.WhyDoc() != "mock why doc" {
		t.Errorf("expected WhyDoc=mock why doc, got %s", rule.WhyDoc())
	}
}

func TestMockRuleWithCheckFunc(t *testing.T) {
	rule := &mockRule{
		id:   "MOCK-002",
		name: "mock rule with func",
		checkFunc: func(query *QueryMetadata, schema *TableSchema) []Suggestion {
			if query.TableName == "blocked" {
				return nil
			}
			return []Suggestion{
				{RuleID: "MOCK-002", Message: "found issue"},
			}
		},
	}

	// Test with normal table
	suggestions := rule.Check(&QueryMetadata{TableName: "users"}, &TableSchema{})
	if len(suggestions) != 1 {
		t.Errorf("expected 1 suggestion, got %d", len(suggestions))
	}

	// Test with blocked table
	suggestions = rule.Check(&QueryMetadata{TableName: "blocked"}, &TableSchema{})
	if suggestions != nil {
		t.Errorf("expected nil suggestions for blocked table, got %d", len(suggestions))
	}
}
