package provider

import (
	"fmt"
	"path/filepath"
	"strings"

	"magic/internal/catalog"
)

// Use modes after resolution. Product semantics: official | <providerName>.
// "company" is never a mode — only an alias for the library active provider.
const (
	ModeOfficial = "official"
	ModeProvider = "provider"
)

// ResolvedTarget is the outcome of resolving a CLI use/test target against providers.yaml.
type ResolvedTarget struct {
	Mode     string // ModeOfficial | ModeProvider
	Name     string // provider name when ModeProvider; empty when official
	Provider catalog.Provider
	Raw      string // original input
}

// IsCompanyAlias reports whether s is the UX alias for "active third-party provider".
func IsCompanyAlias(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "company", "relay", "gateway", "third-party", "thirdparty", "中转":
		return true
	default:
		return false
	}
}

// IsOfficialAlias reports official / native login path aliases.
func IsOfficialAlias(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "official", "native", "default-official":
		return true
	default:
		return false
	}
}

// LoadCatalog loads providers.yaml from MAGIC_HOME.
func LoadCatalog(magicHome string) (*catalog.File, error) {
	return catalog.Load(filepath.Join(magicHome, "providers.yaml"))
}

// ResolveUseTarget maps raw CLI arg → official | named provider from the library.
// company/relay/… → ActiveProviderName (MAGIC_ACTIVE_PROVIDER or providers.active).
// Never hardcodes a vendor id (e.g. shu) into the product layer.
func ResolveUseTarget(f *catalog.File, raw string) (ResolvedTarget, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ResolvedTarget{}, fmt.Errorf("provider target required: official | <providerName> (company = active third-party)")
	}
	lower := strings.ToLower(raw)

	if IsOfficialAlias(lower) {
		return ResolvedTarget{Mode: ModeOfficial, Raw: raw}, nil
	}

	name := raw
	if IsCompanyAlias(lower) {
		active := catalog.ActiveProviderName(f)
		if active == "" {
			return ResolvedTarget{}, fmt.Errorf("company is an alias for the active third-party provider — set providers.active in providers.yaml or MAGIC_ACTIVE_PROVIDER")
		}
		name = active
	} else {
		// mild aliases that still resolve via library names, not hardcoded vendor
		if a, ok := ProviderNameAliases[lower]; ok {
			name = a
		}
	}

	if f == nil || f.Providers == nil {
		return ResolvedTarget{}, fmt.Errorf("providers.yaml empty — run magic init / magic provider seed")
	}
	p, ok := f.Providers[name]
	if !ok {
		// try case-insensitive
		found := ""
		for k := range f.Providers {
			if strings.EqualFold(k, name) {
				found = k
				break
			}
		}
		if found == "" {
			return ResolvedTarget{}, fmt.Errorf("provider %q not found in providers.yaml", name)
		}
		name = found
		p = f.Providers[name]
	}
	return ResolvedTarget{Mode: ModeProvider, Name: name, Provider: p, Raw: raw}, nil
}

// ProviderNameAliases are soft name aliases → providers.yaml keys (seeded catalog).
// Prefer real provider ids; these only help old script habits.
var ProviderNameAliases = map[string]string{
	"anthropic": "official", // handled separately; kept for list docs
}

// PreferredEndpoint picks the best endpoint for a harness dialect.
func PreferredEndpoint(p catalog.Provider, harness string) (catalog.Endpoint, error) {
	var want []string
	switch NormalizeHarness(harness) {
	case HarnessClaude:
		want = []string{catalog.ProtocolAnthropicMessages, "anthropic"}
	case HarnessCodex:
		want = []string{catalog.ProtocolOpenAIResponses, catalog.ProtocolOpenAIChatCompletions, "openai"}
	case HarnessGrok:
		want = []string{catalog.ProtocolOpenAIResponses, catalog.ProtocolOpenAIChatCompletions, catalog.ProtocolAnthropicMessages}
	case HarnessPi, HarnessOpenCode, HarnessWorkBuddy:
		want = catalog.PiOpenAIProtocolPreference
	default:
		want = nil
	}
	if ep, ok := catalog.PickEndpoint(p, want...); ok {
		return ep, nil
	}
	if ep, ok := catalog.PickEndpoint(p); ok {
		return ep, nil
	}
	return catalog.Endpoint{}, fmt.Errorf("provider has no endpoints (set endpoints[] or base_url/protocol)")
}

// ApplyRequest is what a SwitchWriter receives after library resolution + secret lookup.
type ApplyRequest struct {
	Official bool
	Name     string // provider name when !Official
	Provider catalog.Provider
	Endpoint catalog.Endpoint
	APIKey   string // plaintext when harness needs inline token (Claude); else may be empty
}
