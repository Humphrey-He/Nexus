package indexadvisor

import "gore/dialect"

// Severity indicates issue severity level.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarn
	SeverityError
)

// Issue represents a potential index issue.
type Issue struct {
	Code     string
	Message  string
	Severity Severity
}

// IndexMeta holds metadata for an index.
type IndexMeta struct {
	Name    string
	Table   string
	Columns []string
	Unique  bool
}

// Catalog contains index metadata for analysis.
type Catalog struct {
	Indexes []IndexMeta
}

// Advisor analyzes queries for index issues.
type Advisor struct {
	catalog Catalog
}

// New creates an Advisor.
func New(catalog Catalog) *Advisor {
	return &Advisor{catalog: catalog}
}

// Analyze inspects a QueryAST and returns index issues.
func (a *Advisor) Analyze(ast *dialect.QueryAST) []Issue {
	_ = ast
	// TODO: parse conditions and match against leftmost-prefix rules.
	return nil
}
