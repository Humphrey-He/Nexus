# RFC-0002: Change Tracker 方案对比与决策

**状态**：Proposed  
**优先级**：P0（核心特性）  
**作者**：[Your Name]  
**创建日期**：2026-03-13  
**讨论链接**：[GitHub Issue #2]  
**相关 RFC**：RFC-0001 (DbContext 设计)

---

## 1. 问题陈述 (Problem Statement)

Change Tracking 是 ORM 框架的核心特性，负责：
- **状态追踪**：识别实体的 Added/Modified/Deleted 状态
- **增量更新**：仅更新变更字段，而非全量 UPDATE
- **工作单元模式**：支持 SaveChanges() 的事务语义

**痛点**：
- GORM 缺乏原生 Change Tracking，需要手动指定更新字段
- ent 通过代码生成实现，但生成代码量大、维护困难
- C# EF 的 Change Tracking 优雅，但 Go 难以直接复刻

**目标**：
- 选择一个 **性能与易用性平衡** 的方案
- 支持 **大对象追踪**，避免内存溢出或显著 GC 压力
- 提供 **可选追踪控制**（AsNoTracking、AsTracking）

---

## 2. 候选方案 (Proposed Solutions)

### 方案 A: 快照对比法 (Snapshot Diffing) ⭐ 推荐

**原理**：
1. 实体从 DB 加载时，在 Session 缓存中存储原始值的 **深拷贝** 或 **哈希**
2. SaveChanges() 时逐字段对比当前值与快照
3. 仅将变化字段加入 UPDATE 的 SET 子句

**实现伪代码**：
```go
// 加载时
user := &User{ID: 1, Name: "Alice", Age: 30}
session.Attach(user)
session.snapshot[user] = deepCopy(user)  // 或 hashSnapshot(user)

// 修改
user.Name = "Bob"

// SaveChanges 时
changes := diffSnapshot(session.snapshot[user], user)
// changes = {Name: "Bob"}
```

**优点**：
- 实现路径清晰，适配 Go 反射能力
- 追踪准确度高
- 与 EF 行为一致，易于理解

**缺点**：
- 大对象深拷贝导致内存消耗大
- SaveChanges() 开销与对象大小成正比

---

### 方案 B: Dirty Flag（字段级别变更标记）

**原理**：
1. 通过 setter 或代码生成，为每个字段设置 dirty 标记
2. SaveChanges() 时只更新 dirty 字段

**优点**：
- SaveChanges() 开销低
- 对象大时内存占用更低

**缺点**：
- 需要代码生成或侵入式 setter
- 不符合 Go 结构体直接赋值习惯
- 与现有生态不匹配

---

### 方案 C: 代理对象 (Proxy Tracking)

**原理**：
1. 通过包装代理对象拦截字段写操作
2. 记录变更字段与状态

**优点**：
- 运行时可追踪，无需深拷贝

**缺点**：
- Go 不支持真正的运行时代理
- 侵入性高，API 复杂度高

---

## 3. 方案对比 (Comparison)

| 维度 | 方案 A: 快照对比 | 方案 B: Dirty Flag | 方案 C: 代理 |
| --- | --- | --- | --- |
| 实现复杂度 | 中 | 高 | 高 |
| 运行时开销 | 中 | 低 | 中 |
| 内存开销 | 高 | 低 | 中 |
| 侵入性 | 低 | 高 | 高 |
| Go 适配性 | 高 | 低 | 低 |

---

## 4. 决策 (Decision)

采用 **方案 A: 快照对比法**。

**理由**：
1. Go 生态可行性最高，落地成本最低
2. 与 EF 的使用习惯一致，开发者易理解
3. 可通过配置项和优化策略缓解内存问题

---

## 5. 实施计划 (Implementation Plan)

### Phase 1（v0.1）
- 提供基础 Change Tracker（Snapshot Diffing）
- 仅支持简单实体的 Added/Modified/Deleted 追踪

### Phase 2（v0.2）
- 引入 **hash-based snapshot**（避免深拷贝）
- 支持按需追踪与 AsNoTracking

### Phase 3（v0.3）
- 研究 Dirty Flag 作为可选优化路径

---

## 6. 风险与对策 (Risks & Mitigation)

- **风险**：快照对比对大对象内存压力大
  - **对策**：提供 `AsNoTracking()`；支持 Hash Snapshot

- **风险**：反射 diff 性能不足
  - **对策**：引入字段映射缓存；对热点对象提供 codegen 优化路径

---

## 7. 度量指标 (Metrics)

- SaveChanges() 单次 diff 时间 < 1ms（中等对象）
- 大对象内存增长 < 10%（基于 50MB 实体样本）
- 追踪准确率 >= 99%

---

## 8. 开放问题 (Open Questions)

- 是否引入默认 hash snapshot？
- 何时触发快照更新（SaveChanges 后还是按事务边界）？
- 如何定义 AsTracking 的默认行为？

---

## 9. 相关 RFC (Related RFCs)

- RFC-0001: DbContext 设计
- RFC-0003: Index Advisor 架构设计
