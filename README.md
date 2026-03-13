# Nexus (gore ORM)

规范驱动的 Go ORM 实验项目，参考字节 RFC 与腾讯 TDR 流程，强调问题导向、演进式设计与可度量指标。

## 现状

- 已完成 SDD 规范文档
- gore 模块骨架、Change Tracker 原型
- Index Advisor 规则引擎与 gore-lint CLI 骨架
- PostgreSQL pg_catalog 元数据读取

## 文档

- SDD: `SDD.md`
- RFC:
  - `docs/rfc/0001-dbcontext-design.md`
  - `docs/rfc/0002-change-tracker.md`
  - `docs/rfc/0003-index-advisor.md`
- API 文档: `docs/api/README.md`
- 设计文档: `docs/design/README.md`

## 快速开始

```
cd gore

go test ./...
```

## gore-lint

```
cd gore

# 使用离线 schema JSON
go run ./cmd/gore-lint check --schema /path/to/schema.json ./...
```

## 许可证

内部研发试验用途。