package modelsdev

import (
	"encoding/json"
	"fmt"
	"os"
)

// ParseCatalog unmarshals raw models.dev api.json bytes.
func ParseCatalog(raw []byte) (Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse models.dev catalog: %w", err)
	}
	if c == nil {
		c = Catalog{}
	}
	return c, nil
}

// LoadCatalog reads models.dev.json from disk.
func LoadCatalog(path string) (Catalog, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseCatalog(raw)
}
