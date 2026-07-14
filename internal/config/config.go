// Package config loads source connection configuration.
package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Mode controls the MCP execution policy.
type Mode string

const (
	// QuickMode permits the standard execution policy.
	QuickMode Mode = "quick"
	// StrictMode enables the restrictive execution policy.
	StrictMode Mode = "strict"
)

var environmentReference = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$`)

// SourceConfig identifies a configured database source and its resolved DSN.
type SourceConfig struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	DSN  string `yaml:"dsn"`
}

// Config contains MCP execution mode and configured database sources.
type Config struct {
	Mode    Mode           `yaml:"mode"`
	Sources []SourceConfig `yaml:"sources"`
}

// MySQLConfig is retained temporarily for legacy callers. New code must use
// Config and Load instead.
//
// Deprecated: use SourceConfig from Load.
type MySQLConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DSN returns a MySQL connection string for legacy callers.
//
// Deprecated: use SourceConfig.DSN from Load.
func (c *MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local&timeout=30s",
		c.User, c.Password, c.Host, c.Port, c.Database)
}

// ServerConfig is retained temporarily for the legacy command entrypoint.
//
// Deprecated: construct the MCP server from Config loaded with Load.
type ServerConfig struct {
	Name    string
	Version string
}

// LegacyConfig is retained temporarily so unchanged legacy callers compile.
// It preserves their environment-based startup behavior until migration.
//
// Deprecated: use Config and Load.
type LegacyConfig struct {
	MySQL  MySQLConfig
	Server ServerConfig
}

// Default returns environment-derived compatibility settings for the legacy command.
//
// Deprecated: use Load with an explicit YAML path and environment lookup.
func Default() *LegacyConfig {
	return &LegacyConfig{
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
			Name:    "mysql-mcp-server",
			Version: "1.0.0",
		},
	}
}

// Validate checks the retained compatibility configuration.
//
// Deprecated: Load validates new configuration before returning it.
func (c *LegacyConfig) Validate() error {
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

// Load reads and validates source configuration from path. DSNs must be exact
// environment references in the form ${NAME}; resolved values are never included
// in returned errors.
func Load(path string, lookupEnv func(string) string) (Config, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read configuration: %w", err)
	}

	var config Config
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("invalid configuration file")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Config{}, fmt.Errorf("invalid configuration file")
	}
	if config.Mode == "" {
		config.Mode = QuickMode
	}
	if config.Mode != QuickMode && config.Mode != StrictMode {
		return Config{}, fmt.Errorf("invalid mode %q", config.Mode)
	}
	if len(config.Sources) == 0 {
		return Config{}, fmt.Errorf("at least one source is required")
	}
	names := make(map[string]struct{}, len(config.Sources))
	for i := range config.Sources {
		source := &config.Sources[i]
		if strings.TrimSpace(source.Name) == "" {
			return Config{}, fmt.Errorf("source name is required")
		}
		if _, exists := names[source.Name]; exists {
			return Config{}, fmt.Errorf("duplicate source name %q", source.Name)
		}
		names[source.Name] = struct{}{}

		if source.Type != "mysql" && source.Type != "clickhouse" {
			return Config{}, fmt.Errorf("source %q has unsupported type %q", source.Name, source.Type)
		}
		if strings.TrimSpace(source.DSN) == "" {
			return Config{}, fmt.Errorf("source %q DSN is required", source.Name)
		}
		matches := environmentReference.FindStringSubmatch(source.DSN)
		if matches == nil {
			return Config{}, fmt.Errorf("source %q has an invalid DSN environment reference", source.Name)
		}
		if lookupEnv == nil {
			return Config{}, fmt.Errorf("source %q DSN environment value is required", source.Name)
		}
		if resolved := lookupEnv(matches[1]); resolved != "" {
			source.DSN = resolved
		} else {
			return Config{}, fmt.Errorf("source %q DSN environment value is required", source.Name)
		}
	}

	return config, nil
}

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
