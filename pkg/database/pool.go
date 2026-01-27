package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// ConnectionPool 管理数据库连接池
type ConnectionPool struct {
	db           *sql.DB
	mu           sync.RWMutex
	maxConns     int
	idleConns    int
	connLifetime time.Duration
}

// Config 数据库配置
type Config struct {
	Host         string
	Port         int
	Username     string
	Password     string
	Database     string
	MaxConns     int
	IdleConns    int
	ConnLifetime time.Duration
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Host:         getEnvOrDefault("MYSQL_HOST", "localhost"),
		Port:         getEnvAsInt("MYSQL_PORT", 3306),
		Username:     getEnvOrDefault("MYSQL_USER", "root"),
		Password:     getEnvOrDefault("MYSQL_PASSWORD", ""),
		Database:     getEnvOrDefault("MYSQL_DATABASE", "test"),
		MaxConns:     getEnvAsInt("MYSQL_MAX_CONNS", 10),
		IdleConns:    getEnvAsInt("MYSQL_IDLE_CONNS", 5),
		ConnLifetime: time.Duration(getEnvAsInt("MYSQL_CONN_LIFETIME_MINUTES", 30)) * time.Minute,
	}
}

// NewConnectionPool 创建新的数据库连接池
func NewConnectionPool() (*ConnectionPool, error) {
	config := DefaultConfig()

	// 构建连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	log.Printf("Connecting to MySQL at %s:%d/%s", config.Host, config.Port, config.Database)

	// 打开数据库连接
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(config.MaxConns)
	db.SetMaxIdleConns(config.IdleConns)
	db.SetConnMaxLifetime(config.ConnLifetime)

	pool := &ConnectionPool{
		db:           db,
		maxConns:     config.MaxConns,
		idleConns:    config.IdleConns,
		connLifetime: config.ConnLifetime,
	}

	log.Printf("Database connection pool initialized: max=%d, idle=%d, lifetime=%v",
		config.MaxConns, config.IdleConns, config.ConnLifetime)

	return pool, nil
}

// GetConnection 获取数据库连接
func (p *ConnectionPool) GetConnection() (*sql.DB, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.db == nil {
		return nil, fmt.Errorf("database connection pool is closed")
	}

	// 测试连接是否仍然有效
	if err := p.db.Ping(); err != nil {
		return nil, fmt.Errorf("database connection is not healthy: %w", err)
	}

	return p.db, nil
}

// BeginTransaction 开始一个新的事务
func (p *ConnectionPool) BeginTransaction() (*sql.Tx, error) {
	db, err := p.GetConnection()
	if err != nil {
		return nil, err
	}

	return db.Begin()
}

// Close 关闭连接池
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.db != nil {
		err := p.db.Close()
		p.db = nil
		return err
	}
	return nil
}

// Stats 返回连接池统计信息
func (p *ConnectionPool) Stats() sql.DBStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.db != nil {
		return p.db.Stats()
	}
	return sql.DBStats{}
}

// GetConfig 返回当前配置
func (p *ConnectionPool) GetConfig() Config {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return Config{
		MaxConns:     p.maxConns,
		IdleConns:    p.idleConns,
		ConnLifetime: p.connLifetime,
	}
}

// 辅助函数：从环境变量获取值，如果不存在则返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// 辅助函数：从环境变量获取整数值，如果不存在则返回默认值
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}
