package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"magic/internal/adapter"
	"magic/internal/catalog"
	"magic/internal/modelfile"
	"magic/internal/provider"
	"magic/internal/secrets"
)

// ApplyProviders projects providers.yaml into Claude Code settings and Pi models.json
// for adapters that Detect on the user home (or when --only selects them).
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

	hint := catalog.ActiveProviderName(f)

	for _, a := range adapters {
		if !a.Implemented() {
			continue
		}
		id := string(a.ID())
		if opts.Only != "" && opts.Only != id {
			continue
		}
		switch id {
		case "claude":
			if !a.Detect(opts.UserHome) && opts.Only != "claude" {
				log = append(log, "[claude] providers: skip (not detected)")
				continue
			}
			extra, err := applyClaudeProvider(f, hint, secMap, opts)
			if err != nil {
				return log, err
			}
			log = append(log, extra...)
		case "pi":
			if !a.Detect(opts.UserHome) && opts.Only != "pi" {
				log = append(log, "[pi] providers: skip (not detected)")
				continue
			}
			extra, err := applyPiProvider(f, hint, secMap, opts)
			if err != nil {
				return log, err
			}
			log = append(log, extra...)
		case "opencode", "workbuddy":
			if !a.Detect(opts.UserHome) && opts.Only != id {
				log = append(log, fmt.Sprintf("[%s] providers: skip (not detected)", id))
				continue
			}
			extra, err := applySwitchProvider(id, f, hint, opts)
			if err != nil {
				return log, err
			}
			log = append(log, extra...)
		}
	}
	return log, nil
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

func applyClaudeProvider(f *catalog.File, hint string, secMap map[string]string, opts ApplyOptions) ([]string, error) {
	var log []string
	name, p, ep, ok, msg := catalog.ResolveProviderForProtocols(f, hint, catalog.ProtocolAnthropicMessages, "anthropic")
	if !ok {
		log = append(log, fmt.Sprintf("[claude] providers: skip — %s", msg))
		return log, nil
	}
	apiKey := resolveProviderSecret(p, name, secMap)
	if apiKey == "" {
		keyPh := p.KeyEnv
		if keyPh == "" {
			keyPh = name
		}
		log = append(log, fmt.Sprintf("[claude] providers: skip — secret for %s missing (set with magic secrets set %s --value …; need MAGIC_PASSPHRASE)", name, keyPh))
		return log, nil
	}
	env, err := provider.BuildClaudeEnvFromProvider(opts.MagicHome, name, p, ep, apiKey)
	if err != nil {
		return log, fmt.Errorf("[claude] providers: %w", err)
	}
	path, _, err := catalog.WriteClaudeSettings(opts.UserHome, env, opts.DryRun)
	if err != nil {
		return log, fmt.Errorf("[claude] providers: %w", err)
	}
	action := "upsert"
	if opts.DryRun {
		action = "would upsert"
	}
	log = append(log, fmt.Sprintf("[claude] providers: %s %s (provider=%s endpoint=%s protocol=%s)", action, path, name, ep.ID, ep.Protocol))
	return log, nil
}

func applyPiProvider(f *catalog.File, hint string, secMap map[string]string, opts ApplyOptions) ([]string, error) {
	var log []string
	name, p, ep, ok, msg := catalog.ResolveProviderForProtocols(f, hint, catalog.PiOpenAIProtocolPreference...)
	if !ok {
		log = append(log, fmt.Sprintf("[pi] providers: skip — %s", msg))
		return log, nil
	}
	apiKey := resolveProviderSecret(p, name, secMap)
	if apiKey == "" {
		keyPh := p.KeyEnv
		if keyPh == "" {
			keyPh = name
		}
		log = append(log, fmt.Sprintf("[pi] providers: skip — secret for %s missing (set with magic secrets set %s --value …)", name, keyPh))
		return log, nil
	}
	pid := catalog.MagicPiProviderID(name)
	resolve := modelfile.MetaResolver(opts.MagicHome)
	entry := catalog.BuildPiProviderEntry(ep, apiKey, p.Pick, resolve)
	path, err := catalog.WritePiModels(opts.UserHome, pid, entry, opts.DryRun)
	if err != nil {
		return log, fmt.Errorf("[pi] providers: %w", err)
	}
	// Also write a small Magic sidecar for clarity (same entry).
	if !opts.DryRun {
		side := filepath.Join(opts.UserHome, ".pi", "agent", "models.magic.json")
		payload, mErr := catalog.MergePiModelsJSON(nil, pid, entry)
		if mErr == nil {
			_ = os.WriteFile(side, payload, 0o600)
		}
	}
	action := "merge"
	if opts.DryRun {
		action = "would merge"
	}
	log = append(log, fmt.Sprintf("[pi] providers: %s %s provider=%s (endpoint=%s protocol=%s api=%s apiKey=set)", action, path, pid, ep.ID, ep.Protocol, entry.API))
	return log, nil
}

func applySwitchProvider(harness string, f *catalog.File, hint string, opts ApplyOptions) ([]string, error) {
	var log []string
	target := strings.TrimSpace(hint)
	if target == "" {
		name, _, _, ok, msg := catalog.ResolveProviderForProtocols(f, "", catalog.PiOpenAIProtocolPreference...)
		if !ok {
			log = append(log, fmt.Sprintf("[%s] providers: skip — %s", harness, msg))
			return log, nil
		}
		target = name
	}
	if opts.DryRun {
		log = append(log, fmt.Sprintf("[%s] providers: would apply %s", harness, target))
		return log, nil
	}
	ctx := provider.Context{MagicHome: opts.MagicHome, UserHome: opts.UserHome}
	msg, err := provider.Use(ctx, harness, target)
	if err != nil {
		log = append(log, fmt.Sprintf("[%s] providers: skip — %v", harness, err))
		return log, nil
	}
	for _, line := range strings.Split(strings.TrimSpace(msg), "\n") {
		if line != "" {
			log = append(log, fmt.Sprintf("[%s] providers: %s", harness, line))
		}
	}
	return log, nil
}
