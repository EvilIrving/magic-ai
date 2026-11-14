package opencode

import (
	"path/filepath"
	"testing"
)

// The whole config tree moves with $XDG_CONFIG_HOME: mirror root, skills and
// agents are all children of the same root, so a split there would project into
// a directory OpenCode does not read.
func TestPathsFollowXDGConfigHome(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg-config")
	t.Setenv("XDG_CONFIG_HOME", xdg)

	a := New()
	root := filepath.Join(xdg, "opencode")
	cases := map[string]struct{ got, want string }{
		"MirrorRoot": {a.MirrorRoot(home), root},
		"SkillsDir":  {a.SkillsDir(home), filepath.Join(root, "skills")},
		"AgentsDir":  {a.AgentsDir(home), filepath.Join(root, "agents")},
	}
	for name, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", name, c.got, c.want)
		}
	}
}

func TestPathsIgnoreRelativeXDGConfigHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join("relative", "xdg-config"))

	if got, want := New().MirrorRoot(home), filepath.Join(home, ".config", "opencode"); got != want {
		t.Errorf("MirrorRoot = %q, want fallback %q", got, want)
	}
}

func TestPathsFallBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")

	if got, want := New().SkillsDir(home), filepath.Join(home, ".config", "opencode", "skills"); got != want {
		t.Errorf("SkillsDir = %q, want %q", got, want)
	}
}
