package mcp

import (
	"strings"
	"testing"
)

func TestUpsertTOMLSectionsAddsAndPreserves(t *testing.T) {
	text := "model_provider = \"openai\"\n\n[mcp_servers.user]\ncommand = \"u\"\ncustom = \"keep\"\n"
	entries := map[string]map[string]any{
		"context7": {"command": "npx", "args": []any{"-y", "pkg"}, "enabled": true},
	}
	out := upsertTOMLSections(text, "mcp_servers", entries)
	for _, want := range []string{
		`model_provider = "openai"`,
		"[mcp_servers.user]",
		`custom = "keep"`,
		"[mcp_servers.context7]",
		`command = "npx"`,
		`args = ["-y", "pkg"]`,
		"enabled = true",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestUpsertTOMLSectionReplacesManagedKeysKeepsExtras(t *testing.T) {
	text := "[mcp_servers.s]\ncommand = \"old\"\nuser_extra = \"x\"\n"
	entries := map[string]map[string]any{"s": {"command": "new"}}
	out := upsertTOMLSections(text, "mcp_servers", entries)
	if !strings.Contains(out, `command = "new"`) || strings.Contains(out, `command = "old"`) {
		t.Fatalf("managed key not replaced:\n%s", out)
	}
	if !strings.Contains(out, `user_extra = "x"`) {
		t.Fatalf("user extra key dropped:\n%s", out)
	}
	if strings.Count(out, "[mcp_servers.s]") != 1 {
		t.Fatalf("duplicate section:\n%s", out)
	}
}

func TestTOMLInlineTableAndQuotedKey(t *testing.T) {
	entries := map[string]map[string]any{
		"a.b": {"url": "https://x", "http_headers": map[string]any{"Authorization": "Bearer t"}},
	}
	out := upsertTOMLSections("", "mcp_servers", entries)
	if !strings.Contains(out, `[mcp_servers."a.b"]`) {
		t.Fatalf("quoted key missing:\n%s", out)
	}
	if !strings.Contains(out, `http_headers = { Authorization = "Bearer t" }`) {
		t.Fatalf("inline table wrong:\n%s", out)
	}
}
