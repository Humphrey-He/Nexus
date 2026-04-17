# MySQL 功能接入 - 开发需求清单

**文档版本**：v0.2
**状态**：✅ 已完成（待 MySQL 8.0 真实验证）
**负责人**：[已完成]
**创建日期**：2026-04-17
**更新日期**：2026-04-17

---

## 一、背景与目标

### 1.1 现状

当前项目 `gore ORM` 已实现 PostgreSQL 方言支持：
- `gore/dialect/postgres/` - PostgreSQL SQL 构建器
- `gore/dialect/postgres/metadata_provider.go` - PostgreSQL 元数据拉取
- `gore/internal/advisor/rules/` - 索引诊断规则（基于 PostgreSQL）

### 1.2 目标

接入 MySQL 8.0 方言支持，包括：
- MySQL Dialector 实现
- MySQL 元数据拉取
- gore-lint MySQL 支持
- MySQL 特有索引诊断规则

### 1.3 MySQL 8.0 新特性范围

| 类别 | 特性 | 说明 |
|------|------|------|
| **窗口函数** | RANK/DENSE_RANK/ROW_NUMBER | 排名窗口函数 |
| | LAG/LEAD | 前后行访问 |
| | FIRST_VALUE/LAST_VALUE | 首尾值 |
| | SUM/AVG/COUNT OVER | 聚合窗口化 |
| **CTE** | WITH 子句 | 公共表表达式 |
| | 递归 CTE | 树形结构查询 |
| **JSON** | JSON_TABLE | JSON 转表 |
| | JSON 函数增强 | JSON_ARRAY, JSON_OBJECT 等 |
| **索引** | Invisible Index | 不可见索引（灰度发布） |
| | Descending Index | 降序索引 |
| **其他** | Hash Join | 哈希连接 |
| | 窗口函数增强 | 滑动窗口 |

---

## 二、功能需求

### 2.1 MySQL Dialector 实现

**目标**：实现 `gore/dialect/mysql/mysql.go`

#### 2.1.1 Dialector 接口实现

```go
package mysql

type Dialector struct{}

func (d *Dialector) Name() string { return "mysql" }

func (d *Dialector) BuildSelect(ast *dialect.QueryAST) (string, []any, error)
func (d *Dialector) BuildInsert(ast *dialect.InsertAST) (string, []any, error)
func (d *Dialector) BuildUpdate(ast *dialect.UpdateAST) (string, []any, error)
func (d *Dialector) BuildDelete(ast *dialect.DeleteAST) (string, []any, error)
```

#### 2.1.2 MySQL vs PostgreSQL 语法差异

| 功能 | PostgreSQL | MySQL | 说明 |
|------|------------|-------|------|
| LIMIT 语法 | `LIMIT n OFFSET m` | `LIMIT m, n` | MySQL offset 在前 |
| 自增主键 | `SERIAL` | `AUTO_INCREMENT` | |
| 字符串引号 | `"` 双引号 | `` ` `` 反引号 | MySQL 反引号 |
| 字符串连接 | `\|\|` | `CONCAT()` | |
| 占位符 | `$1, $2, ...` | `?` | MySQL 仅用 `?` |
| UPSERT | `ON CONFLICT DO UPDATE` | `INSERT ... ON DUPLICATE KEY UPDATE` | |
| NULL 判断 | `IS NULL` | `IS NULL` | 相同 |
| 分页优化 | OFFSET 优化 | 书签式分页 | MySQL 推荐书签式 |

#### 2.1.3 任务清单

| 任务 | 优先级 | 工作内容 | 状态 |
|------|--------|----------|------|
| 创建 `gore/dialect/mysql/` 目录 | P0 | 创建包结构 | ✅ 已完成 |
| 实现 `Name()` 方法 | P0 | 返回 "mysql" | ✅ 已完成 |
| 实现 `BuildSelect()` | P0 | MySQL 语法 LIMIT offset,count | ✅ 已完成 |
| 实现 `BuildInsert()` | P0 | INSERT INTO VALUES | ✅ 已完成 |
| 实现 `BuildUpdate()` | P0 | UPDATE SET WHERE | ✅ 已完成 |
| 实现 `BuildDelete()` | P0 | DELETE FROM WHERE | ✅ 已完成 |
| 添加单元测试 | P1 | `mysql_test.go` | ✅ 已完成 |

### 2.2 MySQL 元数据拉取

**目标**：实现 `gore/dialect/mysql/metadata_provider.go`

#### 2.2.1 Provider 接口实现

```go
type MetadataProvider struct {
    db *sql.DB
}

func (p *MetadataProvider) Indexes(ctx context.Context, table string) ([]metadata.IndexInfo, error)
```

#### 2.2.2 MySQL 系统表查询

```sql
-- 索引信息（MySQL 8.0+）
SELECT
    INDEX_NAME,
    TABLE_NAME,
    NON_UNIQUE,
    INDEX_TYPE,
    COLUMN_NAME,
    SEQ_IN_INDEX,
    COLLATION,
    IS_VISIBLE
