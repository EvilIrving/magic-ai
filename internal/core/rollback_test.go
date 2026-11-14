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
	"magic/internal/backup"
)

func writeBackup(t *testing.T, magicHome, stamp, adapterID, rel, content string) {
	t.Helper()
	writeFile(t, filepath.Join(magicHome, "backups", stamp, adapterID, rel), content)
}

// requireLegacyNotice asserts the rollback output says an old backup cannot
// delete the files apply created.
func requireLegacyNotice(t *testing.T, skips []string) {
	t.Helper()
	for _, s := range skips {
		if strings.Contains(s, "旧备份，只恢复文件不删除新建") {
			return
		}
	}
	t.Fatalf("old backup must be announced, res.Skipped = %v", skips)
}

func TestRollbackRestoresBackupContent(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeBackup(t, magicHome, "20260101T000000Z", "pi", "AGENTS.md", "old rules\n")
	target := filepath.Join(userHome, ".pi", "agent", "AGENTS.md")
	writeFile(t, target, "new rules\n")

	res, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	requireLegacyNotice(t, res.Skipped)
	if len(res.Restored) != 1 || res.Restored[0].Path != target {
		t.Fatalf("res.Restored = %+v", res.Restored)
	}
	if got := readFile(t, target); got != "old rules\n" {
		t.Fatalf("content = %q, want the backup", got)
	}
}

func TestRollbackDryRunWritesNothing(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeBackup(t, magicHome, "20260101T000000Z", "pi", "AGENTS.md", "old rules\n")
	target := filepath.Join(userHome, ".pi", "agent", "AGENTS.md")
	writeFile(t, target, "new rules\n")

	res, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Restored) != 1 {
		t.Fatalf("res.Restored = %+v", res.Restored)
	}
	if got := readFile(t, target); got != "new rules\n" {
		t.Fatalf("dry-run changed the file: %q", got)
	}
}

func TestRollbackClaudeJSONLivesOutsideMirror(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeBackup(t, magicHome, "20260101T000000Z", "claude", "_claude.json", `{"old":true}`)
	writeFile(t, filepath.Join(userHome, ".claude.json"), `{"old":false}`)

	if _, err := Rollback([]adapter.Adapter{claude.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(userHome, ".claude.json")); got != `{"old":true}` {
		t.Fatalf("content = %q", got)
	}
}

func TestRollbackSkipsUnknownAdapter(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeBackup(t, magicHome, "20260101T000000Z", "gone", "x.md", "x")
	writeBackup(t, magicHome, "20260101T000000Z", "pi", "AGENTS.md", "old\n")

	res, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Restored) != 1 {
		t.Fatalf("res.Restored = %+v", res.Restored)
	}
	if len(res.Skipped) == 0 {
		t.Fatal("an unknown adapter directory must be reported, not silently dropped")
	}
}

func TestRollbackUsesNewestSetAndHonoursStamp(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeBackup(t, magicHome, "20260101T000000Z", "pi", "AGENTS.md", "first\n")
	writeBackup(t, magicHome, "20260202T000000Z", "pi", "AGENTS.md", "second\n")
	target := filepath.Join(userHome, ".pi", "agent", "AGENTS.md")

	if _, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, target); got != "second\n" {
		t.Fatalf("default rollback used %q, want the newest set", got)
	}
	if _, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome, Stamp: "20260101T000000Z"}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, target); got != "first\n" {
		t.Fatalf("--to used %q, want the named set", got)
	}
	sets, err := ListBackups(magicHome)
	if err != nil {
		t.Fatal(err)
	}
	if len(sets) != 2 || sets[0].Stamp != "20260101T000000Z" || sets[1].Files != 1 {
		t.Fatalf("sets = %+v", sets)
	}
}

func TestRollbackMissingBackupsIsAnError(t *testing.T) {
	if _, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: t.TempDir(), UserHome: t.TempDir()}); err == nil {
		t.Fatal("expected an error when no backup exists")
	}
}

