# Hamsa — Building Management MVP (P0)
# Dev setup: docker compose up -d   (PostgreSQL 16 on :5432)
#            cp backend/config.example.yaml backend/config.yaml

.PHONY: help up down lint test build backend-test mobile-test mobile-lint mobile-gen

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

up: ## Start PostgreSQL (docker compose)
	docker compose up -d

down: ## Stop PostgreSQL
	docker compose down

## Backend (Go) ---------------------------------------------------------------

lint: ## Lint backend (golangci-lint) + mobile (flutter analyze)
	cd backend && golangci-lint run ./...
	cd mobile && flutter analyze

test: backend-test mobile-test ## Run all tests

backend-test: ## Run backend Go tests
	cd backend && go test ./...

build: ## Build backend binary
	cd backend && go build -o bin/server ./cmd/server

## Mobile (Flutter) -----------------------------------------------------------

mobile-lint: ## Analyze mobile app
	cd mobile && flutter analyze

mobile-test: ## Run mobile unit/widget tests
	cd mobile && flutter test

mobile-gen: ## Run freezed/json code generation
	cd mobile && dart run build_runner build --delete-conflicting-outputs
