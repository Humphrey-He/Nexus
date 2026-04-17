# gore Index Advisor & CLI 后续规划

**作者**：架构评审委员会
**评审日期**：2026-04-15
**更新日期**：2026-04-15（第一轮优化完成）
**评审角色**：资深架构师 + DBA

---

## 一、现状全景评估

### 1.1 代码结构总览

```
gore/
├── cmd/gore-lint/main.go          # CLI 入口 (~700 行)
├── internal/advisor/              # ✅ 规则引擎（统一入口）
│   ├── advisor.go                 # 数据结构定义
│   ├── engine.go                  # 规则执行引擎
│   └── rules/                     # 10/10 规则已全部实现
├── dialect/postgres/
│   └── metadata_provider.go       # ✅ PostgreSQL 元数据拉取（已集成）
└── api/
    ├── dbcontext.go
    ├── dbset.go
    ├── query_builder.go           # QueryMetadata 埋点
    └── metrics.go
```

### 1.2 已实现功能清单

| 模块 | 完成度 | 说明 |
|------|--------|------|
| gore-lint CLI `check` 命令 | 95% | 静态分析，JSON/text 双格式输出 |
| 静态分析器 (AST) | 90% | 支持 From/WhereField/WhereIn/WhereLike/OrderBy/Limit/Offset |
| 常量/字面量解析 | 90% | 支持字符串/整数/浮点/布尔/负数/二元运算/类型包装器/函数调用 |
| Schema JSON 加载 | 100% | 完整实现 |
| PostgreSQL 实时拉取 | 100% | ✅ 已集成 `--dsn` 支持 |
| 规则引擎 | 95% | 接口设计合理，支持聚合规则 |
| IDX-001 最左匹配 | 90% | 基础实现 |
| IDX-002 函数索引 | 95% | ✅ Bug 已修复，支持 SelectorExpr 检测 |
| IDX-003 隐式类型转换 | 70% | 类型匹配逻辑偏宽松 |
| IDX-004 前缀通配 LIKE | 95% | 完整实现 |
| IDX-005 否定条件 | 100% | ✅ **新增** |
| IDX-006 缺失索引 | 100% | ✅ **新增**（频率聚合分析） |
| IDX-007 冗余索引 | 100% | ✅ **新增** |
| IDX-008 排序字段索引 | 75% | 仅检查首列 |
| IDX-009 JOIN 字段索引 | 75% | 仅检查首列 |
| IDX-010 低选择性索引 | 100% | ✅ **新增**（需要 DSN 拉取 selectivity 元数据） |

### 1.3 第一轮优化完成项（2026-04-15）

#### ✅ 已完成

| 问题 | 修复方案 |
|------|----------|
| P0 Bug: IDX-002 函数检测失效 | 新增 `evalFieldFuncName()` 识别 SelectorExpr/CallExpr |
| P0: 否定条件规则 IDX-005 缺失 | 新增 `rules/negation.go` |
| P1: CLI 未集成 --dsn | 新增 `loadLiveSchema()` + `fetchColumns/fetchIndexes()` |
| P1: 废弃包 internal/indexadvisor | 已删除整个目录 |
| P2: 路径排除功能缺失 | 新增 `--exclude` 标志 |
| P2: 输出格式单一 | 新增 `--format json\|text` |
| P1: IDX-006/007/010 缺失 | 全部实现 |

#### ⚠️ 仍需关注

| 问题 | 说明 |
|------|------|
| IDX-003 类型匹配 | `isTypeCompatible` 逻辑偏宽松（int/float 互认） |
| IDX-008/009 索引方向 | 未校验 ASC/DESC 与索引方向一致性 |
| 跨文件常量解析 | 当前仅支持文件内常量 |
| go/packages 集成 | 建议替代单文件解析以支持跨文件

#### P2 逻辑缺陷：IDX-001 未正确处理否定条件

`LeftmostMatchRule` 中的逻辑：

```go
if cond.Field == firstCol && !cond.IsNegated {
```

但 `parseWhereIn` 从未设置 `IsNegated = true`，且 AST 解析器也没有处理 `!=`、`NOT IN` 等否定操作符。

---

## 二、架构评审意见

### 2.1 CLI 设计评审

**优点**：
- 基于 `go/ast` 的静态分析路径正确，零运行时开销
- 支持 `gore-lint check ./...` 的标准 Go 包路径风格
- JSON 报告结构清晰，severity 分级合理
- 常量折叠能力（支持 `na"me"` 二元运算拼接）超出预期

