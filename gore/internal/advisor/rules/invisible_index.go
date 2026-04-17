package rules

import (
	"fmt"

	"gore/internal/advisor"
)

// InvisibleIndexRule detects invisible indexes that may be unused.
type InvisibleIndexRule struct{}

func NewInvisibleIndexRule() *InvisibleIndexRule {
	return &InvisibleIndexRule{}
}

func (r *InvisibleIndexRule) ID() string { return "IDX-MYSQL-001" }

func (r *InvisibleIndexRule) Name() string { return "Invisible Index Detection" }

func (r *InvisibleIndexRule) Description() string {
	return "检测 MySQL 8.0+ 的不可见索引"
}

func (r *InvisibleIndexRule) Severity() advisor.Severity { return advisor.SeverityInfo }

func (r *InvisibleIndexRule) WhyDoc() string { return "" }

func (r *InvisibleIndexRule) Check(query *advisor.QueryMetadata, schema *advisor.TableSchema) []advisor.Suggestion {
	if query == nil || schema == nil {
		return nil
	}

	// Extract fields used in the query
	usedFields := make(map[string]bool)
	for _, cond := range query.Conditions {
		usedFields[cond.Field] = true
	}
	for _, orderBy := range query.OrderBy {
		usedFields[orderBy.Field] = true
	}

	var suggestions []advisor.Suggestion

	for _, idx := range schema.Indexes {
		// Check if index is invisible via IsVisible field (MySQL 8.0+)
		if !idx.IsVisible {
			// Index is invisible, check if it's being used
			indexBeingUsed := false
			for _, col := range idx.Columns {
				if usedFields[col] {
					indexBeingUsed = true
					break
				}
			}

			if !indexBeingUsed {
				suggestions = append(suggestions, advisor.Suggestion{
					RuleID:         r.ID(),
					Severity:       r.Severity(),
					Message:        fmt.Sprintf("不可见索引 %s 未被当前查询使用", idx.Name),
					Reason:         "不可见索引对查询不可见，会导致全表扫描",
					Recommendation: "考虑删除该不可见索引或显式使用 USE INDEX",
					Confidence:     0.9,
					Tags:           []string{"mysql", "invisible-index"},
					SourceFile:     query.SourceFile,
					LineNumber:     query.LineNumber,
				})
			}
		}
	}

	return suggestions
}
