package api

import (
	"context"
	"time"

	"gore/dialect"
	"gore/internal/executor"
	"gore/internal/metadata"
	"gore/internal/tracker"
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
	tracker    *tracker.Tracker
	metrics    Metrics
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
		tracker:    tracker.New(),
	}
}

// Set returns a DbSet for type T.
// Note: Go does not support generic methods on types; use api.Set[T](ctx).
func Set[T any](c *Context) *DbSet[T] {
	return &DbSet[T]{ctx: c}
}

// WithMetrics attaches a metrics recorder.
func (c *Context) WithMetrics(m Metrics) *Context {
	clone := *c
	clone.metrics = m
	return &clone
}

// SaveChanges commits the tracked changes.
func (c *Context) SaveChanges(ctx context.Context) (int, error) {
	_ = ctx
	if !c.trackingOn {
		return 0, ErrTrackingDisabled
	}

	start := time.Now()
	changes, err := c.tracker.DetectChanges()
	if c.metrics != nil {
		c.metrics.ObserveChangeTracking(time.Since(start), len(changes))
	}
	if err != nil {
		return 0, err
	}

	return len(changes), nil
}

// AsNoTracking disables change tracking for this context.
func (c *Context) AsNoTracking() DbContext {
	clone := *c
	clone.trackingOn = false
	return &clone
}

// AsTracking enables change tracking for this context.
func (c *Context) AsTracking() DbContext {
	clone := *c
	clone.trackingOn = true
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

// Tracker returns the change tracker.
func (c *Context) Tracker() *tracker.Tracker {
	return c.tracker
}
