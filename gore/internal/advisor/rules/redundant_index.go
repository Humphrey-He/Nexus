package rules

import (
	"fmt"
	"strings"

	"gore/internal/advisor"
)

// RedundantIndexRule detects redundant indexes where one index is covered by another.
type RedundantIndexRule struct{}

func NewRedundantIndexRule() *RedundantIndexRule { return &RedundantIndexRule{} }

func (r *RedundantIndexRule) ID() string { return "IDX-007" }

func (r *RedundantIndexRule) Name() string { return "Redundant Index" }

func (r *RedundantIndexRule) Description() string {
	return "冗余索引浪费存储空间并增加写入开销"
}

func (r *RedundantIndexRule) Severity() advisor.Severity { return advisor.SeverityInfo }

func (r *RedundantIndexRule) WhyDoc() string { return "https://docs.gore.io/rules/IDX-007" }

// Check evaluates if one index makes another redundant.
func (r *RedundantIndexRule) Check(query *advisor.QueryMetadata, schema *advisor.TableSchema) []advisor.Suggestion {
	if schema == nil || len(schema.Indexes) < 2 {
		return nil
	}

	var out []advisor.Suggestion

	for i, idxA := range schema.Indexes {
		for j, idxB := range schema.Indexes {
			if i >= j {
				continue
			}

			// Check if idxA is redundant to idxB
			if r.isRedundant(idxA, idxB) {
				out = append(out, advisor.Suggestion{
					RuleID:         r.ID(),
					Severity:       r.Severity(),
					Message:        fmt.Sprintf("索引 %s 冗余于索引 %s", idxA.Name, idxB.Name),
					Reason:         "索引列是另一索引列的前缀，且无唯一约束",
					Recommendation: fmt.Sprintf("考虑删除索引 %s 以节省存储并提升写入性能", idxA.Name),
					Confidence:     0.85,
					Tags:           []string{"index", "redundant"},
				})
			}
		}
	}

	return out
}

// isRedundant returns true if index A is redundant to index B.
// A is redundant to B if: A's columns are a prefix of B's columns,
// and A is not unique while B is also not unique.
// (If A is unique but B is not, A is NOT redundant because it provides uniqueness guarantee)
func (r *RedundantIndexRule) isRedundant(idxA, idxB advisor.IndexInfo) bool {
	// If A has more columns than B, it can't be redundant to B
	if len(idxA.Columns) > len(idxB.Columns) {
		return false
	}

	// If A is unique but B is not, A is not redundant (it provides extra constraint)
	if idxA.Unique && !idxB.Unique {
		return false
	}

	// Check if A's columns are a prefix of B's columns
	for i, col := range idxA.Columns {
		if i >= len(idxB.Columns) || !strings.EqualFold(col, idxB.Columns[i]) {
			return false
		}
	}

	// A's columns are a prefix of B's columns
	return true
}
