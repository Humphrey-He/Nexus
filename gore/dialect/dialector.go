package dialect

// QueryAST represents a select query abstract syntax tree.
type QueryAST struct {
	Table   string
	Columns []string
	Where   []string
	OrderBy []string
	Limit   int
	Offset  int
}

// InsertAST represents an insert statement AST.
type InsertAST struct {
	Table   string
	Columns []string
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
