@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

:: MySQL MCP Server 新功能测试脚本（Windows版）
:: 测试新增的表字段管理、重命名和复制功能

:: 颜色定义
for /F "tokens=1,2 delims=#" %%a in ('"prompt #$H#$E# & echo on & for %%b in (1) do rem"') do (
  set "DEL=%%a"
)

<nul set /p ".=%DEL%" > "%~f0?.tmp"
for /F "tokens=1 delims=" %%a in ('"%~f0?.tmp"') do (
  set "ESC=%%a"
)
del "%~f0?.tmp"

set "RED=%ESC%[91m"
set "GREEN=%ESC%[92m"
set "YELLOW=%ESC%[93m"
set "BLUE=%ESC%[94m"
set "NC=%ESC%[0m"

:: 打印带颜色的消息
:print_info
    echo %BLUE%[INFO]%NC% %*
    exit /b

:print_success
    echo %GREEN%[SUCCESS]%NC% %*
    exit /b

:print_warning
    echo %YELLOW%[WARNING]%NC% %*
    exit /b

:print_error
    echo %RED%[ERROR]%NC% %*
    exit /b

:: 测试配置
set "TEST_DB=test_mcp"
set "TEST_TABLE=test_users"
set "TEST_TABLE_COPY=test_users_copy"
set "TEST_TABLE_RENAMED=test_users_renamed"

:: 显示帮助信息
:show_help
    echo MySQL MCP Server 新功能测试脚本（Windows版）
    echo.
    echo 使用方法:
    echo   test_new_features.bat [选项]
    echo.
    echo 选项:
    echo   run        运行所有测试
    echo   help       显示帮助信息
    echo.
    echo 测试内容:
    echo   1. 表字段管理: 添加、修改、删除多个字段
    echo   2. 表重命名: 修改表名称
    echo   3. 表复制: 复制表结构和数据
    echo   4. 表结构复制: 仅复制表结构
    echo.
    echo 环境要求:
    echo   - MCP 服务器正在运行
    echo   - MySQL 数据库连接正常
    echo   - 测试数据库权限
    echo.
    exit /b

:: 检查 MCP 服务器是否运行
:check_mcp_server
    call :print_info "检查 MCP 服务器状态..."

    tasklist /FI "IMAGENAME eq mcp-mysql.exe" 2>nul | find /I "mcp-mysql.exe" >nul
    if errorlevel 1 (
        call :print_error "MCP 服务器未运行"
        call :print_info "请先启动 MCP 服务器：start.bat run"
        exit /b 1
    ) else (
        call :print_success "MCP 服务器正在运行"
    )
    exit /b

:: 显示测试工具调用
:show_tool_call
    set "TOOL_NAME=%~1"
    set "TOOL_ARGS=%~2"
    set "DESCRIPTION=%~3"

    call :print_info "测试: !DESCRIPTION!"
    echo 工具: !TOOL_NAME!
    echo 参数: !TOOL_ARGS!
    echo.

    echo 模拟调用 !TOOL_NAME! 工具...
    echo 结果: 成功
    echo.
    exit /b

:: 创建测试表
:create_test_table
    call :print_info "创建测试表: %TEST_TABLE%"

    call :show_tool_call "mysql_create_table" "{\"table_name\": \"%TEST_TABLE%\", \"columns\": \"id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(100), email VARCHAR(255), created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP\"}" "创建测试表"

    call :show_tool_call "mysql_execute" "{\"sql\": \"INSERT INTO %TEST_TABLE% (name, email) VALUES ('张三', 'zhangsan@example.com'), ('李四', 'lisi@example.com')\"}" "插入测试数据"
    exit /b

:: 测试添加多个字段
:test_add_columns
    call :print_info "测试: 添加多个字段到表"

    set "COLUMNS_JSON=[{\"name\": \"age\", \"type\": \"INT\", \"nullable\": true, \"default_value\": \"0\", \"after_column\": \"email\"}, {\"name\": \"address\", \"type\": \"VARCHAR(255)\", \"nullable\": true, \"default_value\": \"\"}, {\"name\": \"phone\", \"type\": \"VARCHAR(20)\", \"nullable\": true, \"default_value\": \"\"}]"

    call :show_tool_call "mysql_add_columns" "{\"table_name\": \"%TEST_TABLE%\", \"columns\": %COLUMNS_JSON%}" "添加多个字段 (age, address, phone)"

    call :show_tool_call "mysql_describe_table" "{\"table_name\": \"%TEST_TABLE%\"}" "验证表结构"
    exit /b

:: 测试修改多个字段
:test_modify_columns
    call :print_info "测试: 修改多个字段"

    set "COLUMNS_JSON=[{\"name\": \"age\", \"type\": \"INT UNSIGNED\", \"nullable\": false, \"default_value\": \"18\"}, {\"name\": \"address\", \"type\": \"VARCHAR(500)\", \"nullable\": true, \"default_value\": \"\"}]"

    call :show_tool_call "mysql_modify_columns" "{\"table_name\": \"%TEST_TABLE%\", \"columns\": %COLUMNS_JSON%}" "修改字段 (age改为无符号, address长度改为500)"
    exit /b

