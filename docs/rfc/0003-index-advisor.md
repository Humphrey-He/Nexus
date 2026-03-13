# RFC-0003: Index Advisor 系统设计

| 字段 | 内容 |
|------|------|
| **RFC 号** | RFC-0003 |
| **标题** | Index Advisor: 编译期索引风险诊断系统 |
| **状态** | Proposed |
| **优先级** | P1（工程化核心特性） |
| **作者** | [Your Name] |
| **创建日期** | 2026-03-13 |
| **目标版本** | gore v1.1.0-alpha |
| **相关 RFC** | RFC-0001 (DbContext 设计), RFC-0002 (Change Tracker) |
| **交付周期** | M3-M4（共 4 周） |

---

## 1. 问题陈述 (Problem Statement)

### 1.1 核心痛点

在大型分布式系统开发中，由于 ORM 框架隐藏了底层 SQL 逻辑，开发者常面临以下问题：

| 痛点 | 现象 | 影响 |
|------|------|------|
| **黑盒查询** | 开发者不清楚 WHERE 条件是否命中了索引 | 无法在开发期发现问题 |
| **慢查询滞后性** | 索引问题通常在生产环境数据量增大后才暴露 | 修复成本高、业务影响大 |
| **索引失效隐患** | 函数计算、类型不匹配、违反最左匹配原则 | 线上 P0 故障频发 |
| **重复索引浪费** | 无法识别冗余索引 | 存储空间浪费、写入性能下降 |

### 1.2 设计目标

**主目标：**
- 在编译期或开发期发现潜在索引缺失风险
- 提供可解释性的优化建议，而非简单错误提示
- 支持 Live Schema 校验，确保建议基于真实数据库状态

**次目标：**
- 零运行时开销（Static Mode）
- 支持离线 CI 环境（Schema Cache Mode）
- 与 IDE 和 GitHub Actions 无缝集成

### 1.3 成功指标 (Success Criteria)

| 指标 | 目标 | 验收条件 |
|------|------|---------|
| **误报率** | < 5% | 手工审核 100 个建议，误报不超过 5 个 |
| **漏报率** | < 10% | 已知的 10 个常见索引问题，捕获率 ≥ 90% |
| **诊断耗时** | < 100ms | 单次分析耗时不超过 100ms |
| **覆盖度** | ≥ 80% | 支持至少 80% 的常见查询模式 |

---

## 2. 方案概览 (High-level Design)

### 2.1 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                    gore-lint CLI Tool                       │
│                  (主入口，支持多种模式)                      │
└──────────────────────┬──────────────────────────────────────┘
                       │
        ┌──────────────┼──────────────┐
        │              │              │
    ┌───▼────┐    ┌───▼────┐    ┌───▼────┐
    │ Static │    │ Runtime│    │ Offline│
    │Analyzer│    │Interceptor  │ Schema │
    └───┬────┘    └───┬────┘    └───┬────┘
        │              │              │
        └──────────────┼──────────────┘
                       │
        ┌──────────────▼──────────────┐
        │   SQL Semantic Extractor    │
        │  (AST 解析 + 查询语义抽取)   │
        └──────────────┬──────────────┘
                       │
        ┌──────────────▼──────────────┐
        │ Heuristic Rule Engine       │
        │ (规则匹配 + 诊断输出)        │
        └──────────────┬──────────────┘
                       │
        ┌──────────────▼──────────────┐
        │ Schema Metadata Provider    │
        │ (PG + Cache + Mock)         │
        └─────────────────────────────┘
```

### 2.2 核心组件职责

| 组件 | 职责 | 输入 | 输出 |
|------|------|------|------|
| **SQL Semantic Extractor** | 从 ORM 调用链或 SQL AST 中提取查询语义 | Query Builder 链 / SQL 字符串 | `QueryMetadata` |
| **Schema Metadata Provider** | 获取实时或缓存的数据库 Schema 信息 | 表名、列名 | `TableSchema` |
| **Heuristic Rule Engine** | 执行诊断规则，生成建议 | `QueryMetadata` + `TableSchema` | `[]Suggestion` |
| **gore-lint CLI** | 命令行入口，支持多种运行模式 | 源代码路径 / 配置文件 | JSON / HTML 报告 |

---

## 3. 详细设计 (Detailed Design)

### 3.1 诊断模式选择

#### Mode A: Static Analyzer (Linter 模式) ⭐ 优先实现

**实现方式：**
- 基于 `go/analysis` 框架开发静态分析器
- 集成为 `golangci-lint` 插件或独立工具
- 在编译期扫描 ORM 调用链

**优点：**
- 零运行时开销
- 集成于 CI 流程，无需数据库连接
- 反馈快速，开发体验好

**缺点：**
- 无法处理高度动态的 SQL 拼接
- 需要静态配置 Schema 信息

**适用场景：**
- 标准 ORM 查询（推荐用法）
- CI/CD 流程集成
- 代码审查自动化

---

#### Mode B: Debug Interceptor (运行时诊断模式)

**实现方式：**
- 在 `DbContext.Execute()` 前后拦截
- 连接真实数据库，动态获取 Schema
- 在日志中输出诊断结果

**优点：**
- 获取最终的真实 SQL，诊断最准确
- 支持动态 SQL 拼接
- 基于实际数据分布的建议

**缺点：**
- 需要数据库连接，有运行时开销
- 仅在 Debug 环境可用

**适用场景：**
- 本地开发调试
- 性能测试环节
- 生产环境慢查询分析

---

#### Mode C: Offline Schema Cache (离线模式)

**实现方式：**
- 将 Schema 信息导出为 `schema.json`
- 在 CI 环境中加载缓存文件进行分析

**优点：**
- 支持离线 CI 环境
- 快速反馈，无需网络

**缺点：**
- Schema 信息可能过期
- 需要手动同步

**适用场景：**
- 完全隔离的 CI 环境
- 多环境部署

---

### 3.2 SQL 语义提取 (Semantic Extraction)

#### 3.2.1 数据模型定义

```go
// QueryMetadata 查询元数据
type QueryMetadata struct {
    TableName  string
    Columns    []string
    Conditions []Condition
    OrderBy    []OrderField
    GroupBy    []string
    Joins      []JoinClause
    Limit      *int
    Offset     *int
    IsDistinct bool
    SourceFile string
    LineNumber int
}

