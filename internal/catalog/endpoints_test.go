package catalog

import (
	"strings"
	"testing"
)

func TestNormalizeEndpointsMultiAndCompat(t *testing.T) {
	multi := Provider{
		KeyEnv: "SHU_API_KEY",
		Pick:   []string{"gpt-5.6-sol"},
		Endpoints: []Endpoint{
			{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: "openai-responses"},
			{ID: "openai-chat", BaseURL: "https://api.shu.cool/v1", Protocol: "openai-chat-completions"},
			{ID: "anthropic", BaseURL: "https://api.shu.cool", Protocol: "anthropic-messages"},
		},
	}
	eps := NormalizeEndpoints(multi)
	if len(eps) != 3 {
		t.Fatalf("len=%d", len(eps))
	}
	if eps[0].Protocol != ProtocolOpenAIResponses || eps[2].Protocol != ProtocolAnthropicMessages {
		t.Fatalf("%+v", eps)
	}

	legacy := Provider{BaseURL: "https://api.deepseek.com/v1", Protocol: "openai", KeyEnv: "DEEPSEEK_API_KEY"}
	eps2 := NormalizeEndpoints(legacy)
	if len(eps2) != 1 || eps2[0].BaseURL != legacy.BaseURL || eps2[0].Protocol != ProtocolOpenAIChatCompletions {
		t.Fatalf("%+v", eps2)
	}

	empty := NormalizeEndpoints(Provider{KeyEnv: "X"})
	if empty != nil {
		t.Fatalf("%+v", empty)
	}
}

func TestPickEndpoint(t *testing.T) {
	p := Provider{
		Endpoints: []Endpoint{
			{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: "openai-responses"},
			{ID: "openai-chat", BaseURL: "https://api.shu.cool/v1", Protocol: "openai"},
			{ID: "anthropic", BaseURL: "https://api.shu.cool", Protocol: "anthropic"},
		},
	}
	ep, ok := PickEndpoint(p)
	if !ok || ep.ID != "openai-responses" {
		t.Fatalf("default first: %+v ok=%v", ep, ok)
	}
	ep, ok = PickEndpoint(p, "anthropic-messages")
	if !ok || ep.ID != "anthropic" || ep.BaseURL != "https://api.shu.cool" {
		t.Fatalf("anthropic: %+v", ep)
	}
	ep, ok = PickEndpoint(p, ProtocolOpenAIResponses, ProtocolOpenAIChatCompletions)
	if !ok || ep.ID != "openai-responses" {
		t.Fatalf("pref responses: %+v", ep)
	}
	ep, ok = PickEndpoint(p, "nope")
	if ok {
		t.Fatalf("expected miss, got %+v", ep)
	}
	ep, ok = PickEndpointByID(p, "openai-chat")
	if !ok || NormalizeProtocol(ep.Protocol) != ProtocolOpenAIChatCompletions {
		t.Fatalf("%+v", ep)
	}
}

func TestNormalizeProtocolAliases(t *testing.T) {
	cases := map[string]string{
		"openai":                  ProtocolOpenAIChatCompletions,
		"OpenAI-Completions":      ProtocolOpenAIChatCompletions,
		"openai-chat-completions": ProtocolOpenAIChatCompletions,
		"responses":               ProtocolOpenAIResponses,
		"anthropic":               ProtocolAnthropicMessages,
		"ANTHROPIC-MESSAGES":      ProtocolAnthropicMessages,
	}
	for in, want := range cases {
		if got := NormalizeProtocol(in); got != want {
			t.Fatalf("%q → %q want %q", in, got, want)
		}
	}
}

func TestResolveCopyEndpoint(t *testing.T) {
	p := Provider{
		Endpoints: []Endpoint{
			{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: "openai-responses"},
			{ID: "anthropic", BaseURL: "https://api.shu.cool", Protocol: "anthropic-messages"},
		},
	}
	ep, err := ResolveCopyEndpoint(p, "", "anthropic")
	if err != nil || ep.BaseURL != "https://api.shu.cool" {
		t.Fatalf("ep=%+v err=%v", ep, err)
	}
	ep, err = ResolveCopyEndpoint(p, "openai-responses", "")
	if err != nil || ep.Protocol != ProtocolOpenAIResponses {
		t.Fatalf("ep=%+v err=%v", ep, err)
	}
	out := CopyTextEndpoint("shu", p, ep, "", false)
	if !strings.Contains(out, "base_url: https://api.shu.cool/v1") || !strings.Contains(out, "protocol: openai-responses") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "endpoint: openai-responses") {
		t.Fatalf("%s", out)
	}
}

func TestResolveProviderForProtocols(t *testing.T) {
	f := &File{
		Active: "shu",
		Providers: map[string]Provider{
			"shu": {
				KeyEnv: "SHU_API_KEY",
				Endpoints: []Endpoint{
					{ID: "anthropic", BaseURL: "https://api.shu.cool", Protocol: "anthropic-messages"},
					{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: "openai-responses"},
				},
			},
			"other": {
				KeyEnv:   "OTHER",
				BaseURL:  "https://other.example/v1",
				Protocol: "openai-responses",
			},
		},
	}
	name, _, ep, ok, msg := ResolveProviderForProtocols(f, "shu", ProtocolAnthropicMessages)
	if !ok || name != "shu" || ep.ID != "anthropic" {
		t.Fatalf("name=%s ep=%+v ok=%v msg=%s", name, ep, ok, msg)
	}
	// Without hint, anthropic → only shu
	name, _, _, ok, msg = ResolveProviderForProtocols(f, "", ProtocolAnthropicMessages)
	if !ok || name != "shu" {
		t.Fatalf("name=%s ok=%v msg=%s", name, ok, msg)
	}
	// Without hint, openai-responses → multiple
	_, _, _, ok, msg = ResolveProviderForProtocols(f, "", ProtocolOpenAIResponses)
	if ok || !strings.Contains(msg, "multiple") {
		t.Fatalf("ok=%v msg=%s", ok, msg)
	}
}
