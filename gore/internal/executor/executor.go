package executor

import "context"

// Executor abstracts sql execution.
type Executor interface {
	Query(ctx context.Context, query string, args ...any) (Rows, error)
	Exec(ctx context.Context, query string, args ...any) (Result, error)
}

// Rows represents a result set iterator.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
	Err() error
}

// Result represents an execution result.
type Result interface {
	RowsAffected() (int64, error)
}
