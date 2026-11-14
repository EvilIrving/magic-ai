package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An empty library has nothing to project: verify says so and names the two ways
// to put content into the library, instead of blaming the adapters.
func TestVerifyExplainsEmptyLibrary(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()

	out, err := runVerifyOutput(t, magicHome, userHome, verifyArgs(userHome, "", false)...)
	if err == nil {
		t.Fatalf("verify passed an empty library:\n%s", out)
	}
	for _, want := range []string{"没有检查任何文件", "配置库里没有可投影的内容", "magic init", "magic add"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("verify error = %v, want %q", err, want)
		}
	}
}

// A provider whose endpoint protocol the software cannot take is a provider
// problem: verify names the provider and points at doctor.
func TestVerifyExplainsProviderProtocolMismatch(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	doctorFixtureMagicYAML(t, magicHome, "claude")
	doctorFixtureProviders(t, magicHome, openaiChatOnlyEndpoint)
	if err := os.MkdirAll(filepath.Join(userHome, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runVerifyOutput(t, magicHome, userHome, verifyArgs(userHome, "", false)...)
	if err == nil {
		t.Fatalf("verify passed with a provider Claude cannot take:\n%s", out)
	}
	for _, want := range []string{"provider example 没有 Claude Code 支持的协议", "magic doctor"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("verify error = %v, want %q", err, want)
		}
	}
}

// Adapters left out of magic.yaml's enabled list are not checked: verify counts
// them and points at status, not at the adapter configuration.
func TestVerifyExplainsDisabledAdapters(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	doctorFixtureMagicYAML(t, magicHome, "claude")
	writeFileAt(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "# Pi\n")

	out, err := runVerifyOutput(t, magicHome, userHome, verifyArgs(userHome, "", false)...)
	if err == nil {
		t.Fatalf("verify passed while the library's content was disabled:\n%s", out)
	}
	for _, want := range []string{"个 adapter 未启用", "magic status"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("verify error = %v, want %q", err, want)
		}
	}
}

// --only names one adapter: when magic.yaml leaves that adapter disabled, verify
// says which list left it out and points at status, instead of calling every
// adapter skipped.
func TestVerifyExplainsOnlyAdapterDisabled(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	doctorFixtureMagicYAML(t, magicHome, "pi")
	writeFileAt(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "# Pi\n")

	out, err := runVerifyOutput(t, magicHome, userHome, verifyArgs(userHome, "claude", false)...)
	if err == nil {
		t.Fatalf("verify passed with --only naming a disabled adapter:\n%s", out)
	}
	for _, want := range []string{"claude 未启用（magic.yaml 的 adapters.enabled）", "magic status"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("verify error = %v, want %q", err, want)
		}
	}
}

// --only names one adapter that is enabled but has no mirror, no provider and no
// MCP server to project into it: the library has nothing for that adapter, and
// add path is how content gets there.
func TestVerifyExplainsOnlyAdapterWithoutContent(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	doctorFixtureMagicYAML(t, magicHome, "claude")
	writeFileAt(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "# Pi\n")

	out, err := runVerifyOutput(t, magicHome, userHome, verifyArgs(userHome, "claude", false)...)
	if err == nil {
		t.Fatalf("verify passed with --only naming an adapter with no content:\n%s", out)
	}
	for _, want := range []string{"没有可以投影给 claude 的内容", "magic add path"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("verify error = %v, want %q", err, want)
		}
	}
}
