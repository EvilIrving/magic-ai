package goose

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDetect(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOOSE_PATH_ROOT", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	a := New()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	}
	if a.Detect(home) {
		t.Fatal("Detect must be false before the config dir exists")
	}
	if err := os.MkdirAll(a.root(home), 0o755); err != nil {
		t.Fatal(err)
	}
	if !a.Detect(home) {
		t.Fatal("Detect must be true once the config dir exists")
	}
}

func TestDetectHonoursGoosePathRoot(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	t.Setenv("GOOSE_PATH_ROOT", root)
	a := New()
	if a.Detect(home) {
		t.Fatal("Detect must follow GOOSE_PATH_ROOT, not the passed home")
	}
	// goose keeps configuration under <root>/config.
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !a.Detect(home) {
		t.Fatal("Detect must be true for an existing GOOSE_PATH_ROOT/config")
	}
	if got, want := a.MirrorRoot(home), filepath.Join(root, "config"); got != want {
		t.Errorf("MirrorRoot = %q, want %q", got, want)
	}
}

func TestPathsMatchCaps(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOOSE_PATH_ROOT", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	a := New()
	caps := a.Caps()
	if !caps.Skills || a.SkillsDir(home) == "" {
		t.Errorf("goose projects skills, got Skills=%v dir=%q", caps.Skills, a.SkillsDir(home))
	}
	if !caps.Agents {
		t.Fatal("goose declares sub-agent projection")
	}
	if got, want := a.AgentsDir(home), filepath.Join(home, ".agents", "agents"); got != want {
		t.Errorf("AgentsDir = %q, want %q", got, want)
	}
	if got, want := a.MirrorRoot(home), a.root(home); got != want {
		t.Errorf("MirrorRoot = %q, want config root %q", got, want)
	}
	if got, want := a.LibraryMirrorRel(), filepath.Join("packs", "harness", "goose"); got != want {
		t.Errorf("LibraryMirrorRel = %q, want %q", got, want)
	}
}
