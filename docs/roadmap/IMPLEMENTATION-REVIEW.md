# gore 项目实现程度评审报告

**评审日期**：2026-04-15
**评审角色**：资深架构师 + DBA
**版本**：v0.1

---

## 一、项目全景完成度

| 模块 | 规划功能数 | 已实现 | 完成度 |
|------|-----------|--------|--------|
| CLI 命令 | 13 | 10 | **77%** |
| Index Advisor 规则 | 10 | 10 (含部分) | **90%** |
| API (DbContext/DbSet) | 16 | 15 | **94%** |
| Change Tracker | 5 | 3 | **60%** |
| 诊断模式 | 3 | 2 | **67%** |
| Phase 3 企业级集成 | 8 | 3 | **38%** |
| **总计** | **55** | **43** | **78%** |

---

## 二、Index Advisor 规则实现详情

### 2.1 规则完成度矩阵

| 规则 ID | 规则名称 | 文档描述 | 实际实现 | 完成度 | 质量评级 |
|---------|---------|----------|----------|--------|----------|
| IDX-001 | 最左匹配检查 | 验证联合索引是否使用了最左列 | `LeftmostMatchRule` | 90% | B |
| IDX-002 | 函数索引失效 | WHERE 条件中使用函数导致索引失效 | `FunctionIndexRule` + Bug 修复 | 95% | A |
| IDX-003 | 隐式类型转换 | 字段类型与参数类型不匹配导致索引失效 | `TypeMismatchRule` | 70% | C |
| IDX-004 | 前缀通配 LIKE | `LIKE '%xxx'` 无法使用索引 | `LikePrefixRule` | 95% | A |
| IDX-005 | 否定条件 | `!=`, `<>`, `NOT IN`, `IS NOT NULL` 导致索引失效 | `NegationRule` | 100% | A |
| IDX-006 | 索引缺失 | 高频查询字段缺少索引 | `MissingIndexRule` | 100% | B |
| IDX-007 | 冗余索引 | 存在包含关系的索引浪费存储 | `RedundantIndexRule` | 100% | B |
| IDX-008 | 排序字段索引 | ORDER BY 字段应建索引 | `OrderByIndexRule` | 75% | C |
| IDX-009 | JOIN 字段索引 | JOIN 条件字段应建索引 | `JoinIndexRule` | 75% | C |
| IDX-010 | 低选择性索引 | 索引列唯一值比例过低 | `LowSelectivityRule` | 100% | B |

**质量评级说明**：A=完整准确，B=基本正确有瑕疵，C=有缺陷需改进

### 2.2 规则质量缺陷详情

#### IDX-001 (90% - 质量 B)

**缺陷**：
- 仅检查首列是否出现在 WHERE 中
- 未验证复合索引列顺序与 WHERE 条件顺序的匹配度
- 未校验 ORDER BY 场景下的索引列覆盖

**示例**：
```sql
-- 复合索引 (a, b, c)
WHERE b = 1 AND c = 2  -- IDX-001 不会报警，但实际无法利用索引
```

**建议修复**：检查 WHERE 条件列顺序是否与索引列顺序匹配

#### IDX-003 (70% - 质量 C)

**缺陷**：
- `isTypeCompatible()` 逻辑过于宽松
- `int` 和 `float` 被认为类型兼容（实际可能无法利用索引）
```go
case "int", "int64":
    return strings.Contains(columnType, "int") || strings.Contains(columnType, "numeric")
case "float", "float64":
    return strings.Contains(columnType, "float") || strings.Contains(columnType, "numeric")
```

**影响**：PostgreSQL 中 `int` 和 `float` 混合查询可能导致隐式类型转换

**建议修复**：严格类型匹配，int/Int4/float/Float8 分开判断

#### IDX-008/009 (75% - 质量 C)

**缺陷**：
- 仅检查字段是否有索引覆盖
- 未考虑 ASC/DESC 与索引定义方向的一致性
```go
// 当前实现
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
```

**问题**：
1. 只检查首列，复合索引其他列被忽略
2. 未考虑排序方向：`ORDER BY col DESC` 与 `idx (col ASC)` 方向可能不一致

---

## 三、CLI 命令实现详情

### 3.1 命令矩阵

