package api

import (
	goreerrors "gore/internal/errors"
)

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

// AddBatch registers multiple new entities as Added.
func (s *DbSet[T]) AddBatch(entities []*T) error {
	if s.ctx == nil {
		return ErrNotImplemented
	}
	if !s.ctx.trackingOn {
		return ErrTrackingDisabled
	}
	if len(entities) == 0 {
		return goreerrors.InvalidInput("entities cannot be empty")
	}
	for _, entity := range entities {
		if entity == nil {
			return goreerrors.ErrNilEntity
		}
		_, err := s.ctx.tracker.MarkAdded(entity)
		if err != nil {
			return err
		}
	}
	return nil
}

// AttachBatch starts tracking multiple existing entities.
func (s *DbSet[T]) AttachBatch(entities []*T) error {
	if s.ctx == nil {
		return ErrNotImplemented
	}
	if !s.ctx.trackingOn {
		return ErrTrackingDisabled
	}
	if len(entities) == 0 {
		return goreerrors.InvalidInput("entities cannot be empty")
	}
	for _, entity := range entities {
		if entity == nil {
			return goreerrors.ErrNilEntity
		}
		_, err := s.ctx.tracker.Attach(entity)
		if err != nil {
			return err
		}
	}
	return nil
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

// RemoveBatch registers multiple entities as Deleted.
func (s *DbSet[T]) RemoveBatch(entities []*T) error {
	if s.ctx == nil {
		return ErrNotImplemented
	}
	if !s.ctx.trackingOn {
		return ErrTrackingDisabled
	}
	if len(entities) == 0 {
		return goreerrors.InvalidInput("entities cannot be empty")
	}
	for _, entity := range entities {
		if entity == nil {
			return goreerrors.ErrNilEntity
		}
		_, err := s.ctx.tracker.MarkDeleted(entity)
		if err != nil {
			return err
		}
	}
	return nil
}

// Find is a convenience for primary key lookup (skeleton).
func (s *DbSet[T]) Find(pk any) (*T, error) {
	_ = pk
	return nil, ErrNotImplemented
}

// FindBatch is a convenience for primary key lookup of multiple entities.
func (s *DbSet[T]) FindBatch(pks []any) ([]*T, error) {
	if len(pks) == 0 {
		return nil, goreerrors.InvalidInput("pks cannot be empty")
	}
	return nil, ErrNotImplemented
}

// Update updates multiple entities with the same changes.
func (s *DbSet[T]) Update(entities []*T, field string, value any) error {
	if s.ctx == nil {
		return ErrNotImplemented
	}
	if !s.ctx.trackingOn {
		return ErrTrackingDisabled
	}
	if len(entities) == 0 {
		return goreerrors.InvalidInput("entities cannot be empty")
	}
	if field == "" {
		return goreerrors.InvalidInput("field cannot be empty")
	}
	return ErrNotImplemented
}
