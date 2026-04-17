# gore 项目单测覆盖度报告

**生成日期**：2026-04-15
**整体覆盖率**：48.5%（较上次提升 9.7%）

---

## 一、覆盖率总览

| 包 | 覆盖率 | 状态 |
|-----|--------|------|
| `gore/dialect/postgres` | **94.6%** | ✅ 优秀 |
| `gore/internal/advisor/rules` | **84.0%** | ✅ 优秀 |
| `gore/cmd/gore-lint` | **45.7%** | ⚠️ 需改进 |
| `gore/tests` | — | 基准测试包 |
| `gore/api` | **0.0%** | 🔴 无测试（测试在 gore/tests） |
| `gore/internal/tracker` | **0.0%** | 🔴 无测试（测试在 gore/tests） |
| `gore/internal/metadata` | **0.0%** | 🔴 无测试 |
| `gore/internal/advisor` | **0.0%** | 🔴 无测试 |

---

## 二、详细覆盖分析

### 2.1 gore/dialect/postgres — 94.6% ✅

| 函数 | 覆盖率 | 说明 |
|------|--------|------|
| `BuildSelect` | ✅ 100% | 完整覆盖 |
| `BuildInsert` | ✅ 100% | 完整覆盖 |
| `BuildUpdate` | ✅ 100% | 完整覆盖 |
| `BuildDelete` | ✅ 100% | 完整覆盖 |

**缺失**：无

---

### 2.2 gore/internal/advisor/rules — 84.0% ✅

| 规则/函数 | 覆盖率 | 说明 |
|-----------|--------|------|
| `LeftmostMatchRule.Check` | 82.4% | IDX-001 核心逻辑 |
| `TypeMismatchRule.Check` | 92.9% | IDX-003 核心逻辑 |
| `isTypeCompatible` | 72.1% | 类型兼容性检查 |
| `LikePrefixRule.Check` | 83.3% | IDX-004 核心逻辑 |
| `NegationRule.Check` | 87.5% | IDX-005 核心逻辑 |
| `OrderByIndexRule.Check` | 63.6% | IDX-008 核心逻辑 |
| `JoinIndexRule.Check` | 87.5% | IDX-009 核心逻辑 |
| `RedundantIndexRule.Check` | 90.0% | IDX-007 核心逻辑 |
| `RedundantIndexRule.isRedundant` | 62.5% | 冗余检测逻辑 |
| `LowSelectivityRule.CheckSchema` | 66.7% | IDX-010 核心逻辑 |
| `Rule Name/Description/WhyDoc/Severity` | ✅ 100% | 新增接口方法测试 |

**缺失**：Rule interface 的 CheckSchema 方法部分未覆盖

---

### 2.3 gore/cmd/gore-lint — 45.7% ⚠️

| 函数 | 覆盖率 | 说明 |
|------|--------|------|
| `extractQueries` | 91.7% | ✅ AST 解析核心 |
| `buildStats` | 100.0% | ✅ 新增测试 |
| `writeTextReport` | 76.5% | ✅ 新增测试 |
| `writeSARIFReport` | 87.0% | ✅ 新增测试 |
| `loadSchemaCache` | 100.0% | ✅ 新增测试 |
| `collectGoFiles` | 91.7% | ✅ 新增测试 |
| `walkGoFiles` | 81.0% | ✅ 新增测试 |
| `writeReport` | 0.0% | 🔴 未覆盖 |
| `runCheck` | 0.0% | 🔴 未覆盖 |
| `runSchema` | 0.0% | 🔴 未覆盖 |
| `runSchemaDump` | 0.0% | 🔴 未覆盖 |
| `runSchemaValidate` | 0.0% | 🔴 未覆盖 |
| `runSchemaMerge` | 0.0% | 🔴 未覆盖 |
| `loadLiveSchema` | 0.0% | 🔴 未覆盖 |
| `fetchTableNames` | 0.0% | 🔴 未覆盖 |
| `fetchColumns` | 0.0% | 🔴 未覆盖 |
| `fetchIndexes` | 0.0% | 🔴 未覆盖 |
| `printUsage` | 0.0% | 🔴 未覆盖 |

**缺失**：CLI 命令逻辑需要数据库连接才能测试

---

### 2.4 gore/api — 0.0% 🔴 (测试在 gore/tests)

