// Package login implements `magic login`: resolve a provider skeleton, fetch its
// live model list (with fallbacks), let the user pick, then write providers.yaml
// and the local plaintext secret. Design: docs/provider-onboarding.md (D16).
package login

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"magic/internal/catalog"
	"magic/internal/modelfile"
	"magic/internal/provider"
	"magic/internal/secrets"
)

// ProviderSource is the provider skeleton used to start a login.
type ProviderSource struct {
	Name        string
	KeyEnv      string
	Endpoints   []catalog.Endpoint
	Pick        []string
	FromBuiltin bool // not present in the user's providers.yaml
}

// Plan is a resolved login awaiting selection / confirmation.
type Plan struct {
	Provider  string
	KeyEnv    string
	Endpoints []catalog.Endpoint
	APIKey    string
	// Models are the candidate models to choose from (live or fallback).
	Models []provider.RemoteModel
	// Source records where Models came from: live | builtin-pick | registry | explicit.
	Source string
	// Notes are user-facing fallback / merge explanations.
	Notes []string
	// Selected is the final pick list.
	Selected []string
}

// Options controls Resolve.
type Options struct {
	MagicHome string
	Provider  string
	APIKey    string
	// Pick, when non-empty, is an explicit non-interactive selection.
	Pick   []string
	NoLive bool
	Client *http.Client
}

// LookupProvider builds a provider skeleton from the merged catalog
// (builtin template overlaid by the user file, user wins).
func LookupProvider(magicHome, name string) (ProviderSource, bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ProviderSource{}, false, fmt.Errorf("provider name required")
	}
	merged, err := provider.LoadCatalog(magicHome)
	if err != nil {
		return ProviderSource{}, false, err
	}
	p, ok := merged.Providers[name]
	if !ok {
		return ProviderSource{}, false, nil
	}
	user, err := catalog.Load(filepath.Join(magicHome, "providers.yaml"))
	if err != nil {
		return ProviderSource{}, false, err
	}
	_, inUser := user.Providers[name]
	keyEnv := strings.TrimSpace(p.KeyEnv)
	if keyEnv == "" {
		keyEnv = name
	}
	return ProviderSource{
		Name:        name,
		KeyEnv:      keyEnv,
		Endpoints:   catalog.NormalizeEndpoints(p),
		Pick:        append([]string{}, p.Pick...),
		FromBuiltin: !inUser,
	}, true, nil
}

// Resolve builds a Plan: resolve skeleton, obtain a key, and gather candidate
// models from the live endpoint with documented fallbacks.
func Resolve(opts Options) (Plan, error) {
	src, ok, err := LookupProvider(opts.MagicHome, opts.Provider)
	if err != nil {
		return Plan{}, err
	}
	if !ok {
		return Plan{}, fmt.Errorf("provider %q not found in the builtin template or providers.yaml", opts.Provider)
	}
	plan := Plan{
		Provider:  src.Name,
		KeyEnv:    src.KeyEnv,
		Endpoints: src.Endpoints,
		APIKey:    strings.TrimSpace(opts.APIKey),
	}
	if plan.APIKey == "" {
		if v, ok, err := secrets.GetProviderSecret(filepath.Join(opts.MagicHome, "secrets"), src.KeyEnv); err == nil && ok {
			plan.APIKey = v
			plan.Notes = append(plan.Notes, fmt.Sprintf("using stored key for %s", src.KeyEnv))
		}
	}
	if len(opts.Pick) > 0 {
		plan.Source = "explicit"
		plan.Selected = append([]string{}, opts.Pick...)
		return plan, nil
	}

	if !opts.NoLive && plan.APIKey != "" {
		if models, err := fetchFromEndpoints(opts.Client, src.Endpoints, plan.APIKey); err == nil {
			plan.Models = models
			plan.Source = "live"
			return plan, nil
		} else {
			plan.Notes = append(plan.Notes, fmt.Sprintf("live /models unavailable (%v) — using fallback", err))
		}
	}
	if len(src.Pick) > 0 {
		plan.Models = idsToRemote(src.Pick)
		plan.Source = "builtin-pick"
		plan.Notes = append(plan.Notes, "using default models from the builtin template")
		return plan, nil
	}
	if reg := registryModels(opts.MagicHome, src.Name); len(reg) > 0 {
		plan.Models = reg
		plan.Source = "registry"
		plan.Notes = append(plan.Notes, "using registry/models.json (run: magic registry sync --source models.dev)")
		return plan, nil
	}
	plan.Source = "manual"
	plan.Notes = append(plan.Notes, "no model list available — enter model ids manually")
	return plan, nil
}

