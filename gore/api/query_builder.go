package api

import (
	"fmt"
	"strings"

	"gore/dialect"
)

// Predicate represents a strongly-typed where predicate.
type Predicate[T any] func(*dialect.QueryAST)

// JoinType represents the type of JOIN.
type JoinType = dialect.JoinType

const (
	JoinInner = dialect.JoinInner
	JoinLeft  = dialect.JoinLeft
	JoinRight = dialect.JoinRight
	JoinFull  = dialect.JoinFull
)

// Query is a typed query builder.
type Query[T any] struct {
	ctx      *Context
	where    []Predicate[T]
	limit    int
	offset   int
	order    []string
	table    string
	joins    []dialect.JoinClause
	groupBy  []string
	having   []string
	distinct bool
}

// Join appends an INNER JOIN clause.
func (q *Query[T]) Join(table, on string) *Query[T] {
	q.joins = append(q.joins, dialect.JoinClause{
		Type:  dialect.JoinInner,
		Table: table,
		On:    on,
	})
	return q
}

// LeftJoin appends a LEFT JOIN clause.
func (q *Query[T]) LeftJoin(table, on string) *Query[T] {
	q.joins = append(q.joins, dialect.JoinClause{
		Type:  dialect.JoinLeft,
		Table: table,
		On:    on,
	})
	return q
}

// RightJoin appends a RIGHT JOIN clause.
func (q *Query[T]) RightJoin(table, on string) *Query[T] {
	q.joins = append(q.joins, dialect.JoinClause{
		Type:  dialect.JoinRight,
		Table: table,
		On:    on,
	})
	return q
}

// FullJoin appends a FULL OUTER JOIN clause.
func (q *Query[T]) FullJoin(table, on string) *Query[T] {
	q.joins = append(q.joins, dialect.JoinClause{
		Type:  dialect.JoinFull,
		Table: table,
		On:    on,
	})
	return q
}

// LateralJoin appends a LATERAL JOIN clause.
func (q *Query[T]) LateralJoin(table, on string) *Query[T] {
	q.joins = append(q.joins, dialect.JoinClause{
		Type:    dialect.JoinInner,
		Table:   table,
		On:      on,
		Lateral: true,
	})
	return q
}

// Where appends a predicate.
func (q *Query[T]) Where(predicate Predicate[T]) *Query[T] {
	if predicate != nil {
		q.where = append(q.where, predicate)
	}
	return q
}

// From sets the target table name.
func (q *Query[T]) From(table string) *Query[T] {
	if table != "" {
		q.table = table
	}
	return q
}

// WhereField appends a simple field predicate for static analysis.
func (q *Query[T]) WhereField(field, op string, value any) *Query[T] {
	if field == "" || op == "" {
		return q
	}

	q.where = append(q.where, func(ast *dialect.QueryAST) {
		ast.Where = append(ast.Where, fmt.Sprintf("%s %s ?", field, op))
	})
	_ = value
	return q
}

// WhereIn appends an IN predicate for static analysis.
func (q *Query[T]) WhereIn(field string, values ...any) *Query[T] {
	if field == "" || len(values) == 0 {
		return q
	}

	placeholders := strings.Repeat("?,", len(values))
	placeholders = strings.TrimRight(placeholders, ",")

	q.where = append(q.where, func(ast *dialect.QueryAST) {
		ast.Where = append(ast.Where, fmt.Sprintf("%s IN (%s)", field, placeholders))
	})
	return q
}

// WhereLike appends a LIKE predicate for static analysis.
func (q *Query[T]) WhereLike(field string, pattern string) *Query[T] {
	if field == "" || pattern == "" {
		return q
	}

	q.where = append(q.where, func(ast *dialect.QueryAST) {
		ast.Where = append(ast.Where, fmt.Sprintf("%s LIKE ?", field))
	})
	return q
}

// Having appends a HAVING clause.
func (q *Query[T]) Having(condition string) *Query[T] {
	if condition != "" {
		q.having = append(q.having, condition)
	}
	return q
}

// OrderBy appends an order expression.
func (q *Query[T]) OrderBy(expr string) *Query[T] {
	if expr != "" {
		q.order = append(q.order, expr)
	}
	return q
}

// Limit sets the maximum number of rows.
func (q *Query[T]) Limit(n int) *Query[T] {
	q.limit = n
	return q
}

// Offset sets the number of rows to skip.
func (q *Query[T]) Offset(n int) *Query[T] {
	q.offset = n
	return q
}

// GroupBy sets the GROUP BY fields.
func (q *Query[T]) GroupBy(fields ...string) *Query[T] {
	if len(fields) > 0 {
		q.groupBy = append(q.groupBy, fields...)
	}
	return q
}

// Distinct enables DISTINCT selection.
func (q *Query[T]) Distinct() *Query[T] {
	q.distinct = true
	return q
}

// ToAST builds a QueryAST (skeleton).
func (q *Query[T]) ToAST() *dialect.QueryAST {
	ast := &dialect.QueryAST{
		Table: q.table,
	}
	for _, p := range q.where {
		p(ast)
	}
	ast.Limit = q.limit
	ast.Offset = q.offset
	ast.OrderBy = append(ast.OrderBy, q.order...)
	ast.Joins = q.joins
	ast.GroupBy = q.groupBy
	ast.Having = q.having
	ast.Distinct = q.distinct
	return ast
}
