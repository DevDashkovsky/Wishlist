COMPOSE ?= docker compose

.PHONY: test test-race lint test-e2e test-load test-load-script
test:
	go test ./...

test-race:
	go test -race ./...

lint:
	golangci-lint run ./...

test-e2e:
	COMPOSE='$(COMPOSE)' sh scripts/test-e2e.sh

test-load:
	COMPOSE='$(COMPOSE)' sh scripts/test-load.sh

test-load-script:
	node --experimental-vm-modules --test scripts/load-test.test.mjs
