# ONR DSL for VS Code

VS Code extension for editing Open Next Router (ONR) provider DSL (`*.conf`).

<p align="center">
  <a href="https://github.com/r9s-ai/onr-lsp/actions/workflows/publish-vscode.yml"><img src="https://github.com/r9s-ai/onr-lsp/actions/workflows/publish-vscode.yml/badge.svg" alt="CI" /></a>
  <a href="https://github.com/r9s-ai/onr-lsp/releases"><img src="https://img.shields.io/github/v/release/r9s-ai/onr-lsp" alt="GitHub Release" /></a>
  <a href="https://marketplace.visualstudio.com/items?itemName=r9s-ai.onr-dsl"><img src="https://img.shields.io/visual-studio-marketplace/v/r9s-ai.onr-dsl?label=Marketplace" alt="VS Code Marketplace" /></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License" /></a>
</p>

<p align="center">
  <a href="https://deepwiki.com/r9s-ai/onr-lsp">Ask DeepWiki</a>
  <a href="https://zread.ai/r9s-ai/onr-lsp"><img src="https://img.shields.io/badge/Ask_Zread-_.svg?style=flat&color=00b0aa&labelColor=000000&logo=data%3Aimage%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB3aWR0aD0iMTYiIGhlaWdodD0iMTYiIHZpZXdCb3g9IjAgMCAxNiAxNiIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTQuOTYxNTYgMS42MDAxSDIuMjQxNTZDMS44ODgxIDEuNjAwMSAxLjYwMTU2IDEuODg2NjQgMS42MDE1NiAyLjI0MDFWNC45NjAxQzEuNjAxNTYgNS4zMTM1NiAxLjg4ODEgNS42MDAxIDIuMjQxNTYgNS42MDAxSDQuOTYxNTZDNS4zMTUwMiA1LjYwMDEgNS42MDE1NiA1LjMxMzU2IDUuNjAxNTYgNC45NjAxVjIuMjQwMUM1LjYwMTU2IDEuODg2NjQgNS4zMTUwMiAxLjYwMDEgNC45NjE1NiAxLjYwMDFaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik00Ljk2MTU2IDEwLjM5OTlIMi4yNDE1NkMxLjg4ODEgMTAuMzk5OSAxLjYwMTU2IDEwLjY4NjQgMS42MDE1NiAxMS4wMzk5VjEzLjc1OTlDMS42MDE1NiAxNC4xMTM0IDEuODg4MSAxNC4zOTk5IDIuMjQxNTYgMTQuMzk5OUg0Ljk2MTU2QzUuMzE1MDIgMTQuMzk5OSA1LjYwMTU2IDE0LjExMzQgNS42MDE1NiAxMy43NTk5VjExLjAzOTlDNS42MDE1NiAxMC42ODY0IDUuMzE1MDIgMTAuMzk5OSA0Ljk2MTU2IDEwLjM5OTlaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik0xMy43NTg0IDEuNjAwMUgxMS4wMzg0QzEwLjY4NSAxLjYwMDEgMTAuMzk4NCAxLjg4NjY0IDEwLjM5ODQgMi4yNDAxVjQuOTYwMUMxMC4zOTg0IDUuMzEzNTYgMTAuNjg1IDUuNjAwMSAxMS4wMzg0IDUuNjAwMUgxMy43NTg0QzE0LjExMTkgNS42MDAxIDE0LjM5ODQgNS4zMTM1NiAxNC4zOTg0IDQuOTYwMVYyLjI0MDFDMTQuMzk4NCAxLjg4NjY0IDE0LjExMTkgMS42MDAxIDEzLjc1ODQgMS42MDAxWiIgZmlsbD0iI2ZmZiIvPgo8cGF0aCBkPSJNNCAxMkwxMiA0TDQgMTJaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik00IDEyTDEyIDQiIHN0cm9rZT0iI2ZmZiIgc3Ryb2tlLXdpZHRoPSIxLjUiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIvPgo8L3N2Zz4K&logoColor=ffffff" alt="Ask Zread" /></a>
  <a href="https://t.me/opennextrouter"><img src="https://img.shields.io/badge/Telegram-Join-blue?logo=telegram" alt="Telegram" /></a>
  <a href="https://discord.gg/HBM67dP8"><img src="https://img.shields.io/badge/Discord-Join-5865F2?logo=discord&logoColor=white" alt="Discord" /></a>
</p>

<p align="center">
  <img src="https://raw.githubusercontent.com/r9s-ai/onr-lsp/main/preview-image.png" alt="ONR DSL extension preview" width="520" />
</p>

## Features

- Syntax highlight
  - TextMate grammar for immediate lexical highlight
  - Semantic tokens from `onr-lsp` for context-aware token coloring
- Completion
  - Directive completion by current DSL block
  - Mapper mode completion for directives like `req_map`, `resp_map`, `sse_parse`
  - Enum value completion for selected directives (for example `balance_unit`, `method`, `oauth_content_type`)
- Hover
  - Short directive documentation from ONR DSL metadata
- Diagnostics
  - Basic syntax diagnostics (missing braces, unknown directives)
  - Semantic diagnostics for invalid mapper modes
- Formatting
  - Document formatting via `textDocument/formatting` from `onr-lsp`

## Scope

The extension client is activated for:

- `**/providers/*.conf`

## Server Resolution

The extension resolves language server binary in this order:

1. `onrLsp.serverPath` (if configured)
2. bundled binary in extension package (`bin/<platform>-<arch>/onr-lsp`)
3. `onr-lsp` from system `PATH`

## Configuration

- `onrLsp.serverPath`
  - Optional absolute path or command name for `onr-lsp`
  - Keep empty to use bundled binary first

Language defaults provided by this extension:

- `[onr-dsl].editor.defaultFormatter = "r9s-ai.onr-dsl"`
- `[onr-dsl].editor.formatOnSave = true`

You can still override these defaults in User/Workspace `settings.json`.

## Build and Package (Repo Local)

From `onr-lsp/`:

```bash
make vscode-compile
make vscode-package
```

## Git Hooks (prek)

```bash
# install git hooks (force-replace if pre-commit hooks already exist)
prek install -f

# run all hooks manually
prek run --all-files
```

## Format CLI (Server Binary Test)

```bash
# Build onr-lsp binary first
go build -o bin/onr-lsp ./cmd/onr-lsp

# Read from stdin, write formatted result to stdout
cat config/providers/openai.conf | ./bin/onr-lsp format

# Format one file, output to stdout
./bin/onr-lsp format config/providers/openai.conf

# Format one file in-place
./bin/onr-lsp format --write config/providers/openai.conf
```

## Notes

- If you just installed/updated the extension, run `Developer: Reload Window` once.
- This extension only provides editor/LSP capabilities. Runtime behavior is still defined by ONR DSL config and ONR server.