func TestRollbackUnknownStampIsAnError(t *testing.T) {
	magicHome := t.TempDir()
	writeBackup(t, magicHome, "20260101T000000Z", "pi", "AGENTS.md", "x")
	_, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: t.TempDir(), Stamp: "nope"})
	if err == nil {
		t.Fatal("expected an error for an unknown stamp")
	}
	if !os.IsNotExist(err) {
		// The error must name the stamp problem, not a filesystem accident.
		if got := err.Error(); got == "" {
			t.Fatal("empty error")
		}
	}
}

// providerBackupDirs are backups/<harness>/: flat provider copies, not apply
// sets. Rollback must ignore them and `--list` reports them separately.
func TestBackupDirsSeparateProviderFromApplySets(t *testing.T) {
	magicHome := t.TempDir()
	writeBackup(t, magicHome, "20260101T000000Z", "pi", "AGENTS.md", "x")
	writeFile(t, filepath.Join(magicHome, "backups", "workbuddy", "settings.20260101-000000.json"), "{}")

	sets, err := ListBackups(magicHome)
	if err != nil {
		t.Fatal(err)
	}
	if len(sets) != 1 || sets[0].Stamp != "20260101T000000Z" {
		t.Fatalf("sets = %+v, want only the stamped set", sets)
	}
	providers, err := ListProviderBackupDirs(magicHome)
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) != 1 || providers[0] != "workbuddy" {
		t.Fatalf("provider dirs = %v, want [workbuddy]", providers)
	}
}

// A provider directory must not be mistaken for the newest apply set: with only
// provider backups, rollback has nothing to restore.
func TestRollbackIgnoresProviderBackupDir(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "backups", "workbuddy", "settings.20260101-000000.json"), "{}")
	target := filepath.Join(userHome, ".codebuddy", "settings.json")
	writeFile(t, target, `{"user":true}`)

	if _, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome}); err == nil {
		t.Fatal("provider backups alone are not an apply set")
	}
	if got := readFile(t, target); got != `{"user":true}` {
		t.Fatalf("rollback touched the target: %q", got)
	}
}

// apply → rollback must be a full round trip: what apply overwrote comes back,
// what apply created is gone.
func TestApplyThenRollbackRestoresAndDeletes(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "library rules\n")
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "themes", "dark.json"), `{"bg":"#111"}`)
	targetAgents := filepath.Join(userHome, ".pi", "agent", "AGENTS.md")
	targetTheme := filepath.Join(userHome, ".pi", "agent", "themes", "dark.json")
	writeFile(t, targetAgents, "user rules\n")

	backupDir, err := NewBackupDir(magicHome)
	if err != nil {
		t.Fatal(err)
	}
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome}
	if _, err := ApplyAdapter(pi.New(), opts, backupDir); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, targetAgents); got != "library rules\n" {
		t.Fatalf("apply did not write: %q", got)
	}

	m, err := backup.Read(backupDir)
	if err != nil || m == nil {
		t.Fatalf("manifest = %+v err=%v", m, err)
	}
	got := map[string]backup.Entry{}
	for _, e := range m.Entries {
		got[filepath.ToSlash(e.Rel)] = e
	}
	if e := got["AGENTS.md"]; e.Action != backup.ActionUpdate || e.Backup != filepath.Join("pi", "AGENTS.md") || e.Mode != "0644" {
		t.Fatalf("AGENTS.md entry = %+v, want update with a pi/AGENTS.md copy", e)
	}
	if e := got["themes/dark.json"]; e.Action != backup.ActionCreate || e.Backup != "" {
		t.Fatalf("dark.json entry = %+v, want create without a copy", e)
	}
	if e := got["AGENTS.md"]; e.Target != targetAgents {
		t.Fatalf("target = %s, want %s", e.Target, targetAgents)
	}

	res, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range res.Skipped {
		t.Fatalf("res.Skipped = %v", s)
	}
	if len(res.Restored) != 2 {
		t.Fatalf("res.Restored = %+v", res.Restored)
	}
	if got := readFile(t, targetAgents); got != "user rules\n" {
		t.Fatalf("AGENTS.md = %q, want the pre-apply content", got)
	}
	if _, err := os.Stat(targetTheme); !os.IsNotExist(err) {
		t.Fatalf("rollback must delete what apply created (stat err = %v)", err)
	}
}

