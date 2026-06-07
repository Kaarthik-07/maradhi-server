# Makefile for Maradhi API
# All commands assume you're inside the Nix dev shell (`nix develop` or direnv).
# Run `make help` to see all commands.

APP_NAME  = maradhi-api
BUILD_DIR = ./bin
TMP_DIR   = ./tmp
MAIN_PATH = ./cmd/api

# ─── Nix ──────────────────────────────────────────────────────────────────────

.PHONY: dev-shell
dev-shell: ## Enter the Nix dev shell manually (if not using direnv)
	nix develop

.PHONY: nix-build
nix-build: ## Build reproducible binary via Nix (output: ./result/bin/api)
	nix build

.PHONY: nix-docker
nix-docker: ## Build Docker image via Nix and load it into Docker
	nix build .#docker
	docker load < result
	@echo "Image loaded: $(APP_NAME)"

.PHONY: nix-run
nix-run: ## Build and run directly with nix run
	nix run

.PHONY: nix-check
nix-check: ## Run nix flake checks (builds package, checks flake structure)
	nix flake check

.PHONY: nix-update
nix-update: ## Update all flake inputs and regenerate flake.lock
	nix flake update

# ─── Dev ──────────────────────────────────────────────────────────────────────

.PHONY: run
run: ## Run the server directly with `go run`
	go run $(MAIN_PATH)/main.go

.PHONY: watch
watch: ## Live reload (air watches .go files and restarts on change)
	air

.PHONY: build
build: ## Compile binary to ./bin/
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Built: $(BUILD_DIR)/$(APP_NAME)"

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR) $(TMP_DIR) result

# ─── Test & Quality ───────────────────────────────────────────────────────────

.PHONY: test
test: ## Run all tests with race detector and coverage
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

.PHONY: test-v
test-v: ## Run tests with verbose output
	go test -race -v ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: vuln
vuln: ## Scan for known vulnerabilities
	govulncheck ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: fmt
fmt: ## Format Go files with goimports (handles imports + formatting)
	goimports -w .

# ─── DB / Migrations ──────────────────────────────────────────────────────────

.PHONY: migrate
migrate: ## Apply all migrations (requires DATABASE_URL in env)
	@if [ -z "$$DATABASE_URL" ]; then echo "ERROR: DATABASE_URL not set"; exit 1; fi
	@echo "Applying migrations..."
	@for f in migrations/*.sql; do \
		echo "  -> $$f"; \
		psql "$$DATABASE_URL" -f $$f; \
	done
	@echo "Done."

.PHONY: db-connect
db-connect: ## Open a psql session to the database
	@if [ -z "$$DATABASE_URL" ]; then echo "ERROR: DATABASE_URL not set"; exit 1; fi
	psql "$$DATABASE_URL"

# ─── Go modules ───────────────────────────────────────────────────────────────

.PHONY: tidy
tidy: ## Tidy go.mod and verify modules
	go mod tidy
	go mod verify

.PHONY: vendor-hash
vendor-hash: ## Print the vendorHash for flake.nix after updating go.mod
	@echo "Updating vendor directory..."
	go mod vendor
	nix hash path vendor/
	@echo "Paste this hash as vendorHash in flake.nix"

# ─── Help ─────────────────────────────────────────────────────────────────────

.PHONY: help
help: ## Show this help
	@echo ""
	@echo "  Maradhi API — available commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
	@echo ""
