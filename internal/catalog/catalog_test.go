package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndCopyPlaceholder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.yaml")
	_ = os.WriteFile(path, []byte(ExampleProvidersYAML), 0o644)
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := f.Providers["deepseek"]
	if !ok {
		t.Fatal("missing deepseek")
	}

	out := CopyText("deepseek", p, "", false)
	if !strings.Contains(out, "base_url:") || !strings.Contains(out, "api_key:") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "key_env: DEEPSEEK_API_KEY") {
		t.Fatalf("missing key_env field:\n%s", out)
	}
	if !strings.Contains(out, "api_key: <DEEPSEEK_API_KEY>") {
		t.Fatalf("expected placeholder:\n%s", out)
	}
	if !strings.Contains(out, "secret missing") {
		t.Fatalf("expected missing-secret note:\n%s", out)
	}
	if strings.Contains(out, "sk-") {
		t.Fatal("must not invent real keys")
	}
}

func TestCopyTextRedactAndReveal(t *testing.T) {
	p := Provider{
		BaseURL:  "https://api.deepseek.com/v1",
		Protocol: "openai",
		KeyEnv:   "DEEPSEEK_API_KEY",
		Pick:     []string{"deepseek-chat"},
	}
	const full = "sk-TEST-DEEPSEEK-KEY99"

	out := CopyText("deepseek", p, full, false)
	if strings.Contains(out, full) {
		t.Fatalf("full key leaked:\n%s", out)
	}
	if !strings.Contains(out, "sk-T") || !strings.Contains(out, "…") {
		t.Fatalf("expected redacted key:\n%s", out)
	}
	if !strings.Contains(out, "key_env: DEEPSEEK_API_KEY") {
		t.Fatalf("missing key_env:\n%s", out)
	}
	if !strings.Contains(out, "models:\n  - deepseek-chat\n") {
		t.Fatalf("models:\n%s", out)
	}
	if strings.Contains(out, "secret missing") {
		t.Fatalf("should not note missing when key present:\n%s", out)
	}

	revealed := CopyText("deepseek", p, full, true)
	if !strings.Contains(revealed, full) {
		t.Fatalf("reveal-key missing full key:\n%s", revealed)
	}
}

func TestCopyTextIgnoresProcessEnv(t *testing.T) {
	p := Provider{
		BaseURL:  "https://api.deepseek.com/v1",
		Protocol: "openai",
		KeyEnv:   "DEEPSEEK_API_KEY",
	}
	const envLeak = "sk-FROM-ENV-SHOULD-NOT-APPEAR"
	t.Setenv("DEEPSEEK_API_KEY", envLeak)

	// Empty resolved key → placeholder even if env is set (copy never reads env).
	out := CopyText("deepseek", p, "", false)
	if strings.Contains(out, envLeak) || strings.Contains(out, "sk-FROM") {
		t.Fatalf("env must not leak into copy:\n%s", out)
	}
	if !strings.Contains(out, "api_key: <DEEPSEEK_API_KEY>") {
		t.Fatalf("expected placeholder:\n%s", out)
	}

	// Resolved from secrets store wins; env still ignored.
	const fromSecrets = "sk-FROM-SECRETS-STORE99"
	out2 := CopyText("deepseek", p, fromSecrets, true)
	if !strings.Contains(out2, fromSecrets) {
		t.Fatalf("expected secrets value:\n%s", out2)
	}
	if strings.Contains(out2, envLeak) {
		t.Fatalf("env leaked:\n%s", out2)
	}
}

func TestRedactKey(t *testing.T) {
	if RedactKey("sk-TEST-OPENAI") != "sk-T…ENAI" {
		t.Fatalf("%q", RedactKey("sk-TEST-OPENAI"))
	}
	if RedactKey("short") != "****" {
		t.Fatalf("%q", RedactKey("short"))
	}
	if RedactKey("") != "" {
		t.Fatalf("%q", RedactKey(""))
	}
}

func TestAddPickAndMeta(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.yaml")
	_ = os.WriteFile(path, []byte(ExampleProvidersYAML), 0o644)
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := f.AddPick("deepseek", "z-ai/glm-4.5")
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	changed, err = f.AddPick("deepseek", "z-ai/glm-4.5")
	if err != nil || changed {
		t.Fatalf("dup should be no-op: changed=%v err=%v", changed, err)
	}
	f.SetModelMeta("z-ai/glm-4.5", ModelMeta{Name: "GLM 4.5", ContextLength: 131072, Source: "openrouter"})
	if err := Save(path, f); err != nil {
		t.Fatal(err)
	}
	f2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range f2.Providers["deepseek"].Pick {
		if m == "z-ai/glm-4.5" {
			found = true
		}
	}
	if !found {
		t.Fatalf("pick=%v", f2.Providers["deepseek"].Pick)
	}
	if f2.ModelsMeta["z-ai/glm-4.5"].Name != "GLM 4.5" {
		t.Fatalf("meta=%v", f2.ModelsMeta)
	}
}
