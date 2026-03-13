package tests

import (
	"sync"
	"testing"

	"gore/api"
	"gore/dialect"
)

func TestDbSetQuery(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Query().Limit(10).Offset(2).OrderBy("id DESC")
	if q == nil {
		t.Fatalf("expected Query, got nil")
	}
}

func TestDbSetQueryFromWhereField(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	ast := set.Query().From("users").WhereField("name", "=", "alice").ToAST()
	if ast.Table != "users" {
		t.Fatalf("expected table users, got %s", ast.Table)
	}
	if len(ast.Where) != 1 {
		t.Fatalf("expected 1 where clause, got %d", len(ast.Where))
	}
}

func TestDbSetAttachAddRemove(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)

	u1 := &User{ID: 1, Name: "Alice"}
	u2 := &User{ID: 2, Name: "Bob"}

	if err := set.Attach(u1); err != nil {
		t.Fatalf("attach failed: %v", err)
	}
	if err := set.Add(u2); err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if err := set.Remove(u1); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	entries := ctx.Tracker().Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 tracked entries, got %d", len(entries))
	}
}

func TestDbSetQueryConcurrent(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)

	const workers = 8
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				q := set.Query().Where(func(ast *dialect.QueryAST) { _ = ast }).Limit(5).Offset(1)
				if q == nil {
					t.Errorf("expected Query, got nil")
					return
				}
			}
		}()
	}

	wg.Wait()
}
