package mcp

import "testing"

func boolPtr(b bool) *bool { return &b }

func TestTranslateStdioPerStyle(t *testing.T) {
	s := Server{Command: "npx", Args: []string{"-y", "pkg"}, Env: map[string]string{"K": "v"}}
	cases := []struct {
		style string
		keys  []string
	}{
		{StyleClaude, []string{"command", "args", "env"}},
		{StyleCodex, []string{"command", "args", "env", "enabled"}},
		{StyleGrok, []string{"command", "args", "env"}},
		{StylePi, []string{"command", "args", "env", "disabled"}},
		{StyleCopilot, []string{"command", "args", "env", "type"}},
		{StyleGemini, []string{"command", "args", "env"}},
	}
	for _, c := range cases {
		entry, ok, reason := Translate(s, c.style)
		if !ok {
			t.Fatalf("%s: not ok: %s", c.style, reason)
		}
		if entry["command"] != "npx" {
			t.Fatalf("%s: command=%v", c.style, entry["command"])
		}
		for _, k := range c.keys {
			if _, exists := entry[k]; !exists {
				t.Errorf("%s: missing key %q in %v", c.style, k, entry)
			}
		}
	}
}

func TestTranslateOpenCodeLocalCommandIsArray(t *testing.T) {
	entry, ok, _ := Translate(Server{Command: "npx", Args: []string{"-y", "x"}}, StyleOpenCode)
	if !ok {
		t.Fatal("not ok")
	}
	if entry["type"] != "local" {
		t.Fatalf("type=%v", entry["type"])
	}
	cmd, _ := entry["command"].([]any)
	if len(cmd) != 3 || cmd[0] != "npx" {
		t.Fatalf("command=%v", entry["command"])
	}
	if entry["enabled"] != true {
		t.Fatalf("enabled=%v", entry["enabled"])
	}
}

func TestTranslateGeminiCwd(t *testing.T) {
	e, ok, _ := Translate(Server{Command: "npx", Cwd: "/tmp"}, StyleGemini)
	if !ok || e["cwd"] != "/tmp" {
		t.Fatalf("%v", e)
	}
}

func TestTranslateRemotePerStyle(t *testing.T) {
	s := Server{Transport: "http", URL: "https://x/mcp", Headers: map[string]string{"Authorization": "Bearer t"}}
	if e, ok, _ := Translate(s, StyleClaude); !ok || e["type"] != "http" || e["url"] != "https://x/mcp" {
		t.Fatalf("claude: %v", e)
	}
	if e, ok, _ := Translate(s, StyleGemini); !ok || e["httpUrl"] != "https://x/mcp" {
		t.Fatalf("gemini http: %v", e)
	}
	sse := Server{Transport: "sse", URL: "https://x/sse"}
	if e, ok, _ := Translate(sse, StyleGemini); !ok || e["url"] != "https://x/sse" {
		t.Fatalf("gemini sse: %v", e)
	}
	if _, ok, reason := Translate(sse, StyleCodex); ok || reason == "" {
		t.Fatalf("codex sse should be unsupported: ok=%v reason=%q", ok, reason)
	}
	if e, ok, _ := Translate(s, StyleOpenCode); !ok || e["type"] != "remote" {
		t.Fatalf("opencode remote: %v", e)
	}
	if _, ok, reason := Translate(s, StylePi); ok || reason == "" {
		t.Fatalf("pi remote should be unsupported: ok=%v reason=%q", ok, reason)
	}
}

func TestTranslateEnabledFlag(t *testing.T) {
	off := false
	e, ok, _ := Translate(Server{Command: "x", Enabled: boolPtr(off)}, StylePi)
	if !ok || e["disabled"] != true {
		t.Fatalf("pi disabled: %v", e)
	}
	e, ok, _ = Translate(Server{Command: "x", Enabled: boolPtr(off)}, StyleCodex)
	if !ok || e["enabled"] != false {
		t.Fatalf("codex enabled: %v", e)
	}
}

func TestTranslateV1LongTailStyles(t *testing.T) {
	stdio := Server{Command: "npx", Args: []string{"-y", "pkg"}}
	remote := Server{Transport: "http", URL: "https://x/mcp", Headers: map[string]string{"A": "B"}}
	sse := Server{Transport: "sse", URL: "https://x/sse"}

	// Cursor: no type on remote, plain stdio.
	if e, ok, _ := Translate(remote, StyleCursor); !ok || e["url"] != "https://x/mcp" || e["type"] != nil {
		t.Fatalf("cursor remote: %v", e)
	}
	if e, ok, _ := Translate(stdio, StyleCursor); !ok || e["command"] != "npx" {
		t.Fatalf("cursor stdio: %v", e)
	}

	// Antigravity: remote key is serverUrl.
	if e, ok, _ := Translate(remote, StyleAntigravity); !ok || e["serverUrl"] != "https://x/mcp" {
		t.Fatalf("antigravity remote: %v", e)
	}

	// Kimi: sse adds transport=sse, http stays url-only.
	if e, ok, _ := Translate(sse, StyleKimi); !ok || e["transport"] != "sse" || e["url"] != "https://x/sse" {
		t.Fatalf("kimi sse: %v", e)
	}
	if e, ok, _ := Translate(remote, StyleKimi); !ok || e["transport"] != nil {
		t.Fatalf("kimi http: %v", e)
	}

	// Qoder / Cline: typed remote.
	if e, ok, _ := Translate(remote, StyleQoder); !ok || e["type"] != "http" {
		t.Fatalf("qoder: %v", e)
	}
	if e, ok, _ := Translate(remote, StyleCline); !ok || e["type"] != "streamableHttp" {
		t.Fatalf("cline http: %v", e)
	}
	if e, ok, _ := Translate(sse, StyleCline); !ok || e["type"] != "sse" {
		t.Fatalf("cline sse: %v", e)
	}
}

