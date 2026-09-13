package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mergeJSONObjects must never delete user keys the library doesn't mention,
// and must signal "skip" (ok=false) instead of failing when the destination
// is missing or unparsable — apply treats that as "write another way".
func TestMergeJSONMissingDstSkips(t *testing.T) {
	_, ok, err := mergeJSONObjects([]byte(`{"a":1}`), filepath.Join(t.TempDir(), "nope.json"))
	if err != nil || ok {
		t.Fatalf("missing dst: ok=%v err=%v, want ok=false err=nil", ok, err)
	}
}

func TestMergeJSONInvalidDstSkips(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "bad.json")
	_ = os.WriteFile(dst, []byte(`{not json`), 0o600)
	_, ok, err := mergeJSONObjects([]byte(`{"a":1}`), dst)
	if err != nil || ok {
		t.Fatalf("invalid dst: ok=%v err=%v, want ok=false err=nil", ok, err)
	}
}

func TestMergeJSONInvalidSrcSkips(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "s.json")
	_ = os.WriteFile(dst, []byte(`{"keep":1}`), 0o600)
	_, ok, err := mergeJSONObjects([]byte(`{broken`), dst)
	if err != nil || ok {
		t.Fatalf("invalid src: ok=%v err=%v, want ok=false err=nil", ok, err)
	}
}

// Library arrays replace destination arrays wholesale (D5: one file, one
// entry at a time); user scalars alongside must survive.
func TestDeepMergeArrayReplacedScalarsKept(t *testing.T) {
	dst := map[string]any{"list": []any{1, 2}, "user": "mine"}
	src := map[string]any{"list": []any{3}}
	out := deepMergeMap(dst, src)
	if got := out["list"].([]any); len(got) != 1 || got[0] != 3 {
		t.Fatalf("array not replaced: %v", out["list"])
	}
	if out["user"] != "mine" {
		t.Fatalf("user scalar dropped: %v", out)
	}
}

// Empty library section must not wipe the user's TOML section.
func TestMergeTOMLEmptySrcKeepsDst(t *testing.T) {
	dst := "[models]\ndefault = \"mine\"\n"
	out := mergeTOML(dst, "")
	if !strings.Contains(out, `default = "mine"`) {
		t.Fatalf("user section wiped:\n%s", out)
	}
}

// A library preamble key updates the value in place without duplicating it.
func TestMergeTOMLPreambleNoDuplicate(t *testing.T) {
	dst := "model = \"a\"\n"
	out := mergeTOML(dst, "model = \"b\"\n")
	if strings.Count(out, "model =") != 1 {
		t.Fatalf("duplicated preamble key:\n%s", out)
	}
	if !strings.Contains(out, `model = "b"`) {
		t.Fatalf("library value not applied:\n%s", out)
	}
}
