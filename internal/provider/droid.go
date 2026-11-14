package provider

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"magic/internal/adapters/droid"
	"magic/internal/catalog"
	"magic/internal/modelfile"
)

// DroidWriter projects the resolved selection into Factory Droid's
// ~/.factory/settings.json customModels array: one Magic-managed model per pick.
type DroidWriter struct{}

func (DroidWriter) Harness() string        { return HarnessDroid }
func (DroidWriter) Implemented() bool      { return true }
func (DroidWriter) SupportsOfficial() bool { return true } // Factory's own models are the first-party path; official = drop Magic's custom entries
func (DroidWriter) BackupPrefix() string   { return "settings" }
func (DroidWriter) BackupExt() string      { return ".json" }

// ConfigPath reuses the adapter's root (~/.factory).
func (DroidWriter) ConfigPath(userHome string) string {
	return filepath.Join(droid.New().MirrorRoot(userHome), "settings.json")
}

// Source: vendor-doc | docs.factory.com/model-independence/byok
//   | checked 2026-11-14 | verified
//   Custom models live in ~/.factory/settings.json under `customModels`, with
//   model / displayName / baseUrl / apiKey / provider (/ maxOutputTokens /
//   maxContextLimit). `apiKey` may be a literal or `${VAR_NAME}`; an empty one
//   is invalid, and it is optional only for keyless endpoints.
//   provider vocabulary: anthropic (Anthropic Messages) | openai (OpenAI
//   Responses) | generic-chat-completion-api (OpenAI Chat Completions) |
//   bedrock-converse.
//
// Source: vendor-doc | docs.factory.com/droid-cli/settings
//   (`customModels` default []; `model` is "any available model ID") | checked 2026-11-14 | verified
//
// Source: vendor-doc | docs.factory.com/model-independence/byok (legacy
//   config.json note) | checked 2026-11-14 | verified
//   Droid copies `custom_models` from ~/.factory/config.json into settings.json
//   once and then ignores further edits there, so Magic writes settings.json only.
//
// Magic deviation: Droid has no id field on a custom model entry, so the stable
// marker is `displayName: magic-<providerName>/<pick>` — the label the model
// picker shows, and unique per entry so two picks never look alike.
// `provider use none` / `official` remove exactly the entries carrying the
// `magic-` prefix.
//
// TODO(FUTURE): selecting a Magic entry as the default is left to Droid
// (`/model`). The docs do not state how `settings.model` references a
// customModels entry, and guessing would write a default model that never
// resolves.

// droidMagicPrefix is the displayName prefix that marks Magic-owned models.
const droidMagicPrefix = "magic-"

func (w DroidWriter) Files(ctx Context, req ApplyRequest) ([]FileWrite, error) {
	path := w.ConfigPath(ctx.UserHome)
	raw, err := readTarget(ctx, path)
	if err != nil {
		return nil, err
	}
	root := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &root); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
	}
	kept := keepNonMagicDroidModels(root["customModels"])
	dropped := len(droidEntries(root["customModels"])) - len(kept)
	if req.Official && dropped == 0 {
		// Nothing Magic wrote is here: do not re-serialize the user's settings.
		return nil, nil
	}
	models := kept
	if !req.Official {
		picks := droidPicks(req)
		if len(picks) == 0 {
			return nil, fmt.Errorf("provider %q has no pick (models) to project into %s", req.Name, path)
		}
		resolve := modelfile.MetaResolver(ctx.MagicHome)
		for _, pick := range picks {
			models = append(models, buildDroidModel(req, pick, resolve))
		}
	}
	if len(models) == 0 {
		delete(root, "customModels")
	} else {
		root["customModels"] = models
	}
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return []FileWrite{{Path: path, Data: append(out, '\n'), Mode: 0o600, Role: RoleModels}}, nil
}

// droidEntries is root["customModels"] as an entry list, or nil.
func droidEntries(value any) []any {
	arr, _ := value.([]any)
	return arr
}

