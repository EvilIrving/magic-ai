package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"magic/internal/catalog"
	"magic/internal/modelfile"
)

// ClaudeSettingsPath is ~/.claude/settings.json under userHome.
func ClaudeSettingsPath(userHome string) string {
	return filepath.Join(userHome, ".claude", "settings.json")
}

// BuildClaudeEnvFromProvider builds Magic-managed env from library provider + endpoint + key.
// Secret: ANTHROPIC_AUTH_TOKEN is primary (DeepSeek docs); ANTHROPIC_API_KEY is also set to
// the same value for broader Claude Code version compatibility.
// Optional CLAUDE_CODE_EFFORT_LEVEL / CLAUDE_CODE_AUTO_COMPACT_WINDOW come from registry meta
// of the default model when available (never hardcode DeepSeek-only values here).
func BuildClaudeEnvFromProvider(magicHome, name string, p catalog.Provider, ep catalog.Endpoint, apiKey string) (map[string]string, error) {
	if strings.TrimSpace(apiKey) == "" {
		keyEnv := p.KeyEnv
		if keyEnv == "" {
			keyEnv = name
		}
		return nil, fmt.Errorf("%s is not set — run: magic secrets set %s --value …", keyEnv, keyEnv)
	}
	env := map[string]string{
		"ANTHROPIC_BASE_URL":   ep.BaseURL,
		"ANTHROPIC_AUTH_TOKEN": apiKey,
		"ANTHROPIC_API_KEY":    apiKey,
	}
	for k, v := range catalog.ClaudeModelRoleEnv(p) {
		env[k] = v
	}
	enrichClaudeExtrasFromMeta(magicHome, env)
	return env, nil
}

// enrichClaudeExtrasFromMeta sets CLAUDE_CODE_EFFORT_LEVEL and CLAUDE_CODE_AUTO_COMPACT_WINDOW
// from registry meta of ANTHROPIC_MODEL when present.
func enrichClaudeExtrasFromMeta(magicHome string, env map[string]string) {
	modelID := strings.TrimSpace(env["ANTHROPIC_MODEL"])
	if modelID == "" || magicHome == "" {
		return
	}
	meta, ok := modelfile.ResolvePickMeta(magicHome, modelID)
	if !ok {
		return
	}
	if effort := claudeEffortFromMeta(meta); effort != "" {
		env["CLAUDE_CODE_EFFORT_LEVEL"] = effort
	}
	if meta.ContextWindow > 0 {
		// Prefer meta-derived compact window (~75% of context), not a hardcoded vendor value.
		compact := meta.ContextWindow * 75 / 100
		if compact > 0 {
			env["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] = strconv.FormatInt(compact, 10)
		}
	}
}

// claudeEffortFromMeta picks the highest available thinking level key (max > xhigh > high > …).
func claudeEffortFromMeta(meta modelfile.Model) string {
	if meta.ThinkingLevelMap == nil || len(meta.ThinkingLevelMap) == 0 {
		return ""
	}
	prefer := []string{"max", "xhigh", "high", "medium", "low"}
	for _, k := range prefer {
		if _, ok := meta.ThinkingLevelMap[k]; ok {
			return k
		}
	}
	for k := range meta.ThinkingLevelMap {
		return k
	}
	return ""
}

// UpsertClaudeSettings merges managed env into settings.json bytes.
// When clearManaged is true (official), all ClaudeManagedEnvKeys are removed.
func UpsertClaudeSettings(existing []byte, env map[string]string, clearManaged bool) ([]byte, error) {
	root := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &root); err != nil {
			return nil, fmt.Errorf("parse settings.json: %w", err)
		}
	}
	envObj := map[string]any{}
	if raw, ok := root["env"]; ok && raw != nil {
		switch t := raw.(type) {
		case map[string]any:
			envObj = t
		case map[string]string:
			for k, v := range t {
				envObj[k] = v
			}
		}
	}
	if clearManaged {
		for k := range ClaudeManagedEnvKeys {
			delete(envObj, k)
		}
	}
	for k, v := range env {
		envObj[k] = v
	}
	root["env"] = envObj
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// DetectClaudeActive returns a best-effort active provider name by matching ANTHROPIC_BASE_URL
// against providers.yaml anthropic endpoints (or "official" if no base URL).
func DetectClaudeActive(settings []byte, f *catalog.File) string {
	if len(settings) == 0 {
		return ModeOfficial
	}
	var root map[string]any
	if err := json.Unmarshal(settings, &root); err != nil {
		return "unknown"
	}
	env, _ := root["env"].(map[string]any)
	if env == nil {
		return ModeOfficial
	}
	baseURL := strings.TrimRight(fmt.Sprint(env["ANTHROPIC_BASE_URL"]), "/")
	if baseURL == "" || baseURL == "<nil>" {
		return ModeOfficial
	}
	model := strings.TrimSpace(fmt.Sprint(env["ANTHROPIC_MODEL"]))
	if model == "<nil>" {
		model = ""
	}
	if f == nil {
		return "unknown"
	}
	var matches []string
	for name, p := range f.Providers {
		ep, ok := catalog.PickEndpoint(p, catalog.ProtocolAnthropicMessages, "anthropic")
		if !ok {
			continue
		}
		if strings.TrimRight(ep.BaseURL, "/") == baseURL {
			matches = append(matches, name)
		}
	}
	if len(matches) == 1 {
		return matches[0]
	}
	if model != "" && len(matches) > 1 {
		var mm []string
		for _, name := range matches {
			p := f.Providers[name]
			def := ""
			if p.Models != nil {
				def = p.Models["default"]
			}
			if def == "" && len(p.Pick) > 0 {
				def = p.Pick[0]
			}
			if def == model {
				mm = append(mm, name)
			}
		}
		if len(mm) == 1 {
			return mm[0]
		}
	}
	if len(matches) == 0 {
		return "unknown"
	}
	return "unknown"
}

