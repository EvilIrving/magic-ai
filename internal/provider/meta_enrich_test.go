package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"magic/internal/catalog"
	"magic/internal/modelfile"
	"magic/internal/secrets"
)

func TestGrokBlockUsesMetaContext(t *testing.T) {
	magic := t.TempDir()
	_ = os.MkdirAll(filepath.Join(magic, "registry"), 0o755)
	high := "high"
	xhigh := "xhigh"
	cat := &modelfile.Catalog{
		Version: 1,
		Source:  "test",
		Models: []modelfile.Model{{
			ID:            "gpt-5.6-sol",
			Name:          "GPT-5.6 Sol",
			Provider:      "openai",
			Reasoning:     true,
			ContextWindow: 1050000,
			MaxTokens:     64000,
			ThinkingLevelMap: map[string]*string{
				"high":  &high,
				"xhigh": &xhigh,
			},
		}},
	}
	if err := modelfile.WriteCatalog(magic, cat); err != nil {
		t.Fatal(err)
	}
	req := ApplyRequest{
		Name: "shu",
		Provider: catalog.Provider{
			KeyEnv: "SHU_API_KEY",
			Pick:   []string{"gpt-5.6-sol"},
		},
		Endpoint: catalog.Endpoint{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: catalog.ProtocolOpenAIResponses},
	}
	out, err := SetGrokManagedConfig("", "provider", magic, req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "context_window = 1050000") {
		t.Fatalf("expected meta context: %s", out)
	}
	if !strings.Contains(out, "max_completion_tokens = 64000") {
		t.Fatalf("expected meta max tokens: %s", out)
	}
	if !strings.Contains(out, `name = "GPT-5.6 Sol"`) {
		t.Fatalf("expected meta name: %s", out)
	}
	if strings.Contains(out, "context_window = 256000") {
		t.Fatalf("should not keep hardcoded default when meta present: %s", out)
	}
	if !strings.Contains(out, `value = "xhigh"`) || strings.Contains(out, `value = "low"`) {
		t.Fatalf("efforts should come from thinkingLevelMap: %s", out)
	}
}

func TestGrokBlockFallbackWithoutMeta(t *testing.T) {
	req := ApplyRequest{
		Name: "shu",
		Provider: catalog.Provider{
			KeyEnv: "SHU_API_KEY",
			Pick:   []string{"totally-unknown-model"},
		},
		Endpoint: catalog.Endpoint{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: catalog.ProtocolOpenAIResponses},
	}
	out, err := SetGrokManagedConfig("", "provider", t.TempDir(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "context_window = 256000") {
		t.Fatalf("fallback context: %s", out)
	}
	if !strings.Contains(out, "max_completion_tokens = 128000") {
		t.Fatalf("fallback max: %s", out)
	}
	if !strings.Contains(out, `value = "low"`) {
		t.Fatalf("fallback efforts: %s", out)
	}
}

func TestPiApplyEnrichesModelsJSON(t *testing.T) {
	magic := t.TempDir()
	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(magic, "registry"), 0o755)
	high := "high"
	cat := &modelfile.Catalog{
		Version: 1,
		Source:  "test",
		Models: []modelfile.Model{{
			ID:               "gpt-5.6-sol",
			Name:             "GPT-5.6 Sol",
			Provider:         "openai",
			Reasoning:        true,
			ContextWindow:    1050000,
			MaxTokens:        128000,
			ThinkingLevelMap: map[string]*string{"high": &high},
		}},
	}
	if err := modelfile.WriteCatalog(magic, cat); err != nil {
		t.Fatal(err)
	}
	f := &catalog.File{
		Active: "shu",
		Providers: map[string]catalog.Provider{
			"shu": {
				KeyEnv: "SHU_API_KEY",
				Pick:   []string{"gpt-5.6-sol"},
				Endpoints: []catalog.Endpoint{
					{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: catalog.ProtocolOpenAIResponses},
				},
			},
		},
	}
	ctx := Context{MagicHome: magic, UserHome: home, Catalog: f}
	w := PiWriter{}
	req := ApplyRequest{Name: "shu", Provider: f.Providers["shu"], Endpoint: f.Providers["shu"].Endpoints[0]}
	if _, err := ApplyFiles(ctx, w, req); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(w.ConfigPath(home))
	s := string(raw)
	if !strings.Contains(s, `"contextWindow": 1050000`) {
		t.Fatalf("missing contextWindow: %s", s)
	}
	if !strings.Contains(s, `"thinkingLevelMap"`) {
		t.Fatalf("missing thinkingLevelMap: %s", s)
	}
}