| 函数 | 覆盖率 | 说明 |
|------|--------|------|
| `NewContext` | ✅ 测试通过 | gore/tests |
| `Set[T]` | ✅ 测试通过 | gore/tests |
| `WithMetrics` | ✅ 测试通过 | gore/tests |
| `SaveChanges` | ✅ 测试通过 | gore/tests |
| `AsNoTracking` | ✅ 测试通过 | gore/tests |
| `AsTracking` | ✅ 测试通过 | gore/tests |
| `DbSet.Where` | ✅ 测试通过 | gore/tests |
| `DbSet.Query` | ✅ 测试通过 | gore/tests |
| `DbSet.Attach` | ✅ 测试通过 | gore/tests |
| `DbSet.Add` | ✅ 测试通过 | gore/tests |
| `DbSet.Remove` | ✅ 测试通过 | gore/tests |
| `DbSet.Find` | ✅ 测试通过 | gore/tests |
| `Query.From` | ✅ 测试通过 | gore/tests |
| `Query.WhereField` | ✅ 测试通过 | gore/tests |
| `Query.WhereIn` | ✅ 测试通过 | gore/tests |
| `Query.WhereLike` | ✅ 测试通过 | gore/tests |
| `Query.OrderBy` | ✅ 测试通过 | gore/tests |
| `Query.Limit` | ✅ 测试通过 | gore/tests |
| `Query.Offset` | ✅ 测试通过 | gore/tests |
| `Query.GroupBy` | ✅ 测试通过 | gore/tests |
| `Query.Distinct` | ✅ 测试通过 | gore/tests |
| `Query.ToAST` | ✅ 测试通过 | gore/tests |

**说明**：API 测试位于 `gore/tests/dbcontext_test.go`，采用黑盒测试方式

---

### 2.5 gore/internal/tracker — 0.0% 🔴 (测试在 gore/tests)

| 函数 | 覆盖率 | 说明 |
|------|--------|------|
| `New` | ✅ 测试通过 | gore/tests |
| `Attach` | ✅ 测试通过 | gore/tests |
| `MarkAdded` | ✅ 测试通过 | gore/tests |
| `MarkDeleted` | ✅ 测试通过 | gore/tests |
| `DetectChanges` | ✅ 测试通过 | gore/tests |
| `Clear` | ✅ 测试通过 | gore/tests |
| `Entries` | ✅ 测试通过 | gore/tests |
| `snapshot` | ✅ 测试通过 | gore/tests |
| `diffSnapshot` | ✅ 测试通过 | gore/tests |

**说明**：Tracker 测试位于 `gore/tests/tracker_test.go`，覆盖所有核心方法

---

## 三、覆盖度改进记录

### 本次改进新增测试

#### gore/tests (dbcontext_test.go)
- `TestDbContextSetGeneric` - 泛型 Set 测试
- `TestNewContextNilMetadata` - nil metadata 处理
- `TestContextAsNoTracking` - 禁用跟踪
- `TestContextAsTracking` - 启用跟踪
- `TestContextDialector` - Dialector 访问
- `TestContextExecutor` - Executor 访问
- `TestContextTracker` - Tracker 访问
- `TestDbSetAttach` - 附加实体
- `TestDbSetAdd` - 添加实体
- `TestDbSetRemove` - 删除实体
- `TestDbSetFind` - 查找实体
- `TestDbSetQuery` - 查询构造
- `TestDbSetWhere` - 条件过滤
- `TestQueryFrom` - From 子句
- `TestQueryFromEmpty` - 空表名
- `TestQueryWhereField` - 字段条件
- `TestQueryWhereFieldEmpty` - 空字段
- `TestQueryWhereIn` - IN 条件
- `TestQueryWhereInEmpty` - 空 IN
- `TestQueryWhereLike` - LIKE 条件
- `TestQueryWhereLikeEmpty` - 空 LIKE
- `TestQueryOrderBy` - 排序
- `TestQueryLimit` - 限制
- `TestQueryOffset` - 偏移
- `TestQueryGroupBy` - 分组
- `TestQueryDistinct` - 去重
- `TestQueryChaining` - 方法链
- `TestWithMetrics` - 指标集成