| 命令/标志 | 文档描述 | 实现状态 | 代码位置 |
|----------|----------|----------|----------|
| `gore-lint check` | 静态分析主命令 | ✅ 完整 | `main.go:runCheck()` |
| `--dsn <string>` | PostgreSQL 实时拉取 | ✅ 完整 | `main.go:loadLiveSchema()` |
| `--schema <file>` | 离线 Schema JSON | ✅ 完整 | `main.go:loadSchemaCache()` |
| `--format json` | JSON 报告 | ✅ 完整 | `main.go:writeReport()` |
| `--format text` | 人类可读报告 | ✅ 完整 | `main.go:writeTextReport()` |
| `--format sarif` | GitHub SARIF 格式 | ✅ 完整 | `main.go:writeSARIFReport()` |
| `--exclude <path>` | 路径排除 | ✅ 完整 | `main.go:walkGoFiles()` |
| `--stdout` | 输出到 stdout | ✅ 完整 | `main.go` |
| `gore-lint schema dump` | 导出 Schema | ✅ 完整 | `main.go:runSchemaDump()` |
| `gore-lint schema validate` | 验证 Schema | ✅ 完整 | `main.go:runSchemaValidate()` |
| `gore-lint schema merge` | 合并 Schema | ✅ 完整 | `main.go:runSchemaMerge()` |
| `--format html` | 可视化报告 | ❌ 未实现 | - |
| `gore-lint doctor` | 健康检查 | ❌ 未实现 | - |

### 3.2 AST 静态分析能力

| 能力 | 文档描述 | 实现状态 |
|------|----------|----------|
| `From(table)` | 静态字符串 | ✅ |
| `WhereField(field, op, value)` | 字段谓词 | ✅ |
| `WhereIn(field, values...)` | IN 谓词 | ✅ |
| `WhereLike(field, pattern)` | LIKE 谓词 | ✅ |
| `OrderBy(expr)` | 排序 | ✅ |
| `Limit(n) / Offset(n)` | 分页 | ✅ |
| 常量折叠 (`"na" + "me"`) | 二元运算拼接 | ✅ |
| 类型包装器 (`int(x)`) | 类型转换 | ✅ |
| 负数字面量 (`-5`) | 负数支持 | ✅ |
| 函数调用 (`LOWER(name)`) | 函数检测 | ✅ (Bug 已修复) |
| `!=`, `<>` 否定操作符 | 否定条件 | ✅ |
| 变量引用 | `var table = ...` | ❌ 无法静态解析 |
| 跨文件常量 | const 定义在别的文件 | ❌ 仅文件内 |
| `GroupBy` 字段 | 分组字段提取 | ❌ 未实现 |
| `Distinct` 标记 | 去重标记 | ❌ 未实现 |
| 嵌套函数调用 | `FOO(BAR(x))` | ❌ 未实现 |

---

## 四、API 实现详情

### 4.1 DbContext / DbSet API

| API | 文档描述 | 实现状态 | 代码位置 |
|-----|----------|----------|----------|
| `NewContext(exec, meta, dialector)` | 创建上下文 | ✅ | `api/dbcontext.go` |
| `Set[T any](ctx)` | 返回 DbSet[T] | ✅ | `api/dbcontext.go` |
| `SaveChanges(ctx)` | 提交变更追踪 | ✅ | `api/dbcontext.go:58` |
| `AsNoTracking()` | 禁用变更追踪 | ✅ | `api/dbcontext.go:77` |
| `AsTracking()` | 启用变更追踪 | ✅ | `api/dbcontext.go:84` |
| `WithMetrics(m)` | 注入指标采集器 | ✅ | `api/dbcontext.go:51` |
| `Attach(entity)` | 启用已有实体追踪 | ✅ | `api/dbset.go:23` |
| `Add(entity)` | 标记新增实体 | ✅ | `api/dbset.go:35` |
| `Remove(entity)` | 标记删除实体 | ✅ | `api/dbset.go:47` |
| `Query()` | 返回查询构造器 | ✅ | `api/dbset.go:18` |
| `Find(pk)` | 主键查询 | ⚠️ 骨架 | `api/dbset.go:59` |

### 4.2 Query Builder API

| API | 文档描述 | 实现状态 | 代码位置 |
|-----|----------|----------|----------|
| `From(table)` | 设置表名 | ✅ | `api/query_builder.go:32` |
| `WhereField(field, op, value)` | 字段谓词 | ✅ | `api/query_builder.go:40` |
| `WhereIn(field, values...)` | IN 谓词 | ✅ | `api/query_builder.go:53` |
| `WhereLike(field, pattern)` | LIKE 谓词 | ✅ | `api/query_builder.go:68` |
| `Where(predicate)` | 泛型谓词 | ✅ | `api/query_builder.go:24` |
| `OrderBy(expr)` | 排序 | ✅ | `api/query_builder.go:80` |
| `Limit(n)` | 限制行数 | ✅ | `api/query_builder.go:88` |
| `Offset(n)` | 跳过行数 | ✅ | `api/query_builder.go:94` |
| `ToAST()` | 构建 QueryAST | ✅ | `api/query_builder.go:100` |

