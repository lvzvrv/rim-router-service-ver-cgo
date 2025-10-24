# ===============================
# Makefile for rim-router-service-ver-cgo
# ===============================

APP_NAME := router-service
BUILD_DIR := build
LOG_DIR := logs
TEST_LOG_DIR := internal/handlers/tir_logs
OUTPUT := $(BUILD_DIR)/$(APP_NAME)
SRC := ./cmd/server
GO_FLAGS := -ldflags="-s -w" -trimpath
UPX_FLAGS := --best --lzma
GO := go

# Detect OS (for upx binary naming, if needed later)
UNAME_S := $(shell uname -s)

# -------------------------------
# Default build (optimized, small binary)
# -------------------------------
.PHONY: build
build:
	@echo "🚀 Building optimized binary..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GO_FLAGS) -o $(OUTPUT) $(SRC)
	@echo "✅ Build complete: $(OUTPUT)"
	@ls -lh $(OUTPUT)

# -------------------------------
# Compress binary with UPX
# -------------------------------
.PHONY: upx
upx: build
	@echo "📦 Compressing binary with UPX..."
	@upx $(UPX_FLAGS) $(OUTPUT)
	@echo "✅ Compression complete:"
	@ls -lh $(OUTPUT)

# -------------------------------
# Run the server directly
# -------------------------------
.PHONY: run
run:
	@echo "🏃 Running $(APP_NAME)..."
	@$(GO) run $(SRC)

# -------------------------------
# Clean build artifacts and logs
# -------------------------------
.PHONY: clean
clean:
	@echo "🧹 Cleaning build artifacts and logs..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(LOG_DIR)
	@rm -rf $(TEST_LOG_DIR)
	@echo "✅ Clean complete."

# -------------------------------
# Run all Go tests and cleanup logs
# -------------------------------
.PHONY: test
test:
	@echo "🧪 Running all tests..."
	@$(GO) test ./... -v
	@echo "🧹 Removing temporary test logs..."
	@if [ -d "$(TEST_LOG_DIR)" ]; then rm -rf $(TEST_LOG_DIR); fi
	@echo "✅ All tests completed successfully and logs cleaned."

# -------------------------------
# Run tests with cache cleared first
# -------------------------------
.PHONY: test-clean
test-clean:
	@echo "♻️  Cleaning Go test cache..."
	@$(GO) clean -testcache
	@echo "🧪 Running fresh tests..."
	@$(GO) test ./... -v
	@echo "🧹 Removing temporary test logs..."
	@if [ -d "$(TEST_LOG_DIR)" ]; then rm -rf $(TEST_LOG_DIR); fi
	@echo "✅ Fresh tests completed successfully and logs cleaned."

# -------------------------------
# Cross-compile for Linux (CGO)
# -------------------------------
.PHONY: linux
linux:
	@echo "🐧 Cross-compiling for Linux amd64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=1 $(GO) build $(GO_FLAGS) -o $(OUTPUT)-linux $(SRC)
	@ls -lh $(OUTPUT)-linux
