package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProviderSecretsRoundTrip(t *testing.T) {
	t.Setenv("MAGIC_SECRETS_MODE", "")
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

func TestProviderSecretsPlaintextMode(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MAGIC_SECRETS_MODE", "plaintext")
	const keyEnv = "FOO_API_KEY"
	const val = "sk-plain-text-123"

	if err := SetProviderSecret(dir, "", keyEnv, val); err != nil {
		t.Fatalf("plaintext set without passphrase: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ProvidersFileName)); err == nil {
		t.Fatal("encrypted file must not exist in plaintext mode")
	}
	path := filepath.Join(dir, ProvidersPlaintextFileName)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("perm=%v, want 0600", perm)
	}
	got, ok, err := GetProviderSecret(dir, "", keyEnv)
	if err != nil || !ok || got != val {
		t.Fatalf("got %q ok=%v err=%v", got, ok, err)
	}
	if ResolveSecretMode(dir) != SecretModePlaintext {
		t.Fatalf("mode=%v", ResolveSecretMode(dir))
	}
	if SecretStorePath(dir) != path {
		t.Fatalf("store path=%q", SecretStorePath(dir))
	}
}

func TestProviderSecretsAutoDetectPlaintext(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MAGIC_SECRETS_MODE", "")
	if err := os.WriteFile(filepath.Join(dir, ProvidersPlaintextFileName), []byte(`{"FOO_API_KEY":"bar"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if ResolveSecretMode(dir) != SecretModePlaintext {
		t.Fatalf("auto-detect should pick plaintext, got %v", ResolveSecretMode(dir))
	}
	v, ok, err := GetProviderSecret(dir, "", "FOO_API_KEY")
	if err != nil || !ok || v != "bar" {
		t.Fatalf("got %q ok=%v err=%v", v, ok, err)
	}
}

func TestProviderSecretsEncryptedWinsOverPlaintext(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MAGIC_SECRETS_MODE", "")
	const pass = "pw"
	if err := SetProviderSecret(dir, pass, "A_API_KEY", "1"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ProvidersPlaintextFileName), []byte(`{"A_API_KEY":"DECOY"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if ResolveSecretMode(dir) != SecretModeEncrypted {
		t.Fatalf("encrypted should win, got %v", ResolveSecretMode(dir))
	}
	v, _, err := GetProviderSecret(dir, pass, "A_API_KEY")
	if err != nil || v != "1" {
		t.Fatalf("got %q err=%v", v, err)
	}
}
