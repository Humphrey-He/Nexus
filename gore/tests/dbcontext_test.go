package tests

import (
	"context"
	"testing"

	"gore/api"
	"gore/dialect"
	"gore/internal/executor"
)

type stubExecutor struct{}

type stubRows struct{}

type stubResult struct{}

func (s *stubExecutor) Query(ctx context.Context, query string, args ...any) (executor.Rows, error) {
	_ = ctx
	_ = query
	_ = args
	return &stubRows{}, nil
}

func (s *stubExecutor) Exec(ctx context.Context, query string, args ...any) (executor.Result, error) {
	_ = ctx
	_ = query
	_ = args
	return &stubResult{}, nil
}

func (r *stubRows) Next() bool                     { return false }
func (r *stubRows) Scan(dest ...any) error         { _ = dest; return nil }
func (r *stubRows) Close() error                   { return nil }
func (r *stubRows) Err() error                     { return nil }
func (r *stubResult) RowsAffected() (int64, error) { return 0, nil }

type stubDialector struct{}

func (d *stubDialector) Name() string { return "stub" }
func (d *stubDialector) BuildSelect(ast *dialect.QueryAST) (string, []any, error) {
	_ = ast
	return "", nil, nil
}
func (d *stubDialector) BuildInsert(ast *dialect.InsertAST) (string, []any, error) {
	_ = ast
	return "", nil, nil
}
func (d *stubDialector) BuildUpdate(ast *dialect.UpdateAST) (string, []any, error) {
	_ = ast
	return "", nil, nil
}
func (d *stubDialector) BuildDelete(ast *dialect.DeleteAST) (string, []any, error) {
	_ = ast
	return "", nil, nil
}

func newContext() *api.Context {
	return api.NewContext(&stubExecutor{}, nil, &stubDialector{})
}

type User struct {
	ID   int
	Name string
}

func TestDbContextSet(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	if set == nil {
		t.Fatalf("expected DbSet, got nil")
	}
}

func TestDbContextSaveChangesNoTracking(t *testing.T) {
	ctx := newContext().AsNoTracking()
	if _, err := ctx.SaveChanges(context.Background()); err != api.ErrTrackingDisabled {
		t.Fatalf("expected ErrTrackingDisabled, got %v", err)
	}
}

func TestDbContextSaveChangesNoChanges(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	user := &User{ID: 1, Name: "Alice"}
	if err := set.Attach(user); err != nil {
		t.Fatalf("attach failed: %v", err)
	}

	changes, err := ctx.SaveChanges(context.Background())
	if err != nil {
		t.Fatalf("save changes failed: %v", err)
	}
	if changes != 0 {
		t.Fatalf("expected 0 changes, got %d", changes)
	}
}
