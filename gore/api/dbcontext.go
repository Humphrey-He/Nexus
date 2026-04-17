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
	Transaction(ctx context.Context, fn func(DbContext) error) error
}

// Context is the default DbContext implementation.
type Context struct {
	exec       executor.Executor
	meta       *metadata.Registry
	dialector  dialect.Dialector
	trackingOn bool
	tracker    *tracker.Tracker
	metrics    Metrics
	logger     Logger
	txDepth    int // Nested transaction depth
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
		logger:     &nopLogger{},
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

// WithLogger attaches a logger.
func (c *Context) WithLogger(l Logger) *Context {
	clone := *c
	clone.logger = l
	return &clone
}

// SaveChanges commits the tracked changes.
func (c *Context) SaveChanges(ctx context.Context) (int, error) {
	if !c.trackingOn {
		return 0, ErrTrackingDisabled
	}

	start := time.Now()
	c.logger.Debug("SaveChanges started", "txDepth", c.txDepth)

	changes, err := c.tracker.DetectChanges()
	if c.metrics != nil {
		c.metrics.ObserveChangeTracking(time.Since(start), len(changes))
	}
	if err != nil {
		c.logger.Error("SaveChanges failed", "error", err)
		return 0, err
	}

	c.logger.Debug("SaveChanges completed", "changes", len(changes))
	return len(changes), nil
}

// Transaction executes fn within a database transaction.
// If fn returns an error, the transaction is rolled back.
// If fn succeeds, the transaction is committed.
func (c *Context) Transaction(ctx context.Context, fn func(DbContext) error) error {
	if c.txDepth > 0 {
		// Nested transaction - execute without explicit BEGIN/COMMIT
		c.txDepth++
		c.logger.Debug("nested transaction", "depth", c.txDepth)
		return fn(c)
	}

	c.txDepth = 1
	c.logger.Debug("transaction started")

	// Begin transaction
	if _, err := c.exec.Exec(ctx, "BEGIN", nil); err != nil {
		return err
	}

	// Create a new context for the transaction with tracking
	txCtx := &Context{
		exec:       c.exec,
		meta:       c.meta,
		dialector:  c.dialector,
		trackingOn: true,
		tracker:    tracker.New(),
		metrics:    c.metrics,
		logger:     c.logger,
		txDepth:    1,
	}

	// Execute the function
	err := fn(txCtx)

	if err != nil {
		c.logger.Debug("transaction rollback", "error", err)
		// Rollback
		txCtx.exec.Exec(ctx, "ROLLBACK", nil)
		return err
	}

	// Commit
	c.logger.Debug("transaction commit")
	if _, err := txCtx.exec.Exec(ctx, "COMMIT", nil); err != nil {
		txCtx.exec.Exec(ctx, "ROLLBACK", nil)
		return err
	}

	c.txDepth = 0
	return nil
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