func TestRollbackManifestDryRunWritesNothing(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "library\n")
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "extra.md"), "created\n")
	targetAgents := filepath.Join(userHome, ".pi", "agent", "AGENTS.md")
	targetExtra := filepath.Join(userHome, ".pi", "agent", "extra.md")
	writeFile(t, targetAgents, "user\n")

	backupDir, err := NewBackupDir(magicHome)
	if err != nil {
		t.Fatal(err)
	}
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome}
	if _, err := ApplyAdapter(pi.New(), opts, backupDir); err != nil {
		t.Fatal(err)
	}

	res, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Restored) != 2 {
		t.Fatalf("res.Restored = %+v", res.Restored)
	}
	if got := readFile(t, targetAgents); got != "library\n" {
		t.Fatalf("dry-run rewrote AGENTS.md: %q", got)
	}
	if got := readFile(t, targetExtra); got != "created\n" {
		t.Fatalf("dry-run deleted extra.md: %q", got)
	}
}

// An old backup has no manifest: rollback restores the copies and says that it
// cannot delete the files apply created.
func TestRollbackLegacyBackupKeepsCreatedFiles(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeBackup(t, magicHome, "20260101T000000Z", "pi", "AGENTS.md", "old rules\n")
	targetAgents := filepath.Join(userHome, ".pi", "agent", "AGENTS.md")
	targetExtra := filepath.Join(userHome, ".pi", "agent", "extra.md")
	writeFile(t, targetAgents, "new rules\n")
	writeFile(t, targetExtra, "created by apply\n")

	if res, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome}); err != nil {
		t.Fatal(err)
	} else {
		requireLegacyNotice(t, res.Skipped)
	}
	if got := readFile(t, targetAgents); got != "old rules\n" {
		t.Fatalf("AGENTS.md = %q, want the backup", got)
	}
	if got := readFile(t, targetExtra); got != "created by apply\n" {
		t.Fatalf("a legacy backup cannot delete created files, extra.md = %q", got)
	}
}

// --list must count manifest entries, not copies: a create-only set has no files
// under the adapter directory but still undoes one target.
func TestListBackupsCountsManifestEntries(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "library\n")
	backupDir, err := NewBackupDir(magicHome)
	if err != nil {
		t.Fatal(err)
	}
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome}
	if _, err := ApplyAdapter(pi.New(), opts, backupDir); err != nil {
		t.Fatal(err)
	}

	sets, err := ListBackups(magicHome)
	if err != nil {
		t.Fatal(err)
	}
	if len(sets) != 1 || sets[0].Files != 1 {
		t.Fatalf("sets = %+v, want one create entry counted", sets)
	}
	if len(sets[0].Adapters) != 1 || sets[0].Adapters[0] != "pi" {
		t.Fatalf("adapters = %v", sets[0].Adapters)
	}
}

// An old MCP backup stored ~/.claude.json as claude/.claude.json. Restoring it as a
// mirror-relative path would create ~/.claude/.claude.json; the name maps to the
// file it came from instead.
func TestRollbackLegacyDotClaudeJSONRestoresHomeFile(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeBackup(t, magicHome, "20260101T000000Z", "claude", ".claude.json", `{"old":true}`)
	writeFile(t, filepath.Join(userHome, ".claude.json"), `{"old":false}`)

	if _, err := Rollback([]adapter.Adapter{claude.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(userHome, ".claude.json")); got != `{"old":true}` {
		t.Fatalf("~/.claude.json = %q, want the legacy backup", got)
	}
	if _, err := os.Stat(filepath.Join(userHome, ".claude", ".claude.json")); !os.IsNotExist(err) {
		t.Fatalf("rollback wrote into the mirror root instead of the home file (err = %v)", err)
	}
}

// With only provider backups, rollback must say why it has nothing to do instead
// of suggesting the apply sets are broken.
func TestRollbackNamesProviderBackupsWhenNoApplySet(t *testing.T) {
	magicHome := t.TempDir()
	writeFile(t, filepath.Join(magicHome, "backups", "pi", "settings.20260101-000000.json"), "{}")
	_, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "provider 备份") {
		t.Fatalf("err = %v, want a provider-backup explanation", err)
	}
}

