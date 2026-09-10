package openrouter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultAPIURL = "https://openrouter.ai/api/v1/models"

// CacheFileName is the filename under $MAGIC_HOME/registry/.
const CacheFileName = "openrouter-models.json"

// Model is a slimmed OpenRouter model entry.
type Model struct {
	ID            string         `json:"id"`
	Name          string         `json:"name,omitempty"`
	ContextLength int64          `json:"context_length,omitempty"`
	Pricing       map[string]any `json:"pricing,omitempty"`
	Architecture  *Architecture  `json:"architecture,omitempty"`
}

// Architecture holds modality info.
type Architecture struct {
	Modality         string   `json:"modality,omitempty"`
	InputModalities  []string `json:"input_modalities,omitempty"`
	OutputModalities []string `json:"output_modalities,omitempty"`
}

// Cache is the on-disk registry cache.
type Cache struct {
	SyncedAt string  `json:"synced_at"`
	Source   string  `json:"source"`
	Models   []Model `json:"models"`
}

// CachePath returns $MAGIC_HOME/registry/openrouter-models.json.
func CachePath(magicHome string) string {
	return filepath.Join(magicHome, "registry", CacheFileName)
}

// SyncOptions configures Sync.
type SyncOptions struct {
	MagicHome string
	APIURL    string // empty → DefaultAPIURL
	Client    *http.Client
}

// Sync fetches OpenRouter models and writes the cache.
func Sync(opts SyncOptions) (*Cache, error) {
	url := opts.APIURL
	if url == "" {
		url = DefaultAPIURL
	}
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openrouter fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("openrouter HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	models, err := ParseAPIResponse(raw)
	if err != nil {
		return nil, err
	}
	cache := &Cache{
		SyncedAt: time.Now().UTC().Format(time.RFC3339),
		Source:   url,
		Models:   models,
	}
	if err := WriteCache(opts.MagicHome, cache); err != nil {
		return nil, err
	}
	return cache, nil
}

// ParseAPIResponse slims the OpenRouter /models JSON.
func ParseAPIResponse(raw []byte) ([]Model, error) {
	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("parse openrouter response: %w", err)
	}
	out := make([]Model, 0, len(envelope.Data))
	for _, item := range envelope.Data {
		var m struct {
			ID            string         `json:"id"`
			Name          string         `json:"name"`
			ContextLength int64          `json:"context_length"`
			Pricing       map[string]any `json:"pricing"`
			Architecture  *Architecture  `json:"architecture"`
		}
		if err := json.Unmarshal(item, &m); err != nil {
			continue
		}
		if m.ID == "" {
			continue
		}
		out = append(out, Model{
			ID:            m.ID,
			Name:          m.Name,
			ContextLength: m.ContextLength,
			Pricing:       m.Pricing,
			Architecture:  m.Architecture,
		})
	}
	return out, nil
}

// WriteCache writes cache JSON under magicHome/registry/.
func WriteCache(magicHome string, cache *Cache) error {
	dir := filepath.Join(magicHome, "registry")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
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

// LoadCache reads the on-disk cache.
func LoadCache(magicHome string) (*Cache, error) {
	path := CachePath(magicHome)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Cache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Search returns models whose id or name contains query (case-insensitive).
func Search(cache *Cache, query string) []Model {
	q := strings.ToLower(strings.TrimSpace(query))
	if cache == nil || q == "" {
		return nil
	}
	var out []Model
	for _, m := range cache.Models {
		if strings.Contains(strings.ToLower(m.ID), q) || strings.Contains(strings.ToLower(m.Name), q) {
			out = append(out, m)
		}
	}
	return out
}

// Find returns one model by exact id, or false.
func Find(cache *Cache, id string) (Model, bool) {
	if cache == nil {
		return Model{}, false
	}
	for _, m := range cache.Models {
		if m.ID == id {
			return m, true
		}
	}
	return Model{}, false
}

// FormatModel prints one model for CLI show.
func FormatModel(m Model) string {
	var b strings.Builder
	fmt.Fprintf(&b, "id: %s\n", m.ID)
	if m.Name != "" {
		fmt.Fprintf(&b, "name: %s\n", m.Name)
	}
	if m.ContextLength > 0 {
		fmt.Fprintf(&b, "context_length: %d\n", m.ContextLength)
	}
	if m.Pricing != nil {
		raw, _ := json.Marshal(m.Pricing)
		fmt.Fprintf(&b, "pricing: %s\n", raw)
	}
	if m.Architecture != nil {
		raw, _ := json.Marshal(m.Architecture)
		fmt.Fprintf(&b, "architecture: %s\n", raw)
	}
	return b.String()
}

// FormatSearchList prints search hits.
func FormatSearchList(models []Model) string {
	if len(models) == 0 {
		return "(no matches)\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "ID\tNAME\tCONTEXT\n")
	for _, m := range models {
		fmt.Fprintf(&b, "%s\t%s\t%d\n", m.ID, m.Name, m.ContextLength)
	}
	return b.String()
}