### 4.3 Metrics API

| API | 文档描述 | 实现状态 | 说明 |
|-----|----------|----------|------|
| `ObserveChangeTracking(dur, n)` | 追踪耗时 | ✅ | 已实现 |
| `ObserveSQL(op, dur)` | SQL 执行耗时 | ❌ | 预留未实现 |

### 4.4 Change Tracker 实现

| 功能 | 文档规划 | 实现状态 |
|------|----------|----------|
| 快照对比法 (Snapshot Diffing) | Phase 1 | ✅ |
| Added 状态追踪 | Phase 1 | ✅ |
| Modified 状态追踪 | Phase 1 | ✅ |
| Deleted 状态追踪 | Phase 1 | ✅ |
| Hash Snapshot (优化) | Phase 2 | ❌ |
| AsNoTracking | Phase 2 | ✅ |
| Dirty Flag (可选) | Phase 3 | ❌ |

---

## 五、诊断模式实现详情

| 模式 | 文档描述 | 实现状态 | 说明 |
|------|----------|----------|------|
| **Mode A: Static Analyzer** | go/analysis 框架/Linter | ✅ | `gore-lint check` |
| **Mode B: Runtime Interceptor** | Debug 拦截器 | ❌ | 未实现 |
| **Mode C: Offline Schema Cache** | Schema JSON 缓存 | ✅ | `--schema` |

**Mode B 未实现说明**：Runtime Interceptor 需要在 `DbContext.Execute()` 前后拦截，需要完整的 SQL 执行链路。当前 executor 接口存在但 Build* 方法返回空，无法实际执行 SQL。

---

## 六、未完成功能清单

### 6.1 🔴 完全未实现 (0%)

| 功能 | 所属模块 | 优先级 | 估计工时 |
|--------|---------|--------|----------|
| `ObserveSQL` | Metrics | P2 | 0.5 天 |
| Mode B: Runtime Interceptor | Index Advisor | P3 | 3 天 |
| Hash Snapshot | Change Tracker | P3 | 2 天 |
| Dirty Flag | Change Tracker | P3 | 3 天 |
| `GroupBy` 字段提取 | AST 解析 | P2 | 1 天 |
| `Distinct` 标记 | AST 解析 | P2 | 0.5 天 |
| 嵌套函数调用 | AST 解析 | P2 | 1 天 |

### 6.2 🟡 部分实现 (50-90%)

| 功能 | 当前完成度 | 问题 | 建议修复 |
|------|-----------|------|----------|
| IDX-001 最左匹配 | 90% | 未校验列顺序匹配度 | 增加列顺序校验 |
| IDX-003 隐式类型转换 | 70% | int/float 兼容过宽 | 严格类型判断 |
| IDX-008 排序索引 | 75% | 未校验 ASC/DESC 方向 | 增加方向一致性检查 |
| IDX-009 JOIN 索引 | 75% | 未校验 ASC/DESC 方向 | 增加方向一致性检查 |

### 6.3 🟠 Phase 3 企业级集成 (P2)

| 功能 | 优先级 | 说明 |
|------|--------|------|
| HTML 报告 | P2 | 可视化报告 |
| CI/CD 模板 | P2 | GitHub Actions / GitLab CI |
| go/packages 跨文件解析 | P2 | 支持跨文件常量 |
| 规则禁用配置 | P2 | `gore-lint.yaml` |
| `gore-lint doctor` | P3 | 健康检查 |

---

## 七、下一步计划

### Phase 4.1: Bug 修复与质量提升 (预计 2 天)

| 序号 | 任务 | 优先级 | 说明 |
|------|------|--------|------|
| 1 | 修复 IDX-001 列顺序校验 | P1 | 复合索引列顺序应与 WHERE 条件顺序匹配 |
| 2 | 修复 IDX-003 类型兼容逻辑 | P1 | 严格区分 int/float，避免隐式转换 |
| 3 | 修复 IDX-008/009 ASC/DESC 方向 | P2 | ORDER BY 方向与索引定义一致性 |
| 4 | 完善 `Find(pk)` 骨架 | P2 | 实现主键查询功能 |

