.PHONY: all build run test clean fmt lint docker-build docker-run help

# 项目信息
PROJECT_NAME := mcp-database
BINARY_NAME := mcp-database
VERSION := 1.0.0
BUILD_TIME := $(shell date +%Y%m%d%H%M%S)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go 配置
GO := go
GO_BUILD := $(GO) build
GO_TEST := $(GO) test
GO_MOD := $(GO) mod
GO_FMT := $(GO) fmt
GO_LINT := golangci-lint run

# 目录配置
CMD_DIR := cmd
BUILD_DIR := build
DIST_DIR := dist

# 构建标志
BUILD_FLAGS := -trimpath
CONFIG ?= config.yaml
ENV_FILE ?= .env

# 默认目标
all: build

# 构建项目
build:
	@echo "Building $(PROJECT_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# 运行项目
run:
	@echo "Running $(PROJECT_NAME)..."
	$(GO) run ./$(CMD_DIR) -config $(CONFIG)

# 运行测试
test:
	@echo "Running tests..."
	$(GO_TEST) -v ./...

# 运行集成测试（需要 MySQL）
test-integration:
	@echo "Running integration tests..."
	$(GO_TEST) -tags=integration -v ./...

# 清理构建文件
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	$(GO) clean -cache
	@echo "Clean complete"

# 格式化代码
fmt:
	@echo "Formatting code..."
	$(GO_FMT) ./...

# 代码检查
lint:
	@echo "Running linter..."
	$(if $(shell which golangci-lint),,$(error "golangci-lint not found, install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"))
	$(GO_LINT) ./...

# 安装依赖
deps:
	@echo "Installing dependencies..."
	$(GO_MOD) download
	$(GO_MOD) tidy

# 更新依赖
update-deps:
	@echo "Updating dependencies..."
	$(GO_MOD) download
	$(GO_MOD) tidy -v

# 生成版本信息
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

# 创建发布包
dist: build
	@echo "Creating distribution package..."
	@mkdir -p $(DIST_DIR)
	cp $(BUILD_DIR)/$(BINARY_NAME) $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-$(shell uname -s | tr '[:upper:]' '[:lower:]')-$(shell uname -m)
	@echo "Distribution package created in $(DIST_DIR)"

# Docker 构建
docker-build:
	@echo "Building Docker image..."
	docker build -t $(PROJECT_NAME):$(VERSION) .
	docker tag $(PROJECT_NAME):$(VERSION) $(PROJECT_NAME):latest

# Docker 运行
docker-run:
	@echo "Running Docker container..."
	docker run --rm -i \
		-v "$(abspath $(CONFIG)):/app/config.yaml:ro" \
		--env-file $(ENV_FILE) \
		$(PROJECT_NAME):latest

# 显示帮助信息
help:
	@echo "Available targets:"
	@echo "  all              - Build the project (default)"
	@echo "  build            - Build the binary"
	@echo "  run              - Run the project"
	@echo "  test             - Run unit tests"
	@echo "  test-integration - Run integration tests (requires MySQL)"
	@echo "  clean            - Clean build artifacts"
	@echo "  fmt              - Format source code"
	@echo "  lint             - Run linter"
	@echo "  deps             - Install dependencies"
	@echo "  update-deps      - Update dependencies"
	@echo "  version          - Show version information"
	@echo "  dist             - Create distribution package"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-run       - Run Docker container"
	@echo "  help             - Show this help message"

# 显示项目信息
info:
	@echo "Project: $(PROJECT_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Binary: $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "Go Version: $(shell go version)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
