package rules

import (
	"fmt"
	"strings"

	"gore/internal/advisor"
)

// LeftmostMatchRule checks if composite indexes use the leftmost column in the correct order.
type LeftmostMatchRule struct{}

func NewLeftmostMatchRule() *LeftmostMatchRule {
	return &LeftmostMatchRule{}
}

func (r *LeftmostMatchRule) ID() string { return "IDX-001" }

func (r *LeftmostMatchRule) Name() string { return "Leftmost Match Validation" }

func (r *LeftmostMatchRule) Description() string {
	return "验证联合索引是否从第一列开始并按顺序匹配"
}

func (r *LeftmostMatchRule) Severity() advisor.Severity { return advisor.SeverityWarn }

func (r *LeftmostMatchRule) WhyDoc() string { return "https://docs.gore.io/rules/IDX-001" }

func (r *LeftmostMatchRule) Check(query *advisor.QueryMetadata, schema *advisor.TableSchema) []advisor.Suggestion {
	if query == nil || schema == nil {
		return nil
	}

	// Extract all condition fields (including negated, for index coverage check)
	allCondFields := make([]string, 0, len(query.Conditions))
	for _, cond := range query.Conditions {
		if !cond.IsFunction {
			allCondFields = append(allCondFields, cond.Field)
		}
	}

	// Extract non-negated, non-function conditions in order for order checking
	condFields := make([]string, 0)
	for _, cond := range query.Conditions {
		if !cond.IsNegated && !cond.IsFunction {
			condFields = append(condFields, cond.Field)
		}
	}

	// If no conditions at all, nothing to check
	if len(allCondFields) == 0 {
		return nil
	}

	var suggestions []advisor.Suggestion
	for _, idx := range schema.Indexes {
		if len(idx.Columns) <= 1 {
			continue
		}

		// Check if ANY column in this index is used in the query
		anyIndexColUsed := false
		for _, condField := range allCondFields {
			for _, idxCol := range idx.Columns {
				if strings.EqualFold(condField, idxCol) {
					anyIndexColUsed = true
					break
				}
			}
			if anyIndexColUsed {
				break
			}
		}

		// If no columns from this index are used, skip this index
		if !anyIndexColUsed {
			continue
		}

		// Check if the first index column appears in conditions (including negated)
		firstCol := idx.Columns[0]
		firstFound := false
		for _, field := range allCondFields {
			if strings.EqualFold(field, firstCol) {
				firstFound = true
				break
			}
		}

		if !firstFound {
			suggestions = append(suggestions, advisor.Suggestion{
				RuleID:         r.ID(),
				Severity:       r.Severity(),
				Message:        fmt.Sprintf("联合索引 %s 的首列 %s 未在 WHERE 中出现", idx.Name, firstCol),
				Reason:         "B-Tree 索引遵循最左匹配原则，必须从第一列开始",
				Recommendation: "考虑在 WHERE 中加入 " + firstCol + " 条件或调整索引顺序",
				Confidence:     0.95,
				Tags:           []string{"index", "leftmost-match"},
				SourceFile:     query.SourceFile,
				LineNumber:     query.LineNumber,
			})
			continue
		}

		// Check column order using non-negated conditions
		// We can skip columns from the right (e.g., WHERE a=1 AND c=2 with index (a,b,c) is OK)
		// But we cannot skip columns from the left (e.g., WHERE b=1 AND c=2 with index (a,b,c) is NOT OK)
		matchedIdx := -1
		for _, condField := range condFields {
			matchedIdx++
			// Find this condition's position in the index
			idxPos := -1
			for i, idxCol := range idx.Columns {
				if strings.EqualFold(condField, idxCol) {
					idxPos = i
					break
				}
			}
			if idxPos == -1 {
				// Field not in index, skip
				continue
			}
			if idxPos != matchedIdx && matchedIdx < len(idx.Columns) {
				// Position mismatch: we expected matchedIdx but got idxPos
				// Allow if we're matching a column further right (skipping intermediate columns)
				// But NOT if we're trying to match a column that would require skipping the first column
				if matchedIdx == 0 && idxPos > 0 {
					suggestions = append(suggestions, advisor.Suggestion{
						RuleID:         r.ID(),
						Severity:       r.Severity(),
						Message:        fmt.Sprintf("联合索引 %s 的列顺序与 WHERE 条件不匹配", idx.Name),
						Reason:         "WHERE 条件跳过了索引列，无法有效利用索引",
						Recommendation: "调整 WHERE 条件顺序或索引列顺序",
						Confidence:     0.85,
						Tags:           []string{"index", "leftmost-match", "order-mismatch"},
						SourceFile:     query.SourceFile,
						LineNumber:     query.LineNumber,
					})
					break
				}
			}
		}
	}

	return suggestions
}
