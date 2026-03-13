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

// Attach starts tracking an existing entity.
func (s *DbSet[T]) Attach(entity *T) error {
	if s.ctx == nil {
		return ErrNotImplemented
	}
	if !s.ctx.trackingOn {
		return ErrTrackingDisabled
	}
	_, err := s.ctx.tracker.Attach(entity)
	return err
}

// Add registers a new entity as Added.
func (s *DbSet[T]) Add(entity *T) error {
	if s.ctx == nil {
		return ErrNotImplemented
	}
	if !s.ctx.trackingOn {
		return ErrTrackingDisabled
	}
	_, err := s.ctx.tracker.MarkAdded(entity)
	return err
}

// Remove registers an entity as Deleted.
func (s *DbSet[T]) Remove(entity *T) error {
	if s.ctx == nil {
		return ErrNotImplemented
	}
	if !s.ctx.trackingOn {
		return ErrTrackingDisabled
	}
	_, err := s.ctx.tracker.MarkDeleted(entity)
	return err
}

// Find is a convenience for primary key lookup (skeleton).
func (s *DbSet[T]) Find(pk any) (*T, error) {
	_ = pk
	return nil, ErrNotImplemented
}
