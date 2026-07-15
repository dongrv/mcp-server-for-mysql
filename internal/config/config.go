// Package config loads source connection configuration.
package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// Mode controls the MCP execution policy.
type Mode string

const (
	// QuickMode permits the standard execution policy.
	QuickMode Mode = "quick"
	// StrictMode enables the restrictive execution policy.
	StrictMode Mode = "strict"

	// MaxDisplayNameRunes is the maximum length of a source display name.
	MaxDisplayNameRunes = 80
	// MaxDescriptionRunes is the maximum length of a source description.
	MaxDescriptionRunes = 500
	// MaxProfileItemRunes is the maximum length of one alias or keyword.
	MaxProfileItemRunes = 80
	// MaxAliases is the maximum number of aliases for one source.
	MaxAliases = 20
	// MaxKeywords is the maximum number of keywords for one source.
	MaxKeywords = 30
)

var environmentReference = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$`)

// SourceConfig identifies a configured database source and its resolved DSN.
type SourceConfig struct {
	Name        string   `yaml:"name"`
	DisplayName string   `yaml:"display_name"`
	Description string   `yaml:"description"`
	Aliases     []string `yaml:"aliases,omitempty"`
	Keywords    []string `yaml:"keywords,omitempty"`
	Type        string   `yaml:"type"`
	DSN         string   `yaml:"dsn"`
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
	environmentNames := make([]string, len(config.Sources))
	for i := range config.Sources {
		source := &config.Sources[i]
		if strings.TrimSpace(source.Name) == "" {
			return Config{}, fmt.Errorf("source name is required")
		}
		if _, exists := names[source.Name]; exists {
			return Config{}, fmt.Errorf("duplicate source name %q", source.Name)
		}
		names[source.Name] = struct{}{}
		if err := validateSourceProfile(source); err != nil {
			return Config{}, err
		}

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
		environmentNames[i] = matches[1]
	}

	if lookupEnv == nil {
		return Config{}, fmt.Errorf("source %q DSN environment value is required", config.Sources[0].Name)
	}
	for i := range config.Sources {
		source := &config.Sources[i]
		if resolved, ok := lookupEnv(environmentNames[i]); ok && strings.TrimSpace(resolved) != "" {
			source.DSN = resolved
		} else {
			return Config{}, fmt.Errorf("source %q DSN environment value is required", source.Name)
		}
	}

	return config, nil
}

func normalizeSourceProfile(source *SourceConfig) {
	source.DisplayName = strings.TrimSpace(source.DisplayName)
	source.Description = strings.TrimSpace(source.Description)
	for i := range source.Aliases {
		source.Aliases[i] = strings.TrimSpace(source.Aliases[i])
	}
	for i := range source.Keywords {
		source.Keywords[i] = strings.TrimSpace(source.Keywords[i])
	}
}

func validateSourceProfile(source *SourceConfig) error {
	normalizeSourceProfile(source)

	if err := validateRequiredProfileField(source.Name, "display_name", source.DisplayName, MaxDisplayNameRunes); err != nil {
		return err
	}
	if err := validateRequiredProfileField(source.Name, "description", source.Description, MaxDescriptionRunes); err != nil {
		return err
	}
	if err := validateProfileItems(source.Name, "aliases", source.Aliases, MaxAliases); err != nil {
		return err
	}
	return validateProfileItems(source.Name, "keywords", source.Keywords, MaxKeywords)
}

func validateRequiredProfileField(sourceName, field, value string, maxRunes int) error {
	if value == "" {
		return fmt.Errorf("source %q %s is required", sourceName, field)
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("source %q %s must be valid UTF-8", sourceName, field)
	}
	if utf8.RuneCountInString(value) > maxRunes {
		return fmt.Errorf("source %q %s exceeds %d runes", sourceName, field, maxRunes)
	}
	return nil
}

func validateProfileItems(sourceName, field string, items []string, maxItems int) error {
	if len(items) > maxItems {
		return fmt.Errorf("source %q %s exceeds %d items", sourceName, field, maxItems)
	}
	seen := make(map[string]struct{}, len(items))
	for i, item := range items {
		if item == "" {
			return fmt.Errorf("source %q %s[%d] is required", sourceName, field, i)
		}
		if !utf8.ValidString(item) {
			return fmt.Errorf("source %q %s[%d] must be valid UTF-8", sourceName, field, i)
		}
		if utf8.RuneCountInString(item) > MaxProfileItemRunes {
			return fmt.Errorf("source %q %s[%d] exceeds %d runes", sourceName, field, i, MaxProfileItemRunes)
		}
		if _, exists := seen[item]; exists {
			return fmt.Errorf("source %q %s contains duplicate item", sourceName, field)
		}
		seen[item] = struct{}{}
	}
	return nil
}
