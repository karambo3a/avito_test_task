# Run Service
up:
	docker compose up -d

down:
	docker compose down


# Run Test Service
test-up:
	docker compose -f test/compose.integration.yaml up -d

test-down:
	docker compose -f test/compose.integration.yaml down


# Run Tests
test: test-up
	go test -v ./test/integration/...
	$(MAKE) test-down

# Linters
lint:
	golangci-lint run ./...
