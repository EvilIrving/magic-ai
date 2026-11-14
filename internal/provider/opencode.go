package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"magic/internal/catalog"
	"magic/internal/secrets"
)

const opencodeMagicPrefix = "magic-"

// OpenCodeWriter projects providers.yaml into ~/.config/opencode/opencode.json.
// Credentials may also live in ~/.local/share/opencode/auth.json (/connect);
// third-party Apply writes apiKey into provider.options for a working offline path.
type OpenCodeWriter struct{}

func (OpenCodeWriter) Harness() string        { return HarnessOpenCode }
func (OpenCodeWriter) Implemented() bool      { return true }
func (OpenCodeWriter) SupportsOfficial() bool { return false } // OpenCode is a multi-provider aggregator
func (OpenCodeWriter) BackupPrefix() string   { return "opencode" }
func (OpenCodeWriter) BackupExt() string      { return ".json" }
func (OpenCodeWriter) ConfigPath(userHome string) string {
	dir := filepath.Join(userHome, ".config", "opencode")
	jsonc := filepath.Join(dir, "opencode.jsonc")
	jsonp := filepath.Join(dir, "opencode.json")
	if _, err := os.Stat(jsonp); err == nil {
		return jsonp
	}
	if _, err := os.Stat(jsonc); err == nil {
		return jsonc
	}
	return jsonp
}

func opencodeProviderID(name string) string {
	return opencodeMagicPrefix + strings.TrimSpace(name)
}

// Files projects the resolved selection into ~/.config/opencode/opencode.json:
// one Magic-managed provider block plus the default model, or the managed
// entries removed for none.
func (w OpenCodeWriter) Files(ctx Context, req ApplyRequest) ([]FileWrite, error) {
	path := w.ConfigPath(ctx.UserHome)
	raw, err := readTarget(ctx, path)
	if err != nil {
		return nil, err
	}
	root := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(secrets.StripJSONC(raw), &root); err != nil {
			return nil, fmt.Errorf("parse opencode.json: %w", err)
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
		return nil, err
	}
	out = append(out, '\n')
	return []FileWrite{{Path: path, Data: out, Mode: 0o600, Role: RoleProviderBlock}}, nil
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
	_ = json.Unmarshal(secrets.StripJSONC(raw), &root)
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
	b.WriteString("  none           clear Magic-managed providers in opencode.json (OpenCode is multi-provider)\n")
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