func applyClaude(ctx Context, req ApplyRequest) (string, error) {
	path := ClaudeSettingsPath(ctx.UserHome)
	var existing []byte
	if raw, err := os.ReadFile(path); err == nil {
		existing = raw
	} else if !os.IsNotExist(err) {
		return "", err
	}

	var env map[string]string
	clear := req.Official
	if !clear {
		var err error
		env, err = BuildClaudeEnvFromProvider(ctx.MagicHome, req.Name, req.Provider, req.Endpoint, req.APIKey)
		if err != nil {
			return "", err
		}
	}

	out, err := UpsertClaudeSettings(existing, env, clear)
	if err != nil {
		return "", err
	}

	bak, err := CreateBackup(ctx.MagicHome, HarnessClaude, path, "settings", ".json")
	if err != nil {
		return "", err
	}
	if err := atomicWrite(path, out, 0o600); err != nil {
		return "", err
	}
	var b strings.Builder
	if bak != "" {
		fmt.Fprintf(&b, "Backup: %s\n", bak)
	}
	if req.Official {
		b.WriteString("Active: official (Magic-managed ANTHROPIC_* / CLAUDE_CODE_* overrides cleared)\n")
	} else {
		fmt.Fprintf(&b, "Active provider: %s (endpoint=%s protocol=%s)\n", req.Name, req.Endpoint.ID, req.Endpoint.Protocol)
	}
	b.WriteString("Restart Claude Code for the change to take effect.\n")
	return b.String(), nil
}

func claudeStatusText(ctx Context) string {
	path := ClaudeSettingsPath(ctx.UserHome)
	raw, err := os.ReadFile(path)
	f, _ := ctx.File()
	active := "missing settings"
	envBase, envModel, auth := "Anthropic official", "Claude default", "no"
	if err == nil {
		active = DetectClaudeActive(raw, f)
		var root map[string]any
		_ = json.Unmarshal(raw, &root)
		if e, ok := root["env"].(map[string]any); ok {
			if v := fmt.Sprint(e["ANTHROPIC_BASE_URL"]); v != "" && v != "<nil>" {
				envBase = v
			}
			if v := fmt.Sprint(e["ANTHROPIC_MODEL"]); v != "" && v != "<nil>" {
				envModel = v
			}
			if e["ANTHROPIC_AUTH_TOKEN"] != nil || e["ANTHROPIC_API_KEY"] != nil {
				auth = "yes"
			}
		}
	}
	keyState := "not required"
	if f != nil && active != ModeOfficial && active != "unknown" && active != "missing settings" {
		if p, ok := f.Providers[active]; ok {
			ke := p.KeyEnv
			if ke == "" {
				ke = active
			}
			if SecretPresent(ctx.SecretsDir(), ke) {
				keyState = "set"
			} else {
				keyState = "missing"
			}
		}
	}
	var b strings.Builder
	b.WriteString("===== Claude Code provider status =====\n")
	fmt.Fprintf(&b, "Settings file : %s\n", path)
	fmt.Fprintf(&b, "Provider      : %s\n", active)
	fmt.Fprintf(&b, "Default model : %s\n", envModel)
	fmt.Fprintf(&b, "Base URL      : %s\n", envBase)
	fmt.Fprintf(&b, "Persistent key: %s\n", keyState)
	fmt.Fprintf(&b, "Auth in config: %s\n", auth)
	return b.String()
}

func claudeListText(ctx Context) string {
	f, _ := ctx.File()
	path := ClaudeSettingsPath(ctx.UserHome)
	raw, _ := os.ReadFile(path)
	active := DetectClaudeActive(raw, f)
	var b strings.Builder
	fmt.Fprintf(&b, "%s %-14s %s\n", map[bool]string{true: "*", false: " "}[active == ModeOfficial], "official", "Anthropic Official login")
	if f == nil {
		return b.String()
	}
	activeLib := catalog.ActiveProviderName(f)
	for name, p := range f.Providers {
		ep, ok := catalog.PickEndpoint(p, catalog.ProtocolAnthropicMessages, "anthropic")
		if !ok {
			continue
		}
		marker := " "
		if name == active {
			marker = "*"
		}
		keyState := "missing"
		ke := p.KeyEnv
		if ke == "" {
			ke = name
		}
		if SecretPresent(ctx.SecretsDir(), ke) {
			keyState = "set"
		}
		model := ""
		if p.Models != nil {
			model = p.Models["default"]
		}
		if model == "" && len(p.Pick) > 0 {
			model = p.Pick[0]
		}
		alias := ""
		if name == activeLib {
			alias = " [active→company]"
		}
		fmt.Fprintf(&b, "%s %-14s %-26s %s [%s]%s\n", marker, name, model, ep.BaseURL, keyState, alias)
	}
	return b.String()
}
