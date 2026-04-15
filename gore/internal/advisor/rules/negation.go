package rules

import (
	"fmt"

	"gore/internal/advisor"
)

// NegationRule detects negated predicates that may prevent index usage.
type NegationRule struct{}

func NewNegationRule() *NegationRule { return &NegationRule{} }

func (r *NegationRule) ID() string { return "IDX-005" }

func (r *NegationRule) Name() string { return "Negated Predicate" }

func (r *NegationRule) Description() string {
	return "否定条件 (!=, <>, NOT IN, NOT LIKE) 可能导致索引无法命中"
}

func (r *NegationRule) Severity() advisor.Severity { return advisor.SeverityWarn }

func (r *NegationRule) WhyDoc() string { return "https://docs.gore.io/rules/IDX-005" }

func (r *NegationRule) Check(query *advisor.QueryMetadata, schema *advisor.TableSchema) []advisor.Suggestion {
	if query == nil || schema == nil {
		return nil
	}

	var out []advisor.Suggestion
	for _, cond := range query.Conditions {
		if !cond.IsNegated {
			continue
		}
		out = append(out, advisor.Suggestion{
			RuleID:         r.ID(),
			Severity:       r.Severity(),
			Message:        fmt.Sprintf("字段 %s 使用了否定条件操作符 %s", cond.Field, cond.Operator),
			Reason:         "否定条件通常无法利用索引扫描",
			Recommendation: "考虑改写为正向条件或确认业务逻辑",
			Confidence:     0.8,
			Tags:           []string{"index", "negation"},
			SourceFile:     query.SourceFile,
			LineNumber:     query.LineNumber,
		})
	}

	return out
}
