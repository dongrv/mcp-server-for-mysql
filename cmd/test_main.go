package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dongrv/mcp-server-for-mysql/internal/config"
	"github.com/dongrv/mcp-server-for-mysql/internal/mysql"
	"github.com/dongrv/mcp-server-for-mysql/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestMain 验证新的模块化结构
func TestMain() {
	fmt.Println("=== 测试 MySQL MCP Server 新结构 ===")

	// 1. 测试配置加载
	testConfig()

	// 2. 测试数据库连接池
	testConnectionPool()

	// 3. 测试工具注册
	testToolRegistration()

	// 4. 测试事务管理器
	testTransactionManager()

	fmt.Println("=== 所有测试通过 ===")
}

func testConfig() {
	fmt.Println("\n1. 测试配置加载...")

	cfg := config.Default()

	// 验证默认配置
	if cfg.MySQL.Host != "localhost" {
		log.Fatalf("默认主机名错误: %s", cfg.MySQL.Host)
	}
	if cfg.MySQL.Port != 3306 {
		log.Fatalf("默认端口错误: %d", cfg.MySQL.Port)
	}
	if cfg.MySQL.User != "root" {
		log.Fatalf("默认用户错误: %s", cfg.MySQL.User)
	}

	// 测试 DSN 生成
	dsn := cfg.MySQL.DSN()
	expectedDSN := "root:@tcp(localhost:3306)/test?parseTime=true&loc=Local&timeout=30s"
	if dsn != expectedDSN {
		log.Fatalf("DSN 生成错误:\n期望: %s\n实际: %s", expectedDSN, dsn)
	}

	// 测试配置验证
	if err := cfg.Validate(); err != nil {
		log.Fatalf("配置验证失败: %v", err)
	}

	fmt.Println("✓ 配置加载测试通过")
}

func testConnectionPool() {
	fmt.Println("\n2. 测试数据库连接池...")

	// 创建测试配置
	testCfg := &config.MySQLConfig{
		Host:            "localhost",
		Port:            3306,
		User:            "test_user",
		Password:        "test_pass",
		Database:        "test_db",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}

	// 测试 DSN 生成
	dsn := testCfg.DSN()
	fmt.Printf("测试 DSN: %s\n", dsn)

	// 注意：这里不实际连接数据库，因为需要真实的 MySQL 实例
	// 在实际测试中，应该使用测试数据库或模拟连接

	fmt.Println("✓ 连接池配置测试通过（跳过实际连接测试）")
}

func testToolRegistration() {
	fmt.Println("\n3. 测试工具注册...")

	// 创建模拟的数据库连接池（用于测试）
	_ = &mysql.Pool{}

	// 创建工具注册表
	registry := tools.NewRegistry()

	// 测试注册表基本功能
	if names := registry.Names(); len(names) != 0 {
		log.Fatalf("新注册表应该为空，但有 %d 个工具", len(names))
	}

	// 测试重复注册
	mockHandler := &mockToolHandler{name: "test_tool"}
	if err := registry.Register(mockHandler); err != nil {
		log.Fatalf("注册工具失败: %v", err)
	}

	if err := registry.Register(mockHandler); err == nil {
		log.Fatal("重复注册应该失败")
	}

	// 测试获取工具
	if handler, exists := registry.Get("test_tool"); !exists || handler == nil {
		log.Fatal("获取已注册工具失败")
	}

	// 测试获取不存在的工具
	if _, exists := registry.Get("non_existent"); exists {
		log.Fatal("不存在的工具不应该被找到")
	}

	fmt.Println("✓ 工具注册测试通过")
}

func testTransactionManager() {
	fmt.Println("\n4. 测试事务管理器...")

	// 创建模拟的数据库连接池（用于测试）
	_ = &mysql.Pool{}

	// 创建事务管理器
	txManager := mysql.NewTxManager(&mysql.Pool{})

	// 测试初始状态
	if count := txManager.Count(); count != 0 {
		log.Fatalf("新事务管理器应该没有事务，但有 %d 个", count)
	}

	// 测试获取不存在的交易
	if _, exists := txManager.Get("non_existent"); exists {
		log.Fatal("不存在的交易不应该被找到")
	}

	fmt.Println("✓ 事务管理器测试通过")
}

// mockToolHandler 用于测试的模拟工具处理器
type mockToolHandler struct {
	name string
}

func (m *mockToolHandler) Name() string {
	return m.name
}

func (m *mockToolHandler) Description() string {
	return "测试工具"
}

func (m *mockToolHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	response := map[string]interface{}{
		"message": "测试响应",
		"tool":    m.name,
	}

	jsonBytes, _ := json.MarshalIndent(response, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonBytes)},
		},
	}, response, nil
}

// 运行测试
func testMain() {
	// 设置测试环境变量
	os.Setenv("MYSQL_HOST", "localhost")
	os.Setenv("MYSQL_PORT", "3306")
	os.Setenv("MYSQL_USER", "test_user")
	os.Setenv("MYSQL_PASSWORD", "test_pass")
	os.Setenv("MYSQL_DATABASE", "test_db")

	TestMain()
}
