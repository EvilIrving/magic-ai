package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"magic/internal/adapters/qoder"
	"magic/internal/catalog"
	"magic/internal/modelfile"
)

// QoderWriter projects the resolved selection into Qoder CLI's
// ~/.qoder/settings.json ($QODER_CONFIG_DIR): one entry in the user-level
// `providers` map plus the `model.name` pointer that selects it.
type QoderWriter struct{}

func (QoderWriter) Harness() string        { return HarnessQoder }
func (QoderWriter) Implemented() bool      { return true }
func (QoderWriter) SupportsOfficial() bool { return false } // Qoder's first party is the Qoder account model list, not a file Magic may write
func (QoderWriter) BackupPrefix() string   { return "settings" }
func (QoderWriter) BackupExt() string      { return ".json" }

// ConfigPath is Qoder CLI's user settings.json. The adapter's root (~/.qoder)
// is the default; QODER_CONFIG_DIR moves it, so the projection follows the CLI.
func (QoderWriter) ConfigPath(userHome string) string {
	if dir := strings.TrimSpace(os.Getenv("QODER_CONFIG_DIR")); dir != "" {
		return filepath.Join(filepath.Clean(dir), "settings.json")
	}
	return filepath.Join(qoder.New().MirrorRoot(userHome), "settings.json")
}

// Source: vendor-doc | docs.qoder.com/cli/settings-reference | checked 2026-11-15 | verified
//   User level settings live in ~/.qoder/settings.json (project level
//   <project>/.qoder/settings.json, local <project>/.qoder/settings.local.json),
//   and QODER_CONFIG_DIR moves the user configuration directory (default ~/.qoder).
//
// Source: upstream | @qoder-ai/qodercli 1.1.89 bundle/qodercli.js (settings
//   schema: `providers` — label "Custom Model Providers", mergeStrategy
//   shallow_merge, additionalProperties.type = object with properties type
//   (enum openai-compatible | alibaba | aliyun-bailian), protocol (enum openai |
//   openai-responses | anthropic, default openai), authType (bearer | api-key),
//   baseUrl, apiKey, displayName, model, models[] {model, displayName,
//   contextWindow, maxOutputTokens, maxTokensField, capabilities}; top-level
//   `model` = {name, reasoningEffort, contextWindow, preferences, …})
//   | checked 2026-11-15 | verified
//   The same file's custom-provider loader requires: a provider id matching
//   /^[A-Za-z0-9][A-Za-z0-9._-]*$/ and not "qoder", a valid baseUrl, a non-empty
//   apiKey, and a default model that is one of the provider's models; a model is
//   referenced elsewhere as "<providerId>/<modelId>".
//
// Magic deviation: Magic's provider id is magic-<providerName>, so `none` can
// drop exactly the entries Magic wrote and keep the user's own providers. The
// model Magic selects is recorded in the projection state sidecar, because
// model.name is a value the user (or Qoder's /model picker) owns.

// qoderMagicPrefix marks the providers entries Magic owns.
const qoderMagicPrefix = "magic-"

// qoderModelNameKey is the settings path Magic displaces when it selects a model.
const qoderModelNameKey = "model.name"

func (w QoderWriter) Files(ctx Context, req ApplyRequest) ([]FileWrite, error) {
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
	root := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &root); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
	}

	providers, providersIsMap := root["providers"].(map[string]any)
	dropped := 0
	for id := range providers {
		if strings.HasPrefix(id, qoderMagicPrefix) {
			delete(providers, id)
			dropped++
		}
	}
	if req.Official {
		if dropped == 0 && !st.touched(qoderModelNameKey) {
			// Nothing Magic wrote is here: do not re-serialize the user's settings.
			return nil, nil
		}
		if st.touched(qoderModelNameKey) {
			if current, present := qoderModelName(root); st.stillMagic(qoderModelNameKey, current) {
				if prior, ok := st.prior(qoderModelNameKey); ok {
					qoderSetModelName(root, prior)
				} else if present {
					qoderDropModelName(root)
				}
			}
		}
		if providersIsMap && len(providers) == 0 {
			delete(root, "providers")
		}
		out, err := w.encode(root)
		if err != nil {
			return nil, err
		}
		return []FileWrite{
			{Path: path, Data: out, Mode: 0o600, Role: RoleProviderBlock},
			clearFile(statePath),
		}, nil
	}

	picks := qoderPicks(req)
	if len(picks) == 0 {
		return nil, fmt.Errorf("provider %q has no pick (models) to project into %s", req.Name, path)
	}
	providerID := catalog.MagicPiProviderID(req.Name)
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		// Qoder requires a non-empty apiKey on a custom provider and has no env
		// var reference form, so there is nothing honest to write here.
		return nil, fmt.Errorf("provider %q has no API key to project into %s (run: magic secrets set %s --value …)", req.Name, path, providerKeyEnv(req))
	}
	entry := qoderProviderEntry(req, providerID, picks, apiKey, modelfile.MetaResolver(ctx.MagicHome))
	if providers == nil {
		providers = map[string]any{}
		root["providers"] = providers
	}
	providers[providerID] = entry

	ref := providerID + "/" + picks[0]
	current, present := qoderModelName(root)
	if !st.touched(qoderModelNameKey) {
		st.remember(qoderModelNameKey, current, present, ref)
	} else {
		st.Written[qoderModelNameKey] = ref
	}
	qoderSetModelName(root, ref)

	out, err := w.encode(root)
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

