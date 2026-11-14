package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"magic/internal/catalog"
)

func TestTargetAliases(t *testing.T) {
	for _, s := range []string{"none", "off", "clear", "-"} {
		if !IsClearAlias(s) {
			t.Errorf("%q should be a clear alias", s)
		}
	}
	for _, s := range []string{"official", "native"} {
		if !IsOfficialAlias(s) {
			t.Errorf("%q should be an official alias", s)
		}
	}
	f := catalogFileForTest(t)
	rt, err := ResolveUseTarget(f, "none")
	if err != nil || rt.Mode != ModeClear {
		t.Fatalf("none -> mode=%q err=%v", rt.Mode, err)
	}
	rt, err = ResolveUseTarget(f, "official")
	if err != nil || rt.Mode != ModeOfficial {
		t.Fatalf("official -> mode=%q err=%v", rt.Mode, err)
	}
}

// catalogFileForTest returns an empty (but non-nil) catalog.File.
func catalogFileForTest(t *testing.T) *catalog.File {
	t.Helper()
	f, err := LoadCatalog(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestSupportsOfficialFlags(t *testing.T) {
	// Multi-provider harnesses, and harnesses whose first-party path is an
	// interactive account login (Kimi Code /login, Vibe's Mistral account,
	// Hermes `hermes auth`) rather than a file Magic may write.
	unsupported := map[string]bool{
		"pi": true, "opencode": true, "kimi-code": true, "vibe": true,
		"hermes": true, "omp": true, "qoder": true, "mmx": true,
	}
	for _, w := range Writers() {
		if !w.Implemented() {
			continue
		}
		want := !unsupported[w.Harness()]
		if w.SupportsOfficial() != want {
			t.Errorf("%s.SupportsOfficial()=%v want %v", w.Harness(), w.SupportsOfficial(), want)
		}
	}
}

func TestUseOfficialRejectedForMultiProvider(t *testing.T) {
	home := t.TempDir()
	magic := t.TempDir()
	ctx := Context{MagicHome: magic, UserHome: home}
	if _, err := Use(ctx, "pi", "official"); err == nil || !strings.Contains(err.Error(), "no official provider") {
		t.Fatalf("pi official should error, got: %v", err)
	}
	if _, err := Use(ctx, "opencode", "official"); err == nil {
		t.Fatal("opencode official should error")
	}
}

func TestUseNoneClearsPi(t *testing.T) {
	home := t.TempDir()
	magic := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".pi", "agent"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-seed a Magic provider + a user provider.
	seed := `{"providers":{"magic-shu":{"baseUrl":"https://x","api":"openai-completions","apiKey":"sk","models":[]},"keep":{"baseUrl":"https://k","api":"openai-completions","apiKey":"$K","models":[]}}}`
	if err := os.WriteFile(filepath.Join(home, ".pi", "agent", "models.json"), []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := Context{MagicHome: magic, UserHome: home}
	if _, err := Use(ctx, "pi", "none"); err != nil {
		t.Fatalf("pi none: %v", err)
	}
	raw, _ := os.ReadFile(filepath.Join(home, ".pi", "agent", "models.json"))
	if strings.Contains(string(raw), "magic-shu") {
		t.Fatalf("magic-* not cleared:\n%s", raw)
	}
	if !strings.Contains(string(raw), "keep") {
		t.Fatalf("user provider dropped:\n%s", raw)
	}
}
