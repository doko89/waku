.PHONY: help build run test clean docker-build docker-run docker-stop docker-clean install dev

# Variables
APP_NAME=waku
VERSION?=dev
DOCKER_IMAGE=waku-api
DOCKER_TAG?=latest

# Colors for output
BLUE=\033[0;34m
GREEN=\033[0;32m
RED=\033[0;31m
NC=\033[0m # No Color

help: ## Show this help message
	@echo '$(BLUE)Available commands:$(NC)'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

install: ## Install dependencies
	@echo '$(BLUE)Installing dependencies...$(NC)'
	go mod download
	go mod tidy
	@echo '$(GREEN)Dependencies installed!$(NC)'

build: ## Build the application
	@echo '$(BLUE)Building $(APP_NAME)...$(NC)'
	CGO_ENABLED=0 go build -ldflags="-w -s -X main.Version=$(VERSION)" -o $(APP_NAME) .
	@echo '$(GREEN)Build complete: ./$(APP_NAME)$(NC)'

build-all: ## Build for all platforms
	@echo '$(BLUE)Building for all platforms...$(NC)'
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o dist/$(APP_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-w -s" -o dist/$(APP_NAME)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o dist/$(APP_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-w -s" -o dist/$(APP_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o dist/$(APP_NAME)-windows-amd64.exe .
	@echo '$(GREEN)Multi-platform build complete!$(NC)'

run: ## Run the application
	@echo '$(BLUE)Running $(APP_NAME)...$(NC)'
	go run main.go

dev: ## Run in development mode with auto-reload (requires air)
	@echo '$(BLUE)Starting development server...$(NC)'
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo '$(RED)air not found. Install it with: go install github.com/cosmtrek/air@latest$(NC)'; \
		echo '$(BLUE)Running without auto-reload...$(NC)'; \
		go run main.go; \
	fi

test: ## Run tests
	@echo '$(BLUE)Running tests...$(NC)'
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@echo '$(GREEN)Tests complete!$(NC)'

test-coverage: test ## Run tests with coverage report
	@echo '$(BLUE)Generating coverage report...$(NC)'
	go tool cover -html=coverage.txt -o coverage.html
	@echo '$(GREEN)Coverage report: coverage.html$(NC)'

clean: ## Clean build artifacts
	@echo '$(BLUE)Cleaning...$(NC)'
	rm -f $(APP_NAME)
	rm -rf dist/
	rm -f coverage.txt coverage.html
	@echo '$(GREEN)Clean complete!$(NC)'

docker-build: ## Build Docker image
	@echo '$(BLUE)Building Docker image...$(NC)'
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo '$(GREEN)Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)'

docker-build-multiarch: ## Build multi-architecture Docker image
	@echo '$(BLUE)Building multi-arch Docker image...$(NC)'
	docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo '$(GREEN)Multi-arch Docker image built!$(NC)'

docker-run: ## Run Docker container
	@echo '$(BLUE)Starting Docker container...$(NC)'
	docker-compose up -d
	@echo '$(GREEN)Container started!$(NC)'
	@echo 'View logs: make docker-logs'

docker-stop: ## Stop Docker container
	@echo '$(BLUE)Stopping Docker container...$(NC)'
	docker-compose down
	@echo '$(GREEN)Container stopped!$(NC)'

docker-logs: ## View Docker container logs
	docker-compose logs -f

docker-clean: docker-stop ## Clean Docker resources
	@echo '$(BLUE)Cleaning Docker resources...$(NC)'
	docker-compose down -v
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) 2>/dev/null || true
	@echo '$(GREEN)Docker cleanup complete!$(NC)'

docker-shell: ## Open shell in running container
	docker-compose exec waku sh

fmt: ## Format code
	@echo '$(BLUE)Formatting code...$(NC)'
	go fmt ./...
	@echo '$(GREEN)Code formatted!$(NC)'

lint: ## Run linter
	@echo '$(BLUE)Running linter...$(NC)'
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo '$(RED)golangci-lint not found. Install it from: https://golangci-lint.run/usage/install/$(NC)'; \
	fi

tidy: ## Tidy dependencies
	@echo '$(BLUE)Tidying dependencies...$(NC)'
	go mod tidy
	@echo '$(GREEN)Dependencies tidied!$(NC)'

setup: install ## Setup development environment
	@echo '$(BLUE)Setting up development environment...$(NC)'
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo '$(GREEN).env file created from .env.example$(NC)'; \
	fi
	@mkdir -p sessions temp
	@echo '$(GREEN)Development environment ready!$(NC)'

.DEFAULT_GOAL := help

