package mongodb

import (
	"fmt"
	"strings"

	"gore/dialect"
)

// Dialector is the MongoDB dialect implementation.
type Dialector struct{}

// Name returns the dialect name.
func (d *Dialector) Name() string { return "mongodb" }

// BuildSelect builds a find query from QueryAST.
// MongoDB uses find() with query documents, not SQL SELECT statements.
func (d *Dialector) BuildSelect(ast *dialect.QueryAST) (string, []any, error) {
	if ast == nil {
		return "", nil, fmt.Errorf("ast is nil")
	}
	if ast.Table == "" {
		return "", nil, fmt.Errorf("collection name is empty")
	}

	var sb strings.Builder
	sb.WriteString("db.")
	sb.WriteString(ast.Table)
	sb.WriteString(".find(")

	// Build query document
	sb.WriteString(d.buildQueryDoc(ast))

	// Build projection if columns specified
	if len(ast.Columns) > 0 {
		sb.WriteString(", { ")
		sb.WriteString(d.buildProjection(ast.Columns))
		sb.WriteString(" }")
	}

	sb.WriteString(")")

	// Apply order
	if len(ast.OrderBy) > 0 {
		sb.WriteString(".sort(")
		sb.WriteString(d.buildSort(ast.OrderBy))
		sb.WriteString(")")
	}

	// Apply limit
	if ast.Limit > 0 {
		sb.WriteString(fmt.Sprintf(".limit(%d)", ast.Limit))
	}

	// Apply skip/offset
	if ast.Offset > 0 {
		sb.WriteString(fmt.Sprintf(".skip(%d)", ast.Offset))
	}

	return sb.String(), nil, nil
}

// buildQueryDoc builds a MongoDB query document from WHERE clauses.
func (d *Dialector) buildQueryDoc(ast *dialect.QueryAST) string {
	if len(ast.Where) == 0 {
		return "{}"
	}

	var sb strings.Builder
	sb.WriteString("{ ")
	for i, cond := range ast.Where {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(cond)
	}
	sb.WriteString(" }")
	return sb.String()
}

// buildProjection builds a MongoDB projection document for selected columns.
func (d *Dialector) buildProjection(columns []string) string {
	var sb strings.Builder
	for i, col := range columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%s: 1", col))
	}
	return sb.String()
}

// buildSort builds a MongoDB sort document.
func (d *Dialector) buildSort(orderBy []string) string {
	var sb strings.Builder
	sb.WriteString("{ ")
	for i, order := range orderBy {
		if i > 0 {
			sb.WriteString(", ")
		}
		// MongoDB sort syntax: { field: 1 } for ascending, { field: -1 } for descending
		// Default to ascending; descending would be indicated by "field DESC"
		if strings.HasSuffix(order, " DESC") {
			field := strings.TrimSuffix(order, " DESC")
			sb.WriteString(fmt.Sprintf("%s: -1", field))
		} else if strings.HasSuffix(order, " ASC") {
			field := strings.TrimSuffix(order, " ASC")
			sb.WriteString(fmt.Sprintf("%s: 1", field))
		} else {
			sb.WriteString(fmt.Sprintf("%s: 1", order))
		}
	}
	sb.WriteString(" }")
	return sb.String()
}

// BuildInsert builds an insertOne or insertMany query from InsertAST.
func (d *Dialector) BuildInsert(ast *dialect.InsertAST) (string, []any, error) {
	if ast == nil {
		return "", nil, fmt.Errorf("ast is nil")
	}
	if ast.Table == "" {
		return "", nil, fmt.Errorf("collection name is empty")
	}
	if len(ast.Columns) == 0 {
		return "", nil, fmt.Errorf("no columns specified")
	}

	if len(ast.Values) == 1 {
		// Single insert - use insertOne
		var sb strings.Builder
		sb.WriteString("db.")
		sb.WriteString(ast.Table)
		sb.WriteString(".insertOne({ ")
		sb.WriteString(d.buildDocument(ast.Columns, ast.Values[0]))
		sb.WriteString(" })")
		return sb.String(), nil, nil
	}

	// Batch insert - use insertMany
	var sb strings.Builder
	sb.WriteString("db.")
	sb.WriteString(ast.Table)
	sb.WriteString(".insertMany([")
	for i, row := range ast.Values {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("{ ")
		sb.WriteString(d.buildDocument(ast.Columns, row))
		sb.WriteString(" }")
	}
	sb.WriteString("])")
	return sb.String(), nil, nil
}

// buildDocument builds a MongoDB document from columns and values.
func (d *Dialector) buildDocument(columns []string, values []any) string {
	var sb strings.Builder
	for i, col := range columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%s: %v", col, values[i]))
	}
	return sb.String()
}

// BuildUpdate builds an updateOne query from UpdateAST.
func (d *Dialector) BuildUpdate(ast *dialect.UpdateAST) (string, []any, error) {
	if ast == nil {
		return "", nil, fmt.Errorf("ast is nil")
	}
	if ast.Table == "" {
		return "", nil, fmt.Errorf("collection name is empty")
	}
	if len(ast.Columns) == 0 {
		return "", nil, fmt.Errorf("no columns specified")
	}

	var sb strings.Builder
	sb.WriteString("db.")
	sb.WriteString(ast.Table)
	sb.WriteString(".updateOne(")

	// Build filter from WHERE
	if len(ast.Where) > 0 {
		sb.WriteString("{ ")
		sb.WriteString(strings.Join(ast.Where, ", "))
		sb.WriteString(" }")
	} else {
		sb.WriteString("{}")
	}

	// Build update document
	sb.WriteString(", { $set: { ")
	setClauses := make([]string, len(ast.Columns))
	for i, col := range ast.Columns {
		setClauses[i] = fmt.Sprintf("%s: ?", col)
	}
	sb.WriteString(strings.Join(setClauses, ", "))
	sb.WriteString(" } })")

	return sb.String(), nil, nil
}

// BuildDelete builds a deleteOne query from DeleteAST.
func (d *Dialector) BuildDelete(ast *dialect.DeleteAST) (string, []any, error) {
	if ast == nil {
		return "", nil, fmt.Errorf("ast is nil")
	}
	if ast.Table == "" {
		return "", nil, fmt.Errorf("collection name is empty")
	}

	var sb strings.Builder
	sb.WriteString("db.")
	sb.WriteString(ast.Table)
	sb.WriteString(".deleteOne(")

	// Build filter from WHERE
	if len(ast.Where) > 0 {
		sb.WriteString("{ ")
		sb.WriteString(strings.Join(ast.Where, ", "))
		sb.WriteString(" }")
	} else {
		sb.WriteString("{}")
	}

	sb.WriteString(")")

	return sb.String(), nil, nil
}
