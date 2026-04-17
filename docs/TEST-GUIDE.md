# gore-lint 手动测试指南

**测试日期**: 2026-04-15
**前提条件**: PostgreSQL 已启动，gore-lint 已编译

---

## 一、环境准备

### 1.1 启动 PostgreSQL

```bash
# 检查容器状态
docker ps --filter "name=gore-postgres"

# 如果未启动，运行:
docker run -d --name gore-postgres -e POSTGRES_USER=gore -e POSTGRES_PASSWORD=gore123 -e POSTGRES_DB=gore -p 5432:5432 postgres:15-alpine

# 验证连接
docker exec gore-postgres psql -U gore -d gore -c "SELECT version();"
```

### 1.2 编译 gore-lint

```bash
cd E:/awesomeProject/Nexus/gore
go build -o gore-lint.exe ./cmd/gore-lint/
```

### 1.3 创建测试数据库表

```bash
docker exec gore-postgres psql -U gore -d gore -c "
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS users CASCADE;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE,
    age INT,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id),
    amount DECIMAL(10,2),
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX users_name_idx ON users(name);
CREATE INDEX users_name_age_idx ON users(name, age);
CREATE INDEX users_created_at_idx ON users(created_at);
CREATE INDEX users_age_idx ON users(age);
CREATE INDEX orders_user_id_idx ON orders(user_id);
CREATE INDEX orders_status_idx ON orders(status);
"
```

---

## 二、Schema 管理命令测试

### 2.1 Schema Dump (导出数据库结构)

```bash
# 导出到文件
./gore-lint.exe schema dump --dsn "postgres://gore:gore123@localhost:5432/gore?sslmode=disable" -o schema.json

# 验证文件
cat schema.json | head -30
```

**预期结果**: 生成包含 users 和 orders 表结构的 JSON 文件

---

### 2.2 Schema Validate (验证 Schema 文件)

```bash
./gore-lint.exe schema validate schema.json
```

**预期结果**: `Schema valid: 2 table(s)`

---

### 2.3 Schema Merge (合并多个 Schema)

```bash
# 创建额外 schema
echo '{"tables":[{"tableName":"products","columns":[{"name":"id","type":"integer"}],"indexes":[]}]}' > extra.json

# 合并
./gore-lint.exe schema merge -o merged.json schema.json extra.json

# 验证
./gore-lint.exe schema validate merged.json
```

**预期结果**: `Schema valid: 3 table(s)`

---

## 三、代码检查命令测试

### 3.1 创建测试代码文件

```bash
mkdir -p testcode && cat > testcode/sample.go << 'EOF'
package sample

import "context"

type User struct {
    ID        int
    Name      string
    Email     string
    Age       int
    Status    string
    CreatedAt string
}

var ctx context.Context

// ✅ GOOD: Uses first column of composite index
func getUserByName() {
    _ = Set[User](ctx).Query().From("users").WhereField("name", "=", "alice")
}

// ✅ GOOD: Uses first then second column
func getUserByNameAndAge() {
    _ = Set[User](ctx).Query().From("users").WhereField("name", "=", "alice").WhereField("age", ">", 18)
}

// ✅ GOOD: Uses primary key
func getUserByID() {
    _ = Set[User](ctx).Query().From("users").WhereField("id", "=", 1)
}

// ❌ BAD: Skips first column of composite index
func getUsersByAge() {
    _ = Set[User](ctx).Query().From("users").WhereField("age", ">", 18)
}

// ❌ BAD: Leading wildcard cannot use index
func searchUsersByName() {
    _ = Set[User](ctx).Query().From("users").WhereField("name", "LIKE", "%alice%")
}

// ❌ BAD: Negation
func getInactiveUsers() {
    _ = Set[User](ctx).Query().From("users").WhereField("status", "!=", "deleted")
}

// ❌ BAD: Type mismatch
func getUsersByAgeString() {
    _ = Set[User](ctx).Query().From("users").WhereField("age", "=", "twenty")
}
EOF
```

### 3.2 检查命令 - JSON 格式

```bash
./gore-lint.exe check --format json --schema schema.json testcode/
```

**预期结果**: JSON 输出包含 5 个 suggestions (IDX-001 x2, IDX-003, IDX-004, IDX-005)

---

### 3.3 检查命令 - TEXT 格式

```bash
./gore-lint.exe check --format text --schema schema.json testcode/
```

