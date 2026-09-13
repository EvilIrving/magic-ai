package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectJSONPreservesUserEntries(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"user-server":{"command":"u"}},"other":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	f := &File{MCP: map[string]Server{
		"context7": {Command: "npx", Args: []string{"-y", "pkg"}, Env: map[string]string{"K": "v"}},
	}}
	tgt, _ := TargetFor("claude")
	logs, err := Project(tgt, home, f, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) == 0 {
		t.Fatal("expected a log line")
	}
	raw, _ := os.ReadFile(filepath.Join(home, ".claude.json"))
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatal(err)
	}
	if root["other"] != float64(1) {
		t.Fatal("unrelated key dropped")
	}
	servers := root["mcpServers"].(map[string]any)
	if _, ok := servers["user-server"]; !ok {
		t.Fatal("user MCP server dropped")
	}
	entry, ok := servers["context7"].(map[string]any)
	if !ok || entry["command"] != "npx" {
		t.Fatalf("context7 not written: %v", servers["context7"])
	}
}

func TestProjectTOMLPreservesConfig(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "model_provider = \"openai\"\n\n[mcp_servers.user]\ncommand = \"u\"\n"
	if err := os.WriteFile(filepath.Join(home, ".codex", "config.toml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	f := &File{MCP: map[string]Server{"ctx": {Command: "npx"}}}
	tgt, _ := TargetFor("codex")
	if _, err := Project(tgt, home, f, false, ""); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	s := string(raw)
	if !strings.Contains(s, `model_provider = "openai"`) || !strings.Contains(s, "[mcp_servers.user]") {
		t.Fatalf("existing config dropped:\n%s", s)
	}
	if !strings.Contains(s, "[mcp_servers.ctx]") {
		t.Fatalf("new server missing:\n%s", s)
	}
}

func TestProjectSkipsDisabledWithoutEnabledField(t *testing.T) {
	home := t.TempDir()
	off := false
	f := &File{MCP: map[string]Server{"x": {Command: "c", Enabled: &off}}}
	tgt, _ := TargetFor("claude")
	logs, err := Project(tgt, home, f, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude.json")); err == nil {
		t.Fatal("disabled server must not create a file for a no-enabled-flag harness")
	}
	if !strings.Contains(strings.Join(logs, "\n"), "n/a") {
		t.Fatalf("expected n/a log, got %v", logs)
	}
}

func TestProjectPiRemoteSkipped(t *testing.T) {
	home := t.TempDir()
	f := &File{MCP: map[string]Server{"remote": {Transport: "http", URL: "https://x"}}}
	tgt, _ := TargetFor("pi")
	logs, err := Project(tgt, home, f, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(logs, "\n"), "n/a") {
		t.Fatalf("expected n/a, got %v", logs)
	}
}

func TestLoadCanonical(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.yaml")
	yaml := "mcp:\n  a:\n    transport: stdio\n    command: npx\n    enabled: false\n  b:\n    transport: sse\n    url: https://x\n"
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.MCP) != 2 || f.MCP["a"].IsEnabled() || !f.MCP["b"].IsEnabled() {
		t.Fatalf("%+v", f.MCP)
	}
	if f.MCP["b"].TransportKind() != TransportSSE || !f.MCP["b"].IsRemote() {
		t.Fatalf("b transport: %+v", f.MCP["b"])
	}
}

func TestProjectZCodeNestedKey(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".zcode", "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zcode", "cli", "config.json"), []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	f := &File{MCP: map[string]Server{"ctx": {Command: "npx"}}}
	tgt, _ := TargetFor("zcode")
	if _, err := Project(tgt, home, f, false, ""); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(home, ".zcode", "cli", "config.json"))
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatal(err)
	}
	if root["theme"] != "dark" {
		t.Fatal("unrelated key dropped")
	}
	mcpObj, _ := root["mcp"].(map[string]any)
	servers, _ := mcpObj["servers"].(map[string]any)
	if _, ok := servers["ctx"]; !ok {
		t.Fatalf("nested mcp.servers not written: %s", raw)
	}
}

func TestInspectNestedKey(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".zcode", "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(home, ".zcode", "cli", "config.json"), []byte(`{"mcp":{"servers":{"here":{"command":"x"}}}}`), 0o600)
	tgt, _ := TargetFor("zcode")
	present, exists, err := Inspect(tgt, home, []string{"here", "absent"})
	if err != nil || !exists || !present["here"] || present["absent"] {
		t.Fatalf("present=%v exists=%v err=%v", present, exists, err)
	}
}

