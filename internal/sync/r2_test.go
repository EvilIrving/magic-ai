package sync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVaultDestRejectsEscape(t *testing.T) {
	home := filepath.Join(t.TempDir(), "lib")
	if _, ok := vaultDest(home, "vault/../../.ssh/authorized_keys"); ok {
		t.Fatal("expected reject")
	}
	if _, ok := vaultDest(home, "vault/"); ok {
		t.Fatal("empty rel")
	}
	dst, ok := vaultDest(home, "vault/providers.yaml")
	if !ok {
		t.Fatal("expected ok")
	}
	want := filepath.Join(filepath.Clean(home), "providers.yaml")
	if dst != want {
		t.Fatalf("dst=%s want=%s", dst, want)
	}
}

func TestSecretBearing(t *testing.T) {
	cases := map[string]bool{
		filepath.Join("secrets", "providers.json"):     true,
		filepath.Join("secrets", "providers.json.enc"): false,
		filepath.Join("secrets", "pi.json"):            true,
		"providers.yaml":                               false,
		filepath.Join("packs", "skills", "a.md"):       false,
	}
	for rel, want := range cases {
		if got := isSecretBearing(rel); got != want {
			t.Errorf("isSecretBearing(%q)=%v want %v", rel, got, want)
		}
	}
}

func TestContainsAnySecret(t *testing.T) {
	vals := []string{"sk-abc123456", "sk-def999999"}
	if !containsAnySecret([]byte(`{"apiKey":"sk-abc123456"}`), vals) {
		t.Fatal("should match embedded secret")
	}
	if containsAnySecret([]byte(`{"apiKey":"$KEY_ENV"}`), vals) {
		t.Fatal("must not match placeholder")
	}
	if containsAnySecret([]byte("plain config"), nil) {
		t.Fatal("nil values must not match")
	}
}

func TestLoadSecretValues(t *testing.T) {
	magic := t.TempDir()
	if err := os.MkdirAll(filepath.Join(magic, "secrets"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(magic, "secrets", "providers.json"), []byte(`{"A":"sk-longenough1","B":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	vals := loadSecretValues(magic)
	if len(vals) != 1 || vals[0] != "sk-longenough1" {
		t.Fatalf("vals=%v (short values must be ignored)", vals)
	}
}

func TestPushPlan(t *testing.T) {
	key, encrypt, skip := pushPlan("secrets/providers.json")
	if skip || !encrypt || key != "vault/secrets/providers.json.enc" {
		t.Fatalf("secret: key=%q encrypt=%v skip=%v", key, encrypt, skip)
	}
	key, encrypt, skip = pushPlan("providers.yaml")
	if skip || encrypt || key != "vault/providers.yaml" {
		t.Fatalf("plain: key=%q encrypt=%v skip=%v", key, encrypt, skip)
	}
	_, _, skip = pushPlan("secrets/providers.json.enc")
	if !skip {
		t.Fatal("local .enc must be skipped")
	}
}
