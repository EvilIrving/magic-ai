package modelfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ResolvePickMeta looks up pickID in $MAGIC_HOME/registry/models.json via FindFuzzy.
// Missing/unreadable catalog → (zero, false); callers should use sensible defaults
// and must not fail apply.
//
// Optional future aliases.json (id → canonical id) is applied when present;
// absent or invalid aliases are ignored.
func ResolvePickMeta(magicHome, pickID string) (Model, bool) {
	pickID = strings.TrimSpace(pickID)
	if magicHome == "" || pickID == "" {
		return Model{}, false
	}
	c, err := LoadCatalog(magicHome)
	if err != nil || c == nil {
		return Model{}, false
	}
	pickID = resolveAlias(magicHome, pickID)
	return FindFuzzy(c, pickID, "")
}

// MetaResolver returns a pick→Model resolver that loads models.json once.
// Always safe to call; missing catalog yields a resolver that always returns false.
func MetaResolver(magicHome string) func(pickID string) (Model, bool) {
	c, err := LoadCatalog(magicHome)
	aliases := loadAliases(magicHome)
	return func(pickID string) (Model, bool) {
		pickID = strings.TrimSpace(pickID)
		if pickID == "" || err != nil || c == nil {
			return Model{}, false
		}
		if aliases != nil {
			if canon, ok := aliases[pickID]; ok && strings.TrimSpace(canon) != "" {
				pickID = strings.TrimSpace(canon)
			}
		}
		return FindFuzzy(c, pickID, "")
	}
}

func resolveAlias(magicHome, pickID string) string {
	aliases := loadAliases(magicHome)
	if aliases == nil {
		return pickID
	}
	if canon, ok := aliases[pickID]; ok && strings.TrimSpace(canon) != "" {
		return strings.TrimSpace(canon)
	}
	return pickID
}

// loadAliases reads optional $MAGIC_HOME/registry/aliases.json as { "alias": "canonical-id" }.
// Returns nil when missing or invalid (aliases are optional / not yet required).
func loadAliases(magicHome string) map[string]string {
	path := filepath.Join(magicHome, "registry", "aliases.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil || len(m) == 0 {
		return nil
	}
	return m
}