func TestInspect(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"present":{"command":"x"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	tgt, _ := TargetFor("claude")
	present, exists, err := Inspect(tgt, home, []string{"present", "absent"})
	if err != nil || !exists {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
	if !present["present"] || present["absent"] {
		t.Fatalf("%v", present)
	}
}

func TestProjectYAMLPreservesConfigAndComments(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".hermes"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "# Hermes config\nmodel: nous\n\nmcp_servers:\n  user-tools:\n    command: user\n\ntelemetry: false\n"
	path := filepath.Join(home, ".hermes", "config.yaml")
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	f := &File{MCP: map[string]Server{"ctx": {Command: "npx", Args: []string{"-y", "pkg"}}}}
	tgt, _ := TargetFor("hermes")
	if _, err := Project(tgt, home, f, false, ""); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	s := string(raw)
	if !strings.Contains(s, "# Hermes config") {
		t.Fatalf("comment lost:\n%s", s)
	}
	if !strings.Contains(s, "model: nous") || !strings.Contains(s, "telemetry: false") {
		t.Fatalf("unrelated key lost:\n%s", s)
	}
	if !strings.Contains(s, "user-tools") {
		t.Fatalf("user MCP server lost:\n%s", s)
	}
	if !strings.Contains(s, "command: npx") {
		t.Fatalf("new server missing:\n%s", s)
	}
}

func TestProjectMimoCodeUsesOpenCodeDialect(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".config", "mimocode"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".config", "mimocode", "mimocode.json")
	if err := os.WriteFile(path, []byte(`{"model":"mimo"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	f := &File{MCP: map[string]Server{"ctx": {Command: "npx", Args: []string{"-y", "pkg"}}}}
	tgt, _ := TargetFor("mimocode")
	if _, err := Project(tgt, home, f, false, ""); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatal(err)
	}
	if root["model"] != "mimo" {
		t.Fatal("unrelated key dropped")
	}
	mcp, _ := root["mcp"].(map[string]any)
	entry, _ := mcp["ctx"].(map[string]any)
	if entry["type"] != "local" {
		t.Fatalf("expected opencode local dialect: %v", entry)
	}
}

func TestProjectDimAgentSkipsSSE(t *testing.T) {
	home := t.TempDir()
	f := &File{MCP: map[string]Server{
		"ok":  {Transport: "http", URL: "https://x/mcp"},
		"bad": {Transport: "sse", URL: "https://x/sse"},
	}}
	tgt, _ := TargetFor("dimagent")
	logs, err := Project(tgt, home, f, false, "")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(home, ".dim", "v2", "mcp.json"))
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatal(err)
	}
	servers, _ := root["mcpServers"].(map[string]any)
	if _, ok := servers["ok"]; !ok {
		t.Fatalf("ok server missing: %v", servers)
	}
	if _, ok := servers["bad"]; ok {
		t.Fatalf("sse server should be skipped: %v", servers)
	}
	found := false
	for _, l := range logs {
		if strings.Contains(l, "sse") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an n/a log for sse: %v", logs)
	}
}

func TestProjectOpenClawLeavesJSON5Untouched(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".openclaw"), 0o755); err != nil {
		t.Fatal(err)
	}
	// JSON5-ish: unquoted key + trailing comma, which StripJSONC does not repair.
	original := "{\n  gateway: { port: 1 },\n}\n"
	path := filepath.Join(home, ".openclaw", "openclaw.json")
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	f := &File{MCP: map[string]Server{"ctx": {Command: "npx"}}}
	tgt, _ := TargetFor("openclaw")
	logs, err := Project(tgt, home, f, false, "")
	if err != nil {
		t.Fatalf("lenient projection must not error: %v", err)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != original {
		t.Fatalf("file was modified:\n%s", raw)
	}
	found := false
	for _, l := range logs {
		if strings.Contains(l, "left untouched") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an n/a log: %v", logs)
	}
}

func TestProjectOpenClawWritesPlainJSON(t *testing.T) {
	home := t.TempDir()
	f := &File{MCP: map[string]Server{"ctx": {Command: "npx", Args: []string{"-y", "pkg"}}}}
	tgt, _ := TargetFor("openclaw")
	if _, err := Project(tgt, home, f, false, ""); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(home, ".openclaw", "openclaw.json"))
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatal(err)
	}
	mcp, _ := root["mcp"].(map[string]any)
	if _, ok := mcp["servers"].(map[string]any)["ctx"]; !ok {
		t.Fatalf("nested mcp.servers entry missing: %v", root)
	}
}
