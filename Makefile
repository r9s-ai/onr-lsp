.PHONY: help build run test fmt tidy clean \
	vscode-install vscode-compile vscode-watch vscode-package vscode-release-check vscode-bundle-bins vscode-generate-syntax vscode-install-vsix

BIN_DIR := bin
LSP_BIN := $(BIN_DIR)/onr-lsp
GO := go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w \
	-X main.Version=$(VERSION) \
	-X main.Commit=$(COMMIT) \
	-X main.BuildDate=$(BUILD_DATE)
VSIX_NAME := $(shell cd vscode && node -p "require('./package.json').name + '-' + require('./package.json').version + '.vsix'")

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

build: ## Build ONR LSP server binary
	mkdir -p $(BIN_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(LSP_BIN) ./cmd/onr-lsp

run: ## Run ONR LSP server (stdio)
	$(GO) run -ldflags "$(LDFLAGS)" ./cmd/onr-lsp

test: ## Run Go tests
	$(GO) test ./...

fmt: ## Format Go code
	$(GO) fmt ./...

tidy: ## Tidy Go modules
	$(GO) mod tidy

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)/
	$(GO) clean

vscode-install: ## Install VSCode client dependencies
	cd vscode && npm install

vscode-generate-syntax: ## Generate TextMate grammar from onr-core directive metadata
	$(GO) run ./cmd/onr-tmgen -output vscode/syntaxes/onr.tmLanguage.json

vscode-compile: vscode-generate-syntax ## Compile VSCode client extension
	cd vscode && npm run compile

vscode-watch: ## Watch-compile VSCode client extension
	cd vscode && npm run watch

	vscode-bundle-bins: ## Build bundled onr-lsp binaries for VSCode extension
	rm -rf vscode/bin
	mkdir -p vscode/bin/linux-x64 vscode/bin/linux-arm64 vscode/bin/darwin-x64 vscode/bin/darwin-arm64 vscode/bin/win32-x64 vscode/bin/win32-arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o vscode/bin/linux-x64/onr-lsp ./cmd/onr-lsp
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o vscode/bin/linux-arm64/onr-lsp ./cmd/onr-lsp
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o vscode/bin/darwin-x64/onr-lsp ./cmd/onr-lsp
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o vscode/bin/darwin-arm64/onr-lsp ./cmd/onr-lsp
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o vscode/bin/win32-x64/onr-lsp.exe ./cmd/onr-lsp
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o vscode/bin/win32-arm64/onr-lsp.exe ./cmd/onr-lsp

vscode-package: vscode-generate-syntax vscode-bundle-bins ## Package VSCode client extension (.vsix)
	cd vscode && npm run package

vscode-install-vsix: vscode-package ## Package and install VSIX into VSCode
	code --install-extension vscode/$(VSIX_NAME) --force

vscode-release-check: vscode-generate-syntax vscode-bundle-bins ## Release pre-check for VSCode extension (compile + package listing)
	cd vscode && npm run compile
	cd vscode && npx vsce ls --tree
	cd vscode && npm run package
