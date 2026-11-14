package catalog

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMergePiModelsSmoke(t *testing.T) {
	existing := []byte(`{"providers":{"keep-me":{"baseUrl":"https://x","api":"openai-completions","apiKey":"$X","models":[]}}}`)
	ep := Endpoint{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: ProtocolOpenAIResponses}
	entry := BuildPiProviderEntry(ep, "sk-real-pi-key", []string{"gpt-5.6-sol"}, nil)
	if entry.APIKey != "sk-real-pi-key" {
		t.Fatalf("apiKey must be the real value, got %q", entry.APIKey)
	}
	if entry.API != "openai-responses" {
		t.Fatalf("api=%s", entry.API)
	}
	if len(entry.Models) != 1 || entry.Models[0].ID != "gpt-5.6-sol" {
		t.Fatalf("%+v", entry.Models)
	}
	out, err := MergePiModelsJSON(existing, MagicPiProviderID("shu"), entry)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(out, &root); err != nil {
		t.Fatal(err)
	}
	provs := root["providers"].(map[string]any)
	if _, ok := provs["keep-me"]; !ok {
		t.Fatal("keep-me wiped")
	}
	magic := provs["magic-shu"].(map[string]any)
	if magic["baseUrl"] != "https://api.shu.cool/v1" {
		t.Fatalf("%v", magic)
	}
	if magic["apiKey"] != "sk-real-pi-key" {
		t.Fatalf("pi must carry the real key (D3), got %v", magic["apiKey"])
	}

	if !strings.Contains(string(out), "sk-real-pi-key") {
		t.Fatalf("plaintext key missing: %s", out)
	}
}
