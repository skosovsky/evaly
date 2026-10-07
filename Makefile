GO ?= go
GOCACHE ?= /tmp/evaly-go-build
GOPATH ?= /tmp/evaly-gopath
export GOCACHE GOPATH
GOLANGCI_LINT_CACHE ?= /tmp/evaly-golangci-cache
GOLANGCI_LINT_VERSION := $(shell python3 -c 'import json; print(json.load(open("checks/registry.json"))["linter"])')
GOLANGCI_LINT ?= $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOLANGCI_LINT_RUN := env GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) $(GOLANGCI_LINT) run --allow-serial-runners --max-issues-per-linter=0 --max-same-issues=0 --uniq-by-line=false
MODULES := $(shell python3 -c 'import json; print(" ".join(json.load(open("checks/registry.json"))["modules"]))')
RELEASE_MODULES := $(shell python3 -c 'import json; print(" ".join(json.load(open("checks/registry.json"))["release_modules"]))')
FUZZTIME ?= 30s
FUZZPARALLEL ?= 2

.PHONY: check-plan check test-fast config-check fmt fmt-check format vet lint fix test validate examples bench fuzz cover schemas fixtures release release-patch release-break

check-plan:
	@python3 scripts/checks.py plan

check:
	@python3 scripts/checks.py check

test-fast:
	@python3 scripts/checks.py fast

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
	@python3 scripts/checks.py lint

fix:
	@if [ -f "go.work" ]; then $(GO) work sync; fi
	@for dir in $(MODULES); do \
		echo "fix & tidy - $$dir"; \
		(cd "$$dir" && $(GO) fix ./... && $(GO) mod tidy) || exit 1; \
		(cd "$$dir" && $(GOLANGCI_LINT_RUN) --fix ./...) || exit 1; \
	done

test:
	@python3 scripts/checks.py test

validate: check

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

release-patch:
	@bash ./scripts/release.sh patch "$(RELEASE_MODULES)"

release-break:
	@bash ./scripts/release.sh break "$(RELEASE_MODULES)"

schemas:
	$(GO) run ./internal/schemagen

fixtures:
	$(GO) run ./cmd/evaly fixture --store /tmp/evaly-fixtures --id baseline

# Optional SDK consumers use pinned published dependencies by default.
.PHONY: consumer-test consumer-source
consumer-test:
	@python3 scripts/checks.py operation consumer-published

consumer-source:
	@python3 scripts/checks.py operation consumer-source
