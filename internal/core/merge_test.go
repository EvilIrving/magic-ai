package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectBytesKeepsUserKeys(t *testing.T) {
	dst := []byte(`{"permissions":{"allow":["Bash"]},"env":{"OTHER":"1","KEEP":"x"}}`)
	src := []byte(`{"env":{"OTHER":"2"},"statusLine":{"type":"command"}}`)
	merged, strategy, err := projectBytes("settings.json", src, dst, true)
	if err != nil || strategy != StrategyJSONMerge {
		t.Fatalf("strategy=%q err=%v", strategy, err)
	}
	s := string(merged)
	if !strings.Contains(s, `"KEEP"`) || !strings.Contains(s, `"permissions"`) {
		t.Fatalf("user keys dropped:\n%s", s)
	}
	if !strings.Contains(s, `"OTHER": "2"`) || !strings.Contains(s, "statusLine") {
		t.Fatalf("library keys not applied:\n%s", s)
	}
}

func TestDeepMergeMapNested(t *testing.T) {
	dst := map[string]any{"a": map[string]any{"x": 1, "y": 2}, "keep": true}
	src := map[string]any{"a": map[string]any{"y": 9, "z": 3}}
	out := deepMergeMap(dst, src)
	a := out["a"].(map[string]any)
	if a["x"] != 1 || a["y"] != 9 || a["z"] != 3 {
		t.Fatalf("nested merge wrong: %v", a)
	}
	if out["keep"] != true {
		t.Fatalf("sibling dropped: %v", out)
	}
}

func TestMergeTOMLUpsertsAndKeepsUserKeys(t *testing.T) {
	dst := "" +
		"model_provider = \"openai\"\n" +
		"user_key = \"keep\"\n\n" +
		"[models]\n" +
		"default = \"grok-4.5\"\n" +
		"user_model = \"mine\"\n"
	src := "" +
		"model_provider = \"shu\"\n\n" +
		"[models]\n" +
		"default = \"gpt-5.6-sol\"\n"
	out := mergeTOML(dst, src)
	if !strings.Contains(out, `model_provider = "shu"`) {
		t.Fatalf("preamble not updated:\n%s", out)
	}
	if !strings.Contains(out, `user_key = "keep"`) {
		t.Fatalf("user preamble key dropped:\n%s", out)
	}
	if !strings.Contains(out, `default = "gpt-5.6-sol"`) {
		t.Fatalf("section value not updated:\n%s", out)
	}
	if !strings.Contains(out, `user_model = "mine"`) {
		t.Fatalf("user section key dropped:\n%s", out)
	}
	if strings.Count(out, "[models]") != 1 {
		t.Fatalf("duplicate section:\n%s", out)
	}
}

func TestMergeTOMLArrayOfTablesReplaced(t *testing.T) {
	dst := "" +
		"[[model.\"x\".reasoning_efforts]]\n" +
		"value = \"old\"\n\n" +
		"[[model.\"x\".reasoning_efforts]]\n" +
		"value = \"old2\"\n"
	src := "" +
		"[[model.\"x\".reasoning_efforts]]\n" +
		"value = \"low\"\n\n" +
		"[[model.\"x\".reasoning_efforts]]\n" +
		"value = \"high\"\n"
	out := mergeTOML(dst, src)
	if strings.Contains(out, "old") {
		t.Fatalf("stale array entry kept:\n%s", out)
	}
	if strings.Count(out, "reasoning_efforts") != 2 {
		t.Fatalf("expected library array blocks:\n%s", out)
	}
}

func TestMergeLibraryFileOverwritesWholeFile(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "AGENTS.md")
	_ = os.WriteFile(dst, []byte("user local\n"), 0o644)
	if err := mergeLibraryFile("", "", dst, "AGENTS.md", []byte("library\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(dst)
	if string(raw) != "library\n" {
		t.Fatalf("whole-file semantics broken: %s", raw)
	}
}
