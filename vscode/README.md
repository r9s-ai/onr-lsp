# ONR DSL for VS Code

VS Code extension for editing Open Next Router (ONR) provider DSL (`*.conf`).

![ONR DSL extension preview](https://raw.githubusercontent.com/r9s-ai/onr-lsp/main/preview-image.png)

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
