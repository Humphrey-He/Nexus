package main

import (
	"context"

	"gore/api"
	"gore/dialect"
	"gore/dialect/postgres"
	"gore/internal/executor"
)

type noopExecutor struct{}

type noopRows struct{}

type noopResult struct{}

func (n *noopExecutor) Query(ctx context.Context, query string, args ...any) (executor.Rows, error) {
	_ = ctx
	_ = query
	_ = args
	return &noopRows{}, nil
}

func (n *noopExecutor) Exec(ctx context.Context, query string, args ...any) (executor.Result, error) {
	_ = ctx
	_ = query
	_ = args
	return &noopResult{}, nil
}

func (r *noopRows) Next() bool                     { return false }
func (r *noopRows) Scan(dest ...any) error         { _ = dest; return nil }
func (r *noopRows) Close() error                   { return nil }
func (r *noopRows) Err() error                     { return nil }
func (r *noopResult) RowsAffected() (int64, error) { return 0, nil }

type User struct {
	ID   int
	Name string
}

func main() {
	ctx := api.NewContext(&noopExecutor{}, nil, &postgres.Dialector{})
	_ = api.Set[User](ctx).Query().Where(func(ast *dialect.QueryAST) { _ = ast })
}
