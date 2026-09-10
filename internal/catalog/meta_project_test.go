package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"magic/internal/modelfile"
)

func TestBuildPiProviderEntryEnrichesFromMeta(t *testing.T) {
	magic := t.TempDir()
	_ = os.MkdirAll(filepath.Join(magic, "registry"), 0o755)
	off := "none"
	high := "high"
	xhigh := "xhigh"
	cat := &modelfile.Catalog{
		Version: 1,
		Source:  "test",
		Models: []modelfile.Model{{
			ID:            "gpt-5.6-sol",
			Name:          "GPT-5.6 Sol",
			Provider:      "openai",
			Reasoning:     true,
			Input:         []string{"text", "image"},
			ContextWindow: 1050000,
			MaxTokens:     128000,
			ThinkingLevelMap: map[string]*string{
				"off":   &off,
				"high":  &high,
				"xhigh": &xhigh,
			},
			Cost: modelfile.Cost{Input: 1.5, Output: 12, CacheRead: 0.75},
		}},
	}
	if err := modelfile.WriteCatalog(magic, cat); err != nil {
		t.Fatal(err)
	}

	ep := Endpoint{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: ProtocolOpenAIResponses}
	entry := BuildPiProviderEntry(ep, "SHU_API_KEY", []string{"gpt-5.6-sol", "unknown-pick"}, modelfile.MetaResolver(magic))
	if len(entry.Models) != 2 {
		t.Fatalf("%+v", entry.Models)
	}
	sol := entry.Models[0]
	if sol.Name != "GPT-5.6 Sol" || sol.ContextWindow != 1050000 || sol.MaxTokens != 128000 {
		t.Fatalf("sol meta: %+v", sol)
	}
	if !sol.Reasoning || sol.ThinkingLevelMap == nil || sol.ThinkingLevelMap["high"] == nil {
		t.Fatalf("thinking: %+v", sol)
	}
	if sol.Cost == nil || sol.Cost.Input != 1.5 {
		t.Fatalf("cost: %+v", sol.Cost)
	}
	unk := entry.Models[1]
	if unk.ID != "unknown-pick" || unk.Name != "unknown-pick" || unk.ContextWindow != 0 || unk.ThinkingLevelMap != nil {
		t.Fatalf("unknown should stay minimal: %+v", unk)
	}

	raw, err := MergePiModelsJSON(nil, "magic-shu", entry)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatal(err)
	}
	provs := root["providers"].(map[string]any)
	magicProv := provs["magic-shu"].(map[string]any)
	models := magicProv["models"].([]any)
	first := models[0].(map[string]any)
	if first["contextWindow"].(float64) != 1050000 {
		t.Fatalf("json contextWindow: %v", first)
	}
	if _, ok := first["thinkingLevelMap"]; !ok {
		t.Fatalf("json missing thinkingLevelMap: %s", raw)
	}
	// unknown model should omit empty optional fields
	second := models[1].(map[string]any)
	if _, ok := second["contextWindow"]; ok {
		t.Fatalf("unknown should omit contextWindow: %v", second)
	}
}

func TestClaudeModelRoleEnvDefaultFromPick(t *testing.T) {
	env := ClaudeModelRoleEnv(Provider{
		Pick:   []string{"gpt-5.6-sol"},
		Models: map[string]string{"haiku": "gpt-5.6-luna"},
	})
	if env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] != "gpt-5.6-luna" {
		t.Fatalf("%v", env)
	}
	if env["ANTHROPIC_MODEL"] != "gpt-5.6-sol" {
		t.Fatalf("expected first pick as default when Models[default] unset: %v", env)
	}
	if env["CLAUDE_CODE_SUBAGENT_MODEL"] != "gpt-5.6-luna" {
		t.Fatalf("subagent should inherit haiku: %v", env)
	}
	env2 := ClaudeModelRoleEnv(Provider{
		Pick:   []string{"ignored"},
		Models: map[string]string{"default": "explicit-default"},
	})
	if env2["ANTHROPIC_MODEL"] != "explicit-default" {
		t.Fatalf("%v", env2)
	}
}
