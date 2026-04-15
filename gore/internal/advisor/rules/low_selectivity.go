package rules

import (
	"fmt"

	"gore/internal/advisor"
)

// LowSelectivityRule detects indexes with very low cardinality that provide little benefit.
type LowSelectivityRule struct{}

func NewLowSelectivityRule() *LowSelectivityRule { return &LowSelectivityRule{} }

func (r *LowSelectivityRule) ID() string { return "IDX-010" }

func (r *LowSelectivityRule) Name() string { return "Low Selectivity Index" }

func (r *LowSelectivityRule) Description() string {
	return "低选择率索引（唯一值比例极低）可能无实际价值"
}

func (r *LowSelectivityRule) Severity() advisor.Severity { return advisor.SeverityInfo }

func (r *LowSelectivityRule) WhyDoc() string { return "https://docs.gore.io/rules/IDX-010" }

// Check checks single indexes for low selectivity based on stored metadata.
// Selectivity is determined by index.Columns[0]'s selectivity value if available.
// A selectivity < 0.01 (1%) is considered very low.
func (r *LowSelectivityRule) Check(query *advisor.QueryMetadata, schema *advisor.TableSchema) []advisor.Suggestion {
	// This rule works on schema.Indexes, not on individual queries
	// It requires index metadata with selectivity information
	return nil
}

// CheckSchema analyzes indexes in the schema for low selectivity.
// Selectivity data must be populated in IndexInfo.Metadata["selectivity"].
func (r *LowSelectivityRule) CheckSchema(schema *advisor.TableSchema) []advisor.Suggestion {
	if schema == nil || len(schema.Indexes) == 0 {
		return nil
	}

	var out []advisor.Suggestion

	for _, idx := range schema.Indexes {
		// selectivity is stored in index metadata if available from live DSN
		selectivity, ok := idx.Metadata["selectivity"]
		if !ok {
			// No selectivity data available - skip with low confidence
			continue
		}

		sel, ok := selectivity.(float64)
		if !ok {
			continue
		}

		if sel < 0.01 {
			out = append(out, advisor.Suggestion{
				RuleID:         r.ID(),
				Severity:       r.Severity(),
				Message:        fmt.Sprintf("索引 %s 选择率极低 (%.2f%%)，可能无实际价值", idx.Name, sel*100),
				Reason:         "索引列唯一值比例低于1%，索引扫描效果接近全表扫描",
				Recommendation: "考虑移除该索引或使用组合索引提升选择性",
				Confidence:     0.9,
				Tags:           []string{"index", "selectivity"},
			})
		} else if sel < 0.1 {
			out = append(out, advisor.Suggestion{
				RuleID:         r.ID(),
				Severity:       advisor.SeverityWarn,
				Message:        fmt.Sprintf("索引 %s 选择率较低 (%.2f%%)，效果有限", idx.Name, sel*100),
				Reason:         "索引列唯一值比例低于10%，索引效率有限",
				Recommendation: "考虑使用组合索引或确认该字段是否适合作为索引列",
				Confidence:     0.75,
				Tags:           []string{"index", "selectivity"},
			})
		}
	}

	return out
}
