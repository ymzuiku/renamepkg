# renamepkg

A simple tool to rename Go packages and modules. No AST, just regex magic ✨

English | [中文](README-zh.md)

## Install

```sh
go install github.com/ymzuiku/renamepkg/cmd/renamepkg@latest
```

## Rename Module

Change your entire module path in one go:

```bash
renamepkg --mod github.com/pillar/doaddon
```

- Reads current module from `go.mod`
- Updates all imports across your codebase

## Rename Package

Rename a subpackage and update all references:

```bash
renamepkg --from internal/server/di --to internal/server/difish
```

- Automatically reads module path from `go.mod`
- Renames the directory
- Updates package declarations
- Updates all imports (keeps original package name as alias)

That's it. Simple, fast, regex-powered. 🚀
