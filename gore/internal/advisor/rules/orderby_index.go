package rules

import (
	"fmt"
	"strings"

	"gore/internal/advisor"
)

// OrderByIndexRule checks order by fields for index coverage.
type OrderByIndexRule struct{}

func NewOrderByIndexRule() *OrderByIndexRule { return &OrderByIndexRule{} }

func (r *OrderByIndexRule) ID() string { return "IDX-008" }

func (r *OrderByIndexRule) Name() string { return "Order By Index Coverage" }

func (r *OrderByIndexRule) Description() string {
	return "ORDER BY 字段缺少索引时可能导致排序开销"
}

func (r *OrderByIndexRule) Severity() advisor.Severity { return advisor.SeverityInfo }

func (r *OrderByIndexRule) WhyDoc() string { return "https://docs.gore.io/rules/IDX-008" }

func (r *OrderByIndexRule) Check(query *advisor.QueryMetadata, schema *advisor.TableSchema) []advisor.Suggestion {
	if query == nil || schema == nil {
		return nil
	}

	var out []advisor.Suggestion
	for _, order := range query.OrderBy {
		if !hasIndexOn(schema.Indexes, order.Field) {
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
		}
	}

	return out
}

func hasIndexOn(indexes []advisor.IndexInfo, field string) bool {
	for _, idx := range indexes {
		for i, col := range idx.Columns {
			if strings.EqualFold(col, field) && i == 0 {
				return true
			}
		}
	}
	return false
}
