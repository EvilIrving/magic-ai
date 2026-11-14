package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"magic/internal/adapters/pi"
	"magic/internal/backup"
)

// An unchanged target keeps its inode, mtime and mode: apply really writes
// nothing, which is what a plan's `unchanged` promises.
func TestApplySkipsUnchangedTarget(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	const body = `{"model":"kimi"}`
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "models.json"), body)
	target := filepath.Join(userHome, ".pi", "agent", "models.json")
	writeFile(t, target, body)
	if err := os.Chmod(target, 0o600); err != nil {
		t.Fatal(err)
	}
	stamp := time.Unix(1600000000, 0)
	if err := os.Chtimes(target, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}

	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome}
	if _, err := ApplyAdapter(pi.New(), opts, ""); err != nil {
		t.Fatal(err)
	}

	after, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(stamp) {
		t.Fatalf("mtime changed: %v, want %v", after.ModTime(), stamp)
	}
	if !os.SameFile(before, after) {
		t.Fatal("apply replaced the file instead of skipping an unchanged target")
	}
	if after.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %04o, want 0600", after.Mode().Perm())
	}
}

// Same content, wrong permission: apply fixes the mode and leaves the content
// (and its mtime) alone.
func TestApplyFixesModeWithoutRewriting(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	const body = `{"model":"kimi"}`
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "models.json"), body)
	target := filepath.Join(userHome, ".pi", "agent", "models.json")
	writeFile(t, target, body) // 0644 from writeFile
	stamp := time.Unix(1600000000, 0)
	if err := os.Chtimes(target, stamp, stamp); err != nil {
		t.Fatal(err)
	}

	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome}
	if _, err := ApplyAdapter(pi.New(), opts, ""); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %04o, want 0600", info.Mode().Perm())
	}
	if !info.ModTime().Equal(stamp) {
		t.Fatalf("content was rewritten for a mode fix: mtime %v", info.ModTime())
	}
}

// D5: the partial merge is field-level. A nested object the user already has
// (mcpServers) keeps its entries; the library only adds or wins on its own keys.
func TestApplyClaudePartialDeepMergesUserKeys(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "claude", "claude.json.partial.json"),
		`{"mcpServers":{"docs":{"command":"npx"}},"env":{"MAGIC":"1"}}`)
	writeFile(t, filepath.Join(userHome, ".claude.json"),
		`{"mcpServers":{"mine":{"command":"x"}},"env":{"USER":"2"},"projects":{"/p":{}}}`)

	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome}
	if _, err := applyClaudePartial(opts, ""); err != nil {
		t.Fatal(err)
	}

	var root map[string]any
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(userHome, ".claude.json"))), &root); err != nil {
		t.Fatal(err)
	}
	servers, _ := root["mcpServers"].(map[string]any)
	if _, ok := servers["mine"]; !ok {
		t.Fatalf("user MCP server dropped: %v", servers)
	}
	if _, ok := servers["docs"]; !ok {
		t.Fatalf("library MCP server missing: %v", servers)
	}
	env, _ := root["env"].(map[string]any)
	if env["USER"] != "2" || env["MAGIC"] != "1" {
		t.Fatalf("env not merged field by field: %v", env)
	}
	if _, ok := root["projects"]; !ok {
		t.Fatalf("unrelated key dropped: %v", root)
	}
}

// The claude partial merge is recorded like every other write: update with a
// copy under claude/_claude.json, mode 0600.
func TestApplyClaudePartialRecordsManifest(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "claude", "claude.json.partial.json"),
		`{"mcpServers":{"docs":{"command":"npx"}}}`)
	const original = `{"projects":{}}`
	writeFile(t, filepath.Join(userHome, ".claude.json"), original)

	backupDir, err := NewBackupDir(magicHome)
	if err != nil {
		t.Fatal(err)
	}
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome}
	if _, err := applyClaudePartial(opts, backupDir); err != nil {
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
	if e.Adapter != "claude" || e.Rel != "_claude.json" || e.Action != backup.ActionUpdate {
		t.Fatalf("entry = %+v", e)
	}
	if e.Backup != filepath.Join("claude", "_claude.json") || e.Mode != "0600" {
		t.Fatalf("entry = %+v, want claude/_claude.json at 0600", e)
	}
	if e.Target != filepath.Join(userHome, ".claude.json") {
		t.Fatalf("target = %s", e.Target)
	}
	if got := readFile(t, filepath.Join(backupDir, e.Backup)); got != original {
		t.Fatalf("backup = %q, want the pre-apply content", got)
	}
}

// Nothing to write means nothing to back up: an unchanged target must not leave a
// copy in the backup set, so `backup: false` in a plan stays true.
func TestApplyDoesNotBackUpUnchangedTarget(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	const body = `{"model":"kimi"}`
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "models.json"), body)
	target := filepath.Join(userHome, ".pi", "agent", "models.json")
	writeFile(t, target, body)
	if err := os.Chmod(target, 0o600); err != nil {
		t.Fatal(err)
	}

	backupDir, err := NewBackupDir(magicHome)
	if err != nil {
		t.Fatal(err)
	}
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome}
	if _, err := ApplyAdapter(pi.New(), opts, backupDir); err != nil {
		t.Fatal(err)
	}

	m, err := backup.Read(backupDir)
	if err != nil {
		t.Fatal(err)
	}
	if m != nil && len(m.Entries) != 0 {
		t.Fatalf("unchanged target was recorded: %+v", m.Entries)
	}
	if files, err := os.ReadDir(backupDir); err != nil {
		t.Fatal(err)
	} else if len(files) != 0 {
		t.Fatalf("backup dir is not empty: %v", files)
	}
}
