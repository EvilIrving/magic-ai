package builtin

import (
	"testing"

	"magic/internal/catalog"
)

func TestLoadTemplateValid(t *testing.T) {
	provs, err := Providers()
	if err != nil {
		t.Fatal(err)
	}
	if len(provs) < 20 {
		t.Fatalf("expected a broad official template, got %d providers", len(provs))
	}
	for _, relay := range []string{"shu", "shu-deepseek", "shu-grok", "shu-qwen", "openai-next", "opencode-go", "groq", "together"} {
		if _, ok := provs[relay]; ok {
			t.Errorf("relay %q must not be in the builtin catalog", relay)
		}
	}
	allowed := map[string]bool{
		catalog.ProtocolOpenAIChatCompletions: true,
		catalog.ProtocolOpenAIResponses:       true,
		catalog.ProtocolAnthropicMessages:     true,
	}
	for name, p := range provs {
		if p.KeyEnv == "" {
			t.Errorf("%s: missing key_env", name)
		}
		eps := catalog.NormalizeEndpoints(p)
		if len(eps) == 0 {
			t.Errorf("%s: no endpoints", name)
		}
		for _, ep := range eps {
			if !allowed[ep.Protocol] {
				t.Errorf("%s/%s: unsupported protocol %q", name, ep.ID, ep.Protocol)
			}
			if ep.BaseURL == "" {
				t.Errorf("%s/%s: empty base_url", name, ep.ID)
			}
			if ep.ID == "" {
				t.Errorf("%s: endpoint with empty id", name)
			}
		}
	}
}

func TestDeepSeekPickIsCurrent(t *testing.T) {
	provs, err := Providers()
	if err != nil {
		t.Fatal(err)
	}
	p := provs["deepseek"]
	for _, id := range p.Pick {
		if id == "deepseek-chat" || id == "deepseek-reasoner" {
			t.Fatalf("retired DeepSeek id in pick: %q", id)
		}
	}
	want := map[string]bool{"deepseek-flash": false, "deepseek-v4-pro": false}
	for _, id := range p.Pick {
		if _, ok := want[id]; ok {
			want[id] = true
		}
	}
	for id, ok := range want {
		if !ok {
			t.Errorf("missing current DeepSeek pick %q", id)
		}
	}
}

func TestLoadCarriesDisplayMetadata(t *testing.T) {
	provs, err := Providers()
	if err != nil {
		t.Fatal(err)
	}
	zai, ok := provs["zai"]
	if !ok {
		t.Fatal("zai missing from template")
	}
	if zai.Name == "" {
		t.Fatal("zai: name not parsed (catalog.Provider.Name)")
	}
}

func TestTemplateHasDeepSeekAnthropicEndpoint(t *testing.T) {
	provs, err := Providers()
	if err != nil {
		t.Fatal(err)
	}
	for _, ep := range catalog.NormalizeEndpoints(provs["deepseek"]) {
		if ep.Protocol == catalog.ProtocolAnthropicMessages && ep.ID == "anthropic" {
			return
		}
	}
	t.Fatal("deepseek: expected anthropic-messages endpoint")
}
