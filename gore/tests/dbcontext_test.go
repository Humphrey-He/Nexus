package tests

import (
	"context"
	"testing"
	"time"

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

func TestDbContextSetGeneric(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	if set == nil {
		t.Fatalf("expected DbSet for User, got nil")
	}
	set2 := api.Set[Product](ctx)
	if set2 == nil {
		t.Fatalf("expected DbSet for Product, got nil")
	}
}

func TestNewContextNilMetadata(t *testing.T) {
	// NewContext with nil metadata should create a default registry
	ctx := api.NewContext(&stubExecutor{}, nil, &stubDialector{})
	if ctx == nil {
		t.Fatalf("expected non-nil context")
	}
	if ctx.Metadata() == nil {
		t.Fatalf("expected non-nil metadata registry")
	}
}

func TestContextAsNoTracking(t *testing.T) {
	ctx := newContext()
	noTrack := ctx.AsNoTracking()
	if noTrack == nil {
		t.Fatal("AsNoTracking returned nil")
	}
	// Should be a different context instance
	if noTrack == ctx {
		t.Fatal("AsNoTracking should return a clone")
	}
}

func TestContextAsTracking(t *testing.T) {
	ctx := newContext()
	track := ctx.AsTracking()
	if track == nil {
		t.Fatal("AsTracking returned nil")
	}
	if track == ctx {
		t.Fatal("AsTracking should return a clone")
	}
}

func TestContextDialector(t *testing.T) {
	ctx := newContext()
	d := ctx.Dialector()
	if d == nil {
		t.Fatal("expected non-nil dialector")
	}
	if d.Name() != "stub" {
		t.Fatalf("expected stub dialector, got %s", d.Name())
	}
}

func TestContextExecutor(t *testing.T) {
	ctx := newContext()
	exec := ctx.Executor()
	if exec == nil {
		t.Fatal("expected non-nil executor")
	}
}

func TestContextTracker(t *testing.T) {
	ctx := newContext()
	tr := ctx.Tracker()
	if tr == nil {
		t.Fatal("expected non-nil tracker")
	}
}

func TestDbSetAttach(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	user := &User{ID: 1, Name: "Alice"}
	if err := set.Attach(user); err != nil {
		t.Fatalf("attach failed: %v", err)
	}
	// Attach again should not fail (idempotent for unchanged entity)
	if err := set.Attach(user); err != nil {
		t.Fatalf("re-attach failed: %v", err)
	}
}

func TestDbSetAttachNoTracking(t *testing.T) {
	ctx := newContext()
	noTrackCtx := ctx.AsNoTracking()
	// AsNoTracking returns DbContext interface
	// When SaveChanges is called, it should return ErrTrackingDisabled
	_, err := noTrackCtx.SaveChanges(context.Background())
	if err != api.ErrTrackingDisabled {
		t.Fatalf("expected ErrTrackingDisabled, got %v", err)
	}
}

func TestDbSetAdd(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	user := &User{ID: 1, Name: "Bob"}
	if err := set.Add(user); err != nil {
		t.Fatalf("add failed: %v", err)
	}
}

func TestDbSetAddNoTracking(t *testing.T) {
	ctx := newContext()
	noTrackCtx := ctx.AsNoTracking()
	_, err := noTrackCtx.SaveChanges(context.Background())
	if err != api.ErrTrackingDisabled {
		t.Fatalf("expected ErrTrackingDisabled, got %v", err)
	}
}

func TestDbSetRemove(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	user := &User{ID: 1, Name: "Charlie"}
	if err := set.Remove(user); err != nil {
		t.Fatalf("remove failed: %v", err)
	}
}

func TestDbSetRemoveNoTracking(t *testing.T) {
	ctx := newContext()
	noTrackCtx := ctx.AsNoTracking()
	_, err := noTrackCtx.SaveChanges(context.Background())
	if err != api.ErrTrackingDisabled {
		t.Fatalf("expected ErrTrackingDisabled, got %v", err)
	}
}

func TestDbSetFind(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	_, err := set.Find(1)
	if err != api.ErrNotImplemented {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestDbSetWhere(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Where(func(ast *dialect.QueryAST) {
		ast.Where = append(ast.Where, "name = ?")
	})
	if q == nil {
		t.Fatal("expected non-nil query from Where")
	}
}

func TestQueryFrom(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Query().From("users")
	ast := q.ToAST()
	if ast.Table != "users" {
		t.Fatalf("expected table 'users', got %s", ast.Table)
	}
}

func TestQueryFromEmpty(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Query().From("")
	ast := q.ToAST()
	if ast.Table != "" {
		t.Fatalf("expected empty table, got %s", ast.Table)
	}
}

func TestQueryWhereField(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Query().From("users").WhereField("name", "=", "Alice")
	ast := q.ToAST()
	if len(ast.Where) != 1 {
		t.Fatalf("expected 1 where clause, got %d", len(ast.Where))
	}
	if ast.Where[0] != "name = ?" {
		t.Fatalf("expected 'name = ?', got %s", ast.Where[0])
	}
}

func TestQueryWhereFieldEmpty(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Query().From("users").WhereField("", "=", "Alice")
	ast := q.ToAST()
	if len(ast.Where) != 0 {
		t.Fatalf("expected 0 where clauses, got %d", len(ast.Where))
	}
}

func TestQueryWhereIn(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Query().From("users").WhereIn("id", 1, 2, 3)
	ast := q.ToAST()
	if len(ast.Where) != 1 {
		t.Fatalf("expected 1 where clause, got %d", len(ast.Where))
	}
	if ast.Where[0] != "id IN (?,?,?)" {
		t.Fatalf("expected 'id IN (?,?,?)', got %s", ast.Where[0])
	}
}

func TestQueryWhereInEmpty(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Query().From("users").WhereIn("id")
	ast := q.ToAST()
	if len(ast.Where) != 0 {
		t.Fatalf("expected 0 where clauses, got %d", len(ast.Where))
	}
}

func TestQueryWhereLike(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Query().From("users").WhereLike("name", "%lice%")
	ast := q.ToAST()
	if len(ast.Where) != 1 {
		t.Fatalf("expected 1 where clause, got %d", len(ast.Where))
	}
	if ast.Where[0] != "name LIKE ?" {
		t.Fatalf("expected 'name LIKE ?', got %s", ast.Where[0])
	}
}

func TestQueryWhereLikeEmpty(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Query().From("users").WhereLike("", "%lice%")
	ast := q.ToAST()
	if len(ast.Where) != 0 {
		t.Fatalf("expected 0 where clauses, got %d", len(ast.Where))
	}
}

func TestQueryOrderBy(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Query().From("users").OrderBy("name ASC").OrderBy("created_at DESC")
	ast := q.ToAST()
	if len(ast.OrderBy) != 2 {
		t.Fatalf("expected 2 order clauses, got %d", len(ast.OrderBy))
	}
	if ast.OrderBy[0] != "name ASC" {
		t.Fatalf("expected 'name ASC', got %s", ast.OrderBy[0])
	}
	if ast.OrderBy[1] != "created_at DESC" {
		t.Fatalf("expected 'created_at DESC', got %s", ast.OrderBy[1])
	}
}

func TestQueryLimit(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Query().From("users").Limit(10)
	ast := q.ToAST()
	if ast.Limit != 10 {
		t.Fatalf("expected limit 10, got %d", ast.Limit)
	}
}

func TestQueryOffset(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Query().From("users").Offset(20)
	ast := q.ToAST()
	if ast.Offset != 20 {
		t.Fatalf("expected offset 20, got %d", ast.Offset)
	}
}

func TestQueryGroupBy(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Query().From("users").GroupBy("status", "category")
	if q == nil {
		t.Fatal("expected non-nil query after GroupBy")
	}
	// Verify method chaining works
	q2 := q.OrderBy("name ASC")
	if q2 == nil {
		t.Fatal("expected non-nil query after OrderBy")
	}
}

func TestQueryDistinct(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Query().From("users").Distinct()
	if q == nil {
		t.Fatal("expected non-nil query after Distinct")
	}
	// Verify method chaining works
	q2 := q.Limit(10)
	if q2 == nil {
		t.Fatal("expected non-nil query after Limit")
	}
}

func TestQueryChaining(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Query().From("users").WhereField("age", ">", 18).OrderBy("name ASC").Limit(100).Offset(50)
	ast := q.ToAST()
	if ast.Table != "users" {
		t.Fatalf("expected table 'users', got %s", ast.Table)
	}
	if len(ast.Where) != 1 {
		t.Fatalf("expected 1 where clause, got %d", len(ast.Where))
	}
	if ast.Limit != 100 {
		t.Fatalf("expected limit 100, got %d", ast.Limit)
	}
	if ast.Offset != 50 {
		t.Fatalf("expected offset 50, got %d", ast.Offset)
	}
}

func TestWithMetrics(t *testing.T) {
	ctx := newContext()
	metricsCalled := false
	m := &stubMetrics{called: &metricsCalled}
	ctx = ctx.WithMetrics(m)
	if ctx == nil {
		t.Fatal("expected non-nil context from WithMetrics")
	}
	// Trigger SaveChanges which will call metrics
	ctx.SaveChanges(context.Background())
	if !metricsCalled {
		t.Fatal("expected metrics to be called")
	}
}

type stubMetrics struct {
	called *bool
}

func (m *stubMetrics) ObserveChangeTracking(duration time.Duration, entries int) {
	*m.called = true
}

func (m *stubMetrics) ObserveSQL(operation string, duration time.Duration) {
}

type Product struct {
	ID   int
	Name string
	SKU  string
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
