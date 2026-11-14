package catalog

import (
	"encoding/json"
	"fmt"
	"strings"

	"magic/internal/modelfile"
)

// ClaudeModelEnvKeys maps role → settings.json env key.
var ClaudeModelEnvKeys = map[string]string{
	"haiku":     "ANTHROPIC_DEFAULT_HAIKU_MODEL",
	"sonnet":    "ANTHROPIC_DEFAULT_SONNET_MODEL",
	"opus":      "ANTHROPIC_DEFAULT_OPUS_MODEL",
	"default":   "ANTHROPIC_MODEL",
	"reasoning": "ANTHROPIC_REASONING_MODEL",
	"subagent":  "CLAUDE_CODE_SUBAGENT_MODEL",
}

// ClaudeModelRoleEnv upserts ANTHROPIC_* / CLAUDE_CODE_SUBAGENT_MODEL role envs from
// Provider.Models and/or first pick as default when Models["default"] is unset.
// When subagent is unset but haiku is set, subagent inherits haiku (DeepSeek pattern).
func ClaudeModelRoleEnv(p Provider) map[string]string {
	out := map[string]string{}
	for role, model := range p.Models {
		if key, ok := ClaudeModelEnvKeys[role]; ok && model != "" {
			out[key] = model
		}
	}
	if _, hasDefault := out["ANTHROPIC_MODEL"]; !hasDefault {
		if len(p.Pick) > 0 {
			if id := strings.TrimSpace(p.Pick[0]); id != "" {
				out["ANTHROPIC_MODEL"] = id
			}
		}
	}
	if _, hasSub := out["CLAUDE_CODE_SUBAGENT_MODEL"]; !hasSub {
		if haiku := out["ANTHROPIC_DEFAULT_HAIKU_MODEL"]; haiku != "" {
			out["CLAUDE_CODE_SUBAGENT_MODEL"] = haiku
		}
	}
	return out
}

// PiOpenAIProtocolPreference is the order used when projecting to Pi.
var PiOpenAIProtocolPreference = []string{
	ProtocolOpenAIResponses,
	ProtocolOpenAIChatCompletions,
	"openai",
}

// PiProviderEntry is one providers{} value in Pi models.json.
type PiProviderEntry struct {
	BaseURL string         `json:"baseUrl"`
	API     string         `json:"api"`
	APIKey  string         `json:"apiKey"` // real provider key (decisions.md D3)
	Models  []PiModelEntry `json:"models"`
}

// PiCost is USD per million tokens (Pi-shaped).
type PiCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

// PiModelEntry is a model row under a Pi provider. Optional fields use omitempty
// so missing registry meta keeps the JSON minimal (id + name).
type PiModelEntry struct {
	ID               string             `json:"id"`
	Name             string             `json:"name,omitempty"`
	Reasoning        bool               `json:"reasoning,omitempty"`
	Input            []string           `json:"input,omitempty"`
	Cost             *PiCost            `json:"cost,omitempty"`
	ContextWindow    int64              `json:"contextWindow,omitempty"`
	MaxTokens        int64              `json:"maxTokens,omitempty"`
	ThinkingLevelMap map[string]*string `json:"thinkingLevelMap,omitempty"`
}

// PickMetaResolver looks up registry meta for a pick id. Nil or false → defaults.
type PickMetaResolver func(pickID string) (modelfile.Model, bool)

// BuildPiProviderEntry builds a Magic-managed Pi provider block.
// apiKey is the real secret value (decisions.md D3); callers without a key
// should skip projection instead of writing a placeholder.
// resolvePickMeta is optional; when set, each pick is enriched from registry/models.json.
func BuildPiProviderEntry(ep Endpoint, apiKey string, pick []string, resolvePickMeta PickMetaResolver) PiProviderEntry {
	models := make([]PiModelEntry, 0, len(pick))
	for _, id := range pick {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		models = append(models, BuildPiModelEntry(id, resolvePickMeta))
	}
	return PiProviderEntry{
		BaseURL: ep.BaseURL,
		API:     ProtocolToPiAPI(ep.Protocol),
		APIKey:  apiKey,
		Models:  models,
	}
}

// BuildPiModelEntry projects one pick into a Pi model row, enriching from meta when found.
func BuildPiModelEntry(pickID string, resolvePickMeta PickMetaResolver) PiModelEntry {
	entry := PiModelEntry{ID: pickID, Name: pickID}
	if resolvePickMeta == nil {
		return entry
	}
	meta, ok := resolvePickMeta(pickID)
	if !ok {
		return entry
	}
	if meta.Name != "" {
		entry.Name = meta.Name
	}
	entry.Reasoning = meta.Reasoning
	if len(meta.Input) > 0 {
		entry.Input = append([]string{}, meta.Input...)
	}
	if meta.ContextWindow > 0 {
		entry.ContextWindow = meta.ContextWindow
	}
	if meta.MaxTokens > 0 {
		entry.MaxTokens = meta.MaxTokens
	}
	if meta.ThinkingLevelMap != nil {
		entry.ThinkingLevelMap = meta.ThinkingLevelMap
	}
	if meta.Cost.Input != 0 || meta.Cost.Output != 0 || meta.Cost.CacheRead != 0 || meta.Cost.CacheWrite != 0 {
		entry.Cost = &PiCost{
			Input:      meta.Cost.Input,
			Output:     meta.Cost.Output,
			CacheRead:  meta.Cost.CacheRead,
			CacheWrite: meta.Cost.CacheWrite,
		}
	}
	return entry
}

// MagicPiProviderID returns the stable Magic-managed id in models.json providers map.
func MagicPiProviderID(providerName string) string {
	return "magic-" + providerName
}

// MergePiModelsJSON merges providerID → entry into existing models.json (providers map).
// Other providers are preserved. Returns pretty JSON + newline.
func MergePiModelsJSON(existing []byte, providerID string, entry PiProviderEntry) ([]byte, error) {
	root := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &root); err != nil {
			return nil, fmt.Errorf("parse models.json: %w", err)
		}
	}
	providers := map[string]any{}
	if raw, ok := root["providers"]; ok && raw != nil {
		if m, ok := raw.(map[string]any); ok {
			providers = m
		}
	}
	// round-trip entry through JSON for a plain map
	b, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}
	var entryMap map[string]any
	if err := json.Unmarshal(b, &entryMap); err != nil {
		return nil, err
	}
	providers[providerID] = entryMap
	root["providers"] = providers
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}
