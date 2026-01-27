// Package config provides configuration management for MySQL MCP Server.
package config

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

	// Server settings
	Server ServerConfig

	// Logging settings
	Logging LoggingConfig
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

// ServerConfig holds server-specific configuration.
type ServerConfig struct {
	Name        string
	Version     string
	Description string
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level  string
	Format string
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
		Server: ServerConfig{
			Name:        "mysql-mcp-server",
			Version:     "1.0.0",
			Description: "MySQL MCP Server for database operations",
		},
		Logging: LoggingConfig{
			Level:  getEnvOrDefault("LOG_LEVEL", "info"),
			Format: getEnvOrDefault("LOG_FORMAT", "text"),
		},
	}
}

// DSN returns the MySQL Data Source Name.
func (c *MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local&timeout=30s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.MySQL.Host == "" {
		return fmt.Errorf("mysql host is required")
	}
	if c.MySQL.User == "" {
		return fmt.Errorf("mysql user is required")
	}
	if c.MySQL.Database == "" {
		return fmt.Errorf("mysql database is required")
	}
	if c.MySQL.MaxOpenConns <= 0 {
		return fmt.Errorf("mysql max open connections must be positive")
	}
	if c.MySQL.MaxIdleConns < 0 {
		return fmt.Errorf("mysql max idle connections cannot be negative")
	}
	if c.MySQL.MaxIdleConns > c.MySQL.MaxOpenConns {
		return fmt.Errorf("mysql max idle connections cannot exceed max open connections")
	}

	return nil
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

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if dur, err := time.ParseDuration(value); err == nil {
			return dur
		}
	}
	return defaultValue
}
