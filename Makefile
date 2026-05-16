.PHONY: build test test-e2e lint gosec clean run

# Default target runs build
all: build

# Build the CLI binary locally
build:
	@echo "🔨 Building Axen..."
	@mkdir -p bin
	@go build -o ./bin/axen ./cmd/axen
	@echo "✓ Binary built at ./bin/axen"

# Run all unit tests
test:
	@echo "🧪 Running unit tests..."
	@go test ./internal/...

# Run E2E tests
test-e2e:
	@echo "🧪 Running E2E tests..."
	@go test ./test/e2e -v

# Run golangci-lint with auto-install fallback
lint:
	@echo "🧹 Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	elif [ -f $(HOME)/go/bin/golangci-lint ]; then \
		$(HOME)/go/bin/golangci-lint run; \
	else \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		$(HOME)/go/bin/golangci-lint run; \
	fi

# Run security checks matching CI exclusions with auto-install fallback
gosec:
	@echo "🛡️ Running gosec scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -exclude=G104,G122,G204,G301,G304,G306,G703 ./...; \
	elif [ -f $(HOME)/go/bin/gosec ]; then \
		$(HOME)/go/bin/gosec -exclude=G104,G122,G204,G301,G304,G306,G703 ./...; \
	else \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
		$(HOME)/go/bin/gosec -exclude=G104,G122,G204,G301,G304,G306,G703 ./...; \
	fi

# Clean up built binaries and local coverage outputs
clean:
	@echo "🧽 Cleaning up..."
	@rm -rf bin/ coverage.out
	@echo "✓ Cleanup complete."

# Build and run the local CLI directly
run: build
	@./bin/axen $(ARGS)
