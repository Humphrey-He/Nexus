package rules

import (
	"fmt"
	"strings"

	"gore/internal/advisor"
)

// OrderByIndexRule checks order by fields for index coverage and direction consistency.
type OrderByIndexRule struct{}

func NewOrderByIndexRule() *OrderByIndexRule { return &OrderByIndexRule{} }

func (r *OrderByIndexRule) ID() string { return "IDX-008" }

func (r *OrderByIndexRule) Name() string { return "Order By Index Coverage" }

func (r *OrderByIndexRule) Description() string {
	return "ORDER BY 字段缺少索引或方向不一致时可能导致排序开销"
}

func (r *OrderByIndexRule) Severity() advisor.Severity { return advisor.SeverityInfo }

func (r *OrderByIndexRule) WhyDoc() string { return "https://docs.gore.io/rules/IDX-008" }

func (r *OrderByIndexRule) Check(query *advisor.QueryMetadata, schema *advisor.TableSchema) []advisor.Suggestion {
	if query == nil || schema == nil {
		return nil
	}

	var out []advisor.Suggestion

	// Build a set of WHERE condition fields to determine if we can use index for sorting
	whereFields := make(map[string]bool)
	for _, cond := range query.Conditions {
		whereFields[cond.Field] = true
	}

	for _, order := range query.OrderBy {
		idxInfo, idxPos := findIndexCoveringField(schema.Indexes, order.Field)

		if idxInfo == nil {
			// No index covers this field
			out = append(out, advisor.Suggestion{
				RuleID:         r.ID(),
				Severity:       r.Severity(),
				Message:        fmt.Sprintf("ORDER BY 字段 %s 缺少索引", order.Field),
				Reason:         "无索引排序可能导致内存或磁盘排序",
				Recommendation: "考虑为排序字段建立索引",
				Confidence:     0.6,
				Tags:           []string{"index", "orderby"},
				SourceFile:     query.SourceFile,
				LineNumber:     query.LineNumber,
			})
			continue
		}

		// Check if the index direction matches ORDER BY direction
		// For PostgreSQL B-tree: ASC index can serve ORDER BY ASC efficiently
		// But ORDER BY DESC can also use ASC index (just扫描方向相反)
		// The real issue is: can this index be used at all for this ORDER BY?
		//
		// If the field is the first column in the index AND all preceding index columns
		// appear in WHERE conditions, then the index can be used for sorting
		if idxPos > 0 {
			// Field is not the first column - check if all preceding columns are in WHERE
			canUseIndex := true
			for i := 0; i < idxPos; i++ {
				if !whereFields[idxInfo.Columns[i]] {
					canUseIndex = false
					break
				}
			}
			if !canUseIndex {
				out = append(out, advisor.Suggestion{
					RuleID:         r.ID(),
					Severity:       r.Severity(),
					Message:        fmt.Sprintf("ORDER BY 字段 %s 无法利用索引 %s (跳过了前面的索引列)", order.Field, idxInfo.Name),
					Reason:         "ORDER BY 字段不是索引首列，且前面的索引列未出现在 WHERE 中",
					Recommendation: "调整 WHERE 条件包含所有前面的索引列，或调整索引顺序",
					Confidence:     0.7,
					Tags:           []string{"index", "orderby", "index-coverage"},
					SourceFile:     query.SourceFile,
					LineNumber:     query.LineNumber,
				})
			}
		}

		// Check index direction consistency
		// For B-tree indexes, ORDER BY ASC with ASC index is optimal
		// ORDER BY DESC can still use ASC index but may be slightly less efficient
		// unless the index is defined with the same direction
		if !r.checkIndexDirectionMatch(idxInfo, idxPos, order.Direction) {
			out = append(out, advisor.Suggestion{
				RuleID:         r.ID(),
				Severity:       r.Severity(),
				Message:        fmt.Sprintf("ORDER BY %s %s 与索引 %s 定义方向可能不一致", order.Field, order.Direction, idxInfo.Name),
				Reason:         "索引方向与排序方向不一致可能导致额外的排序开销",
				Recommendation: "考虑创建与常见排序方向一致的索引",
				Confidence:     0.5,
				Tags:           []string{"index", "orderby", "direction"},
				SourceFile:     query.SourceFile,
				LineNumber:     query.LineNumber,
			})
		}
	}

	return out
}

// checkIndexDirectionMatch checks if the ORDER BY direction is compatible with the index.
// PostgreSQL B-tree indexes store data in ASC order by default.
// For a field at position idxPos in a composite index:
// - If idxPos > 0, the index can only be used if all preceding columns are in WHERE
// - The direction matching is informational only (PostgreSQL can scan index backwards)
func (r *OrderByIndexRule) checkIndexDirectionMatch(idx *advisor.IndexInfo, fieldPos int, orderDir string) bool {
	// For now, we only warn about direction mismatch for single-column indexes
	// Composite indexes have more complex direction semantics
	if len(idx.Columns) == 1 {
		// Check if the index was defined with a specific direction
		// Note: PostgreSQL doesn't support index direction per column in standard syntax
		// This is more of an informational check
		orderUpper := strings.ToUpper(orderDir)
		if orderUpper == "DESC" {
			// PostgreSQL can scan ASC index backwards for DESC, but it's not optimal
			return false
		}
	}
	return true
}

// findIndexCoveringField finds an index that covers the given field.
// Returns the index info and the position of the field in the index.
func findIndexCoveringField(indexes []advisor.IndexInfo, field string) (*advisor.IndexInfo, int) {
	for i := range indexes {
		for j, col := range indexes[i].Columns {
			if strings.EqualFold(col, field) {
				return &indexes[i], j
			}
		}
	}
	return nil, -1
}
