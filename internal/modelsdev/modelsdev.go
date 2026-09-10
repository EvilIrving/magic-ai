package modelsdev

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultAPIURL is the models.dev catalog dump.
const DefaultAPIURL = "https://models.dev/api.json"

// CacheFileName is the filename under $MAGIC_HOME/registry/.
const CacheFileName = "models.dev.json"

// CachePath returns $MAGIC_HOME/registry/models.dev.json.
func CachePath(magicHome string) string {
	return filepath.Join(magicHome, "registry", CacheFileName)
}

// SyncOptions configures Sync.
type SyncOptions struct {
	MagicHome string
	APIURL    string // empty → DefaultAPIURL
	Client    *http.Client
}

// SyncResult is metadata about a successful raw download.
type SyncResult struct {
	Path       string
	Bytes      int
	Source     string
	SyncedAt   string
	ETag       string
	StatusCode int
}

// Sync GETs models.dev api.json and writes it verbatim under registry/.
// Raw download only; callers run modelfile.Normalize afterward (see docs/model-registry-from-pi.md).
func Sync(opts SyncOptions) (*SyncResult, error) {
	url := opts.APIURL
	if url == "" {
		url = DefaultAPIURL
	}
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "magic-registry/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("models.dev fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("models.dev HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("models.dev returned empty body")
	}
	path := CachePath(opts.MagicHome)
	if err := writeAtomic(path, raw); err != nil {
		return nil, err
	}
	return &SyncResult{
		Path:       path,
		Bytes:      len(raw),
		Source:     url,
		SyncedAt:   time.Now().UTC().Format(time.RFC3339),
		ETag:       resp.Header.Get("ETag"),
		StatusCode: resp.StatusCode,
	}, nil
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
