package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"magic/internal/catalog"
	"magic/internal/modelfile"
)

const (
	grokOfficialModel = "grok-4.5"
	grokManagedStart  = "# magic-managed-catalog-start"
	grokManagedEnd    = "# magic-managed-catalog-end"
	grokLegacyStart   = "# groks-managed-catalog-start"
	grokLegacyEnd     = "# groks-managed-catalog-end"
)

// GrokConfigPath is ~/.grok/config.toml.
func GrokConfigPath(userHome string) string {
	return filepath.Join(userHome, ".grok", "config.toml")
}

func GrokModePath(userHome string) string {
	return filepath.Join(userHome, ".grok", "provider-mode")
}

// GrokWriter projects library provider → Grok managed model catalog (or official default).
type GrokWriter struct{}

func (GrokWriter) Harness() string        { return HarnessGrok }
func (GrokWriter) Implemented() bool      { return true }
func (GrokWriter) SupportsOfficial() bool { return true }
func (GrokWriter) BackupPrefix() string   { return "config" }
func (GrokWriter) BackupExt() string      { return ".toml" }
func (GrokWriter) ConfigPath(userHome string) string {
	return GrokConfigPath(userHome)
}

func (w GrokWriter) Apply(ctx Context, req ApplyRequest) (string, error) {
	path := GrokConfigPath(ctx.UserHome)
	current, _ := readFileString(path)
	mode := "official"
	if !req.Official {
		mode = "provider"
	}
	updated, err := SetGrokManagedConfig(current, mode, ctx.MagicHome, req)
	if err != nil {
		return "", err
	}
	label := mode
	if !req.Official {
		label = req.Name
	}
	if updated == current && fileExists(path) {
		_ = writeProviderMode(ctx.UserHome, modeLabel(req))
		return fmt.Sprintf("Configuration already projecting %s", label), nil
	}
	bak, err := CreateBackup(ctx.MagicHome, HarnessGrok, path, "config", ".toml")
	if err != nil {
		return "", err
	}
	if err := atomicWrite(path, []byte(updated), 0o600); err != nil {
		return "", err
	}
	_ = writeProviderMode(ctx.UserHome, modeLabel(req))
	var b strings.Builder
	if bak != "" {
		fmt.Fprintf(&b, "Backup: %s\n", bak)
	}
	if req.Official {
		b.WriteString("Grok configured for official mode (Magic-managed catalog cleared).\n")
	} else {
		fmt.Fprintf(&b, "Grok configured for provider %q (endpoint=%s api_backend=%s).\n",
			req.Name, req.Endpoint.ID, ProtocolToGrokBackend(req.Endpoint.Protocol))
	}
	b.WriteString("Restart Grok for the change to take effect.\n")
	return b.String(), nil
}

func modeLabel(req ApplyRequest) string {
	if req.Official {
		return "official"
	}
	return req.Name
}

func (w GrokWriter) Status(ctx Context) string {
	path := GrokConfigPath(ctx.UserHome)
	text, _ := readFileString(path)
	mode := DetectGrokMode(text)
	def := GetGrokDefaultModel(text)
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("===== Grok provider status =====\n")
	fmt.Fprintf(&b, "Config           : %s\n", path)
	fmt.Fprintf(&b, "Mode             : %s\n", mode)
	fmt.Fprintf(&b, "Default model    : %s\n", def)
	if f != nil {
		fmt.Fprintf(&b, "Library active   : %s\n", catalog.ActiveProviderName(f))
	}
	return b.String()
}

func (w GrokWriter) List(ctx Context) string {
	path := GrokConfigPath(ctx.UserHome)
	text, _ := readFileString(path)
	mode := DetectGrokMode(text)
	f, _ := ctx.File()
	var b strings.Builder
	marker := " "
	if mode == "official" {
		marker = "*"
	}
	fmt.Fprintf(&b, "%s %-14s xAI official (no Magic catalog block)\n", marker, "official")
	if f == nil {
		return b.String()
	}
	libActive := catalog.ActiveProviderName(f)
	for name, p := range f.Providers {
		eps := catalog.NormalizeEndpoints(p)
		if len(eps) == 0 {
			continue
		}
		marker = " "
		if mode == name || (mode == "company" && name == libActive) {
			marker = "*"
		}
		tag := ""
		if name == libActive {
			tag = " [active→company]"
		}
		fmt.Fprintf(&b, "%s %-14s pick=%v%s\n", marker, name, p.Pick, tag)
	}
	return b.String()
}

