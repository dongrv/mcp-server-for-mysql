package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModeConstants(t *testing.T) {
	if QuickMode != Mode("quick") {
		t.Errorf("QuickMode = %q, want quick", QuickMode)
	}
	if StrictMode != Mode("strict") {
		t.Errorf("StrictMode = %q, want strict", StrictMode)
	}
}

func TestLoadDefaultsToQuickAndExpandsSourceDSNs(t *testing.T) {
	path := writeConfig(t, `
sources:
  - name: orders
    type: mysql
    dsn: ${ORDERS_DSN}
  - name: analytics
    type: clickhouse
    dsn: ${ANALYTICS_DSN}
`)

	config, err := Load(path, envLookup(map[string]string{
		"ORDERS_DSN":    "orders-secret-dsn",
		"ANALYTICS_DSN": "analytics-secret-dsn",
	}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Mode != QuickMode {
		t.Errorf("Mode = %q, want %q", config.Mode, QuickMode)
	}
	want := []SourceConfig{
		{Name: "orders", Type: "mysql", DSN: "orders-secret-dsn"},
		{Name: "analytics", Type: "clickhouse", DSN: "analytics-secret-dsn"},
	}
	if len(config.Sources) != len(want) {
		t.Fatalf("Sources length = %d, want %d", len(config.Sources), len(want))
	}
	for i, source := range config.Sources {
		if source != want[i] {
			t.Errorf("Sources[%d] = %#v, want %#v", i, source, want[i])
		}
	}
}

func TestLoadAcceptsStrictMode(t *testing.T) {
	path := writeConfig(t, `
mode: strict
sources:
  - name: orders
    type: mysql
    dsn: ${ORDERS_DSN}
`)

	config, err := Load(path, envLookup(map[string]string{"ORDERS_DSN": "orders-dsn"}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Mode != StrictMode {
		t.Errorf("Mode = %q, want %q", config.Mode, StrictMode)
	}
}

func TestLoadRejectsUnknownYAMLFields(t *testing.T) {
	path := writeConfig(t, `
mdoe: strict
sources:
  - name: orders
    type: mysql
    dsn: ${ORDERS_DSN}
`)

	_, err := Load(path, envLookup(map[string]string{"ORDERS_DSN": "orders-secret-dsn"}))
	if err == nil {
		t.Fatal("Load() error = nil, want unknown YAML field error")
	}
	if err.Error() != "invalid configuration file" {
		t.Errorf("Load() error = %q, want non-secret configuration error", err)
	}
}

func TestLoadRejectsMultipleYAMLDocuments(t *testing.T) {
	path := writeConfig(t, `
sources:
  - name: orders
    type: mysql
    dsn: ${ORDERS_DSN}
---
mdoe: strict
`)

	_, err := Load(path, envLookup(map[string]string{"ORDERS_DSN": "orders-secret-dsn"}))
	if err == nil {
		t.Fatal("Load() error = nil, want multiple YAML documents error")
	}
	if err.Error() != "invalid configuration file" {
		t.Errorf("Load() error = %q, want non-secret configuration error", err)
	}
}

func TestLoadRejectsMissingEnvironmentValueWithoutLeakingDSN(t *testing.T) {
	path := writeConfig(t, `
sources:
  - name: orders
    type: mysql
    dsn: ${ORDERS_DSN}
`)

	_, err := Load(path, envLookup(nil))
	if err == nil {
		t.Fatal("Load() error = nil, want missing environment value error")
	}
	if strings.Contains(err.Error(), "${ORDERS_DSN}") || strings.Contains(err.Error(), "orders-secret-dsn") {
		t.Errorf("Load() error leaked DSN data: %v", err)
	}
}

func TestLoadRejectsEmptyEnvironmentValue(t *testing.T) {
	path := writeConfig(t, `
sources:
  - name: orders
    type: mysql
    dsn: ${ORDERS_DSN}
`)

	_, err := Load(path, func(string) (string, bool) { return "", true })
	if err == nil || !strings.Contains(err.Error(), "environment value is required") {
		t.Fatalf("Load() error = %v, want empty environment value rejection", err)
	}
}

func TestLoadRejectsUnknownMode(t *testing.T) {
	path := writeConfig(t, `
mode: audit
sources:
  - name: orders
    type: mysql
    dsn: ${ORDERS_DSN}
`)

	assertLoadError(t, path, envLookup(map[string]string{"ORDERS_DSN": "orders-dsn"}))
}

func TestLoadRejectsUnknownSourceType(t *testing.T) {
	path := writeConfig(t, `
sources:
  - name: orders
    type: postgres
    dsn: ${ORDERS_DSN}
`)

	assertLoadError(t, path, envLookup(map[string]string{"ORDERS_DSN": "orders-dsn"}))
}

func TestLoadRejectsDuplicateSourceNames(t *testing.T) {
	path := writeConfig(t, `
sources:
  - name: orders
    type: mysql
    dsn: ${ORDERS_DSN}
  - name: orders
    type: clickhouse
    dsn: ${ANALYTICS_DSN}
`)

	assertLoadError(t, path, envLookup(map[string]string{
		"ORDERS_DSN":    "orders-dsn",
		"ANALYTICS_DSN": "analytics-dsn",
	}))
}

func TestLoadRejectsMalformedDSNPlaceholders(t *testing.T) {
	for _, dsn := range []string{"orders-dsn", "${}", "$ORDERS_DSN", "mysql://${ORDERS_DSN}"} {
		t.Run(dsn, func(t *testing.T) {
			path := writeConfig(t, "sources:\n  - name: orders\n    type: mysql\n    dsn: "+dsn+"\n")
			assertLoadError(t, path, envLookup(map[string]string{"ORDERS_DSN": "orders-dsn"}))
		})
	}
}

func TestLoadRejectsNoSources(t *testing.T) {
	path := writeConfig(t, "mode: quick\n")

	assertLoadErrorContains(t, path, envLookup(nil), "at least one source is required")
}

func TestLoadRejectsEmptySourceName(t *testing.T) {
	path := writeConfig(t, `
sources:
  - name: ""
    type: mysql
    dsn: ${ORDERS_DSN}
`)

	assertLoadErrorContains(t, path, envLookup(map[string]string{"ORDERS_DSN": "orders-dsn"}), "source name is required")
}

func TestLoadRejectsEmptyDSN(t *testing.T) {
	path := writeConfig(t, `
sources:
  - name: orders
    type: mysql
    dsn: ""
`)

	assertLoadErrorContains(t, path, envLookup(nil), "DSN is required")
}

func TestLoadRedactsExpandedDSNFromLaterInvalidSource(t *testing.T) {
	const secretDSN = "mysql://user:super-secret-password@host/orders"
	path := writeConfig(t, `
sources:
  - name: orders
    type: mysql
    dsn: ${SECRET_DSN}
  - name: analytics
    type: unknown
    dsn: ${ANALYTICS_DSN}
`)

	secretLookedUp := false
	_, err := Load(path, func(name string) (string, bool) {
		if name == "SECRET_DSN" {
			secretLookedUp = true
			return secretDSN, true
		}
		return "analytics-dsn", true
	})
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
	if !secretLookedUp {
		t.Fatal("Load() did not expand the first source DSN before failing")
	}
	if !strings.Contains(err.Error(), "unsupported type") {
		t.Errorf("Load() error = %q, want later source type validation error", err)
	}
	if strings.Contains(err.Error(), secretDSN) {
		t.Errorf("Load() error leaked DSN: %v", err)
	}
}

func assertLoadError(t *testing.T, path string, lookupEnv func(string) (string, bool)) {
	t.Helper()
	if _, err := Load(path, lookupEnv); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func assertLoadErrorContains(t *testing.T, path string, lookupEnv func(string) (string, bool), want string) {
	t.Helper()
	_, err := Load(path, lookupEnv)
	if err == nil {
		t.Fatalf("Load() error = nil, want error containing %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Load() error = %q, want error containing %q", err, want)
	}
}

func envLookup(values map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
