BINARY := cairnlint

GO ?= go
GOFLAGS_TEST ?= -v -race -count=1

.DEFAULT_GOAL := help
.PHONY: help test test-fast eval eval-update build install lint vet tidy update-deps clean

help: ## List available targets
	@awk 'BEGIN { FS = ":.*##"; printf "cairnlint targets:\n" } \
	      /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' \
	      $(MAKEFILE_LIST)

test: ## Run the full test suite the same way CI does
	$(GO) test $(GOFLAGS_TEST) ./...

test-fast: ## Run tests without -race for quick local loops
	$(GO) test -count=1 ./...

eval: ## Run only the eval: the built binary over analyzers/testdata/src, against testdata/eval.txt
	$(GO) test -count=1 -run '^TestEval' .

eval-update: ## Re-record the flag lines of testdata/eval.txt from the current binary, then run the eval; review the diff
	$(GO) test -count=1 -run '^TestEval$$' . -args -update
	$(GO) test -count=1 -run '^TestEval' .

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

# A hijacked release is usually found and pulled within days, so update-deps takes nothing younger.
MIN_AGE_DAYS := 3

# pinact runs as a one-shot so nothing is installed; the token keeps it under the API rate limit.
update-deps: ## Move Go modules, the go directive, and action pins to the newest release at least MIN_AGE_DAYS old
	$(GO) run ./scripts/updatedeps -min-age-days $(MIN_AGE_DAYS)
	GITHUB_TOKEN="$$(gh auth token)" $(GO) run github.com/suzuki-shunsuke/pinact/v3/cmd/pinact@latest run --update --min-age $(MIN_AGE_DAYS)

clean: ## Remove the local binary and the go test cache
	$(GO) clean -testcache
	rm -f $(BINARY)
