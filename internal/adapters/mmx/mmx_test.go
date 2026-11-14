package mmx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {
	home := t.TempDir()
	t.Setenv("MINIMAX_DATA_DIR", "")
	t.Setenv("MAVIS_DATA_DIR", "")
	a := New()
	if a.Detect(home) {
		t.Fatal("Detect must be false before ~/.minimax exists")
	}
	if err := os.MkdirAll(a.root(home), 0o755); err != nil {
		t.Fatal(err)
	}
	if !a.Detect(home) {
		t.Fatal("Detect must be true once ~/.minimax exists")
	}
}

func TestDetectHonoursDataDir(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(t.TempDir(), "minimax-data")
	t.Setenv("MINIMAX_DATA_DIR", root)
	t.Setenv("MAVIS_DATA_DIR", "")
	a := New()
	if a.Detect(home) {
		t.Fatal("Detect must follow MINIMAX_DATA_DIR, not the passed home")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if !a.Detect(home) {
		t.Fatal("Detect must be true for an existing MINIMAX_DATA_DIR")
	}
	// MAVIS_DATA_DIR is the legacy alias, used only when the new name is unset.
	t.Setenv("MINIMAX_DATA_DIR", "")
	t.Setenv("MAVIS_DATA_DIR", root)
	if got := a.MirrorRoot(home); got != root {
		t.Errorf("MirrorRoot = %q, want legacy MAVIS_DATA_DIR %q", got, root)
	}
}

func TestPathsMatchCaps(t *testing.T) {
	home := t.TempDir()
	t.Setenv("MINIMAX_DATA_DIR", "")
	t.Setenv("MAVIS_DATA_DIR", "")
	a := New()
	caps := a.Caps()
	if got, want := a.SkillsDir(home), filepath.Join(home, ".minimax", "skills"); caps.Skills && got != want {
		t.Errorf("SkillsDir = %q, want %q", got, want)
	}
	if !caps.Skills && a.SkillsDir(home) != "" {
		t.Error("SkillsDir must be empty when Caps.Skills is false")
	}
	if !caps.Agents && a.AgentsDir(home) != "" {
		t.Error("AgentsDir must be empty when Caps.Agents is false (agent dirs are agent.md folders)")
	}
	if got, want := a.MirrorRoot(home), filepath.Join(home, ".minimax"); got != want {
		t.Errorf("MirrorRoot = %q, want %q", got, want)
	}
	if got, want := a.LibraryMirrorRel(), filepath.Join("packs", "harness", "mmx"); got != want {
		t.Errorf("LibraryMirrorRel = %q, want %q", got, want)
	}
}
