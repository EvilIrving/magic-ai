package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProviderSecretsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	const pass = "test-passphrase"
	const keyEnv = "DEEPSEEK_API_KEY"
	const val = "sk-TEST-PROVIDER-KEY99"

	m, err := LoadProviderSecrets(dir, pass)
	if err != nil || len(m) != 0 {
		t.Fatalf("missing → empty: m=%v err=%v", m, err)
	}

	if err := SetProviderSecret(dir, pass, keyEnv, val); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ProvidersFileName)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}

	got, ok, err := GetProviderSecret(dir, pass, keyEnv)
	if err != nil || !ok || got != val {
		t.Fatalf("got %q ok=%v err=%v", got, ok, err)
	}

	// merge second key
	if err := SetProviderSecret(dir, pass, "OPENROUTER_API_KEY", "sk-or-xxx"); err != nil {
		t.Fatal(err)
	}
	names, err := ListProviderSecretNames(dir, pass)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "DEEPSEEK_API_KEY" || names[1] != "OPENROUTER_API_KEY" {
		t.Fatalf("names=%v", names)
	}

	// wrong passphrase
	_, err = LoadProviderSecrets(dir, "wrong")
	if err == nil {
		t.Fatal("expected decrypt error")
	}

	// passphrase unset while file exists
	_, err = LoadProviderSecrets(dir, "")
	if err == nil {
		t.Fatal("expected passphrase error")
	}

	// empty passphrase rejected on set
	if err := SetProviderSecret(dir, "", "X", "y"); err == nil {
		t.Fatal("expected passphrase required")
	}
}
