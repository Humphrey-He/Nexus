package api

// DbSet provides strongly-typed access for entity T.
type DbSet[T any] struct {
    ctx *Context
}

// Where starts a filtered query.
func (s *DbSet[T]) Where(predicate Predicate[T]) *Query[T] {
    q := &Query[T]{
        ctx:   s.ctx,
        where: []Predicate[T]{predicate},
    }
    return q
}

// Query returns a new query without filters.
func (s *DbSet[T]) Query() *Query[T] {
    return &Query[T]{ctx: s.ctx}
}

// Find is a convenience for primary key lookup (skeleton).
func (s *DbSet[T]) Find(pk any) (*T, error) {
    _ = pk
    return nil, ErrNotImplemented
}
