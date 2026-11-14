package vscode

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDetect(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	}
	a := New()
	if a.Detect(home) {
		t.Fatal("Detect must be false before the user profile exists")
	}
	if err := os.MkdirAll(a.userDir(home), 0o755); err != nil {
		t.Fatal(err)
	}
	if !a.Detect(home) {
		t.Fatal("Detect must be true once Code/User exists")
	}
}

func TestUserDirPerPlatform(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	if runtime.GOOS == "windows" {
		appData := filepath.Join(home, "AppData", "Roaming")
		t.Setenv("APPDATA", appData)
		if got, want := New().userDir(home), filepath.Join(appData, "Code", "User"); got != want {
			t.Errorf("user dir = %q, want %q", got, want)
		}
		return
	}
	if got := New().userDir(home); !filepath.IsAbs(got) {
		t.Errorf("user dir %q must be absolute", got)
	}
}

func TestPathsMatchCaps(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	a := New()
	caps := a.Caps()
	if caps.Skills || caps.Agents || caps.MCP {
		t.Fatalf("vscode reuses other harnesses' surfaces, got %+v", caps)
	}
	if got := a.SkillsDir(home); got != "" {
		t.Errorf("SkillsDir = %q, want empty", got)
	}
	if got := a.AgentsDir(home); got != "" {
		t.Errorf("AgentsDir = %q, want empty", got)
	}
	if got, want := a.MirrorRoot(home), a.userDir(home); got != want {
		t.Errorf("MirrorRoot = %q, want user profile %q", got, want)
	}
	if got, want := a.LibraryMirrorRel(), filepath.Join("packs", "harness", "vscode"); got != want {
		t.Errorf("LibraryMirrorRel = %q, want %q", got, want)
	}
}
