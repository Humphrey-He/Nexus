package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractQueriesResolvesLiteralsAndWrappers(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")

	code := `package sample

const table = "users"
const field = "na" + "me"
var op = "="
var limit = 5

func example() {
	_ = Set[User](ctx).Query().From(table).WhereField(field, op, string("alice")).Limit(int(limit)).Offset(2)
}

type User struct{}
var ctx *Context
`

	if err := os.WriteFile(file, []byte(code), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	queries, err := extractQueries(dir)
	if err != nil {
		t.Fatalf("extract queries: %v", err)
	}
	if len(queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(queries))
	}

	q := queries[0]
	if q.TableName != "users" {
		t.Fatalf("expected table users, got %s", q.TableName)
	}
	if len(q.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(q.Conditions))
	}
	if q.Conditions[0].Field != "name" {
		t.Fatalf("expected field name, got %s", q.Conditions[0].Field)
	}
	if q.Conditions[0].Operator != "=" {
		t.Fatalf("expected operator =, got %s", q.Conditions[0].Operator)
	}
	if q.Limit == nil || *q.Limit != 5 {
		t.Fatalf("expected limit 5, got %v", q.Limit)
	}
}
