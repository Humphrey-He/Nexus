package api

import (
    "context"

    "gore/dialect"
    "gore/internal/executor"
    "gore/internal/metadata"
)

// DbContext manages a Unit of Work lifecycle.
type DbContext interface {
    SaveChanges(ctx context.Context) (int, error)
    AsNoTracking() DbContext
}

// Context is the default DbContext implementation.
type Context struct {
    exec       executor.Executor
    meta       *metadata.Registry
    dialector  dialect.Dialector
    trackingOn bool
}

// NewContext creates a DbContext with required dependencies.
func NewContext(exec executor.Executor, meta *metadata.Registry, dialector dialect.Dialector) *Context {
    if meta == nil {
        meta = metadata.NewRegistry()
    }

    return &Context{
        exec:       exec,
        meta:       meta,
        dialector:  dialector,
        trackingOn: true,
    }
}

// Set returns a DbSet for type T.
// Note: Go does not support generic methods on types; use api.Set[T](ctx).
func Set[T any](c *Context) *DbSet[T] {
    return &DbSet[T]{ctx: c}
}

// SaveChanges commits the tracked changes.
func (c *Context) SaveChanges(ctx context.Context) (int, error) {
    _ = ctx
    return 0, ErrNotImplemented
}

// AsNoTracking disables change tracking for this context.
func (c *Context) AsNoTracking() DbContext {
    clone := *c
    clone.trackingOn = false
    return &clone
}

// Dialector returns the configured dialector.
func (c *Context) Dialector() dialect.Dialector {
    return c.dialector
}

// Executor returns the configured executor.
func (c *Context) Executor() executor.Executor {
    return c.exec
}

// Metadata returns the metadata registry.
func (c *Context) Metadata() *metadata.Registry {
    return c.meta
}