func writeProviderMode(userHome, mode string) error {
	path := GrokModePath(userHome)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return atomicWrite(path, []byte(mode+"\n"), 0o644)
}

// DetectGrokMode returns official | providerName (from provider-mode file or heuristics).
func DetectGrokMode(text string) string {
	if strings.Contains(text, grokManagedStart) || strings.Contains(text, grokLegacyStart) {
		// Prefer reading provider-mode is done by Status via file; content heuristic:
		return "provider"
	}
	return "official"
}

// GetGrokDefaultModel reads [models].default.
func GetGrokDefaultModel(text string) string {
	lines := splitLines(text)
	inModels := false
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[models]") {
			inModels = true
			continue
		}
		if inModels && strings.HasPrefix(trim, "[") {
			break
		}
		if inModels {
			if v, ok := parseQuotedAssign(trim, "default"); ok {
				return v
			}
		}
	}
	return grokOfficialModel
}

// SetGrokManagedConfig applies Magic-managed catalog from ApplyRequest or clears for official.
// magicHome is used to resolve pick meta from registry/models.json (missing → defaults).
func SetGrokManagedConfig(text, mode, magicHome string, req ApplyRequest) (string, error) {
	lines := splitLines(text)
	lines = removeManagedBlock(lines)
	if mode == "official" || req.Official {
		lines = upsertModelsString(lines, "default", grokOfficialModel)
		lines = upsertModelsString(lines, "default_reasoning_effort", "high")
		return joinLines(lines), nil
	}

	defaultModel := ""
	if len(req.Provider.Pick) > 0 {
		defaultModel = req.Provider.Pick[0]
	}
	if req.Provider.Models != nil && req.Provider.Models["default"] != "" {
		defaultModel = req.Provider.Models["default"]
	}
	if defaultModel == "" {
		defaultModel = "relay-default"
	}

	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
		lines = append(lines, "")
	}
	lines = append(lines, buildGrokBlockFromProvider(magicHome, req)...)
	lines = upsertModelsString(lines, "default", defaultModel)
	lines = upsertModelsString(lines, "default_reasoning_effort", "high")
	result := joinLines(lines)
	if GetGrokDefaultModel(result) != defaultModel {
		return "", fmt.Errorf("default model verification failed (got %s want %s)", GetGrokDefaultModel(result), defaultModel)
	}
	return result, nil
}

func removeManagedBlock(lines []string) []string {
	startMark := map[string]bool{
		grokManagedStart: true, grokLegacyStart: true,
		"# groks-managed-model-start": true, "# magic-managed-model-start": true,
	}
	endMark := map[string]bool{
		grokManagedEnd: true, grokLegacyEnd: true,
		"# groks-managed-model-end": true, "# magic-managed-model-end": true,
	}
	var out []string
	skipping := false
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if startMark[trim] {
			skipping = true
			continue
		}
		if endMark[trim] {
			skipping = false
			continue
		}
		if skipping {
			continue
		}
		out = append(out, line)
	}
	return out
}

func upsertModelsString(lines []string, key, value string) []string {
	settingLine := fmt.Sprintf("%s = %q", key, value)
	sectionStarts := []int{}
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "[models]") {
			sectionStarts = append(sectionStarts, i)
		}
	}
	if len(sectionStarts) == 0 {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		return append(lines, "[models]", settingLine)
	}
	start := sectionStarts[0]
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "[") {
			end = i
			break
		}
	}
	found := false
	var body []string
	for i := start + 1; i < end; i++ {
		cur := lines[i]
		if strings.HasPrefix(strings.TrimSpace(cur), key) && strings.Contains(cur, "=") {
			if !found {
				body = append(body, settingLine)
				found = true
			}
			continue
		}
		body = append(body, cur)
	}
	if !found {
		body = append([]string{settingLine}, body...)
	}
	out := append([]string{}, lines[:start+1]...)
	out = append(out, body...)
	if end < len(lines) {
		out = append(out, lines[end:]...)
	}
	return out
}

