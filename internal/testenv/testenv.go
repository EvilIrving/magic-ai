// Package testenv confines a test binary to a throwaway home. Tests already use
// t.TempDir() for the home they pass in, but the harnesses' own environment
// overrides win over that argument: on a developer machine with
// CLAUDE_CONFIG_DIR or MINIMAX_DATA_DIR exported, a test would read and write
// the real harness configuration. Main redirects every home the code can resolve
// and clears the per-harness overrides before any test runs.
package testenv

import (
	"fmt"
	"os"
	"testing"
)

// platformHomeVars are the variables the Go runtime and the harnesses resolve a
// user home or platform config root from. The sandbox points them all at the
// same throwaway directory.
var platformHomeVars = []string{
	"HOME", "USERPROFILE",
	"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME",
	"APPDATA", "LOCALAPPDATA",
}

// harnessHomeVars are the per-harness roots internal/adapters and
// internal/provider read instead of the user home:
//
//	grep -rn "os.Getenv" internal/adapters internal/provider
//
// plus MAVIS_DATA_DIR, the second name in adapters/mmx's root lookup. An ambient
// value would point a test at the developer's real harness configuration, so the
// sandbox clears them.
var harnessHomeVars = []string{
	"CRAFT_CONFIG_DIR",    // adapters/craft
	"DIMCODE_HOME",        // adapters/dimagent
	"DSH_HOME",            // adapters/dsh
	"GOOSE_PATH_ROOT",     // adapters/goose
	"HERMES_HOME",         // adapters/hermes
	"KIMI_CODE_HOME",      // adapters/kimicode
	"MAVIS_DATA_DIR",      // adapters/mmx
	"MIMOCODE_HOME",       // adapters/mimocode
	"MINIMAX_DATA_DIR",    // adapters/mmx, provider/mmx
	"OPENCLAW_HOME",       // adapters/openclaw
	"OPENCLAW_STATE_DIR",  // adapters/openclaw
	"PI_CODING_AGENT_DIR", // adapters/omp
	"PI_CONFIG_DIR",       // adapters/omp
	"QODER_CONFIG_DIR",    // provider/qoder
	"QWEN_CODE_HOME",      // adapters/qwencode
	"VIBE_HOME",           // adapters/vibe
	// The harnesses' own CLIs read these. magic does not today, so clearing them
	// costs nothing and keeps an ambient value out of a test that shells out or
	// out of an adapter that starts honoring them.
	"CLAUDE_CONFIG_DIR",
	"CODEX_HOME",
}

// envValue remembers whether a variable was set at all, so restore can tell an
// empty value from an unset one.
type envValue struct {
	value string
	set   bool
}

// Main is a package's TestMain: it moves every home into a fresh temporary
// directory, clears the harness overrides, runs the tests, and puts the
// environment back. Use it as
//
//	func TestMain(m *testing.M) { os.Exit(testenv.Main(m)) }
func Main(m *testing.M) int {
	dir, err := os.MkdirTemp("", "magic-test-home-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "testenv: create sandbox home: %v\n", err)
		return 1
	}
	saved := saveEnv()
	sandbox(dir)
	code := m.Run()
	restoreEnv(saved)
	if err := os.RemoveAll(dir); err != nil {
		fmt.Fprintf(os.Stderr, "testenv: remove sandbox home: %v\n", err)
	}
	return code
}

// SetHome gives one test its own home inside the sandbox, replacing the one
// Main created. It cannot be used from a parallel test: t.Setenv panics there.
func SetHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range platformHomeVars {
		t.Setenv(name, dir)
	}
	for _, name := range harnessHomeVars {
		t.Setenv(name, "")
	}
	return dir
}

// sandbox points the platform homes at dir and clears the harness overrides.
func sandbox(dir string) {
	for _, name := range platformHomeVars {
		os.Setenv(name, dir)
	}
	for _, name := range harnessHomeVars {
		os.Unsetenv(name)
	}
}

// saveEnv records every variable sandbox touches.
func saveEnv() map[string]envValue {
	names := make([]string, 0, len(platformHomeVars)+len(harnessHomeVars))
	names = append(names, platformHomeVars...)
	names = append(names, harnessHomeVars...)
	saved := make(map[string]envValue, len(names))
	for _, name := range names {
		value, set := os.LookupEnv(name)
		saved[name] = envValue{value: value, set: set}
	}
	return saved
}

// restoreEnv puts every recorded variable back as it was.
func restoreEnv(saved map[string]envValue) {
	for name, v := range saved {
		if v.set {
			os.Setenv(name, v.value)
			continue
		}
		os.Unsetenv(name)
	}
}
