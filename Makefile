GO ?= go
GOCACHE ?= /tmp/evaly-go-build
GOPATH ?= /tmp/evaly-gopath
export GOCACHE GOPATH
GOLANGCI_LINT_CACHE ?= /tmp/evaly-golangci-cache
GOLANGCI_LINT_VERSION := v2.14.0
GOLANGCI_LINT ?= $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOLANGCI_LINT_RUN := env GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) $(GOLANGCI_LINT) run --allow-serial-runners --max-issues-per-linter=0 --max-same-issues=0 --uniq-by-line=false
MODULES := $(shell find . -type d \( -name ".*" -not -name "." -o -name "vendor" \) -prune -o -type f -name "go.mod" -exec dirname {} \;)
RELEASE_MODULES := .
FUZZTIME ?= 30s
FUZZPARALLEL ?= 2

.PHONY: config-check fmt fmt-check format vet lint fix test validate examples bench fuzz cover schemas fixtures release release-patch release-break

config-check:
	@$(GOLANGCI_LINT) config verify

fmt:
	@for dir in $(MODULES); do \
		(cd "$$dir" && $(GOLANGCI_LINT) fmt) || exit 1; \
	done

fmt-check:
	@for dir in $(MODULES); do \
		(cd "$$dir" && $(GOLANGCI_LINT) fmt --diff) || exit 1; \
	done

format: fmt-check

vet:
	@for dir in $(MODULES); do \
		(cd "$$dir" && $(GO) vet ./...) || exit 1; \
	done

lint:
	@for dir in $(MODULES); do \
		echo "golangci-lint - $$dir"; \
		(cd "$$dir" && $(GOLANGCI_LINT_RUN) ./...) || exit 1; \
	done

fix:
	@if [ -f "go.work" ]; then $(GO) work sync; fi
	@for dir in $(MODULES); do \
		echo "fix & tidy - $$dir"; \
		(cd "$$dir" && $(GO) fix ./... && $(GO) mod tidy) || exit 1; \
		(cd "$$dir" && $(GOLANGCI_LINT_RUN) --fix ./...) || exit 1; \
	done

test:
	@for dir in $(MODULES); do \
		echo "test - $$dir"; \
		(cd "$$dir" && $(GO) test -v -race ./...) || exit 1; \
	done

validate: config-check format vet lint test examples

examples:
	$(GO) run ./examples/calculation
	$(GO) run ./examples/crm
	$(GO) run ./examples/protocols
	$(GO) run ./examples/http
	$(GO) run ./examples/observation
	$(GO) run ./examples/optimizer
	$(GO) run ./examples/integration

bench:
	@for dir in $(MODULES); do \
		echo "bench - $$dir"; \
		(cd "$$dir" && $(GO) test -bench=. -benchmem -run=^$$ ./...) || exit 1; \
	done

fuzz:
	@for dir in $(MODULES); do \
		(cd "$$dir" && \
			packages=$$($(GO) list -tags=fuzz ./...) && \
			for pkg in $$packages; do \
				targets=$$($(GO) test -tags=fuzz -list '^Fuzz' "$$pkg") || exit 1; \
				for target in $$targets; do \
					case "$$target" in Fuzz*) \
						$(GO) test -tags=fuzz -run='^$$' -fuzz="^$$target$$" \
							-fuzztime=$(FUZZTIME) -parallel=$(FUZZPARALLEL) "$$pkg" || exit 1 ;; \
					esac; \
				done; \
			done) || exit 1; \
	done

cover:
	@for dir in $(MODULES); do \
		echo "cover - $$dir"; \
		(cd "$$dir" && $(GO) test -coverprofile=coverage.out ./... && $(GO) tool cover -func=coverage.out) || exit 1; \
	done

release: release-break

release-patch: validate
	@bash ./scripts/release.sh patch "$(RELEASE_MODULES)"

release-break: validate
	@bash ./scripts/release.sh break "$(RELEASE_MODULES)"

schemas:
	$(GO) run ./internal/schemagen

fixtures:
	$(GO) run ./cmd/evaly fixture --store /tmp/evaly-fixtures --id baseline
