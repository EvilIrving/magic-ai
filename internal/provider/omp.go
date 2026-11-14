package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"magic/internal/adapters/omp"
	"magic/internal/catalog"
	"magic/internal/modelfile"
)

// OmpWriter projects the resolved selection into Oh My Pi's model registry,
// ~/.omp/agent/models.yml (PI_CONFIG_DIR / PI_CODING_AGENT_DIR aware):
// providers.magic-<name> with the same provider/model vocabulary Pi uses.
type OmpWriter struct{}

func (OmpWriter) Harness() string        { return HarnessOmp }
func (OmpWriter) Implemented() bool      { return true }
func (OmpWriter) SupportsOfficial() bool { return false } // omp is a multi-provider aggregator
func (OmpWriter) BackupPrefix() string   { return "models" }
func (OmpWriter) BackupExt() string      { return ".yml" }

// ConfigPath reuses the adapter's agent dir (PI_CONFIG_DIR / PI_CODING_AGENT_DIR)
// and keeps Oh My Pi's own precedence: models.yml, else models.yaml, else a new
// models.yml. Writing models.yml while the user's registry is models.yaml would
// shadow their file, so the existing name wins.
func (w OmpWriter) ConfigPath(userHome string) string {
	dir := omp.New().MirrorRoot(userHome)
	yml := filepath.Join(dir, "models.yml")
	if _, err := os.Stat(yml); err == nil {
		return yml
	}
	yamlPath := filepath.Join(dir, "models.yaml")
	if _, err := os.Stat(yamlPath); err == nil {
		return yamlPath
	}
	return yml
}

// Source: upstream | can1357/oh-my-pi docs/models.md (config paths
//   ~/.omp/agent/models.yml then models.yaml; a legacy models.json is migrated
//   to models.yml, so Magic writes YAML, not JSON) | checked 2026-11-14 | verified
//
// Source: upstream | oh-my-pi packages/coding-agent/src/config/models-config-schema-bundle.ts
//   (ProviderConfigSchema: baseUrl / apiKey / api / headers / auth / models[];
//   ApiSchema: openai-completions | openai-responses | anthropic-messages | …)
//   | checked 2026-11-14 | verified
//
// Source: upstream | oh-my-pi packages/coding-agent/src/config/model-registry.ts
//   (a provider's apiKey is installed as the provider credential; `!cmd` values
//   are re-materialized, so a plain secret is valid here) | checked 2026-11-14 | verified
//
// `providers` is the only root key the registry reads, and providers are merged
// by id, so Magic's entry never replaces a provider the user declared.
//
// Magic reuses catalog.BuildPiProviderEntry: Oh My Pi is a Pi fork and keeps
// Pi's provider/model field names (baseUrl, api, apiKey, models[].id/name/…),
// so the Pi dialect projection is the same object, serialized as YAML.

// ompMagicPrefix marks the providers Magic owns in models.yml.
const ompMagicPrefix = "magic-"

func (w OmpWriter) Files(ctx Context, req ApplyRequest) ([]FileWrite, error) {
	path := w.ConfigPath(ctx.UserHome)
	raw, err := readTarget(ctx, path)
	if err != nil {
		return nil, err
	}
	doc, err := parseYAMLMap(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	root := yamlDocumentRoot(doc)

	providers := yamlMappingChild(root, "providers")
	dropped := 0
	for _, key := range yamlMapKeys(providers) {
		if strings.HasPrefix(key, ompMagicPrefix) {
			yamlDeleteKey(providers, key)
			dropped++
		}
	}
	if req.Official && dropped == 0 {
		// Nothing Magic wrote is here: do not re-serialize the user's registry.
		return nil, nil
	}
	if !req.Official {
		if providers == nil {
			providers = yamlChildMap(root, "providers")
		}
		picks := ompPicks(req)
		if len(picks) == 0 {
			return nil, fmt.Errorf("provider %q has no pick (models) to project into %s", req.Name, path)
		}
		entry := catalog.BuildPiProviderEntry(req.Endpoint, strings.TrimSpace(req.APIKey), picks, modelfile.MetaResolver(ctx.MagicHome))
		value, err := jsonShape(entry)
		if err != nil {
			return nil, err
		}
		if err := yamlSetValue(providers, catalog.MagicPiProviderID(req.Name), value); err != nil {
			return nil, err
		}
	}
	if providers != nil && len(providers.Content) == 0 {
		yamlDeleteKey(root, "providers")
	}
	out, err := encodeYAML(doc)
	if err != nil {
		return nil, err
	}
	return []FileWrite{{Path: path, Data: out, Mode: 0o600, Role: RoleModels}}, nil
}

// jsonShape converts a JSON-tagged struct into the map/slice shape yaml.v3
// renders with the same key names.
func jsonShape(value any) (any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func ompPicks(req ApplyRequest) []string {
	var picks []string
	for _, pick := range req.Provider.Pick {
		if p := strings.TrimSpace(pick); p != "" {
			picks = append(picks, p)
		}
	}
	if len(picks) == 0 {
		if def := strings.TrimSpace(req.Provider.Models["default"]); def != "" {
			picks = []string{def}
		}
	}
	return picks
}

func (w OmpWriter) Status(ctx Context) string {
	path := w.ConfigPath(ctx.UserHome)
	text, _ := readFileString(path)
	doc, err := parseYAMLMap(text)
	var magic []string
	if err == nil {
		providers := yamlMappingChild(yamlDocumentRoot(doc), "providers")
		for _, key := range yamlMapKeys(providers) {
			if strings.HasPrefix(key, ompMagicPrefix) {
				magic = append(magic, key)
			}
		}
	}
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("===== Oh My Pi provider status =====\n")
	fmt.Fprintf(&b, "Models file      : %s\n", path)
	fmt.Fprintf(&b, "Magic providers  : %v\n", magic)
	if f != nil {
		fmt.Fprintf(&b, "Library active   : %s\n", catalog.ActiveProviderName(f))
	}
	return b.String()
}

func (w OmpWriter) List(ctx Context) string {
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("  none           clear Magic-managed providers in models.yml (Oh My Pi is multi-provider)\n")
	if f == nil {
		return b.String()
	}
	libActive := catalog.ActiveProviderName(f)
	for name, p := range f.Providers {
		if _, ok := catalog.PickEndpoint(p, catalog.PiOpenAIProtocolPreference...); !ok {
			continue
		}
		tag := ""
		if name == libActive {
			tag = " [active→company]"
		}
		fmt.Fprintf(&b, "  %-14s → providers.%s%s\n", name, catalog.MagicPiProviderID(name), tag)
	}
	return b.String()
}
