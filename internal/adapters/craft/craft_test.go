package craft

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CRAFT_CONFIG_DIR", "")
	a := New()
	if a.Detect(home) {
		t.Fatal("Detect must be false before ~/.craft-agent exists")
	}
	if err := os.MkdirAll(a.root(home), 0o755); err != nil {
		t.Fatal(err)
	}
	if !a.Detect(home) {
		t.Fatal("Detect must be true once ~/.craft-agent exists")
	}
}

func TestDetectHonoursCraftConfigDir(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(t.TempDir(), "craft-instance-1")
	t.Setenv("CRAFT_CONFIG_DIR", root)
	a := New()
	if a.Detect(home) {
		t.Fatal("Detect must follow CRAFT_CONFIG_DIR, not the passed home")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if !a.Detect(home) {
		t.Fatal("Detect must be true for an existing CRAFT_CONFIG_DIR")
	}
	if got := a.MirrorRoot(home); got != root {
		t.Errorf("MirrorRoot = %q, want %q", got, root)
	}
}

func TestPathsMatchCaps(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CRAFT_CONFIG_DIR", "")
	a := New()
	caps := a.Caps()
	if caps.Skills || caps.Agents || caps.MCP {
		t.Fatalf("craft has no user-level projection, got %+v", caps)
	}
	if got := a.SkillsDir(home); got != "" {
		t.Errorf("SkillsDir = %q, want empty (skills are per workspace)", got)
	}
	if got := a.AgentsDir(home); got != "" {
		t.Errorf("AgentsDir = %q, want empty", got)
	}
	if got, want := a.MirrorRoot(home), filepath.Join(home, ".craft-agent"); got != want {
		t.Errorf("MirrorRoot = %q, want %q", got, want)
	}
	if got, want := a.LibraryMirrorRel(), filepath.Join("packs", "harness", "craft"); got != want {
		t.Errorf("LibraryMirrorRel = %q, want %q", got, want)
	}
}
