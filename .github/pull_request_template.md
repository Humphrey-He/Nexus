## 描述
简要说明本 PR 的目的

## 关联 Issue
Fixes #123

## 变更类型
- [ ] 新增功能 (non-breaking)
- [ ] Bug 修复
- [ ] Breaking Change (需走 RFC)
- [ ] 文档更新

## 性能影响
```
基准测试对比：
- Before: 1200 ns/op, 5 allocs/op
- After:  1100 ns/op, 4 allocs/op
```

## 评审清单
- [ ] 通过 golangci-lint
- [ ] 测试覆盖率 ≥ 80%
- [ ] 无内存泄漏 (go test -race)
- [ ] godoc 文档完整
