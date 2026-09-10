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

// WorkBuddyWriter projects providers.yaml into ~/.codebuddy/models.json
// (falls back to ~/.workbuddy/models.json when CodeBuddy dir is absent).
type WorkBuddyWriter struct{}

func (WorkBuddyWriter) Harness() string      { return HarnessWorkBuddy }
func (WorkBuddyWriter) Implemented() bool    { return true }
func (WorkBuddyWriter) BackupPrefix() string { return "models" }
func (WorkBuddyWriter) BackupExt() string    { return ".json" }

func (w WorkBuddyWriter) ConfigPath(userHome string) string {
	cb := filepath.Join(userHome, ".codebuddy", "models.json")
	if st, err := os.Stat(filepath.Dir(cb)); err == nil && st.IsDir() {
		return cb
	}
	if _, err := os.Stat(cb); err == nil {
		return cb
	}
	wb := filepath.Join(userHome, ".workbuddy", "models.json")
	if st, err := os.Stat(filepath.Dir(wb)); err == nil && st.IsDir() {
		return wb
	}
	return cb
}

func (w WorkBuddyWriter) Apply(ctx Context, req ApplyRequest) (string, error) {
	path := w.ConfigPath(ctx.UserHome)
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	bak, err := CreateBackup(ctx.MagicHome, HarnessWorkBuddy, path, "models", ".json")
	if err != nil {
		return "", err
	}

	root := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &root); err != nil {
			return "", fmt.Errorf("parse models.json: %w", err)
		}
	}

	if req.Official {
		root["models"] = filterNonMagicWorkBuddyModels(root["models"])
		if avail, ok := root["availableModels"].([]any); ok {
			root["availableModels"] = filterMagicAvailable(avail)
		}
	} else {
		models := buildWorkBuddyModels(req, modelfile.MetaResolver(ctx.MagicHome))
		// Replace Magic-managed entries; keep user non-magic models.
		kept := filterNonMagicWorkBuddyModels(root["models"])
		merged := append(kept, models...)
		root["models"] = merged
		var prevAvail []any
		if avail, ok := root["availableModels"].([]any); ok {
			prevAvail = filterMagicAvailable(avail)
		}
		root["availableModels"] = unionModelIDs(prevAvail, modelIDs(merged))
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
		b.WriteString("WorkBuddy: removed Magic-managed models (official).\n")
	} else {
		fmt.Fprintf(&b, "WorkBuddy: wrote %d model(s) from provider %s → %s\n",
			len(req.Provider.Pick), req.Name, path)
	}
	return b.String(), nil
}

func isMagicWorkBuddyID(id string) bool {
	return strings.HasPrefix(id, "magic-")
}

func filterNonMagicWorkBuddyModels(v any) []any {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []any
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := m["id"].(string)
		if isMagicWorkBuddyID(id) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func filterMagicAvailable(avail []any) []any {
	var out []any
	for _, a := range avail {
		s, _ := a.(string)
		if isMagicWorkBuddyID(s) {
			continue
		}
		out = append(out, a)
	}
	return out
}

func buildWorkBuddyModels(req ApplyRequest, resolve func(pickID string) (modelfile.Model, bool)) []any {
	base := strings.TrimRight(req.Endpoint.BaseURL, "/")
	// WorkBuddy expects full chat completions URL.
	url := base
	if base != "" && !strings.HasSuffix(base, "/chat/completions") {
		if strings.HasSuffix(base, "/v1") {
			url = base + "/chat/completions"
		} else {
			url = base + "/v1/chat/completions"
		}
	}
	keyRef := strings.TrimSpace(req.Provider.KeyEnv)
	if keyRef == "" {
		keyRef = req.Name
	}
	apiKey := "${" + keyRef + "}"
	vendor := req.Name
	var out []any
	for _, pick := range req.Provider.Pick {
		id := strings.TrimSpace(pick)
		if id == "" {
			continue
		}
		magicID := "magic-" + req.Name + "-" + sanitizeModelID(id)
		name := id
		maxIn, maxOut := 128000, 8192
		if resolve != nil {
			if meta, ok := resolve(id); ok {
				if meta.Name != "" {
					name = meta.Name
				}
				if meta.ContextWindow > 0 {
					maxIn = int(meta.ContextWindow)
				}
				if meta.MaxTokens > 0 {
					maxOut = int(meta.MaxTokens)
				}
			}
		}
		out = append(out, map[string]any{
			"id":               magicID,
			"name":             name,
			"vendor":           vendor,
			"url":              url,
			"apiKey":           apiKey,
			"maxInputTokens":   maxIn,
			"maxOutputTokens":  maxOut,
			"supportsToolCall": true,
			"supportsImages":   false,
			"_magic":           true,
			"_sourcePick":      id,
		})
	}
	return out
}

func modelIDs(models []any) []any {
	var out []any
	for _, item := range models {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := m["id"].(string); ok && id != "" {
			out = append(out, id)
		}
	}
	return out
}

func unionModelIDs(parts ...[]any) []any {
	seen := map[string]bool{}
	var out []any
	for _, part := range parts {
		for _, v := range part {
			s, _ := v.(string)
			if s == "" || seen[s] {
				continue
			}
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func sanitizeModelID(id string) string {
	id = strings.ReplaceAll(id, "/", "-")
	id = strings.ReplaceAll(id, " ", "-")
	return id
}

func (w WorkBuddyWriter) Status(ctx Context) string {
	path := w.ConfigPath(ctx.UserHome)
	var b strings.Builder
	b.WriteString("===== WorkBuddy provider status =====\n")
	fmt.Fprintf(&b, "Models file      : %s\n", path)
	raw, err := os.ReadFile(path)
	if err != nil {
		b.WriteString("State            : missing\n")
		return b.String()
	}
	var root map[string]any
	_ = json.Unmarshal(raw, &root)
	n := 0
	if arr, ok := root["models"].([]any); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				if id, _ := m["id"].(string); isMagicWorkBuddyID(id) {
					n++
				}
			}
		}
	}
	fmt.Fprintf(&b, "Magic models     : %d\n", n)
	if f, _ := ctx.File(); f != nil {
		fmt.Fprintf(&b, "Library active   : %s\n", catalog.ActiveProviderName(f))
	}
	return b.String()
}

func (w WorkBuddyWriter) List(ctx Context) string {
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("  official       clear Magic-managed models in models.json\n")
	if f == nil {
		return b.String()
	}
	libActive := catalog.ActiveProviderName(f)
	for name := range f.Providers {
		tag := ""
		if name == libActive {
			tag = "  (active)"
		}
		fmt.Fprintf(&b, "  %-14s third-party → ~/.codebuddy/models.json%s\n", name, tag)
	}
	return b.String()
}