func TestTargetsCoverV1LongTail(t *testing.T) {
	for _, id := range []string{"cursor", "antigravity-cli", "kimi-code", "qoder", "qoder-cn", "cline"} {
		if _, ok := TargetFor(id); !ok {
			t.Errorf("missing MCP target for %s", id)
		}
	}
}

func TestTranslateResearchedStyles(t *testing.T) {
	stdio := Server{Command: "npx", Args: []string{"-y", "pkg"}}
	remote := Server{Transport: "http", URL: "https://x/mcp", Headers: map[string]string{"A": "B"}}

	// Kiro: disabled flag on both shapes.
	if e, ok, _ := Translate(stdio, StyleKiro); !ok || e["disabled"] != false {
		t.Fatalf("kiro stdio: %v", e)
	}
	if e, ok, _ := Translate(remote, StyleKiro); !ok || e["url"] != "https://x/mcp" || e["disabled"] != false {
		t.Fatalf("kiro remote: %v", e)
	}

	// ZCode: enable flag + typed remote.
	if e, ok, _ := Translate(remote, StyleZCode); !ok || e["enable"] != true || e["type"] != "http" {
		t.Fatalf("zcode remote: %v", e)
	}
	if e, ok, _ := Translate(stdio, StyleZCode); !ok || e["command"] != "npx" {
		t.Fatalf("zcode stdio: %v", e)
	}

	// Zed: stdio only.
	if e, ok, _ := Translate(stdio, StyleZed); !ok || e["command"] != "npx" {
		t.Fatalf("zed stdio: %v", e)
	}
	if _, ok, reason := Translate(remote, StyleZed); ok || reason == "" {
		t.Fatalf("zed remote should be unsupported: ok=%v reason=%q", ok, reason)
	}
}

func TestTargetsCoverResearchedBatch(t *testing.T) {
	for _, id := range []string{"kiro", "droid", "zcode", "qwen-code", "iflow-cli", "kilo", "zed"} {
		if _, ok := TargetFor(id); !ok {
			t.Errorf("missing MCP target for %s", id)
		}
	}
}

func TestTranslateRolloutStyles(t *testing.T) {
	stdio := Server{Command: "npx", Args: []string{"-y", "pkg"}, Cwd: "/tmp"}
	remote := Server{Transport: "http", URL: "https://x/mcp", Headers: map[string]string{"A": "B"}}
	sse := Server{Transport: "sse", URL: "https://x/sse"}
	off := false

	// Oh My Pi: mcpServers with enabled + remote support.
	if e, ok, _ := Translate(stdio, StyleOMP); !ok || e["command"] != "npx" || e["cwd"] != "/tmp" || e["enabled"] != true {
		t.Fatalf("omp stdio: %v", e)
	}
	if e, ok, _ := Translate(remote, StyleOMP); !ok || e["url"] != "https://x/mcp" || e["enabled"] != true {
		t.Fatalf("omp remote: %v", e)
	}
	if e, ok, _ := Translate(remote, StyleOMP); !ok || e["headers"] == nil {
		t.Fatalf("omp headers: %v", e)
	}
	if e, ok, _ := Translate(Server{Command: "x", Enabled: boolPtr(off)}, StyleOMP); !ok || e["enabled"] != false {
		t.Fatalf("omp disabled: %v", e)
	}

	// Hermes: mcp_servers shape, url + headers on remote.
	if e, ok, _ := Translate(stdio, StyleHermes); !ok || e["command"] != "npx" || e["enabled"] != true {
		t.Fatalf("hermes stdio: %v", e)
	}
	if e, ok, _ := Translate(remote, StyleHermes); !ok || e["url"] != "https://x/mcp" || e["headers"] == nil {
		t.Fatalf("hermes remote: %v", e)
	}

	// DimAgent: typed http remote; sse is unsupported.
	if e, ok, _ := Translate(remote, StyleDimAgent); !ok || e["type"] != "http" || e["enabled"] != true {
		t.Fatalf("dimagent remote: %v", e)
	}
	if e, ok, _ := Translate(stdio, StyleDimAgent); !ok || e["command"] != "npx" || e["enabled"] != true {
		t.Fatalf("dimagent stdio: %v", e)
	}
	if _, ok, reason := Translate(sse, StyleDimAgent); ok || reason == "" {
		t.Fatalf("dimagent sse should be unsupported: ok=%v reason=%q", ok, reason)
	}

	// OpenClaw: mcp.servers entry, transport on remote.
	if e, ok, _ := Translate(stdio, StyleOpenClaw); !ok || e["command"] != "npx" {
		t.Fatalf("openclaw stdio: %v", e)
	}
	if e, ok, _ := Translate(remote, StyleOpenClaw); !ok || e["transport"] != "streamable-http" {
		t.Fatalf("openclaw remote: %v", e)
	}
	if e, ok, _ := Translate(sse, StyleOpenClaw); !ok || e["transport"] != "sse" {
		t.Fatalf("openclaw sse: %v", e)
	}
}

func TestTargetsCoverRolloutBatch(t *testing.T) {
	for _, id := range []string{"omp", "hermes", "mimocode", "dimagent", "openclaw"} {
		if _, ok := TargetFor(id); !ok {
			t.Errorf("missing MCP target for %s", id)
		}
	}
}
