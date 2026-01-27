package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the MySQL MCP Server.
type Config struct {
	// MySQL connection settings
	MySQL MySQLConfig
}

// MySQLConfig holds MySQL-specific configuration.
type MySQLConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string

	// Connection pool settings
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// Default returns a Config with default values.
func Default() *Config {
	return &Config{
		MySQL: MySQLConfig{
			Host:            getEnvOrDefault("MYSQL_HOST", "localhost"),
			Port:            getEnvAsInt("MYSQL_PORT", 3306),
			User:            getEnvOrDefault("MYSQL_USER", "root"),
			Password:        getEnvOrDefault("MYSQL_PASSWORD", ""),
			Database:        getEnvOrDefault("MYSQL_DATABASE", "test"),
			MaxOpenConns:    getEnvAsInt("MYSQL_MAX_OPEN_CONNS", 10),
			MaxIdleConns:    getEnvAsInt("MYSQL_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: time.Duration(getEnvAsInt("MYSQL_CONN_MAX_LIFETIME_MINUTES", 30)) * time.Minute,
			ConnMaxIdleTime: time.Duration(getEnvAsInt("MYSQL_CONN_MAX_IDLE_TIME_MINUTES", 5)) * time.Minute,
		},
	}
}

// Helper functions for environment variable handling
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func main() {
	fmt.Println("=== MySQL MCP Server 环境变量测试 ===")
	fmt.Println()

	// 显示所有环境变量
	fmt.Println("=== 所有环境变量 ===")
	for _, env := range os.Environ() {
		fmt.Println(env)
	}
	fmt.Println()

	// 检查特定的MySQL环境变量
	mysqlEnvVars := []string{
		"MYSQL_HOST",
		"MYSQL_PORT",
		"MYSQL_USER",
		"MYSQL_PASSWORD",
		"MYSQL_DATABASE",
		"MYSQL_MAX_OPEN_CONNS",
		"MYSQL_MAX_IDLE_CONNS",
		"MYSQL_CONN_MAX_LIFETIME_MINUTES",
		"MYSQL_CONN_MAX_IDLE_TIME_MINUTES",
		"LOG_LEVEL",
		"LOG_FORMAT",
	}

	fmt.Println("=== MySQL相关环境变量 ===")
	for _, key := range mysqlEnvVars {
		value := os.Getenv(key)
		if value == "" {
			fmt.Printf("%s: [未设置]\n", key)
		} else {
			fmt.Printf("%s: %s\n", key, value)
		}
	}
	fmt.Println()

	// 测试配置加载
	fmt.Println("=== 配置加载测试 ===")
	config := Default()
	fmt.Printf("MySQL Host: %s\n", config.MySQL.Host)
	fmt.Printf("MySQL Port: %d\n", config.MySQL.Port)
	fmt.Printf("MySQL User: %s\n", config.MySQL.User)
	fmt.Printf("MySQL Database: %s\n", config.MySQL.Database)
	fmt.Printf("MySQL MaxOpenConns: %d\n", config.MySQL.MaxOpenConns)
	fmt.Printf("MySQL MaxIdleConns: %d\n", config.MySQL.MaxIdleConns)
	fmt.Printf("MySQL ConnMaxLifetime: %v\n", config.MySQL.ConnMaxLifetime)
	fmt.Printf("MySQL ConnMaxIdleTime: %v\n", config.MySQL.ConnMaxIdleTime)
	fmt.Println()

	// 验证配置
	fmt.Println("=== 配置验证 ===")
	if config.MySQL.Host == "" {
		fmt.Println("❌ MYSQL_HOST 未设置")
	} else {
		fmt.Println("✅ MYSQL_HOST 已设置")
	}

	if config.MySQL.User == "" {
		fmt.Println("❌ MYSQL_USER 未设置")
	} else {
		fmt.Println("✅ MYSQL_USER 已设置")
	}

	if config.MySQL.Database == "" {
		fmt.Println("❌ MYSQL_DATABASE 未设置")
	} else {
		fmt.Println("✅ MYSQL_DATABASE 已设置")
	}

	fmt.Println()
	fmt.Println("=== 测试完成 ===")
}
