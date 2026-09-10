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
