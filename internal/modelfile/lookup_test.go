package modelfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePickMetaFuzzy(t *testing.T) {
	magic := t.TempDir()
	_ = os.MkdirAll(filepath.Join(magic, "registry"), 0o755)
	cat := &Catalog{
		Version: 1,
		Source:  "test",
		Models: []Model{
			{
				ID:            "gpt-5.6-sol",
				Name:          "GPT-5.6 Sol",
				Provider:      "openai",
				Reasoning:     true,
				Input:         []string{"text", "image"},
				ContextWindow: 1050000,
				MaxTokens:     128000,
				ThinkingLevelMap: map[string]*string{
					"off":   strPtr("none"),
					"high":  strPtr("high"),
					"xhigh": strPtr("xhigh"),
				},
				Cost: Cost{Input: 1, Output: 2},
			},
		},
	}
	if err := WriteCatalog(magic, cat); err != nil {
		t.Fatal(err)
	}

	m, ok := ResolvePickMeta(magic, "openai/gpt-5.6-sol")
	if !ok || m.ContextWindow != 1050000 {
		t.Fatalf("exact+vendor: ok=%v %+v", ok, m)
	}
	m, ok = ResolvePickMeta(magic, "gpt_5_6_sol")
	if !ok || m.Name != "GPT-5.6 Sol" {
		t.Fatalf("collapse: ok=%v %+v", ok, m)
	}
	_, ok = ResolvePickMeta(magic, "missing-model")
	if ok {
		t.Fatal("expected miss")
	}
	_, ok = ResolvePickMeta(t.TempDir(), "gpt-5.6-sol")
	if ok {
		t.Fatal("missing catalog should miss")
	}
}

func strPtr(s string) *string { return &s }
