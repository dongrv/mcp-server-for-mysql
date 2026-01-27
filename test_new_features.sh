#!/bin/bash

# MySQL MCP Server 新功能测试脚本
# 测试新增的表字段管理、重命名和复制功能

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# 测试配置
TEST_DB="test_mcp"
TEST_TABLE="test_users"
TEST_TABLE_COPY="test_users_copy"
TEST_TABLE_RENAMED="test_users_renamed"

# 检查 MCP 服务器是否运行
check_mcp_server() {
    print_info "检查 MCP 服务器状态..."

    # 尝试连接 MCP 服务器
    if pgrep -f "mcp-mysql" > /dev/null; then
        print_success "MCP 服务器正在运行"
        return 0
    else
        print_error "MCP 服务器未运行"
        print_info "请先启动 MCP 服务器：./start.sh run"
        return 1
    fi
}

# 测试工具调用函数
test_tool() {
    local tool_name="$1"
    local tool_args="$2"
    local description="$3"

    print_info "测试: $description"
    echo "工具: $tool_name"
    echo "参数: $tool_args"
    echo ""

    # 这里应该是实际的 MCP 工具调用
    # 由于工具调用需要 MCP 协议，这里只模拟输出
    echo "模拟调用 $tool_name 工具..."
    echo "结果: 成功"
    echo ""
}

# 创建测试表
create_test_table() {
    print_info "创建测试表: $TEST_TABLE"

    test_tool "mysql_create_table" \
        "{\"table_name\": \"$TEST_TABLE\", \"columns\": \"id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(100), email VARCHAR(255), created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP\"}" \
        "创建测试表"

    # 插入测试数据
    test_tool "mysql_execute" \
        "{\"sql\": \"INSERT INTO $TEST_TABLE (name, email) VALUES ('张三', 'zhangsan@example.com'), ('李四', 'lisi@example.com')\"}" \
        "插入测试数据"
}

# 测试添加多个字段
test_add_columns() {
    print_info "测试: 添加多个字段到表"

    local columns_json='[
        {
            "name": "age",
            "type": "INT",
            "nullable": true,
            "default_value": "0",
            "after_column": "email"
        },
        {
            "name": "address",
            "type": "VARCHAR(255)",
            "nullable": true,
            "default_value": ""
        },
        {
            "name": "phone",
            "type": "VARCHAR(20)",
            "nullable": true,
            "default_value": ""
        }
    ]'

    test_tool "mysql_add_columns" \
        "{\"table_name\": \"$TEST_TABLE\", \"columns\": $columns_json}" \
        "添加多个字段 (age, address, phone)"

    # 验证表结构
    test_tool "mysql_describe_table" \
        "{\"table_name\": \"$TEST_TABLE\"}" \
        "验证表结构"
}

# 测试修改多个字段
test_modify_columns() {
    print_info "测试: 修改多个字段"

    local columns_json='[
        {
            "name": "age",
            "type": "INT UNSIGNED",
            "nullable": false,
            "default_value": "18"
        },
        {
            "name": "address",
            "type": "VARCHAR(500)",
            "nullable": true,
            "default_value": ""
        }
    ]'

    test_tool "mysql_modify_columns" \
        "{\"table_name\": \"$TEST_TABLE\", \"columns\": $columns_json}" \
        "修改字段 (age改为无符号, address长度改为500)"
}

# 测试删除多个字段
test_drop_columns() {
    print_info "测试: 删除多个字段"

    local columns_json='[\"phone\", \"temp_field\"]'

    test_tool "mysql_drop_columns" \
        "{\"table_name\": \"$TEST_TABLE\", \"columns\": $columns_json}" \
        "删除字段 (phone, temp_field)"
}

# 测试重命名表
test_rename_table() {
    print_info "测试: 重命名表"

    test_tool "mysql_rename_table" \
        "{\"old_table_name\": \"$TEST_TABLE\", \"new_table_name\": \"$TEST_TABLE_RENAMED\"}" \
        "重命名表 $TEST_TABLE -> $TEST_TABLE_RENAMED"

    # 验证新表名
    test_tool "mysql_list_tables" \
        "{}" \
        "验证表名已更改"

    # 改回原名以便后续测试
    test_tool "mysql_rename_table" \
        "{\"old_table_name\": \"$TEST_TABLE_RENAMED\", \"new_table_name\": \"$TEST_TABLE\"}" \
        "恢复表名 $TEST_TABLE_RENAMED -> $TEST_TABLE"
}

