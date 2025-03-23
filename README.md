# NixVis

Nginx 网站日志分析工具。

## Commit Message Generator

由于远程 Copilot 无法直接生成提交记录，可手动生成 diff 文件并交给聊天工具自动生成提交信息。 

```bash
git add .
git diff --staged > diff.patch
```

```markdown
Generate Chinese commit messages according to these English guidelines:

Format:
    [Type] Concise description (Chinese)
    (Use multi-line format only when changes are unrelated)

Examples:
    - feat 添加用户注册功能

Key Rules:
    1. Use concise verbs to start the description.Verbs including but not limited to:feat、fix、style、refactor、docs、perf
    2. Combine strongly related changes under one type (e.g. 架构调整+布局优化=refactor)
    3. Split only when changes have different nature,maximum 3 type lines per commit
    4. Give me a concise and efficient submission record directly, and there is no need for any explanation
```

