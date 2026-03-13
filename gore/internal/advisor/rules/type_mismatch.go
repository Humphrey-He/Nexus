package rules

import (
	"fmt"
	"strings"

	"gore/internal/advisor"
)

// TypeMismatchRule detects implicit type conversions.
type TypeMismatchRule struct{}

func NewTypeMismatchRule() *TypeMismatchRule { return &TypeMismatchRule{} }

func (r *TypeMismatchRule) ID() string { return "IDX-003" }

func (r *TypeMismatchRule) Name() string { return "Implicit Type Conversion" }

func (r *TypeMismatchRule) Description() string {
	return "字段类型与参数类型不匹配可能导致索引失效"
}

func (r *TypeMismatchRule) Severity() advisor.Severity { return advisor.SeverityWarn }

func (r *TypeMismatchRule) WhyDoc() string { return "https://docs.gore.io/rules/IDX-003" }

func (r *TypeMismatchRule) Check(query *advisor.QueryMetadata, schema *advisor.TableSchema) []advisor.Suggestion {
	if query == nil || schema == nil || len(schema.Columns) == 0 {
		return nil
	}

	colTypes := map[string]string{}
	for _, col := range schema.Columns {
		colTypes[col.Name] = strings.ToLower(col.Type)
	}

	var out []advisor.Suggestion
	for _, cond := range query.Conditions {
		colType, ok := colTypes[cond.Field]
		if !ok || cond.ValueType == "" {
			continue
		}
		if isTypeCompatible(colType, cond.ValueType) {
			continue
		}

		out = append(out, advisor.Suggestion{
			RuleID:         r.ID(),
			Severity:       r.Severity(),
			Message:        fmt.Sprintf("字段 %s 类型(%s) 与参数类型(%s) 不匹配", cond.Field, colType, cond.ValueType),
			Reason:         "隐式类型转换可能导致索引无法命中",
			Recommendation: "保持参数类型与字段类型一致，避免隐式转换",
			Confidence:     0.7,
			Tags:           []string{"index", "type"},
			SourceFile:     query.SourceFile,
			LineNumber:     query.LineNumber,
		})
	}

	return out
}

func isTypeCompatible(columnType, valueType string) bool {
	switch valueType {
	case "int", "int64":
		return strings.Contains(columnType, "int") || strings.Contains(columnType, "numeric")
	case "float", "float64":
		return strings.Contains(columnType, "float") || strings.Contains(columnType, "numeric")
	case "string":
		return strings.Contains(columnType, "char") || strings.Contains(columnType, "text") || strings.Contains(columnType, "uuid")
	case "bool":
		return strings.Contains(columnType, "bool")
	default:
		return true
	}
}