**需改进**：

| 问题 | 严重度 | 说明 |
|------|--------|------|
| 不支持 `--dsn` 实时拉取 | P1 | PostgreSQL provider 已实现但未集成 |
| 不支持多 Schema 文件合并 | P2 | 大型项目有多 Schema 文件，当前仅支持单个 |
| 输出格式单一 | P2 | 仅 JSON，应支持 SARIF（GitHub Security Lab）格式便于 CI 集成 |
| 不支持排除路径 | P2 | 如 `gore-lint check ./... --exclude vendor --exclude *_test.go` |
| 无缓存机制 | P2 | 重复分析相同代码时无增量加速 |

### 2.2 规则引擎设计评审

**优点**：
- `Rule` 接口设计简洁，符合 Go idiom
- `Engine.Analyze` 并行友好的结构（未来可并行化规则执行）
- `Suggestion` 结构包含 Confidence 字段，为可信度传播预留空间

**需改进**：

| 问题 | 严重度 | 说明 |
|------|--------|------|
| 规则无注册机制 | P2 | `NewEngine` 手动传入所有规则，大型化后需自动发现 |
| 无规则禁用/白名单 | P2 | 无法按项目忽略特定规则 |
| 无规则优先级 | P3 | 高频规则可优先执行，早期短路 |
| 规则间无依赖声明 | P3 | IDX-007 冗余索引检测依赖 IDX-006 先完成 |

### 2.3 静态分析能力评审

**当前支持**：
```
✅ From(table)                    # 静态字符串
✅ WhereField(field, op, value)  # 支持常量、类型包装器 (int(x), string(y))
✅ WhereIn(field, values...)      # 支持展开字面量切片
✅ WhereLike(field, pattern)      # 支持常量
✅ OrderBy(expr)                  # 支持 "field ASC/DESC" 格式
✅ Limit(n) / Offset(n)
✅ 常量折叠：const a = "u" + "ser" → "user"
✅ 类型包装器：int(limit), string(name)
✅ 负数：-5
✅ 二元运算（仅 ADD）："na" + "me"
```

**缺失支持（P1-P2）**：
- 变量引用（`var table = getTableName()` — 无法静态解析）
- 条件表达式中的字段间接引用
- `GroupBy` 字段提取
- `Distinct` 标记
- 嵌套函数调用 `FOO(BAR(x))`
- 跨文件常量引用（当前仅限文件内）

### 2.4 索引诊断规则评审

| 规则 | 当前质量 | 评审意见 |
|------|----------|----------|
| IDX-001 | B | 仅检查首列出现在 WHERE 中，未验证复合索引列顺序与 WHERE 条件顺序的匹配度 |
| IDX-002 | C | **Bug：IsFunction 检测路径失效，仅依赖字符串包含 `(`**，存在误报/漏报 |
| IDX-003 | B | `isTypeCompatible` 过于宽松（如 `int` 和 `float` 不应被认为兼容） |
| IDX-004 | A | 完整实现，逻辑正确 |
| IDX-005 | F | **未实现** |
| IDX-006 | F | **未实现**（依赖 IDX-007 统计分析） |
| IDX-007 | F | **未实现** |
| IDX-008 | C | 仅检查 ORDER BY 首列是否有索引，未考虑 ASC/DESC 与索引方向一致性 |
| IDX-009 | C | 同 IDX-008 |
| IDX-010 | F | **未实现** |

---

## 三、后续规划（Phase 1-3）

### Phase 1：Bug 修复与 CLI 核心能力补全

**目标**：修复 P0 Bug，补齐 CLI 基本功能，达到生产可用状态

#### 1.1 IDX-002 Bug 修复

**问题**：AST 解析器未正确提取 `IsFunction` 和 `FuncName`

**修复方案**：
- 在 `buildQueryMetadata` 中，当检测到 `SelectorExpr` 模式（如 `LOWER(name)`）时，识别为函数调用
- 扩展 `Condition` 结构：

```go
// Condition 新增字段
IsFunctionCall bool   `json:"isFunctionCall"`
FunctionName   string `json:"functionName,omitempty"`
```

- 重构 `parseWhereField` 以识别 `ast.SelectorExpr`

**预期工时**：1-2 天

#### 1.2 实现 `--dsn` 实时拉取

**现状**：PostgreSQL provider 已实现，未集成

