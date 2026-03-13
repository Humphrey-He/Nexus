package advisor

// Rule evaluates a query against a schema and returns suggestions.
type Rule interface {
	ID() string
	Name() string
	Description() string
	Severity() Severity
	Check(query *QueryMetadata, schema *TableSchema) []Suggestion
	WhyDoc() string
}

// Engine applies rules and aggregates suggestions.
type Engine struct {
	rules []Rule
}

// NewEngine creates a rule engine.
func NewEngine(rules ...Rule) *Engine {
	return &Engine{rules: rules}
}

// Analyze runs all rules and returns suggestions.
func (e *Engine) Analyze(query *QueryMetadata, schema *TableSchema) []Suggestion {
	if query == nil || schema == nil {
		return nil
	}

	var out []Suggestion
	for _, rule := range e.rules {
		suggestions := rule.Check(query, schema)
		if len(suggestions) > 0 {
			out = append(out, suggestions...)
		}
	}

	return out
}
