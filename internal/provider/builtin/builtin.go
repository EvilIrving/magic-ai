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
	"sort"
	"strings"

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

// Names returns the template provider ids (sorted).
func Names() ([]string, error) {
	provs, err := Providers()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(provs))
	for name := range provs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// Merge returns effective = merge(template, user) with the user file winning.
// Merge happens in memory; nothing is written to disk. Semantics follow
// docs/apply-design.md and builtin/README.md: scalar fields use the user value,
// endpoints[] / pick replace the template list whole when the user defines any,
// and models merge per role with the user winning.
func Merge(user *catalog.File) (*catalog.File, error) {
	tpl, err := Load()
	if err != nil {
		return nil, err
	}
	out := &catalog.File{Providers: map[string]catalog.Provider{}}
	if user != nil {
		out.Active = user.Active
		out.ModelsMeta = user.ModelsMeta
	}
	for name, p := range tpl.Providers {
		out.Providers[name] = p
	}
	if user != nil {
		for name, p := range user.Providers {
			out.Providers[name] = MergeProvider(out.Providers[name], p)
		}
	}
	return out, nil
}

// MergeProvider overlays a user provider onto the template entry (user wins).
func MergeProvider(base, user catalog.Provider) catalog.Provider {
	out := base
	if s := strings.TrimSpace(user.Name); s != "" {
		out.Name = s
	}
	if s := strings.TrimSpace(user.KeyEnv); s != "" {
		out.KeyEnv = s
	}
	if s := strings.TrimSpace(user.Note); s != "" {
		out.Note = s
	}
	switch {
	case len(user.Endpoints) > 0:
		// User-defined endpoints replace the template list whole.
		out.Endpoints = user.Endpoints
		out.BaseURL, out.Protocol = "", ""
	case strings.TrimSpace(user.BaseURL) != "" || strings.TrimSpace(user.Protocol) != "":
		// Legacy top-level endpoint: drop template endpoints so it is synthesized.
		out.Endpoints = nil
		out.BaseURL = user.BaseURL
		out.Protocol = user.Protocol
	}
	if len(user.Pick) > 0 {
		out.Pick = user.Pick
	}
	if len(user.Models) > 0 {
		m := map[string]string{}
		for k, v := range out.Models {
			m[k] = v
		}
		for k, v := range user.Models {
			m[k] = v
		}
		out.Models = m
	}
	return out
}

// AddSelection returns a copy of user with the named template providers added.
// Providers already present in user are left untouched (user wins). Unknown
// names are returned in missing. Nothing is written to disk.
func AddSelection(user *catalog.File, names []string) (out *catalog.File, added, skipped, missing []string, err error) {
	tpl, err := Load()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if user == nil {
		user = &catalog.File{}
	}
	out = &catalog.File{Active: user.Active, ModelsMeta: user.ModelsMeta, Providers: map[string]catalog.Provider{}}
	for k, v := range user.Providers {
		out.Providers[k] = v
	}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		p, ok := tpl.Providers[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		if _, exists := out.Providers[name]; exists {
			skipped = append(skipped, name)
			continue
		}
		out.Providers[name] = p
		added = append(added, name)
	}
	return out, added, skipped, missing, nil
}
