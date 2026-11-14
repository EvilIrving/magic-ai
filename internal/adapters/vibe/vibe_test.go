package vibe

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {
	home := t.TempDir()
	t.Setenv("VIBE_HOME", "")
	a := New()
	if a.Detect(home) {
		t.Fatal("Detect must be false before ~/.vibe exists")
	}
	if err := os.MkdirAll(a.root(home), 0o755); err != nil {
		t.Fatal(err)
	}
	if !a.Detect(home) {
		t.Fatal("Detect must be true once ~/.vibe exists")
	}
}

func TestDetectHonoursVibeHome(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(t.TempDir(), "vibe-home")
	t.Setenv("VIBE_HOME", root)
	a := New()
	if a.Detect(home) {
		t.Fatal("Detect must follow VIBE_HOME, not the passed home")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if !a.Detect(home) {
		t.Fatal("Detect must be true for an existing VIBE_HOME")
	}
	// A tilde-prefixed VIBE_HOME expands against the passed home, as upstream does.
	t.Setenv("VIBE_HOME", "~/vibe-tilde")
	if got, want := a.root(home), filepath.Join(home, "vibe-tilde"); got != want {
		t.Errorf("tilde VIBE_HOME root = %q, want %q", got, want)
	}
}

func TestPathsMatchCaps(t *testing.T) {
	home := t.TempDir()
	t.Setenv("VIBE_HOME", "")
	a := New()
	caps := a.Caps()
	if got, want := a.SkillsDir(home), filepath.Join(home, ".vibe", "skills"); caps.Skills && got != want {
		t.Errorf("SkillsDir = %q, want %q", got, want)
	}
	if !caps.Skills && a.SkillsDir(home) != "" {
		t.Error("SkillsDir must be empty when Caps.Skills is false")
	}
	if !caps.Agents && a.AgentsDir(home) != "" {
		t.Error("AgentsDir must be empty when Caps.Agents is false (agent profiles are TOML)")
	}
	if got, want := a.MirrorRoot(home), filepath.Join(home, ".vibe"); got != want {
		t.Errorf("MirrorRoot = %q, want %q", got, want)
	}
	if got, want := a.LibraryMirrorRel(), filepath.Join("packs", "harness", "vibe"); got != want {
		t.Errorf("LibraryMirrorRel = %q, want %q", got, want)
	}
}
