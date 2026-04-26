BINARY := cairnlint

GO ?= go
GOFLAGS_TEST ?= -v -race -count=1

.DEFAULT_GOAL := help
.PHONY: help test test-fast build install lint vet tidy clean

help: ## List available targets
	@awk 'BEGIN { FS = ":.*##"; printf "cairnlint targets:\n" } \
	      /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' \
	      $(MAKEFILE_LIST)

test: ## Run the full test suite the same way CI does
	$(GO) test $(GOFLAGS_TEST) ./...

test-fast: ## Run tests without -race for quick local loops
	$(GO) test -count=1 ./...

build: ## Build the cairnlint binary into the repo root
	$(GO) build -o $(BINARY) .

install: ## go install . (drops the binary into GOBIN)
	$(GO) install .

lint: ## golangci-lint run ./...
	golangci-lint run ./...

vet: ## go vet ./...
	$(GO) vet ./...

tidy: ## go mod tidy
	$(GO) mod tidy

clean: ## Remove the local binary and the go test cache
	$(GO) clean -testcache
	rm -f $(BINARY)
