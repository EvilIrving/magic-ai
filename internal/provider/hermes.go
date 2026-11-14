package provider

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"magic/internal/adapters/hermes"
	"magic/internal/catalog"
)

// HermesWriter projects the resolved selection into Hermes Agent's
// ~/.hermes/config.yaml ($HERMES_HOME): the `model:` block that the agent reads
// for its main route (provider / default / base_url / api_key / api_mode).
type HermesWriter struct{}

func (HermesWriter) Harness() string        { return HarnessHermes }
func (HermesWriter) Implemented() bool      { return true }
func (HermesWriter) SupportsOfficial() bool { return false } // Hermes' first party is `hermes auth` (Nous Portal OAuth), not a file Magic may write
func (HermesWriter) BackupPrefix() string   { return "config" }
func (HermesWriter) BackupExt() string      { return ".yaml" }

// ConfigPath reuses the adapter's root (it honors HERMES_HOME).
func (HermesWriter) ConfigPath(userHome string) string {
	return filepath.Join(hermes.New().MirrorRoot(userHome), "config.yaml")
}

// Source: upstream | NousResearch/hermes-agent cli-config.yaml.example
//   (`model:` block: default / provider / api_key / base_url) | checked 2026-11-14 | verified
//
// Source: vendor-doc | NousResearch/hermes-agent website/docs/user-guide/configuring-models.md
//   | checked 2026-11-14 | verified
//   The dashboard writes `model: {provider, default, base_url, api_mode}` into
//   ~/.hermes/config.yaml, and a brand-new install ships `model: ""` until the
//   first setup upgrades it to that mapping.
//
// Source: vendor-doc | website/docs/user-guide/configuration.md §model
//   | checked 2026-11-14 | verified
//   For a custom OpenAI-compatible endpoint the main route is
//   `provider: custom` + `base_url` + `api_key`; api_mode is one of
//   chat_completions | codex_responses | anthropic_messages. A named entry in
//   the top-level `providers:` dict is valid for auxiliary/fallback routes but
//   NOT for `model.provider`, which is why Magic writes the model block.
//
// Magic deviation: `provider: custom` is a shared value, so the Magic-managed
// route is not derivable from the file alone. The projection state sidecar
// records what model.* held before, and `none` restores it — otherwise clearing
// would either strand a relay base_url or delete a value the user set.

const (
	hermesModelKey    = "model"
	hermesProviderKey = "model.provider"
	hermesDefaultKey  = "model.default"
	hermesBaseURLKey  = "model.base_url"
	hermesAPIKeyKey   = "model.api_key"
	hermesAPIModeKey  = "model.api_mode"
)

// hermesManagedKeys are the model fields Magic writes and later restores.
var hermesManagedKeys = []string{
	hermesProviderKey, hermesDefaultKey, hermesBaseURLKey, hermesAPIKeyKey, hermesAPIModeKey,
}

// hermesModelField is one field Magic writes into the model block. Files walks
// a slice, never a map: the emitted key order has to be the same on every call,
// or plan computes bytes apply never writes.
type hermesModelField struct {
	field string
	value string
}

