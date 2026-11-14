package testenv_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"magic/internal/testenv"
)

func TestMain(m *testing.M) { os.Exit(testenv.Main(m)) }

// The sandbox is what every dependent package relies on: the homes are a fresh
// temporary directory, and no harness override survives from the developer's
// shell. Running this test as
//
//	CLAUDE_CONFIG_DIR=/tmp/real-claude go test ./internal/testenv
//
// proves the override is gone before assertions run.
func TestMainSandboxesTheHome(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		t.Fatal("HOME is empty in the sandbox")
	}
	if !strings.HasPrefix(filepath.Clean(home), filepath.Clean(os.TempDir())) {
		t.Fatalf("HOME = %q, want a directory under %q", home, os.TempDir())
	}
	if st, err := os.Stat(home); err != nil || !st.IsDir() {
		t.Fatalf("HOME = %q is not a directory (err = %v)", home, err)
	}
	for _, name := range []string{"USERPROFILE", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "APPDATA", "LOCALAPPDATA"} {
		if got := os.Getenv(name); got != home {
			t.Fatalf("%s = %q, want the sandbox home %q", name, got, home)
		}
	}
	for _, name := range []string{
		"CLAUDE_CONFIG_DIR", "CODEX_HOME", "QODER_CONFIG_DIR", "MINIMAX_DATA_DIR",
		"CRAFT_CONFIG_DIR", "DSH_HOME", "KIMI_CODE_HOME", "HERMES_HOME",
	} {
		if got := os.Getenv(name); got != "" {
			t.Fatalf("%s = %q in the sandbox, want empty", name, got)
		}
	}
}

// A test that needs its own home gets one, not the one Main created.
func TestSetHomeGivesOneTestItsOwnHome(t *testing.T) {
	dir := testenv.SetHome(t)
	if got := os.Getenv("HOME"); got != dir {
		t.Fatalf("HOME = %q, want %q", got, dir)
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		t.Fatalf("SetHome dir %q is not a directory (err = %v)", dir, err)
	}
	if got := os.Getenv("MINIMAX_DATA_DIR"); got != "" {
		t.Fatalf("MINIMAX_DATA_DIR = %q after SetHome, want empty", got)
	}
}
