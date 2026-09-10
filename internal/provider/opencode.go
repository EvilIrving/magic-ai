package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"magic/internal/catalog"
)

const opencodeMagicPrefix = "magic-"

// OpenCodeWriter projects providers.yaml into ~/.config/opencode/opencode.json.
// Credentials may also live in ~/.local/share/opencode/auth.json (/connect);
// third-party Apply writes apiKey into provider.options for a working offline path.
type OpenCodeWriter struct{}

func (OpenCodeWriter) Harness() string      { return HarnessOpenCode }
func (OpenCodeWriter) Implemented() bool    { return true }
func (OpenCodeWriter) BackupPrefix() string { return "opencode" }
func (OpenCodeWriter) BackupExt() string    { return ".json" }
func (OpenCodeWriter) ConfigPath(userHome string) string {
	return filepath.Join(userHome, ".config", "opencode", "opencode.json")
}

func opencodeProviderID(name string) string {
	return opencodeMagicPrefix + strings.TrimSpace(name)
}

func (w OpenCodeWriter) Apply(ctx Context, req ApplyRequest) (string, error) {
	path := w.ConfigPath(ctx.UserHome)
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	bak, err := CreateBackup(ctx.MagicHome, HarnessOpenCode, path, "opencode", ".json")
	if err != nil {
		return "", err
	}

	root := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &root); err != nil {
			return "", fmt.Errorf("parse opencode.json: %w", err)
		}
	}
	if req.Official {
		clearOpenCodeMagic(root)
	} else {
		pid := opencodeProviderID(req.Name)
		provObj, _ := root["provider"].(map[string]any)
		if provObj == nil {
			provObj = map[string]any{}
		}
		// Drop other Magic-managed providers so only one active third-party remains.
		for k := range provObj {
			if strings.HasPrefix(k, opencodeMagicPrefix) {
				delete(provObj, k)
			}
		}
		opts := map[string]any{}
		if req.Endpoint.BaseURL != "" {
			opts["baseURL"] = strings.TrimRight(req.Endpoint.BaseURL, "/")
		}
		if req.APIKey != "" {
			opts["apiKey"] = req.APIKey
		}
		entry := map[string]any{"options": opts}
		if len(req.Provider.Pick) > 0 {
			wl := make([]string, 0, len(req.Provider.Pick))
			for _, p := range req.Provider.Pick {
				if s := strings.TrimSpace(p); s != "" {
					wl = append(wl, s)
				}
			}
			if len(wl) > 0 {
				entry["whitelist"] = wl
			}
		}
		provObj[pid] = entry
		root["provider"] = provObj
		if len(req.Provider.Pick) > 0 {
			pick := strings.TrimSpace(req.Provider.Pick[0])
			if pick != "" {
				root["model"] = pid + "/" + pick
			}
		}
	}
	if _, ok := root["$schema"]; !ok {
		root["$schema"] = "https://opencode.ai/config.json"
	}
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return "", err
	}
	out = append(out, '\n')
	if err := atomicWrite(path, out, 0o600); err != nil {
		return "", err
	}
	var b strings.Builder
	if bak != "" {
		fmt.Fprintf(&b, "Backup: %s\n", bak)
	}
	if req.Official {
		b.WriteString("OpenCode: cleared Magic-managed provider.* / model (official).\n")
	} else {
		fmt.Fprintf(&b, "OpenCode: provider %s baseURL=%s model=%v\n",
			opencodeProviderID(req.Name), req.Endpoint.BaseURL, root["model"])
	}
	return b.String(), nil
}

func clearOpenCodeMagic(root map[string]any) {
	if provObj, ok := root["provider"].(map[string]any); ok {
		for k := range provObj {
			if strings.HasPrefix(k, opencodeMagicPrefix) {
				delete(provObj, k)
			}
		}
		if len(provObj) == 0 {
			delete(root, "provider")
		} else {
			root["provider"] = provObj
		}
	}
	if m, ok := root["model"].(string); ok && strings.HasPrefix(m, opencodeMagicPrefix) {
		delete(root, "model")
	}
}

func (w OpenCodeWriter) Status(ctx Context) string {
	path := w.ConfigPath(ctx.UserHome)
	var b strings.Builder
	b.WriteString("===== OpenCode provider status =====\n")
	fmt.Fprintf(&b, "Config           : %s\n", path)
	raw, err := os.ReadFile(path)
	if err != nil {
		b.WriteString("State            : missing\n")
		return b.String()
	}
	var root map[string]any
	_ = json.Unmarshal(raw, &root)
	fmt.Fprintf(&b, "Model            : %v\n", root["model"])
	magic := []string{}
	if provObj, ok := root["provider"].(map[string]any); ok {
		for k := range provObj {
			if strings.HasPrefix(k, opencodeMagicPrefix) {
				magic = append(magic, k)
			}
		}
	}
	fmt.Fprintf(&b, "Magic providers  : %v\n", magic)
	if f, _ := ctx.File(); f != nil {
		fmt.Fprintf(&b, "Library active   : %s\n", catalog.ActiveProviderName(f))
	}
	return b.String()
}

func (w OpenCodeWriter) List(ctx Context) string {
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("  official       clear Magic-managed providers in opencode.json\n")
	if f == nil {
		return b.String()
	}
	libActive := catalog.ActiveProviderName(f)
	for name := range f.Providers {
		tag := ""
		if name == libActive {
			tag = "  (active)"
		}
		fmt.Fprintf(&b, "  %-14s third-party → provider.magic-%s%s\n", name, name, tag)
	}
	return b.String()
}
