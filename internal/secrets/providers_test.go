package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProviderSecretsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	const keyEnv = "DEEPSEEK_API_KEY"
	const val = "sk-TEST-PROVIDER-KEY99"

	m, err := LoadProviderSecrets(dir)
	if err != nil || len(m) != 0 {
		t.Fatalf("missing → empty: m=%v err=%v", m, err)
	}

	if err := SetProviderSecret(dir, keyEnv, val); err != nil {
		t.Fatal(err)
	}
	path := ProvidersPath(dir)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("perm=%v, want 0600", perm)
	}
	if filepath.Base(path) != ProvidersFileName || ProvidersFileName != "providers.json" {
		t.Fatalf("local store must be plaintext providers.json, got %s", ProvidersFileName)
	}

	got, ok, err := GetProviderSecret(dir, keyEnv)
	if err != nil || !ok || got != val {
		t.Fatalf("got %q ok=%v err=%v", got, ok, err)
	}

	// merge second key
	if err := SetProviderSecret(dir, "OPENROUTER_API_KEY", "sk-or-xxx"); err != nil {
		t.Fatal(err)
	}
	names, err := ListProviderSecretNames(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "DEEPSEEK_API_KEY" || names[1] != "OPENROUTER_API_KEY" {
		t.Fatalf("names=%v", names)
	}

	// empty key_env rejected
	if err := SetProviderSecret(dir, "", "y"); err == nil {
		t.Fatal("expected empty key_env to fail")
	}
}

func TestProviderSecretsFileIsPlainJSON(t *testing.T) {
	dir := t.TempDir()
	if err := SetProviderSecret(dir, "FOO_API_KEY", "sk-plain-text-123"); err != nil {
		t.Fatalf("set: %v", err)
	}
	raw, err := os.ReadFile(ProvidersPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"FOO_API_KEY\": \"sk-plain-text-123\"\n}"
	if string(raw) != want {
		t.Fatalf("local file must be plaintext JSON:\ngot  %s\nwant %s", raw, want)
	}
	// No hidden encrypted companion is written.
	if _, err := os.Stat(filepath.Join(dir, "providers.json.enc")); err == nil {
		t.Fatal("local .enc must not exist")
	}
}

func TestProviderSecretsMissingKey(t *testing.T) {
	dir := t.TempDir()
	_, ok, err := GetProviderSecret(dir, "NOPE_API_KEY")
	if err != nil || ok {
		t.Fatalf("missing key: ok=%v err=%v", ok, err)
	}
}
