.PHONY: help test lint clean build-mono build-micro infra-up infra-down run-mono run-gw run-router

# --- Project Variables ---
BINARY_DIR := bin
CMD_MONO_PATH := ./cmd/monolith/main.go
CMD_GW_PATH := ./cmd/microservices/gateway/main.go
CMD_ROUTER_PATH := ./cmd/microservices/router/main.go
CMD_ENGINE_PATH := ./cmd/microservices/engine/main.go
CMD_EMBEDDER_PATH := ./cmd/microservices/embedder/main.go

# --- Build Variables ---
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME := $(shell date +%FT%T%z)

# LDFLAGS: -s -w remove debug information (reduces the binary size), -X inject variables
LDFLAGS := -ldflags "-s -w -X main.CommitHash=$(COMMIT_HASH) -X main.BuildTime=$(BUILD_TIME)"
GO_BUILD_FLAGS := -trimpath

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\n\033[1mUsage:\033[0m\n  make \033[36m<target>\033[0m\n"} \
	/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } \
	/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Local Development (Monolith)

run-mono: ## Run application in All-in-One Monolith mode (In-Memory)
	@echo "🚀 Starting Monolith..."
	@go run $(LDFLAGS) $(CMD_MONO_PATH) --env=local

##@ Distributed Development (Microservices)

infra-up: ## Start ONLY infrastructure (NATS, S3)
	@echo "🐳 Starting infrastructure..."
	@docker-compose -f docker-compose.yaml up -d

infra-down: ## Stop infrastructure
	@echo "🛑 Stopping infrastructure..."
	@docker-compose -f docker-compose.yaml down

full-up: ## Start EVERYTHING (Infra + All Microservices)
	@echo "🚀 Starting full distributed stack..."
	@docker-compose -f docker-compose.yaml -f docker-compose.apps.yaml up -d --build

full-down: ## Stop full stack
	@echo "🛑 Stopping full stack..."
	@docker-compose -f docker-compose.yaml -f docker-compose.apps.yaml down

run-gw: ## Run ONLY the Gateway microservice
	@echo "🚀 Starting Gateway Microservice..."
	@APP_ENV=local BLOB_PROVIDER=s3 go run $(LDFLAGS) $(CMD_GW_PATH) --env=local

run-router: ## Run ONLY the Router microservice
	@echo "🚀 Starting Router Microservice..."
	@APP_ENV=local go run $(LDFLAGS) $(CMD_ROUTER_PATH) --env=local

run-engine: ## Run ONLY the Engine microservice
	@echo "🚀 Starting Engine Microservice..."
	@APP_ENV=local go run $(LDFLAGS) $(CMD_ENGINE_PATH) --env=local

run-embedder: ## Run ONLY the Embedder microservice
	@echo "🚀 Starting Embedder Microservice..."
	@APP_ENV=local go run $(LDFLAGS) $(CMD_EMBEDDER_PATH) --env=local

##@ Build & Quality

build-mono: clean ## Build the monolith binary
	@echo "🔨 Building monolith..."
	@CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_DIR)/monolith $(CMD_MONO_PATH)

build-micro: clean ## Build microservices binaries
	@echo "🔨 Building microservices..."
	@CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_DIR)/gateway $(CMD_GW_PATH)
	@CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_DIR)/router $(CMD_ROUTER_PATH)
	@CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_DIR)/engine $(CMD_ENGINE_PATH)
	@CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_DIR)/embedder $(CMD_EMBEDDER_PATH)

test: ## Run unit tests with race detector
	@go test -v -race ./...

lint: ## Run golangci-lint
	@golangci-lint run ./...

clean: ## Remove build artifacts
	@rm -rf $(BINARY_DIR)/
