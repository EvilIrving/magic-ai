package core

import (
	"fmt"
	"strings"

	"magic/internal/adapter"
	"magic/internal/catalog"
	"magic/internal/provider"
	"magic/internal/secrets"
)

// claudeProviderProtocols is the endpoint preference apply and plan resolve
// Claude Code's provider with; provider.PreferredEndpoint uses the same order.
var claudeProviderProtocols = []string{catalog.ProtocolAnthropicMessages, "anthropic"}

// providerProjectionHarnesses are the harnesses `magic apply` projects provider
// config into.
//
// Codex and Grok have a SwitchWriter too — `magic provider use codex|grok` writes
// them — but apply does not rewrite a user's primary config.toml. That needs its
// own decision (docs/apply-design.md lists both as projection targets), so keep
// the gap explicit instead of reading it as an oversight.
// gemini/copilot/trae only have stub writers: they have nothing to project.
var providerProjectionHarnesses = map[string]bool{
	"claude": true, "pi": true, "opencode": true, "workbuddy": true,
}

// ApplyProviders projects providers.yaml into every provider harness that
// Detect on the user home (or that --only names).
//
// The projection itself is the harness's SwitchWriter: Files computes the files
// and provider.ApplyFiles writes them, the same call `magic provider use` makes.
func ApplyProviders(adapters []adapter.Adapter, opts ApplyOptions) ([]string, error) {
	var log []string
	paths := NewPaths(opts.MagicHome)
	// Effective catalog = builtin template overlaid by the user file (user wins).
	f, err := provider.LoadCatalog(opts.MagicHome)
	if err != nil {
		return nil, err
	}
	if f == nil || len(f.Providers) == 0 {
		log = append(log, "[providers] skip (no providers.yaml entries)")
		return log, nil
	}

	secMap, secErr := secrets.LoadProviderSecrets(paths.Secrets)
	if secErr != nil {
		log = append(log, fmt.Sprintf("[providers] warning: %v", secErr))
		secMap = map[string]string{}
	}

	for _, a := range adapters {
		if !a.Implemented() {
			continue
		}
		id := string(a.ID())
		if !providerProjectionHarnesses[id] {
			continue
		}
		w := provider.Writer(id)
		if w == nil || !w.Implemented() {
			continue
		}
		if opts.Only != "" && opts.Only != id {
			continue
		}
		if !a.Detect(opts.UserHome) && opts.Only != id {
			log = append(log, fmt.Sprintf("[%s] providers: skip (not detected)", id))
			continue
		}
		req, why := providerProjectionRequest(w, f, secMap)
		if why != "" {
			log = append(log, fmt.Sprintf("[%s] providers: skip — %s", id, why))
			continue
		}
		ctx := provider.Context{
			MagicHome: opts.MagicHome,
			UserHome:  opts.UserHome,
			BackupDir: opts.BackupDir,
			Catalog:   f,
		}
		if opts.DryRun {
			writes, err := w.Files(ctx, req)
			if err != nil {
				return log, fmt.Errorf("[%s] providers: %w", id, err)
			}
			log = append(log, fmt.Sprintf("[%s] providers: would write %d file(s) for provider=%s (endpoint=%s protocol=%s)",
				id, len(writes), req.Name, req.Endpoint.ID, req.Endpoint.Protocol))
			continue
		}
		msg, err := provider.ApplyFiles(ctx, w, req)
		if err != nil {
			return log, fmt.Errorf("[%s] providers: %w", id, err)
		}
		for _, line := range strings.Split(strings.TrimSpace(msg), "\n") {
			if line != "" {
				log = append(log, fmt.Sprintf("[%s] providers: %s", id, line))
			}
		}
	}
	return log, nil
}

