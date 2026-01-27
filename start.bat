@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

:: MySQL MCP Server Windows 启动脚本
:: 版本: 1.0.0

:: 颜色定义
set "RED=[91m"
set "GREEN=[92m"
set "YELLOW=[93m"
set "BLUE=[94m"
set "NC=[0m"

:: 项目信息
set "PROJECT_NAME=MySQL MCP Server"
set "VERSION=1.0.0"
set "BINARY_NAME=mcp-mysql.exe"
set "BUILD_DIR=build"
set "ENV_FILE=.env"

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

:: 显示帮助信息
:show_help
    echo MySQL MCP Server Windows 启动脚本
    echo.
    echo 使用方法:
    echo   start.bat [命令]
    echo.
    echo 命令:
    echo   build       构建项目
    echo   run         运行项目
    echo   setup       设置环境
    echo   clean       清理构建文件
    echo   test        运行测试
    echo   help        显示此帮助信息
    echo.
    echo 示例:
    echo   start.bat build     ^> 构建项目
    echo   start.bat run       ^> 运行项目
    echo   start.bat setup     ^> 设置环境变量
    echo.
    exit /b

:: 检查依赖
:check_dependencies
    call :print_info "检查依赖..."

    :: 检查 Go
    where go >nul 2>&1
    if errorlevel 1 (
        call :print_error "Go 未安装，请先安装 Go (^>= 1.23)"
        exit /b 1
    )

    :: 获取 Go 版本
    for /f "tokens=3" %%i in ('go version') do (
        set "GO_VERSION=%%i"
        set "GO_VERSION=!GO_VERSION:go=!"
    )

    :: 检查 Go 版本
    call :print_success "Go 版本: !GO_VERSION!"

    :: 检查 MySQL 客户端（可选）
    where mysql >nul 2>&1
    if errorlevel 1 (
        call :print_warning "MySQL 客户端未安装，某些功能可能受限"
    ) else (
        call :print_success "MySQL 客户端已安装"
    )

    call :print_success "依赖检查完成"
    exit /b

:: 设置环境
:setup_environment
    call :print_info "设置环境..."

    if not exist "%ENV_FILE%" (
        if exist "env.example" (
            copy "env.example" "%ENV_FILE%" >nul
            call :print_success "已创建环境配置文件: %ENV_FILE%"
            call :print_warning "请编辑 %ENV_FILE% 文件以配置数据库连接"
        ) else (
            call :print_error "找不到 env.example 文件"
            exit /b 1
        )
    ) else (
        call :print_info "环境配置文件已存在: %ENV_FILE%"
    )

    :: 检查必要的环境变量
    set "REQUIRED_VARS=MYSQL_HOST MYSQL_USER MYSQL_DATABASE"
    set "MISSING_VARS="

    for %%v in (%REQUIRED_VARS%) do (
        if not defined %%v (
            if defined MISSING_VARS (
                set "MISSING_VARS=!MISSING_VARS!, %%v"
            ) else (
                set "MISSING_VARS=%%v"
            )
        )
    )

    if not "!MISSING_VARS!"=="" (
        call :print_warning "以下环境变量未设置: !MISSING_VARS!"
        call :print_warning "请在 %ENV_FILE% 文件中设置这些变量"
    )

    call :print_success "环境设置完成"
    exit /b

:: 构建项目
:build_project
    call :print_info "构建项目..."

    call :check_dependencies
    if errorlevel 1 exit /b 1

    :: 创建构建目录
    if not exist "%BUILD_DIR%" mkdir "%BUILD_DIR%"

    :: 构建
    go build -o "%BUILD_DIR%\%BINARY_NAME%" ./cmd
    if errorlevel 1 (
        call :print_error "项目构建失败"
        exit /b 1
    )

    call :print_success "项目构建成功: %BUILD_DIR%\%BINARY_NAME%"

    echo.
    echo 构建信息:
    echo   项目: %PROJECT_NAME%
    echo   版本: %VERSION%
    echo   二进制: %BUILD_DIR%\%BINARY_NAME%
    for /f "tokens=1-3" %%i in ('go version') do echo   Go 版本: %%i %%j %%k
    echo.
    exit /b

:: 运行项目
:run_project
    call :print_info "启动 MySQL MCP Server..."

    :: 检查是否已构建
    if not exist "%BUILD_DIR%\%BINARY_NAME%" (
        call :print_warning "未找到构建文件，正在构建..."
        call :build_project
        if errorlevel 1 exit /b 1
    )

    call :setup_environment

    :: 显示配置信息
    echo.
    echo 服务器配置:
    echo   主机: %MYSQL_HOST%
    echo   端口: %MYSQL_PORT%
    echo   数据库: %MYSQL_DATABASE%
    echo   用户: %MYSQL_USER%
    echo.

    call :print_info "按 Ctrl+C 停止服务器"
    echo.

    :: 运行服务器
    "%BUILD_DIR%\%BINARY_NAME%"
    exit /b

:: 清理构建文件
:clean_project
    call :print_info "清理构建文件..."

    if exist "%BUILD_DIR%" (
        rmdir /s /q "%BUILD_DIR%"
        call :print_success "已清理构建目录: %BUILD_DIR%"
    ) else (
        call :print_info "构建目录不存在: %BUILD_DIR%"
    )

    :: 清理 Go 缓存
    go clean -cache
    call :print_success "Go 缓存已清理"
    exit /b

:: 运行测试
:run_tests
    call :print_info "运行测试..."

    call :check_dependencies
    if errorlevel 1 exit /b 1

    go test ./...
    if errorlevel 1 (
        call :print_error "测试失败"
        exit /b 1
    )

    call :print_success "测试通过"
    exit /b

:: 主函数
:main
    if "%~1"=="" goto show_help

    set "COMMAND=%~1"

    if "%COMMAND%"=="build" goto build
    if "%COMMAND%"=="run" goto run
    if "%COMMAND%"=="setup" goto setup
    if "%COMMAND%"=="clean" goto clean
    if "%COMMAND%"=="test" goto test
    if "%COMMAND%"=="help" goto show_help
    if "%COMMAND%"=="-h" goto show_help
    if "%COMMAND%"=="--help" goto show_help

    call :print_error "未知命令: %COMMAND%"
    echo.
    goto show_help

:build
    call :build_project
    goto :eof

:run
    call :run_project
    goto :eof

:setup
    call :setup_environment
    goto :eof

:clean
    call :clean_project
    goto :eof

:test
    call :run_tests
    goto :eof

:show_help
    call :show_help
    goto :eof

:: 启用颜色
for /F "tokens=1,2 delims=#" %%a in ('"prompt #$H#$E# & echo on & for %%b in (1) do rem"') do (
  set "DEL=%%a"
)

<nul set /p ".=%DEL%" > "%~f0?.tmp"
for /F "tokens=1 delims=" %%a in ('"%~f0?.tmp"') do (
  set "ESC=%%a"
)
del "%~f0?.tmp"

set "RED=%ESC%%RED%"
set "GREEN=%ESC%%GREEN%"
set "YELLOW=%ESC%%YELLOW%"
set "BLUE=%ESC%%BLUE%"
set "NC=%ESC%%NC%"

:: 执行主函数
call :main %*
endlocal
