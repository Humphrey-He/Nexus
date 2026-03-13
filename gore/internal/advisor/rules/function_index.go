package rules

import (
	"fmt"
	"strings"

	"gore/internal/advisor"
)

// FunctionIndexRule detects function usage in predicates.
type FunctionIndexRule struct{}

func NewFunctionIndexRule() *FunctionIndexRule { return &FunctionIndexRule{} }

func (r *FunctionIndexRule) ID() string { return "IDX-002" }

func (r *FunctionIndexRule) Name() string { return "Function Index Invalidates" }

func (r *FunctionIndexRule) Description() string {
	return "WHERE 条件中使用函数可能导致索引失效"
}

func (r *FunctionIndexRule) Severity() advisor.Severity { return advisor.SeverityHigh }

func (r *FunctionIndexRule) WhyDoc() string { return "https://docs.gore.io/rules/IDX-002" }

func (r *FunctionIndexRule) Check(query *advisor.QueryMetadata, schema *advisor.TableSchema) []advisor.Suggestion {
	if query == nil || schema == nil {
		return nil
	}

	var out []advisor.Suggestion
	for _, cond := range query.Conditions {
		if !cond.IsFunction && !strings.Contains(cond.Field, "(") {
			continue
		}
		funcName := cond.FuncName
		if funcName == "" {
			funcName = "function"
		}
		out = append(out, advisor.Suggestion{
			RuleID:         r.ID(),
			Severity:       r.Severity(),
			Message:        fmt.Sprintf("字段 %s 使用了函数 %s", cond.Field, funcName),
			Reason:         "函数包裹字段会导致索引失效或无法命中",
			Recommendation: "避免在 WHERE 字段上直接使用函数，或考虑函数索引",
			Confidence:     0.85,
			Tags:           []string{"index", "function"},
			SourceFile:     query.SourceFile,
			LineNumber:     query.LineNumber,
		})
	}

	return out
}
