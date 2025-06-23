# Makefile for smtp-cli cross-compilation

# Binary name
BINARY_NAME = smtp-cli

# Source files
MAIN = main.go

# Go compiler
GO = $(shell which go 2>/dev/null || echo $(HOME)/go-install/go/bin/go)

# Build flags
LDFLAGS = -s -w
BUILDFLAGS = -trimpath

# Ensure Go modules are used
export GO111MODULE = on

# Default target
all: clean build-all

# Build all targets
build-all: \
	build-windows-amd64 \
	build-windows-arm64 \
	build-darwin-amd64 \
	build-darwin-arm64 \
	build-linux-amd64 \
	build-linux-arm64

# Windows Intel (AMD64)
build-windows-amd64:
	@echo "Building for Windows AMD64..."
	GOOS=windows GOARCH=amd64 $(GO) build $(BUILDFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)-windows-amd64.exe $(MAIN)

# Windows ARM64
build-windows-arm64:
	@echo "Building for Windows ARM64..."
	GOOS=windows GOARCH=arm64 $(GO) build $(BUILDFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)-windows-arm64.exe $(MAIN)

# macOS Intel (AMD64)
build-darwin-amd64:
	@echo "Building for macOS AMD64..."
	GOOS=darwin GOARCH=amd64 $(GO) build $(BUILDFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)-darwin-amd64 $(MAIN)

# macOS ARM64 (Apple Silicon)
build-darwin-arm64:
	@echo "Building for macOS ARM64..."
	GOOS=darwin GOARCH=arm64 $(GO) build $(BUILDFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)-darwin-arm64 $(MAIN)

# Linux Intel (AMD64)
build-linux-amd64:
	@echo "Building for Linux AMD64..."
	GOOS=linux GOARCH=amd64 $(GO) build $(BUILDFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)-linux-amd64 $(MAIN)

# Linux ARM64
build-linux-arm64:
	@echo "Building for Linux ARM64..."
	GOOS=linux GOARCH=arm64 $(GO) build $(BUILDFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)-linux-arm64 $(MAIN)

# Build for current platform only
build:
	@echo "Building for current platform..."
	$(GO) build $(BUILDFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(MAIN)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@rm -f $(BINARY_NAME)-*

# List all built binaries
list:
	@echo "Built binaries:"
	@ls -la $(BINARY_NAME)-* 2>/dev/null || echo "No binaries built yet"

# Create a release directory with all binaries
release: clean build-all
	@echo "Creating release directory..."
	@mkdir -p release
	@mv $(BINARY_NAME)-* release/
	@echo "Release binaries created in ./release/"
	@ls -la release/

# Compress all binaries for distribution
compress: release
	@echo "Compressing binaries..."
	@cd release && for file in $(BINARY_NAME)-*; do \
		if [ -f "$$file" ]; then \
			echo "Compressing $$file..."; \
			if command -v zip >/dev/null 2>&1; then \
				zip "$$file.zip" "$$file"; \
			elif command -v gzip >/dev/null 2>&1; then \
				gzip -c "$$file" > "$$file.gz"; \
			else \
				echo "No compression tool found (zip or gzip)"; \
			fi; \
		fi; \
	done
	@echo "Compressed binaries created in ./release/"

# Test the build for current platform
test: build
	./$(BINARY_NAME) --version

# Show help
help:
	@echo "smtp-cli Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  all              - Clean and build for all platforms (default)"
	@echo "  build            - Build for current platform only"
	@echo "  build-all        - Build for all supported platforms"
	@echo "  clean            - Remove all built binaries"
	@echo "  list             - List all built binaries"
	@echo "  release          - Build all and move to release directory"
	@echo "  compress         - Create compressed archives of all binaries"
	@echo "  test             - Build and test current platform binary"
	@echo ""
	@echo "Individual platform targets:"
	@echo "  build-windows-amd64  - Build for Windows Intel/AMD64"
	@echo "  build-windows-arm64  - Build for Windows ARM64"
	@echo "  build-darwin-amd64   - Build for macOS Intel"
	@echo "  build-darwin-arm64   - Build for macOS Apple Silicon"
	@echo "  build-linux-amd64    - Build for Linux Intel/AMD64"
	@echo "  build-linux-arm64    - Build for Linux ARM64"
	@echo ""
	@echo "The binaries will be named:"
	@echo "  smtp-cli-windows-amd64.exe  (Windows Intel)"
	@echo "  smtp-cli-windows-arm64.exe  (Windows ARM)"
	@echo "  smtp-cli-darwin-amd64       (macOS Intel)"
	@echo "  smtp-cli-darwin-arm64       (macOS ARM/Apple Silicon)"
	@echo "  smtp-cli-linux-amd64        (Linux Intel)"
	@echo "  smtp-cli-linux-arm64        (Linux ARM)"

.PHONY: all build build-all clean list release compress test help \
	build-windows-amd64 build-windows-arm64 \
	build-darwin-amd64 build-darwin-arm64 \
	build-linux-amd64 build-linux-arm64