package provider

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"magic/internal/adapters/mmx"
	"magic/internal/catalog"
	"magic/internal/modelfile"
)

// MmxWriter projects the resolved selection into MiniMax Code's
// <data dir>/config.yaml ($MINIMAX_DATA_DIR | $MAVIS_DATA_DIR | ~/.minimax):
// one `custom_provider.magic-<name>` entry plus the `defaultModel` pointer that
// selects one of its models.
type MmxWriter struct{}

func (MmxWriter) Harness() string        { return HarnessMmx }
func (MmxWriter) Implemented() bool      { return true }
func (MmxWriter) SupportsOfficial() bool { return false } // MiniMax Code's first party is the MiniMax account (/login), not a file Magic may write
func (MmxWriter) BackupPrefix() string   { return "config" }
func (MmxWriter) BackupExt() string      { return ".yaml" }

// ConfigPath reuses the adapter's data dir ($MINIMAX_DATA_DIR / $MAVIS_DATA_DIR
// aware), so the projection and the skills mirror never disagree.
func (MmxWriter) ConfigPath(userHome string) string {
	return filepath.Join(mmx.New().MirrorRoot(userHome), "config.yaml")
}

// Source: upstream | MiniMax-AI/minimax-code packages/config/src/config.ts
//   (MinimaxConfig.custom_provider: Record<providerKey, CustomProviderConfig> —
//   CustomProviderConfig extends ProviderConfig with kind/enabled, and
//   ProviderConfig is api/name/models/options; ProviderOptions is apiKey /
//   baseURL / authMode / headers; ModelConfig is name/limit{context,output}/
//   reasoning/headers; root `defaultModel` is "<providerId>/<modelId>")
//   | checked 2026-11-15 | verified
//
// Source: upstream | packages/local-runtime/src/runtime/model-resolver-byok.ts
//   (planCustomProviderResolution: custom_provider[providerKey] → options.apiKey
//   + options.baseURL are required, `enabled: false` disables the provider, and
//   resolveCustomProviderApi accepts openai-completions | openai-responses |
//   openai-codex-responses, everything else being anthropic-messages)
//   | checked 2026-11-15 | verified
//
// Source: upstream | packages/local-runtime/src/config/model-key.ts
//   (a model is named "<providerId>/<modelId>", split at the FIRST slash, with a
//   custom provider's id spelled "custom_provider:<providerKey>")
//   | checked 2026-11-15 | verified
//
// Magic deviation: MiniMax Code rewrites config.yaml itself (/model, provider
// commands), so Magic edits its own entry in place through the YAML node tree —
// every other key, comment and the user's own custom providers stay. The
// default model Magic displaced is recorded in the projection state sidecar and
// put back by `none`; the model id inside a custom provider needs no escaping
// because the reference splits at the first slash.

// mmxMagicPrefix marks the custom_provider entries Magic owns.
const mmxMagicPrefix = "magic-"

// mmxDefaultModelKey is the root model pointer Magic displaces.
const mmxDefaultModelKey = "defaultModel"

