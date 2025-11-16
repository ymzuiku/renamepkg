# renamepkg

一个简单的重命名 Go 包和模块的工具。无需 AST，仅使用正则表达式魔法 ✨

[English](README.md) | 中文

## 重命名模块

一键更改整个模块路径：

```bash
renamepkg --mod github.com/pillar/doaddon
```

- 从 `go.mod` 读取当前模块
- 更新代码库中的所有导入

## 重命名包

重命名子包并更新所有引用：

```bash
renamepkg --from internal/server/di --to internal/server/difish
```

- 自动从 `go.mod` 读取模块路径
- 重命名目录
- 更新包声明
- 更新所有导入（保留原始包名作为别名）
- 使用 `--force` 覆盖现有目录

就是这样。简单、快速、基于正则表达式。🚀
