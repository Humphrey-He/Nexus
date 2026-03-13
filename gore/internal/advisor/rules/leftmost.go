package rules

import (
	"fmt"

	"gore/internal/advisor"
)

// LeftmostMatchRule checks if composite indexes use the leftmost column.
type LeftmostMatchRule struct{}

func NewLeftmostMatchRule() *LeftmostMatchRule {
	return &LeftmostMatchRule{}
}

func (r *LeftmostMatchRule) ID() string { return "IDX-001" }

func (r *LeftmostMatchRule) Name() string { return "Leftmost Match Validation" }

func (r *LeftmostMatchRule) Description() string {
	return "验证联合索引是否使用了最左列"
}

func (r *LeftmostMatchRule) Severity() advisor.Severity { return advisor.SeverityWarn }

func (r *LeftmostMatchRule) WhyDoc() string { return "https://docs.gore.io/rules/IDX-001" }

func (r *LeftmostMatchRule) Check(query *advisor.QueryMetadata, schema *advisor.TableSchema) []advisor.Suggestion {
	if query == nil || schema == nil {
		return nil
	}

	var suggestions []advisor.Suggestion
	for _, idx := range schema.Indexes {
		if len(idx.Columns) <= 1 {
			continue
		}

		firstCol := idx.Columns[0]
		found := false
		for _, cond := range query.Conditions {
			if cond.Field == firstCol && !cond.IsNegated {
				found = true
				break
			}
		}

		if !found {
			suggestions = append(suggestions, advisor.Suggestion{
				RuleID:         r.ID(),
				Severity:       r.Severity(),
				Message:        fmt.Sprintf("联合索引 %s 的首列 %s 未在 WHERE 中出现", idx.Name, firstCol),
				Reason:         "B-Tree 索引遵循最左匹配原则，必须从第一列开始",
				Recommendation: fmt.Sprintf("考虑在 WHERE 中加入 %s 或调整索引顺序", firstCol),
				Confidence:     0.95,
				Tags:           []string{"index", "leftmost-match"},
				SourceFile:     query.SourceFile,
				LineNumber:     query.LineNumber,
			})
		}
	}

	return suggestions
}
