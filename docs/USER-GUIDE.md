# Nexus (gore ORM) 用户使用手册

## 项目概述

Nexus（gore ORM）是一个规范驱动的 Go ORM 实验项目，参考字节跳动 RFC 与腾讯 TDR 流程设计，强调问题导向、演进式设计与可度量指标。

**核心特性：**
- 强类型查询构建器（避免字符串硬编码）
- 变更追踪机制（Change Tracking + Unit of Work）
- SQL 索引诊断引擎（gore-lint CLI）
- PostgreSQL / MySQL / MongoDB 方言支持
- JOIN 支持（Inner/Left/Right/Full/Lateral）
- 事务管理（Transaction with rollback）
- 批量操作（AddBatch/AttachBatch/RemoveBatch）
- 结构化日志（Logger 接口 + slog 支持）
- 连接池管理（PooledExecutor）
- 标准化错误处理（GoreError with Error Codes）

---

## 目录

1. [快速开始](#快速开始)
2. [核心概念](#核心概念)
3. [API 用法](#api-用法)
4. [gore-lint CLI](#gore-lint-cli)
5. [配置与环境](#配置与环境)
6. [测试](#测试)
7. [常见问题](#常见问题)

---

## 快速开始

### 安装

```bash
cd gore
go test ./...
```

### 运行 gore-lint

```bash
# 使用离线 schema JSON
go run ./cmd/gore-lint check --schema /path/to/schema.json ./...

# 使用 DSN 实时拉取（开发中）
go run ./cmd/gore-lint check --dsn "postgres://user:pass@localhost:5432/db" ./...
```

---

## 核心概念

### 1. DbContext（工作单元上下文）

`DbContext` 是请求级别的容器，负责管理实体的变更追踪与提交。

```go
// 创建 Context
ctx := api.NewContext(exec, meta, dialector)

// 禁用变更追踪（提升性能）
noTrackCtx := ctx.AsNoTracking()
```

### 2. DbSet[T]（实体集合）

`DbSet[T]` 提供对类型 `T` 实体的强类型访问入口。

```go
set := api.Set[User](ctx)
```

### 3. Query[T]（查询构建器）

链式调用构建查询，支持 `Where`、`WhereField`、`WhereIn`、`WhereLike`、`OrderBy`、`Limit`、`Offset`、`GroupBy`、`Having`、`Join` 等。

#### JOIN 查询
```go
// INNER JOIN
q := api.Set[User](ctx).Query().
    From("users").
    Join("orders", "users.id = orders.user_id")

// LEFT JOIN
q := api.Set[User](ctx).Query().
    From("users").
    LeftJoin("orders", "users.id = orders.user_id")

// RIGHT JOIN / FULL JOIN
q := api.Set[User](ctx).Query().
    From("users").
    RightJoin("orders", "users.id = orders.user_id")

// 多个 JOIN
q := api.Set[User](ctx).Query().
    From("users").
    Join("orders", "users.id = orders.user_id").
    LeftJoin("products", "orders.product_id = products.id")
```

#### HAVING 子句
```go
q := api.Set[User](ctx).Query().
    From("users").
    GroupBy("status").
    Having("COUNT(*) > 10")
```

### 4. Change Tracker（变更追踪）

采用**快照对比法**（Snapshot Diffing）：
- `Attach` - 跟踪已有实体
- `Add` - 标记新增
- `Remove` - 标记删除
- `SaveChanges` - 检测并返回变更数量

### 5. 事务支持（Transaction）

支持显式事务管理，自动 rollback on error：

```go
err := ctx.Transaction(context.Background(), func(tx api.DbContext) error {
    // 在事务中执行操作
    api.Set[User](tx.(*api.Context)).Add(&User{Name: "Alice"})
    api.Set[Order](tx.(*api.Context)).Add(&Order{Amount: 100})
    return nil // 返回 nil 则提交
})
// err != nil 则自动回滚
```

### 6. 批量操作（Batch Operations）

```go
// 批量新增
users := []*User{
    {Name: "Alice"},
    {Name: "Bob"},
}
set.AddBatch(users)

// 批量 Attach
set.AttachBatch([]*User{{ID: 1}, {ID: 2}})

// 批量删除
set.RemoveBatch([]*User{{ID: 1}, {ID: 2}})
```

### 7. 日志框架（Logging）

```go
// 注入自定义日志器
ctx := ctx.WithLogger(&myLogger{})

// myLogger 实现 api.Logger 接口:
// Debug(msg string, args ...any)
// Info(msg string, args ...any)
// Warn(msg string, args ...any)
// Error(msg string, args ...any)
```

### 8. 连接池（Connection Pooling）

```go
import "gore/internal/executor"

// 创建带连接池的 Executor
pool := executor.NewPooledExecutor(db, &executor.PoolConfig{
    MaxOpenConns:    25,
    MaxIdleConns:    5,
    ConnMaxLifetime: 5 * time.Minute,
})

// 获取连接池统计
stats := pool.Stats()
```

抽象数据库差异，目前支持：
- PostgreSQL (`gore/dialect/postgres`)
- MySQL 8.0+ (`gore/dialect/mysql`) - 支持 Invisible Index、Descending Index
- MongoDB 7.0+ (`gore/dialect/mongodb`) - 支持索引管理、集合操作、CRUD

---

## API 用法

### 实体定义

```go
type User struct {
    ID   int
    Name string
}
```

### 创建 DbContext

```go
import (
    "gore/api"
    "gore/dialect/postgres"
    "gore/internal/executor"
)

// 使用真实 Executor
ctx := api.NewContext(yourExecutor, nil, &postgres.Dialector{})
```

### CRUD 操作

```go
// 获取实体集合
set := api.Set[User](ctx)

// 新增实体
user := &User{ID: 1, Name: "Alice"}
set.Add(user)

// Attach 已有实体（开始追踪）
set.Attach(user)

// 标记删除
set.Remove(user)

// 提交变更
count, err := ctx.SaveChanges(context.Background())
```

### 查询构建

```go
// 构建查询
q := set.Query().
    From("users").
    WhereField("name", "=", "Alice").
    WhereField("age", ">", 18).
    OrderBy("created_at DESC").
    Limit(10).
    Offset(20)

// IN 查询
q := set.Query().From("users").WhereIn("id", 1, 2, 3)

// LIKE 查询
q := set.Query().From("users").WhereLike("name", "%lice%")

// 生成 AST
ast := q.ToAST()
```

### 变更追踪控制

```go
// 禁用追踪（适合只读查询）
roCtx := ctx.AsNoTracking()

// 重新启用追踪
trackCtx := roCtx.AsTracking()
```

### 注入 Metrics

```go
type myMetrics struct{}

func (m *myMetrics) ObserveChangeTracking(duration time.Duration, entries int) {
    // 记录变更追踪耗时
}

func (m *myMetrics) ObserveSQL(operation string, duration time.Duration) {
    // 记录 SQL 执行耗时
}

ctx := ctx.WithMetrics(&myMetrics{})
```

---

## gore-lint CLI

gore-lint 是一个静态分析工具，用于诊断 SQL 索引相关问题。

### 命令行用法

```bash
gore-lint check [flags] <target>
```

### 参数说明

| 参数 | 说明 |
|------|------|
| `--dsn <dsn>` | 数据库 DSN，用于实时拉取 Schema |
| `--schema <file>` | 离线 Schema JSON 文件路径 |
| `--db-type postgres\|mysql\|mongodb` | 数据库类型 (默认 postgres) |
| `--stdout` | 输出到 stdout（默认 true） |
| `--format json\|text\|sarif` | 输出格式（默认 json） |

### MySQL Schema 支持

gore-lint 支持 MySQL 8.0+ 特性：
- Invisible Index (不可见索引)
- Descending Index (降序索引)
- InnoDB 存储引擎

### Schema JSON 格式

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

### 示例

```bash
# 使用离线 Schema 分析
go run ./cmd/gore-lint check --schema ./schema.json ./...

# 输出格式
go run ./cmd/gore-lint check --schema ./schema.json --format text ./...
```

### 诊断规则

| 规则 ID | 名称 | 说明 |
|---------|------|------|
| IDX-001 | Leftmost Match | 验证联合索引是否从第一列开始 |
| IDX-002 | Function Index | 检测函数操作导致的索引失效 |
| IDX-003 | Type Mismatch | 检测隐式类型转换 |
| IDX-004 | Like Prefix | 检测前缀通配 LIKE |
| IDX-005 | Negation | 检测否定条件（!=, NOT IN 等） |
| IDX-006 | Missing Index | 建议缺失索引 |
| IDX-007 | Redundant Index | 检测冗余索引 |
| IDX-008 | OrderBy Index | 检测排序字段索引 |
| IDX-009 | Join Index | 检测 JOIN 字段索引 |
| IDX-010 | Low Selectivity | 检测低选择性索引 |

**MySQL 特定规则:**

| 规则 ID | 名称 | 说明 |
|---------|------|------|
| IDX-MYSQL-001 | Invisible Index | 检测 MySQL 8.0+ 不可见索引 |
| IDX-MYSQL-002 | Descending Index | 验证 ORDER BY 方向与索引排序一致性 |
| IDX-MYSQL-003 | Hash Index Range | 检测 HASH 索引上的范围查询 |

---

## 配置与环境

### DSN 配置

通过环境变量 `GORE_DSN` 配置数据库连接：

```bash
export GORE_DSN="postgres://gore:gore123@localhost:5432/gore?sslmode=disable"
```

默认开发环境 DSN：
```
postgres://gore:gore123@localhost:5432/gore?sslmode=disable
```

### 环境判断

```go
import "gore/config"

if config.IsDevelopment() {
    // 开发环境逻辑
}

dsn := config.DSN() // 获取 DSN
```

---

## 测试

### 运行测试

```bash
cd gore
go test ./...
```

### 测试覆盖模块

- `gore/tests/dbcontext_test.go` - DbContext 单元测试
- `gore/tests/dbset_test.go` - DbSet 单元测试
- `gore/tests/tracker_test.go` - 变更追踪测试
- `gore/tests/bench_test.go` - 性能基准测试

### 基准测试

```bash
go test -bench=. -benchmem ./...
```

---

## 常见问题

### Q: 为什么叫 gore？

gore = Go + EF (Entity Framework)，旨在提供类似 EF Core 的开发体验。

### Q: 与 GORM/ent 的区别？

- **vs GORM**：减少反射使用，提升性能
- **vs ent**：更轻量，无需代码生成
- **特色**：内置 Index Advisor 静态分析

### Q: 变更追踪如何工作？

采用快照对比法（Snapshot Diffing）：
1. `Attach` 时存储实体快照
2. `SaveChanges` 时对比当前值与快照
3. 仅将变更字段加入 UPDATE

### Q: gore-lint 支持哪些数据库？

支持 PostgreSQL、MySQL 8.0+ 和 MongoDB 7.0+。MySQL 支持包括 Invisible Index 和 Descending Index。MongoDB 支持包括索引管理和集合操作。

### Q: 支持事务吗？

支持。通过 `ctx.Transaction()` 方法：
- 自动 BEGIN/COMMIT
- 错误时自动 ROLLBACK
- 支持嵌套事务

### Q: 如何使用连接池？

```go
import "gore/internal/executor"

pool := executor.NewPooledExecutor(db, nil) // 使用默认配置
ctx := api.NewContext(pool, nil, dialector)
```

---

## 项目结构

```
gore/
├── api/                    # 核心 API
│   ├── dbcontext.go       # DbContext 实现 + Transaction
│   ├── dbset.go           # DbSet 实现 + Batch Operations
│   ├── query_builder.go   # Query Builder + JOIN/Having
│   ├── logger.go          # Logger 接口
│   └── metrics.go          # Metrics 接口
├── dialect/               # 数据库方言
│   ├── dialector.go       # 方言接口 (JoinClause, QueryAST)
│   ├── postgres/          # PostgreSQL 方言实现
│   ├── mysql/             # MySQL 方言实现
│   └── mongodb/           # MongoDB 方言实现
├── internal/
│   ├── errors/            # 标准化错误处理 (GoreError)
│   ├── executor/          # SQL 执行器 + 连接池
│   │   └── pool.go       # PooledExecutor
│   ├── advisor/           # 索引诊断引擎
│   │   ├── advisor.go    # 核心结构
│   │   ├── engine.go     # 规则引擎
│   │   └── rules/        # 诊断规则
│   ├── tracker/           # 变更追踪
│   └── metadata/          # 元数据管理
├── cmd/gore-lint/         # CLI 工具
├── config/                # 配置管理
├── tests/                # 单元测试
└── testcode/              # 用户示例测试
```

---

## 下一步

- 查看 [SDD.md](./SDD.md) 了解完整设计规范
- 查看 [docs/rfc/](docs/rfc/) 了解各模块设计决策
- 查看 [docs/api/README.md](docs/api/README.md) 了解 API 详情