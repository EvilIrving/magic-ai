package provider

import "strings"

// Harness identifiers for provider switch / status / backup / test.
const (
	HarnessClaude    = "claude"
	HarnessCodex     = "codex"
	HarnessGrok      = "grok"
	HarnessPi        = "pi"
	HarnessOpenCode  = "opencode"
	HarnessWorkBuddy = "workbuddy"
	HarnessKimiCode  = "kimi-code"
	HarnessVibe      = "vibe"
	HarnessHermes    = "hermes"
	HarnessOmp       = "omp"
	HarnessDroid     = "droid"
	HarnessQoder     = "qoder"
	HarnessMmx       = "mmx"
)

// BackupLimit matches prompts-can scripts (keep latest 3).
const BackupLimit = 3

// ClaudeModelEnvKeys maps role → settings.json env key (claudes.sh MODEL_ENV_KEYS + DeepSeek extras).
var ClaudeModelEnvKeys = map[string]string{
	"haiku":     "ANTHROPIC_DEFAULT_HAIKU_MODEL",
	"sonnet":    "ANTHROPIC_DEFAULT_SONNET_MODEL",
	"opus":      "ANTHROPIC_DEFAULT_OPUS_MODEL",
	"default":   "ANTHROPIC_MODEL",
	"reasoning": "ANTHROPIC_REASONING_MODEL",
	"subagent":  "CLAUDE_CODE_SUBAGENT_MODEL",
}

// ClaudeManagedEnvKeys are Magic-managed Claude settings.env keys (upsert/clear only these).
// Includes CLAUDE_CODE_* extras used by DeepSeek / third-party Anthropic-compatible endpoints.
var ClaudeManagedEnvKeys = map[string]struct{}{
	"ANTHROPIC_API_KEY":               {},
	"ANTHROPIC_AUTH_TOKEN":            {},
	"ANTHROPIC_BASE_URL":              {},
	"ANTHROPIC_DEFAULT_FABLE_MODEL":   {},
	"ANTHROPIC_DEFAULT_HAIKU_MODEL":   {},
	"ANTHROPIC_DEFAULT_SONNET_MODEL":  {},
	"ANTHROPIC_DEFAULT_OPUS_MODEL":    {},
	"ANTHROPIC_MODEL":                 {},
	"ANTHROPIC_REASONING_MODEL":       {},
	"CLAUDE_CODE_SUBAGENT_MODEL":      {},
	"CLAUDE_CODE_EFFORT_LEVEL":        {},
	"CLAUDE_CODE_AUTO_COMPACT_WINDOW": {},
}

// NormalizeHarness maps aliases to harness ids.
func NormalizeHarness(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "claude", "claudes", "anthropic":
		return HarnessClaude
	case "codex", "gpts", "chatgpt":
		return HarnessCodex
	case "grok", "groks", "xai":
		return HarnessGrok
	case "pi":
		return HarnessPi
	case "opencode", "open-code":
		return HarnessOpenCode
	case "workbuddy", "codebuddy":
		return HarnessWorkBuddy
	case "kimi-code", "kimicode", "kimi":
		return HarnessKimiCode
	case "vibe", "mistral-vibe":
		return HarnessVibe
	case "hermes", "hermes-agent":
		return HarnessHermes
	case "omp", "oh-my-pi":
		return HarnessOmp
	case "droid", "factory":
		return HarnessDroid
	default:
		return strings.ToLower(strings.TrimSpace(s))
	}
}

// ProtocolToGrokBackend maps Magic protocol → Grok config.toml api_backend.
func ProtocolToGrokBackend(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "openai-responses", "responses":
		return "responses"
	case "openai-chat-completions", "openai", "chat_completions", "chat-completions":
		return "chat_completions"
	case "anthropic-messages", "anthropic", "messages":
		return "messages"
	default:
		return strings.TrimSpace(protocol)
	}
}

// ProtocolToCodexWire maps Magic protocol → Codex wire_api.
func ProtocolToCodexWire(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "openai-responses", "responses":
		return "responses"
	case "openai-chat-completions", "openai", "chat_completions", "chat-completions":
		return "chat_completions"
	default:
		return "responses"
	}
}