// writeManifest writes a backup set's manifest by hand, so a test can describe
// entries apply would never produce (an old manifest without fingerprints, a
// path outside the user home, an escaping backup path).
func writeManifest(t *testing.T, magicHome, stamp string, entries []backup.Entry) {
	t.Helper()
	dir := filepath.Join(magicHome, "backups", stamp)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(backup.Manifest{Version: backup.ManifestVersion, Stamp: stamp, Entries: entries}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, backup.ManifestFileName), string(raw)+"\n")
}

// requireSkipContaining asserts one res.Skipped reason carries the wanted text.
func requireSkipContaining(t *testing.T, skips []string, want string) {
	t.Helper()
	for _, s := range skips {
		if strings.Contains(s, want) {
			return
		}
	}
	t.Fatalf("res.Skipped = %v, want a reason containing %q", skips, want)
}

// apply must record what it wrote, so rollback can recognise its own output:
// an update entry keeps the fingerprint of the copy it took, both keep the
// fingerprint of the bytes apply wrote.
func TestApplyRecordsContentFingerprints(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "library rules\n")
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "themes", "dark.json"), `{"bg":"#111"}`)
	targetAgents := filepath.Join(userHome, ".pi", "agent", "AGENTS.md")
	writeFile(t, targetAgents, "user rules\n")

	backupDir, err := NewBackupDir(magicHome)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyAdapter(pi.New(), ApplyOptions{MagicHome: magicHome, UserHome: userHome}, backupDir); err != nil {
		t.Fatal(err)
	}

	m, err := backup.Read(backupDir)
	if err != nil || m == nil {
		t.Fatalf("manifest = %+v err=%v", m, err)
	}
	got := map[string]backup.Entry{}
	for _, e := range m.Entries {
		got[filepath.ToSlash(e.Rel)] = e
	}
	if e := got["AGENTS.md"]; e.Before != FingerprintOf([]byte("user rules\n")) || e.After != FingerprintOf([]byte("library rules\n")) {
		t.Fatalf("AGENTS.md entry = %+v, want the pre-apply and post-apply fingerprints", e)
	}
	if e := got["themes/dark.json"]; e.Before != "" || e.After != FingerprintOf([]byte(`{"bg":"#111"}`)) {
		t.Fatalf("dark.json entry = %+v, want a create entry with only after", e)
	}
}

// apply created the file, then the user wrote in it: rollback must not delete
// their content. --force is the explicit way to delete anyway.
func TestRollbackKeepsCreatedFileTheUserEdited(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	const stamp = "20260101T000000Z"
	target := filepath.Join(userHome, ".pi", "agent", "extra.md")
	writeFile(t, target, "user edit after apply\n")
	writeManifest(t, magicHome, stamp, []backup.Entry{{
		Adapter: "pi", Rel: "extra.md", Target: target,
		Action: backup.ActionCreate, Mode: "0644",
		After: FingerprintOf([]byte("what apply wrote\n")),
	}})

	res, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Restored) != 0 {
		t.Fatalf("res.Restored = %+v, want the edited file left alone", res.Restored)
	}
	requireSkipContaining(t, res.Skipped, "目标在 apply 之后被改动，跳过（加 --force 删除）")
	if got := readFile(t, target); got != "user edit after apply\n" {
		t.Fatalf("target = %q, want the user's content", got)
	}

	res, err = Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) != 0 {
		t.Fatalf("--force must not skip, res.Skipped = %v", res.Skipped)
	}
	if len(res.Restored) != 1 || !res.Restored[0].Delete || res.Restored[0].Path != target {
		t.Fatalf("res.Restored = %+v, want the created file deleted", res.Restored)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("--force must delete what apply created (stat err = %v)", err)
	}
}

