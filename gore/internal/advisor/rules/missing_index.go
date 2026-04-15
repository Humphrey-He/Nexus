package rules

import (
	"fmt"

	"gore/internal/advisor"
)

// MissingIndexRule detects fields that are frequently queried but lack index coverage.
type MissingIndexRule struct {
	minFrequency int
}

func NewMissingIndexRule() *MissingIndexRule { return &MissingIndexRule{minFrequency: 3} }

func (r *MissingIndexRule) ID() string { return "IDX-006" }

func (r *MissingIndexRule) Name() string { return "Missing Index" }

func (r *MissingIndexRule) Description() string {
	return "高频查询字段缺少索引可能导致全表扫描"
}

func (r *MissingIndexRule) Severity() advisor.Severity { return advisor.SeverityHigh }

func (r *MissingIndexRule) WhyDoc() string { return "https://docs.gore.io/rules/IDX-006" }

// Check evaluates a single query. For aggregate analysis, use CheckAll.
func (r *MissingIndexRule) Check(query *advisor.QueryMetadata, schema *advisor.TableSchema) []advisor.Suggestion {
	// Single-query check is not meaningful for IDX-006
	// This rule requires aggregate frequency analysis
	return nil
}

// CheckAll analyzes all queries collectively to find frequently accessed fields missing indexes.
func (r *MissingIndexRule) CheckAll(queries []*advisor.QueryMetadata, schema *advisor.TableSchema) []advisor.Suggestion {
	if schema == nil || len(queries) == 0 {
		return nil
	}

	// Aggregate field frequency across all queries for this table
	freq := make(map[string]int)
	for _, q := range queries {
		if q.TableName != schema.TableName {
			continue
		}
		for _, cond := range q.Conditions {
			// Skip negated conditions and functions (can't use index anyway)
			if cond.IsNegated || cond.IsFunction {
				continue
			}
			freq[cond.Field]++
		}
	}

	// Build a set of indexed fields
	indexedFields := make(map[string]bool)
	for _, idx := range schema.Indexes {
		for _, col := range idx.Columns {
			indexedFields[col] = true
		}
	}

	var out []advisor.Suggestion
	for field, count := range freq {
		if count < r.minFrequency {
			continue
		}
		if indexedFields[field] {
			continue
		}

		// Field is frequently queried but not indexed
		out = append(out, advisor.Suggestion{
			RuleID:         r.ID(),
			Severity:       r.Severity(),
			Message:        fmt.Sprintf("字段 %s 在 %d 个查询中出现但缺少索引", field, count),
			Reason:         "高频查询字段无索引会导致全表扫描",
			Recommendation: fmt.Sprintf("考虑为字段 %s 建立索引", field),
			Confidence:     0.7,
			Tags:           []string{"index", "missing"},
		})
	}

	return out
}
