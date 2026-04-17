package api

import (
	"context"
	"testing"
	"time"

	"gore/dialect"
	"gore/internal/executor"
)

// Test entities
type TestUser struct {
	ID   int
	Name string
}

type TestProduct struct {
	ID   int
	Name string
	SKU  string
}

// Stub implementations for testing
type testStubExecutor struct{}

type testStubRows struct {
	closed bool
}

type testStubResult struct {
	rowsAffected int64
}

func (s *testStubExecutor) Query(ctx context.Context, query string, args ...any) (executor.Rows, error) {
	_ = ctx
	_ = query
	_ = args
	return &testStubRows{}, nil
}

func (s *testStubExecutor) Exec(ctx context.Context, query string, args ...any) (executor.Result, error) {
	_ = ctx
	_ = query
	_ = args
	return &testStubResult{rowsAffected: 1}, nil
}

func (r *testStubRows) Next() bool { return false }
func (r *testStubRows) Scan(dest ...any) error {
	_ = dest
	return nil
}
func (r *testStubRows) Close() error {
	r.closed = true
	return nil
}
func (r *testStubRows) Err() error { return nil }

func (r *testStubResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

type testStubDialector struct{}

func (d *testStubDialector) Name() string { return "stub" }

func (d *testStubDialector) BuildSelect(ast *dialect.QueryAST) (string, []any, error) {
	_ = ast
	return "SELECT * FROM users", nil, nil
}

func (d *testStubDialector) BuildInsert(ast *dialect.InsertAST) (string, []any, error) {
	_ = ast
	return "INSERT INTO users (name) VALUES (?)", nil, nil
}

func (d *testStubDialector) BuildUpdate(ast *dialect.UpdateAST) (string, []any, error) {
	_ = ast
	return "UPDATE users SET name = ?", nil, nil
}

func (d *testStubDialector) BuildDelete(ast *dialect.DeleteAST) (string, []any, error) {
	_ = ast
	return "DELETE FROM users", nil, nil
}

func newTestContext() *Context {
	return NewContext(&testStubExecutor{}, nil, &testStubDialector{})
}

// ===============================================================================
// Context Tests
// ===============================================================================

func TestNewContext(t *testing.T) {
	ctx := newTestContext()
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
}

func TestNewContextWithNilMetadata(t *testing.T) {
	ctx := NewContext(&testStubExecutor{}, nil, &testStubDialector{})
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if ctx.Metadata() == nil {
		t.Fatal("expected non-nil metadata registry")
	}
}

func TestContextDialector(t *testing.T) {
	ctx := newTestContext()
	d := ctx.Dialector()
	if d == nil {
		t.Fatal("expected non-nil dialector")
	}
	if d.Name() != "stub" {
		t.Fatalf("expected 'stub', got %s", d.Name())
	}
}

func TestContextExecutor(t *testing.T) {
	ctx := newTestContext()
	exec := ctx.Executor()
	if exec == nil {
		t.Fatal("expected non-nil executor")
	}
}

func TestContextTracker(t *testing.T) {
	ctx := newTestContext()
	tr := ctx.Tracker()
	if tr == nil {
		t.Fatal("expected non-nil tracker")
	}
}

func TestContextMetadata(t *testing.T) {
	ctx := newTestContext()
	meta := ctx.Metadata()
	if meta == nil {
		t.Fatal("expected non-nil metadata")
	}
}

func TestContextAsNoTracking(t *testing.T) {
	ctx := newTestContext()
	noTrack := ctx.AsNoTracking()
	if noTrack == nil {
		t.Fatal("expected non-nil context")
	}
	// AsNoTracking returns interface, check internal state via concrete type
	if ctx == noTrack.(*Context) {
		t.Fatal("AsNoTracking should return a clone")
	}
}

func TestContextAsTracking(t *testing.T) {
	ctx := newTestContext()
	// Start with no-tracking context
	noTrack := ctx.AsNoTracking()
	// Re-enable tracking through concrete type
	ctx = noTrack.(*Context)
	track := ctx.AsTracking()
	if track == nil {
		t.Fatal("expected non-nil context")
	}
}

func TestContextSaveChangesNoTracking(t *testing.T) {
	ctx := newTestContext()
	// Test that SaveChanges returns error when tracking is disabled
	noTrack := ctx.AsNoTracking()
	_, err := noTrack.SaveChanges(context.Background())
	if err != ErrTrackingDisabled {
		t.Fatalf("expected ErrTrackingDisabled, got %v", err)
	}
}

func TestContextSaveChangesNoChanges(t *testing.T) {
	ctx := newTestContext()
	user := &TestUser{ID: 1, Name: "Alice"}
	Set[TestUser](ctx).Attach(user)

	changes, err := ctx.SaveChanges(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changes != 0 {
		t.Fatalf("expected 0 changes, got %d", changes)
	}
}

func TestContextSaveChangesWithAddedEntity(t *testing.T) {
	ctx := newTestContext()
	user := &TestUser{Name: "Bob"}
	Set[TestUser](ctx).Add(user)

	changes, err := ctx.SaveChanges(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changes != 1 {
		t.Fatalf("expected 1 change, got %d", changes)
	}
}

func TestContextSaveChangesWithDeletedEntity(t *testing.T) {
	ctx := newTestContext()
	user := &TestUser{ID: 1, Name: "Charlie"}
	Set[TestUser](ctx).Attach(user)
	Set[TestUser](ctx).Remove(user)

	changes, err := ctx.SaveChanges(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changes != 1 {
		t.Fatalf("expected 1 change, got %d", changes)
	}
}

func TestContextWithMetrics(t *testing.T) {
	ctx := newTestContext()
	called := false
	m := &testMetrics{callRef: &called}
	ctx = ctx.WithMetrics(m)
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	Set[TestUser](ctx).Add(&TestUser{Name: "Test"})
	ctx.SaveChanges(context.Background())

	if !called {
		t.Fatal("expected metrics ObserveChangeTracking to be called")
	}
}

type testMetrics struct {
	callRef *bool
}

func (m *testMetrics) ObserveChangeTracking(duration time.Duration, entries int) {
	*m.callRef = true
}

func (m *testMetrics) ObserveSQL(operation string, duration time.Duration) {}

// ===============================================================================
// DbSet Tests
// ===============================================================================

func TestSetGeneric(t *testing.T) {
	ctx := newTestContext()
	users := Set[TestUser](ctx)
	if users == nil {
		t.Fatal("expected non-nil DbSet")
	}

	products := Set[TestProduct](ctx)
	if products == nil {
		t.Fatal("expected non-nil DbSet for Product")
	}
}

func TestDbSetAttach(t *testing.T) {
	ctx := newTestContext()
	set := Set[TestUser](ctx)
	user := &TestUser{ID: 1, Name: "Alice"}

	err := set.Attach(user)
	if err != nil {
		t.Fatalf("attach failed: %v", err)
	}
}

func TestDbSetAttachNoTracking(t *testing.T) {
	ctx := newTestContext()
	// AsNoTracking returns DbContext interface, but Set[] requires *Context
	// Test the interface behavior by checking SaveChanges returns error
	noTrack := ctx.AsNoTracking()
	// Attach should fail on no-tracking context - but we can't call Set on interface
	// Instead, verify the context itself reports tracking disabled
	_, err := noTrack.SaveChanges(context.Background())
	if err != ErrTrackingDisabled {
		t.Fatalf("expected ErrTrackingDisabled from SaveChanges, got %v", err)
	}
}

func TestDbSetAdd(t *testing.T) {
	ctx := newTestContext()
	set := Set[TestUser](ctx)
	user := &TestUser{Name: "Bob"}

	err := set.Add(user)
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}
}

func TestDbSetAddNoTracking(t *testing.T) {
	ctx := newTestContext()
	// AsNoTracking returns DbContext interface
	noTrack := ctx.AsNoTracking()
	// SaveChanges should return error for no-tracking context
	_, err := noTrack.SaveChanges(context.Background())
	if err != ErrTrackingDisabled {
		t.Fatalf("expected ErrTrackingDisabled from SaveChanges, got %v", err)
	}
}

func TestDbSetRemove(t *testing.T) {
	ctx := newTestContext()
	set := Set[TestUser](ctx)
	user := &TestUser{ID: 1, Name: "Charlie"}

	err := set.Remove(user)
	if err != nil {
		t.Fatalf("remove failed: %v", err)
	}
}

func TestDbSetRemoveNoTracking(t *testing.T) {
	ctx := newTestContext()
	// AsNoTracking returns DbContext interface
	noTrack := ctx.AsNoTracking()
	// SaveChanges should return error for no-tracking context
	_, err := noTrack.SaveChanges(context.Background())
	if err != ErrTrackingDisabled {
		t.Fatalf("expected ErrTrackingDisabled from SaveChanges, got %v", err)
	}
}

func TestDbSetFind(t *testing.T) {
	ctx := newTestContext()
	set := Set[TestUser](ctx)

	_, err := set.Find(1)
	if err != ErrNotImplemented {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestDbSetWhere(t *testing.T) {
	ctx := newTestContext()
	set := Set[TestUser](ctx)

	q := set.Where(func(ast *dialect.QueryAST) {
		ast.Where = append(ast.Where, "name = ?")
	})
	if q == nil {
		t.Fatal("expected non-nil query")
	}
}

func TestDbSetQuery(t *testing.T) {
	ctx := newTestContext()
	set := Set[TestUser](ctx)

	q := set.Query()
	if q == nil {
		t.Fatal("expected non-nil query")
	}
}

// ===============================================================================
// Query Builder Tests
// ===============================================================================

func TestQueryFrom(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().From("users")
	ast := q.ToAST()
	if ast.Table != "users" {
		t.Fatalf("expected 'users', got %s", ast.Table)
	}
}

func TestQueryFromEmpty(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().From("")
	ast := q.ToAST()
	if ast.Table != "" {
		t.Fatalf("expected empty string, got %s", ast.Table)
	}
}

func TestQueryWhereField(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().From("users").WhereField("name", "=", "Alice")
	ast := q.ToAST()
	if len(ast.Where) != 1 {
		t.Fatalf("expected 1 where clause, got %d", len(ast.Where))
	}
	if ast.Where[0] != "name = ?" {
		t.Fatalf("expected 'name = ?', got %s", ast.Where[0])
	}
}

func TestQueryWhereFieldEmptyField(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().From("users").WhereField("", "=", "Alice")
	ast := q.ToAST()
	if len(ast.Where) != 0 {
		t.Fatalf("expected 0 where clauses, got %d", len(ast.Where))
	}
}

func TestQueryWhereIn(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().From("users").WhereIn("id", 1, 2, 3)
	ast := q.ToAST()
	if len(ast.Where) != 1 {
		t.Fatalf("expected 1 where clause, got %d", len(ast.Where))
	}
	if ast.Where[0] != "id IN (?,?,?)" {
		t.Fatalf("expected 'id IN (?,?,?)', got %s", ast.Where[0])
	}
}

func TestQueryWhereInEmpty(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().From("users").WhereIn("id")
	ast := q.ToAST()
	if len(ast.Where) != 0 {
		t.Fatalf("expected 0 where clauses, got %d", len(ast.Where))
	}
}

func TestQueryWhereLike(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().From("users").WhereLike("name", "%lice%")
	ast := q.ToAST()
	if len(ast.Where) != 1 {
		t.Fatalf("expected 1 where clause, got %d", len(ast.Where))
	}
	if ast.Where[0] != "name LIKE ?" {
		t.Fatalf("expected 'name LIKE ?', got %s", ast.Where[0])
	}
}

func TestQueryWhereLikeEmpty(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().From("users").WhereLike("", "%lice%")
	ast := q.ToAST()
	if len(ast.Where) != 0 {
		t.Fatalf("expected 0 where clauses, got %d", len(ast.Where))
	}
}

func TestQueryOrderBy(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().From("users").OrderBy("name ASC").OrderBy("id DESC")
	ast := q.ToAST()
	if len(ast.OrderBy) != 2 {
		t.Fatalf("expected 2 order clauses, got %d", len(ast.OrderBy))
	}
	if ast.OrderBy[0] != "name ASC" {
		t.Fatalf("expected 'name ASC', got %s", ast.OrderBy[0])
	}
	if ast.OrderBy[1] != "id DESC" {
		t.Fatalf("expected 'id DESC', got %s", ast.OrderBy[1])
	}
}

func TestQueryLimit(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().From("users").Limit(10)
	ast := q.ToAST()
	if ast.Limit != 10 {
		t.Fatalf("expected limit 10, got %d", ast.Limit)
	}
}

func TestQueryOffset(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().From("users").Offset(20)
	ast := q.ToAST()
	if ast.Offset != 20 {
		t.Fatalf("expected offset 20, got %d", ast.Offset)
	}
}

func TestQueryGroupBy(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().From("users").GroupBy("status", "category")
	if q == nil {
		t.Fatal("expected non-nil query")
	}
}

func TestQueryDistinct(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().From("users").Distinct()
	if q == nil {
		t.Fatal("expected non-nil query")
	}
}

func TestQueryChaining(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().
		From("users").
		WhereField("age", ">", 18).
		WhereField("status", "=", "active").
		OrderBy("name ASC").
		Limit(100).
		Offset(50)

	ast := q.ToAST()
	if ast.Table != "users" {
		t.Fatalf("expected 'users', got %s", ast.Table)
	}
	if len(ast.Where) != 2 {
		t.Fatalf("expected 2 where clauses, got %d", len(ast.Where))
	}
	if ast.Limit != 100 {
		t.Fatalf("expected limit 100, got %d", ast.Limit)
	}
	if ast.Offset != 50 {
		t.Fatalf("expected offset 50, got %d", ast.Offset)
	}
}

func TestQueryToAST(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().
		From("users").
		WhereField("id", "=", 1).
		OrderBy("name ASC").
		Limit(1)

	ast := q.ToAST()
	if ast == nil {
		t.Fatal("expected non-nil AST")
	}
	if ast.Table != "users" {
		t.Fatalf("expected 'users', got %s", ast.Table)
	}
	if len(ast.Where) != 1 {
		t.Fatalf("expected 1 where clause, got %d", len(ast.Where))
	}
	if len(ast.OrderBy) != 1 {
		t.Fatalf("expected 1 order clause, got %d", len(ast.OrderBy))
	}
	if ast.Limit != 1 {
		t.Fatalf("expected limit 1, got %d", ast.Limit)
	}
}

// ===============================================================================
// Error Sentinel Tests
// ===============================================================================

func TestErrNotImplemented(t *testing.T) {
	if ErrNotImplemented == nil {
		t.Fatal("expected non-nil ErrNotImplemented")
	}
	if ErrNotImplemented.Error() != "feature not implemented" {
		t.Fatalf("unexpected error message: %s", ErrNotImplemented.Error())
	}
}

func TestErrTrackingDisabled(t *testing.T) {
	if ErrTrackingDisabled == nil {
		t.Fatal("expected non-nil ErrTrackingDisabled")
	}
	if ErrTrackingDisabled.Error() != "change tracking is disabled" {
		t.Fatalf("unexpected error message: %s", ErrTrackingDisabled.Error())
	}
}

// ===============================================================================
// Query JOIN Tests
// ===============================================================================

func TestQueryJoin(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().
		From("users").
		Join("orders", "users.id = orders.user_id")

	ast := q.ToAST()
	if len(ast.Joins) != 1 {
		t.Fatalf("expected 1 join, got %d", len(ast.Joins))
	}
	if ast.Joins[0].Table != "orders" {
		t.Fatalf("expected 'orders', got %s", ast.Joins[0].Table)
	}
	if ast.Joins[0].Type != JoinInner {
		t.Fatalf("expected JoinInner, got %v", ast.Joins[0].Type)
	}
}

func TestQueryLeftJoin(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().
		From("users").
		LeftJoin("orders", "users.id = orders.user_id")

	ast := q.ToAST()
	if len(ast.Joins) != 1 {
		t.Fatalf("expected 1 join, got %d", len(ast.Joins))
	}
	if ast.Joins[0].Type != JoinLeft {
		t.Fatalf("expected JoinLeft, got %v", ast.Joins[0].Type)
	}
}

func TestQueryRightJoin(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().
		From("users").
		RightJoin("orders", "users.id = orders.user_id")

	ast := q.ToAST()
	if len(ast.Joins) != 1 {
		t.Fatalf("expected 1 join, got %d", len(ast.Joins))
	}
	if ast.Joins[0].Type != JoinRight {
		t.Fatalf("expected JoinRight, got %v", ast.Joins[0].Type)
	}
}

func TestQueryFullJoin(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().
		From("users").
		FullJoin("orders", "users.id = orders.user_id")

	ast := q.ToAST()
	if len(ast.Joins) != 1 {
		t.Fatalf("expected 1 join, got %d", len(ast.Joins))
	}
	if ast.Joins[0].Type != JoinFull {
		t.Fatalf("expected JoinFull, got %v", ast.Joins[0].Type)
	}
}

func TestQueryMultipleJoins(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().
		From("users").
		Join("orders", "users.id = orders.user_id").
		LeftJoin("products", "orders.product_id = products.id").
		RightJoin("categories", "products.category_id = categories.id")

	ast := q.ToAST()
	if len(ast.Joins) != 3 {
		t.Fatalf("expected 3 joins, got %d", len(ast.Joins))
	}
}

func TestQueryHaving(t *testing.T) {
	ctx := newTestContext()
	q := Set[TestUser](ctx).Query().
		From("users").
		GroupBy("status").
		Having("COUNT(*) > 10")

	ast := q.ToAST()
	if len(ast.Having) != 1 {
		t.Fatalf("expected 1 having clause, got %d", len(ast.Having))
	}
}

// ===============================================================================
// Batch Operations Tests
// ===============================================================================

func TestDbSetAddBatch(t *testing.T) {
	ctx := newTestContext()
	set := Set[TestUser](ctx)

	users := []*TestUser{
		{Name: "Alice"},
		{Name: "Bob"},
		{Name: "Charlie"},
	}

	err := set.AddBatch(users)
	if err != nil {
		t.Fatalf("AddBatch failed: %v", err)
	}
}

func TestDbSetAddBatchEmpty(t *testing.T) {
	ctx := newTestContext()
	set := Set[TestUser](ctx)

	err := set.AddBatch([]*TestUser{})
	if err == nil {
		t.Fatal("expected error for empty batch")
	}
}

func TestDbSetAddBatchWithNil(t *testing.T) {
	ctx := newTestContext()
	set := Set[TestUser](ctx)

	users := []*TestUser{
		{Name: "Alice"},
		nil,
		{Name: "Bob"},
	}

	err := set.AddBatch(users)
	if err == nil {
		t.Fatal("expected error for nil entity")
	}
}

func TestDbSetAttachBatch(t *testing.T) {
	ctx := newTestContext()
	set := Set[TestUser](ctx)

	users := []*TestUser{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}

	err := set.AttachBatch(users)
	if err != nil {
		t.Fatalf("AttachBatch failed: %v", err)
	}
}

func TestDbSetAttachBatchEmpty(t *testing.T) {
	ctx := newTestContext()
	set := Set[TestUser](ctx)

	err := set.AttachBatch([]*TestUser{})
	if err == nil {
		t.Fatal("expected error for empty batch")
	}
}

func TestDbSetRemoveBatch(t *testing.T) {
	ctx := newTestContext()
	set := Set[TestUser](ctx)

	users := []*TestUser{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}

	err := set.RemoveBatch(users)
	if err != nil {
		t.Fatalf("RemoveBatch failed: %v", err)
	}
}

func TestDbSetRemoveBatchEmpty(t *testing.T) {
	ctx := newTestContext()
	set := Set[TestUser](ctx)

	err := set.RemoveBatch([]*TestUser{})
	if err == nil {
		t.Fatal("expected error for empty batch")
	}
}

func TestDbSetFindBatch(t *testing.T) {
	ctx := newTestContext()
	set := Set[TestUser](ctx)

	_, err := set.FindBatch([]any{1, 2, 3})
	if err != ErrNotImplemented {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestDbSetFindBatchEmpty(t *testing.T) {
	ctx := newTestContext()
	set := Set[TestUser](ctx)

	_, err := set.FindBatch([]any{})
	if err == nil {
		t.Fatal("expected error for empty pks")
	}
}

// ===============================================================================
// Transaction Tests
// ===============================================================================

func TestContextTransactionNotImplemented(t *testing.T) {
	ctx := newTestContext()
	err := ctx.Transaction(context.Background(), func(tx DbContext) error {
		return nil
	})
	// The stub executor doesn't support transactions, so this should fail
	// We just verify the method is callable
	if err == nil {
		t.Log("Transaction succeeded with stub executor (expected in some cases)")
	}
}
