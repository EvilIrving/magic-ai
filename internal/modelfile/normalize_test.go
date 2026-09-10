package modelfile

import (
	"os"
	"path/filepath"
	"testing"

	"magic/internal/modelsdev"
	"magic/internal/openrouter"
)

func loadFixture(t *testing.T) modelsdev.Catalog {
	t.Helper()
	// testdata lives at module root
	path := filepath.Join("..", "..", "testdata", "registry", "models.dev.fixture.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	c, err := modelsdev.ParseCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNormalizeFixtureFields(t *testing.T) {
	cat, err := Normalize(NormalizeOptions{
		ModelsDev: loadFixture(t),
		SyncedAt:  "2026-09-09T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cat.Version != 1 || cat.Source != "models.dev" {
		t.Fatalf("envelope: %+v", cat)
	}
	if cat.SyncedAt != "2026-09-09T00:00:00Z" {
		t.Fatalf("synced_at=%s", cat.SyncedAt)
	}
	byKey := map[string]Model{}
	for _, m := range cat.Models {
		byKey[m.Provider+"/"+m.ID] = m
	}

	gpt4o, ok := byKey["openai/gpt-4o"]
	if !ok {
		t.Fatal("missing gpt-4o")
	}
	if gpt4o.API != "openai-responses" || gpt4o.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("gpt-4o api/base=%s %s", gpt4o.API, gpt4o.BaseURL)
	}
	if gpt4o.ContextWindow != 128000 || gpt4o.MaxTokens != 16384 {
		t.Fatalf("limits %+v", gpt4o)
	}
	if gpt4o.Cost.Input != 2.5 || gpt4o.Cost.Output != 10 || gpt4o.Cost.CacheRead != 1.25 {
		t.Fatalf("cost %+v", gpt4o.Cost)
	}
	if len(gpt4o.Input) != 2 || gpt4o.Input[1] != "image" {
		t.Fatalf("input %v", gpt4o.Input)
	}
	if gpt4o.Reasoning {
		t.Fatal("gpt-4o reasoning false")
	}
	if gpt4o.ThinkingLevelMap != nil {
		t.Fatal("gpt-4o should have no thinking map")
	}

	nano, ok := byKey["openai/gpt-5-nano"]
	if !ok {
		t.Fatal("missing gpt-5-nano")
	}
	if !nano.Reasoning {
		t.Fatal("nano reasoning")
	}
	tlm := nano.ThinkingLevelMap
	if tlm == nil || tlm["off"] != nil {
		t.Fatalf("nano off=%v", tlm["off"])
	}
	if tlm["minimal"] == nil || *tlm["minimal"] != "minimal" {
		t.Fatal("minimal")
	}
	if tlm["max"] != nil {
		t.Fatal("max should be null")
	}

	opus, ok := byKey["anthropic/claude-opus-4-6"]
	if !ok {
		t.Fatal("missing opus")
	}
	if opus.API != "anthropic-messages" || opus.BaseURL != "https://api.anthropic.com" {
		t.Fatalf("opus api %s %s", opus.API, opus.BaseURL)
	}
	if opus.ThinkingLevelMap["max"] == nil || *opus.ThinkingLevelMap["max"] != "max" {
		t.Fatal("opus max")
	}
	if opus.ThinkingLevelMap["minimal"] != nil {
		t.Fatal("opus minimal null")
	}

	ds, ok := byKey["deepseek/deepseek-v4-flash"]
	if !ok {
		t.Fatal("missing deepseek")
	}
	if ds.API != "openai-completions" || ds.BaseURL != "https://api.deepseek.com" {
		t.Fatalf("deepseek %s %s", ds.API, ds.BaseURL)
	}
	if ds.ThinkingLevelMap["low"] == nil || *ds.ThinkingLevelMap["low"] != "low" {
		t.Fatal("ds low")
	}

	// include-all: no-tools model present with toolCall=false
	nt, ok := byKey["deepseek/no-tools-model"]
	if !ok {
		t.Fatal("expected no-tools-model included")
	}
	if nt.ToolCall == nil || *nt.ToolCall {
		t.Fatalf("toolCall=%v", nt.ToolCall)
	}

	lab, ok := byKey["obscure-lab/lab-1"]
	if !ok {
		t.Fatal("obscure-lab")
	}
	if !lab.Unsupported {
		t.Fatal("unknown provider should be unsupported")
	}
	if lab.BaseURL != "https://api.obscure.example/v1" {
		t.Fatalf("baseUrl from provider.api: %s", lab.BaseURL)
	}
}

func TestNormalizeOpenRouterEnrichmentNoOverwrite(t *testing.T) {
	or := &openrouter.Cache{
		Models: []openrouter.Model{
			{ID: "openai/gpt-4o", Name: "OR GPT-4o", ContextLength: 999},
			{ID: "z-ai/glm-4.5", Name: "GLM 4.5", ContextLength: 128000},
		},
	}
	cat, err := Normalize(NormalizeOptions{
		ModelsDev:  loadFixture(t),
		OpenRouter: or,
		SyncedAt:   "2026-09-09T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	// models.dev openai/gpt-4o kept; OR adds openrouter/openai/gpt-4o as separate key
	var mdGpt, orGpt, glm *Model
	for i := range cat.Models {
		m := &cat.Models[i]
		if m.Provider == "openai" && m.ID == "gpt-4o" {
			mdGpt = m
		}
		if m.Provider == "openrouter" && m.ID == "openai/gpt-4o" {
			orGpt = m
		}
		if m.Provider == "openrouter" && m.ID == "z-ai/glm-4.5" {
			glm = m
		}
	}
	if mdGpt == nil || mdGpt.ContextWindow != 128000 {
		t.Fatalf("models.dev gpt-4o overwritten? %+v", mdGpt)
	}
	if orGpt == nil {
		t.Fatal("expected OR enrichment entry openrouter/openai/gpt-4o")
	}
	if glm == nil || glm.ContextWindow != 128000 {
		t.Fatalf("glm %+v", glm)
	}
	if cat.Source != "models.dev+openrouter" {
		t.Fatalf("source=%s", cat.Source)
	}
}

func TestFindFuzzy(t *testing.T) {
	cat, err := Normalize(NormalizeOptions{ModelsDev: loadFixture(t), SyncedAt: "t"})
	if err != nil {
		t.Fatal(err)
	}
	m, ok := FindFuzzy(cat, "gpt-4o", "")
	if !ok || m.Provider != "openai" {
		t.Fatalf("exact id: %+v ok=%v", m, ok)
	}
	m, ok = FindFuzzy(cat, "openai/gpt-4o", "")
	if !ok || m.ID != "gpt-4o" {
		t.Fatalf("provider/id: %+v", m)
	}
	m, ok = FindFuzzy(cat, "gpt.4o", "openai")
	if !ok || m.ID != "gpt-4o" {
		t.Fatalf("collapse: %+v", m)
	}
	m, ok = FindFuzzy(cat, "claude_opus_4_6", "anthropic")
	if !ok || m.ID != "claude-opus-4-6" {
		t.Fatalf("collapse dashes: %+v", m)
	}
}

func TestWriteLoadRoundtrip(t *testing.T) {
	cat, err := Normalize(NormalizeOptions{ModelsDev: loadFixture(t), SyncedAt: "2026-09-09T12:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	mh := t.TempDir()
	if err := WriteCatalog(mh, cat); err != nil {
		t.Fatal(err)
	}
	got, err := LoadCatalog(mh)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Models) != len(cat.Models) {
		t.Fatalf("len %d vs %d", len(got.Models), len(cat.Models))
	}
	hits := Search(got, "opus")
	if len(hits) != 1 {
		t.Fatalf("search opus: %d", len(hits))
	}
}
