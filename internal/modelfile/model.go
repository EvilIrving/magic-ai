package modelfile

import (
	"path/filepath"
)

// CacheFileName is the normalized catalog under $MAGIC_HOME/registry/.
const CacheFileName = "models.json"

// SchemaVersion is the models.json envelope version.
const SchemaVersion = 1

// Catalog is the on-disk normalized Magic model registry.
type Catalog struct {
	Version  int     `json:"version"`
	Source   string  `json:"source"`
	SyncedAt string  `json:"synced_at"`
	Models   []Model `json:"models"`
}

// Model is one Pi-shaped Magic model meta entry.
type Model struct {
	ID               string             `json:"id"`
	Name             string             `json:"name,omitempty"`
	Provider         string             `json:"provider"`
	API              string             `json:"api,omitempty"`
	BaseURL          string             `json:"baseUrl,omitempty"`
	Reasoning        bool               `json:"reasoning"`
	Input            []string           `json:"input,omitempty"`
	Cost             Cost               `json:"cost"`
	ContextWindow    int64              `json:"contextWindow"`
	MaxTokens        int64              `json:"maxTokens"`
	ThinkingLevelMap map[string]*string `json:"thinkingLevelMap,omitempty"`
	Source           string             `json:"source,omitempty"`
	ModelsDevProvider string            `json:"modelsDevProvider,omitempty"`
	ToolCall         *bool              `json:"toolCall,omitempty"`
	Unsupported      bool               `json:"unsupported,omitempty"`
}

// Cost is USD per million tokens (same as models.dev / Pi).
type Cost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

// CachePath returns $MAGIC_HOME/registry/models.json.
func CachePath(magicHome string) string {
	return filepath.Join(magicHome, "registry", CacheFileName)
}
