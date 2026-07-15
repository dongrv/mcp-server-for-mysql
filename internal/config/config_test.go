package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	localGuideStarterHeading      = "## 2. 最小可运行配置：一个 orders MySQL 数据源"
	localGuideUniqueSourceRule    = "唯一匹配时，AI 必须使用该候选的精确 `source_id`。"
	localGuideAmbiguousSourceRule = "多候选时，AI 必须列出候选的 `display_name` 和 `description`，并等待用户选择。"
	localGuideNoGuessRule         = "AI 不得猜测，也不得同时查询所有候选。"
	localGuideReadOnlyQueryRule   = "`query` 只接受只读 SQL。"
	localGuideDestructiveRule     = "破坏性操作必须使用 `execute_sql`。"
	localGuideNoWebSetupRule      = "本指南不提供 ChatGPT web 接入配置。"
)

func TestModeConstants(t *testing.T) {
	if QuickMode != Mode("quick") {
		t.Errorf("QuickMode = %q, want quick", QuickMode)
	}
	if StrictMode != Mode("strict") {
		t.Errorf("StrictMode = %q, want strict", StrictMode)
	}
}

func TestLocalClientGuideContract(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "docs", "getting-started-local-clients.md"))
	if err != nil {
		t.Fatalf("read local client guide: %v", err)
	}

	guide := string(content)
	for _, anchor := range []string{
		"Claude Desktop",
		"Codex",
		"本地二进制",
		"Docker",
		"list_sources",
		"list_tables",
		"preview_hash",
		"Codex Docker（PowerShell）",
	} {
		if !strings.Contains(guide, anchor) {
			t.Errorf("local client guide must contain %q", anchor)
		}
	}
	for _, legacy := range []string{"MYSQL_HOST", "mysql_begin_transaction"} {
		if strings.Contains(guide, legacy) {
			t.Errorf("local client guide must not contain legacy setup guidance %q", legacy)
		}
	}
	for _, rule := range []string{
		localGuideUniqueSourceRule,
		localGuideAmbiguousSourceRule,
		localGuideNoGuessRule,
		localGuideReadOnlyQueryRule,
		localGuideDestructiveRule,
		localGuideNoWebSetupRule,
	} {
		if !strings.Contains(guide, rule) {
			t.Errorf("local client guide must contain exact safety rule %q", rule)
		}
	}

	starter, ok := markdownFencedBlockAfter(guide, localGuideStarterHeading, "yaml")
	if !ok {
		t.Errorf("local client guide must contain a YAML block after %q", localGuideStarterHeading)
	} else {
		if strings.Count(starter, "- name:") != 1 || !strings.Contains(starter, "- name: orders") {
			t.Errorf("starter YAML must contain exactly one orders source:\n%s", starter)
		}
		if !strings.Contains(starter, "${ORDERS_DSN}") {
			t.Error("starter YAML must reference ORDERS_DSN")
		}
		for _, extra := range []string{"LOGS_DSN", "ANALYTICS_DSN"} {
			if strings.Contains(starter, extra) {
				t.Errorf("starter YAML must not require %s", extra)
			}
		}
	}

	if containsUnsafeSourceSelectionGuidance(guide) {
		t.Error("local client guide must not allow guessing or querying all plausible sources")
	}
	if containsChatGPTWebSetup(guide) {
		t.Error("local client guide must not contain ChatGPT web setup instructions")
	}

	t.Run("detects unsafe source selection guidance", func(t *testing.T) {
		tests := []struct {
			name string
			text string
			want bool
		}{
			{name: "safe Chinese rule", text: localGuideNoGuessRule, want: false},
			{name: "safe English rule", text: "AI MUST NOT guess or query all candidates.", want: false},
			{name: "unsafe Chinese guess", text: "AI 可以猜测最可能的数据源。", want: true},
			{name: "unsafe English guess", text: "AI may guess the most likely source.", want: true},
			{name: "unsafe query all", text: "AI 可以同时查询所有候选。", want: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := containsUnsafeSourceSelectionGuidance(tt.text); got != tt.want {
					t.Errorf("containsUnsafeSourceSelectionGuidance(%q) = %t, want %t", tt.text, got, tt.want)
				}
			})
		}
	})

	t.Run("detects ChatGPT web setup without rejecting an exclusion", func(t *testing.T) {
		tests := []struct {
			name string
			text string
			want bool
		}{
			{name: "English setup heading", text: "## ChatGPT web setup", want: true},
			{name: "remote web section", text: "## ChatGPT web\nOpen Settings and add a server.", want: true},
			{name: "Chinese setup heading", text: "## ChatGPT 网页版配置", want: true},
			{name: "Chinese setup instruction", text: "在 ChatGPT 网页版的 Settings 中添加 MCP Server。", want: true},
			{name: "explicit exclusion", text: "本文不包含 ChatGPT web 接入步骤。", want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := containsChatGPTWebSetup(tt.text); got != tt.want {
					t.Errorf("containsChatGPTWebSetup(%q) = %t, want %t", tt.text, got, tt.want)
				}
			})
		}
	})
}

