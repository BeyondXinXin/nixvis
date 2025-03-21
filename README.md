


## Commit Message Generator

由于远程 Copilot 无法直接生成提交记录，可手动生成 diff 文件并交给聊天工具自动生成提交信息。 

```bash
git add .
git diff --staged > diff.patch
```

```markdown
Please generate a commit message in Chinese based on the following guidelines:

Format: 
type1 简明描述变更内容1
type2 简明描述变更内容2


Examples: 
- feat 添加用户注册功能
- fix 修复密码强度验证逻辑错误
- style 统一组件命名格式
- refactor 重构通讯模块
- docs 更新文档
- perf 优化数据加载接口响应时间


Key Rules:
1. Use concise verbs to start the description.
2. Use the following types (or others as appropriate):feat: New features、fix: Bug fixes、style: Code style adjustments、refactor: Code refactoring、docs: Documentation updates、perf: Performance improvements.
3. New Rule: Allow multiple types in a single commit.
```

