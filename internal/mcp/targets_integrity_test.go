package mcp

import (
	"strings"
	"testing"
)

// Target table integrity: every row must be a complete contract
// (Style + Format + TopKey + a Path inside the user home).
// A wrong row writes an invalid config into a user's installed tool.
func TestTargetsIntegrity(t *testing.T) {
	seen := map[string]bool{}
	for _, tgt := range Targets() {
		if tgt.Harness == "" {
			t.Error("target with empty Harness")
		}
		if seen[tgt.Harness] {
			t.Errorf("duplicate target for %q", tgt.Harness)
		}
		seen[tgt.Harness] = true
		if tgt.Style == "" {
			t.Errorf("%s: empty Style", tgt.Harness)
		}
		if tgt.Format != FormatJSON && tgt.Format != FormatTOML && tgt.Format != FormatYAML {
			t.Errorf("%s: unknown Format %q", tgt.Harness, tgt.Format)
		}
		if tgt.TopKey == "" {
			t.Errorf("%s: empty TopKey", tgt.Harness)
		}
		if tgt.Path == nil {
			t.Errorf("%s: nil Path", tgt.Harness)
			continue
		}
		p := tgt.Path("/fake/home")
		if !strings.HasPrefix(p, "/fake/home") || p == "/fake/home" {
			t.Errorf("%s: path escapes home or is home itself: %q", tgt.Harness, p)
		}
		// TargetFor must round-trip every registered harness.
		if _, ok := TargetFor(tgt.Harness); !ok {
			t.Errorf("%s: TargetFor cannot find registered harness", tgt.Harness)
		}
	}
}

func TestTargetForUnknown(t *testing.T) {
	if _, ok := TargetFor("no-such-harness"); ok {
		t.Fatal("unknown harness should not resolve")
	}
	if _, ok := TargetFor(""); ok {
		t.Fatal("empty harness should not resolve")
	}
}

// An empty Server carries neither a command nor a URL. Translate is
// currently lenient: stdio styles still emit an entry (with an empty
// command) instead of failing. Pin that behavior here so any future
// hardening (reject empty entries) shows up as a test change, not a
// silent projection of an invalid entry into a user's tool config.
func TestTranslateEmptyServerLenient(t *testing.T) {
	e, ok, _ := Translate(Server{}, StyleClaude)
	if !ok {
		t.Fatal("claude currently accepts empty server")
	}
	if cmd, present := e["command"]; present && cmd != "" {
		t.Fatalf("expected absent/empty command passthrough, got %v", e)
	}
	if (Server{}).IsRemote() {
		t.Fatal("empty server must not count as remote")
	}
}

func TestTranslateUnknownStyle(t *testing.T) {
	if _, ok, reason := Translate(Server{Command: "x"}, "bogus-style"); ok || reason == "" {
		t.Fatalf("unknown style should be unsupported: ok=%v reason=%q", ok, reason)
	}
}

// D8/D18: MCP secrets are written as plaintext values, never redacted
// or turned into ${ENV} references at translate time.
func TestTranslateKeepsSecretEnvPlaintext(t *testing.T) {
	s := Server{Command: "npx", Env: map[string]string{"GH_TOKEN": "ghp-live-secret"}}
	// OpenCode names the env map "environment" (vendor dialect), the rest "env".
	envKeys := map[string]string{
		StyleClaude: "env", StyleCodex: "env", StylePi: "env",
		StyleOpenCode: "environment", StyleGeneric: "env",
	}
	for style, key := range envKeys {
		e, ok, _ := Translate(s, style)
		if !ok {
			continue // style may not support stdio; covered elsewhere
		}
		env, _ := e[key].(map[string]any)
		if env == nil {
			t.Fatalf("%s: %q dropped: %v", style, key, e)
		}
		if env["GH_TOKEN"] != "ghp-live-secret" {
			t.Errorf("%s: secret must stay plaintext, got %v", style, env["GH_TOKEN"])
		}
	}
}