func markdownFencedBlockAfter(content, heading, language string) (string, bool) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	headingIndex := strings.Index(content, heading)
	if headingIndex < 0 {
		return "", false
	}
	fence := "```" + language + "\n"
	fenceIndex := strings.Index(content[headingIndex+len(heading):], fence)
	if fenceIndex < 0 {
		return "", false
	}
	blockStart := headingIndex + len(heading) + fenceIndex + len(fence)
	blockEnd := strings.Index(content[blockStart:], "\n```")
	if blockEnd < 0 {
		return "", false
	}
	return content[blockStart : blockStart+blockEnd], true
}

func containsUnsafeSourceSelectionGuidance(content string) bool {
	for _, line := range strings.Split(strings.ToLower(content), "\n") {
		mentionsGuess := strings.Contains(line, "guess") || strings.Contains(line, "猜测")
		mentionsQueryAll := strings.Contains(line, "query all candidates") || strings.Contains(line, "同时查询所有候选") || strings.Contains(line, "查询全部候选")
		if !mentionsGuess && !mentionsQueryAll {
			continue
		}
		forbidsAction := strings.Contains(line, "must not") || strings.Contains(line, "do not") || strings.Contains(line, "never") || strings.Contains(line, "不得") || strings.Contains(line, "不能") || strings.Contains(line, "不应") || strings.Contains(line, "不要") || strings.Contains(line, "禁止")
		if !forbidsAction {
			return true
		}
	}
	return false
}

func containsChatGPTWebSetup(content string) bool {
	for _, line := range strings.Split(strings.ToLower(content), "\n") {
		if !strings.Contains(line, "chatgpt") {
			continue
		}
		mentionsWeb := strings.Contains(line, "web") || strings.Contains(line, "网页版") || strings.Contains(line, "网页端")
		mentionsSetup := strings.Contains(line, "setup") || strings.Contains(line, "配置") || strings.Contains(line, "接入") || strings.Contains(line, "添加") || strings.Contains(line, "settings")
		isSectionHeading := strings.HasPrefix(strings.TrimSpace(line), "#")
		if !mentionsWeb || (!mentionsSetup && !isSectionHeading) {
			continue
		}
		excludesSetup := strings.Contains(line, "不包含") || strings.Contains(line, "不提供") || strings.Contains(line, "不扩展") || strings.Contains(line, "exclude") || strings.Contains(line, "does not")
		if !excludesSetup {
			return true
		}
	}
	return false
}

func TestExampleConfigurationContainsNoCredentialValue(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "config.example.yaml"))
	if err != nil {
		t.Fatalf("read example configuration: %v", err)
	}
	if strings.Contains(string(content), "password:") {
		t.Fatal("example configuration must not contain a password field")
	}
	if !strings.Contains(string(content), "${ORDERS_DSN}") {
		t.Fatal("example configuration must contain the MySQL DSN environment reference")
	}
	if !strings.Contains(string(content), "${LOGS_DSN}") || !strings.Contains(string(content), "${ANALYTICS_DSN}") {
		t.Fatal("example configuration must contain distinct logs and analytics DSN environment references")
	}

	cfg, err := Load(filepath.Join("..", "..", "config.example.yaml"), envLookup(map[string]string{
		"ORDERS_DSN":    "orders-dsn",
		"LOGS_DSN":      "logs-dsn",
		"ANALYTICS_DSN": "analytics-dsn",
	}))
	if err != nil {
		t.Fatalf("Load(example) error = %v", err)
	}
	want := []SourceConfig{
		{Name: "orders", DisplayName: "订单库", Description: "用户充值订单、支付状态与退款记录", Aliases: []string{"充值库", "支付订单"}, Keywords: []string{"充值", "支付", "退款"}, Type: "mysql", DSN: "orders-dsn"},
		{Name: "logs", DisplayName: "日志库", Description: "服务运行日志、错误事件与审计记录", Aliases: []string{"服务日志", "错误日志"}, Keywords: []string{"报错", "审计", "故障排查"}, Type: "mysql", DSN: "logs-dsn"},
		{Name: "analytics", DisplayName: "分析库", Description: "聚合指标、用户行为与经营分析数据", Aliases: []string{"数仓", "经营分析"}, Keywords: []string{"指标", "趋势", "用户行为"}, Type: "clickhouse", DSN: "analytics-dsn"},
	}
	if !reflect.DeepEqual(cfg.Sources, want) {
		t.Errorf("example Sources = %#v, want %#v", cfg.Sources, want)
	}
}

