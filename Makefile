.PHONY: help test test-unit test-integration test-all test-coverage clean docker-test-up docker-test-down

help:
	@echo "Available targets:"
	@echo "  test            - Run all tests (unit + integration)"
	@echo "  test-unit       - Run unit tests only"
	@echo "  test-integration - Run integration tests"
	@echo "  test-coverage   - Run tests with coverage"
	@echo "  docker-test-up  - Start test database"
	@echo "  docker-test-down - Stop test database"
	@echo "  clean           - Clean test cache"

test: test-unit test-integration

test-unit:
	@echo "Running unit tests..."
	go test -v -short ./internals/... ./pkg/...

test-integration: docker-test-up
	@echo "Waiting for test database to be ready..."
	@sleep 3
	@echo "Running integration tests..."
	TEST_DB_URL="postgresql://admin:admin@localhost:54323/avito_test?sslmode=disable" go test -v ./tests/...
	@$(MAKE) docker-test-down

test-all:
	@echo "Running all tests..."
	@$(MAKE) test-unit
	@$(MAKE) test-integration

test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

docker-test-up:
	@echo "Starting test database..."
	docker-compose -f docker-compose.test.yml up -d
	@echo "Waiting for database to be healthy..."
	@sleep 5

docker-test-down:
	@echo "Stopping test database..."
	docker-compose -f docker-compose.test.yml down -v

clean:
	@echo "Cleaning test cache..."
	go clean -testcache
	rm -f coverage.out coverage.html

build:
	@echo "Building application..."
	go build -o bin/app cmd/main.go

run:
	@echo "Running application..."
	go run cmd/main.go

docker-up:
	@echo "Starting application with docker-compose..."
	docker-compose up -d

docker-down:
	@echo "Stopping application..."
	docker-compose down

docker-logs:
	docker-compose logs -f

lint:
	@echo "Running linter..."
	golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	go fmt ./...

vet:
	@echo "Running go vet..."
	go vet ./...

tidy:
	@echo "Tidying go modules..."
	go mod tidy

install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.DEFAULT_GOAL := help