// apply overwrote the file, then the user edited it: rollback must not throw the
// edit away. --force restores the pre-apply content anyway.
func TestRollbackKeepsUpdatedFileTheUserEdited(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	const stamp = "20260101T000000Z"
	const before, applied, edited = "pre-apply\n", "apply wrote this\n", "user edited this\n"
	writeBackup(t, magicHome, stamp, "pi", "AGENTS.md", before)
	target := filepath.Join(userHome, ".pi", "agent", "AGENTS.md")
	writeFile(t, target, edited)
	writeManifest(t, magicHome, stamp, []backup.Entry{{
		Adapter: "pi", Rel: "AGENTS.md", Target: target,
		Backup: filepath.Join("pi", "AGENTS.md"),
		Action: backup.ActionUpdate, Mode: "0644",
		Before: FingerprintOf([]byte(before)), After: FingerprintOf([]byte(applied)),
	}})

	res, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Restored) != 0 {
		t.Fatalf("res.Restored = %+v, want the edited file left alone", res.Restored)
	}
	requireSkipContaining(t, res.Skipped, "目标在 apply 之后被改动，跳过（加 --force 覆盖）")
	if got := readFile(t, target); got != edited {
		t.Fatalf("target = %q, want the user's edit", got)
	}

	if res, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome, Force: true}); err != nil {
		t.Fatal(err)
	} else if len(res.Skipped) != 0 {
		t.Fatalf("--force must not skip, res.Skipped = %v", res.Skipped)
	}
	if got := readFile(t, target); got != before {
		t.Fatalf("--force target = %q, want the pre-apply content", got)
	}
}

// The target still holds what apply wrote: rollback restores the copy.
func TestRollbackRestoresUneditedManifestTarget(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	const stamp = "20260101T000000Z"
	const before, applied = "pre-apply\n", "apply wrote this\n"
	writeBackup(t, magicHome, stamp, "pi", "AGENTS.md", before)
	target := filepath.Join(userHome, ".pi", "agent", "AGENTS.md")
	writeFile(t, target, applied)
	writeManifest(t, magicHome, stamp, []backup.Entry{{
		Adapter: "pi", Rel: "AGENTS.md", Target: target,
		Backup: filepath.Join("pi", "AGENTS.md"),
		Action: backup.ActionUpdate, Mode: "0644",
		Before: FingerprintOf([]byte(before)), After: FingerprintOf([]byte(applied)),
	}})

	res, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) != 0 {
		t.Fatalf("an unedited target must restore, res.Skipped = %v", res.Skipped)
	}
	if len(res.Restored) != 1 || res.Restored[0].Path != target {
		t.Fatalf("res.Restored = %+v", res.Restored)
	}
	if got := readFile(t, target); got != before {
		t.Fatalf("target = %q, want the pre-apply content", got)
	}
}

// A manifest entry whose target is outside the user home is not touched at all,
// even with --force: the bounds check is not a content check.
func TestRollbackRefusesTargetOutsideUserHome(t *testing.T) {
	magicHome, userHome, outside := t.TempDir(), t.TempDir(), t.TempDir()
	const stamp = "20260101T000000Z"
	const before, applied = "pre-apply\n", "apply wrote this\n"
	writeBackup(t, magicHome, stamp, "pi", "AGENTS.md", before)
	target := filepath.Join(outside, "AGENTS.md")
	writeFile(t, target, applied)
	writeManifest(t, magicHome, stamp, []backup.Entry{{
		Adapter: "pi", Rel: "AGENTS.md", Target: target,
		Backup: filepath.Join("pi", "AGENTS.md"),
		Action: backup.ActionUpdate, Mode: "0644",
		Before: FingerprintOf([]byte(before)), After: FingerprintOf([]byte(applied)),
	}})

	for _, force := range []bool{false, true} {
		res, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome, Force: force})
		if err != nil {
			t.Fatalf("force=%v: %v", force, err)
		}
		if len(res.Restored) != 0 {
			t.Fatalf("force=%v res.Restored = %+v, want nothing touched", force, res.Restored)
		}
		requireSkipContaining(t, res.Refused, "不在 user home")
		if len(res.Skipped) != 0 {
			t.Fatalf("force=%v: a bounds refusal must not be offered to --force, skipped = %v", force, res.Skipped)
		}
		// Every entry was refused, yet the summary can still name the set.
		if res.Stamp != stamp {
			t.Fatalf("force=%v: stamp = %q, want %q", force, res.Stamp, stamp)
		}
		if got := readFile(t, target); got != applied {
			t.Fatalf("force=%v: target outside the user home was rewritten: %q", force, got)
		}
	}
}