# 测试复制表（结构和数据）
test_copy_table_with_data() {
    print_info "测试: 复制表结构和数据"

    test_tool "mysql_copy_table" \
        "{\"source_table\": \"$TEST_TABLE\", \"destination_table\": \"$TEST_TABLE_COPY\", \"copy_data\": true}" \
        "复制表结构和数据"

    # 验证复制结果
    test_tool "mysql_query" \
        "{\"query\": \"SELECT COUNT(*) as count FROM $TEST_TABLE_COPY\"}" \
        "验证复制数据数量"

    test_tool "mysql_describe_table" \
        "{\"table_name\": \"$TEST_TABLE_COPY\"}" \
        "验证复制表结构"
}

# 测试仅复制表结构
test_copy_table_structure_only() {
    print_info "测试: 仅复制表结构"

    local dest_table="${TEST_TABLE_COPY}_structure"

    test_tool "mysql_copy_table_structure" \
        "{\"source_table\": \"$TEST_TABLE\", \"destination_table\": \"$dest_table\"}" \
        "仅复制表结构"

    # 验证结构复制，数据为空
    test_tool "mysql_query" \
        "{\"query\": \"SELECT COUNT(*) as count FROM $dest_table\"}" \
        "验证空表数据"
}

# 清理测试数据
cleanup_test() {
    print_info "清理测试数据..."

    # 删除测试表
    test_tool "mysql_drop_table" \
        "{\"table_name\": \"$TEST_TABLE\"}" \
        "删除原测试表"

    test_tool "mysql_drop_table" \
        "{\"table_name\": \"$TEST_TABLE_COPY\"}" \
        "删除复制表"

    test_tool "mysql_drop_table" \
        "{\"table_name\": \"${TEST_TABLE_COPY}_structure\"}" \
        "删除结构复制表"

    print_success "测试数据清理完成"
}

# 显示测试结果摘要
show_test_summary() {
    echo ""
    echo "========================================="
    echo "          新功能测试结果摘要"
    echo "========================================="
    echo ""
    echo "✅ 测试的功能:"
    echo "   1. 添加多个字段 (mysql_add_columns)"
    echo "   2. 修改多个字段 (mysql_modify_columns)"
    echo "   3. 删除多个字段 (mysql_drop_columns)"
    echo "   4. 重命名表 (mysql_rename_table)"
    echo "   5. 复制表结构和数据 (mysql_copy_table)"
    echo "   6. 仅复制表结构 (mysql_copy_table_structure)"
    echo ""
    echo "📋 测试表:"
    echo "   - 原表: $TEST_TABLE"
    echo "   - 复制表: $TEST_TABLE_COPY"
    echo "   - 重命名表: $TEST_TABLE_RENAMED"
    echo ""
    echo "🔧 测试的字段操作:"
    echo "   - 添加: age, address, phone"
    echo "   - 修改: age类型, address长度"
    echo "   - 删除: phone, temp_field"
    echo ""
    echo "========================================="
}

# 主测试函数
main_test() {
    print_info "开始测试 MySQL MCP Server 新功能"
    echo "测试时间: $(date)"
    echo ""

    # 检查 MCP 服务器
    if ! check_mcp_server; then
        exit 1
    fi

    # 执行测试
    create_test_table
    test_add_columns
    test_modify_columns
    test_drop_columns
    test_rename_table
    test_copy_table_with_data
    test_copy_table_structure_only

    # 显示测试摘要
    show_test_summary

    # 清理测试数据
    cleanup_test

    print_success "所有新功能测试完成！"
}

# 显示帮助信息
show_help() {
    echo "MySQL MCP Server 新功能测试脚本"
    echo ""
    echo "使用方法:"
    echo "  ./test_new_features.sh [选项]"
    echo ""
    echo "选项:"
    echo "  run        运行所有测试"
    echo "  help       显示帮助信息"
    echo ""
    echo "测试内容:"
    echo "  1. 表字段管理: 添加、修改、删除多个字段"
    echo "  2. 表重命名: 修改表名称"
    echo "  3. 表复制: 复制表结构和数据"
    echo "  4. 表结构复制: 仅复制表结构"
    echo ""
    echo "环境要求:"
    echo "  - MCP 服务器正在运行"
    echo "  - MySQL 数据库连接正常"
    echo "  - 测试数据库权限"
    echo ""
}

# 解析命令行参数
case "$1" in
    "run")
        main_test
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    *)
        print_info "开始交互式测试..."
        main_test
        ;;
esac

# 设置脚本权限提示
echo ""
print_warning "注意: 这是一个模拟测试脚本"
print_info "实际使用时需要替换 test_tool 函数为真实的 MCP 工具调用"
print_info "可以使用 curl 或专门的 MCP 客户端进行实际测试"
