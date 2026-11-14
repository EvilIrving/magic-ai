package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"magic/internal/backup"
	"magic/internal/core"
)

// writeRollbackManifest writes a backup manifest by hand, so a test can describe
// an entry apply would never produce.
func writeRollbackManifest(t *testing.T, magicHome, stamp string, entries []backup.Entry) {
	t.Helper()
	dir := filepath.Join(magicHome, "backups", stamp)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(backup.Manifest{Version: backup.ManifestVersion, Stamp: stamp, Entries: entries}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFileAt(t, filepath.Join(dir, backup.ManifestFileName), string(raw)+"\n")
}

func runRollback(t *testing.T, magicHome, userHome string, args ...string) string {
	t.Helper()
	args = append([]string{"--home", userHome}, args...)
	out, err := runMagic(t, magicHome, cmdRollback(), args...)
	if err != nil {
		t.Fatalf("rollback %v: %v\n%s", args, err, out)
	}
	return out
}

// When every entry is skipped the summary still names the backup set it replayed:
// "备份 -" told the user nothing.
func TestRollbackSummaryKeepsStampWhenEverythingIsSkipped(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFileAt(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "library rules\n")
	runApply(t, magicHome, userHome, "pi")

	// The user edits the file after apply: rollback must not delete it.
	target := filepath.Join(userHome, ".pi", "agent", "AGENTS.md")
	writeFileAt(t, target, "user edit after apply\n")

	sets, err := core.ListBackups(magicHome)
	if err != nil || len(sets) != 1 {
		t.Fatalf("backups = %+v err=%v", sets, err)
	}
	stamp := sets[0].Stamp

	out := runRollback(t, magicHome, userHome)
	if !strings.Contains(out, "备份 "+stamp) {
		t.Fatalf("summary must name the backup set it used:\n%s", out)
	}
	if strings.Contains(out, "备份 -") {
		t.Fatalf("summary fell back to \"备份 -\":\n%s", out)
	}
	if !strings.Contains(out, "加 --force 强制按 manifest 执行") {
		t.Fatalf("a content skip must offer --force:\n%s", out)
	}
}

// A path refusal is not a content skip: it is reported apart, and the --force
// hint does not apply to it.
func TestRollbackReportsRefusalsApartFromForceableSkips(t *testing.T) {
	magicHome, userHome, outside := t.TempDir(), t.TempDir(), t.TempDir()
	const stamp = "20260101T000000Z"
	const applied = "apply wrote this\n"
	target := filepath.Join(outside, "AGENTS.md")
	writeFileAt(t, target, applied)
	writeRollbackManifest(t, magicHome, stamp, []backup.Entry{{
		Adapter: "pi", Rel: "AGENTS.md", Target: target,
		Action: backup.ActionUpdate, Mode: "0644",
		After: core.FingerprintOf([]byte(applied)),
	}})

	out := runRollback(t, magicHome, userHome, "--force")
	if !strings.Contains(out, "拒绝 1 个（见上），--force 也不执行") {
		t.Fatalf("a refused entry must be reported apart from --force:\n%s", out)
	}
	if strings.Contains(out, "加 --force 强制按 manifest 执行") {
		t.Fatalf("a bounds refusal must not suggest --force:\n%s", out)
	}
	if !strings.Contains(out, "备份 "+stamp) {
		t.Fatalf("summary must name the backup set:\n%s", out)
	}
	if got := readFileAt(t, target); got != applied {
		t.Fatalf("target outside the user home was rewritten: %q", got)
	}
}
