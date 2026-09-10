package provider

import (
	"fmt"
	"path/filepath"

	"magic/internal/catalog"
)

// SeedDefaults merges reference provider profiles into providers.yaml when missing.
// Vendor URLs (e.g. shu.cool) live here as library seed data — not in switch core types.
// Sets active to "shu" only when active is empty and shu was inserted or already present.
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
	if f.Active == "" {
		if _, ok := f.Providers["shu"]; ok {
			f.Active = "shu"
		}
	}
	if err := catalog.Save(path, f); err != nil {
		return added, err
	}
	return added, nil
}

// DefaultProviderProfiles is the library seed (claudes/gpts/groks/pi goldens → yaml).
func DefaultProviderProfiles() map[string]catalog.Provider {
	return map[string]catalog.Provider{
		"shu": {
			KeyEnv: "SHU_API_KEY",
			Pick:   []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"},
			Endpoints: []catalog.Endpoint{
				{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: catalog.ProtocolOpenAIResponses},
				{ID: "openai-chat", BaseURL: "https://api.shu.cool/v1", Protocol: catalog.ProtocolOpenAIChatCompletions},
				{ID: "anthropic", BaseURL: "https://api.shu.cool", Protocol: catalog.ProtocolAnthropicMessages},
			},
			Models: map[string]string{
				"haiku": "gpt-5.6-luna", "sonnet": "gpt-5.6-terra",
				"opus": "gpt-5.6-sol", "default": "gpt-5.6-sol",
			},
		},
		"shu-deepseek": {
			KeyEnv: "SHU_DEEPSEEK_API_KEY",
			Pick:   []string{"deepseek-v4-pro", "deepseek-v4-flash-vision-exp"},
			Endpoints: []catalog.Endpoint{
				{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: catalog.ProtocolOpenAIResponses},
				{ID: "openai-chat", BaseURL: "https://api.shu.cool/v1", Protocol: catalog.ProtocolOpenAIChatCompletions},
				{ID: "anthropic", BaseURL: "https://api.shu.cool", Protocol: catalog.ProtocolAnthropicMessages},
			},
			Models: map[string]string{
				"haiku": "deepseek-v4-flash-vision-exp", "sonnet": "deepseek-v4-pro",
				"opus": "deepseek-v4-pro", "default": "deepseek-v4-pro",
			},
		},
		"shu-grok": {
			KeyEnv: "SHU_GROK_API_KEY",
			Pick:   []string{"grok-4.5", "grok-4.6", "grok-composer-2.5-fast"},
			Endpoints: []catalog.Endpoint{
				{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: catalog.ProtocolOpenAIResponses},
				{ID: "anthropic", BaseURL: "https://api.shu.cool", Protocol: catalog.ProtocolAnthropicMessages},
			},
			Models: map[string]string{
				"haiku": "grok-composer-2.5-fast", "sonnet": "grok-4.5",
				"opus": "grok-4.5", "default": "grok-4.5",
			},
		},
		"deepseek": {
			KeyEnv: "DEEPSEEK_API_KEY",
			// Codex picks: pro / flash / vision-exp (matches DeepSeek setup script).
			Pick: []string{"deepseek-v4-pro", "deepseek-v4-flash", "deepseek-v4-flash-vision-exp"},
			Endpoints: []catalog.Endpoint{
				// Claude Code: anthropic-messages dialect requires /anthropic suffix.
				{ID: "anthropic", BaseURL: "https://api.deepseek.com/anthropic", Protocol: catalog.ProtocolAnthropicMessages},
				// Codex: Responses API (wire_api=responses); script default base is https://api.deepseek.com/
				{ID: "openai-responses", BaseURL: "https://api.deepseek.com/", Protocol: catalog.ProtocolOpenAIResponses},
				{ID: "openai-chat", BaseURL: "https://api.deepseek.com/v1", Protocol: catalog.ProtocolOpenAIChatCompletions},
			},
			// Claude role map per DeepSeek docs: opus/sonnet/default → pro; haiku/subagent → flash.
			Models: map[string]string{
				"haiku": "deepseek-v4-flash", "sonnet": "deepseek-v4-pro",
				"opus": "deepseek-v4-pro", "default": "deepseek-v4-pro",
				"subagent": "deepseek-v4-flash",
			},
		},
		"kimi": {
			KeyEnv: "KIMI_API_KEY",
			Pick:   []string{"kimi-k2.5"},
			Endpoints: []catalog.Endpoint{
				{ID: "anthropic", BaseURL: "https://api.moonshot.cn/anthropic", Protocol: catalog.ProtocolAnthropicMessages},
			},
			Models: map[string]string{
				"haiku": "kimi-k2.5", "sonnet": "kimi-k2.5", "opus": "kimi-k2.5",
				"default": "kimi-k2.5", "reasoning": "kimi-k2.5",
			},
		},
		"minimax": {
			KeyEnv: "MINIMAX_API_KEY",
			Pick:   []string{"MiniMax-M2.7"},
			Endpoints: []catalog.Endpoint{
				{ID: "anthropic", BaseURL: "https://api.minimaxi.com/anthropic", Protocol: catalog.ProtocolAnthropicMessages},
			},
			Models: map[string]string{
				"haiku": "MiniMax-M2.7", "sonnet": "MiniMax-M2.7",
				"opus": "MiniMax-M2.7", "default": "MiniMax-M2.7",
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