// encode renders settings.json the way Qoder's own writer does (two-space
// indent, trailing newline). Encoding a map sorts the keys, so the projection is
// byte-stable across calls.
func (QoderWriter) encode(root map[string]any) ([]byte, error) {
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// qoderProviderEntry is one `providers.<id>` object: the connection settings
// plus the model list Qoder shows in its picker.
func qoderProviderEntry(req ApplyRequest, providerID string, picks []string, apiKey string, resolve func(string) (modelfile.Model, bool)) map[string]any {
	entry := map[string]any{
		"type":        "openai-compatible",
		"protocol":    qoderProtocol(req.Endpoint.Protocol),
		"displayName": providerID,
		"apiKey":      apiKey,
		"model":       picks[0],
	}
	if base := strings.TrimRight(strings.TrimSpace(req.Endpoint.BaseURL), "/"); base != "" {
		entry["baseUrl"] = base
	}
	models := make([]any, 0, len(picks))
	for _, pick := range picks {
		model := map[string]any{
			"model":       pick,
			"displayName": pick,
		}
		if meta, ok := resolve(pick); ok {
			if name := strings.TrimSpace(meta.Name); name != "" {
				model["displayName"] = name
			}
			if meta.ContextWindow > 0 {
				model["contextWindow"] = meta.ContextWindow
			}
			if meta.MaxTokens > 0 {
				model["maxOutputTokens"] = meta.MaxTokens
			}
		}
		models = append(models, model)
	}
	entry["models"] = models
	return entry
}

// qoderProtocol maps a Magic endpoint protocol onto Qoder's `protocol` value,
// which picks the wire format for a custom provider.
func qoderProtocol(protocol string) string {
	switch catalog.NormalizeProtocol(protocol) {
	case catalog.ProtocolAnthropicMessages:
		return "anthropic"
	case catalog.ProtocolOpenAIResponses:
		return "openai-responses"
	default:
		return "openai"
	}
}

// qoderModelName reads settings.model.name.
func qoderModelName(root map[string]any) (string, bool) {
	block, ok := root["model"].(map[string]any)
	if !ok {
		return "", false
	}
	name, ok := block["name"].(string)
	return name, ok
}

// qoderSetModelName writes settings.model.name, keeping the rest of the block.
func qoderSetModelName(root map[string]any, name string) {
	block, ok := root["model"].(map[string]any)
	if !ok {
		block = map[string]any{}
		root["model"] = block
	}
	block["name"] = name
}

// qoderDropModelName removes settings.model.name, and the block with it when
// nothing else is left in it.
func qoderDropModelName(root map[string]any) {
	block, ok := root["model"].(map[string]any)
	if !ok {
		return
	}
	delete(block, "name")
	if len(block) == 0 {
		delete(root, "model")
	}
}

func qoderPicks(req ApplyRequest) []string {
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

func (w QoderWriter) Status(ctx Context) string {
	path := w.ConfigPath(ctx.UserHome)
	text, _ := readFileString(path)
	root := map[string]any{}
	_ = json.Unmarshal([]byte(text), &root)
	var magic []string
	if providers, ok := root["providers"].(map[string]any); ok {
		for id := range providers {
			if strings.HasPrefix(id, qoderMagicPrefix) {
				magic = append(magic, id)
			}
		}
	}
	sort.Strings(magic)
	active, _ := qoderModelName(root)
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("===== Qoder provider status =====\n")
	fmt.Fprintf(&b, "Settings         : %s\n", path)
	fmt.Fprintf(&b, "model.name       : %s\n", orNone(active))
	fmt.Fprintf(&b, "Magic providers  : %v\n", magic)
	if f != nil {
		fmt.Fprintf(&b, "Library active   : %s\n", catalog.ActiveProviderName(f))
	}
	return b.String()
}

func (w QoderWriter) List(ctx Context) string {
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("  none           clear Magic-managed providers in settings.json (Qoder's first party is the Qoder account)\n")
	if f == nil {
		return b.String()
	}
	libActive := catalog.ActiveProviderName(f)
	for name, p := range f.Providers {
		if _, ok := catalog.PickEndpoint(p, qoderProtocolPreference...); !ok {
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

// qoderProtocolPreference is the endpoint order Qoder is projected with: a
// custom provider can speak any of the three documented protocols.
var qoderProtocolPreference = []string{
	catalog.ProtocolOpenAIChatCompletions,
	catalog.ProtocolOpenAIResponses,
	catalog.ProtocolAnthropicMessages,
}