FROM INFORMATION_SCHEMA.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = ?
ORDER BY INDEX_NAME, SEQ_IN_INDEX;

-- 索引方法映射
BTREE -> "btree"
HASH  -> "hash"
```

#### 2.2.3 MySQL 特有元数据

| 字段 | MySQL | PostgreSQL | 说明 |
|------|-------|------------|------|
| 索引方法 | INDEX_TYPE (BTREE/HASH) | amname | MySQL 支持 HASH |
| 可见性 | IS_VISIBLE | 无 | MySQL 8.0+ 支持 |
| 顺序 | COLLATION (A/D) | 无 | 降序索引 |

#### 2.2.4 任务清单

| 任务 | 优先级 | 工作内容 | 状态 |
|------|--------|----------|------|
| 实现 `NewMetadataProvider()` | P0 | 创建 Provider 实例 | ✅ 已完成 |
| 实现 `Indexes()` 查询 | P0 | 查询 INFORMATION_SCHEMA | ✅ 已完成 |
| 实现 `Columns()` 查询 | P1 | 查询列信息（可选） | ✅ 已完成 |
| 实现 `Tables()` 查询 | P2 | 查询表信息（可选） | ✅ 已完成 |
| 添加单元测试 | P1 | 使用 stub DB 测试 | ⏳ 待完成 |

### 2.3 gore-lint MySQL 支持

**目标**：扩展 gore-lint 支持 MySQL DSN

#### 2.3.1 DSN 格式差异

| 数据库 | DSN 格式 |
|--------|----------|
| PostgreSQL | `postgres://user:pass@host:5432/db` |
| MySQL | `mysql://user:pass@host:3306/db` |

#### 2.3.2 任务清单

| 任务 | 优先级 | 工作内容 | 状态 |
|------|--------|----------|------|
| 添加 `--db-type` 参数 | P0 | 指定 mysql/postgres | ✅ 已完成 |
| 修改 `runCheck()` 区分数据库类型 | P0 | 根据类型选择 Provider | ✅ 已完成 |
| 更新帮助文档 | P1 | 说明 MySQL 支持 |

### 2.4 MySQL 特有索引诊断规则

**目标**：新增 MySQL 特有诊断规则

#### 2.4.1 Invisible Index 检测

**规则 ID**：IDX-MYSQL-001

**背景**：MySQL 8.0 支持 `INVISIBLE` 索引，用于灰度发布测试

```sql
CREATE INDEX idx_name ON table(col) INVISIBLE;
```

**诊断逻辑**：
- 查询 `INFORMATION_SCHEMA.STATISTICS` 中 `IS_VISIBLE = 'NO'`
- 检查是否有查询使用该索引字段
- 若查询未使用，提示可删除以减少维护负担

**严重度**：Info

#### 2.4.2 Descending Index 检测

**规则 ID**：IDX-MYSQL-002

**背景**：MySQL 8.0 支持 `DESC` 索引排序

```sql
CREATE INDEX idx_name ON table(col DESC);
```

**诊断逻辑**：
- 检查 `COLLATION = 'D'` 的索引
- 验证 `ORDER BY col DESC` 是否与索引方向匹配
- 与 IDX-008 规则协同

**严重度**：Warn

#### 2.4.3 Hash Index 提示

**规则 ID**：IDX-MYSQL-003

**背景**：MySQL MEMORY/InnoDB HASH 索引仅支持等值查询

**诊断逻辑**：
- 检测 HASH 索引上的范围查询（>, <, LIKE 前缀）
- 提示 HASH 索引无法用于范围扫描

**严重度**：Warn

#### 2.4.4 任务清单

| 任务 | 优先级 | 工作内容 | 状态 |
|------|--------|----------|------|
| 实现 `InvisibleIndexRule` | P1 | IDX-MYSQL-001 | ✅ 已完成 |
| 实现 `DescendingIndexRule` | P2 | IDX-MYSQL-002 | ✅ 已完成 |
| 实现 `HashIndexRangeRule` | P2 | IDX-MYSQL-003 | ✅ 已完成 |
| 添加 MySQL 特有字段到 TableSchema | P0 | IS_VISIBLE, COLLATION | ✅ 已完成 |

---

## 三、非功能需求

### 3.1 性能要求

| 指标 | 目标 | 说明 |
|------|------|------|
| 元数据拉取 | < 100ms/表 | INFORMATION_SCHEMA 查询优化 |
| 内存分配 | 与 PostgreSQL 持平 | 避免额外分配 |

### 3.2 兼容性要求

- 支持 MySQL 8.0+
- 兼容 MySQL 5.7（部分特性不可用）
- DSN 格式遵循 `mysql://` 前缀

### 3.3 可扩展性

