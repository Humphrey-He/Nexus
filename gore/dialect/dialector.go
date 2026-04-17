package dialect

// JoinType represents the type of JOIN.
type JoinType int

const (
	JoinInner JoinType = iota
	JoinLeft
	JoinRight
	JoinFull
)

// JoinClause represents a JOIN clause.
type JoinClause struct {
	Type     JoinType
	Table    string
	On       string
	Using    []string
	Lateral  bool
}

// QueryAST represents a select query abstract syntax tree.
type QueryAST struct {
	Table    string
	Columns  []string
	Joins    []JoinClause
	Where    []string
	OrderBy  []string
	GroupBy  []string
	Having   []string
	Limit    int
	Offset   int
	Distinct bool
}

// InsertAST represents an insert statement AST.
type InsertAST struct {
	Table    string
	Columns  []string
	Values   [][]any // For batch insert
}

// UpdateAST represents an update statement AST.
type UpdateAST struct {
	Table   string
	Columns []string
	Where   []string
}

// DeleteAST represents a delete statement AST.
type DeleteAST struct {
	Table string
	Where []string
}

// Dialector defines database dialect behaviors.
type Dialector interface {
	Name() string
	BuildSelect(ast *QueryAST) (string, []any, error)
	BuildInsert(ast *InsertAST) (string, []any, error)
	BuildUpdate(ast *UpdateAST) (string, []any, error)
	BuildDelete(ast *DeleteAST) (string, []any, error)
}
