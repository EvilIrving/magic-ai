package core

import (
	"strings"
	"testing"
)

// projectBytes must signal the whole-file fallback instead of merging when the
// destination is not plain JSON — apply then writes the library bytes, and the
// plan says so.
func TestProjectBytesInvalidDstFallsBackToWholeFile(t *testing.T) {
	src := []byte(`{"a":1}`)
	out, strategy, err := projectBytes("settings.json", src, []byte(`{not json`), true)
	if err != nil {
		t.Fatalf("invalid dst: err=%v, want nil", err)
	}
	if strategy != StrategyJSONFallback {
		t.Fatalf("strategy = %q, want %q", strategy, StrategyJSONFallback)
	}
	if string(out) != string(src) {
		t.Fatalf("out = %q, want the library bytes", out)
	}
}

func TestProjectBytesInvalidSrcFallsBackToWholeFile(t *testing.T) {
	src := []byte(`{broken`)
	out, strategy, err := projectBytes("settings.json", src, []byte(`{"keep":1}`), true)
	if err != nil {
		t.Fatalf("invalid src: err=%v, want nil", err)
	}
	if strategy != StrategyJSONFallback {
		t.Fatalf("strategy = %q, want %q", strategy, StrategyJSONFallback)
	}
	if string(out) != string(src) {
		t.Fatalf("out = %q, want the library bytes", out)
	}
}

func TestProjectBytesMissingTargetCreates(t *testing.T) {
	src := []byte(`{"a":1}`)
	out, strategy, err := projectBytes("settings.json", src, nil, false)
	if err != nil || strategy != StrategyCreate {
		t.Fatalf("missing dst: strategy=%q err=%v, want %q", strategy, err, StrategyCreate)
	}
	if string(out) != string(src) {
		t.Fatalf("out = %q, want the library bytes", out)
	}
}

func TestProjectBytesMergesParsableJSON(t *testing.T) {
	out, strategy, err := projectBytes("settings.json", []byte(`{"a":{"lib":1}}`), []byte(`{"a":{"user":2},"keep":3}`), true)
	if err != nil || strategy != StrategyJSONMerge {
		t.Fatalf("strategy=%q err=%v, want %q", strategy, err, StrategyJSONMerge)
	}
	if !strings.Contains(string(out), `"user": 2`) || !strings.Contains(string(out), `"lib": 1`) || !strings.Contains(string(out), `"keep": 3`) {
		t.Fatalf("merge lost keys:\n%s", out)
	}
}

func TestProjectBytesTOMLMerge(t *testing.T) {
	out, strategy, err := projectBytes("config.toml", []byte("model = \"lib\"\n"), []byte("model = \"mine\"\napproval = \"never\"\n"), true)
	if err != nil || strategy != StrategyTOMLMerge {
		t.Fatalf("strategy=%q err=%v, want %q", strategy, err, StrategyTOMLMerge)
	}
	if !strings.Contains(string(out), `model = "lib"`) || !strings.Contains(string(out), `approval = "never"`) {
		t.Fatalf("toml merge wrong:\n%s", out)
	}
}

func TestProjectBytesWholeFileForPlainText(t *testing.T) {
	out, strategy, err := projectBytes("AGENTS.md", []byte("library\n"), []byte("user\n"), true)
	if err != nil || strategy != StrategyWholeFile {
		t.Fatalf("strategy=%q err=%v, want %q", strategy, err, StrategyWholeFile)
	}
	if string(out) != "library\n" {
		t.Fatalf("out = %q", out)
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
