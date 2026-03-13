package api

import "gore/dialect"

// Predicate represents a strongly-typed where predicate.
type Predicate[T any] func(*dialect.QueryAST)

// Query is a typed query builder.
type Query[T any] struct {
	ctx    *Context
	where  []Predicate[T]
	limit  int
	offset int
	order  []string
}

// Where appends a predicate.
func (q *Query[T]) Where(predicate Predicate[T]) *Query[T] {
	if predicate != nil {
		q.where = append(q.where, predicate)
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

// ToAST builds a QueryAST (skeleton).
func (q *Query[T]) ToAST() *dialect.QueryAST {
	ast := &dialect.QueryAST{}
	for _, p := range q.where {
		p(ast)
	}
	ast.Limit = q.limit
	ast.Offset = q.offset
	ast.OrderBy = append(ast.OrderBy, q.order...)
	return ast
}
