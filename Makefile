COMPOSE ?= docker compose

.PHONY: test test-race lint test-e2e
test:
	go test ./...

test-race:
	go test -race ./...

lint:
	golangci-lint run ./...

test-e2e:
	COMPOSE='$(COMPOSE)' sh scripts/test-e2e.sh
