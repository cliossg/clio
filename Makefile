APP_NAME = clio
BUILD_DIR = build
BINARY = $(BUILD_DIR)/$(APP_NAME)

all: build

help:
	@echo "Targets:"
	@echo ""
	@echo "Build & Run:"
	@echo "  build           - Build the application"
	@echo "  run             - Build and run (dev mode)"
	@echo "  run-prod        - Build and run (prod mode)"
	@echo "  kill            - Stop the application"
	@echo ""
	@echo "Testing:"
	@echo "  test            - Run tests"
	@echo "  test-v          - Run tests verbose"
	@echo "  test-short      - Run tests in short mode"
	@echo "  test-coverage   - Run tests with coverage"
	@echo "  test-coverage-profile - Generate coverage profile"
	@echo "  test-coverage-html - Generate HTML coverage report"
	@echo "  test-coverage-func - Show function-level coverage"
	@echo "  test-coverage-check - Check coverage meets 85%"
	@echo "  test-coverage-100 - Check coverage is 100%"
	@echo "  test-coverage-summary - Display coverage by package"
	@echo ""
	@echo "Quality:"
	@echo "  check           - Run all quality checks"
	@echo "  lint            - Run golangci-lint"
	@echo "  fmt             - Format code"
	@echo "  vet             - Run go vet"
	@echo ""
	@echo "Tools:"
	@echo "  sqlc            - Generate sqlc code"
	@echo "  clean           - Clean build files"
	@echo "  dev-db-reset    - Delete dev database and workspace"

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

test-short:
	@go test -short ./...

test-coverage:
	@go test -cover ./...

test-coverage-profile:
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

test-coverage-html: test-coverage-profile
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Report: coverage.html"

test-coverage-func: test-coverage-profile
	@go tool cover -func=coverage.out

test-coverage-check: test-coverage-profile
	@COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Coverage: $$COVERAGE%"; \
	if [ $$(awk -v cov="$$COVERAGE" 'BEGIN {print (cov < 85)}') -eq 1 ]; then \
		echo "❌ Below 85% threshold"; exit 1; \
	else \
		echo "✅ Meets 85% threshold"; \
	fi

test-coverage-100: test-coverage-profile
	@COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Coverage: $$COVERAGE%"; \
	if [ "$$COVERAGE" != "100.0" ]; then \
		echo "❌ Coverage is not 100%"; \
		go tool cover -func=coverage.out | grep -v "100.0%"; \
		exit 1; \
	else \
		echo "✅ Perfect! 100% coverage"; \
	fi

test-coverage-summary:
	@echo "Coverage by package:"
	@echo "┌──────────────────────────────────────────────┬──────────┐"
	@echo "│ Package                                      │ Coverage │"
	@echo "├──────────────────────────────────────────────┼──────────┤"
	@for pkg in $$(go list ./... 2>/dev/null); do \
		pkgname=$$(echo $$pkg | sed 's|github.com/cliossg/clio/||'); \
		result=$$(go test -cover $$pkg 2>&1); \
		cov=$$(echo "$$result" | grep -oE '[0-9]+\.[0-9]+%' | tail -1); \
		if [ -z "$$cov" ]; then \
			if echo "$$result" | grep -q "no test files"; then \
				cov="no tests"; \
			else \
				cov="0.0%"; \
			fi; \
		fi; \
		printf "│ %-44s │ %8s │\n" "$$pkgname" "$$cov"; \
	done
	@echo "└──────────────────────────────────────────────┴──────────┘"

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

check: fmt vet test test-coverage-check lint
	@echo "✅ All checks passed"

clean:
	@rm -rf $(BUILD_DIR) coverage.out coverage.html
	@go clean -testcache

dev-db-reset:
	@echo "Resetting dev database..."
	@rm -f _workspace/db/clio.db
	@rm -rf _workspace/sites
	@echo "Done. Run 'make run' to recreate."

.PHONY: all build run run-prod kill test test-v test-short test-coverage test-coverage-profile test-coverage-html test-coverage-func test-coverage-check test-coverage-100 test-coverage-summary fmt vet lint sqlc tidy check clean dev-db-reset help