**实施步骤**：
1. 在 `main.go` 中引入 `dialect/postgres` 包
2. 根据 DSN 格式（`postgres://user:pass@host:5432/db`）创建 `*sql.DB` 连接
3. 调用 `postgres.NewMetadataProvider(db).Indexes(ctx, tableName)` 填充 Schema
4. 添加连接池配置（max_open_conns, max_idle_conns, conn_max_lifetime）

**预期工时**：1 天

#### 1.3 否定条件检测（IDX-005）

**需求分析**：
- 支持的操作符：`!=`、`<>`、`NOT IN`、`NOT LIKE`、`IS NOT NULL`
- 否定条件本身就是 Warning 级别（数据库通常无法利用索引处理否定条件）

**实现方案**：
1. 扩展 `parseWhereField` 以识别否定操作符
2. 在 `Condition` 中设置 `IsNegated = true`
3. 实现 `NegationRule`（IDX-005）

**预期工时**：1 天

#### 1.4 支持 `--exclude` 路径排除

**实现**：在 `walkGoFiles` 中添加跳过逻辑

```go
func walkGoFiles(root string, excludePatterns []string) ([]string, error)
```

**预期工时**：0.5 天

---

### Phase 2：规则完善与诊断精度提升

**目标**：补全 IDX-006/007/010，实现企业级诊断能力

#### 2.1 IDX-006 缺失索引检测

**核心问题**：如何判断"哪些字段应该建索引但没建"？

**实现思路**：
- 统计高频查询字段（从 QueryMetadata 聚合）
- 对高频字段（出现 >= 3 次）检查是否有索引覆盖
- 排除明显低选择率字段（如 `status` 只有 3 个值）

**数据结构变更**：
```go
// 在 Engine 中添加聚合层
type QueryAggregator struct {
    fieldFrequency map[string]int  // 字段 → 出现频率
    tableQueries   map[string]int  // 表名 → 查询次数
}
```

#### 2.2 IDX-007 冗余索引检测

**核心问题**：如何检测"已存在的索引是冗余的"？

**实现思路**：
- 索引 A 冗余于索引 B 当且仅当 A 的列是 B 的前缀，且 A/B 都非唯一
- 示例：`(a, b, c)` 冗余于 `(a, b)`；`(a)` 冗余于 `(a, b)` 如果无唯一约束

**前置条件**：需要 Column 类型信息（判断是否为主键/唯一索引）

#### 2.3 IDX-010 低选择性索引

**核心问题**：如何判断索引列的选择性？

**实现思路**：
- 若 `--dsn` 连接数据库，通过 `SELECT COUNT(DISTINCT col) / COUNT(*)` 计算选择性
- 若使用 `--schema`，标记为"无法检测"（低置信度 Suggestion）

**阈值**：
| 选择性 | 建议 |
|--------|------|
| < 0.01 | Critical — 索引无实际价值 |
| 0.01-0.1 | Warning — 考虑组合索引 |
| > 0.1 | OK |

#### 2.4 规则框架增强

| 改进项 | 说明 |
|--------|------|
| 规则注册表 | 实现 `rules.Register(rule Rule)` 自动注册，避免手动传入 |
| 规则禁用 | 支持配置文件或 CLI flag 禁用特定规则 |
| 规则优先级 | `Order() int` 接口，高优先级规则可早期短路 |

---

### Phase 3：CLI 工具链与企业级集成

**目标**：成为企业级开发标配工具，覆盖 CI/CD 和 IDE 生态

#### 3.1 多格式输出支持

```
--format json    # 默认，机器可读
--format sarif   # GitHub Security Lab 标准，CI 集成
--format text    # 人类可读，适合本地开发
--format html    # 可视化报告
```

#### 3.2 CI/CD 集成

**GitHub Actions 模板**：
```yaml
- name: Run gore-lint
  uses: gore-io/gore-lint-action@v1
  with:
    schema: ${{ secrets.DATABASE_URL }}
    rules: IDX-001,IDX-002,IDX-003,IDX-004
```

**GitLab CI 模板**：
```yaml
gore-lint:
  script:
    - gore-lint check ./... --dsn "$DATABASE_URL" --format sarif > gl-sarif.json
  artifacts:
    reports:
      sast: gl-sarif.json
```

#### 3.3 Schema 管理命令

