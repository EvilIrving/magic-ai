package secrets

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactJSON(t *testing.T) {
	in := []byte(`{"api_key":"sk-secret","name":"ok","nested":{"auth_token":"tok"}}`)
	r := RedactBytes("x.json", in)
	if !r.Redacted {
		t.Fatal("expected redacted")
	}
	if !strings.Contains(string(r.Data), "${MAGIC_SECRET:") {
		t.Fatalf("body=%s", r.Data)
	}
	if r.Secrets["api_key"] != "sk-secret" {
		t.Fatalf("secrets=%v", r.Secrets)
	}
	if strings.Contains(string(r.Data), "sk-secret") {
		t.Fatal("raw secret leaked")
	}
}

func TestRedactTOML(t *testing.T) {
	in := []byte("name = \"x\"\napi_key = \"secret123\"\n")
	r := RedactBytes("c.toml", in)
	if !r.Redacted || r.Secrets["api_key"] != "secret123" {
		t.Fatalf("%v %s", r.Secrets, r.Data)
	}
}

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

func TestRedactReinjectRoundTrip(t *testing.T) {
	in := []byte(`{"api_key":"sk-live-abc","nested":{"auth_token":"tok-xyz"},"name":"keep"}`)
	redacted, restored, secrets, ok := RedactReinjectRoundTrip("cfg.json", in)
	if !ok {
		t.Fatal("expected redaction")
	}
	if strings.Contains(string(redacted), "sk-live-abc") {
		t.Fatal("secret in redacted")
	}
	if !strings.Contains(string(restored), "sk-live-abc") || !strings.Contains(string(restored), "tok-xyz") {
		t.Fatalf("reinject failed: %s", restored)
	}
	if !strings.Contains(string(restored), `"name": "keep"`) && !strings.Contains(string(restored), `"name":"keep"`) {
		t.Fatalf("non-secret lost: %s", restored)
	}
	if secrets["api_key"] != "sk-live-abc" {
		t.Fatalf("secrets=%v", secrets)
	}
}

func TestRedactJSONCAndEnvRefs(t *testing.T) {
	in := []byte(`{
  // comment
  "apiKey": "sk-live",
  "key": "sk-oauth",
  "access": "tok-access",
  "nested": { "env": { "GITHUB_PERSONAL_ACCESS_TOKEN": "ghp-secret" } },
  "piKey": "$SHU_API_KEY",
  "wbKey": "${DEEPSEEK_API_KEY}"
}`)
	r := RedactBytes("opencode.jsonc", in)
	if !r.Redacted {
		t.Fatal("expected redaction")
	}
	body := string(r.Data)
	if strings.Contains(body, "sk-live") || strings.Contains(body, "ghp-secret") || strings.Contains(body, "tok-access") {
		t.Fatalf("secret leaked: %s", body)
	}
	if !strings.Contains(body, "$SHU_API_KEY") || !strings.Contains(body, "${DEEPSEEK_API_KEY}") {
		t.Fatalf("env refs rewritten: %s", body)
	}
	if r.Secrets["apiKey"] != "sk-live" || r.Secrets["key"] != "sk-oauth" {
		t.Fatalf("secrets=%v", r.Secrets)
	}
}

func TestStoreLoadSecrets(t *testing.T) {
	dir := t.TempDir()
	sec := map[string]string{"api_key": "secret-val"}
	warned, err := StoreSecrets(dir, "pi", sec, "test-pass")
	if err != nil || warned {
		t.Fatalf("store: warned=%v err=%v", warned, err)
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
	_, err = LoadSecrets(dir, "pi", "")
	if err == nil {
		t.Fatal("expected error without passphrase")
	}
	_, err = LoadSecrets(filepath.Join(dir, "missing"), "nope", "x")
	if err != nil {
		t.Fatal(err)
	}
}
