package sync

import (
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

func TestPlaintextProviderStoreExcluded(t *testing.T) {
	cases := map[string]bool{
		filepath.Join("secrets", "providers.json"):     true,
		filepath.Join("secrets", "providers.json.enc"): false,
		filepath.Join("secrets", "pi.json.enc"):        false,
		filepath.Join("providers.yaml"):                false,
	}
	for rel, want := range cases {
		if got := isPlaintextProviderStore(rel); got != want {
			t.Errorf("isPlaintextProviderStore(%q)=%v want %v", rel, got, want)
		}
	}
}
