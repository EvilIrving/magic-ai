package kilo

import (
	"path/filepath"
	"testing"
)

// Kilo has no native hook events (docs/hooks-design.md), so the adapter must not
// declare them: `magic adapters` reads Caps, and a true here advertises a
// projection Magic never writes.
func TestCapsDoNotClaimHooks(t *testing.T) {
	c := New().Caps()
	if c.Hooks {
		t.Fatal("Caps.Hooks is true, but Kilo has no native hook events")
	}
	if !c.Skills || !c.MCP {
		t.Fatalf("skills and MCP must stay true: %+v", c)
	}
}

// The config root follows $XDG_CONFIG_HOME (Kilo's own XDG move), but skills do
// not: Kilo reads them from ~/.kilo/skills, outside the config root.
func TestPathsFollowXDGConfigHome(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg-config")
	t.Setenv("XDG_CONFIG_HOME", xdg)

	a := New()
	if got, want := a.MirrorRoot(home), filepath.Join(xdg, "kilo"); got != want {
		t.Errorf("MirrorRoot = %q, want %q", got, want)
	}
	if got, want := a.SkillsDir(home), filepath.Join(home, ".kilo", "skills"); got != want {
		t.Errorf("SkillsDir = %q, want %q (not under the config root)", got, want)
	}
}

func TestPathsIgnoreRelativeXDGConfigHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join("relative", "xdg-config"))

	if got, want := New().MirrorRoot(home), filepath.Join(home, ".config", "kilo"); got != want {
		t.Errorf("MirrorRoot = %q, want fallback %q", got, want)
	}
}

func TestPathsFallBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")

	if got, want := New().MirrorRoot(home), filepath.Join(home, ".config", "kilo"); got != want {
		t.Errorf("MirrorRoot = %q, want %q", got, want)
	}
}
