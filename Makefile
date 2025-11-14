# Kottg Compiler Makefile

# Variables
BINARY_NAME=kottg
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WIN=$(BINARY_NAME).exe
VERSION=1.0.0

# Default target
.PHONY: all
all: build

# Build the compiler
.PHONY: build
build:
	@echo "Building kottg compiler..."
	go build -o $(BINARY_NAME) main.go
	@echo "Build complete: $(BINARY_NAME)"

# Build for Windows
.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_WIN) main.go
	@echo "Build complete: $(BINARY_WIN)"

# Build for Linux
.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_UNIX) main.go
	@echo "Build complete: $(BINARY_UNIX)"

# Build for macOS
.PHONY: build-darwin
build-darwin:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)_darwin main.go
	@echo "Build complete: $(BINARY_NAME)_darwin"

# Build for all platforms
.PHONY: build-all
build-all: build-linux build-windows build-darwin
	@echo "All builds complete"

# Install the compiler globally
.PHONY: install
install:
	@echo "Installing kottg..."
	go install
	@echo "Installation complete"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -cover ./...

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME) $(BINARY_UNIX) $(BINARY_WIN) $(BINARY_NAME)_darwin
	rm -f contracts/*.bin
	@echo "Clean complete"

# Compile example contracts
.PHONY: examples
examples: build
	@echo "Compiling example contracts..."
	./$(BINARY_NAME) -input contracts/StakingQuery.kg -verbose
	@echo "Examples compiled"

# Show example usage
.PHONY: demo
demo: build
	@echo "=== Kottg Compiler Demo ==="
	@echo ""
	@echo "1. Showing version:"
	./$(BINARY_NAME) -version
	@echo ""
	@echo "2. Compiling with AST output:"
	./$(BINARY_NAME) -input contracts/StakingQuery.kg -ast
	@echo ""
	@echo "3. Compiling with disassembly:"
	./$(BINARY_NAME) -input contracts/StakingQuery.kg -disasm

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Format complete"

# Vet code
.PHONY: vet
vet:
	@echo "Vetting code..."
	go vet ./...
	@echo "Vet complete"

# Run linter
.PHONY: lint
lint: fmt vet
	@echo "Linting complete"

# Development setup
.PHONY: setup
setup:
	@echo "Setting up development environment..."
	go mod tidy
	@echo "Setup complete"

# Show help
.PHONY: help
help:
	@echo "Kottg Compiler Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build          Build the compiler for current platform"
	@echo "  build-windows  Build for Windows"
	@echo "  build-linux    Build for Linux"
	@echo "  build-darwin   Build for macOS"
	@echo "  build-all      Build for all platforms"
	@echo "  install        Install globally"
	@echo "  test           Run tests"
	@echo "  test-coverage  Run tests with coverage"
	@echo "  clean          Clean build artifacts"
	@echo "  examples       Compile example contracts"
	@echo "  demo           Show usage demo"
	@echo "  fmt            Format code"
	@echo "  vet            Vet code"
	@echo "  lint           Run linter (fmt + vet)"
	@echo "  setup          Setup development environment"
	@echo "  help           Show this help message"
