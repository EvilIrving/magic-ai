package zed

import (
	"os"
	"path/filepath"
	"testing"
)

// MirrorRoot is where Magic writes settings.json, so it is the path that must
// follow $XDG_CONFIG_HOME. Skills are shared with other harnesses in
// ~/.agents/skills and must not move with it.
func TestPathsFollowXDGConfigHome(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg-config")
	t.Setenv("XDG_CONFIG_HOME", xdg)

	a := New()
	if got, want := a.MirrorRoot(home), filepath.Join(xdg, "zed"); got != want {
		t.Errorf("MirrorRoot = %q, want %q", got, want)
	}
	if got, want := a.SkillsDir(home), filepath.Join(home, ".agents", "skills"); got != want {
		t.Errorf("SkillsDir = %q, want %q (not under the config root)", got, want)
	}

	if err := os.MkdirAll(a.MirrorRoot(home), 0o755); err != nil {
		t.Fatal(err)
	}
	if !a.Detect(home) {
		t.Error("Detect must look at the XDG config root")
	}
}

func TestPathsIgnoreRelativeXDGConfigHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join("relative", "xdg-config"))

	if got, want := New().MirrorRoot(home), filepath.Join(home, ".config", "zed"); got != want {
		t.Errorf("MirrorRoot = %q, want fallback %q", got, want)
	}
}

func TestPathsFallBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")

	if got, want := New().MirrorRoot(home), filepath.Join(home, ".config", "zed"); got != want {
		t.Errorf("MirrorRoot = %q, want %q", got, want)
	}
}
