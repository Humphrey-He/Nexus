# 设计文档

本目录用于描述 gore ORM 的核心设计与实现要点，帮助在 SDD 规范下进行系统化演进。

## 目录

- 变更追踪设计（RFC-0002）
- Index Advisor 设计（RFC-0003）
- 查询链路与执行路径
- 性能与可观测性

---

## 查询链路与执行路径

1. `DbSet[T].Query()` 创建查询构造器。
2. `QueryBuilder` 收集语义（From/Where/Order/Limit/Offset）。
3. `gore-lint` 静态分析提取 QueryMetadata。
4. Rule Engine 生成诊断建议。

---

## 性能与可观测性

### 热点路径

- 变更追踪 (`DetectChanges`) 属于热点路径，未来目标是减少反射与分配。

### 指标

- `ObserveChangeTracking`：统计 SaveChanges 的变更追踪耗时。
- `ObserveSQL`：预留统计 SQL 执行耗时。

---

## Index Advisor 规则

当前实现规则：

- IDX-001 最左匹配检查
- IDX-002 函数索引失效
- IDX-003 隐式类型转换
- IDX-004 前缀通配 LIKE
- IDX-008 排序字段索引
- IDX-009 JOIN 字段索引