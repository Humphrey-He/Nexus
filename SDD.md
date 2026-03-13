# 软件概要设计文档 (SDD)

**项目名称**：Go-EF (Gore) ORM Framework

**文档版本**：v0.1-draft

**状态**：In Review

**负责人**：[Your Name]

**编写依据**：参考字节跳动 RFC 流程与腾讯 TDR 规范，强调问题导向、演进式设计、可度量技术指标。

---

## 目录

1. 业务背景与动机 (Background & Motivation)
2. 目标与范围 (Goals & Scope)
3. 术语与缩写 (Terms & Acronyms)
4. 需求与约束 (Requirements & Constraints)
5. 方案概览 (High-level Design)
6. 详细设计 (Detailed Design)
7. 接口与数据结构 (Interfaces & Data Structures)
8. 安全与合规 (Security & Compliance)
9. 性能与可观测性 (Performance & Observability)
10. 兼容性与迁移 (Compatibility & Migration)
11. 风险评估与对策 (Risk Assessment)
12. 测试与验收 (Testing & Acceptance)
13. 里程碑与执行计划 (Milestones)
14. 决策记录 (Decision Log)
15. 参考与附录 (References & Appendix)

---

## 1. 业务背景与动机 (Background & Motivation)

### 1.1 现状与痛点