func (w HermesWriter) Files(ctx Context, req ApplyRequest) ([]FileWrite, error) {
	path := w.ConfigPath(ctx.UserHome)
	statePath := ProjectionStatePath(filepath.Dir(path))
	raw, err := readTarget(ctx, path)
	if err != nil {
		return nil, err
	}
	st, err := loadProjectionState(ctx, statePath)
	if err != nil {
		return nil, err
	}
	doc, err := parseYAMLMap(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	root := yamlDocumentRoot(doc)

	if req.Official {
		if !clearHermesModel(root, st) {
			// Nothing Magic wrote is here: do not re-serialize the user's file.
			return nil, nil
		}
		out, err := encodeYAML(doc)
		if err != nil {
			return nil, err
		}
		return []FileWrite{
			{Path: path, Data: out, Mode: 0o600, Role: RoleProviderBlock},
			clearFile(statePath),
		}, nil
	}

	pick := hermesPick(req)
	if pick == "" {
		return nil, fmt.Errorf("provider %q has no pick (models) to project into %s", req.Name, path)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(req.Endpoint.BaseURL), "/")
	apiKey := strings.TrimSpace(req.APIKey)
	apiMode := hermesAPIMode(req.Endpoint.Protocol)

	// The shipped sentinel is `model: ""`; remember it so clearing puts the
	// file back to a state Hermes itself knows how to upgrade. A sentinel already
	// on record keeps winning: remembering the per-field values now would record
	// Magic's own model block as the user's, and change the sidecar on every run.
	sentinel := st.touched(hermesModelKey)
	if !sentinel {
		if block := yamlMappingChild(root, hermesModelKey); block == nil || block.Kind != yaml.MappingNode {
			current, ok := yamlScalarString(root, hermesModelKey)
			st.remember(hermesModelKey, current, ok, "")
			sentinel = true
		}
	}
	model := yamlChildMap(root, hermesModelKey)
	written := []hermesModelField{
		{"provider", "custom"},
		{"default", pick},
		{"api_mode", apiMode},
	}
	if baseURL != "" {
		written = append(written, hermesModelField{"base_url", baseURL})
	}
	if apiKey != "" {
		written = append(written, hermesModelField{"api_key", apiKey})
	}
	if !sentinel {
		for _, f := range written {
			key := hermesModelKey + "." + f.field
			current, present := yamlScalarString(model, f.field)
			st.remember(key, current, present, f.value)
		}
	}
	for _, f := range written {
		yamlSetString(model, f.field, f.value)
	}

	out, err := encodeYAML(doc)
	if err != nil {
		return nil, err
	}
	stateWrite, err := st.file(statePath)
	if err != nil {
		return nil, err
	}
	return []FileWrite{
		{Path: path, Data: out, Mode: 0o600, Role: RoleProviderBlock},
		stateWrite,
	}, nil
}

// clearHermesModel restores the model block Magic displaced: each documented
// field goes back to its recorded value, or goes away when it was absent.
// A `model: ""` sentinel recorded before the first projection wins outright.
// It reports whether it changed anything — a file Magic never wrote is left
// byte-for-byte alone.
func clearHermesModel(root *yaml.Node, st projectionState) bool {
	if sentinel, ok := st.prior(hermesModelKey); ok {
		yamlSetString(root, hermesModelKey, sentinel)
		return true
	}
	model := yamlMappingChild(root, hermesModelKey)
	if model == nil || model.Kind != yaml.MappingNode {
		return false
	}
	changed := false
	for _, key := range hermesManagedKeys {
		field := strings.TrimPrefix(key, hermesModelKey+".")
		current, _ := yamlScalarString(model, field)
		if !st.touched(key) || !st.stillMagic(key, current) {
			continue
		}
		changed = true
		if prior, ok := st.prior(key); ok {
			yamlSetString(model, field, prior)
			continue
		}
		yamlDeleteKey(model, field)
	}
	if len(model.Content) == 0 {
		yamlDeleteKey(root, hermesModelKey)
	}
	return changed
}

func hermesPick(req ApplyRequest) string {
	for _, pick := range req.Provider.Pick {
		if p := strings.TrimSpace(pick); p != "" {
			return p
		}
	}
	return strings.TrimSpace(req.Provider.Models["default"])
}

// hermesAPIMode maps a Magic endpoint protocol onto Hermes' api_mode value
// (the three values configuration.md documents).
func hermesAPIMode(protocol string) string {
	switch catalog.NormalizeProtocol(protocol) {
	case catalog.ProtocolAnthropicMessages:
		return "anthropic_messages"
	case catalog.ProtocolOpenAIResponses:
		return "codex_responses"
	default:
		return "chat_completions"
	}
}

func (w HermesWriter) Status(ctx Context) string {
	path := w.ConfigPath(ctx.UserHome)
	text, _ := readFileString(path)
	doc, err := parseYAMLMap(text)
	var provider, model, baseURL, apiMode string
	if err == nil {
		if block := yamlMappingChild(yamlDocumentRoot(doc), hermesModelKey); block != nil && block.Kind == yaml.MappingNode {
			provider, _ = yamlScalarString(block, "provider")
			model, _ = yamlScalarString(block, "default")
			baseURL, _ = yamlScalarString(block, "base_url")
			apiMode, _ = yamlScalarString(block, "api_mode")
		}
	}
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("===== Hermes Agent provider status =====\n")
	fmt.Fprintf(&b, "Config           : %s\n", path)
	fmt.Fprintf(&b, "model.provider   : %s\n", orNone(provider))
	fmt.Fprintf(&b, "model.default    : %s\n", orNone(model))
	fmt.Fprintf(&b, "model.base_url   : %s\n", orNone(baseURL))
	fmt.Fprintf(&b, "model.api_mode   : %s\n", orNone(apiMode))
	if f != nil {
		fmt.Fprintf(&b, "Library active   : %s\n", catalog.ActiveProviderName(f))
	}
	return b.String()
}

func (w HermesWriter) List(ctx Context) string {
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("  none           restore the model: block Magic displaced (Hermes' first party is `hermes auth`)\n")
	if f == nil {
		return b.String()
	}
	libActive := catalog.ActiveProviderName(f)
	for name, p := range f.Providers {
		if _, ok := catalog.PickEndpoint(p, hermesProtocolPreference...); !ok {
			continue
		}
		tag := ""
		if name == libActive {
			tag = " [active→company]"
		}
		fmt.Fprintf(&b, "  %-14s → model: {provider: custom, base_url: …}%s\n", name, tag)
	}
	return b.String()
}

// hermesProtocolPreference is the endpoint order Hermes is projected with.
var hermesProtocolPreference = []string{
	catalog.ProtocolOpenAIChatCompletions,
	catalog.ProtocolOpenAIResponses,
	catalog.ProtocolAnthropicMessages,
}
