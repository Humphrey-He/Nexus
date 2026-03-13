# API 文档

本目录包含 gore ORM 的公开 API 规范与用法说明。文档遵循 SDD 规范驱动开发，优先描述接口契约、约束与可测指标。

## 目录

- DbContext / DbSet
- Query Builder
- Change Tracker
- Metrics
- gore-lint CLI

---

## DbContext / DbSet

### DbContext

- `SaveChanges(ctx context.Context) (int, error)`
  - 提交变更追踪结果（当前返回变更数量）。
  - 若禁用追踪，返回 `ErrTrackingDisabled`。

- `AsNoTracking() DbContext`
  - 禁用变更追踪。

- `AsTracking() DbContext`
  - 启用变更追踪。

- `WithMetrics(m Metrics) *Context`
  - 注入指标采集器（性能与可观测性）。

### DbSet[T]

- `Attach(entity *T) error`
  - 对已有实体启用变更追踪。

- `Add(entity *T) error`
  - 标记为新增实体。

- `Remove(entity *T) error`
  - 标记为删除实体。

- `Query() *Query[T]`
  - 返回查询构造器。

---

## Query Builder

### 核心方法

- `From(table string) *Query[T]`
- `WhereField(field, op string, value any) *Query[T]`
- `WhereIn(field string, values ...any) *Query[T]`
- `WhereLike(field string, pattern string) *Query[T]`
- `OrderBy(expr string) *Query[T]`
- `Limit(n int) *Query[T]`
- `Offset(n int) *Query[T]`

### 诊断说明

- `WhereField/WhereIn/WhereLike/From` 会被 gore-lint 静态分析器识别并生成 QueryMetadata。
- 建议在关键路径中优先使用上述方法，保证诊断覆盖率。

---

## Change Tracker

### 行为与限制

- 快照对比法（Snapshot Diffing）。
- `SaveChanges()` 内部执行 `DetectChanges()`，并返回变更数量。

---

## Metrics

```go
type Metrics interface {
    ObserveChangeTracking(duration time.Duration, entries int)
    ObserveSQL(operation string, duration time.Duration)
}
```

- 当前实现仅接入 `ObserveChangeTracking`。
- `ObserveSQL` 预留用于 SQL 执行链路。

---

## gore-lint CLI

```
Usage:
  gore-lint check [--dsn <dsn> | --schema <file>] [--stdout] <target>
```

- `--schema`: 读取离线 Schema JSON
- `--dsn`: 预留实时 Schema 拉取（未实现）

---

## Schema JSON 结构

```json
{
  "tables": [
    {
      "tableName": "users",
      "columns": [
        { "name": "id", "type": "int" },
        { "name": "name", "type": "text" }
      ],
      "indexes": [
        { "name": "users_id_idx", "columns": ["id"], "unique": true, "method": "btree", "isBtree": true }
      ]
    }
  ]
}
```