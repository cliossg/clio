APP_NAME = clio
BUILD_DIR = build
BINARY = $(BUILD_DIR)/$(APP_NAME)

all: build

help:
	@echo "Targets:"
	@echo "  build       - Build the application"
	@echo "  run         - Build and run (dev mode)"
	@echo "  run-prod    - Build and run (prod mode)"
	@echo "  test        - Run tests"
	@echo "  test-v      - Run tests verbose"
	@echo "  coverage    - Run tests with coverage"
	@echo "  coverage-html - Generate HTML coverage report"
	@echo "  check       - Run all quality checks"
	@echo "  lint        - Run golangci-lint"
	@echo "  fmt         - Format code"
	@echo "  vet         - Run go vet"
	@echo "  sqlc        - Generate sqlc code"
	@echo "  clean       - Clean build files"
	@echo "  db-reset    - Delete dev database and workspace"

build:
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BINARY) .

run: build
	@CLIO_ENV=dev $(BINARY)

run-prod: build
	@CLIO_ENV=prod $(BINARY)

kill:
	@pkill -9 $(APP_NAME) 2>/dev/null || true
	@lsof -ti :8080 | xargs kill -9 2>/dev/null || true

test:
	@go test ./...

test-v:
	@go test -v ./...

coverage:
	@go test -cover ./...

coverage-profile:
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

coverage-html: coverage-profile
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Report: coverage.html"

coverage-check: coverage-profile
	@COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE < 85" | bc -l) -eq 1 ]; then \
		echo "❌ Below 85% threshold"; exit 1; \
	else \
		echo "✅ Meets 85% threshold"; \
	fi

fmt:
	@gofmt -w .

vet:
	@go vet ./...

lint:
	@golangci-lint run --fix

sqlc:
	@sqlc generate

tidy:
	@go mod tidy

check: fmt vet test coverage-check lint
	@echo "✅ All checks passed"

clean:
	@rm -rf $(BUILD_DIR) coverage.out coverage.html
	@go clean -testcache

db-reset:
	@echo "Resetting dev database..."
	@rm -f _workspace/db/clio.db
	@rm -rf _workspace/sites
	@echo "Done. Run 'make run' to recreate."

.PHONY: all build run run-prod kill test test-v coverage coverage-profile coverage-html coverage-check fmt vet lint sqlc tidy check clean db-reset help
