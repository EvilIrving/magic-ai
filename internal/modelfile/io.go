package modelfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// WriteCatalog writes models.json atomically under magicHome/registry/.
func WriteCatalog(magicHome string, c *Catalog) error {
	if c == nil {
		return fmt.Errorf("write catalog: nil")
	}
	dir := filepath.Join(magicHome, "registry")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := CachePath(magicHome)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadCatalog reads normalized models.json.
func LoadCatalog(magicHome string) (*Catalog, error) {
	path := CachePath(magicHome)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Search returns models whose id, name, or provider contains query (case-insensitive).
func Search(c *Catalog, query string) []Model {
	q := strings.ToLower(strings.TrimSpace(query))
	if c == nil || q == "" {
		return nil
	}
	var out []Model
	for _, m := range c.Models {
		if strings.Contains(strings.ToLower(m.ID), q) ||
			strings.Contains(strings.ToLower(m.Name), q) ||
			strings.Contains(strings.ToLower(m.Provider), q) ||
			strings.Contains(strings.ToLower(m.Provider+"/"+m.ID), q) {
			out = append(out, m)
		}
	}
	return out
}

// FindExact returns a model by exact provider+id or unique exact id.
func FindExact(c *Catalog, id string) (Model, bool) {
	if c == nil {
		return Model{}, false
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return Model{}, false
	}
	// provider/id form
	if i := strings.IndexByte(id, '/'); i > 0 {
		prov, mid := id[:i], id[i+1:]
		for _, m := range c.Models {
			if m.Provider == prov && m.ID == mid {
				return m, true
			}
		}
	}
	var hits []Model
	for _, m := range c.Models {
		if m.ID == id {
			hits = append(hits, m)
		}
	}
	if len(hits) == 1 {
		return hits[0], true
	}
	if len(hits) > 1 {
		// Prefer non-openrouter if multiple exact id matches.
		for _, m := range hits {
			if m.Provider != "openrouter" {
				return m, true
			}
		}
		return hits[0], true
	}
	return Model{}, false
}

// FindFuzzy resolves a model id with: exact → strip vendor prefix → collapse .-_
// Optionally preferProvider when set.
func FindFuzzy(c *Catalog, id string, preferProvider string) (Model, bool) {
	if c == nil {
		return Model{}, false
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return Model{}, false
	}
	preferProvider = strings.TrimSpace(preferProvider)

	if m, ok := FindExact(c, id); ok {
		if preferProvider == "" || m.Provider == preferProvider {
			return m, true
		}
	}

	candidates := c.Models
	if preferProvider != "" {
		var filtered []Model
		for _, m := range candidates {
			if m.Provider == preferProvider {
				filtered = append(filtered, m)
			}
		}
		if len(filtered) > 0 {
			candidates = filtered
		}
	}

	// Strip vendor prefix: openai/gpt-4o → gpt-4o
	stripped := id
	if i := strings.LastIndexByte(id, '/'); i >= 0 && i+1 < len(id) {
		stripped = id[i+1:]
	}

	tryIDs := []string{id, stripped}
	for _, try := range tryIDs {
		for _, m := range candidates {
			if m.ID == try {
				return m, true
			}
		}
	}

	// Collapse .-_ and compare
	norm := collapseID(stripped)
	var hits []Model
	for _, m := range candidates {
		if collapseID(m.ID) == norm {
			hits = append(hits, m)
		}
	}
	if len(hits) == 1 {
		return hits[0], true
	}
	if len(hits) > 1 {
		for _, m := range hits {
			if m.Provider != "openrouter" {
				return m, true
			}
		}
		return hits[0], true
	}
	return Model{}, false
}

func collapseID(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
		// skip . - _
	}
	return b.String()
}

// FormatModel prints one normalized model for CLI show.
func FormatModel(m Model) string {
	var b strings.Builder
	fmt.Fprintf(&b, "id: %s\n", m.ID)
	fmt.Fprintf(&b, "provider: %s\n", m.Provider)
	if m.Name != "" {
		fmt.Fprintf(&b, "name: %s\n", m.Name)
	}
	if m.API != "" {
		fmt.Fprintf(&b, "api: %s\n", m.API)
	}
	if m.BaseURL != "" {
		fmt.Fprintf(&b, "baseUrl: %s\n", m.BaseURL)
	}
	fmt.Fprintf(&b, "reasoning: %v\n", m.Reasoning)
	if len(m.Input) > 0 {
		raw, _ := json.Marshal(m.Input)
		fmt.Fprintf(&b, "input: %s\n", raw)
	}
	fmt.Fprintf(&b, "contextWindow: %d\n", m.ContextWindow)
	fmt.Fprintf(&b, "maxTokens: %d\n", m.MaxTokens)
	costRaw, _ := json.Marshal(m.Cost)
	fmt.Fprintf(&b, "cost: %s\n", costRaw)
	if m.ThinkingLevelMap != nil {
		raw, _ := json.Marshal(m.ThinkingLevelMap)
		fmt.Fprintf(&b, "thinkingLevelMap: %s\n", raw)
	}
	if m.Source != "" {
		fmt.Fprintf(&b, "source: %s\n", m.Source)
	}
	if m.ToolCall != nil {
		fmt.Fprintf(&b, "toolCall: %v\n", *m.ToolCall)
	}
	if m.Unsupported {
		fmt.Fprintf(&b, "unsupported: true\n")
	}
	return b.String()
}

// FormatSearchList prints search hits.
func FormatSearchList(models []Model) string {
	if len(models) == 0 {
		return "(no matches)\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "PROVIDER\tID\tNAME\tCONTEXT\tAPI\n")
	for _, m := range models {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%d\t%s\n", m.Provider, m.ID, m.Name, m.ContextWindow, m.API)
	}
	return b.String()
}
