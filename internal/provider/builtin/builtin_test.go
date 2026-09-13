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

func TestMergeUserWins(t *testing.T) {
	user := &catalog.File{
		Active: "mine",
		Providers: map[string]catalog.Provider{
			"deepseek": {
				KeyEnv:    "MY_DEEPSEEK_KEY",
				Pick:      []string{"custom-model"},
				Endpoints: []catalog.Endpoint{{ID: "relay", BaseURL: "https://relay.example.com", Protocol: catalog.ProtocolOpenAIChatCompletions}},
				Models:    map[string]string{"haiku": "h", "default": "d-user"},
			},
		},
	}
	merged, err := Merge(user)
	if err != nil {
		t.Fatal(err)
	}
	if merged.Active != "mine" {
		t.Fatalf("active lost: %q", merged.Active)
	}
	// Template-only providers are available.
	if _, ok := merged.Providers["zai"]; !ok {
		t.Fatal("template provider zai missing after merge")
	}
	ds := merged.Providers["deepseek"]
	if ds.KeyEnv != "MY_DEEPSEEK_KEY" {
		t.Fatalf("user key_env must win: %q", ds.KeyEnv)
	}
	if len(ds.Endpoints) != 1 || ds.Endpoints[0].ID != "relay" {
		t.Fatalf("user endpoints must replace whole: %+v", ds.Endpoints)
	}
	if len(ds.Pick) != 1 || ds.Pick[0] != "custom-model" {
		t.Fatalf("user pick must replace whole: %+v", ds.Pick)
	}
	if ds.Models["default"] != "d-user" {
		t.Fatalf("user model role must win: %+v", ds.Models)
	}
}

func TestAddSelection(t *testing.T) {
	user := &catalog.File{Providers: map[string]catalog.Provider{"deepseek": {KeyEnv: "KEEP"}}}
	out, added, skipped, missing, err := AddSelection(user, []string{"zai", "deepseek", "nope"})
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 1 || added[0] != "zai" {
		t.Fatalf("added=%v", added)
	}
	if len(skipped) != 1 || skipped[0] != "deepseek" {
		t.Fatalf("skipped=%v", skipped)
	}
	if len(missing) != 1 || missing[0] != "nope" {
		t.Fatalf("missing=%v", missing)
	}
	if out.Providers["deepseek"].KeyEnv != "KEEP" {
		t.Fatal("existing provider overwritten")
	}
	if _, ok := out.Providers["zai"]; !ok {
		t.Fatal("zai not added")
	}
	// Original file must be untouched.
	if len(user.Providers) != 1 {
		t.Fatal("AddSelection mutated the input file")
	}
}
