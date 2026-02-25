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
