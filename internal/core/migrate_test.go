package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"magic/internal/backup"
)

// legacyMagicYAML is a magic.yaml from before schema_version existed, with a
// comment and a user field migrate must not lose.
const legacyMagicYAML = `# 我的配置库
version: "1.0"
# 只投影这两家
adapters:
  enabled:
    - pi   # 自定义顺序
    - claude
active_profile: work
`

func writeMagicYAML(t *testing.T, magicHome, content string) {
	t.Helper()
	writeFile(t, filepath.Join(magicHome, "magic.yaml"), content)
}

// migrate adds schema_version without re-marshalling the file: the user's
// comments, other fields and field order are still there afterwards.
func TestMigrateAddsSchemaVersionKeepingUserFieldsAndComments(t *testing.T) {
	magicHome := t.TempDir()
	writeMagicYAML(t, magicHome, legacyMagicYAML)

	res, err := MigrateLibrary(MigrateOptions{MagicHome: magicHome})
	if err != nil {
		t.Fatal(err)
	}
	if res.SchemaVersion != 0 || res.TargetVersion != LibrarySchemaVersion {
		t.Fatalf("res = %+v, want 0 → %d", res, LibrarySchemaVersion)
	}
	if res.AlreadyCurrent {
		t.Fatal("a schema-0 library is not current")
	}
	if len(res.Changes) != 1 || res.Changes[0].Field != "schema_version" ||
		res.Changes[0].From != "0" || res.Changes[0].To != "1" {
		t.Fatalf("changes = %+v, want one schema_version 0 → 1", res.Changes)
	}

	got := readFile(t, filepath.Join(magicHome, "magic.yaml"))
	for _, want := range []string{
		"# 我的配置库",
		"# 只投影这两家",
		"- pi",
		"# 自定义顺序",
		"active_profile: work",
		"schema_version: 1",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("magic.yaml lost %q:\n%s", want, got)
		}
	}
	// init writes schema_version right after version; migrate keeps that order.
	if strings.Index(got, "schema_version") > strings.Index(got, "adapters:") {
		t.Fatalf("schema_version must stay above adapters:\n%s", got)
	}

	cfg, err := LoadMagicConfig(filepath.Join(magicHome, "magic.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Schema() != SchemaCurrent {
		t.Fatalf("schema = %v, want current", cfg.Schema())
	}
	if len(cfg.Adapters.Enabled) != 2 || cfg.Adapters.Enabled[0] != "pi" {
		t.Fatalf("adapters.enabled = %v, want the user's list", cfg.Adapters.Enabled)
	}
}

// A file without a version key still keeps its header comment on top: migrate
// inserts schema_version first, and the header must not move below it.
func TestMigrateKeepsHeaderCommentWhenVersionIsAbsent(t *testing.T) {
	magicHome := t.TempDir()
	writeMagicYAML(t, magicHome, "# 文件头注释\nadapters:\n  enabled: [pi]\n")

	if _, err := MigrateLibrary(MigrateOptions{MagicHome: magicHome}); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(magicHome, "magic.yaml"))
	if !strings.HasPrefix(got, "# 文件头注释\nschema_version: 1\n") {
		t.Fatalf("header comment moved:\n%s", got)
	}
}

// Before the write magic.yaml is copied into backups/<stamp>/, so the migration
// is as recoverable as an apply.
func TestMigrateBacksUpMagicYAMLBeforeWriting(t *testing.T) {
	magicHome := t.TempDir()
	writeMagicYAML(t, magicHome, legacyMagicYAML)

	res, err := MigrateLibrary(MigrateOptions{MagicHome: magicHome})
	if err != nil {
		t.Fatal(err)
	}
	if res.BackupDir == "" {
		t.Fatal("migrate wrote without a backup")
	}
	if _, err := time.Parse(BackupStampFormat, filepath.Base(res.BackupDir)); err != nil {
		t.Fatalf("backup dir %q is not a stamp: %v", res.BackupDir, err)
	}
	if got := readFile(t, filepath.Join(res.BackupDir, "magic.yaml")); got != legacyMagicYAML {
		t.Fatalf("backup content = %q, want the pre-migration file", got)
	}
	m, err := backup.Read(res.BackupDir)
	if err != nil || m == nil {
		t.Fatalf("manifest = %+v, err = %v; want a recorded set", m, err)
	}
	if len(m.Entries) != 1 || m.Entries[0].Rel != "magic.yaml" || m.Entries[0].Action != backup.ActionUpdate {
		t.Fatalf("manifest entries = %+v, want one magic.yaml update", m.Entries)
	}
}

