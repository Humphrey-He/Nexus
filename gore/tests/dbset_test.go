package tests

import (
	"testing"

	"gore/api"
)

func TestDbSetQuery(t *testing.T) {
	ctx := newContext()
	set := api.Set[User](ctx)
	q := set.Query().Limit(10).Offset(2).OrderBy("id DESC")
	if q == nil {
		t.Fatalf("expected Query, got nil")
	}
}