// A backup path that climbs out of the set is refused: rollback must not read a
// file from anywhere else on disk.
func TestRollbackRefusesBackupPathEscape(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	const stamp = "20260101T000000Z"
	const applied, escaped = "apply wrote this\n", "outside the set\n"
	outsideCopy := filepath.Join(magicHome, "escape", "AGENTS.md")
	writeFile(t, outsideCopy, escaped)
	target := filepath.Join(userHome, ".pi", "agent", "AGENTS.md")
	writeFile(t, target, applied)
	writeManifest(t, magicHome, stamp, []backup.Entry{{
		Adapter: "pi", Rel: "AGENTS.md", Target: target,
		Backup: filepath.Join("..", "..", "escape", "AGENTS.md"),
		Action: backup.ActionUpdate, Mode: "0644",
		After: FingerprintOf([]byte(applied)),
	}})

	res, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Restored) != 0 {
		t.Fatalf("res.Restored = %+v, want an escaping backup path refused", res.Restored)
	}
	requireSkipContaining(t, res.Refused, "逃出备份集合")
	if len(res.Skipped) != 0 {
		t.Fatalf("an escaping backup path must not be offered to --force, skipped = %v", res.Skipped)
	}
	if got := readFile(t, target); got != applied {
		t.Fatalf("target = %q, want it untouched", got)
	}
}

// A manifest written before fingerprints existed cannot be checked: rollback
// skips it and says --force is the way to replay it.
func TestRollbackSkipsManifestWithoutFingerprints(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	const stamp = "20260101T000000Z"
	writeBackup(t, magicHome, stamp, "pi", "AGENTS.md", "pre-apply\n")
	targetAgents := filepath.Join(userHome, ".pi", "agent", "AGENTS.md")
	targetExtra := filepath.Join(userHome, ".pi", "agent", "extra.md")
	writeFile(t, targetAgents, "apply wrote this\n")
	writeFile(t, targetExtra, "created by apply\n")
	writeManifest(t, magicHome, stamp, []backup.Entry{
		{
			Adapter: "pi", Rel: "AGENTS.md", Target: targetAgents,
			Backup: filepath.Join("pi", "AGENTS.md"),
			Action: backup.ActionUpdate, Mode: "0644",
		},
		{
			Adapter: "pi", Rel: "extra.md", Target: targetExtra,
			Action: backup.ActionCreate, Mode: "0644",
		},
	})

	res, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Restored) != 0 {
		t.Fatalf("res.Restored = %+v, want an unchecked old manifest res.Skipped", res.Restored)
	}
	requireSkipContaining(t, res.Skipped, "manifest 没有内容指纹（旧备份）")
	if len(res.Skipped) != 2 {
		t.Fatalf("res.Skipped = %v, want one reason per entry", res.Skipped)
	}
	if !strings.Contains(strings.Join(res.Skipped, "\n"), "--force") {
		t.Fatalf("res.Skipped = %v, want the --force hint", res.Skipped)
	}
	if got := readFile(t, targetAgents); got != "apply wrote this\n" {
		t.Fatalf("AGENTS.md = %q, want it untouched", got)
	}
	if got := readFile(t, targetExtra); got != "created by apply\n" {
		t.Fatalf("extra.md = %q, want it untouched", got)
	}

	if res, err := Rollback([]adapter.Adapter{pi.New()}, RollbackOptions{MagicHome: magicHome, UserHome: userHome, Force: true}); err != nil {
		t.Fatal(err)
	} else if len(res.Skipped) != 0 {
		t.Fatalf("--force must replay the old manifest, res.Skipped = %v", res.Skipped)
	}
	if got := readFile(t, targetAgents); got != "pre-apply\n" {
		t.Fatalf("--force AGENTS.md = %q, want the copy", got)
	}
	if _, err := os.Stat(targetExtra); !os.IsNotExist(err) {
		t.Fatalf("--force must delete the created file (stat err = %v)", err)
	}
}
