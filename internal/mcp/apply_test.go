package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"magic/internal/backup"
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
	logs, err := Project(tgt, home, "", f, false, "")
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
	if _, err := Project(tgt, home, "", f, false, ""); err != nil {
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
	logs, err := Project(tgt, home, "", f, false, "")
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
	logs, err := Project(tgt, home, "", f, false, "")
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
	if _, err := Project(tgt, home, "", f, false, ""); err != nil {
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
	if _, err := Project(tgt, home, "", f, false, ""); err != nil {
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
	xdg := filepath.Join(home, "xdg-config")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	if err := os.MkdirAll(filepath.Join(xdg, "mimocode"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(xdg, "mimocode", "mimocode.json")
	if err := os.WriteFile(path, []byte(`{"model":"mimo"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	f := &File{MCP: map[string]Server{"ctx": {Command: "npx", Args: []string{"-y", "pkg"}}}}
	tgt, _ := TargetFor("mimocode")
	if _, err := Project(tgt, home, "", f, false, ""); err != nil {
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
	logs, err := Project(tgt, home, "", f, false, "")
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
	logs, err := Project(tgt, home, "", f, false, "")
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
	if _, err := Project(tgt, home, "", f, false, ""); err != nil {
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

// The backup entry keeps the target's position under the harness's mirror root,
// so Kiro's settings/mcp.json is restored to settings/mcp.json, not to a flat
// mcp.json at the root.
func TestProjectRecordsManifestWithSubdirectory(t *testing.T) {
	home := t.TempDir()
	mirrorRoot := filepath.Join(home, ".kiro")
	target := filepath.Join(mirrorRoot, "settings", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	const original = `{"mcpServers":{"mine":{"command":"x"}}}`
	if err := os.WriteFile(target, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	backupDir := filepath.Join(t.TempDir(), "20260101T000000Z")

	f := &File{MCP: map[string]Server{"ctx": {Command: "npx"}}}
	tgt, _ := TargetFor("kiro")
	if _, err := Project(tgt, home, mirrorRoot, f, false, backupDir); err != nil {
		t.Fatal(err)
	}

	m, err := backup.Read(backupDir)
	if err != nil || m == nil {
		t.Fatalf("manifest = %+v err=%v", m, err)
	}
	if len(m.Entries) != 1 {
		t.Fatalf("entries = %+v", m.Entries)
	}
	e := m.Entries[0]
	if e.Adapter != "kiro" || e.Rel != filepath.Join("settings", "mcp.json") || e.Action != backup.ActionUpdate {
		t.Fatalf("entry = %+v", e)
	}
	if e.Backup != filepath.Join("kiro", "settings", "mcp.json") {
		t.Fatalf("backup = %q, want the subdirectory preserved", e.Backup)
	}
	raw, err := os.ReadFile(filepath.Join(backupDir, e.Backup))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != original {
		t.Fatalf("backup = %q, want the pre-projection content", raw)
	}
}

// A projection that creates the file records a create entry, so rollback can
// delete it.
func TestProjectRecordsCreateEntry(t *testing.T) {
	home := t.TempDir()
	backupDir := filepath.Join(t.TempDir(), "20260101T000000Z")

	f := &File{MCP: map[string]Server{"ctx": {Command: "npx"}}}
	tgt, _ := TargetFor("pi")
	if _, err := Project(tgt, home, filepath.Join(home, ".pi", "agent"), f, false, backupDir); err != nil {
		t.Fatal(err)
	}

	m, err := backup.Read(backupDir)
	if err != nil || m == nil {
		t.Fatalf("manifest = %+v err=%v", m, err)
	}
	if len(m.Entries) != 1 {
		t.Fatalf("entries = %+v", m.Entries)
	}
	if e := m.Entries[0]; e.Action != backup.ActionCreate || e.Backup != "" {
		t.Fatalf("entry = %+v, want create without a copy", e)
	}
}

// ~/.claude.json lives outside Claude's mirror root; both writers must record the
// same name so one target has one manifest entry.
func TestProjectClaudeJSONUsesApplyName(t *testing.T) {
	home := t.TempDir()
	backupDir := filepath.Join(t.TempDir(), "20260101T000000Z")
	target := filepath.Join(home, ".claude.json")
	if err := os.WriteFile(target, []byte(`{"projects":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	f := &File{MCP: map[string]Server{"ctx": {Command: "npx"}}}
	tgt, _ := TargetFor("claude")
	if _, err := Project(tgt, home, filepath.Join(home, ".claude"), f, false, backupDir); err != nil {
		t.Fatal(err)
	}
	m, err := backup.Read(backupDir)
	if err != nil || m == nil {
		t.Fatalf("manifest = %+v err=%v", m, err)
	}
	if len(m.Entries) != 1 || m.Entries[0].Rel != backup.ClaudeJSONRel {
		t.Fatalf("entries = %+v, want rel %q", m.Entries, backup.ClaudeJSONRel)
	}
	if m.Entries[0].Backup != filepath.Join("claude", backup.ClaudeJSONRel) {
		t.Fatalf("backup = %q", m.Entries[0].Backup)
	}
}

// ProjectBytes is what plan calls, so it must return exactly the bytes Project
// writes — for every dialect, and without touching the target.
func TestProjectBytesMatchesProject(t *testing.T) {
	f := &File{MCP: map[string]Server{"ctx": {Command: "npx", Args: []string{"-y", "pkg"}}}}
	cases := []struct {
		harness  string
		existing string
	}{
		{"claude", `{"mcpServers":{"mine":{"command":"x"}},"theme":"dark"}`},
		{"codex", "model_provider = \"openai\"\n\n[mcp_servers.user]\ncommand = \"u\"\n"},
		{"hermes", "# Hermes config\nmodel: nous\n\ntelemetry: false\n"},
		{"zcode", `{"theme":"dark"}`},
		{"openclaw", `{"gateway":{"port":1}}`},
	}
	for _, tc := range cases {
		t.Run(tc.harness, func(t *testing.T) {
			tgt, ok := TargetFor(tc.harness)
			if !ok {
				t.Fatalf("no target for %s", tc.harness)
			}
			planned, applied := t.TempDir(), t.TempDir()
			for _, home := range []string{planned, applied} {
				path := tgt.Path(home)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(tc.existing), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			path, out, entries, err := ProjectBytes(tgt, planned, f)
			if err != nil {
				t.Fatal(err)
			}
			if entries != 1 {
				t.Fatalf("entries = %d, want 1", entries)
			}
			if got, _ := os.ReadFile(tgt.Path(planned)); string(got) != tc.existing {
				t.Fatalf("ProjectBytes wrote to the target:\n%s", got)
			}
			if _, err := Project(tgt, applied, "", f, false, ""); err != nil {
				t.Fatal(err)
			}
			if path != tgt.Path(planned) {
				t.Fatalf("ProjectBytes path = %s, want %s", path, tgt.Path(planned))
			}
			wrote, err := os.ReadFile(tgt.Path(applied))
			if err != nil {
				t.Fatal(err)
			}
			if string(out) != string(wrote) {
				t.Fatalf("ProjectBytes out != apply output\n--- plan ---\n%s\n--- apply ---\n%s", out, wrote)
			}
		})
	}
}

// A lenient target that is not plain JSON has nothing to write: out is nil so
// plan reports no change instead of inventing one.
func TestProjectBytesNilForUntouchedLenientJSON(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".openclaw", "openclaw.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	const original = "{\n  gateway: { port: 1 },\n}\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	tgt, _ := TargetFor("openclaw")
	f := &File{MCP: map[string]Server{"ctx": {Command: "npx"}}}
	_, out, entries, err := ProjectBytes(tgt, home, f)
	if err != nil {
		t.Fatal(err)
	}
	if entries != 1 || out != nil {
		t.Fatalf("entries = %d out = %q, want one server and no write", entries, out)
	}
}

// MergeProjection merges into the bytes a previous layer left, not into the
// file on disk: that is how plan stacks layers.
func TestMergeProjectionUsesGivenLayerOutput(t *testing.T) {
	home := t.TempDir()
	tgt, _ := TargetFor("claude")
	f := &File{MCP: map[string]Server{"ctx": {Command: "npx"}}}
	// The target is missing on disk; the previous layer's output stands in for it.
	out, err := MergeProjection(tgt, tgt.Path(home), f, []byte(`{"mcpServers":{"mine":{"command":"x"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(out, &root); err != nil {
		t.Fatalf("out is not JSON: %v\n%s", err, out)
	}
	servers, _ := root["mcpServers"].(map[string]any)
	if _, ok := servers["mine"]; !ok {
		t.Fatalf("the previous layer's entries were dropped:\n%s", out)
	}
	if _, ok := servers["ctx"]; !ok {
		t.Fatalf("the new server is missing:\n%s", out)
	}
	if _, err := os.Stat(tgt.Path(home)); !os.IsNotExist(err) {
		t.Fatalf("MergeProjection wrote the target (stat err = %v)", err)
	}
}