// keepNonMagicDroidModels drops the entries Magic owns and keeps everything else
// verbatim, including entries whose shape Magic does not know.
func keepNonMagicDroidModels(value any) []any {
	arr, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]any, 0, len(arr))
	for _, item := range arr {
		entry, ok := item.(map[string]any)
		if !ok {
			out = append(out, item)
			continue
		}
		if name, _ := entry["displayName"].(string); strings.HasPrefix(name, droidMagicPrefix) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func buildDroidModel(req ApplyRequest, pick string, resolve func(string) (modelfile.Model, bool)) map[string]any {
	entry := map[string]any{
		"model":       pick,
		"displayName": droidMagicPrefix + req.Name + "/" + pick,
		"provider":    droidProvider(req.Endpoint.Protocol),
	}
	if base := strings.TrimRight(strings.TrimSpace(req.Endpoint.BaseURL), "/"); base != "" {
		entry["baseUrl"] = base
	}
	// An empty apiKey is invalid for Droid; fall back to its documented
	// ${VAR} reference so the entry stays usable without Magic holding the value.
	if key := strings.TrimSpace(req.APIKey); key != "" {
		entry["apiKey"] = key
	} else {
		entry["apiKey"] = "${" + providerKeyEnv(req) + "}"
	}
	if meta, ok := resolve(pick); ok {
		if meta.MaxTokens > 0 {
			entry["maxOutputTokens"] = meta.MaxTokens
		}
		if meta.ContextWindow > 0 {
			entry["maxContextLimit"] = meta.ContextWindow
		}
	}
	return entry
}

func droidPicks(req ApplyRequest) []string {
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

// droidProvider maps a Magic endpoint protocol onto Droid's provider value,
// which selects the wire format (see the Source block above).
func droidProvider(protocol string) string {
	switch catalog.NormalizeProtocol(protocol) {
	case catalog.ProtocolAnthropicMessages:
		return "anthropic"
	case catalog.ProtocolOpenAIResponses:
		return "openai"
	default:
		return "generic-chat-completion-api"
	}
}

func (w DroidWriter) Status(ctx Context) string {
	path := w.ConfigPath(ctx.UserHome)
	raw, _ := readFileString(path)
	root := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &root)
	var magic []string
	if arr, ok := root["customModels"].([]any); ok {
		for _, item := range arr {
			entry, _ := item.(map[string]any)
			name, _ := entry["displayName"].(string)
			if strings.HasPrefix(name, droidMagicPrefix) {
				model, _ := entry["model"].(string)
				magic = append(magic, name+" ("+model+")")
			}
		}
	}
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("===== Droid (Factory) provider status =====\n")
	fmt.Fprintf(&b, "Settings         : %s\n", path)
	fmt.Fprintf(&b, "Magic models     : %v\n", magic)
	if f != nil {
		fmt.Fprintf(&b, "Library active   : %s\n", catalog.ActiveProviderName(f))
	}
	return b.String()
}

func (w DroidWriter) List(ctx Context) string {
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("  none           clear Magic-managed customModels entries (Factory models stay)\n")
	if f == nil {
		return b.String()
	}
	libActive := catalog.ActiveProviderName(f)
	for name, p := range f.Providers {
		if _, ok := catalog.PickEndpoint(p, droidProtocolPreference...); !ok {
			continue
		}
		tag := ""
		if name == libActive {
			tag = " [active→company]"
		}
		fmt.Fprintf(&b, "  %-14s → customModels displayName=%s%s\n", name, droidMagicPrefix+name, tag)
	}
	return b.String()
}

// droidProtocolPreference is the endpoint order Droid is projected with.
var droidProtocolPreference = []string{
	catalog.ProtocolOpenAIChatCompletions,
	catalog.ProtocolOpenAIResponses,
	catalog.ProtocolAnthropicMessages,
}
