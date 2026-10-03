# Hamsa — Building Management MVP (P0)
# Dev setup: docker compose up -d   (PostgreSQL 16 on :5432)
#            cp backend/config.example.yaml backend/config.yaml

.PHONY: help up down lint test build backend-test mobile-test mobile-lint mobile-gen \
        web-install web-dev web-build web-preview web-lint web-check web-test web-e2e web-fonts

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

up: ## Start PostgreSQL (docker compose)
	docker compose up -d

down: ## Stop PostgreSQL
	docker compose down

## Backend (Go) ---------------------------------------------------------------

lint: ## Lint backend (golangci-lint) + mobile (flutter analyze) + web
	cd backend && golangci-lint run ./...
	cd mobile && flutter analyze
	cd web && npm run lint

test: backend-test mobile-test web-test ## Run all tests

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

## Web (SvelteKit) -------------------------------------------------------------
# The dev server proxies /api and /files to localhost:8080, so the backend must
# be running for anything that touches the API.

web-install: ## Install web dependencies
	cd web && npm install

web-dev: ## Run the web dev server (proxies /api to :8080)
	cd web && npm run dev

web-build: ## Build the static web bundle to web/build
	cd web && npm run build

web-preview: ## Serve the production web build
	cd web && npm run preview

web-check: ## Type-check the web app (svelte-check)
	cd web && npm run check

web-lint: ## Lint + format-check the web app
	cd web && npm run lint

web-test: ## Run web unit + component tests
	cd web && npm run test:unit -- --run

web-e2e: ## Run web E2E tests against a built app
	cd web && npm run test:e2e

web-fonts: ## Rebuild the subsetted Vazirmatn woff2 faces
	cd web && node scripts/build-fonts.mjs
