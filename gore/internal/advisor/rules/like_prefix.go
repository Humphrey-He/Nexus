package rules

import (
	"fmt"
	"strings"

	"gore/internal/advisor"
)

// LikePrefixRule detects leading wildcard patterns.
type LikePrefixRule struct{}

func NewLikePrefixRule() *LikePrefixRule { return &LikePrefixRule{} }

func (r *LikePrefixRule) ID() string { return "IDX-004" }

func (r *LikePrefixRule) Name() string { return "Prefix Wildcard LIKE" }

func (r *LikePrefixRule) Description() string {
	return "LIKE '%xxx' 无法使用索引"
}

func (r *LikePrefixRule) Severity() advisor.Severity { return advisor.SeverityWarn }

func (r *LikePrefixRule) WhyDoc() string { return "https://docs.gore.io/rules/IDX-004" }

func (r *LikePrefixRule) Check(query *advisor.QueryMetadata, schema *advisor.TableSchema) []advisor.Suggestion {
	if query == nil || schema == nil {
		return nil
	}

	var out []advisor.Suggestion
	for _, cond := range query.Conditions {
		if strings.ToUpper(cond.Operator) != "LIKE" {
			continue
		}
		pattern, ok := cond.Value.(string)
		if !ok {
			continue
		}
		if strings.HasPrefix(pattern, "%") {
			out = append(out, advisor.Suggestion{
				RuleID:         r.ID(),
				Severity:       r.Severity(),
				Message:        fmt.Sprintf("字段 %s 使用前缀通配 LIKE 模式", cond.Field),
				Reason:         "前缀通配会导致索引无法命中",
				Recommendation: "避免前缀通配，改用后缀通配或全文索引",
				Confidence:     0.8,
				Tags:           []string{"index", "like"},
				SourceFile:     query.SourceFile,
				LineNumber:     query.LineNumber,
			})
		}
	}

	return out
}
