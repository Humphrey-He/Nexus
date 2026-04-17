package mysql

import (
	"fmt"
	"strings"

	"gore/dialect"
)

// Dialector is the MySQL dialect implementation.
type Dialector struct{}

// Name returns the dialect name.
func (d *Dialector) Name() string { return "mysql" }

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
		sb.WriteString(d.quoteColumns(ast.Columns))
	} else {
		sb.WriteString("*")
	}

	// Table
	sb.WriteString(" FROM ")
	sb.WriteString(d.quoteIdentifier(ast.Table))

	// WHERE
	if len(ast.Where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(ast.Where, " AND "))
	}

	// GROUP BY
	if len(ast.GroupBy) > 0 {
		sb.WriteString(" GROUP BY ")
		sb.WriteString(d.quoteColumns(ast.GroupBy))
	}

	// ORDER BY
	if len(ast.OrderBy) > 0 {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(strings.Join(ast.OrderBy, ", "))
	}

	// LIMIT/OFFSET - MySQL uses LIMIT offset, count syntax
	if ast.Limit > 0 {
		if ast.Offset > 0 {
			sb.WriteString(fmt.Sprintf(" LIMIT %d, %d", ast.Offset, ast.Limit))
		} else {
			sb.WriteString(fmt.Sprintf(" LIMIT %d", ast.Limit))
		}
	} else if ast.Offset > 0 {
		// MySQL requires LIMIT when using OFFSET
		sb.WriteString(fmt.Sprintf(" LIMIT %d", ast.Offset))
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
	sb.WriteString(d.quoteIdentifier(ast.Table))
	sb.WriteString(" (")
	sb.WriteString(d.quoteColumns(ast.Columns))
	sb.WriteString(") VALUES (")

	placeholders := make([]string, len(ast.Columns))
	for i := range ast.Columns {
		placeholders[i] = "?"
	}
	sb.WriteString(strings.Join(placeholders, ", "))
	sb.WriteString(")")

	return sb.String(), nil, nil
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
	sb.WriteString(d.quoteIdentifier(ast.Table))
	sb.WriteString(" SET ")

	setClauses := make([]string, len(ast.Columns))
	for i, col := range ast.Columns {
		setClauses[i] = fmt.Sprintf("%s = ?", d.quoteIdentifier(col))
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
	sb.WriteString(d.quoteIdentifier(ast.Table))

	if len(ast.Where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(ast.Where, " AND "))
	}

	return sb.String(), nil, nil
}

// quoteIdentifier wraps an identifier with MySQL backticks.
func (d *Dialector) quoteIdentifier(name string) string {
	return "`" + name + "`"
}

// quoteColumns wraps column names with MySQL backticks.
func (d *Dialector) quoteColumns(columns []string) string {
	quoted := make([]string, len(columns))
	for i, col := range columns {
		quoted[i] = d.quoteIdentifier(col)
	}
	return strings.Join(quoted, ", ")
}