func (w MmxWriter) Files(ctx Context, req ApplyRequest) ([]FileWrite, error) {
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

	custom := yamlMappingChild(root, "custom_provider")
	dropped := 0
	for _, key := range yamlMapKeys(custom) {
		if strings.HasPrefix(key, mmxMagicPrefix) {
			yamlDeleteKey(custom, key)
			dropped++
		}
	}
	if req.Official {
		if dropped == 0 && !st.touched(mmxDefaultModelKey) {
			// Nothing Magic wrote is here: do not re-serialize the user's config.
			return nil, nil
		}
		if st.touched(mmxDefaultModelKey) {
			if current, _ := yamlScalarString(root, mmxDefaultModelKey); st.stillMagic(mmxDefaultModelKey, current) {
				if prior, ok := st.prior(mmxDefaultModelKey); ok {
					yamlSetString(root, mmxDefaultModelKey, prior)
				} else {
					yamlDeleteKey(root, mmxDefaultModelKey)
				}
			}
		}
		if custom != nil && custom.Kind == yaml.MappingNode && len(custom.Content) == 0 {
			yamlDeleteKey(root, "custom_provider")
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

	picks := mmxPicks(req)
	if len(picks) == 0 {
		return nil, fmt.Errorf("provider %q has no pick (models) to project into %s", req.Name, path)
	}
	providerID := catalog.MagicPiProviderID(req.Name)
	entry, err := mmxProviderEntry(req, providerID, picks, modelfile.MetaResolver(ctx.MagicHome))
	if err != nil {
		return nil, err
	}
	if custom == nil {
		custom = yamlChildMap(root, "custom_provider")
	}
	if err := yamlSetValue(custom, providerID, entry); err != nil {
		return nil, err
	}

	ref := "custom_provider:" + providerID + "/" + picks[0]
	current, present := yamlScalarString(root, mmxDefaultModelKey)
	if !st.touched(mmxDefaultModelKey) {
		st.remember(mmxDefaultModelKey, current, present, ref)
	} else {
		st.Written[mmxDefaultModelKey] = ref
	}
	yamlSetString(root, mmxDefaultModelKey, ref)

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

// mmxProviderEntry is the custom_provider entry: connection settings under
// `options`, the documented wire protocol under `api`, and one model per pick.
func mmxProviderEntry(req ApplyRequest, providerID string, picks []string, resolve func(string) (modelfile.Model, bool)) (map[string]any, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(req.Endpoint.BaseURL), "/")
	if baseURL == "" {
		// model-resolver-byok refuses a custom provider without a base URL.
		return nil, fmt.Errorf("provider %q has no base URL to project into MiniMax Code", req.Name)
	}
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		// A custom provider is reached with options.apiKey and nothing else: there
		// is no env var reference form, so a keyless entry would never resolve.
		return nil, fmt.Errorf("provider %q has no API key to project into MiniMax Code (run: magic secrets set %s --value …)", req.Name, providerKeyEnv(req))
	}
	options := map[string]any{
		"apiKey":   apiKey,
		"baseURL":  baseURL,
		"authMode": "api-key",
	}
	models := map[string]any{}
	for _, pick := range picks {
		model := map[string]any{"name": pick}
		if meta, ok := resolve(pick); ok {
			if name := strings.TrimSpace(meta.Name); name != "" {
				model["name"] = name
			}
			// ModelLimit's two fields are both required, so a half-known limit
			// is left out and MiniMax Code falls back to its own defaults.
			if meta.ContextWindow > 0 && meta.MaxTokens > 0 {
				model["limit"] = map[string]any{
					"context": meta.ContextWindow,
					"output":  meta.MaxTokens,
				}
			}
		}
		models[pick] = model
	}
	return map[string]any{
		"name":    providerID,
		"kind":    "custom",
		"enabled": true,
		"api":     mmxAPI(req.Endpoint.Protocol),
		"options": options,
		"models":  models,
	}, nil
}

// mmxAPI maps a Magic endpoint protocol onto the `api` value MiniMax Code
// resolves a custom provider with (see the Source block above).
func mmxAPI(protocol string) string {
	switch catalog.NormalizeProtocol(protocol) {
	case catalog.ProtocolOpenAIResponses:
		return "openai-responses"
	case catalog.ProtocolOpenAIChatCompletions:
		return "openai-completions"
	default:
		return "anthropic-messages"
	}
}

func mmxPicks(req ApplyRequest) []string {
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

func (w MmxWriter) Status(ctx Context) string {
	path := w.ConfigPath(ctx.UserHome)
	text, _ := readFileString(path)
	doc, err := parseYAMLMap(text)
	var magic []string
	var def string
	if err == nil {
		root := yamlDocumentRoot(doc)
		def, _ = yamlScalarString(root, mmxDefaultModelKey)
		custom := yamlMappingChild(root, "custom_provider")
		for _, key := range yamlMapKeys(custom) {
			if strings.HasPrefix(key, mmxMagicPrefix) {
				magic = append(magic, key)
			}
		}
	}
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("===== MiniMax Code provider status =====\n")
	fmt.Fprintf(&b, "Config           : %s\n", path)
	fmt.Fprintf(&b, "defaultModel     : %s\n", orNone(def))
	fmt.Fprintf(&b, "Magic providers  : %v\n", magic)
	if f != nil {
		fmt.Fprintf(&b, "Library active   : %s\n", catalog.ActiveProviderName(f))
	}
	return b.String()
}

func (w MmxWriter) List(ctx Context) string {
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("  none           clear the Magic-managed custom_provider entry in config.yaml (MiniMax Code's first party is /login)\n")
	if f == nil {
		return b.String()
	}
	libActive := catalog.ActiveProviderName(f)
	for name, p := range f.Providers {
		if _, ok := catalog.PickEndpoint(p, mmxProtocolPreference...); !ok {
			continue
		}
		tag := ""
		if name == libActive {
			tag = " [active→company]"
		}
		fmt.Fprintf(&b, "  %-14s → custom_provider.%s%s\n", name, catalog.MagicPiProviderID(name), tag)
	}
	return b.String()
}

// mmxProtocolPreference is the endpoint order MiniMax Code is projected with:
// a custom provider can speak any of the four documented protocols.
var mmxProtocolPreference = []string{
	catalog.ProtocolAnthropicMessages,
	catalog.ProtocolOpenAIChatCompletions,
	catalog.ProtocolOpenAIResponses,
}