// DefaultSelection is the preselection offered in the picker: the intersection
// of the candidate models and the template's default pick.
func DefaultSelection(models []provider.RemoteModel, builtinPick []string) []string {
	if len(models) == 0 || len(builtinPick) == 0 {
		return nil
	}
	have := map[string]bool{}
	for _, m := range models {
		have[m.ID] = true
	}
	var out []string
	for _, id := range builtinPick {
		if have[id] {
			out = append(out, id)
		}
	}
	return out
}

// Apply writes the selected pick into providers.yaml and stores the key.
// User-owned fields (endpoints, key_env, models, note) win; pick is replaced
// with the selected list. Returns the providers.yaml path.
func Apply(magicHome string, plan Plan, selected []string) (string, error) {
	selected = cleanIDs(selected)
	if len(selected) == 0 {
		return "", fmt.Errorf("no models selected")
	}
	path := filepath.Join(magicHome, "providers.yaml")
	user, err := catalog.Load(path)
	if err != nil {
		return "", err
	}
	if user.Providers == nil {
		user.Providers = map[string]catalog.Provider{}
	}
	entry := MergeEntry(user.Providers[plan.Provider], plan, selected)
	user.Providers[plan.Provider] = entry
	if err := catalog.Save(path, user); err != nil {
		return "", err
	}
	if strings.TrimSpace(plan.APIKey) != "" {
		if err := secrets.SetProviderSecret(filepath.Join(magicHome, "secrets"), plan.KeyEnv, plan.APIKey); err != nil {
			return "", err
		}
	}
	return path, nil
}

// MergeEntry overlays the login result onto the user's existing provider entry.
// The user's endpoints/key_env/models are kept when present; pick is replaced.
func MergeEntry(existing catalog.Provider, plan Plan, selected []string) catalog.Provider {
	out := existing
	if strings.TrimSpace(out.KeyEnv) == "" {
		out.KeyEnv = plan.KeyEnv
	}
	if len(out.Endpoints) == 0 && strings.TrimSpace(out.BaseURL) == "" {
		out.Endpoints = plan.Endpoints
	}
	out.Pick = cleanIDs(selected)
	return out
}

func fetchFromEndpoints(client *http.Client, eps []catalog.Endpoint, apiKey string) ([]provider.RemoteModel, error) {
	order := orderEndpoints(eps)
	var lastErr error
	for _, ep := range order {
		models, err := provider.FetchModels(client, ep, apiKey)
		if err == nil {
			return models, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("provider has no endpoints")
	}
	return nil, lastErr
}

// orderEndpoints tries OpenAI-protocol endpoints first (most model-list APIs),
// then Anthropic. Endpoints without a protocol keep their original order.
func orderEndpoints(eps []catalog.Endpoint) []catalog.Endpoint {
	rank := func(proto string) int {
		switch catalog.NormalizeProtocol(proto) {
		case catalog.ProtocolOpenAIChatCompletions, catalog.ProtocolOpenAIResponses:
			return 0
		case catalog.ProtocolAnthropicMessages:
			return 1
		default:
			return 0
		}
	}
	out := append([]catalog.Endpoint{}, eps...)
	sort.SliceStable(out, func(i, j int) bool { return rank(out[i].Protocol) < rank(out[j].Protocol) })
	return out
}

func registryModels(magicHome, providerName string) []provider.RemoteModel {
	c, err := modelfile.LoadCatalog(magicHome)
	if err != nil || c == nil {
		return nil
	}
	q := strings.ToLower(strings.TrimSpace(providerName))
	var out []provider.RemoteModel
	for _, m := range c.Models {
		if strings.ToLower(strings.TrimSpace(m.Provider)) == q || strings.ToLower(strings.TrimSpace(m.ModelsDevProvider)) == q {
			out = append(out, provider.RemoteModel{ID: m.ID, Name: m.Name})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func idsToRemote(ids []string) []provider.RemoteModel {
	out := make([]provider.RemoteModel, 0, len(ids))
	for _, id := range cleanIDs(ids) {
		out = append(out, provider.RemoteModel{ID: id})
	}
	return out
}

func cleanIDs(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
