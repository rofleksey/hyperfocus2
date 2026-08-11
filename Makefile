VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X 'hyperfocus/internal/container.Version=$(VERSION)'

.PHONY: all build run tidy vet lint fmt frontend clean docker

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

lint: vet
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed; skipping"

# Build the Vue frontend into web/dist (embedded into the Go binary).
frontend:
	cd web && npm ci && npm run build

# Wipe build artifacts and the embedded frontend (keeps data/ intact).
clean:
	rm -rf bin web/dist web/node_modules

docker:
	docker build --build-arg VERSION=$(VERSION) -t rofleksey/hyperfocus2:$(VERSION) .