func TestCodexSetsTopLevelModelFromPick(t *testing.T) {
	p := catalog.Provider{KeyEnv: "SHU_API_KEY", Pick: []string{"gpt-5.6-sol"}}
	ep := catalog.Endpoint{BaseURL: "https://api.shu.cool/v1", Protocol: catalog.ProtocolOpenAIResponses}
	out, err := SetCodexManagedConfig("", "shu", "shu", p, ep, "/tmp/models.json", "high")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `model = "gpt-5.6-sol"`) {
		t.Fatalf("expected top-level model: %s", out)
	}
	if !strings.Contains(out, `model_provider = "shu"`) {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, `model_catalog_json = "/tmp/models.json"`) {
		t.Fatalf("expected catalog_json: %s", out)
	}
	if !strings.Contains(out, `model_reasoning_effort = "high"`) {
		t.Fatalf("expected reasoning effort: %s", out)
	}
	if strings.Contains(out, "experimental_bearer_token") || strings.Contains(out, "sk-") {
		t.Fatalf("must not write plaintext tokens: %s", out)
	}
}

func TestCodexApplyWritesAndOfficialRemovesCatalog(t *testing.T) {
	magic := t.TempDir()
	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(magic, "secrets"), 0o700)
	os.Setenv("MAGIC_PASSPHRASE", "test-pass-codex-catalog")
	defer os.Unsetenv("MAGIC_PASSPHRASE")
	if err := secrets.SetProviderSecret(filepath.Join(magic, "secrets"), "SHU_API_KEY", "sk-test-key"); err != nil {
		t.Fatal(err)
	}
	high := "high"
	_ = os.MkdirAll(filepath.Join(magic, "registry"), 0o755)
	cat := &modelfile.Catalog{
		Version: 1,
		Source:  "test",
		Models: []modelfile.Model{{
			ID:               "gpt-5.6-sol",
			Name:             "GPT-5.6 Sol",
			Provider:         "openai",
			Reasoning:        true,
			ContextWindow:    1050000,
			Input:            []string{"text"},
			ThinkingLevelMap: map[string]*string{"high": &high, "max": &high},
		}},
	}
	if err := modelfile.WriteCatalog(magic, cat); err != nil {
		t.Fatal(err)
	}
	f := &catalog.File{
		Active: "shu",
		Providers: map[string]catalog.Provider{
			"shu": {
				KeyEnv: "SHU_API_KEY",
				Pick:   []string{"gpt-5.6-sol"},
				Endpoints: []catalog.Endpoint{
					{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: catalog.ProtocolOpenAIResponses},
				},
			},
		},
	}
	_ = catalog.Save(filepath.Join(magic, "providers.yaml"), f)
	ctx := Context{MagicHome: magic, UserHome: home, Catalog: f}
	if _, err := Use(ctx, HarnessCodex, "company"); err != nil {
		t.Fatal(err)
	}
	modelsPath := CodexModelsPath(home)
	raw, err := os.ReadFile(modelsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"slug": "gpt-5.6-sol"`) {
		t.Fatalf("models.json missing slug: %s", raw)
	}
	if !strings.Contains(string(raw), `"context_window": 1050000`) {
		t.Fatalf("models.json missing context: %s", raw)
	}
	cfg, _ := os.ReadFile(CodexConfigPath(home))
	cs := string(cfg)
	if !strings.Contains(cs, "model_catalog_json") {
		t.Fatalf("missing model_catalog_json: %s", cs)
	}
	if !strings.Contains(cs, `env_key = "SHU_API_KEY"`) {
		t.Fatalf("expected env_key not bearer: %s", cs)
	}
	if strings.Contains(cs, "experimental_bearer_token") || strings.Contains(cs, "sk-test") {
		t.Fatalf("plaintext token leaked: %s", cs)
	}
	if _, err := Use(ctx, HarnessCodex, "official"); err != nil {
		t.Fatal(err)
	}
	if fileExists(modelsPath) {
		t.Fatal("official should delete models.json")
	}
	cfg2, _ := os.ReadFile(CodexConfigPath(home))
	if strings.Contains(string(cfg2), "model_catalog_json") {
		t.Fatalf("official should remove catalog_json: %s", cfg2)
	}
	if GetCodexProvider(string(cfg2)) != "openai" {
		t.Fatalf("%s", cfg2)
	}
}

func TestClaudeThirdPartyWritesAuthTokenAndExtras(t *testing.T) {
	magic := t.TempDir()
	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(magic, "secrets"), 0o700)
	os.Setenv("MAGIC_PASSPHRASE", "test-pass-claude-extras")
	defer os.Unsetenv("MAGIC_PASSPHRASE")
	if err := secrets.SetProviderSecret(filepath.Join(magic, "secrets"), "DEEPSEEK_API_KEY", "sk-ds-key"); err != nil {
		t.Fatal(err)
	}
	max := "max"
	_ = os.MkdirAll(filepath.Join(magic, "registry"), 0o755)
	cat := &modelfile.Catalog{
		Version: 1,
		Source:  "test",
		Models: []modelfile.Model{{
			ID:               "deepseek-v4-pro",
			Name:             "DeepSeek V4 Pro",
			Provider:         "deepseek",
			ContextWindow:    1048576,
			ThinkingLevelMap: map[string]*string{"max": &max, "high": &max},
		}},
	}
	if err := modelfile.WriteCatalog(magic, cat); err != nil {
		t.Fatal(err)
	}
	f := &catalog.File{
		Active: "deepseek",
		Providers: map[string]catalog.Provider{
			"deepseek": {
				KeyEnv: "DEEPSEEK_API_KEY",
				Pick:   []string{"deepseek-v4-pro", "deepseek-v4-flash"},
				Models: map[string]string{
					"default": "deepseek-v4-pro", "opus": "deepseek-v4-pro",
					"sonnet": "deepseek-v4-pro", "haiku": "deepseek-v4-flash",
					"subagent": "deepseek-v4-flash",
				},
				Endpoints: []catalog.Endpoint{
					{ID: "anthropic", BaseURL: "https://api.deepseek.com/anthropic", Protocol: catalog.ProtocolAnthropicMessages},
				},
			},
		},
	}
	_ = catalog.Save(filepath.Join(magic, "providers.yaml"), f)
	ctx := Context{MagicHome: magic, UserHome: home, Catalog: f}
	if _, err := Use(ctx, HarnessClaude, "deepseek"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(ClaudeSettingsPath(home))
	s := string(raw)
	for _, want := range []string{
		`"ANTHROPIC_AUTH_TOKEN": "sk-ds-key"`,
		`"ANTHROPIC_API_KEY": "sk-ds-key"`,
		`"ANTHROPIC_BASE_URL": "https://api.deepseek.com/anthropic"`,
		`"ANTHROPIC_MODEL": "deepseek-v4-pro"`,
		`"CLAUDE_CODE_SUBAGENT_MODEL": "deepseek-v4-flash"`,
		`"CLAUDE_CODE_EFFORT_LEVEL": "max"`,
		`"CLAUDE_CODE_AUTO_COMPACT_WINDOW": "786432"`,
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s in:\n%s", want, s)
		}
	}
}
