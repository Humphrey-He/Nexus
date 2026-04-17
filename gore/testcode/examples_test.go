package testcode

import (
	"context"
	"testing"
	"time"

	"gore/api"
	"gore/dialect"
	"gore/dialect/mysql"
	"gore/dialect/postgres"
	"gore/internal/executor"
)

// ===============================================================================
// Entity Definitions
// ===============================================================================

type User struct {
	ID        int
	Name      string
	Email     string
	Age       int
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Order struct {
	ID        int
	UserID    int
	Product   string
	Amount    float64
	Status    string
	CreatedAt time.Time
}

type Product struct {
	ID          int
	Name        string
	Description string
	Price       float64
	Stock       int
	CategoryID  int
}

// ===============================================================================
// Stub Executor Implementation
// ===============================================================================

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

type stubMetrics struct{}

func (m *stubMetrics) ObserveChangeTracking(duration time.Duration, entries int) {}
func (m *stubMetrics) ObserveSQL(operation string, duration time.Duration)       {}

// ===============================================================================
// Example 1: Basic DbContext Setup
// ===============================================================================

// This example demonstrates how to set up a DbContext with different databases.
func DemoBasicDbContextSetup() {
	// PostgreSQL setup
	pgExecutor := &stubExecutor{}
	_ = api.NewContext(pgExecutor, nil, &postgres.Dialector{})

	// MySQL setup
	mysqlExecutor := &stubExecutor{}
	_ = api.NewContext(mysqlExecutor, nil, &mysql.Dialector{})
}

// ===============================================================================
// Example 2: DbSet Basic Operations
// ===============================================================================

// Demonstrate basic DbSet operations: Query, Add, Attach, Remove
func DemoDbSetBasicOperations() {
	ctx := api.NewContext(&stubExecutor{}, nil, &postgres.Dialector{})

	// Get DbSet for User entity
	users := api.Set[User](ctx)

	// Query without filters
	q := users.Query()
	_ = q

	// Attach existing entity for tracking
	user := &User{ID: 1, Name: "Alice", Email: "alice@example.com"}
	_ = users.Attach(user)

	// Mark entity as new (for insertion)
	newUser := &User{Name: "Bob", Email: "bob@example.com"}
	_ = users.Add(newUser)

	// Mark entity for deletion
	_ = users.Remove(user)
}

// ===============================================================================
// Example 3: Query Builder - Select
// ===============================================================================

// Demonstrate query building with various conditions
func DemoQueryBuilderSelect() {
	ctx := api.NewContext(&stubExecutor{}, nil, &postgres.Dialector{})

	// Basic query
	q := api.Set[User](ctx).Query().From("users")
	ast := q.ToAST()
	_ = ast

	// Query with column selection
	q = api.Set[User](ctx).Query().
		From("users").
		WhereField("name", "=", "Alice").
		WhereField("age", ">", 18)
	ast = q.ToAST()
	_ = ast

	// Query with IN clause
	q = api.Set[User](ctx).Query().
		From("users").
		WhereIn("id", 1, 2, 3, 4, 5)
	ast = q.ToAST()
	_ = ast

	// Query with LIKE
	q = api.Set[User](ctx).Query().
		From("users").
		WhereLike("name", "%Alice%")
	ast = q.ToAST()
	_ = ast

	// Query with ordering
	q = api.Set[User](ctx).Query().
		From("users").
		OrderBy("created_at DESC").
		OrderBy("name ASC")
	ast = q.ToAST()
	_ = ast

	// Query with pagination
	q = api.Set[User](ctx).Query().
		From("users").
		Limit(10).
		Offset(20)
	ast = q.ToAST()
	_ = ast

	// Full query
	q = api.Set[User](ctx).Query().
		From("users").
		WhereField("status", "=", "active").
		OrderBy("created_at DESC").
		Limit(20)
	ast = q.ToAST()
	_ = ast
}

// ===============================================================================
// Example 4: Query Builder - Complex Queries
// ===============================================================================

func DemoQueryBuilderComplex() {
	ctx := api.NewContext(&stubExecutor{}, nil, &postgres.Dialector{})

	// Query with custom predicate
	q := api.Set[User](ctx).Query().
		From("users").
		Where(func(ast *dialect.QueryAST) {
			ast.Where = append(ast.Where, "age > 18")
			ast.Where = append(ast.Where, "status = 'active'")
		})
	ast := q.ToAST()
	_ = ast

	// Group by query
	q = api.Set[User](ctx).Query().
		From("users").
		GroupBy("category", "status").
		OrderBy("category ASC")
	ast = q.ToAST()
	_ = ast

	// Distinct query
	q = api.Set[User](ctx).Query().
		From("users").
		Distinct().
		WhereField("name", "=", "Alice")
	ast = q.ToAST()
	_ = ast
}

// ===============================================================================
// Example 5: Change Tracking
// ===============================================================================

// Demonstrate entity change tracking with SaveChanges
func DemoChangeTracking() {
	ctx := api.NewContext(&stubExecutor{}, nil, &postgres.Dialector{})

	// Attach an entity to start tracking
	user := &User{ID: 1, Name: "Alice", Email: "alice@example.com"}
	users := api.Set[User](ctx)
	users.Attach(user)

	// Modify the entity
	user.Name = "Alice Modified"
	user.Age = 25

	// Detect and get changes
	changes, _ := ctx.SaveChanges(context.Background())
	_ = changes // Number of changed entities
}

// ===============================================================================
// Example 6: No Tracking Mode
// ===============================================================================

// Demonstrate AsNoTracking for read-only operations
func DemoNoTrackingMode() {
	ctx := api.NewContext(&stubExecutor{}, nil, &postgres.Dialector{})

	// Disable change tracking for read-only operations
	roCtx := ctx.AsNoTracking()

	// roCtx is a DbContext interface
	// Perform read operations - but we can't use Set with interface directly
	// This demonstrates the pattern
	_ = roCtx

	// Attempting to save changes will return error
	_, err := roCtx.SaveChanges(context.Background())
	if err == api.ErrTrackingDisabled {
		// Expected behavior
	}
}

// ===============================================================================
// Example 7: MySQL Dialect SQL Generation
// ===============================================================================

// Demonstrate MySQL-specific SQL generation
func DemoMySQLDialect() {
	d := &mysql.Dialector{}

	// Basic SELECT
	ast := &dialect.QueryAST{Table: "users"}
	sql, _, _ := d.BuildSelect(ast)
	_ = sql // SELECT * FROM `users`

	// SELECT with columns
	ast = &dialect.QueryAST{
		Table:   "users",
		Columns: []string{"id", "name", "email"},
	}
	sql, _, _ = d.BuildSelect(ast)
	_ = sql // SELECT `id`, `name`, `email` FROM `users`

	// SELECT with WHERE
	ast = &dialect.QueryAST{
		Table: "users",
		Where: []string{"name = ?"},
	}
	sql, _, _ = d.BuildSelect(ast)
	_ = sql // SELECT * FROM `users` WHERE name = ?

	// MySQL LIMIT syntax: LIMIT offset, count
	ast = &dialect.QueryAST{
		Table:  "users",
		Limit:  10,
		Offset: 20,
	}
	sql, _, _ = d.BuildSelect(ast)
	_ = sql // SELECT * FROM `users` LIMIT 20, 10

	// INSERT
	insertAst := &dialect.InsertAST{
		Table:   "users",
		Columns: []string{"name", "email", "age"},
	}
	sql, _, _ = d.BuildInsert(insertAst)
	_ = sql // INSERT INTO `users` (`name`, `email`, `age`) VALUES (?, ?, ?)

	// UPDATE
	updateAst := &dialect.UpdateAST{
		Table:   "users",
		Columns: []string{"name", "email"},
		Where:   []string{"id = ?"},
	}
	sql, _, _ = d.BuildUpdate(updateAst)
	_ = sql // UPDATE `users` SET `name` = ?, `email` = ? WHERE id = ?

	// DELETE
	deleteAst := &dialect.DeleteAST{
		Table: "users",
		Where: []string{"id = ?"},
	}
	sql, _, _ = d.BuildDelete(deleteAst)
	_ = sql // DELETE FROM `users` WHERE id = ?
}

// ===============================================================================
// Example 8: PostgreSQL Dialect SQL Generation
// ===============================================================================

// Demonstrate PostgreSQL-specific SQL generation
func DemoPostgresDialect() {
	d := &postgres.Dialector{}

	// Basic SELECT
	ast := &dialect.QueryAST{Table: "users"}
	sql, _, _ := d.BuildSelect(ast)
	_ = sql // SELECT * FROM users

	// PostgreSQL LIMIT syntax: LIMIT n OFFSET m
	ast = &dialect.QueryAST{
		Table:  "users",
		Limit:  10,
		Offset: 20,
	}
	sql, _, _ = d.BuildSelect(ast)
	_ = sql // SELECT * FROM users LIMIT 10 OFFSET 20

	// PostgreSQL uses $1, $2 for placeholders
	insertAst := &dialect.InsertAST{
		Table:   "users",
		Columns: []string{"name", "email"},
	}
	sql, _, _ = d.BuildInsert(insertAst)
	_ = sql // INSERT INTO users (name, email) VALUES ($1, $2)
}

// ===============================================================================
// Example 9: gore-lint Schema Definition
// ===============================================================================

// This example shows how to define a schema for gore-lint analysis
func DemoSchemaDefinition() {
	// This represents the JSON schema structure used by gore-lint
	schema := `
{
  "tables": [
    {
      "tableName": "users",
      "columns": [
        {"name": "id", "type": "int"},
        {"name": "name", "type": "varchar"},
        {"name": "email", "type": "varchar"},
        {"name": "status", "type": "tinyint"},
        {"name": "created_at", "type": "timestamp"}
      ],
      "indexes": [
        {"name": "PRIMARY", "columns": ["id"], "unique": true, "method": "btree"},
        {"name": "idx_name", "columns": ["name"], "unique": false, "method": "btree"},
        {"name": "idx_status", "columns": ["status"], "unique": false, "method": "btree", "isVisible": false},
        {"name": "idx_name_desc", "columns": ["name"], "unique": false, "method": "btree", "collation": "D"}
      ]
    }
  ]
}
`
	_ = schema
}

// ===============================================================================
// Example 10: API Usage Summary
// ===============================================================================

// Summary of all major API functions
func DemoAPISummary() {
	ctx := api.NewContext(&stubExecutor{}, nil, &postgres.Dialector{})

	// 1. DbSet operations
	users := api.Set[User](ctx)

	// Query building
	q := users.Query().
		From("users").
		WhereField("name", "=", "Alice").
		WhereField("age", ">", 18).
		OrderBy("created_at DESC").
		Limit(10).
		Offset(0)
	_ = q

	// Entity tracking
	users.Add(&User{Name: "Bob"})
	users.Attach(&User{ID: 1, Name: "Alice"})
	users.Remove(&User{ID: 1})

	// Save changes
	count, _ := ctx.SaveChanges(context.Background())
	_ = count

	// No tracking mode
	roCtx := ctx.AsNoTracking()
	_ = roCtx

	// With metrics
	ctx = ctx.WithMetrics(&stubMetrics{})
}

// ===============================================================================
// Test Functions
// ===============================================================================

func TestDemoBasicDbContextSetup(t *testing.T) {
	DemoBasicDbContextSetup()
}

func TestDemoDbSetBasicOperations(t *testing.T) {
	DemoDbSetBasicOperations()
}

func TestDemoQueryBuilderSelect(t *testing.T) {
	DemoQueryBuilderSelect()
}

func TestDemoQueryBuilderComplex(t *testing.T) {
	DemoQueryBuilderComplex()
}

func TestDemoChangeTracking(t *testing.T) {
	DemoChangeTracking()
}

func TestDemoNoTrackingMode(t *testing.T) {
	DemoNoTrackingMode()
}

func TestDemoMySQLDialect(t *testing.T) {
	DemoMySQLDialect()
}

func TestDemoPostgresDialect(t *testing.T) {
	DemoPostgresDialect()
}

func TestDemoSchemaDefinition(t *testing.T) {
	DemoSchemaDefinition()
}

func TestDemoAPISummary(t *testing.T) {
	DemoAPISummary()
}
