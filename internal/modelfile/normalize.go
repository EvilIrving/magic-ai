package modelfile

import (
	"fmt"
	"sort"
	"time"

	"magic/internal/modelsdev"
	"magic/internal/openrouter"
)

const defaultLimit int64 = 4096

// NormalizeOptions configures Normalize.
type NormalizeOptions struct {
	// ModelsDev is the parsed models.dev catalog (required for primary source).
	ModelsDev modelsdev.Catalog
	// OpenRouter is optional slim cache for enrichment (--source all).
	OpenRouter *openrouter.Cache
	// SyncedAt overrides the catalog timestamp (empty → now UTC RFC3339).
	SyncedAt string
	// Source label for the envelope (default models.dev or models.dev+openrouter).
	Source string
}

// Normalize converts models.dev (+ optional OpenRouter) into a Magic Catalog.
// models.dev wins on (provider, id) conflict. Includes all models; tool_call is
// recorded in ToolCall when present (Pi skips tool_call!=true; Magic prefers include-all).
func Normalize(opts NormalizeOptions) (*Catalog, error) {
	if opts.ModelsDev == nil && opts.OpenRouter == nil {
		return nil, fmt.Errorf("normalize: no models.dev or openrouter input")
	}
	syncedAt := opts.SyncedAt
	if syncedAt == "" {
		syncedAt = time.Now().UTC().Format(time.RFC3339)
	}
	source := opts.Source
	if source == "" {
		if opts.OpenRouter != nil && opts.ModelsDev != nil {
			source = "models.dev+openrouter"
		} else if opts.ModelsDev != nil {
			source = "models.dev"
		} else {
			source = "openrouter"
		}
	}

	seen := map[string]struct{}{}
	var models []Model

	if opts.ModelsDev != nil {
		// Stable provider order for deterministic output.
		providers := make([]string, 0, len(opts.ModelsDev))
		for pid := range opts.ModelsDev {
			providers = append(providers, pid)
		}
		sort.Strings(providers)

		for _, pid := range providers {
			p := opts.ModelsDev[pid]
			api, baseURL, unsupported := modelsdev.ResolveProviderDefaults(pid, p)

			modelIDs := make([]string, 0, len(p.Models))
			for mid := range p.Models {
				modelIDs = append(modelIDs, mid)
			}
			sort.Strings(modelIDs)

			for _, mid := range modelIDs {
				m := p.Models[mid]
				id := mid
				if m.ID != "" {
					id = m.ID
				}
				key := pid + "/" + id
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}

				name := m.Name
				if name == "" {
					name = id
				}
				ctx := defaultLimit
				maxTok := defaultLimit
				if m.Limit != nil {
					if m.Limit.Context > 0 {
						ctx = m.Limit.Context
					}
					if m.Limit.Output > 0 {
						maxTok = m.Limit.Output
					}
				}
				cost := Cost{}
				if m.Cost != nil {
					cost = Cost{
						Input:      m.Cost.Input,
						Output:     m.Cost.Output,
						CacheRead:  m.Cost.CacheRead,
						CacheWrite: m.Cost.CacheWrite,
					}
				}
				input := inputModalities(m)
				tlm := modelsdev.EffortThinkingLevelMap(m.ReasoningOptions)

				entry := Model{
					ID:                id,
					Name:              name,
					Provider:          pid,
					API:               api,
					BaseURL:           baseURL,
					Reasoning:         m.Reasoning,
					Input:             input,
					Cost:              cost,
					ContextWindow:     ctx,
					MaxTokens:         maxTok,
					ThinkingLevelMap:  tlm,
					Source:            "models.dev",
					ModelsDevProvider: pid,
					ToolCall:          m.ToolCall,
					Unsupported:       unsupported,
				}
				models = append(models, entry)
			}
		}
	}

	// Optional OpenRouter enrichment: add OR entries not already present as openrouter/<id>.
	if opts.OpenRouter != nil {
		for _, om := range opts.OpenRouter.Models {
			if om.ID == "" {
				continue
			}
			key := "openrouter/" + om.ID
			if _, ok := seen[key]; ok {
				continue
			}
			// Also skip if models.dev already has the same provider/id for non-OR
			// (e.g. openai/gpt-4o) — OR ids are usually org/model; treat as distinct
			// unless exact openrouter key exists. models.dev wins only on same key.
			seen[key] = struct{}{}
			input := []string{"text"}
			if om.Architecture != nil {
				for _, mod := range om.Architecture.InputModalities {
					if mod == "image" {
						input = []string{"text", "image"}
						break
					}
				}
				if len(om.Architecture.InputModalities) == 0 && om.Architecture.Modality != "" {
					if containsImageModality(om.Architecture.Modality) {
						input = []string{"text", "image"}
					}
				}
			}
			ctx := defaultLimit
			if om.ContextLength > 0 {
				ctx = om.ContextLength
			}
			models = append(models, Model{
				ID:            om.ID,
				Name:          om.Name,
				Provider:      "openrouter",
				API:           "openai-completions",
				BaseURL:       "https://openrouter.ai/api/v1",
				Reasoning:     false,
				Input:         input,
				Cost:          costFromOpenRouterPricing(om.Pricing),
				ContextWindow: ctx,
				MaxTokens:     defaultLimit,
				Source:        "openrouter",
			})
		}
	}

	return &Catalog{
		Version:  SchemaVersion,
		Source:   source,
		SyncedAt: syncedAt,
		Models:   models,
	}, nil
}

func inputModalities(m modelsdev.Model) []string {
	hasImage := false
	if m.Modalities != nil {
		for _, in := range m.Modalities.Input {
			if in == "image" {
				hasImage = true
				break
			}
		}
	}
	if !hasImage && m.Attachment {
		hasImage = true
	}
	if hasImage {
		return []string{"text", "image"}
	}
	return []string{"text"}
}

func containsImageModality(modality string) bool {
	// OpenRouter modality strings look like "text+image->text"
	for i := 0; i+5 <= len(modality); i++ {
		if modality[i:i+5] == "image" {
			return true
		}
	}
	return false
}

// costFromOpenRouterPricing converts OpenRouter per-token string prices ×1e6 → $/MTok.
// Missing/unparseable fields stay 0.
func costFromOpenRouterPricing(pricing map[string]any) Cost {
	if pricing == nil {
		return Cost{}
	}
	return Cost{
		Input:      openRouterPerTokenToMTok(pricing["prompt"]),
		Output:     openRouterPerTokenToMTok(pricing["completion"]),
		CacheRead:  openRouterPerTokenToMTok(pricing["input_cache_read"]),
		CacheWrite: openRouterPerTokenToMTok(pricing["input_cache_write"]),
	}
}

func openRouterPerTokenToMTok(v any) float64 {
	var f float64
	switch t := v.(type) {
	case float64:
		f = t
	case string:
		if t == "" {
			return 0
		}
		var parsed float64
		if _, err := fmt.Sscanf(t, "%f", &parsed); err != nil {
			return 0
		}
		f = parsed
	default:
		return 0
	}
	return f * 1e6
}
