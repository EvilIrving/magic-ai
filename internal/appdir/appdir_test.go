package appdir

import (
	"os"
	"path/filepath"
	"testing"
)

// A set, absolute $XDG_CONFIG_HOME wins even though the directory does not exist
// yet: the spec only demands an absolute path, and a harness that has never run
// has no such directory (see the package comment).
func TestConfigHomeHonoursAbsoluteXDG(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg-config")
	t.Setenv(configHomeEnv, xdg)

	if _, err := os.Stat(xdg); !os.IsNotExist(err) {
		t.Fatalf("test precondition: %s must not exist, got err=%v", xdg, err)
	}
	if got := ConfigHome(home); got != xdg {
		t.Errorf("ConfigHome = %q, want %q", got, xdg)
	}
}

// The spec says a relative value is invalid and must be ignored.
func TestConfigHomeIgnoresRelativeXDG(t *testing.T) {
	home := t.TempDir()
	t.Setenv(configHomeEnv, filepath.Join("relative", "xdg-config"))

	want := filepath.Join(home, ".config")
	if got := ConfigHome(home); got != want {
		t.Errorf("ConfigHome = %q, want fallback %q", got, want)
	}
}

func TestConfigHomeFallsBackWhenEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv(configHomeEnv, "")

	want := filepath.Join(home, ".config")
	if got := ConfigHome(home); got != want {
		t.Errorf("ConfigHome = %q, want fallback %q", got, want)
	}
}

func TestConfigHomeFallsBackWhenUnset(t *testing.T) {
	home := t.TempDir()
	unsetForTest(t, configHomeEnv)

	want := filepath.Join(home, ".config")
	if got := ConfigHome(home); got != want {
		t.Errorf("ConfigHome = %q, want fallback %q", got, want)
	}
}

// unsetForTest removes name for the duration of the test, restoring the previous
// value afterwards (t.Setenv cannot express "unset" and must not be mixed with
// os.Unsetenv on the same variable).
func unsetForTest(t *testing.T, name string) {
	t.Helper()
	saved, set := os.LookupEnv(name)
	if err := os.Unsetenv(name); err != nil {
		t.Fatalf("unset %s: %v", name, err)
	}
	t.Cleanup(func() {
		if set {
			os.Setenv(name, saved)
			return
		}
		os.Unsetenv(name)
	})
}

func TestDataHome(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg-data")

	t.Setenv(dataHomeEnv, xdg)
	if got := DataHome(home); got != xdg {
		t.Errorf("absolute XDG_DATA_HOME: got %q, want %q", got, xdg)
	}

	t.Setenv(dataHomeEnv, filepath.Join("relative", "xdg-data"))
	want := filepath.Join(home, ".local", "share")
	if got := DataHome(home); got != want {
		t.Errorf("relative XDG_DATA_HOME: got %q, want fallback %q", got, want)
	}

	t.Setenv(dataHomeEnv, "")
	if got := DataHome(home); got != want {
		t.Errorf("empty XDG_DATA_HOME: got %q, want fallback %q", got, want)
	}
}

func TestExpandConfig(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg-config")
	t.Setenv(configHomeEnv, xdg)

	if got, want := ExpandConfig(home, ".config/opencode/skills"), filepath.Join(xdg, "opencode", "skills"); got != want {
		t.Errorf("ExpandConfig(.config/...) = %q, want %q", got, want)
	}
	// Only .config/ entries move; the rest stay $HOME-relative for the caller.
	if got, want := ExpandConfig(home, ".claude/skills"), ".claude/skills"; got != want {
		t.Errorf("ExpandConfig(other) = %q, want unchanged %q", got, want)
	}

	t.Setenv(configHomeEnv, "")
	if got, want := ExpandConfig(home, ".config/goose/skills"), filepath.Join(home, ".config", "goose", "skills"); got != want {
		t.Errorf("ExpandConfig without XDG = %q, want %q", got, want)
	}
}
