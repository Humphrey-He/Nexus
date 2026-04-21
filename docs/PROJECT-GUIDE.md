# Nexus (gore ORM) 项目说明

## 项目概述

**Nexus** 是一个规范驱动的 Go ORM 实验项目，参考字节跳动 RFC 与腾讯 TDR 流程，强调问题导向、演进式设计与可度量指标。

项目目标是为 Go 项目提供与 Entity Framework 相近的开发体验，包括强类型查询、变更追踪、索引诊断等企业级功能。

## 项目状态

| 模块 | 完成度 | 状态 |
|------|--------|------|
| API (DbContext/DbSet) | 94% | ✅ 主力开发 |
| Query Builder (含 JOIN) | 95% | ✅ 主力开发 |
| Index Advisor 规则 | 90% | ✅ 主力开发 |
| gore-lint CLI | 85% | ✅ 可用 |
| Change Tracker | 80% | ✅ 可用 |
| PostgreSQL 方言 | 100% | ✅ 完成 |
| MySQL 方言 (含 Invisible/Downgrade Index) | 100% | ✅ 完成 |
| MongoDB 方言 | 100% | ✅ 完成 |
| 事务支持 | 80% | ✅ 可用 |
| 批量操作 | 80% | ✅ 可用 |
| 日志框架 | 100% | ✅ 完成 |
| 连接池 | 100% | ✅ 完成 |
| 错误标准化 | 100% | ✅ 完成 |

**总体完成度**: ~88%

**测试覆盖率**:
| 模块 | 覆盖率 |
|------|--------|
| gore/api | 73.4% |
| gore/dialect/mysql | 93.4% |
| gore/dialect/postgres | 92.7% |
| gore/internal/advisor/rules | 85.0% |
| gore/internal/errors | 93.8% |

## 核心功能

### 1. 强类型查询构建器
```go
users := api.Set[User](ctx).
    WhereField("status", "=", "active").
    WhereField("age", ">", 18).
    OrderBy("created_at DESC").
    Limit(10)
```

### 2. 变更追踪 (Change Tracking)
```go
user := &User{Name: "Alice"}
api.Set[User](ctx).Add(user)
// ... 修改 user
count, _ := ctx.SaveChanges(context.Background()) // 自动生成 UPDATE
```

### 3. 索引诊断引擎 (Index Advisor)
自动分析 SQL 查询，检测潜在索引问题：
- 最左匹配检查
- 函数索引失效
- 隐式类型转换
- 前缀通配 LIKE
- 否定条件检测
- 索引缺失/冗余检测

### 4. gore-lint CLI 工具
```bash
# 静态分析
gore-lint check --schema schema.json ./...

# 生成报告
gore-lint check --schema schema.json --format sarif ./... > report.sarif
```

## 目录结构

```
Nexus/
├── SDD.md                    # 软件概要设计文档
├── CHANGELOG.md              # 变更日志
├── LICENSE                   # MIT 开源协议
├── CONTRIBUTING.md           # 贡献指南
├── SECURITY.md               # 安全披露政策
├── gore/                     # 核心 ORM 模块
│   ├── api/                  # DbContext, DbSet, Query Builder, Logger
│   ├── dialect/             # 数据库方言抽象
│   │   ├── postgres/        # PostgreSQL 实现
│   │   ├── mysql/           # MySQL 实现 (含 Invisible/Downgrade Index)
│   │   └── mongodb/         # MongoDB 实现 (含索引/集合管理)
│   ├── internal/
│   │   ├── errors/          # 标准化错误处理
│   │   ├── executor/         # SQL 执行器 + 连接池
│   │   ├── advisor/          # 索引诊断引擎
│   │   │   └── rules/        # 诊断规则 (IDX-001 ~ IDX-010 + MySQL)
│   │   ├── tracker/          # 变更追踪
│   │   └── metadata/          # 元数据管理
│   ├── cmd/gore-lint/        # CLI 工具
│   ├── config/               # 配置管理
│   ├── tests/                # 单元测试
│   └── testcode/              # 用户示例测试
├── docs/
│   ├── api/                  # API 文档
│   ├── design/               # 设计文档
│   ├── rfc/                  # RFC 提案
│   └── roadmap/               # 路线图
└── .github/workflows/        # CI/CD 配置
```

## 技术栈

- **语言**: Go 1.22+
- **数据库**: PostgreSQL ✅, MySQL 8.0+ ✅, MongoDB 7.0+ ✅
- **测试**: go-sqlmock, MySQL 8.0 (Docker)
- **依赖**: lib/pq (PostgreSQL), go-sql-driver/mysql

## 版本历史

- **v0.1-alpha.1** (2026-03-13): 初始版本，核心 API 骨架
- **v0.2** (2026-04-17): MySQL 支持、CI/CD 完善、测试覆盖提升
- **v0.3** (2026-04-20): JOIN 支持、事务、批量操作、日志、连接池、错误标准化

## 许可证

MIT License - 详见 [LICENSE](../LICENSE)

## 相关文档

- [SDD 设计文档](./SDD.md)
- [API 文档](./docs/api/README.md)
- [设计文档](./docs/design/README.md)
- [RFC 提案](./docs/rfc/)
- [实现评审报告](./docs/roadmap/IMPLEMENTATION-REVIEW.md)
