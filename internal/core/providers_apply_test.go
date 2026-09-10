package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"magic/internal/adapter"
	"magic/internal/adapters/claude"
	"magic/internal/adapters/pi"
	"magic/internal/secrets"
)

func TestApplyProvidersClaudeAndPi(t *testing.T) {
	magicHome := t.TempDir()
	userHome := t.TempDir()
	_ = os.MkdirAll(filepath.Join(userHome, ".claude"), 0o755)
	_ = os.MkdirAll(filepath.Join(userHome, ".pi", "agent"), 0o755)
	_ = os.WriteFile(filepath.Join(userHome, ".claude", "settings.json"), []byte(`{"permissions":{"allow":["Bash"]},"env":{"OTHER":"1"}}`), 0o600)
	_ = os.WriteFile(filepath.Join(userHome, ".pi", "agent", "models.json"), []byte(`{"providers":{"keep":{"baseUrl":"https://k","api":"openai-completions","apiKey":"$K","models":[]}}}`), 0o600)

	providersYAML := `
active: shu
providers:
  shu:
    key_env: SHU_API_KEY
    pick: [gpt-5.6-sol]
    endpoints:
      - id: openai-responses
        base_url: https://api.shu.cool/v1
        protocol: openai-responses
      - id: anthropic
        base_url: https://api.shu.cool
        protocol: anthropic-messages
`
	_ = os.WriteFile(filepath.Join(magicHome, "providers.yaml"), []byte(providersYAML), 0o644)
	secDir := filepath.Join(magicHome, "secrets")
	_ = os.MkdirAll(secDir, 0o700)
	pass := "test-passphrase-providers"
	if err := secrets.SetProviderSecret(secDir, pass, "SHU_API_KEY", "sk-SHU-REAL-KEY99"); err != nil {
		t.Fatal(err)
	}

	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome, Passphrase: pass}
	var list []adapter.Adapter
	list = append(list, claude.New(), pi.New())

	logs, err := ApplyProviders(list, opts)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(logs, "\n")
	if !strings.Contains(joined, "anthropic-messages") {
		t.Fatalf("claude log missing protocol:\n%s", joined)
	}
	if !strings.Contains(joined, "openai-responses") {
		t.Fatalf("pi log missing protocol:\n%s", joined)
	}

	raw, err := os.ReadFile(filepath.Join(userHome, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatal(err)
	}
	if _, ok := settings["permissions"]; !ok {
		t.Fatal("permissions wiped")
	}
	env := settings["env"].(map[string]any)
	if env["ANTHROPIC_BASE_URL"] != "https://api.shu.cool" {
		t.Fatalf("%v", env)
	}
	if env["ANTHROPIC_AUTH_TOKEN"] != "sk-SHU-REAL-KEY99" {
		t.Fatalf("auth token: %v", env)
	}
	if env["ANTHROPIC_API_KEY"] != "sk-SHU-REAL-KEY99" {
		t.Fatalf("api key: %v", env)
	}
	if env["OTHER"] != "1" {
		t.Fatalf("OTHER wiped")
	}

	rawPi, err := os.ReadFile(filepath.Join(userHome, ".pi", "agent", "models.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rawPi), "sk-SHU") {
		t.Fatalf("pi must use $KEY_ENV not plaintext:\n%s", rawPi)
	}
	var models map[string]any
	if err := json.Unmarshal(rawPi, &models); err != nil {
		t.Fatal(err)
	}
	provs := models["providers"].(map[string]any)
	if _, ok := provs["keep"]; !ok {
		t.Fatal("keep wiped")
	}
	magic := provs["magic-shu"].(map[string]any)
	if magic["apiKey"] != "$SHU_API_KEY" || magic["api"] != "openai-responses" {
		t.Fatalf("%v", magic)
	}
}

func TestApplyProvidersSkipsClaudeWithoutSecret(t *testing.T) {
	magicHome := t.TempDir()
	userHome := t.TempDir()
	_ = os.MkdirAll(filepath.Join(userHome, ".claude"), 0o755)
	_ = os.WriteFile(filepath.Join(magicHome, "providers.yaml"), []byte(`
providers:
  shu:
    key_env: SHU_API_KEY
    endpoints:
      - id: anthropic
        base_url: https://api.shu.cool
        protocol: anthropic-messages
`), 0o644)
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome}
	logs, err := ApplyProviders([]adapter.Adapter{claude.New()}, opts)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(logs, "\n")
	if !strings.Contains(joined, "secret") {
		t.Fatalf("%s", joined)
	}
}