- **GORM**：底层大量使用 `reflect.Value`，高并发扫描大数据量时 CPU 逃逸分析负担重，且 `interface{}` 的广泛使用导致编译期无法捕获字段名错误。
- **ent**：类型安全但 schema 定义过于繁琐（DSL 模式），生成代码量过大，影响大型项目的编译速度与代码整洁度。
- **EF (C#) 体验缺失**：Go 生态缺乏能像 EF 那样优雅处理 **Change Tracking** 和 **Unit of Work** 的库。

### 1.2 业务动机

- 让 Go 项目具备与 EF 接近的开发体验与工程化质量。
- 在保持类型安全的前提下，降低反射与内存分配成本。
- 引入“开发期诊断”能力，以减少线上性能问题与索引缺失风险。

---

## 2. 目标与范围 (Goals & Scope)

### 2.1 目标 (Objectives)

- **强类型查询**：支持 `db.Set[User]().Where(...)` 链式调用，避免字符串硬编码。
- **轻量化**：核心链路反射开销比 GORM 降低 50% 以上。
- **工程化增强**：内置 SQL 语义分析器，在开发阶段诊断索引缺失。
- **可演进性**：架构支持渐进式能力扩展与灰度启用。

### 2.2 非目标 (Non-goals)

- 不在 v0.1 阶段实现跨数据库事务分布式能力。
- 不在 v0.1 阶段实现完整 LINQ 语法糖（仅覆盖核心查询子集）。

### 2.3 范围 (Scope)

- 支持 PostgreSQL 与 MySQL（首批 Dialector）。
- 提供基础 CRUD、Where、OrderBy、Limit/Offset。
- 提供 Change Tracking 与 Unit of Work 的基础实现。

---

## 3. 术语与缩写 (Terms & Acronyms)

- **DbContext**：一次请求级别的上下文容器，负责跟踪与提交。
- **DbSet[T]**：强类型实体集合入口。
- **Change Tracking**：变更追踪机制，支持 Modified/Added/Deleted。
- **Unit of Work**：工作单元，统一提交事务。
- **Dialector**：数据库方言适配接口。

---

## 4. 需求与约束 (Requirements & Constraints)

### 4.1 功能性需求

- 支持强类型查询构建器。
- 支持实体变更追踪与批量提交。
- 提供 SQL AST 生成与参数化执行。
- 支持元数据缓存与字段映射。

### 4.2 非功能性需求

- **性能**：热点路径尽量实现零分配。
- **可测性**：核心模块必须可独立测试。
- **可维护性**：模块化设计，确保替换成本低。

### 4.3 约束

- Go 运行时无法像 C# 获取完整 MemberInfo。
- 需兼容现有 `database/sql` 生态。

---

## 5. 方案概览 (High-level Design)

### 5.1 逻辑分层

| 模块 | 职责 |
| --- | --- |
| **API Layer (DbSet)** | 提供泛型入口，封装 CRUD 语义，暴露类似 EF 的 API。 |
| **Change Tracker** | 维护实体快照，记录 `Modified/Added/Deleted` 状态。 |
| **SQL Builder** | 将泛型表达式转换为参数化 SQL，支持 AST 预编译。 |
| **Metadata Engine** | 解析 Struct Tag 与 DB Schema，维护字段映射缓存。 |
| **Executor** | 封装 `database/sql`，处理连接池与结果集映射（Fast-path）。 |

### 5.2 数据流

1. API Layer 创建 `Query[T]`。
2. Query 交由 SQL Builder 生成 AST。
3. AST 交由 Dialector 转换为 SQL + 参数。
4. Executor 执行 SQL，Metadata Engine 负责映射。
5. Change Tracker 记录快照并在 `SaveChanges()` 时生成 Patch。

---

## 6. 详细设计 (Detailed Design)

### 6.1 泛型查询器 (Generic Query Builder)

**设计思路**：利用泛型约束 `T any`，结合代码生成的字段常量，避免字符串硬编码。

**实现规范**：

- 通过 `Query[T]` 结构体持有 `builder`。
- 使用 `sync.Pool` 复用 SQL 拼接缓冲区，减少内存分配。
- 查询构建器输出 AST，避免直接拼接字符串。

**性能约束**：

- 单次 Where 构建不超过 2 次分配。
- Query 结构体禁止包含 `interface{}` 类型字段。

### 6.2 变更追踪机制 (Change Tracking Strategy)

**方案选择**：快照对比法 (Snapshot Diffing)。

1. **Attach**：对象从 DB 载入后，在 Session 中存储原始值 Hash 或深拷贝。
2. **Detect**：`SaveChanges()` 时遍历 Session 内对象，对比当前值与原始值。
3. **Patch**：仅将变更字段加入 `UPDATE` 的 `SET` 子句。

**可演进项**：

- v0.2 支持属性级别的 Dirty Flag 追踪。

### 6.3 索引诊断引擎 (Index Advisor)

**集成方式**：作为 `go vet` 插件或独立 Linter。

**逻辑规范**：

- **Live Fetch**：启动时异步拉取 `pg_catalog.pg_index` 数据。
- **Matching**：解析 `Where` 条件，判断是否符合索引最左前缀或存在函数操作（导致索引失效）。

**输出格式**：

- 统一输出到 `stdout`，可被 CI 收集。
- 提供机器可读 JSON 结构（供 IDE/CI 消费）。

---

## 7. 接口与数据结构 (Interfaces & Data Structures)

### 7.1 Dialector 接口

```go
// Dialector defines database dialect behaviors.
type Dialector interface {
    Name() string
    BuildSelect(ast *QueryAST) (string, []any, error)
    BuildInsert(ast *InsertAST) (string, []any, error)
    BuildUpdate(ast *UpdateAST) (string, []any, error)
    BuildDelete(ast *DeleteAST) (string, []any, error)
}
```

### 7.2 DbContext 基础接口

```go
// DbContext manages Unit of Work lifecycle.
type DbContext interface {
    Set[T any]() DbSet[T]
    SaveChanges() (int, error)
    AsNoTracking() DbContext
}
```

---

## 8. 安全与合规 (Security & Compliance)

- SQL Builder 必须确保全部参数化，禁止字符串拼接注入风险。
- Metadata Engine 禁止反射调用不安全字段或私有字段。
- 所有错误输出需脱敏，禁止打印明文连接串。

---

## 9. 性能与可观测性 (Performance & Observability)

### 9.1 性能指标

| 指标项 | 目标值 | 验证手段 |
| --- | --- | --- |
| **单行查询开销** | < 1000 ns/op (不含网络) | `go test -bench` |
| **内存分配** | < 10 allocs/op (Simple Select) | `benchmem` |
| **诊断准确率** | 索引缺失识别率 > 95% | 构造 50 种正负用例测试 |

### 9.2 可观测性要求

- 提供 `TraceID` 贯穿 Query 构建与执行链路。
- 暴露执行耗时、SQL 指标、变更提交耗时等 Metrics。

---

## 10. 兼容性与迁移 (Compatibility & Migration)

- v0.1 支持 PostgreSQL / MySQL。
- 设计保留扩展点，后续可支持 SQL Server / ClickHouse。
- 迁移成本：现有 GORM/ent 项目可通过转换层逐步替换。

---

## 11. 风险评估与对策 (Risk Assessment)

- **风险**：Go 泛型无法像 C# 一样运行时获取字段信息。
  - **对策**：轻量级 codegen 生成字段映射表，仅生成元数据。

- **风险**：快照对比在大对象下有内存压力。
  - **对策**：提供 `AsNoTracking()`，支持按需启用 Change Tracking。

- **风险**：SQL AST 转换导致性能开销。
  - **对策**：提供 AST 预编译与缓存机制。

---

## 12. 测试与验收 (Testing & Acceptance)

### 12.1 测试策略

- **单元测试**：针对 Query Builder / Change Tracker / Dialector。
- **集成测试**：数据库真实执行链路验证。
- **性能测试**：基准测试覆盖核心 CRUD。

### 12.2 验收标准

- 核心功能覆盖率 > 80%。
- 单行查询开销 < 1000 ns/op。
- 索引诊断准确率 > 95%。

---

## 13. 里程碑与执行计划 (Milestones)

| 阶段 | 里程碑 | 交付物 |
| --- | --- | --- |
| **M1** | 原型完成 | `DbContext` + `DbSet` + 简单查询 |
| **M2** | 变更追踪 | Change Tracker 可用 |
| **M3** | SQL Builder | AST + Dialector |
| **M4** | Index Advisor | Linter 原型 |
| **M5** | 性能基准 | Bench 指标达标 |

---

## 14. 决策记录 (Decision Log)

| 日期 | 决策 | 理由 |
| --- | --- | --- |
| 2026-03-13 | 采用 Snapshot Diffing | 实现简单、易于逐步优化 |

---

## 15. 参考与附录 (References & Appendix)

- ByteDance RFC 流程（内部规范参考）
- Tencent TDR 规范（内部规范参考）
- Go `database/sql` 文档

---

### 下一步行动 (Next Steps)

1. 定义 `DbContext` 接口原型，并实现首个 `Find[T]` 方法。
2. 实现 `Change Tracker` 的最简原型并加入单元测试。