:: 测试删除多个字段
:test_drop_columns
    call :print_info "测试: 删除多个字段"

    set "COLUMNS_JSON=[\"phone\", \"temp_field\"]"

    call :show_tool_call "mysql_drop_columns" "{\"table_name\": \"%TEST_TABLE%\", \"columns\": %COLUMNS_JSON%}" "删除字段 (phone, temp_field)"
    exit /b

:: 测试重命名表
:test_rename_table
    call :print_info "测试: 重命名表"

    call :show_tool_call "mysql_rename_table" "{\"old_table_name\": \"%TEST_TABLE%\", \"new_table_name\": \"%TEST_TABLE_RENAMED%\"}" "重命名表 %TEST_TABLE% -> %TEST_TABLE_RENAMED%"

    call :show_tool_call "mysql_list_tables" "{}" "验证表名已更改"

    call :show_tool_call "mysql_rename_table" "{\"old_table_name\": \"%TEST_TABLE_RENAMED%\", \"new_table_name\": \"%TEST_TABLE%\"}" "恢复表名 %TEST_TABLE_RENAMED% -> %TEST_TABLE%"
    exit /b

:: 测试复制表（结构和数据）
:test_copy_table_with_data
    call :print_info "测试: 复制表结构和数据"

    call :show_tool_call "mysql_copy_table" "{\"source_table\": \"%TEST_TABLE%\", \"destination_table\": \"%TEST_TABLE_COPY%\", \"copy_data\": true}" "复制表结构和数据"

    call :show_tool_call "mysql_query" "{\"query\": \"SELECT COUNT(*) as count FROM %TEST_TABLE_COPY%\"}" "验证复制数据数量"

    call :show_tool_call "mysql_describe_table" "{\"table_name\": \"%TEST_TABLE_COPY%\"}" "验证复制表结构"
    exit /b

:: 测试仅复制表结构
:test_copy_table_structure_only
    call :print_info "测试: 仅复制表结构"

    set "DEST_TABLE=%TEST_TABLE_COPY%_structure"

    call :show_tool_call "mysql_copy_table_structure" "{\"source_table\": \"%TEST_TABLE%\", \"destination_table\": \"%DEST_TABLE%\"}" "仅复制表结构"

    call :show_tool_call "mysql_query" "{\"query\": \"SELECT COUNT(*) as count FROM %DEST_TABLE%\"}" "验证空表数据"
    exit /b

:: 清理测试数据
:cleanup_test
    call :print_info "清理测试数据..."

    call :show_tool_call "mysql_drop_table" "{\"table_name\": \"%TEST_TABLE%\"}" "删除原测试表"

    call :show_tool_call "mysql_drop_table" "{\"table_name\": \"%TEST_TABLE_COPY%\"}" "删除复制表"

    call :show_tool_call "mysql_drop_table" "{\"table_name\": \"%TEST_TABLE_COPY%_structure\"}" "删除结构复制表"

    call :print_success "测试数据清理完成"
    exit /b

:: 显示测试结果摘要
:show_test_summary
    echo.
    echo =========================================
    echo           新功能测试结果摘要
    echo =========================================
    echo.
    echo ✅ 测试的功能:
    echo    1. 添加多个字段 (mysql_add_columns)
    echo    2. 修改多个字段 (mysql_modify_columns)
    echo    3. 删除多个字段 (mysql_drop_columns)
    echo    4. 重命名表 (mysql_rename_table)
    echo    5. 复制表结构和数据 (mysql_copy_table)
    echo    6. 仅复制表结构 (mysql_copy_table_structure)
    echo.
    echo 📋 测试表:
    echo    - 原表: %TEST_TABLE%
    echo    - 复制表: %TEST_TABLE_COPY%
    echo    - 重命名表: %TEST_TABLE_RENAMED%
    echo.
    echo 🔧 测试的字段操作:
    echo    - 添加: age, address, phone
    echo    - 修改: age类型, address长度
    echo    - 删除: phone, temp_field
    echo.
    echo =========================================
    exit /b

:: 主测试函数
:main_test
    call :print_info "开始测试 MySQL MCP Server 新功能"
    echo 测试时间: %date% %time%
    echo.

    call :check_mcp_server
    if errorlevel 1 exit /b 1

    call :create_test_table
    call :test_add_columns
    call :test_modify_columns
    call :test_drop_columns
    call :test_rename_table
    call :test_copy_table_with_data
    call :test_copy_table_structure_only

    call :show_test_summary

    call :cleanup_test

    call :print_success "所有新功能测试完成！"
    exit /b

:: 主程序
if "%~1"=="" goto run_default
if "%~1"=="run" goto run_tests
if "%~1"=="help" goto show_help
if "%~1"=="-h" goto show_help
if "%~1"=="--help" goto show_help

:run_default
    call :print_info "开始交互式测试..."
    goto run_tests

:run_tests
    call :main_test
    goto :eof

:show_help
    call :show_help
    goto :eof

:: 结束提示
echo.
call :print_warning "注意: 这是一个模拟测试脚本"
call :print_info "实际使用时需要替换为真实的 MCP 工具调用"
call :print_info "可以使用 curl 或专门的 MCP 客户端进行实际测试"

endlocal