```bash
# 导出数据库 Schema 到 JSON
gore-lint schema dump --dsn "postgres://..." -o schema.json

# 验证 Schema 语法
gore-lint schema validate schema.json

# 合并多 Schema 文件
gore-lint schema merge schema1.json schema2.json -o combined.json
```

#### 3.4 IDE 集成

- **VS Code 插件**：实时诊断，hover 显示规则建议
- **GoLand 插件**：IntelliJ 平台，集成 CodeLens
- **guru/guru-style 分析协议**：支持现有 Go 工具链

#### 3.5 高级静态分析能力

| 能力 | 实现方案 |
|------|----------|
| 跨文件常量解析 | 实现 `go/packages` 而非单文件 `go/parser` |
| 变量类型推断 | 引入 `go/types` 做类型检查 |
| 控制流分析 | 识别 if/else 分支中的查询差异 |
| 批量 Schema 缓存 | 支持 Schema Registry，团队共享 |

---

## 四、里程碑规划

| 阶段 | 目标 | 交付物 | 状态 |
|------|------|--------|------|
| **Phase 1.1** | Bug 修复 | IDX-002 修复、IDX-005 实现、--dsn 集成 | ✅ 已完成 |
| **Phase 1.2** | CLI 基础完善 | 路径排除、文本输出 | ✅ 已完成 |
| **Phase 2** | 规则补全 | IDX-006/007/010、规则框架增强 | ✅ 已完成 |
| **Phase 3** | 企业级集成 | SARIF 格式、CI 模板、IDE 插件设计 | ⏳ 待开始 |

---

## 五、架构升级建议

### 5.1 技术债务清理

| 问题 | 状态 | 说明 |
|------|------|------|
| 删除废弃包 | ✅ 已完成 | `internal/indexadvisor/` 已删除 |
| 统一 Advisor 入口 | ✅ 已完成 | 废弃包已删除，入口统一 |
| 扩展 Schema JSON | ⏳ 待开始 | 可选择性扩展 `tableComment` 等 |

### 5.2 性能优化

| 优化项 | 当前 | 目标 | 方案 |
|--------|------|------|------|
| 重复分析加速 | 每次重新解析 | Schema 缓存 + 增量分析 | 引入 build cache |
| 规则执行 | 串行 | 并行执行独立规则 | goroutine pool |
| 大型代码库 | 全量加载 | 模块化懒加载 | go/packages 分包 |
| go mod tidy | ⏳ 待优化 | 直接依赖清理 | 已添加 lib/pq |

### 5.3 可观测性

- **结构化日志**：引入 `slog` 替代 `fmt.Fprintf`
- **Tracing**：为每个 QueryMetadata 分析添加 trace span
- **健康检查**：`gore-lint doctor` 命令检查 DSN 连接、Schema 有效性

### 5.4 Phase 3 待办事项

| 功能 | 优先级 | 说明 |
|------|--------|------|
| SARIF 格式输出 | P1 | GitHub Security Lab 标准格式，CI 集成必需 |
| HTML 报告 | P2 | 可视化报告 |
| `gore-lint schema dump` | P1 | 导出数据库 Schema 到 JSON |
| `gore-lint schema validate` | P2 | 验证 Schema 语法 |
| `gore-lint schema merge` | P2 | 合并多 Schema 文件 |
| CI/CD 模板 | P2 | GitHub Actions / GitLab CI |
| go/packages 替代 go/parser | P2 | 支持跨文件常量解析 |
| 规则禁用配置 | P2 | 支持 `gore-lint.yaml` 配置文件 |

---

## 六、总结

gore ORM 的 Index Advisor 和 gore-lint CLI 经过第一轮优化，核心功能已基本完善：

### 已解决

| 问题 | 状态 |
|------|------|
| IDX-002 函数检测逻辑失效 | ✅ 已修复 |
| --dsn 实时拉取未集成 | ✅ 已集成 |
| IDX-005/006/007/010 未实现 | ✅ 全部实现 |
| 废弃包 internal/indexadvisor | ✅ 已删除 |
| 路径排除功能 | ✅ 已实现 |
| JSON/text 双格式输出 | ✅ 已实现 |

### 仍需关注

- **IDX-003 类型匹配逻辑偏宽松**（int/float 互认）
- **IDX-008/009 未校验索引方向一致性**
- **Phase 3 待办**：SARIF 格式、CI/CD 模板、Schema 管理命令

整体架构设计合理，静态分析方案比运行时诊断更符合"零侵入、零开销"的现代开发流程趋势，具备生产使用的基础条件。