### Phase 4.2: 功能补全 (预计 3 天)

| 序号 | 任务 | 优先级 | 说明 |
|------|------|--------|------|
| 1 | 实现 `ObserveSQL` | P2 | SQL 执行耗时统计 |
| 2 | 实现 `GroupBy` AST 提取 | P2 | 支持分组字段分析 |
| 3 | 实现 `Distinct` 标记 | P2 | 支持去重分析 |
| 4 | 实现嵌套函数调用检测 | P2 | `FOO(BAR(x))` 模式 |

### Phase 4.3: 企业级集成 (预计 5 天)

| 序号 | 任务 | 优先级 | 说明 |
|------|------|--------|------|
| 1 | HTML 报告生成 | P2 | 可视化报告输出 |
| 2 | GitHub Actions 模板 | P2 | `.github/workflows/gore-lint.yml` |
| 3 | GitLab CI 模板 | P2 | `.gitlab-ci.yml` |
| 4 | go/packages 跨文件解析 | P2 | 替代 go/parser |
| 5 | 规则禁用配置 | P2 | `gore-lint.yaml` |
| 6 | `gore-lint doctor` | P3 | DSN 连接检查、Schema 有效性 |

### Phase 4.4: 高级特性 (预计 7 天)

| 序号 | 任务 | 优先级 | 说明 |
|------|------|--------|------|
| 1 | Hash Snapshot | P3 | 优化大对象内存占用 |
| 2 | Mode B: Runtime Interceptor | P3 | Debug 拦截器，动态 SQL 分析 |
| 3 | Dirty Flag 可选追踪 | P3 | 低内存开销追踪方案 |
| 4 | VS Code 插件 | P4 | IDE 实时诊断 |
| 5 | GoLand 插件 | P4 | IntelliJ 平台支持 |

---

## 八、里程碑规划

| 阶段 | 目标 | 交付物 | 预计工时 | 状态 |
|------|------|--------|----------|------|
| Phase 1-3 | 核心功能 | CLI + 规则引擎 | - | ✅ 已完成 |
| **Phase 4.1** | Bug 修复 | 质量提升 | 2 天 | ⏳ 待开始 |
| **Phase 4.2** | 功能补全 | 观察指标 + AST | 3 天 | ⏳ 待开始 |
| **Phase 4.3** | 企业级集成 | CI/CD + HTML | 5 天 | ⏳ 待开始 |
| **Phase 4.4** | 高级特性 | 拦截器 + IDE | 7 天 | ⏳ 待开始 |

---

## 九、版本路线图

### v0.2 (Phase 4.1-4.2 完成后)
- 所有 P1/P2 Bug 修复
- `ObserveSQL` 指标接入
- `GroupBy`/`Distinct` AST 支持
- 质量评级提升：A 类规则 8 个

### v0.3 (Phase 4.3 完成后)
- HTML 报告
- CI/CD 集成模板
- go/packages 跨文件解析
- 规则配置化

### v1.0 (Phase 4.4 完成后)
- 生产可用状态
- IDE 插件
- Runtime Interceptor (可选)
- Hash Snapshot 优化

---

## 十、附录

### A. 代码统计

| 文件 | 行数 | 说明 |
|------|------|------|
| `cmd/gore-lint/main.go` | 1253 | CLI 主入口 |
| `internal/tracker/tracker.go` | 173 | 变更追踪 |
| `internal/advisor/advisor.go` | 93 | 规则引擎核心 |
| `api/query_builder.go` | 111 | 查询构建器 |
| `api/dbcontext.go` | 108 | DbContext |
| `rules/*.go` (合计) | ~430 | 10 条规则 |

**总代码量**：~3,400 行（核心功能约 1,800 行）

### B. 测试覆盖

| 测试文件 | 覆盖内容 |
|----------|----------|
| `cmd/gore-lint/main_test.go` | CLI AST 解析 |
| `internal/advisor/rules/rules_test.go` | 规则逻辑 |
| `dialect/postgres/metadata_provider_test.go` | PostgreSQL 元数据 |
| `tests/dbcontext_test.go` | DbContext 基本功能 |
| `tests/tracker_test.go` | Change Tracker |
| `tests/dbset_test.go` | DbSet 基本功能 |

### C. 依赖

```
github.com/lib/pq v1.12.3      # PostgreSQL 驱动
github.com/DATA-DOG/go-sqlmock # 测试 mock
```