- Dialector 接口保持稳定
- 新增 MySQL 特有功能通过扩展实现
- 规则引擎支持数据库特有规则

---

## 四、技术方案

### 4.1 目录结构

```
gore/
└── dialect/
    ├── dialector.go              # 现有接口
    ├── postgres/
    │   ├── postgres.go
    │   ├── metadata_provider.go
    │   └── postgres_test.go
    └── mysql/
        ├── mysql.go              # 新增
        ├── metadata_provider.go   # 新增
        └── mysql_test.go          # 新增
```

### 4.2 关键接口变更

#### 4.2.1 TableSchema 扩展

```go
// advisor.TableSchema 新增字段
type TableSchema struct {
    TableName string
    Columns   []ColumnInfo
    Indexes   []IndexInfo

    // MySQL 特有
    IsVisible bool // 默认 true，MySQL 8.0+
}

// IndexInfo 新增字段
type IndexInfo struct {
    Name     string
    Columns  []string
    Unique   bool
    Method   string // "btree" | "hash"
    IsBTree  bool

    // MySQL 8.0+ 特有
    IsVisible  bool  // 默认 true
    Collation  string // "A" | "D"
}
```

### 4.3 依赖

```go
import _ "github.com/go-sql-driver/mysql"
```

---

## 五、测试策略

### 5.1 单元测试

| 模块 | 测试内容 |
|------|----------|
| `mysql.go` | 各 Build* 方法输出格式 |
| `metadata_provider.go` | Mock *sql.DB 测试 |

### 5.2 集成测试

| 测试项 | 说明 |
|--------|------|
| MySQL 8.0 连接 | 真实数据库连接 |
| 元数据拉取 | 验证与 PostgreSQL 差异 |
| gore-lint check | MySQL DSN 模式 |

### 5.3 测试数据

```sql
-- 表结构
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100),
    email VARCHAR(255),
    status TINYINT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_name (name),
    INDEX idx_status (status) INVISIBLE,
    INDEX idx_name_desc (name DESC)
);

-- 测试查询
SELECT * FROM users WHERE name = 'Alice';
SELECT * FROM users WHERE status > 0 ORDER BY name DESC;
```

---

## 六、风险与对策

| 风险 | 影响 | 对策 |
|------|------|------|
| MySQL HASH 索引与 PostgreSQL 差异 | 规则引擎误报 | 条件判断数据库类型 |
| INFORMATION_SCHEMA 查询性能 | 大库延迟 | 添加缓存机制 |
| MySQL 5.7 不支持某些特性 | 兼容性问题 | 版本检测，降级处理 |

---

## 七、里程碑

| 阶段 | 目标 | 交付物 | 状态 |
|------|------|--------|------|
| **M1** | MySQL Dialector 核心 | `mysql.go` 基础 CRUD | ✅ 已完成 |
| **M2** | MySQL 元数据 | `metadata_provider.go` | ✅ 已完成 |
| **M3** | gore-lint 集成 | `--db-type mysql` | ✅ 已完成 |
| **M4** | MySQL 特有规则 | IDX-MYSQL-001/002/003 | ✅ 已完成 |
| **M5** | 集成测试 | MySQL 8.0 真实验证 | ✅ 已完成 |

### M5 集成测试结果

| 测试项 | 状态 | 说明 |
|--------|------|------|
| MySQL 连接 | ✅ PASS | MySQL 8.0.45 连接成功 |
| 元数据 - Indexes | ✅ PASS | `idx_name`, `idx_status`, `idx_name_desc` 等索引正确读取 |
| 元数据 - Columns | ✅ PASS | 列信息正确读取，包含 `auto_increment` |
| 元数据 - Tables | ✅ PASS | `users`, `orders` 表正确列出 |
| Dialector BuildSelect | ✅ PASS | 7 个子测试全部通过 |
| Dialector BuildInsert | ✅ PASS | INSERT 语句生成正确 |
| Dialector BuildUpdate | ✅ PASS | UPDATE 语句生成正确 |
| Dialector BuildDelete | ✅ PASS | DELETE 语句生成正确 |
| Invisible Index 检测 | ✅ PASS | `idx_status` 检测为 INVISIBLE (NO) |
| Descending Index 检测 | ✅ PASS | `idx_name_desc` collation = D |
| 真实查询执行 | ✅ PASS | CRUD 操作正常执行 |
| gore-lint --db-type mysql | ✅ PASS | DSN 模式正常工作 |

---

## 八、参考

- [MySQL 8.0 Reference Manual - INFORMATION_SCHEMA STATISTICS](https://dev.mysql.com/doc/refman/8.0/en/information-schema-statistics-table.html)
- [MySQL 8.0 New Features](https://dev.mysql.com/doc/refman/8.0/en/mysql-nutshell.html)
- 项目现有 PostgreSQL 实现：`gore/dialect/postgres/`