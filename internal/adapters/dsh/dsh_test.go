package dsh

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {
	home := t.TempDir()
	t.Setenv("DSH_HOME", "")
	a := New()
	if a.Detect(home) {
		t.Fatal("Detect must be false before ~/.dsh exists")
	}
	if err := os.MkdirAll(a.root(home), 0o755); err != nil {
		t.Fatal(err)
	}
	if !a.Detect(home) {
		t.Fatal("Detect must be true once ~/.dsh exists")
	}
}

func TestDetectHonoursDSHHome(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(t.TempDir(), "dsh-home")
	t.Setenv("DSH_HOME", root)
	a := New()
	if a.Detect(home) {
		t.Fatal("Detect must follow DSH_HOME, not the passed home")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if !a.Detect(home) {
		t.Fatal("Detect must be true for an existing DSH_HOME")
	}
	if got, want := a.MirrorRoot(home), root; got != want {
		t.Errorf("MirrorRoot = %q, want %q", got, want)
	}
}

func TestPathsMatchCaps(t *testing.T) {
	home := t.TempDir()
	t.Setenv("DSH_HOME", "")
	a := New()
	caps := a.Caps()
	if got, want := a.SkillsDir(home), filepath.Join(home, ".dsh", "skills"); caps.Skills && got != want {
		t.Errorf("SkillsDir = %q, want %q", got, want)
	}
	if !caps.Skills && a.SkillsDir(home) != "" {
		t.Error("SkillsDir must be empty when Caps.Skills is false")
	}
	if !caps.Agents && a.AgentsDir(home) != "" {
		t.Error("AgentsDir must be empty when Caps.Agents is false")
	}
	if got, want := a.MirrorRoot(home), filepath.Join(home, ".dsh"); got != want {
		t.Errorf("MirrorRoot = %q, want %q", got, want)
	}
	if got, want := a.LibraryMirrorRel(), filepath.Join("packs", "harness", "dsh"); got != want {
		t.Errorf("LibraryMirrorRel = %q, want %q", got, want)
	}
}
