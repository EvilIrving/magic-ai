package secrets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptRoundTrip(t *testing.T) {
	ct, err := Encrypt([]byte("hello"), "pass")
	if err != nil {
		t.Fatal(err)
	}
	pt, err := Decrypt(ct, "pass")
	if err != nil || string(pt) != "hello" {
		t.Fatalf("%s %v", pt, err)
	}
}

func TestLoadReinject(t *testing.T) {
	dir := t.TempDir()
	sec := map[string]string{"api_key": "secret-val"}
	payload, _ := json.Marshal(sec)
	enc, err := Encrypt(payload, "test-pass")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pi.json.enc"), enc, 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadSecrets(dir, "pi", "test-pass")
	if err != nil {
		t.Fatal(err)
	}
	if got["api_key"] != "secret-val" {
		t.Fatalf("got=%v", got)
	}
	data := []byte(`{"api_key":"${MAGIC_SECRET:api_key}"}`)
	out := ReinjectBytes(data, got)
	if string(out) != `{"api_key":"secret-val"}` {
		t.Fatalf("reinject=%s", out)
	}

	if _, err := LoadSecrets(dir, "pi", ""); err == nil {
		t.Fatal("expected error without passphrase")
	}
	if got, err := LoadSecrets(filepath.Join(dir, "missing"), "nope", "x"); err != nil || got != nil {
		t.Fatalf("missing file: got=%v err=%v", got, err)
	}
}

func TestStripJSONC(t *testing.T) {
	in := []byte("{\n  // c\n  \"a\": \"http://x//y\", /* b */\n  \"b\": 1\n}")
	out := string(StripJSONC(in))
	if !json.Valid([]byte(out)) {
		t.Fatalf("not valid json: %s", out)
	}
}
