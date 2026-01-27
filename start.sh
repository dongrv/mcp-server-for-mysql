#!/bin/bash

# MySQL MCP Server 快速启动脚本
# 版本: 1.0.0

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目信息
PROJECT_NAME="MySQL MCP Server"
VERSION="1.0.0"
BINARY_NAME="mcp-mysql"
BUILD_DIR="build"
ENV_FILE=".env"

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示帮助信息
show_help() {
    echo "MySQL MCP Server 快速启动脚本"
    echo ""
    echo "使用方法:"
    echo "  ./start.sh [命令]"
    echo ""
    echo "命令:"
    echo "  build       构建项目"
    echo "  run         运行项目"
    echo "  setup       设置环境"
    echo "  clean       清理构建文件"
    echo "  test        运行测试"
    echo "  docker      使用 Docker 运行"
    echo "  help        显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  ./start.sh build     # 构建项目"
    echo "  ./start.sh run       # 运行项目"
    echo "  ./start.sh setup     # 设置环境变量"
    echo ""
}

# 检查依赖
check_dependencies() {
    print_info "检查依赖..."

    # 检查 Go
    if ! command -v go &> /dev/null; then
        print_error "Go 未安装，请先安装 Go (>= 1.23)"
        exit 1
    fi

    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    if [[ "$(printf '%s\n' "1.23" "$GO_VERSION" | sort -V | head -n1)" != "1.23" ]]; then
        print_warning "Go 版本 $GO_VERSION 可能低于 1.23，建议升级"
    fi

    # 检查 MySQL 客户端（可选）
    if command -v mysql &> /dev/null; then
        print_success "MySQL 客户端已安装"
    else
        print_warning "MySQL 客户端未安装，某些功能可能受限"
    fi

    print_success "依赖检查完成"
}

# 设置环境
setup_environment() {
    print_info "设置环境..."

    if [ ! -f "$ENV_FILE" ]; then
        if [ -f "env.example" ]; then
            cp env.example "$ENV_FILE"
            print_success "已创建环境配置文件: $ENV_FILE"
            print_warning "请编辑 $ENV_FILE 文件以配置数据库连接"
        else
            print_error "找不到 env.example 文件"
            exit 1
        fi
    else
        print_info "环境配置文件已存在: $ENV_FILE"
    fi

    # 加载环境变量
    if [ -f "$ENV_FILE" ]; then
        set -a
        source "$ENV_FILE"
        set +a
        print_success "环境变量已加载"
    fi

    # 检查必要的环境变量
    local required_vars=("MYSQL_HOST" "MYSQL_USER" "MYSQL_DATABASE")
    local missing_vars=()

    for var in "${required_vars[@]}"; do
        if [ -z "${!var}" ]; then
            missing_vars+=("$var")
        fi
    done

    if [ ${#missing_vars[@]} -gt 0 ]; then
        print_warning "以下环境变量未设置: ${missing_vars[*]}"
        print_warning "请在 $ENV_FILE 文件中设置这些变量"
    fi

    print_success "环境设置完成"
}

# 构建项目
build_project() {
    print_info "构建项目..."

    check_dependencies

    # 创建构建目录
    mkdir -p "$BUILD_DIR"

    # 构建
    if go build -o "$BUILD_DIR/$BINARY_NAME" ./cmd; then
        print_success "项目构建成功: $BUILD_DIR/$BINARY_NAME"

        # 显示构建信息
        echo ""
        echo "构建信息:"
        echo "  项目: $PROJECT_NAME"
        echo "  版本: $VERSION"
        echo "  二进制: $BUILD_DIR/$BINARY_NAME"
        echo "  Go 版本: $(go version)"
        echo ""
    else
        print_error "项目构建失败"
        exit 1
    fi
}

# 运行项目
run_project() {
    print_info "启动 MySQL MCP Server..."

    # 检查是否已构建
    if [ ! -f "$BUILD_DIR/$BINARY_NAME" ]; then
        print_warning "未找到构建文件，正在构建..."
        build_project
    fi

    setup_environment

    # 显示配置信息
    echo ""
    echo "服务器配置:"
    echo "  主机: ${MYSQL_HOST:-未设置}"
    echo "  端口: ${MYSQL_PORT:-3306}"
    echo "  数据库: ${MYSQL_DATABASE:-未设置}"
    echo "  用户: ${MYSQL_USER:-未设置}"
    echo ""

    print_info "按 Ctrl+C 停止服务器"
    echo ""

    # 运行服务器
    "$BUILD_DIR/$BINARY_NAME"
}

# 清理构建文件
clean_project() {
    print_info "清理构建文件..."

    if [ -d "$BUILD_DIR" ]; then
        rm -rf "$BUILD_DIR"
        print_success "已清理构建目录: $BUILD_DIR"
    else
        print_info "构建目录不存在: $BUILD_DIR"
    fi

    # 清理 Go 缓存
    go clean -cache
    print_success "Go 缓存已清理"
}

# 运行测试
run_tests() {
    print_info "运行测试..."

    check_dependencies

    if go test ./...; then
        print_success "测试通过"
    else
        print_error "测试失败"
        exit 1
    fi
}

# 使用 Docker 运行
run_docker() {
    print_info "使用 Docker 运行..."

    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安装"
        exit 1
    fi

    # 检查 Dockerfile
    if [ ! -f "Dockerfile" ]; then
        print_error "找不到 Dockerfile"
        exit 1
    fi

    # 构建 Docker 镜像
    print_info "构建 Docker 镜像..."
    if docker build -t mcp-mysql:latest .; then
        print_success "Docker 镜像构建成功"
    else
        print_error "Docker 镜像构建失败"
        exit 1
    fi

    # 运行 Docker 容器
    print_info "启动 Docker 容器..."
    echo ""
    echo "运行命令:"
    echo "  docker run --rm -it \\"
    echo "    -e MYSQL_HOST=host.docker.internal \\"
    echo "    -e MYSQL_PORT=3306 \\"
    echo "    -e MYSQL_USER=root \\"
    echo "    -e MYSQL_PASSWORD=your_password \\"
    echo "    -e MYSQL_DATABASE=test \\"
    echo "    mcp-mysql:latest"
    echo ""

    print_warning "请根据实际情况修改环境变量"
}

# 主函数
main() {
    local command=${1:-"help"}

    case "$command" in
        "build")
            build_project
            ;;
        "run")
            run_project
            ;;
        "setup")
            setup_environment
            ;;
        "clean")
            clean_project
            ;;
        "test")
            run_tests
            ;;
        "docker")
            run_docker
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            print_error "未知命令: $command"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
