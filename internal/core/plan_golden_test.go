package core

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"magic/internal/adapter"
	"magic/internal/adapters/claude"
	"magic/internal/adapters/codex"
	"magic/internal/adapters/pi"
	"magic/internal/adapters/stub"
)

var updateGolden = flag.Bool("update-golden", false, "rewrite docs/plan.example.json")

// TestPlanExampleDocument keeps docs/plan.example.json equal to a real plan.
// Run with -update-golden after changing the plan schema.
func TestPlanExampleDocument(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "# Pi rules\n")
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "themes", "dark.json"), `{"bg":"#111"}`)
	writeFile(t, filepath.Join(userHome, ".pi", "agent", "themes", "dark.json"), `{"bg":"#111"}`)
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "models.json"),
		`{"providers":{"kimi":{"api":"https://api.moonshot.cn/v1","apiKey":"sk-live"}}}`)
	writeFile(t, filepath.Join(userHome, ".pi", "agent", "models.json"),
		`{"providers":{"kimi":{"api":"https://old.example/v1","apiKey":"sk-live"}}}`)
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "claude", "settings.json"),
		`{"env":{"ANTHROPIC_BASE_URL":"https://relay.example","ANTHROPIC_AUTH_TOKEN":"sk-live"}}`)
	writeFile(t, filepath.Join(userHome, ".claude", "settings.json"),
		`{"env":{"ANTHROPIC_AUTH_TOKEN":"sk-live"},"statusLine":{"type":"command"}}`)
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "claude", "claude.json.partial.json"),
		`{"mcpServers":{"docs":{"command":"npx"}}}`)
	writeFile(t, filepath.Join(userHome, ".claude.json"),
		`{"projects":{}}`)
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "codex", "config.toml"),
		"model = \"kimi-k2\"\n")
	writeFile(t, filepath.Join(userHome, ".codex", "config.toml"),
		"model = \"gpt-5\"\napproval_policy = \"never\"\n")
	// One provider and one MCP server, so the example shows stacked layers
	// (mirror → provider / mcp) and their fingerprints.
	writeFile(t, filepath.Join(magicHome, "providers.yaml"), `active: deepseek
providers:
  deepseek:
    key_env: DEEPSEEK_API_KEY
    pick: [deepseek-flash]
    endpoints:
      - id: anthropic
        base_url: https://api.deepseek.com/anthropic
        protocol: anthropic-messages
      - id: openai-chat
        base_url: https://api.deepseek.com/v1
        protocol: openai-chat-completions
`)
	writeFile(t, filepath.Join(magicHome, "secrets", "providers.json"),
		`{"DEEPSEEK_API_KEY":"sk-live-example"}`)
	writeFile(t, filepath.Join(magicHome, "packs", "mcp", "mcp.yaml"), `mcp:
  docs:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-docs"]
`)

	adapters := []adapter.Adapter{pi.New(), claude.New(), codex.New(), stub.New("nope", "Nope")}
	p, err := BuildPlan(adapters, ApplyOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	p.MagicHome = "<magic-home>"
	p.UserHome = "<user-home>"
	for i := range p.Adapters {
		for j := range p.Adapters[i].Changes {
			p.Adapters[i].Changes[j].Path = "<user-home>" + trimPrefix(p.Adapters[i].Changes[j].Path, userHome)
		}
	}
	out, err := p.JSON()
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("..", "..", "docs", "plan.example.json")
	if *updateGolden {
		if err := os.WriteFile(golden, out, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read %s: %v (run: go test ./internal/core -update-golden)", golden, err)
	}
	if string(out) != string(want) {
		t.Fatalf("docs/plan.example.json is stale; rerun with -update-golden\n--- got ---\n%s", out)
	}
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}