// Condition WHERE 条件
type Condition struct {
    Field      string
    Operator   string
    Value      any
    ValueType  string
    IsFunction bool
    FuncName   string
    IsNegated  bool
}

// OrderField 排序字段
type OrderField struct {
    Field     string
    Direction string
}

// JoinClause JOIN 信息
type JoinClause struct {
    Type         string
    Table        string
    OnConditions []Condition
}
```

#### 3.2.2 提取策略

**策略 A：ORM 链式调用拦截（推荐）**

```go
// 在 gore 的 Query Builder 中埋点
// 记录条件、排序、表名、字段等元信息
```

**策略 B：SQL AST 解析（备选）**

使用 `pingcap/parser` 或 `vitess` 的 SQL 解析器，从最终 SQL 字符串反向提取元数据。

---

### 3.3 诊断规则引擎 (Rule Engine)

#### 3.3.1 规则体系

```go
// Rule 诊断规则接口
type Rule interface {
    ID() string
    Name() string
    Description() string
    Severity() SeverityLevel
    Check(query *QueryMetadata, schema *TableSchema) []Suggestion
    WhyDoc() string
}

type SeverityLevel string

const (
    SeverityInfo     SeverityLevel = "Info"
    SeverityWarning  SeverityLevel = "Warning"
    SeverityHighRisk SeverityLevel = "HighRisk"
    SeverityCritical SeverityLevel = "Critical"
)

// Suggestion 诊断建议
type Suggestion struct {
    RuleID       string
    Severity     SeverityLevel
    Message      string
    Reason       string
    SQLFix       string
    Confidence   float64
    SourceFile   string
    LineNumber   int
    Tags         []string
    RelatedRules []string
}
```

#### 3.3.2 内置规则清单

| 规则ID | 规则名称 | 严重级别 | 说明 |
|--------|---------|---------|------|
| **IDX-001** | 最左匹配检查 | Warning | 联合索引未命中第一个字段 |
| **IDX-002** | 函数索引失效 | HighRisk | WHERE 条件中使用了函数 |
| **IDX-003** | 隐式类型转换 | Warning | 字段类型与参数类型不匹配 |
| **IDX-004** | 前缀模糊查询 | Warning | LIKE '%abc' 无法使用索引 |
| **IDX-005** | 否定条件 | Warning | 使用 !=、NOT IN 等 |
| **IDX-006** | 索引缺失 | HighRisk | 频繁查询的字段无索引 |
| **IDX-007** | 冗余索引 | Info | 存在包含关系的索引 |
| **IDX-008** | 排序字段索引 | Info | ORDER BY 的字段应建索引 |
| **IDX-009** | JOIN 字段索引 | Warning | JOIN 条件字段应建索引 |
| **IDX-010** | 选择性低索引 | Info | 索引列的唯一值比例过低 |

#### 3.3.3 规则实现示例

**规则 IDX-001：最左匹配检查**

```go
type LeftmostMatchRule struct{}

func (r *LeftmostMatchRule) ID() string { return "IDX-001" }
func (r *LeftmostMatchRule) Name() string { return "Leftmost Match Validation" }
func (r *LeftmostMatchRule) Description() string {
    return "验证联合索引是否使用了最左列"
}
func (r *LeftmostMatchRule) Severity() SeverityLevel { return SeverityWarning }
func (r *LeftmostMatchRule) WhyDoc() string { return "https://docs.gore.io/rules/IDX-001" }

func (r *LeftmostMatchRule) Check(query *QueryMetadata, schema *TableSchema) []Suggestion {
    var suggestions []Suggestion

    for _, idx := range schema.Indexes {
        if len(idx.Columns) <= 1 {
            continue
        }

        firstCol := idx.Columns[0]
        found := false
        for _, cond := range query.Conditions {
            if cond.Field == firstCol && !cond.IsNegated {
                found = true
                break
            }
        }

        if !found {
            suggestions = append(suggestions, Suggestion{
                RuleID:     r.ID(),
                Severity:   r.Severity(),
                Message:    "联合索引首列未出现在 WHERE 条件中",
                Reason:     "B-Tree 索引遵循最左匹配原则，必须从第一列开始",
                Confidence: 0.95,
                Tags:       []string{"index", "leftmost-match"},
            })
        }
    }

    return suggestions
}
```

---

## 4. 开放问题 (Open Questions)

- 是否引入默认 hash snapshot？
- 何时触发快照更新（SaveChanges 后还是按事务边界）？
- AsTracking 的默认行为是否与 EF 保持一致？

---

## 5. 里程碑 (Milestones)

| 阶段 | 里程碑 | 交付物 |
|------|--------|--------|
| **M3** | Linter 结构搭建 | gore-lint CLI 入口 + Rule Engine 骨架 |
| **M4** | 规则实现 | IDX-001 ~ IDX-004 规则落地 |

---

## 6. 参考 (References)

- RFC-0001: DbContext 设计
- RFC-0002: Change Tracker 方案对比