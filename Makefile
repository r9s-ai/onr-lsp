.PHONY: help build run test fmt tidy clean \
	vscode-install vscode-compile vscode-watch vscode-package vscode-release-check

BIN_DIR := bin
LSP_BIN := $(BIN_DIR)/onr-lsp

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

build: ## Build ONR LSP server binary
	mkdir -p $(BIN_DIR)
	go build -o $(LSP_BIN) ./cmd/onr-lsp

run: ## Run ONR LSP server (stdio)
	go run ./cmd/onr-lsp

test: ## Run Go tests
	go test ./...

fmt: ## Format Go code
	go fmt ./...

tidy: ## Tidy Go modules
	go mod tidy

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)/
	go clean

vscode-install: ## Install VSCode client dependencies
	cd vscode && npm install

vscode-compile: ## Compile VSCode client extension
	cd vscode && npm run compile

vscode-watch: ## Watch-compile VSCode client extension
	cd vscode && npm run watch

vscode-package: ## Package VSCode client extension (.vsix)
	cd vscode && npm run package

vscode-release-check: ## Release pre-check for VSCode extension (compile + package listing)
	cd vscode && npm run compile
	cd vscode && npx vsce ls --tree
	cd vscode && npm run package
