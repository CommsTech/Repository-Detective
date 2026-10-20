# Repository Detective Makefile

# Variables
BINARY_NAME=repository-detective
BUILD_DIR=build
DOCKER_IMAGE=repository-detective
DOCKER_TAG=latest

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_UNIX=$(BINARY_NAME)_unix

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(shell git describe --tags --always --dirty) -X main.BuildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S')"

.PHONY: all build clean test coverage deps lint fmt check-fmt docker-build docker-run docker-push help

# Default target
all: clean deps test build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all: clean deps
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	
	# Windows
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	
	# macOS
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	
	@echo "Multi-platform build complete"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...
	@echo "Tests complete"

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	@echo "Dependencies downloaded"

# Tidy dependencies
deps-tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy
	@echo "Dependencies tidied"

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping linting"; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...
	@echo "Code formatting complete"

# Fail if any tracked Go source is not gofmt-clean (RD-031)
check-fmt:
	@echo "Checking gofmt on tracked Go files..."
	@files=$$(git ls-files '*.go' | grep -v '^vendor/' || true); \
	if [ -z "$$files" ]; then echo "No tracked Go files"; exit 0; fi; \
	unformatted=$$(echo "$$files" | xargs -n 200 gofmt -s -l); \
	if [ -n "$$unformatted" ]; then \
		echo "The following tracked Go files need gofmt -s:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi; \
	echo "gofmt-clean: $$(echo "$$files" | wc -l) tracked files"

.PHONY: all build clean test coverage deps lint fmt check-fmt docker-build docker-run docker-push help

# Install the application
install: build
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install .
	@echo "Installation complete"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run in development mode
dev:
	@echo "Running in development mode..."
	$(GOCMD) run .

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "Docker build complete"

docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-push:
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

# Docker Compose targets
compose-up:
	@echo "Starting services with Docker Compose..."
	docker-compose -f docker-compose.yml up -d

compose-down:
	@echo "Stopping services with Docker Compose..."
	docker-compose -f docker-compose.yml down

compose-logs:
	@echo "Showing Docker Compose logs..."
	docker-compose -f docker-compose.yml logs -f

deploy:
	@./deploy.sh

# Development helpers
watch:
	@echo "Watching for changes..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not found, install with: go install github.com/cosmtrek/air@latest"; \
		echo "or use: make dev"; \
	fi

# Generate documentation
docs:
	@echo "Generating documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		godoc -http=:6060; \
	else \
		echo "godoc not found, install with: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# Security scanning
security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found, install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Performance profiling
profile:
	@echo "Running performance profile..."
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Run the application and use pprof for profiling:"
	@echo "go tool pprof http://127.0.0.1:8080/debug/pprof/profile"

# Release preparation
release: clean deps test build-all
	@echo "Preparing release..."
	@mkdir -p release
	@cp $(BUILD_DIR)/* release/
	@cp config/config.yaml release/
	@cp docker-compose.yml docker-compose.minimal.yml docker-compose.offline.yml release/
	@cp docker-compose.offline.yml release/
	@cp .env.example release/
	@cp README.md release/
	@echo "Release prepared in release/ directory"

beta-release:
	@chmod +x scripts/build-beta-release.sh
	@./scripts/build-beta-release.sh

clean-beta-release:
	@chmod +x scripts/clean-beta-release.sh
	@./scripts/clean-beta-release.sh

# Help
help:
	@echo "Available targets:"
	@echo "  all          - Build for all platforms"
	@echo "  build        - Build binary"
	@echo "  test         - Run tests"
	@echo "  beta-release - Build prepackaged beta artifacts locally"
	@echo "  docker-build - Build Docker image"
	@echo "  release      - Prepare release directory"

.PHONY: all build clean test coverage deps lint fmt docker-build docker-run docker-push help release beta-release clean-beta-release
	@echo "Available targets:"
	@echo "  all          - Clean, deps, test, and build"
	@echo "  build        - Build the application"
	@echo "  build-all    - Build for multiple platforms"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  coverage     - Run tests with coverage"
	@echo "  deps         - Download dependencies"
	@echo "  deps-tidy    - Tidy dependencies"
	@echo "  lint         - Run linter"
	@echo "  fmt          - Format code"
	@echo "  install      - Install the application"
	@echo "  run          - Build and run the application"
	@echo "  dev          - Run in development mode"
	@echo "  watch        - Watch for changes (requires air)"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  docker-push  - Push Docker image"
	@echo "  compose-up   - Start services with Docker Compose"
	@echo "  compose-down - Stop services with Docker Compose"
	@echo "  compose-logs - Show Docker Compose logs"
	@echo "  docs         - Generate documentation (requires godoc)"
	@echo "  security     - Run security scan (requires gosec)"
	@echo "  profile      - Prepare for performance profiling"
	@echo "  release      - Prepare release package"
	@echo "  help         - Show this help message"

# Default target
.DEFAULT_GOAL := help
