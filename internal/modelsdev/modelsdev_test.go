package modelsdev

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncWritesRawJSON(t *testing.T) {
	payload := map[string]any{
		"anthropic": map[string]any{
			"models": map[string]any{
				"claude-test": map[string]any{
					"id":   "claude-test",
					"name": "Claude Test",
				},
			},
		},
	}
	body, _ := json.Marshal(payload)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method=%s", r.Method)
		}
		w.Header().Set("ETag", `"fixture-1"`)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	mh := t.TempDir()
	res, err := Sync(SyncOptions{MagicHome: mh, APIURL: srv.URL, Client: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if res.Bytes != len(body) {
		t.Fatalf("bytes=%d want=%d", res.Bytes, len(body))
	}
	if res.ETag != `"fixture-1"` {
		t.Fatalf("etag=%q", res.ETag)
	}
	path := CachePath(mh)
	if path != res.Path {
		t.Fatalf("path=%s res=%s", path, res.Path)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(body) {
		t.Fatalf("file content mismatch")
	}
	if filepath.Base(filepath.Dir(path)) != "registry" {
		t.Fatalf("path=%s", path)
	}
}
