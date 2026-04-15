package rules

import (
	"fmt"
	"strings"

	"gore/internal/advisor"
)

// JoinIndexRule checks join fields for index coverage.
type JoinIndexRule struct{}

func NewJoinIndexRule() *JoinIndexRule { return &JoinIndexRule{} }

func (r *JoinIndexRule) ID() string { return "IDX-009" }

func (r *JoinIndexRule) Name() string { return "Join Field Index" }

func (r *JoinIndexRule) Description() string {
	return "JOIN 条件字段缺少索引会导致大表扫描"
}

func (r *JoinIndexRule) Severity() advisor.Severity { return advisor.SeverityWarn }

func (r *JoinIndexRule) WhyDoc() string { return "https://docs.gore.io/rules/IDX-009" }

func (r *JoinIndexRule) Check(query *advisor.QueryMetadata, schema *advisor.TableSchema) []advisor.Suggestion {
	if query == nil || schema == nil {
		return nil
	}

	var out []advisor.Suggestion
	for _, join := range query.Joins {
		for _, cond := range join.OnConditions {
			if !hasJoinIndexOn(schema.Indexes, cond.Field) {
				out = append(out, advisor.Suggestion{
					RuleID:         r.ID(),
					Severity:       r.Severity(),
					Message:        fmt.Sprintf("JOIN 字段 %s 缺少索引", cond.Field),
					Reason:         "JOIN 字段无索引会导致扫描和高开销",
					Recommendation: "为 JOIN 字段建立索引",
					Confidence:     0.6,
					Tags:           []string{"index", "join"},
					SourceFile:     query.SourceFile,
					LineNumber:     query.LineNumber,
				})
			}
		}
	}

	return out
}

// hasJoinIndexOn checks if a field has an index suitable for JOIN operations.
// For JOIN, the field should ideally be the first column of an index.
func hasJoinIndexOn(indexes []advisor.IndexInfo, field string) bool {
	for _, idx := range indexes {
		if len(idx.Columns) > 0 && strings.EqualFold(idx.Columns[0], field) {
			return true
		}
	}
	return false
}
