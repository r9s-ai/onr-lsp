.PHONY: help build run test fmt tidy clean hooks sync-onr-core-version \
	vscode-install vscode-version-patch vscode-compile vscode-watch vscode-package vscode-release-check vscode-bundle-bins vscode-generate-syntax vscode-install-vsix update-and-install

BIN_DIR := bin
LSP_BIN := $(BIN_DIR)/onr-lsp
GO := go
ONR_REPO_DIR := ..
ONR_CORE_MODULE := github.com/r9s-ai/open-next-router/onr-core
VERSION ?= $(shell cd vscode >/dev/null 2>&1 && node -p "require('./package.json').version" 2>/dev/null || git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS = -s -w \
	-X main.Version=$(VERSION) \
	-X main.Commit=$(COMMIT) \
	-X main.BuildDate=$(BUILD_DATE)
VSIX_NAME = $(shell cd vscode && node -p "require('./package.json').name + '-' + require('./package.json').version + '.vsix'")
VSCODE_BUNDLE_GOOS ?= $(shell $(GO) env GOOS)
VSCODE_BUNDLE_GOARCH ?= $(shell $(GO) env GOARCH)
VSCODE_BUNDLE_PLATFORM = $(if $(filter windows,$(VSCODE_BUNDLE_GOOS)),win32,$(VSCODE_BUNDLE_GOOS))
VSCODE_BUNDLE_ARCH = $(if $(filter amd64,$(VSCODE_BUNDLE_GOARCH)),x64,$(VSCODE_BUNDLE_GOARCH))
VSCODE_BUNDLE_DIR = vscode/bin/$(VSCODE_BUNDLE_PLATFORM)-$(VSCODE_BUNDLE_ARCH)
VSCODE_BUNDLE_EXE = $(if $(filter windows,$(VSCODE_BUNDLE_GOOS)),onr-lsp.exe,onr-lsp)

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

sync-onr-core-version: ## Sync onr-core to the latest local onr-core/v* tag and run go mod tidy
	@set -e; \
	tag=$$(git -C $(ONR_REPO_DIR) tag -l 'onr-core/v*' | sort -V | tail -n 1); \
	if [ -z "$$tag" ]; then \
		echo "No local onr-core/v* tag found in $(ONR_REPO_DIR)"; \
		exit 1; \
	fi; \
	version=$${tag#onr-core/}; \
	echo "Syncing $(ONR_CORE_MODULE) to $$version"; \
	GOWORK=off $(GO) get $(ONR_CORE_MODULE)@$$version; \
	GOWORK=off $(GO) mod tidy

hooks: ## Run prek hooks on all files
	prek run --all-files

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)/
	$(GO) clean

vscode-install: ## Install VSCode client dependencies
	cd vscode && npm install

vscode-version-patch: ## Bump VSCode extension patch version (no git tag)
	cd vscode && npm version patch --no-git-tag-version

vscode-generate-syntax: ## Generate TextMate grammar from onr-core directive metadata
	$(GO) run ./cmd/onr-tmgen -output vscode/syntaxes/onr.tmLanguage.json

vscode-compile: vscode-install vscode-generate-syntax ## Compile VSCode client extension
	cd vscode && npm run compile

vscode-watch: ## Watch-compile VSCode client extension
	cd vscode && npm run watch

vscode-bundle-bins: ## Build bundled onr-lsp binary for the current system
	rm -rf vscode/bin
	mkdir -p $(VSCODE_BUNDLE_DIR)
	CGO_ENABLED=0 GOOS=$(VSCODE_BUNDLE_GOOS) GOARCH=$(VSCODE_BUNDLE_GOARCH) $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(VSCODE_BUNDLE_DIR)/$(VSCODE_BUNDLE_EXE) ./cmd/onr-lsp

vscode-package: vscode-compile vscode-bundle-bins ## Package VSCode client extension (.vsix)
	cd vscode && npm run package

vscode-install-vsix: vscode-package ## Package and install VSIX into VSCode
	code --install-extension vscode/$(VSIX_NAME) --force

vscode-release-check: vscode-install vscode-generate-syntax vscode-bundle-bins ## Release pre-check for VSCode extension (compile + package listing)
	cd vscode && npm run compile
	cd vscode && npm exec vsce ls --tree
	cd vscode && npm run package

update-and-install: sync-onr-core-version vscode-version-patch vscode-install-vsix
