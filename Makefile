.PHONY: help bootstrap up down test contracts-test backend-test frontend-test lint clean

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

bootstrap: ## Install all dependencies (contracts, backend, frontend)
	@echo "==> Foundry"
	@command -v forge >/dev/null || (echo "Install Foundry: curl -L https://foundry.paradigm.xyz | bash && foundryup" && exit 1)
	cd contracts && forge install
	@echo "==> Backend"
	cd backend && go mod tidy
	@echo "==> Frontend"
	cd frontend && pnpm install

up: ## Start local devnet (postgres + anvil)
	docker compose up -d postgres anvil

down: ## Stop local devnet
	docker compose down

test: contracts-test backend-test frontend-test ## Run all tests

contracts-test: ## Run contract tests
	cd contracts && forge test -vvv

backend-test: ## Run backend tests
	cd backend && go test ./...

frontend-test: ## Run frontend tests
	cd frontend && pnpm test

lint: ## Lint everything
	cd contracts && forge fmt --check
	cd backend && go vet ./...
	cd frontend && pnpm lint

clean: ## Remove build artifacts
	cd contracts && forge clean
	rm -rf backend/bin
	rm -rf frontend/.next frontend/out