**预期输出**:
```
gore-lint 0.1
Target: testcode/
Found 5 issue(s): 0 info, 5 warning, 0 high, 0 critical

[WARNING] IDX-001: 联合索引 users_name_age_idx 的首列 name 未在 WHERE 中出现
[WARNING] IDX-003: 字段 age 类型(integer) 与参数类型(string) 不匹配
[WARNING] IDX-004: 字段 name 使用前缀通配 LIKE 模式
[WARNING] IDX-005: 字段 status 使用了否定条件操作符 !=
[WARNING] IDX-001: 联合索引 users_name_age_idx 的首列 name 未在 WHERE 中出现
```

---

### 3.4 检查命令 - SARIF 格式

```bash
./gore-lint.exe check --format sarif --schema schema.json testcode/
```

**预期结果**: 符合 SARIF 2.1.0 规范的 JSON 输出

---

### 3.5 使用 Live DSN (不生成 schema 文件)

```bash
./gore-lint.exe check --format text --dsn "postgres://gore:gore123@localhost:5432/gore?sslmode=disable" testcode/
```

**预期结果**: 与使用 --schema 相同的结果

---

## 四、修复验证测试

### 4.1 创建修复后的代码

```bash
cat > testcode/fixed.go << 'EOF'
package sample

import "context"

type User struct {
    ID        int
    Name      string
    Email     string
    Age       int
    Status    string
    CreatedAt string
}

var ctx context.Context

// ✅ FIXED: Uses name first, then age
func getUsersByNameAndAge() {
    _ = Set[User](ctx).Query().From("users").WhereField("name", "=", "alice").WhereField("age", ">", 18)
}

// ✅ FIXED: Uses suffix wildcard instead of prefix
func searchUsersByName() {
    _ = Set[User](ctx).Query().From("users").WhereField("name", "LIKE", "alice%")
}

// ✅ FIXED: Uses positive condition
func getActiveUsers() {
    _ = Set[User](ctx).Query().From("users").WhereField("status", "=", "active")
}

// ✅ FIXED: Uses correct type
func getUsersByAgeInt() {
    _ = Set[User](ctx).Query().From("users").WhereField("age", ">", 20)
}
EOF
```

### 4.2 验证修复

```bash
./gore-lint.exe check --format text --schema schema.json testcode/fixed.go
```

**预期结果**: 无警告输出 (只有 users_age_idx 首列未使用的 1 个 IDX-001 警告，这是正常的)

---

## 五、规则说明

| 规则ID | 名称 | 检测问题 | 严重级别 |
|--------|------|----------|----------|
| IDX-001 | 最左匹配 | 跳过复合索引首列 | Warning |
| IDX-002 | 函数索引 | 在索引列上使用函数 | Warning |
| IDX-003 | 类型转换 | 隐式类型转换 | Warning |
| IDX-004 | 前缀通配 | LIKE '%xxx' 前缀通配 | Warning |
| IDX-005 | 否定条件 | !=, <>, NOT 条件 | Warning |
| IDX-006 | 缺失索引 | 高频查询字段无索引 | High |
| IDX-007 | 冗余索引 | 冗余或覆盖索引 | Info |
| IDX-008 | 排序索引 | ORDER BY 与索引方向不一致 | Info |
| IDX-009 | JOIN 索引 | JOIN 字段无索引 | Warning |
| IDX-010 | 低选择性 | 选择性过低的索引 | Info |

---

## 六、故障排除

### 6.1 连接失败

```
Error: failed to ping database
```

**解决**: 检查 PostgreSQL 是否启动
```bash
docker ps | grep gore-postgres
```

### 6.2 SSL 错误

```
Error: pq: SSL is not enabled on the server
```

**解决**: 在 DSN 后添加 `?sslmode=disable`

### 6.3 Schema 文件无效

```
Error: invalid JSON
```

**解决**: 确保 schema.json 是有效的 JSON 格式

---

## 七、快速验证脚本

```bash
#!/bin/bash
set -e

cd E:/awesomeProject/Nexus/gore

echo "=== 1. Schema Dump ==="
./gore-lint.exe schema dump --dsn "postgres://gore:gore123@localhost:5432/gore?sslmode=disable" -o /tmp/test_schema.json

echo "=== 2. Schema Validate ==="
./gore-lint.exe schema validate /tmp/test_schema.json

echo "=== 3. Check Test Code ==="
./gore-lint.exe check --format text --schema /tmp/test_schema.json testcode/

echo "=== All tests passed! ==="
```
