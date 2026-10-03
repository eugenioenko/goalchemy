GO ?= go
GOALCHEMY = $(GO) run ./cmd/goalchemy

.PHONY: build test spec-generate spec-check

build:
	$(GO) build ./...

test:
	$(GO) test -short ./...

spec-generate:
	$(GOALCHEMY) spec generate

spec-check:
	$(GOALCHEMY) spec validate
	$(GOALCHEMY) spec generate -check
