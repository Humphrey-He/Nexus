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
	RuleID         string   `json:"ruleId"`
	Severity       Severity `json:"severity"`
	Message        string   `json:"message"`
	Reason         string   `json:"reason"`
	Evidence       []string `json:"evidence,omitempty"`
	Recommendation string   `json:"recommendation,omitempty"`
	Confidence     float64  `json:"confidence"`
	SourceFile     string   `json:"sourceFile,omitempty"`
	LineNumber     int      `json:"lineNumber,omitempty"`
	Tags           []string `json:"tags,omitempty"`
}

// Advisor analyzes query metadata with schema metadata and returns suggestions.
type Advisor interface {
	Name() string
	Analyze(query *QueryMetadata, schema *TableSchema) ([]Suggestion, error)
}

// QueryMetadata captures semantic info extracted from queries.
type QueryMetadata struct {
	TableName  string       `json:"tableName"`
	Columns    []string     `json:"columns,omitempty"`
	Conditions []Condition  `json:"conditions,omitempty"`
	OrderBy    []OrderField `json:"orderBy,omitempty"`
	GroupBy    []string     `json:"groupBy,omitempty"`
	Joins      []JoinClause `json:"joins,omitempty"`
	Limit      *int         `json:"limit,omitempty"`
	Offset     *int         `json:"offset,omitempty"`
	IsDistinct bool         `json:"isDistinct,omitempty"`
	SourceFile string       `json:"sourceFile,omitempty"`
	LineNumber int          `json:"lineNumber,omitempty"`
}

// Condition represents a WHERE clause predicate.
type Condition struct {
	Field      string `json:"field"`
	Operator   string `json:"operator"`
	Value      any    `json:"value,omitempty"`
	ValueType  string `json:"valueType,omitempty"`
	IsFunction bool   `json:"isFunction,omitempty"`
	FuncName   string `json:"funcName,omitempty"`
	IsNegated  bool   `json:"isNegated,omitempty"`
}

// OrderField represents ORDER BY fields.
type OrderField struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

// JoinClause represents JOIN information.
type JoinClause struct {
	Type         string      `json:"type"`
	Table        string      `json:"table"`
	OnConditions []Condition `json:"onConditions,omitempty"`
}

// TableSchema contains table metadata for analysis.
type TableSchema struct {
	TableName string       `json:"tableName"`
	Columns   []ColumnInfo `json:"columns,omitempty"`
	Indexes   []IndexInfo  `json:"indexes,omitempty"`
}

// ColumnInfo describes a table column for type checks.
type ColumnInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// IndexInfo describes an index for advisor analysis.
type IndexInfo struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns,omitempty"`
	Unique  bool     `json:"unique"`
	Method  string   `json:"method"`
	IsBTree bool     `json:"isBtree"`
}
