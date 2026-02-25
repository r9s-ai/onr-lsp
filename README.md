# onr-lsp
Language Server Protocol (LSP) implementation for OpenNextRouter (.conf) configuration files – autocompletion, diagnostics, hover docs for ONR DSL.

## Current Features

- Completion:
  - `req_map <mode>`
  - `resp_map <mode>`
  - `sse_parse <mode>`
- Hover docs for common directives (`provider`, `match`, `req_map`, `resp_map`, `sse_parse`, etc.)
- Diagnostics:
  - missing `}`
  - unknown directive
  - semantic validation via `onr-core/pkg/dslconfig` (e.g. unsupported mapper mode)

## Run Server

```bash
go build -o bin/onr-lsp ./cmd/onr-lsp
./bin/onr-lsp
```

## VSCode Client (minimal)

Client project is under `vscode/`.

```bash
cd vscode
npm install
npm run compile
```

Then run Extension Development Host in VSCode (`F5`) and set:

- `onrLsp.serverPath`: absolute path of your server binary, for example:
  - `/data/code/github/edgefn/next-router/open-next-router/onr-lsp/bin/onr-lsp`
- The language client attaches to provider files by default:
  - `config/providers/*.conf`

Runtime resolution order:

1. `onrLsp.serverPath` (if configured)
2. bundled binary inside extension (`bin/<platform>-<arch>/onr-lsp`)
3. `onr-lsp` from system `PATH`

## GitHub Actions Auto Publish

Workflow file:

- `.github/workflows/publish-vscode.yml`

Behavior:

- Trigger: push to `main` when `vscode/**` changes.
- Runs `go test ./...` and extension compile.
- Builds bundled `onr-lsp` binaries for Linux/macOS/Windows (x64/arm64).
- Publishes extension via `npx vsce publish`.

Required repository secret:

- `VSCE_PAT`: Visual Studio Marketplace Personal Access Token.

Important:

- `vsce publish` requires a new extension version each time.
- Bump `vscode/package.json` `version` before merging to `main`.
