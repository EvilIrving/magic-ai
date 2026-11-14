package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readMagicYAML(t *testing.T, magicHome string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(magicHome, "magic.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// --dry-run lists the field change and leaves magic.yaml alone; the real run then
// writes schema_version: 1 with the user's own field and comment intact.
func TestMigrateCommandDryRunThenWrite(t *testing.T) {
	magicHome := t.TempDir()
	writeFileAt(t, filepath.Join(magicHome, "magic.yaml"), `# 我的库
version: "1.0"
adapters:
  enabled:
    - pi   # 生产用的那家
note: keep me
`)
	before := readMagicYAML(t, magicHome)

	out, err := runMagic(t, magicHome, cmdMigrate(), "--dry-run")
	if err != nil {
		t.Fatalf("migrate --dry-run = %v\n%s", err, out)
	}
	for _, want := range []string{"dry-run", "未写盘", "schema_version: 0 → 1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("dry-run output lacks %q:\n%s", want, out)
		}
	}
	if got := readMagicYAML(t, magicHome); got != before {
		t.Fatalf("dry-run rewrote magic.yaml:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(magicHome, "backups")); !os.IsNotExist(err) {
		t.Fatalf("dry-run created backups/ (err = %v)", err)
	}

	out, err = runMagic(t, magicHome, cmdMigrate())
	if err != nil {
		t.Fatalf("migrate = %v\n%s", err, out)
	}
	for _, want := range []string{"已迁移", "backup →", "schema_version: 0 → 1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("migrate output lacks %q:\n%s", want, out)
		}
	}
	got := readMagicYAML(t, magicHome)
	for _, want := range []string{"# 我的库", "# 生产用的那家", "note: keep me", "schema_version: 1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("migrated magic.yaml lost %q:\n%s", want, got)
		}
	}
}

// A second migrate says it is already current and writes nothing, not even a
// backup.
func TestMigrateCommandIsIdempotent(t *testing.T) {
	magicHome := t.TempDir()
	legacyMagicYAMLFixture(t, magicHome, "pi")
	if out, err := runMagic(t, magicHome, cmdMigrate()); err != nil {
		t.Fatalf("migrate = %v\n%s", err, out)
	}
	after := readMagicYAML(t, magicHome)

	out, err := runMagic(t, magicHome, cmdMigrate())
	if err != nil {
		t.Fatalf("second migrate = %v\n%s", err, out)
	}
	if !strings.Contains(out, "已是最新（schema 1）") {
		t.Fatalf("second migrate output = %q, want the already-current line", out)
	}
	if got := readMagicYAML(t, magicHome); got != after {
		t.Fatalf("second migrate rewrote magic.yaml:\n%s", got)
	}
	entries, err := os.ReadDir(filepath.Join(magicHome, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("backups/ has %d sets, want one", len(entries))
	}
}

// A newer library is refused with the upgrade instruction and stays untouched.
func TestMigrateCommandRefusesNewerLibrary(t *testing.T) {
	magicHome := t.TempDir()
	newer := "version: \"1.0\"\nschema_version: 99\n"
	writeFileAt(t, filepath.Join(magicHome, "magic.yaml"), newer)

	out, err := runMagic(t, magicHome, cmdMigrate())
	if err == nil {
		t.Fatalf("migrate accepted a schema-99 library:\n%s", out)
	}
	if !strings.Contains(err.Error(), "比 magic 支持的 1 新") {
		t.Fatalf("error = %v, want the newer-library refusal", err)
	}
	if got := readMagicYAML(t, magicHome); got != newer {
		t.Fatalf("refused migrate rewrote magic.yaml:\n%s", got)
	}
}
