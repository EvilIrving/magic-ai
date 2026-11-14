package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeProviderFixture writes the provider layer apply reads: one resolvable
// provider with an anthropic and an openai-chat endpoint, plus its stored key.
func writeProviderFixture(t *testing.T, magicHome string) {
	t.Helper()
	writeFileAt(t, filepath.Join(magicHome, "providers.yaml"), `active: example
providers:
  example:
    key_env: EXAMPLE_API_KEY
    pick: [example-model]
    endpoints:
      - id: anthropic
        base_url: https://api.example.com/anthropic
        protocol: anthropic-messages
      - id: openai-chat
        base_url: https://api.example.com/v1
        protocol: openai-chat-completions
`)
	writeFileAt(t, filepath.Join(magicHome, "secrets", "providers.json"), `{"EXAMPLE_API_KEY":"sk-live-abcdef123456"}`)
}

// readJSONFile decodes a JSON object from disk.
func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	var root map[string]any
	if err := json.Unmarshal([]byte(readFileAt(t, path)), &root); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return root
}

func readFileAt(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// writeJSONFile rewrites path as indented JSON with a trailing newline.
func writeJSONFile(t *testing.T, path string, root map[string]any) {
	t.Helper()
	raw, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFileAt(t, path, string(raw)+"\n")
}

// apply → verify is clean once the provider projection is part of the plan, and
// deleting a projected env key is drift.
func TestVerifyProviderClaudeAfterApply(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeProviderFixture(t, magicHome)
	runApply(t, magicHome, userHome, "claude")

	if err := runVerify(t, magicHome, userHome, "claude"); err != nil {
		t.Fatalf("verify after apply = %v, want no drift", err)
	}

	settings := filepath.Join(userHome, ".claude", "settings.json")
	root := readJSONFile(t, settings)
	env := root["env"].(map[string]any)
	if _, ok := env["ANTHROPIC_AUTH_TOKEN"]; !ok {
		t.Fatalf("apply did not project the provider env: %v", env)
	}
	delete(env, "ANTHROPIC_AUTH_TOKEN")
	writeJSONFile(t, settings, root)

	err := runVerify(t, magicHome, userHome, "claude")
	if err == nil || !strings.Contains(err.Error(), "漂移") {
		t.Fatalf("verify with a deleted provider env key = %v, want drift", err)
	}
}

// The Pi provider layer writes models.json and models.magic.json; a dirty
// sidecar is drift, and a clean one is not.
func TestVerifyProviderPiAfterApply(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeProviderFixture(t, magicHome)
	runApply(t, magicHome, userHome, "pi")

	if err := runVerify(t, magicHome, userHome, "pi"); err != nil {
		t.Fatalf("verify after apply = %v, want no drift", err)
	}
	sidecar := filepath.Join(userHome, ".pi", "agent", "models.magic.json")
	root := readJSONFile(t, sidecar)
	if _, ok := root["providers"]; !ok {
		t.Fatalf("apply did not write the sidecar: %v", root)
	}
	root["strayKey"] = "user edit"
	writeJSONFile(t, sidecar, root)

	err := runVerify(t, magicHome, userHome, "pi")
	if err == nil || !strings.Contains(err.Error(), "漂移") {
		t.Fatalf("verify with a dirty models.magic.json = %v, want drift", err)
	}
}

// The MCP projection into ~/.claude.json is part of the plan: clean after apply,
// drift once the server block changes.
func TestVerifyMCPAfterApply(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFileAt(t, filepath.Join(magicHome, "packs", "mcp", "mcp.yaml"), `mcp:
  docs:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-docs"]
`)
	runApply(t, magicHome, userHome, "claude")

	if err := runVerify(t, magicHome, userHome, "claude"); err != nil {
		t.Fatalf("verify after apply = %v, want no drift", err)
	}

	claudeJSON := filepath.Join(userHome, ".claude.json")
	root := readJSONFile(t, claudeJSON)
	servers := root["mcpServers"].(map[string]any)
	if _, ok := servers["docs"]; !ok {
		t.Fatalf("apply did not project the MCP server: %v", servers)
	}
	servers["docs"] = map[string]any{"command": "npx", "args": []any{"--edited"}}
	writeJSONFile(t, claudeJSON, root)

	err := runVerify(t, magicHome, userHome, "claude")
	if err == nil || !strings.Contains(err.Error(), "漂移") {
		t.Fatalf("verify with an edited server block = %v, want drift", err)
	}
}

// The opencode provider switch is part of the plan now: apply → verify is clean,
// and verify checks the file the switch and the MCP projection both write.
func TestVerifyOpencodeProviderAfterApply(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	// opencode honors $XDG_CONFIG_HOME; pin it inside this test's home so the
	// paths asserted below are the ones apply writes.
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(userHome, ".config"))
	writeProviderFixture(t, magicHome)
	writeFileAt(t, filepath.Join(magicHome, "packs", "harness", "opencode", "AGENTS.md"), "# OpenCode\n")
	writeFileAt(t, filepath.Join(magicHome, "packs", "mcp", "mcp.yaml"), "mcp:\n  docs:\n    command: npx\n")
	if err := os.MkdirAll(filepath.Join(userHome, ".config", "opencode"), 0o755); err != nil {
		t.Fatal(err)
	}
	runApply(t, magicHome, userHome, "opencode")

	out, err := runVerifyOutput(t, magicHome, userHome, verifyArgs(userHome, "opencode", false)...)
	if err != nil {
		t.Fatalf("verify after apply = %v, want no drift:\n%s", err, out)
	}
	// The phrase of the old "plan cannot compute this" notice must not come back.
	if strings.Contains(out, "未覆"+"盖") {
		t.Fatalf("verify still reports an uncovered projection:\n%s", out)
	}

	opencodeJSON := filepath.Join(userHome, ".config", "opencode", "opencode.json")
	root := readJSONFile(t, opencodeJSON)
	providers, ok := root["provider"].(map[string]any)
	if !ok {
		t.Fatalf("apply did not project the provider block: %v", root)
	}
	if _, ok := providers["magic-example"]; !ok {
		t.Fatalf("apply did not project provider.magic-example: %v", providers)
	}

	out, err = runVerifyOutput(t, magicHome, userHome, verifyArgs(userHome, "opencode", true)...)
	if err != nil {
		t.Fatalf("verify --json = %v, want no drift:\n%s", err, out)
	}
	var report struct {
		Checked int `json:"checked"`
		Drift   []struct {
			Rel  string `json:"rel"`
			Kind string `json:"kind"`
		} `json:"drift"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("verify --json is not JSON: %v\n%s", err, out)
	}
	if report.Checked < 2 {
		t.Fatalf("checked = %d, want the mirrored AGENTS.md and opencode.json: the provider switch is predicted now", report.Checked)
	}
	if len(report.Drift) != 0 {
		t.Fatalf("drift = %+v, want none", report.Drift)
	}

	// A provider block edited on disk is drift, not an uncovered projection.
	delete(providers, "magic-example")
	writeJSONFile(t, opencodeJSON, root)
	err = runVerify(t, magicHome, userHome, "opencode")
	if err == nil || !strings.Contains(err.Error(), "漂移") {
		t.Fatalf("verify with a removed provider block = %v, want drift", err)
	}
}
