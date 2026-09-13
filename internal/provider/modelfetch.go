package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"magic/internal/catalog"
)

// RemoteModel is one entry from a provider's model-list endpoint.
type RemoteModel struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// ModelsURLFor returns the model-list URL and auth mode for an endpoint.
// auth is "bearer" or "x-api-key".
func ModelsURLFor(ep catalog.Endpoint) (url, auth string) {
	return modelsURL(ep)
}

// FetchModels GETs the provider's model list. The response shape is parsed
// leniently (OpenAI data[], Anthropic data[] with display_name, bare arrays).
func FetchModels(client *http.Client, ep catalog.Endpoint, apiKey string) ([]RemoteModel, error) {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	url, auth := modelsURL(ep)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "magic-login/1.0")
	switch auth {
	case "x-api-key":
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	default:
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200] + "…"
		}
		return nil, fmt.Errorf("GET %s: HTTP %d %s", url, resp.StatusCode, snippet)
	}
	models, err := ParseModelsResponse(body)
	if err != nil {
		return nil, fmt.Errorf("parse models from %s: %w", url, err)
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("GET %s: no models in response", url)
	}
	return models, nil
}

// ParseModelsResponse extracts model ids from a provider model-list body.
// Accepted shapes: {"data":[…]}, {"models":[…]}, or a top-level array.
// Entries may be objects (id/name/display_name/model) or bare strings.
func ParseModelsResponse(body []byte) ([]RemoteModel, error) {
	var probe any
	if err := json.Unmarshal(body, &probe); err != nil {
		return nil, err
	}
	items := modelItems(probe)
	seen := map[string]bool{}
	var out []RemoteModel
	for _, item := range items {
		id, name := modelEntry(item)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, RemoteModel{ID: id, Name: name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func modelItems(probe any) []any {
	switch t := probe.(type) {
	case []any:
		return t
	case map[string]any:
		for _, key := range []string{"data", "models", "items"} {
			if arr, ok := t[key].([]any); ok {
				return arr
			}
		}
	}
	return nil
}

func modelEntry(item any) (id, name string) {
	switch t := item.(type) {
	case string:
		return strings.TrimSpace(t), ""
	case map[string]any:
		id = firstString(t, "id", "model", "name", "slug")
		name = firstString(t, "display_name", "displayName", "name")
		if name == id {
			name = ""
		}
		return strings.TrimSpace(id), strings.TrimSpace(name)
	}
	return "", ""
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
