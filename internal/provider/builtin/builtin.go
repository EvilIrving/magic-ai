// Package builtin holds the product's default provider template.
//
// providers.yaml in this directory is embedded into the binary and parsed on
// demand. See README.md for the schema and merge semantics. The template is
// never written to the user library on its own; callers merge it under
// $MAGIC_HOME/providers.yaml and let the user file win.
package builtin

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"

	"magic/internal/catalog"
)

//go:embed providers.yaml
var providersYAML []byte

// Load parses the embedded built-in provider template.
func Load() (*catalog.File, error) {
	var f catalog.File
	if err := yaml.Unmarshal(providersYAML, &f); err != nil {
		return nil, fmt.Errorf("parse builtin providers.yaml: %w", err)
	}
	if len(f.Providers) == 0 {
		return nil, fmt.Errorf("builtin providers.yaml has no providers")
	}
	return &f, nil
}

// Providers returns the template providers keyed by id.
func Providers() (map[string]catalog.Provider, error) {
	f, err := Load()
	if err != nil {
		return nil, err
	}
	return f.Providers, nil
}

// Names returns the template provider ids.
func Names() ([]string, error) {
	provs, err := Providers()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(provs))
	for name := range provs {
		names = append(names, name)
	}
	return names, nil
}
