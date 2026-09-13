package secrets

import (
	"encoding/json"
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

func TestStripJSONC(t *testing.T) {
	in := []byte("{\n  // c\n  \"a\": \"http://x//y\", /* b */\n  \"b\": 1\n}")
	out := string(StripJSONC(in))
	if !json.Valid([]byte(out)) {
		t.Fatalf("not valid json: %s", out)
	}
}