func TestLoadDefaultsToQuickAndExpandsSourceDSNs(t *testing.T) {
	path := writeConfig(t, `
sources:
  - name: orders
    display_name: Orders
    description: User payment orders
    aliases: [Recharge orders]
    keywords: [payment]
    type: mysql
    dsn: ${ORDERS_DSN}
  - name: analytics
    display_name: Analytics
    description: Aggregated business analytics
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
		{Name: "orders", DisplayName: "Orders", Description: "User payment orders", Aliases: []string{"Recharge orders"}, Keywords: []string{"payment"}, Type: "mysql", DSN: "orders-secret-dsn"},
		{Name: "analytics", DisplayName: "Analytics", Description: "Aggregated business analytics", Type: "clickhouse", DSN: "analytics-secret-dsn"},
	}
	if len(config.Sources) != len(want) {
		t.Fatalf("Sources length = %d, want %d", len(config.Sources), len(want))
	}
	for i, source := range config.Sources {
		if !reflect.DeepEqual(source, want[i]) {
			t.Errorf("Sources[%d] = %#v, want %#v", i, source, want[i])
		}
	}
}

func TestLoadNormalizesSourceProfilesForSameEngineInstances(t *testing.T) {
	path := writeConfig(t, `
sources:
  - name: orders
    display_name: " 订单库 "
    description: " 用户充值订单、支付状态和退款记录 "
    aliases: [" 充值库 ", " 共享业务 "]
    keywords: [" 用户充值 ", " 退款 "]
    type: mysql
    dsn: ${ORDERS_DSN}
  - name: logs
    display_name: " 日志库 "
    description: " 服务运行和错误日志 "
    aliases: [" 服务日志 ", " 共享业务 "]
    keywords: [" 报错 "]
    type: mysql
    dsn: ${LOGS_DSN}
  - name: analytics
    display_name: " 分析库 "
    description: " 聚合分析数据 "
    type: clickhouse
    dsn: ${ANALYTICS_DSN}
`)

	cfg, err := Load(path, envLookup(map[string]string{
		"ORDERS_DSN":    "orders-secret",
		"LOGS_DSN":      "logs-secret",
		"ANALYTICS_DSN": "analytics-secret",
	}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := []SourceConfig{
		{Name: "orders", DisplayName: "订单库", Description: "用户充值订单、支付状态和退款记录", Aliases: []string{"充值库", "共享业务"}, Keywords: []string{"用户充值", "退款"}, Type: "mysql", DSN: "orders-secret"},
		{Name: "logs", DisplayName: "日志库", Description: "服务运行和错误日志", Aliases: []string{"服务日志", "共享业务"}, Keywords: []string{"报错"}, Type: "mysql", DSN: "logs-secret"},
		{Name: "analytics", DisplayName: "分析库", Description: "聚合分析数据", Type: "clickhouse", DSN: "analytics-secret"},
	}
	if !reflect.DeepEqual(cfg.Sources, want) {
		t.Errorf("Sources = %#v, want %#v", cfg.Sources, want)
	}
}

func TestValidateSourceProfileRejectsInvalidMetadata(t *testing.T) {
	invalidUTF8 := " " + string([]byte{0xff}) + " "
	tests := []struct {
		name        string
		configure   func(*SourceConfig)
		wantInError string
	}{
		{name: "missing display name", configure: func(source *SourceConfig) { source.DisplayName = " \t " }, wantInError: "display_name is required"},
		{name: "missing description", configure: func(source *SourceConfig) { source.Description = " \t " }, wantInError: "description is required"},
		{name: "invalid UTF-8 display name", configure: func(source *SourceConfig) { source.DisplayName = invalidUTF8 }, wantInError: "display_name must be valid UTF-8"},
		{name: "invalid UTF-8 description", configure: func(source *SourceConfig) { source.Description = invalidUTF8 }, wantInError: "description must be valid UTF-8"},
		{name: "invalid UTF-8 alias", configure: func(source *SourceConfig) { source.Aliases = []string{invalidUTF8} }, wantInError: "aliases[0] must be valid UTF-8"},
		{name: "invalid UTF-8 keyword", configure: func(source *SourceConfig) { source.Keywords = []string{invalidUTF8} }, wantInError: "keywords[0] must be valid UTF-8"},
		{name: "display name over rune limit", configure: func(source *SourceConfig) { source.DisplayName = strings.Repeat("名", MaxDisplayNameRunes+1) }, wantInError: "display_name exceeds 80 runes"},
		{name: "description over rune limit", configure: func(source *SourceConfig) { source.Description = strings.Repeat("述", MaxDescriptionRunes+1) }, wantInError: "description exceeds 500 runes"},
		{name: "alias over rune limit", configure: func(source *SourceConfig) { source.Aliases = []string{strings.Repeat("名", MaxProfileItemRunes+1)} }, wantInError: "aliases[0] exceeds 80 runes"},
		{name: "keyword over rune limit", configure: func(source *SourceConfig) { source.Keywords = []string{strings.Repeat("词", MaxProfileItemRunes+1)} }, wantInError: "keywords[0] exceeds 80 runes"},
		{name: "too many aliases", configure: func(source *SourceConfig) { source.Aliases = profileItems("alias", MaxAliases+1) }, wantInError: "aliases exceeds 20 items"},
		{name: "too many keywords", configure: func(source *SourceConfig) { source.Keywords = profileItems("keyword", MaxKeywords+1) }, wantInError: "keywords exceeds 30 items"},
		{name: "empty alias", configure: func(source *SourceConfig) { source.Aliases = []string{" \t "} }, wantInError: "aliases[0] is required"},
		{name: "empty keyword", configure: func(source *SourceConfig) { source.Keywords = []string{" \t "} }, wantInError: "keywords[0] is required"},
		{name: "duplicate aliases after trimming", configure: func(source *SourceConfig) { source.Aliases = []string{"Shared", " Shared "} }, wantInError: "aliases contains duplicate item"},
		{name: "duplicate keywords after trimming", configure: func(source *SourceConfig) { source.Keywords = []string{"Payment", " Payment "} }, wantInError: "keywords contains duplicate item"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := validSourceProfile()
			tt.configure(&source)

			err := validateSourceProfile(&source)
			if err == nil {
				t.Fatalf("validateSourceProfile() error = nil, want error containing %q", tt.wantInError)
			}
			if !strings.Contains(err.Error(), `source "orders"`) || !strings.Contains(err.Error(), tt.wantInError) {
				t.Errorf("validateSourceProfile() error = %q, want source name and %q", err, tt.wantInError)
			}
		})
	}
}

func TestValidateSourceProfileAcceptsLimitsAndCaseSensitiveItems(t *testing.T) {
	source := SourceConfig{
		Name:        "orders",
		DisplayName: strings.Repeat("名", MaxDisplayNameRunes),
		Description: strings.Repeat("述", MaxDescriptionRunes),
		Aliases:     profileItems("alias", MaxAliases),
		Keywords:    profileItems("keyword", MaxKeywords),
	}
	source.Aliases[0] = strings.Repeat("别", MaxProfileItemRunes)
	source.Keywords[0] = strings.Repeat("词", MaxProfileItemRunes)
	source.Aliases[1], source.Aliases[2] = "Shared", "shared"
	source.Keywords[1], source.Keywords[2] = "Payment", "payment"

	if err := validateSourceProfile(&source); err != nil {
		t.Fatalf("validateSourceProfile() error = %v", err)
	}
}

func TestLoadRejectsSourceMetadataBeforeResolvingDSN(t *testing.T) {
	const secretDSN = "mysql://user:super-secret-password@host/orders"
	path := writeConfig(t, `
sources:
  - name: orders
    display_name: " "
    description: 用户充值订单
    type: mysql
    dsn: ${SECRET_DSN}
`)

	lookedUp := false
	_, err := Load(path, func(string) (string, bool) {
		lookedUp = true
		return secretDSN, true
	})
	if err == nil {
		t.Fatal("Load() error = nil, want profile validation error")
	}
	if lookedUp {
		t.Fatal("Load() resolved the DSN before validating source metadata")
	}
	if !strings.Contains(err.Error(), `source "orders" display_name`) {
		t.Errorf("Load() error = %q, want source and field", err)
	}
	if strings.Contains(err.Error(), secretDSN) || strings.Contains(err.Error(), "super-secret-password") || strings.Contains(err.Error(), "${SECRET_DSN}") {
		t.Errorf("Load() error leaked DSN data: %v", err)
	}
}

func TestLoadAcceptsStrictMode(t *testing.T) {
	path := writeConfig(t, `
mode: strict
sources:
  - name: orders
    display_name: Orders
    description: User payment orders
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
    display_name: Orders
    description: User payment orders
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
    display_name: Orders
    description: User payment orders
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
    display_name: Orders
    description: User payment orders
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
    display_name: Orders
    description: User payment orders
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
    display_name: Orders
    description: User payment orders
    type: mysql
    dsn: ${ORDERS_DSN}
`)

	assertLoadError(t, path, envLookup(map[string]string{"ORDERS_DSN": "orders-dsn"}))
}

func TestLoadRejectsUnknownSourceType(t *testing.T) {
	path := writeConfig(t, `
sources:
  - name: orders
    display_name: Orders
    description: User payment orders
    type: postgres
    dsn: ${ORDERS_DSN}
`)

	assertLoadError(t, path, envLookup(map[string]string{"ORDERS_DSN": "orders-dsn"}))
}

func TestLoadRejectsDuplicateSourceNames(t *testing.T) {
	path := writeConfig(t, `
sources:
  - name: orders
    display_name: Orders
    description: User payment orders
    type: mysql
    dsn: ${ORDERS_DSN}
  - name: orders
    display_name: Analytics
    description: Aggregated business analytics
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
			path := writeConfig(t, "sources:\n  - name: orders\n    display_name: Orders\n    description: User payment orders\n    type: mysql\n    dsn: "+dsn+"\n")
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
    display_name: Orders
    description: User payment orders
    type: mysql
    dsn: ${ORDERS_DSN}
`)

	assertLoadErrorContains(t, path, envLookup(map[string]string{"ORDERS_DSN": "orders-dsn"}), "source name is required")
}

func TestLoadRejectsEmptyDSN(t *testing.T) {
	path := writeConfig(t, `
sources:
  - name: orders
    display_name: Orders
    description: User payment orders
    type: mysql
    dsn: ""
`)

	assertLoadErrorContains(t, path, envLookup(nil), "DSN is required")
}

func TestLoadValidatesAllSourcesBeforeResolvingDSNs(t *testing.T) {
	const secretDSN = "mysql://user:super-secret-password@host/orders"
	path := writeConfig(t, `
sources:
  - name: orders
    display_name: Orders
    description: User payment orders
    type: mysql
    dsn: ${SECRET_DSN}
  - name: analytics
    display_name: " "
    description: Aggregated business analytics
    type: clickhouse
    dsn: ${ANALYTICS_DSN}
`)

	lookupCalls := 0
	_, err := Load(path, func(string) (string, bool) {
		lookupCalls++
		return secretDSN, true
	})
	if err == nil {
		t.Fatal("Load() error = nil, want profile validation error")
	}
	if lookupCalls != 0 {
		t.Fatalf("lookupEnv calls = %d, want 0 before all sources are validated", lookupCalls)
	}
	if !strings.Contains(err.Error(), `source "analytics" display_name`) {
		t.Errorf("Load() error = %q, want later source profile validation error", err)
	}
	if strings.Contains(err.Error(), secretDSN) || strings.Contains(err.Error(), "super-secret-password") {
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

func validSourceProfile() SourceConfig {
	return SourceConfig{
		Name:        "orders",
		DisplayName: "Orders",
		Description: "User payment orders",
		Aliases:     []string{"Recharge orders"},
		Keywords:    []string{"payment"},
	}
}

func profileItems(prefix string, count int) []string {
	items := make([]string, count)
	for i := range items {
		items[i] = fmt.Sprintf("%s-%d", prefix, i)
	}
	return items
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
