package openrouter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncSearchShow(t *testing.T) {
	payload := map[string]any{
		"data": []any{
			map[string]any{
				"id":             "z-ai/glm-4.5",
				"name":           "GLM 4.5",
				"context_length": 131072,
				"pricing":        map[string]any{"prompt": "0.0001", "completion": "0.0002"},
				"architecture": map[string]any{
					"modality":          "text->text",
					"input_modalities":  []string{"text"},
					"output_modalities": []string{"text"},
				},
			},
			map[string]any{
				"id":             "openai/gpt-4o",
				"name":           "GPT-4o",
				"context_length": 128000,
			},
		},
	}
	body, _ := json.Marshal(payload)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method=%s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	mh := t.TempDir()
	cache, err := Sync(SyncOptions{MagicHome: mh, APIURL: srv.URL, Client: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if len(cache.Models) != 2 {
		t.Fatalf("models=%d", len(cache.Models))
	}
	path := CachePath(mh)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadCache(mh)
	if err != nil {
		t.Fatal(err)
	}
	hits := Search(loaded, "glm")
	if len(hits) != 1 || hits[0].ID != "z-ai/glm-4.5" {
		t.Fatalf("search=%v", hits)
	}
	m, ok := Find(loaded, "openai/gpt-4o")
	if !ok || m.Name != "GPT-4o" {
		t.Fatalf("find=%v ok=%v", m, ok)
	}
	out := FormatModel(m)
	if !strings.Contains(out, "id: openai/gpt-4o") {
		t.Fatalf("%s", out)
	}
	list := FormatSearchList(hits)
	if !strings.Contains(list, "z-ai/glm-4.5") {
		t.Fatalf("%s", list)
	}
	// ensure parent dir created
	if filepath.Base(filepath.Dir(path)) != "registry" {
		t.Fatalf("path=%s", path)
	}
}
