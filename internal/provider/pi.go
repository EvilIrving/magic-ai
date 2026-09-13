package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"magic/internal/catalog"
	"magic/internal/modelfile"
)

// PiWriter projects providers.yaml into ~/.pi/agent/models.json (or clears Magic entry on official).
type PiWriter struct{}

func (PiWriter) Harness() string      { return HarnessPi }
func (PiWriter) Implemented() bool    { return true }
func (PiWriter) BackupPrefix() string { return "models" }
func (PiWriter) BackupExt() string    { return ".json" }
func (PiWriter) ConfigPath(userHome string) string {
	return filepath.Join(userHome, ".pi", "agent", "models.json")
}

func (w PiWriter) Apply(ctx Context, req ApplyRequest) (string, error) {
	path := w.ConfigPath(ctx.UserHome)
	var existing []byte
	if raw, err := os.ReadFile(path); err == nil {
		existing = raw
	} else if !os.IsNotExist(err) {
		return "", err
	}

	bak, err := CreateBackup(ctx.MagicHome, HarnessPi, path, "models", ".json")
	if err != nil {
		return "", err
	}

	var out []byte
	if req.Official {
		out, err = removeMagicPiProviders(existing)
		if err != nil {
			return "", err
		}
	} else {
		pid := catalog.MagicPiProviderID(req.Name)
		resolve := modelfile.MetaResolver(ctx.MagicHome)
		entry := catalog.BuildPiProviderEntry(req.Endpoint, req.Provider.KeyEnv, req.Provider.Pick, resolve)
		out, err = catalog.MergePiModelsJSON(existing, pid, entry)
		if err != nil {
			return "", err
		}
		// sidecar
		side := filepath.Join(ctx.UserHome, ".pi", "agent", "models.magic.json")
		sidePayload, _ := catalog.MergePiModelsJSON(nil, pid, entry)
		_ = atomicWrite(side, sidePayload, 0o600)
	}
	if err := atomicWrite(path, out, 0o600); err != nil {
		return "", err
	}

	// Optional probe summary under MAGIC_HOME/registry
	_ = writeProbeSummary(ctx, req)

	var b strings.Builder
	if bak != "" {
		fmt.Fprintf(&b, "Backup: %s\n", bak)
	}
	if req.Official {
		b.WriteString("Pi: removed Magic-managed providers.* entries (official).\n")
	} else {
		fmt.Fprintf(&b, "Pi: merged provider %s (endpoint=%s api=%s apiKey=$%s).\n",
			catalog.MagicPiProviderID(req.Name), req.Endpoint.ID,
			catalog.ProtocolToPiAPI(req.Endpoint.Protocol),
			strings.TrimPrefix(req.Provider.KeyEnv, "$"))
	}
	return b.String(), nil
}

func (w PiWriter) Status(ctx Context) string {
	path := w.ConfigPath(ctx.UserHome)
	raw, err := os.ReadFile(path)
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("===== Pi provider status =====\n")
	fmt.Fprintf(&b, "Models file      : %s\n", path)
	if err != nil {
		b.WriteString("State            : missing\n")
		return b.String()
	}
	var root map[string]any
	_ = json.Unmarshal(raw, &root)
	provs, _ := root["providers"].(map[string]any)
	magic := []string{}
	for k := range provs {
		if strings.HasPrefix(k, "magic-") {
			magic = append(magic, k)
		}
	}
	fmt.Fprintf(&b, "Magic providers  : %v\n", magic)
	if f != nil {
		fmt.Fprintf(&b, "Library active   : %s\n", catalog.ActiveProviderName(f))
	}
	return b.String()
}

func (w PiWriter) List(ctx Context) string {
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("  official       clear Magic-managed providers in models.json\n")
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
		fmt.Fprintf(&b, "  %-14s → %s%s\n", name, catalog.MagicPiProviderID(name), tag)
	}
	return b.String()
}

func removeMagicPiProviders(existing []byte) ([]byte, error) {
	root := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &root); err != nil {
			return nil, err
		}
	}
	provs := map[string]any{}
	if raw, ok := root["providers"].(map[string]any); ok {
		for k, v := range raw {
			if strings.HasPrefix(k, "magic-") {
				continue
			}
			provs[k] = v
		}
	}
	root["providers"] = provs
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func writeProbeSummary(ctx Context, req ApplyRequest) error {
	if req.Official {
		return nil
	}
	dir := filepath.Join(ctx.MagicHome, "registry")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	summary := map[string]any{
		"provider": req.Name,
		"endpoint": req.Endpoint.ID,
		"base_url": req.Endpoint.BaseURL,
		"protocol": req.Endpoint.Protocol,
		"key_env":  req.Provider.KeyEnv,
		"pick":     req.Provider.Pick,
		"harness":  "pi",
	}
	b, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "gateway-probe.json"), append(b, '\n'), 0o644)
}
