// Package config loads source connection configuration.
package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

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

// Load reads and validates source configuration from path. DSNs must be exact
// environment references in the form ${NAME}; resolved values are never included
// in returned errors.
func Load(path string, lookupEnv func(string) (string, bool)) (Config, error) {
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
		if resolved, ok := lookupEnv(matches[1]); ok && strings.TrimSpace(resolved) != "" {
			source.DSN = resolved
		} else {
			return Config{}, fmt.Errorf("source %q DSN environment value is required", source.Name)
		}
	}

	return config, nil
}