// providerProjectionRequest resolves what apply and plan project for one
// harness: the library's active provider, the endpoint that harness dialect
// prefers, and the provider's secret. why is empty on success; otherwise it says
// why nothing is projected, and plan leaves the layer out for the same reason.
func providerProjectionRequest(w provider.SwitchWriter, f *catalog.File, secMap map[string]string) (provider.ApplyRequest, string) {
	if f == nil || len(f.Providers) == 0 {
		return provider.ApplyRequest{}, "no providers.yaml entries"
	}
	name, p, _, ok, msg := catalog.ResolveProviderForProtocols(f, catalog.ActiveProviderName(f), providerEndpointPreference(w.Harness())...)
	if !ok {
		return provider.ApplyRequest{}, msg
	}
	ep, err := provider.PreferredEndpoint(p, w.Harness())
	if err != nil {
		return provider.ApplyRequest{}, fmt.Sprintf("provider %q: %v", name, err)
	}
	apiKey := resolveProviderSecret(p, name, secMap)
	if apiKey == "" {
		keyPh := strings.TrimSpace(p.KeyEnv)
		if keyPh == "" {
			keyPh = name
		}
		return provider.ApplyRequest{}, fmt.Sprintf("secret for %s missing (set with magic secrets set %s --value …)", name, keyPh)
	}
	return provider.ApplyRequest{Name: name, Provider: p, Endpoint: ep, APIKey: apiKey}, ""
}

// providerEndpointPreference is the endpoint order a harness's provider is
// resolved with; provider.PreferredEndpoint picks from the same list.
func providerEndpointPreference(harness string) []string {
	if harness == provider.HarnessClaude {
		return claudeProviderProtocols
	}
	return catalog.PiOpenAIProtocolPreference
}

// ProviderGap is why apply projects no provider config for one harness even
// though it would: the provider has no endpoint in a protocol the harness
// speaks, no secret, or no active provider is selected.
type ProviderGap struct {
	// Provider is the provider magic tried to project; empty when providers.yaml
	// names no active provider.
	Provider string
	// ProtocolMismatch: the active provider has no endpoint in a protocol this
	// harness can take. It is the "the software answers 400 after switching"
	// case magic doctor checks, and verify reports it by name.
	ProtocolMismatch bool
	// Reason is the resolution message apply logs when it skips the layer.
	Reason string
}

// ProviderProjectionGap returns a's provider-layer gap. false means the layer
// has nothing to report: the harness is not a provider projection target, its
// writer is a stub, it is not installed (and --only does not name it), or the
// provider resolves and would be written. verify calls it so its "nothing was
// checked" message names the provider instead of the adapter list.
func ProviderProjectionGap(a adapter.Adapter, opts ApplyOptions) (ProviderGap, bool) {
	id := string(a.ID())
	if !providerProjectionHarnesses[id] {
		return ProviderGap{}, false
	}
	w := provider.Writer(id)
	if w == nil || !w.Implemented() {
		return ProviderGap{}, false
	}
	if !a.Detect(opts.UserHome) && opts.Only != id {
		return ProviderGap{}, false
	}
	lib, err := loadProjectionLayers(opts)
	if err != nil {
		return ProviderGap{Reason: err.Error()}, true
	}
	_, why := providerProjectionRequest(w, lib.providers, lib.secrets)
	if why == "" {
		return ProviderGap{}, false
	}
	gap := ProviderGap{Reason: why}
	if lib.providers != nil {
		if name := catalog.ActiveProviderName(lib.providers); name != "" {
			gap.Provider = name
			if p, ok := lib.providers.Providers[name]; ok {
				_, matched := catalog.PickEndpoint(p, providerEndpointPreference(w.Harness())...)
				gap.ProtocolMismatch = !matched
			}
		}
	}
	return gap, true
}

func resolveProviderSecret(p catalog.Provider, providerName string, secMap map[string]string) string {
	if key := strings.TrimSpace(p.KeyEnv); key != "" {
		if v := strings.TrimSpace(secMap[key]); v != "" {
			return v
		}
	}
	if v := strings.TrimSpace(secMap[providerName]); v != "" {
		return v
	}
	return ""
}