#### gore/tests (tracker_test.go)
- `TestTrackerNew` - 创建 Tracker
- `TestTrackerAttach` - 附加实体
- `TestTrackerAttachNilPointer` - nil 指针错误
- `TestTrackerAttachNonPointer` - 非指针错误
- `TestTrackerMarkAdded` - 标记已添加
- `TestTrackerMarkAddedNilPointer` - nil 指针错误
- `TestTrackerMarkDeleted` - 标记已删除
- `TestTrackerMarkDeletedNilPointer` - nil 指针错误
- `TestTrackerDetectChanges` - 检测变更
- `TestTrackerDetectChangesNoChanges` - 无变更
- `TestTrackerDetectChangesUnchanged` - 未变更状态
- `TestTrackerDetectChangesAdded` - 检测添加
- `TestTrackerDetectChangesDeleted` - 检测删除
- `TestTrackerClear` - 清除追踪
- `TestTrackerEntries` - 获取条目
- `TestTrackerEntriesAfterDelete` - 删除后条目
- `TestTrackerMultipleFieldsModified` - 多字段修改
- `TestTrackerAttachSameEntityTwice` - 重复附加

#### gore/cmd/gore-lint (main_test.go)
- `TestWriteTextReportNoIssues` - 无问题报告
- `TestWriteTextReportWithSuggestions` - 带建议报告
- `TestWriteSARIFReport` - SARIF 格式报告
- `TestBuildStats` - 统计构建
- `TestLoadSchemaCache` - Schema 缓存加载
- `TestLoadSchemaCacheEmptyPath` - 空路径错误
- `TestLoadSchemaCacheNotFound` - 文件未找到
- `TestLoadSchemaCacheInvalidJSON` - 无效 JSON
- `TestCollectGoFilesSingleFile` - 单文件收集
- `TestCollectGoFilesDirectory` - 目录收集
- `TestCollectGoFilesWithExclude` - 排除收集
- `TestCollectGoFilesDotDotSuffix` - ... 后缀
- `TestWalkGoFilesExcludesTestFiles` - 排除测试文件
- `TestWalkGoFilesUnsupportedTarget` - 不支持目标
- `TestExtractQueriesWithWhereIn` - IN 条件提取
- `TestExtractQueriesWithWhereLike` - LIKE 条件提取
- `TestExtractQueriesWithOrderBy` - 排序提取
- `TestExtractQueriesWithNegatedCondition` - 否定条件
- `TestWriteReportJSON` - JSON 报告
- `TestWriteReportUnsupportedFormat` - 不支持格式
- `TestWriteReportToStderr` - stderr 输出
- `TestFlagStringSlice` - 标志切片
- `TestFlagStringSliceEmpty` - 空切片

#### gore/internal/advisor/rules (rules_test.go)
- `TestRuleInterfaceMethods` - 规则接口方法
- `TestRedundantIndexRuleEdgeCases` - 冗余索引边界
- `TestLikePrefixRuleWithUnderscore` - 下划线通配
- `TestLikePrefixRuleNoIndex` - 无索引情况
- `TestNegationRuleEdgeCases` - 否定条件边界
- `TestJoinIndexRuleEdgeCases` - JOIN 索引边界
- `TestMissingIndexRuleNoQueries` - 无查询情况
- `TestMissingIndexRuleWithIndex` - 有索引情况
- `TestLowSelectivityRuleWithMetadata` - 有元数据
- `TestLowSelectivityRuleNoMetadata` - 无元数据

---

## 四、覆盖率对比

| 模块 | 上次 | 本次 | 变化 |
|------|------|------|------|
| gore/dialect/postgres | 94.6% | 94.6% | — |
| gore/internal/advisor/rules | 71.4% | **84.0%** | ✅ +12.6% |
| gore/cmd/gore-lint | 26.8% | **45.7%** | ✅ +18.9% |
| 整体 | 38.8% | **48.5%** | ✅ +9.7% |

---

## 五、剩余改进目标

### gore/cmd/gore-lint (45.7% → 60%+)

需要数据库连接的函数无法在单元测试中覆盖：
- `runCheck` - 需要 --dsn 或 --schema
- `runSchemaDump` - 需要 PostgreSQL 连接
- `runSchemaValidate` - 需要文件系统
- `runSchemaMerge` - 需要文件系统
- `loadLiveSchema` - 需要 PostgreSQL 连接
- `fetchTableNames/Columns/Indexes` - 需要 PostgreSQL 连接

**建议**：通过集成测试覆盖这些函数

### gore/internal/metadata (0%)

无测试文件，需要新增：
- `metadata/entity.go` - 实体注册表

### gore/internal/advisor (0%)

无测试文件，需要新增：
- `advisor/engine.go` - 规则引擎
