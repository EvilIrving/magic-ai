package registry

import "testing"

// TestV1CoreAdaptersPresent locks the usage-driven v1 core coverage
// (docs/supported-harnesses.md → 用量驱动的 v1 覆盖范围, decision D20).
func TestV1CoreAdaptersPresent(t *testing.T) {
	want := []string{
		"codex", "claude", "cursor", "pi", "grok", "antigravity-cli",
		"opencode", "kimi-code", "copilot", "workbuddy", "trae",
		"qoder", "qoder-cn", "cline",
	}
	for _, id := range want {
		a, ok := ByID(id)
		if !ok {
			t.Errorf("v1 core adapter %q missing", id)
			continue
		}
		if !a.Implemented() {
			t.Errorf("%q must be implemented", id)
		}
		if !a.Caps().Skills && !a.Caps().MCP {
			t.Errorf("%q must support skills or MCP", id)
		}
	}
}

// TestResearchedAdaptersPresent covers the 2026-09-13 research batch.
func TestResearchedAdaptersPresent(t *testing.T) {
	for _, id := range []string{"kiro", "droid", "zcode", "qwen-code", "iflow-cli", "kilo", "zed"} {
		a, ok := ByID(id)
		if !ok {
			t.Errorf("researched adapter %q missing", id)
			continue
		}
		if !a.Implemented() || !a.Caps().MCP {
			t.Errorf("%q must be implemented with MCP", id)
		}
	}
}

func TestNoDuplicateAdapterIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, a := range List() {
		id := string(a.ID())
		if seen[id] {
			t.Errorf("duplicate adapter id %q", id)
		}
		seen[id] = true
	}
}
