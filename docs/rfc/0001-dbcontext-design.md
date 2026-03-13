# RFC-0001: DbContext 与 DbSet 的泛型设计

**状态**：Proposed  
**优先级**：P0  
**作者**：[Your Name]  
**讨论链接**：[GitHub Issue #1]

## 问题陈述
Go 缺乏像 C# EF 那样优雅的泛型 ORM API。

## 设计方案

### 选项 A：Method Receiver 泛型（推荐）
```go
type DbContext struct { ... }

func (ctx *DbContext) Set[T any]() *DbSet[T] {
    return &DbSet[T]{ctx: ctx}
}
```

### 选项 B：泛型结构体
```go
type DbContext[T any] struct { ... }  // 不推荐：过度泛型化
```

### 选项 C：工厂 + 反射映射
```go
type DbContext struct { ... }

func (ctx *DbContext) Set(t reflect.Type) DbSetIface {
    // 反射路径，不推荐
}
```

## 决策
**采用选项 A**，原因：
1. 一个 DbContext 管理多个实体类型
2. 符合 Go 的简洁性原则
3. 最小化反射依赖

## 实现计划
- Week 1: 接口定义 + 骨架
- Week 2: 基础 CRUD 实现
- Week 3: 单测 + 文档

## 风险与对策
- 风险：DbContext 的线程安全定义不清晰
  - 对策：明确声明非线程安全（请求级别），并在 godoc 中强调

## 相关问题
- 如何处理 DbContext 的线程安全性？→ RFC-0004
