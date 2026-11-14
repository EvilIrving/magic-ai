package mimocode

import (
	"path/filepath"
	"testing"
)

// MiMo Code is an OpenCode fork and resolves its config the same way
// (packages/shared/src/global.ts: config = path.join(xdgConfig, "mimocode")).
func TestPathsFollowXDGConfigHome(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg-config")
	t.Setenv("MIMOCODE_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", xdg)

	a := New()
	root := filepath.Join(xdg, "mimocode")
	if got := a.MirrorRoot(home); got != root {
		t.Errorf("MirrorRoot = %q, want %q", got, root)
	}
	if got, want := a.SkillsDir(home), filepath.Join(root, "skills"); got != want {
		t.Errorf("SkillsDir = %q, want %q", got, want)
	}
}

func TestPathsIgnoreRelativeXDGConfigHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("MIMOCODE_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join("relative", "xdg-config"))

	if got, want := New().MirrorRoot(home), filepath.Join(home, ".config", "mimocode"); got != want {
		t.Errorf("MirrorRoot = %q, want fallback %q", got, want)
	}
}

func TestPathsFallBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("MIMOCODE_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	if got, want := New().MirrorRoot(home), filepath.Join(home, ".config", "mimocode"); got != want {
		t.Errorf("MirrorRoot = %q, want %q", got, want)
	}
}

// MIMOCODE_HOME outranks XDG: it names the whole mimocode home, of which config
// is one subdirectory.
func TestMimocodeHomeOutranksXDG(t *testing.T) {
	home := t.TempDir()
	mimoHome := t.TempDir()
	t.Setenv("MIMOCODE_HOME", mimoHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg-config"))

	if got, want := New().MirrorRoot(home), filepath.Join(mimoHome, "config"); got != want {
		t.Errorf("MirrorRoot = %q, want %q", got, want)
	}
}