func buildGrokBlockFromProvider(magicHome string, req ApplyRequest) []string {
	backend := ProtocolToGrokBackend(req.Endpoint.Protocol)
	keyEnv := strings.TrimSpace(req.Provider.KeyEnv)
	if keyEnv == "" {
		keyEnv = req.Name
	}
	picks := req.Provider.Pick
	if len(picks) == 0 {
		if def := req.Provider.Models["default"]; def != "" {
			picks = []string{def}
		} else {
			picks = []string{req.Name}
		}
	}
	resolve := modelfile.MetaResolver(magicHome)
	var lines []string
	lines = append(lines, grokManagedStart)
	lines = append(lines, fmt.Sprintf("# magic provider=%s endpoint=%s", req.Name, req.Endpoint.ID))
	for _, id := range picks {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		name := id
		ctxWin := int64(256000)
		maxTok := int64(128000)
		supportsReasoning := true
		efforts := []string{"low", "medium", "high", "xhigh"}
		if meta, ok := resolve(id); ok {
			if meta.Name != "" {
				name = meta.Name
			}
			if meta.ContextWindow > 0 {
				ctxWin = meta.ContextWindow
			}
			if meta.MaxTokens > 0 {
				maxTok = meta.MaxTokens
			}
			if derived, fromMap := grokEffortsFromMeta(meta); fromMap {
				supportsReasoning = len(derived) > 0
				if len(derived) > 0 {
					efforts = derived
				} else {
					efforts = nil
				}
			} else if meta.Reasoning {
				supportsReasoning = true
				// keep default effort list
			} else {
				supportsReasoning = false
				efforts = nil
			}
		}
		lines = append(lines,
			fmt.Sprintf("[model.%q]", id),
			fmt.Sprintf("model = %q", id),
			fmt.Sprintf("name = %q", name),
			fmt.Sprintf("base_url = %q", req.Endpoint.BaseURL),
			fmt.Sprintf("env_key = %q", keyEnv),
			fmt.Sprintf("api_backend = %q", backend),
			fmt.Sprintf("context_window = %d", ctxWin),
			fmt.Sprintf("max_completion_tokens = %d", maxTok),
			fmt.Sprintf("supports_reasoning_effort = %v", supportsReasoning),
		)
		if supportsReasoning {
			lines = append(lines, `reasoning_effort = "high"`)
		}
		lines = append(lines, "")
		if supportsReasoning {
			for _, v := range efforts {
				def := "false"
				if v == "high" {
					def = "true"
				}
				lines = append(lines,
					fmt.Sprintf("[[model.%q.reasoning_efforts]]", id),
					fmt.Sprintf("value = %q", v),
					fmt.Sprintf("label = %q", effortLabel(v)),
					"default = "+def,
					"",
				)
			}
		}
	}
	lines = append(lines, grokManagedEnd)
	return lines
}

// grokEffortsFromMeta extracts non-null thinkingLevelMap values in stable level order.
// ok=true when a map was present (even if empty after filtering).
func grokEffortsFromMeta(meta modelfile.Model) (efforts []string, ok bool) {
	if meta.ThinkingLevelMap == nil {
		return nil, false
	}
	order := []string{"off", "minimal", "low", "medium", "high", "xhigh", "max"}
	seen := map[string]bool{}
	for _, k := range order {
		v, present := meta.ThinkingLevelMap[k]
		if !present || v == nil {
			continue
		}
		val := strings.TrimSpace(*v)
		if val == "" || seen[val] {
			continue
		}
		seen[val] = true
		efforts = append(efforts, val)
	}
	return efforts, true
}

func effortLabel(v string) string {
	switch v {
	case "none":
		return "None"
	case "low":
		return "Low Effort"
	case "medium":
		return "Medium Effort"
	case "high":
		return "High Effort"
	case "xhigh":
		return "Extra High Effort"
	case "max":
		return "Max Effort"
	default:
		return v
	}
}
