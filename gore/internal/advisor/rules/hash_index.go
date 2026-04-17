package rules

import (
	"fmt"
	"strings"

	"gore/internal/advisor"
)

// HashIndexRangeRule detects range queries on HASH indexes.
type HashIndexRangeRule struct{}

func NewHashIndexRangeRule() *HashIndexRangeRule {
	return &HashIndexRangeRule{}
}

func (r *HashIndexRangeRule) ID() string { return "IDX-MYSQL-003" }

func (r *HashIndexRangeRule) Name() string { return "Hash Index Range Query" }

func (r *HashIndexRangeRule) Description() string {
	return "检测 HASH 索引上的范围查询，这些查询无法使用索引"
}

func (r *HashIndexRangeRule) Severity() advisor.Severity { return advisor.SeverityWarn }

func (r *HashIndexRangeRule) WhyDoc() string { return "" }

func (r *HashIndexRangeRule) Check(query *advisor.QueryMetadata, schema *advisor.TableSchema) []advisor.Suggestion {
	if query == nil || schema == nil {
		return nil
	}

	// Range operators that HASH indexes cannot use
	rangeOperators := map[string]bool{
		">":  true,
		"<":  true,
		">=": true,
		"<=": true,
		"<>": true,
		"!=": true,
	}

	var suggestions []advisor.Suggestion

	// Build a map of HASH indexed columns
	hashIndexedColumns := make(map[string]string) // column -> index name
	for _, idx := range schema.Indexes {
		if strings.ToUpper(idx.Method) == "HASH" {
			for _, col := range idx.Columns {
				hashIndexedColumns[col] = idx.Name
			}
		}
	}

	// Check each condition
	for _, cond := range query.Conditions {
		if hashIdxName, isHashIndexed := hashIndexedColumns[cond.Field]; isHashIndexed {
			op := strings.ToUpper(cond.Operator)
			if rangeOperators[op] || op == "LIKE" {
				suggestions = append(suggestions, advisor.Suggestion{
					RuleID:         r.ID(),
					Severity:       r.Severity(),
					Message:        fmt.Sprintf("字段 %s 使用 HASH 索引 %s 进行范围查询", cond.Field, hashIdxName),
					Reason:         "HASH 索引仅支持等值比较 (=/IN)，范围查询会导致全表扫描",
					Recommendation: "考虑将 HASH 索引改为 BTREE 索引以支持范围查询",
					Confidence:     0.95,
					Tags:           []string{"mysql", "hash-index", "range-query"},
					SourceFile:     query.SourceFile,
					LineNumber:     query.LineNumber,
				})
			}
		}
	}

	return suggestions
}

// HashIndexRule is a type alias for backwards compatibility
type HashIndexRule = HashIndexRangeRule
