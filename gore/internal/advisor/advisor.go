package advisor

// Severity represents the impact of a suggestion.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarn
	SeverityHigh
	SeverityCritical
)

// Suggestion explains an index-related risk and guidance.
type Suggestion struct {
	RuleID         string
	Severity       Severity
	Message        string
	Reason         string
	Evidence       []string
	Recommendation string
	Confidence     float64
	SourceFile     string
	LineNumber     int
	Tags           []string
}

// Advisor analyzes query metadata with schema metadata and returns suggestions.
type Advisor interface {
	Name() string
	Analyze(query *QueryMetadata, schema *TableSchema) ([]Suggestion, error)
}

// QueryMetadata captures semantic info extracted from queries.
type QueryMetadata struct {
	TableName  string
	Columns    []string
	Conditions []Condition
	OrderBy    []OrderField
	GroupBy    []string
	Joins      []JoinClause
	Limit      *int
	Offset     *int
	IsDistinct bool
	SourceFile string
	LineNumber int
}

// Condition represents a WHERE clause predicate.
type Condition struct {
	Field      string
	Operator   string
	Value      any
	ValueType  string
	IsFunction bool
	FuncName   string
	IsNegated  bool
}

// OrderField represents ORDER BY fields.
type OrderField struct {
	Field     string
	Direction string
}

// JoinClause represents JOIN information.
type JoinClause struct {
	Type         string
	Table        string
	OnConditions []Condition
}

// TableSchema contains table metadata for analysis.
type TableSchema struct {
	TableName string
	Indexes   []IndexInfo
}

// IndexInfo describes an index for advisor analysis.
type IndexInfo struct {
	Name    string
	Columns []string
	Unique  bool
	Method  string
	IsBTree bool
}
