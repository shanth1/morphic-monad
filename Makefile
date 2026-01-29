.PHONY: help test lint clean build run docker-build swagger install-swag

# --- Project Variables ---
BINARY_NAME := mm
CMD_MONO_PATH := ./cmd/monolith/main.go

# --- Build Variables ---
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME := $(shell date +%FT%T%z)

LDFLAGS := -ldflags "-s -w -X main.CommitHash=$(COMMIT_HASH) -X main.BuildTime=$(BUILD_TIME)"
GO_BUILD_FLAGS := -trimpath

# --- Docker Variables ---
DOCKER_TAG := $(COMMIT_HASH)

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\n\033[1mUsage:\033[0m\n  make \033[36m<target>\033[0m\n"} \
	/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } \
	/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

run-mono: ## Run app in local mode
	@go run $(CMD_MONO_PATH) --env=local

mocks: ## Generate all mocks
	@echo "Generating mocks..."
	@go generate ./...

audit: ## Run vulnerability check and verify dependencies
	@go list -u -m all
	@go run golang.org/x/vuln/cmd/govulncheck@latest ./...

##@ Testing & Quality

test: ## Run unit tests
	@go test -v -race ./...

lint: ## Run golangci-lint
	@golangci-lint run ./...

lint-install: ## Install golangci-lint
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

clean: ## Remove build artifacts
	@rm -rf build/
