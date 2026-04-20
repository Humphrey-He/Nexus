package main

import (
	"context"
	"fmt"

	"gore/api"
	"gore/dialect/postgres"
	"gore/internal/executor"
	"gore/internal/metadata"
)

type User struct {
	ID    int
	Name  string
	Email string
}

func main() {
	exec := &executor.StubExecutor{}
	meta := metadata.NewRegistry()
	dialector := &postgres.Dialector{}

	ctx := api.NewContext(exec, meta, dialector)
	users := api.Set[User](ctx)

	query := users.Query().
		From("users").
		WhereField("name", "=", "Alice").
		WhereIn("status", 1, 2, 3).
		OrderBy("created_at DESC").
		Limit(10)

	ast := query.ToAST()
	fmt.Printf("Query AST: %+v\n", ast)

	user := &User{Name: "Bob", Email: "bob@example.com"}
	if err := users.Add(user); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	if _, err := ctx.SaveChanges(context.Background()); err != nil {
		fmt.Printf("SaveChanges error: %v\n", err)
	}
}
