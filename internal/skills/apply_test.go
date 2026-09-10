package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureLinkCreateAndRepair(t *testing.T) {
	tmp := t.TempDir()
	central := filepath.Join(tmp, "central")
	skill := filepath.Join(central, "myskill")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("# s"), 0o644); err != nil {
		t.Fatal(err)
	}
	harness := filepath.Join(tmp, "harness", "skills")
	if err := os.MkdirAll(harness, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(harness, "myskill")

	action, err := EnsureLink(link, skill, false)
	if err != nil || (action != "create-symlink" && action != "create-copy" && action != "create-junction") {
		t.Fatalf("create: %s %v", action, err)
	}
	action, err = EnsureLink(link, skill, false)
	if err != nil || action != "ok" {
		t.Fatalf("idempotent: %s %v", action, err)
	}

	// wrong target — unmanaged symlink is kept
	other := filepath.Join(tmp, "other")
	_ = os.MkdirAll(other, 0o755)
	_ = os.Remove(link)
	_ = os.Symlink(other, link)
	action, err = EnsureLink(link, skill, false)
	if err != nil || action != "skip-unmanaged" {
		t.Fatalf("unmanaged: %s %v", action, err)
	}
	got, _ := os.Readlink(link)
	absOther, _ := filepath.Abs(other)
	absGot, _ := filepath.Abs(got)
	if absGot != absOther {
		t.Fatalf("unmanaged link rewritten: %s", absGot)
	}
	action, err = EnsureLinkForce(link, skill, false, true)
	if err != nil || (action != "repair-symlink" && action != "repair-copy" && action != "repair-junction" && action != "repair-link") {
		t.Fatalf("repair: %s %v", action, err)
	}
	// verify points at skill (symlink) or content exists (copy)
	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		got, _ := os.Readlink(link)
		absSkill, _ := filepath.Abs(skill)
		absGot, _ := filepath.Abs(got)
		if absGot != absSkill {
			t.Fatalf("link=%s want %s", absGot, absSkill)
		}
	} else if _, err := os.Stat(filepath.Join(link, "SKILL.md")); err != nil {
		t.Fatal("copy missing SKILL.md")
	}
}

func TestNeverDeleteRealDir(t *testing.T) {
	tmp := t.TempDir()
	central := filepath.Join(tmp, "central", "s")
	_ = os.MkdirAll(central, 0o755)
	real := filepath.Join(tmp, "harness", "s")
	_ = os.MkdirAll(real, 0o755)
	_ = os.WriteFile(filepath.Join(real, "keep.txt"), []byte("x"), 0o644)

	action, err := EnsureLink(real, central, false)
	if action != "skip-real" || err == nil {
		t.Fatalf("expected skip-real error, got %s %v", action, err)
	}
	if _, err := os.Stat(filepath.Join(real, "keep.txt")); err != nil {
		t.Fatal("real dir was damaged")
	}
}

func TestApplyProjects(t *testing.T) {
	tmp := t.TempDir()
	central := filepath.Join(tmp, "packs", "skills")
	_ = os.MkdirAll(filepath.Join(central, "a"), 0o755)
	h1 := filepath.Join(tmp, "pi", "skills")
	logs, err := ApplyProjects(central, []string{h1}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("logs=%v", logs)
	}
	if _, err := os.Lstat(filepath.Join(h1, "a")); err != nil {
		t.Fatal(err)
	}
}

func TestCentralDirIsMagicPacks(t *testing.T) {
	tmp := t.TempDir()
	magic := filepath.Join(tmp, ".magic-ai")
	_ = os.MkdirAll(filepath.Join(magic, "packs", "skills"), 0o755)
	got := CentralDir(magic)
	want := filepath.Join(magic, "packs", "skills")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestApplySkipsCentralAsHarness(t *testing.T) {
	tmp := t.TempDir()
	central := filepath.Join(tmp, "magic-home", "packs", "skills")
	_ = os.MkdirAll(filepath.Join(central, "x"), 0o755)
	logs, err := ApplyProjects(central, []string{central, filepath.Join(tmp, "pi", "skills")}, false)
	if err != nil {
		t.Fatal(err)
	}
	// only one harness dir should get a link
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %v", logs)
	}
}
