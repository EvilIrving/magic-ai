package provider

import (
	"fmt"
	"path/filepath"

	"magic/internal/catalog"
)

// SeedDefaults merges official vendor profiles into providers.yaml when missing.
// Relays and gateways are user-defined — they do not belong in seed data.
// Does not set `active`; the user file (or MAGIC_ACTIVE_PROVIDER) chooses.
func SeedDefaults(magicHome string) (added []string, err error) {
	path := filepath.Join(magicHome, "providers.yaml")
	f, err := catalog.Load(path)
	if err != nil {
		return nil, err
	}
	if f.Providers == nil {
		f.Providers = map[string]catalog.Provider{}
	}
	for name, p := range DefaultProviderProfiles() {
		if _, exists := f.Providers[name]; exists {
			continue
		}
		f.Providers[name] = p
		added = append(added, name)
	}
	if err := catalog.Save(path, f); err != nil {
		return added, err
	}
	return added, nil
}

// DefaultProviderProfiles is official-vendor seed only (no relays).
func DefaultProviderProfiles() map[string]catalog.Provider {
	return map[string]catalog.Provider{
		"deepseek": {
			KeyEnv: "DEEPSEEK_API_KEY",
			Pick:   []string{"deepseek-flash", "deepseek-v4-pro"},
			Endpoints: []catalog.Endpoint{
				// Claude Code: anthropic-messages dialect requires /anthropic suffix.
				{ID: "anthropic", BaseURL: "https://api.deepseek.com/anthropic", Protocol: catalog.ProtocolAnthropicMessages},
				// Codex: Responses API (wire_api=responses); script default base is https://api.deepseek.com/
				{ID: "openai-responses", BaseURL: "https://api.deepseek.com/", Protocol: catalog.ProtocolOpenAIResponses},
				{ID: "openai-chat", BaseURL: "https://api.deepseek.com/v1", Protocol: catalog.ProtocolOpenAIChatCompletions},
			},
			Models: map[string]string{
				"haiku": "deepseek-flash", "sonnet": "deepseek-v4-pro",
				"opus": "deepseek-v4-pro", "default": "deepseek-v4-pro",
				"subagent": "deepseek-flash",
			},
		},
		"kimi": {
			KeyEnv: "KIMI_API_KEY",
			Pick:   []string{"kimi-k3", "kimi-k2.7-code"},
			Endpoints: []catalog.Endpoint{
				{ID: "anthropic", BaseURL: "https://api.moonshot.cn/anthropic", Protocol: catalog.ProtocolAnthropicMessages},
			},
			Models: map[string]string{
				"haiku": "kimi-k2.7-code", "sonnet": "kimi-k3", "opus": "kimi-k3",
				"default": "kimi-k3", "reasoning": "kimi-k3",
			},
		},
		"minimax": {
			KeyEnv: "MINIMAX_API_KEY",
			Pick:   []string{"MiniMax-M3", "MiniMax-M2.7"},
			Endpoints: []catalog.Endpoint{
				{ID: "anthropic", BaseURL: "https://api.minimaxi.com/anthropic", Protocol: catalog.ProtocolAnthropicMessages},
			},
			Models: map[string]string{
				"haiku": "MiniMax-M2.7", "sonnet": "MiniMax-M3",
				"opus": "MiniMax-M3", "default": "MiniMax-M3",
			},
		},
	}
}

// EnsureSeeded runs SeedDefaults and returns a short log line.
func EnsureSeeded(magicHome string) string {
	added, err := SeedDefaults(magicHome)
	if err != nil {
		return fmt.Sprintf("seed warning: %v", err)
	}
	if len(added) == 0 {
		return ""
	}
	return fmt.Sprintf("seeded providers: %v", added)
}
