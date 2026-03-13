package postgres

import "gore/dialect"

// Dialector is the PostgreSQL dialector skeleton.
type Dialector struct{}

// Name returns the dialect name.
func (d *Dialector) Name() string { return "postgres" }

// BuildSelect builds a SELECT statement from QueryAST.
func (d *Dialector) BuildSelect(ast *dialect.QueryAST) (string, []any, error) {
    _ = ast
    return "", nil, nil
}

// BuildInsert builds an INSERT statement from InsertAST.
func (d *Dialector) BuildInsert(ast *dialect.InsertAST) (string, []any, error) {
    _ = ast
    return "", nil, nil
}

// BuildUpdate builds an UPDATE statement from UpdateAST.
func (d *Dialector) BuildUpdate(ast *dialect.UpdateAST) (string, []any, error) {
    _ = ast
    return "", nil, nil
}

// BuildDelete builds a DELETE statement from DeleteAST.
func (d *Dialector) BuildDelete(ast *dialect.DeleteAST) (string, []any, error) {
    _ = ast
    return "", nil, nil
}
