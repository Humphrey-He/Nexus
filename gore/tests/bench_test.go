package tests

import (
	"testing"

	"gore/api"
	"gore/dialect"
)

func BenchmarkQueryBuild(b *testing.B) {
	ctx := newContext()
	set := api.Set[User](ctx)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = set.Query().
			Where(func(ast *dialect.QueryAST) {
				_ = ast
			}).
			OrderBy("id DESC").
			Limit(10).
			Offset(5).
			ToAST()
	}
}
