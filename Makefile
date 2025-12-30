.PHONY: build run clean test fmt vet help

# Variables
APP?=gateway
BINARY_NAME=$(APP)
BINARY_DIR=bin
CMD_DIR=cmd/$(APP)
MAIN_FILE=$(CMD_DIR)/main.go

# Default target
.DEFAULT_GOAL := help

## build: Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@if not exist $(BINARY_DIR) mkdir $(BINARY_DIR)
	@go build -o $(BINARY_DIR)/$(BINARY_NAME).exe $(MAIN_FILE)
	@echo "Build completed: $(BINARY_DIR)/$(BINARY_NAME).exe"

## run: Run the application directly (without building)
run:
	@echo "Running $(BINARY_NAME)..."
	@go run $(MAIN_FILE)

## build-run: Build and run the application
build-run: build
	@echo "Running $(BINARY_NAME)..."
	@$(BINARY_DIR)/$(BINARY_NAME).exe

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@if exist $(BINARY_DIR) rd /s /q $(BINARY_DIR)
	@echo "Clean completed"

## test: Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format completed"

## vet: Run static code analysis
vet:
	@echo "Running static analysis..."
	@go vet ./...
	@echo "Analysis completed"

## tidy: Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy
	@echo "Dependencies tidied"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@echo "Dependencies downloaded"

## all: Format, analyze, test, and build
all: fmt vet test build

## help: Show help information
help:
	@echo "Available commands:"
	@echo "  make build [APP=gateway]     - Build the application"
	@echo "  make run [APP=gateway]       - Run the application directly"
	@echo "  make build-run [APP=gateway] - Build and run the application"
	@echo "  make clean                   - Clean build artifacts"
	@echo "  make test                    - Run tests"
	@echo "  make fmt                     - Format code"
	@echo "  make vet                     - Run static code analysis"
	@echo "  make tidy                    - Tidy dependencies"
	@echo "  make deps                    - Download dependencies"
	@echo "  make all                     - Format, analyze, test, and build"
	@echo "  make help                    - Show this help information"
	@echo ""
	@echo "Examples:"
	@echo "  make build              - Build gateway (default)"
	@echo "  make build APP=gateway  - Build gateway application"
	@echo "  make run APP=myapp      - Run myapp application"
