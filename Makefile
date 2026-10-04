GO ?= go
GOLANGCI_LINT ?= golangci-lint
MODULES := . contracttest
FUZZTIME ?= 30s
RELEASE_VERSION ?= v0.1.0

.PHONY: lint fix fmt fmt-check vet test race validate bench fuzz cover fixtures schemas release

lint:
	@for dir in $(MODULES); do \
		echo "golangci-lint - $$dir"; \
		(cd "$$dir" && $(GOLANGCI_LINT) run --allow-serial-runners ./...) || exit 1; \
	done

fix:
	@for dir in $(MODULES); do \
		echo "fix & tidy - $$dir"; \
		(cd "$$dir" && $(GO) fix ./... && $(GO) mod tidy && $(GOLANGCI_LINT) fmt && $(GOLANGCI_LINT) run --fix --allow-serial-runners ./...) || exit 1; \
	done

fmt:
	@for dir in $(MODULES); do \
		(cd "$$dir" && $(GOLANGCI_LINT) fmt) || exit 1; \
	done

fmt-check:
	@for dir in $(MODULES); do \
		(cd "$$dir" && diff=$$($(GOLANGCI_LINT) fmt --diff) && \
			if [ -n "$$diff" ]; then printf '%s\n' "$$diff"; exit 1; fi) || exit 1; \
	done

vet:
	@for dir in $(MODULES); do \
		(cd "$$dir" && $(GO) vet ./...) || exit 1; \
	done

test:
	@for dir in $(MODULES); do \
		echo "test - $$dir"; \
		(cd "$$dir" && $(GO) test -v -race ./...) || exit 1; \
	done

race: test

validate: fmt-check vet test

bench:
	@for dir in $(MODULES); do \
		echo "bench - $$dir"; \
		(cd "$$dir" && $(GO) test -bench=. -run='^$$' ./...) || exit 1; \
	done

fuzz:
	@for dir in $(MODULES); do \
		echo "fuzz - $$dir"; \
		(cd "$$dir" && \
			for pkg in $$($(GO) list -tags=fuzz ./...); do \
				for target in $$($(GO) test -tags=fuzz -list '^Fuzz' "$$pkg" | sed -n '/^Fuzz/p'); do \
					$(GO) test -tags=fuzz -fuzz="^$$target$$" -fuzztime=$(FUZZTIME) "$$pkg" || exit 1; \
				done; \
			done \
		) || exit 1; \
	done

cover:
	@for dir in $(MODULES); do \
		echo "cover - $$dir"; \
		(cd "$$dir" && $(GO) test -coverprofile=coverage.out ./... && $(GO) tool cover -func=coverage.out) || exit 1; \
	done

fixtures:
	$(GO) run ./cmd/evaly fixture --store /tmp/evaly-fixtures --id baseline

schemas:
	$(GO) run ./internal/schemagen

release: validate
	bash scripts/release.sh "$(RELEASE_VERSION)"
