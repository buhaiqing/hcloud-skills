# hcloud-skills — top-level Makefile.
# Delegates Go commands into skillcheck/ so you can build/release from root.
# Run `make help` to see available targets.

GO ?= go
MODULE_DIR := skillcheck

.PHONY: all build test vet fmt lint self-check clean tidy release help

all: fmt vet test build ## fmt + vet + test + build

build: ## Build skillcheck/bin/skillcheck
	$(GO) build -C $(MODULE_DIR) -trimpath -o bin/skillcheck .

test: ## Run the full test suite
	$(GO) test -C $(MODULE_DIR) ./... -count=1

vet: ## Run go vet
	$(GO) vet -C $(MODULE_DIR) ./...

fmt: ## Check formatting (gofmt -l); non-zero if unformatted files exist
	@out=$$($(GO) fmt -C $(MODULE_DIR) ./...); \
	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

lint: ## Run the bundled `lint go` subcommand (gofmt + go vet)
	$(GO) run -C $(MODULE_DIR) . lint go --root $(MODULE_DIR)

self-check: build ## Exercise the binary against its embedded healthy fixtures
	./$(MODULE_DIR)/bin/skillcheck scan secret trace --self-check
	./$(MODULE_DIR)/bin/skillcheck scan secret summary --self-check
	./$(MODULE_DIR)/bin/skillcheck scan secret alarm-plan --self-check
	./$(MODULE_DIR)/bin/skillcheck aggregate trace --self-check

tidy: ## Tidy go.mod / go.sum
	$(GO) mod -C $(MODULE_DIR) tidy

clean: ## Remove build artifacts
	rm -rf $(MODULE_DIR)/bin

VERSION ?= $(shell git describe --tags --dirty 2>/dev/null || echo "dev")
RELEASE_TAG = $(if $(filter v%,$(VERSION)),$(VERSION),v$(VERSION))
release: all ## Build, tag, and push to trigger GitHub Release (VERSION=X.Y.Z or vX.Y.Z)
	git tag $(RELEASE_TAG)
	git push origin $(RELEASE_TAG)
	@echo "Released $(RELEASE_TAG) — CI will build + publish artifacts"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'