// Running migrate twice is a no-op: no second backup, no rewrite, exit 0.
func TestMigrateIsIdempotent(t *testing.T) {
	magicHome := t.TempDir()
	writeMagicYAML(t, magicHome, legacyMagicYAML)
	if _, err := MigrateLibrary(MigrateOptions{MagicHome: magicHome}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(magicHome, "magic.yaml")
	first := readFile(t, path)

	res, err := MigrateLibrary(MigrateOptions{MagicHome: magicHome})
	if err != nil {
		t.Fatal(err)
	}
	if !res.AlreadyCurrent || res.SchemaVersion != LibrarySchemaVersion {
		t.Fatalf("res = %+v, want already current", res)
	}
	if len(res.Changes) != 0 || res.BackupDir != "" {
		t.Fatalf("res = %+v, want no changes and no backup", res)
	}
	if got := readFile(t, path); got != first {
		t.Fatalf("second migrate rewrote the file:\n%s", got)
	}
	entries, err := os.ReadDir(filepath.Join(magicHome, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("backups/ has %d sets, want only the first migration's", len(entries))
	}
}

// --dry-run lists the change and writes nothing: no backup, same bytes.
func TestMigrateDryRunWritesNothing(t *testing.T) {
	magicHome := t.TempDir()
	writeMagicYAML(t, magicHome, legacyMagicYAML)

	res, err := MigrateLibrary(MigrateOptions{MagicHome: magicHome, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun || res.BackupDir != "" || len(res.Changes) != 1 {
		t.Fatalf("res = %+v, want one change and no backup", res)
	}
	if got := readFile(t, filepath.Join(magicHome, "magic.yaml")); got != legacyMagicYAML {
		t.Fatalf("dry-run rewrote magic.yaml:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(magicHome, "backups")); !os.IsNotExist(err) {
		t.Fatalf("dry-run created backups/ (err = %v)", err)
	}
}

// A library written by a newer magic is refused: this one does not know its
// fields, so migrating would drop them.
func TestMigrateRefusesNewerLibrary(t *testing.T) {
	magicHome := t.TempDir()
	newer := "version: \"1.0\"\nschema_version: 99\nfuture_field: keep\n"
	writeMagicYAML(t, magicHome, newer)

	res, err := MigrateLibrary(MigrateOptions{MagicHome: magicHome})
	if err == nil {
		t.Fatal("migrate accepted a newer library")
	}
	if !strings.Contains(err.Error(), "比 magic 支持的 1 新") {
		t.Fatalf("error = %v, want the newer-library refusal", err)
	}
	if res.SchemaVersion != 99 {
		t.Fatalf("res.SchemaVersion = %d, want the version it found", res.SchemaVersion)
	}
	if got := readFile(t, filepath.Join(magicHome, "magic.yaml")); got != newer {
		t.Fatalf("refused migrate still rewrote magic.yaml:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(magicHome, "backups")); !os.IsNotExist(err) {
		t.Fatalf("refused migrate created backups/ (err = %v)", err)
	}
}

// An absent magic.yaml is not a schema-0 library: init creates the file, and
// migrate says so instead of inventing one.
func TestMigrateRequiresMagicYAML(t *testing.T) {
	_, err := MigrateLibrary(MigrateOptions{MagicHome: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "magic init") {
		t.Fatalf("error = %v, want the magic init instruction", err)
	}
}
