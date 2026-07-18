# Makefile for the `skillcheck` Go CLI.
#
# Local development companion to .github/workflows/build-skillcheck.yml.
# Run `make` to see available targets; `make help` for details.

GO ?= go
BINARY ?= skillcheck
PKGS := ./...

.PHONY: all build test vet fmt lint self-check clean tidy release help

all: fmt vet test build ## fmt + vet + test + build

build: ## Build ./bin/skillcheck
	$(GO) build -trimpath -o bin/$(BINARY) .

test: ## Run the full test suite
	$(GO) test $(PKGS) -count=1

vet: ## Run go vet
	$(GO) vet $(PKGS)

fmt: ## Check formatting (gofmt -l); non-zero if unformatted files exist
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

lint: ## Run the bundled `lint go` subcommand (gofmt + go vet)
	$(GO) run . lint go --root .

self-check: build ## Exercise the binary against its embedded healthy fixtures
	./bin/$(BINARY) scan secret trace --self-check
	./bin/$(BINARY) scan secret summary --self-check
	./bin/$(BINARY) scan secret alarm-plan --self-check
	./bin/$(BINARY) aggregate trace --self-check

tidy: ## Tidy go.mod / go.sum
	$(GO) mod tidy

clean: ## Remove build artifacts
	rm -rf bin

VERSION ?= $(shell git describe --tags --dirty 2>/dev/null || echo "dev")
# Normalize: prepend "v" if VERSION doesn't start with "v"
RELEASE_TAG = $(if $(filter v%,$(VERSION)),$(VERSION),v$(VERSION))
release: all ## Build, tag, and push to trigger GitHub Release (VERSION=X.Y.Z or vX.Y.Z)
	git tag $(RELEASE_TAG)
	git push origin $(RELEASE_TAG)
	@echo "Released $(RELEASE_TAG) — CI will build + publish artifacts"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'
