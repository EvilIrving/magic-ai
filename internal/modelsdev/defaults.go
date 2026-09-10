package modelsdev

import "strings"

// ProviderDefaults maps a models.dev provider id → Magic wire dialect + default base URL.
// models.dev often puts the upstream URL in provider.api, but omits the wire dialect;
// this table supplies both when known. Documented in docs/model-registry-from-pi.md.
//
// Wire dialect values match Pi/Magic:
//
//	openai-responses | openai-completions | anthropic-messages
type ProviderDefaults struct {
	API     string // wire dialect
	BaseURL string
}

// KnownProviders is the Magic-sized hardcoded map (not Pi's full quirk matrix).
var KnownProviders = map[string]ProviderDefaults{
	"openai":           {API: "openai-responses", BaseURL: "https://api.openai.com/v1"},
	"anthropic":        {API: "anthropic-messages", BaseURL: "https://api.anthropic.com"},
	"deepseek":         {API: "openai-completions", BaseURL: "https://api.deepseek.com"},
	"xai":              {API: "openai-responses", BaseURL: "https://api.x.ai/v1"},
	"groq":             {API: "openai-completions", BaseURL: "https://api.groq.com/openai/v1"},
	"openrouter":       {API: "openai-completions", BaseURL: "https://openrouter.ai/api/v1"},
	"moonshotai":       {API: "openai-completions", BaseURL: "https://api.moonshot.ai/v1"},
	"moonshotai-cn":    {API: "openai-completions", BaseURL: "https://api.moonshot.cn/v1"},
	"cerebras":         {API: "openai-completions", BaseURL: "https://api.cerebras.ai/v1"},
	"mistral":          {API: "openai-completions", BaseURL: "https://api.mistral.ai"},
	"minimax":          {API: "anthropic-messages", BaseURL: "https://api.minimax.io/anthropic"},
	"minimax-cn":       {API: "anthropic-messages", BaseURL: "https://api.minimaxi.com/anthropic"},
	"together":         {API: "openai-completions", BaseURL: "https://api.together.xyz/v1"},
	"togetherai":       {API: "openai-completions", BaseURL: "https://api.together.xyz/v1"},
	"fireworks-ai":     {API: "openai-completions", BaseURL: "https://api.fireworks.ai/inference/v1"},
	"huggingface":      {API: "openai-completions", BaseURL: "https://router.huggingface.co/v1"},
	"kimi-for-coding":  {API: "anthropic-messages", BaseURL: "https://api.kimi.com/coding"},
	"google":           {API: "", BaseURL: "https://generativelanguage.googleapis.com/v1beta"}, // unsupported dialect in Magic today
	"google-vertex":    {API: "", BaseURL: ""},
}

// ResolveProviderDefaults returns api dialect + baseUrl for a models.dev provider.
// Priority: KnownProviders → npm heuristic → provider.API as baseUrl only.
// Unknown/empty dialect ⇒ unsupported=true (caller may still emit metadata for search).
func ResolveProviderDefaults(providerID string, p Provider) (api, baseURL string, unsupported bool) {
	id := strings.ToLower(strings.TrimSpace(providerID))
	if d, ok := KnownProviders[id]; ok {
		api, baseURL = d.API, d.BaseURL
		// Prefer models.dev provider.api URL when present (authoritative upstream).
		if u := strings.TrimSpace(p.API); u != "" {
			baseURL = u
		}
		if api == "" {
			unsupported = true
		}
		return api, baseURL, unsupported
	}

	// npm heuristic (same idea as Pi OpenCode mapping, used for unknown providers).
	switch strings.TrimSpace(p.NPM) {
	case "@ai-sdk/openai", "@ai-sdk/xai":
		api = "openai-responses"
	case "@ai-sdk/anthropic":
		api = "anthropic-messages"
	case "@ai-sdk/openai-compatible", "@openrouter/ai-sdk-provider":
		api = "openai-completions"
	case "@ai-sdk/google", "@ai-sdk/google-vertex":
		api = ""
		unsupported = true
	default:
		if strings.Contains(p.NPM, "openai-compatible") {
			api = "openai-completions"
		} else if strings.Contains(p.NPM, "anthropic") {
			api = "anthropic-messages"
		} else if p.NPM != "" {
			// Unknown SDK — leave dialect empty.
			unsupported = true
		} else {
			unsupported = true
		}
	}

	baseURL = strings.TrimSpace(p.API)
	if api == "" {
		unsupported = true
	}
	return api, baseURL, unsupported
}
