package executor

import (
	"context"
	"database/sql"
	"time"
)

// PoolConfig holds connection pool configuration.
type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultPoolConfig returns default pool configuration.
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	}
}

// PooledExecutor wraps an Executor with connection pooling.
type PooledExecutor struct {
	db  *sql.DB
	cfg *PoolConfig
}

// NewPooledExecutor creates a new pooled executor from a database connection.
func NewPooledExecutor(db *sql.DB, cfg *PoolConfig) *PooledExecutor {
	if cfg == nil {
		cfg = DefaultPoolConfig()
	}

	// Configure pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	return &PooledExecutor{
		db:  db,
		cfg: cfg,
	}
}

// Query executes a query with connection pooling.
func (p *PooledExecutor) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRows{rows: rows}, nil
}

// Exec executes a statement with connection pooling.
func (p *PooledExecutor) Exec(ctx context.Context, query string, args ...any) (Result, error) {
	result, err := p.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlResult{result: result}, nil
}

// Close closes the underlying database connection.
func (p *PooledExecutor) Close() error {
	return p.db.Close()
}

// Stats returns connection pool statistics.
func (p *PooledExecutor) Stats() sql.DBStats {
	return p.db.Stats()
}

// sqlRows wraps sql.Rows.
type sqlRows struct {
	rows *sql.Rows
}

func (r *sqlRows) Next() bool {
	return r.rows.Next()
}

func (r *sqlRows) Scan(dest ...any) error {
	return r.rows.Scan(dest...)
}

func (r *sqlRows) Close() error {
	return r.rows.Close()
}

func (r *sqlRows) Err() error {
	return r.rows.Err()
}

// sqlResult wraps sql.Result.
type sqlResult struct {
	result sql.Result
}

func (r *sqlResult) RowsAffected() (int64, error) {
	return r.result.RowsAffected()
}
