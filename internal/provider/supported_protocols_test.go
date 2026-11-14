package provider

import (
	"testing"

	"magic/internal/catalog"
)

// SupportedProtocols is the same table PreferredEndpoint picks from: Claude wants
// anthropic-messages first, Codex prefers responses, and a harness with no dialect
// writer accepts whatever the provider has.
func TestSupportedProtocols(t *testing.T) {
	cases := []struct {
		harness string
		first   string
	}{
		{HarnessClaude, catalog.ProtocolAnthropicMessages},
		{HarnessCodex, catalog.ProtocolOpenAIResponses},
		{HarnessPi, catalog.ProtocolOpenAIResponses},
	}
	for _, c := range cases {
		got := SupportedProtocols(c.harness)
		if len(got) == 0 || got[0] != c.first {
			t.Errorf("SupportedProtocols(%q) = %v, want %q first", c.harness, got, c.first)
		}
	}
	if got := SupportedProtocols("no-such-harness"); got != nil {
		t.Errorf("SupportedProtocols(unknown) = %v, want nil", got)
	}
}

// The endpoint PreferredEndpoint returns for Claude matches the first protocol
// SupportedProtocols declares; a provider without one falls back to its first
// endpoint, which is exactly the mismatch doctor reports.
func TestPreferredEndpointFollowsSupportedProtocols(t *testing.T) {
	p := catalog.Provider{Endpoints: []catalog.Endpoint{
		{ID: "anthropic", BaseURL: "https://api.example.com/anthropic", Protocol: catalog.ProtocolAnthropicMessages},
		{ID: "chat", BaseURL: "https://api.example.com/v1", Protocol: catalog.ProtocolOpenAIChatCompletions},
	}}
	ep, err := PreferredEndpoint(p, HarnessClaude)
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.ProtocolsMatch(ep.Protocol, SupportedProtocols(HarnessClaude)[0]) {
		t.Fatalf("PreferredEndpoint picked %q, want the anthropic endpoint", ep.Protocol)
	}

	chatOnly := catalog.Provider{Endpoints: p.Endpoints[1:]}
	ep, err = PreferredEndpoint(chatOnly, HarnessClaude)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.ProtocolsMatch(ep.Protocol, SupportedProtocols(HarnessClaude)[0]) {
		t.Fatalf("PreferredEndpoint claims Claude can project %q", ep.Protocol)
	}
}
