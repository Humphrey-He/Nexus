package rules

import (
	"fmt"
	"strings"

	"gore/internal/advisor"
)

// DescendingIndexRule detects mismatches between ORDER BY direction and index collation.
type DescendingIndexRule struct{}

func NewDescendingIndexRule() *DescendingIndexRule {
	return &DescendingIndexRule{}
}

func (r *DescendingIndexRule) ID() string { return "IDX-MYSQL-002" }

func (r *DescendingIndexRule) Name() string { return "Descending Index Validation" }

func (r *DescendingIndexRule) Description() string {
	return "检测 MySQL 8.0+ 降序索引与 ORDER BY 方向的匹配"
}

func (r *DescendingIndexRule) Severity() advisor.Severity { return advisor.SeverityWarn }

func (r *DescendingIndexRule) WhyDoc() string { return "" }

func (r *DescendingIndexRule) Check(query *advisor.QueryMetadata, schema *advisor.TableSchema) []advisor.Suggestion {
	if query == nil || schema == nil {
		return nil
	}

	var suggestions []advisor.Suggestion

	// Build a map of index columns with their collation
	indexCollations := make(map[string]map[string]string) // indexName -> column -> collation (A/D)

	for _, idx := range schema.Indexes {
		collation := "A" // default ascending
		if meta, ok := idx.Metadata["collation"]; ok {
			if c, ok := meta.(string); ok {
				collation = c
			}
		}
		indexCollations[idx.Name] = make(map[string]string)
		for _, col := range idx.Columns {
			indexCollations[idx.Name][col] = collation
		}
	}

	// Check each ORDER BY clause
	for _, orderBy := range query.OrderBy {
		direction := strings.ToUpper(orderBy.Direction)
		if direction == "" {
			direction = "ASC"
		}

		// Find which index this ORDER BY could use
		for idxName, cols := range indexCollations {
			if collation, ok := cols[orderBy.Field]; ok {
				// Check if direction matches
				if direction == "DESC" && collation != "D" {
					suggestions = append(suggestions, advisor.Suggestion{
						RuleID:         r.ID(),
						Severity:       r.Severity(),
						Message:        fmt.Sprintf("ORDER BY %s %s 与索引 %s 方向不匹配", orderBy.Field, direction, idxName),
						Reason:         "MySQL 降序索引对 DESC 有优化，方向不匹配会导致额外排序",
						Recommendation: "考虑使用 " + orderBy.Field + " " + oppositeDirection(collation),
						Confidence:     0.85,
						Tags:           []string{"mysql", "descending-index", "order-by"},
						SourceFile:     query.SourceFile,
						LineNumber:     query.LineNumber,
					})
				} else if direction == "ASC" && collation == "D" {
					suggestions = append(suggestions, advisor.Suggestion{
						RuleID:         r.ID(),
						Severity:       r.Severity(),
						Message:        fmt.Sprintf("ORDER BY %s ASC 与索引 %s 降序方向不匹配", orderBy.Field, idxName),
						Reason:         "MySQL 降序索引对 DESC 有优化，ASC 方向无法利用该优化",
						Recommendation: "考虑使用 " + orderBy.Field + " DESC",
						Confidence:     0.85,
						Tags:           []string{"mysql", "descending-index", "order-by"},
						SourceFile:     query.SourceFile,
						LineNumber:     query.LineNumber,
					})
				}
			}
		}
	}

	return suggestions
}

func oppositeDirection(collation string) string {
	if collation == "D" {
		return "ASC"
	}
	return "DESC"
}
