package postgres

import (
	"fmt"
	"strings"

	"gore/dialect"
)

// Dialector is the PostgreSQL dialector.
type Dialector struct{}

// Name returns the dialect name.
func (d *Dialector) Name() string { return "postgres" }

// BuildSelect builds a SELECT statement from QueryAST.
func (d *Dialector) BuildSelect(ast *dialect.QueryAST) (string, []any, error) {
	if ast == nil {
		return "", nil, fmt.Errorf("ast is nil")
	}
	if ast.Table == "" {
		return "", nil, fmt.Errorf("table name is empty")
	}

	var sb strings.Builder
	sb.WriteString("SELECT ")

	// Columns
	if len(ast.Columns) > 0 {
		sb.WriteString(strings.Join(ast.Columns, ", "))
	} else {
		sb.WriteString("*")
	}

	// Table
	sb.WriteString(" FROM ")
	sb.WriteString(ast.Table)

	// WHERE
	if len(ast.Where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(ast.Where, " AND "))
	}

	// ORDER BY
	if len(ast.OrderBy) > 0 {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(strings.Join(ast.OrderBy, ", "))
	}

	// LIMIT
	if ast.Limit > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", ast.Limit))
	}

	// OFFSET
	if ast.Offset > 0 {
		sb.WriteString(fmt.Sprintf(" OFFSET %d", ast.Offset))
	}

	return sb.String(), nil, nil
}

// BuildInsert builds an INSERT statement from InsertAST.
func (d *Dialector) BuildInsert(ast *dialect.InsertAST) (string, []any, error) {
	if ast == nil {
		return "", nil, fmt.Errorf("ast is nil")
	}
	if ast.Table == "" {
		return "", nil, fmt.Errorf("table name is empty")
	}
	if len(ast.Columns) == 0 {
		return "", nil, fmt.Errorf("no columns specified")
	}

	var sb strings.Builder
	sb.WriteString("INSERT INTO ")
	sb.WriteString(ast.Table)
	sb.WriteString(" (")
	sb.WriteString(strings.Join(ast.Columns, ", "))
	sb.WriteString(") VALUES (")

	placeholders := make([]string, len(ast.Columns))
	args := make([]any, len(ast.Columns))
	for i := range ast.Columns {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = nil // Placeholder - actual values would come from entity
	}
	sb.WriteString(strings.Join(placeholders, ", "))
	sb.WriteString(")")

	return sb.String(), args, nil
}

// BuildUpdate builds an UPDATE statement from UpdateAST.
func (d *Dialector) BuildUpdate(ast *dialect.UpdateAST) (string, []any, error) {
	if ast == nil {
		return "", nil, fmt.Errorf("ast is nil")
	}
	if ast.Table == "" {
		return "", nil, fmt.Errorf("table name is empty")
	}
	if len(ast.Columns) == 0 {
		return "", nil, fmt.Errorf("no columns specified")
	}

	var sb strings.Builder
	sb.WriteString("UPDATE ")
	sb.WriteString(ast.Table)
	sb.WriteString(" SET ")

	setClauses := make([]string, len(ast.Columns))
	for i, col := range ast.Columns {
		setClauses[i] = fmt.Sprintf("%s = $%d", col, i+1)
	}
	sb.WriteString(strings.Join(setClauses, ", "))

	if len(ast.Where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(ast.Where, " AND "))
	}

	return sb.String(), nil, nil
}

// BuildDelete builds a DELETE statement from DeleteAST.
func (d *Dialector) BuildDelete(ast *dialect.DeleteAST) (string, []any, error) {
	if ast == nil {
		return "", nil, fmt.Errorf("ast is nil")
	}
	if ast.Table == "" {
		return "", nil, fmt.Errorf("table name is empty")
	}

	var sb strings.Builder
	sb.WriteString("DELETE FROM ")
	sb.WriteString(ast.Table)

	if len(ast.Where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(ast.Where, " AND "))
	}

	return sb.String(), nil, nil
}
