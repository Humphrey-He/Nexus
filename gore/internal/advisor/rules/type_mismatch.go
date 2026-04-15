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
			Reason:         "隐式类型转换会导致索引无法命中",
			Recommendation: "保持参数类型与字段类型一致",
			Confidence:     0.8,
			Tags:           []string{"index", "type"},
			SourceFile:     query.SourceFile,
			LineNumber:     query.LineNumber,
		})
	}

	return out
}

// isTypeCompatible returns true if column type is compatible with value type.
// In PostgreSQL, implicit type conversion can prevent index usage.
func isTypeCompatible(columnType, valueType string) bool {
	col := strings.ToLower(columnType)
	val := strings.ToLower(valueType)

	// Exact match - always compatible
	if col == val {
		return true
	}

	// PostgreSQL type families
	intFamily := []string{"int", "int2", "int4", "int8", "integer", "bigint", "smallint", "serial"}
	floatFamily := []string{"float", "float4", "float8", "real", "double precision", "numeric", "decimal"}
	stringFamily := []string{"char", "text", "varchar", "uuid", "bpchar"}
	boolFamily := []string{"bool", "boolean"}

	isColInt := containsAny(col, intFamily)
	isColFloat := containsAny(col, floatFamily)
	isColString := containsAny(col, stringFamily)
	isColBool := containsAny(col, boolFamily)

	isValInt := val == "int" || val == "int64" || val == "int32"
	isValFloat := val == "float" || val == "float64" || val == "float32"
	isValString := val == "string"
	isValBool := val == "bool"

	// CRITICAL: int and float are NOT compatible - this causes implicit cast
	if (isColInt && isValFloat) || (isColFloat && isValInt) {
		return false
	}

	// int family - allow implicit widening (int → bigint, int → numeric)
	// but NOT narrowing (bigint → int) which may cause issues
	if isColInt && isValInt {
		// PostgreSQL can implicit cast int4 to int8, but not always the other way
		// For safety, only allow if column type is "larger" or equal
		if col == "int8" || col == "bigint" || col == "numeric" {
			return true
		}
		if col == "int4" || col == "integer" || col == "int" {
			// int/int4 can accept int values safely
			return true
		}
		return false
	}

	// float family
	if isColFloat && isValFloat {
		// float4 → float8 is safe, other conversions may have precision loss
		if col == "float8" || col == "double precision" || col == "numeric" || col == "decimal" {
			return true
		}
		if col == "float4" || col == "real" || col == "float" {
			return true
		}
		return false
	}

	// string family - varchar/text/char are mutually compatible
	if isColString && isValString {
		return true
	}

	// CRITICAL: int column with string value is definitely incompatible
	if isColInt && isValString {
		return false
	}

	// CRITICAL: string column with int value is also problematic
	if isColString && isValInt {
		return false
	}

	// bool family
	if isColBool && isValBool {
		return true
	}

	// CRITICAL: bool column with non-bool value is incompatible
	if isColBool && (isValString || isValInt || isValFloat) {
		return false
	}

	// CRITICAL: string column with bool value is incompatible
	if isColString && isValBool {
		return false
	}

	// Unknown types - assume safe to avoid false positives
	return true
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
