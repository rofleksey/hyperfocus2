VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X 'hyperfocus/internal/container.Version=$(VERSION)'

.PHONY: all build run tidy vet lint fmt test typecheck frontend clean distclean docker
.DEFAULT_GOAL := help

help: ## Show this help
	@awk 'BEGIN{FS=":.*## "; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\n"} /^[a-zA-Z_-]+:.*## /{printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: frontend build

# Build the backend binary with the version baked in.
build:
	go build -ldflags "$(LDFLAGS)" -o bin/hyperfocus ./cmd/server

# Run the backend (expects ./config.yaml and a reachable Postgres).
run:
	go run -ldflags "$(LDFLAGS)" ./cmd/server

tidy:
	go mod tidy

vet:
	go vet ./...

fmt:
	gofmt -s -w .

lint: vet ## Run go vet + golangci-lint
	@command -v golangci-lint >/dev/null 2>&1 || { echo "ERROR: golangci-lint not found — install from https://golangci-lint.run"; exit 1; }
	golangci-lint run ./...

test: ## Run tests with race detector
	go test -race -cover ./...

typecheck: ## Run Vue/TS type checker
	cd web && npm run typecheck

# Build the Vue frontend into web/dist (embedded into the Go binary).
web/node_modules: web/package-lock.json
	cd web && npm ci
	touch $@

frontend: web/node_modules ## Build the Vue frontend
	cd web && npm run build

# Wipe build artifacts (keeps data/ and node_modules).
clean:
	rm -rf bin web/dist

# Wipe all artifacts including node_modules.
distclean: clean
	rm -rf web/node_modules

docker: ## Build Docker image
	docker build --build-arg VERSION=$(VERSION) -t rofleksey/hyperfocus2:$(VERSION) .
