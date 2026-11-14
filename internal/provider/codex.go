package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"magic/internal/catalog"
	"magic/internal/modelfile"
)

const codexOfficialProvider = "openai"

var (
	codexConflictKeys = map[string]struct{}{
		"model_catalog_json": {}, "preferred_auth_method": {}, "forced_login_method": {},
		"profile": {}, "openai_base_url": {}, "oss_provider": {},
		// Magic-managed third-party-only; stripped then re-added when projecting.
		"model_reasoning_effort": {},
	}
	deepseekModelRe = regexp.MustCompile(`(?i)^\s*model\s*=\s*"deepseek-[A-Za-z0-9_.-]+`)
)

// CodexConfigPath is ~/.codex/config.toml.
func CodexConfigPath(userHome string) string {
	return filepath.Join(userHome, ".codex", "config.toml")
}

// CodexModelsPath is ~/.codex/models.json (Codex-native catalog; Magic-managed for third-party).
func CodexModelsPath(userHome string) string {
	return filepath.Join(userHome, ".codex", "models.json")
}

// CodexCatalogJSONValue is the model_catalog_json string written to config.toml.
// Uses ~/.codex/models.json when userHome is the process home; otherwise an absolute path
// (tests / custom CODEX_HOME layouts).
func CodexCatalogJSONValue(userHome string) string {
	path := CodexModelsPath(userHome)
	if home, err := os.UserHomeDir(); err == nil && filepath.Clean(userHome) == filepath.Clean(home) {
		return "~/.codex/models.json"
	}
	return path
}

// CodexWriter projects active/named provider → Codex model_provider + [model_providers.<name>].
type CodexWriter struct{}

func (CodexWriter) Harness() string        { return HarnessCodex }
func (CodexWriter) Implemented() bool      { return true }
func (CodexWriter) SupportsOfficial() bool { return true }
func (CodexWriter) BackupPrefix() string   { return "config" }
func (CodexWriter) BackupExt() string      { return ".toml" }
func (CodexWriter) ConfigPath(userHome string) string {
	return CodexConfigPath(userHome)
}

// Files projects the resolved selection into ~/.codex/config.toml and, for a
// third-party provider, the Magic-managed ~/.codex/models.json catalog that
// model_catalog_json points at. Official mode removes that catalog again, so
// Codex stops pointing at one Magic no longer maintains.
func (w CodexWriter) Files(ctx Context, req ApplyRequest) ([]FileWrite, error) {
	path := CodexConfigPath(ctx.UserHome)
	modelsPath := CodexModelsPath(ctx.UserHome)
	current, err := readTarget(ctx, path)
	if err != nil {
		return nil, err
	}
	if req.Official {
		updated, err := SetCodexManagedConfig(string(current), codexOfficialProvider, "", catalog.Provider{}, catalog.Endpoint{}, "", "")
		if err != nil {
			return nil, err
		}
		return []FileWrite{
			{Path: path, Data: []byte(updated), Mode: 0o600, Role: RoleProviderBlock},
			{Path: modelsPath, Mode: 0o600, Role: RoleModels, Remove: true},
		}, nil
	}
	cataloged := BuildCodexModelsCatalog(ctx.MagicHome, req.Provider)
	raw, err := json.MarshalIndent(cataloged, "", "  ")
	if err != nil {
		return nil, err
	}
	raw = append(raw, '\n')
	effort := codexReasoningEffort(ctx.MagicHome, req.Provider)
	updated, err := SetCodexManagedConfig(string(current), req.Name, req.Name, req.Provider, req.Endpoint, CodexCatalogJSONValue(ctx.UserHome), effort)
	if err != nil {
		return nil, err
	}
	return []FileWrite{
		{Path: modelsPath, Data: raw, Mode: 0o600, Role: RoleModels},
		{Path: path, Data: []byte(updated), Mode: 0o600, Role: RoleProviderBlock},
	}, nil
}

func (w CodexWriter) Status(ctx Context) string {
	path := CodexConfigPath(ctx.UserHome)
	text, _ := readFileString(path)
	prov := GetCodexProvider(text)
	f, _ := ctx.File()
	mode := "official"
	if prov != codexOfficialProvider && prov != "" {
		mode = "provider:" + prov
		if f != nil && catalog.ActiveProviderName(f) == prov {
			mode += " (active/company)"
		}
	}
	var b strings.Builder
	b.WriteString("===== Codex provider status =====\n")
	fmt.Fprintf(&b, "Config           : %s\n", path)
	fmt.Fprintf(&b, "model_provider   : %s\n", prov)
	fmt.Fprintf(&b, "Mode             : %s\n", mode)
	fmt.Fprintf(&b, "models.json      : %s\n", map[bool]string{true: "present", false: "absent"}[fileExists(CodexModelsPath(ctx.UserHome))])
	if f != nil {
		fmt.Fprintf(&b, "Library active   : %s\n", catalog.ActiveProviderName(f))
	}
	return b.String()
}

func (w CodexWriter) List(ctx Context) string {
	path := CodexConfigPath(ctx.UserHome)
	text, _ := readFileString(path)
	active := GetCodexProvider(text)
	f, _ := ctx.File()
	var b strings.Builder
	marker := " "
	if active == codexOfficialProvider {
		marker = "*"
	}
	fmt.Fprintf(&b, "%s %-14s official OpenAI / ChatGPT\n", marker, "official")
	if f == nil {
		return b.String()
	}
	libActive := catalog.ActiveProviderName(f)
	for name, p := range f.Providers {
		if _, ok := catalog.PickEndpoint(p, catalog.ProtocolOpenAIResponses, catalog.ProtocolOpenAIChatCompletions, "openai"); !ok {
			continue
		}
		marker = " "
		if name == active {
			marker = "*"
		}
		tag := ""
		if name == libActive {
			tag = " [active→company]"
		}
		fmt.Fprintf(&b, "%s %-14s %s%s\n", marker, name, p.KeyEnv, tag)
	}
	return b.String()
}

// GetCodexProvider reads top-level model_provider (default openai).
func GetCodexProvider(text string) string {
	lines := splitLines(text)
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[") {
			break
		}
		if strings.HasPrefix(trim, "model_provider") {
			if v, ok := parseQuotedAssign(trim, "model_provider"); ok {
				return v
			}
		}
	}
	return codexOfficialProvider
}

// SetCodexManagedConfig sets model_provider and upserts [model_providers.<name>] from library endpoint.
// catalogJSON non-empty → third-party: also set model_catalog_json + model_reasoning_effort.
// When providerID is openai (official), only switches model_provider; leaves MCP/projects/plugins.
// Never writes plaintext bearer tokens (uses env_key only).
func SetCodexManagedConfig(text, modelProvider, sectionName string, p catalog.Provider, ep catalog.Endpoint, catalogJSON, reasoningEffort string) (string, error) {
	lines := splitLines(text)
	firstHeader := len(lines)
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			firstHeader = i
			break
		}
	}

	var kept []string
	for i, line := range lines {
		if i < firstHeader {
			if k, ok := lineKey(line); ok {
				if _, bad := codexConflictKeys[k]; bad {
					continue
				}
			}
			if deepseekModelRe.MatchString(line) {
				continue
			}
		}
		kept = append(kept, line)
	}
	lines = kept
	firstHeader = len(lines)
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			firstHeader = i
			break
		}
	}

	providerLine := fmt.Sprintf("model_provider = %q", modelProvider)
	found := false
	for i := 0; i < firstHeader; i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "model_provider") {
			lines[i] = providerLine
			found = true
			break
		}
	}
	if !found {
		lines = append([]string{providerLine}, lines...)
	}

	lines = filterCodexConflictSections(lines)

	// Official: done after setting model_provider (catalog_json already stripped).
	if modelProvider == codexOfficialProvider || sectionName == "" {
		result := joinLines(lines)
		if GetCodexProvider(result) != modelProvider {
			return "", fmt.Errorf("provider verification failed (want %s)", modelProvider)
		}
		return result, nil
	}

	keyEnv := strings.TrimSpace(p.KeyEnv)
	if keyEnv == "" {
		keyEnv = sectionName
	}
	wire := ProtocolToCodexWire(ep.Protocol)
	sectionHeader := "[model_providers." + sectionName + "]"
	companySettings := []struct{ key, line string }{
		{"name", fmt.Sprintf("name = %q", sectionName+" relay")},
		{"base_url", fmt.Sprintf("base_url = %q", ep.BaseURL)},
		{"env_key", fmt.Sprintf("env_key = %q", keyEnv)},
		{"wire_api", fmt.Sprintf("wire_api = %q", wire)},
		{"requires_openai_auth", "requires_openai_auth = false"},
	}

	starts := []int{}
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[model_providers."+sectionName+"]") {
			starts = append(starts, i)
		}
	}
	if len(starts) > 1 {
		return "", fmt.Errorf("duplicate [model_providers.%s] sections", sectionName)
	}
	if len(starts) == 0 {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		lines = append(lines, sectionHeader)
		for _, s := range companySettings {
			lines = append(lines, s.line)
		}
	} else {
		start := starts[0]
		end := len(lines)
		for i := start + 1; i < len(lines); i++ {
			if strings.HasPrefix(strings.TrimSpace(lines[i]), "[") {
				end = i
				break
			}
		}
		seen := map[string]bool{}
		var section []string
		for i := start + 1; i < end; i++ {
			cur := lines[i]
			matched := ""
			for _, s := range companySettings {
				if strings.HasPrefix(strings.TrimSpace(cur), s.key) && strings.Contains(cur, "=") {
					matched = s.key
					break
				}
			}
			if matched == "" {
				// Drop plaintext bearer tokens if a prior installer wrote them.
				trim := strings.TrimSpace(cur)
				if strings.HasPrefix(trim, "experimental_bearer_token") || strings.HasPrefix(trim, "bearer_token") {
					continue
				}
				section = append(section, cur)
			} else if !seen[matched] {
				for _, s := range companySettings {
					if s.key == matched {
						section = append(section, s.line)
						seen[matched] = true
						break
					}
				}
			}
		}
		for _, s := range companySettings {
			if !seen[s.key] {
				section = append(section, s.line)
			}
		}
		newLines := append([]string{}, lines[:start+1]...)
		newLines = append(newLines, section...)
		if end < len(lines) {
			newLines = append(newLines, lines[end:]...)
		}
		lines = newLines
	}

	if modelID := codexDefaultModel(p); modelID != "" {
		lines = upsertCodexTopLevelString(lines, "model", modelID)
	}
	if catalogJSON != "" {
		lines = upsertCodexTopLevelString(lines, "model_catalog_json", catalogJSON)
	}
	if reasoningEffort == "" {
		reasoningEffort = "high"
	}
	lines = upsertCodexTopLevelString(lines, "model_reasoning_effort", reasoningEffort)

	result := joinLines(lines)
	if GetCodexProvider(result) != modelProvider {
		return "", fmt.Errorf("provider verification failed (want %s)", modelProvider)
	}
	return result, nil
}

func filterCodexConflictSections(lines []string) []string {
	var out []string
	skip := false
	for i := 0; i < len(lines); i++ {
		trim := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trim, "[") && strings.HasSuffix(trim, "]") {
			header := strings.TrimSpace(trim[1 : len(trim)-1])
			header = strings.Trim(header, `"'`)
			switch {
			case header == "profiles", strings.HasPrefix(header, "profiles."),
				header == "model_providers.deepseek", strings.HasPrefix(header, "model_providers.deepseek."):
				skip = true
				continue
			default:
				skip = false
			}
		}
		if skip {
			continue
		}
		out = append(out, lines[i])
	}
	return out
}

func codexDefaultModel(p catalog.Provider) string {
	if p.Models != nil {
		if def := strings.TrimSpace(p.Models["default"]); def != "" {
			return def
		}
	}
	for _, id := range p.Pick {
		if id = strings.TrimSpace(id); id != "" {
			return id
		}
	}
	return ""
}

func codexReasoningEffort(magicHome string, p catalog.Provider) string {
	id := codexDefaultModel(p)
	if id == "" || magicHome == "" {
		return "high"
	}
	meta, ok := modelfile.ResolvePickMeta(magicHome, id)
	if !ok || meta.ThinkingLevelMap == nil {
		return "high"
	}
	// Prefer high (DeepSeek default), then max, then first available.
	prefer := []string{"high", "max", "xhigh", "medium", "low"}
	for _, k := range prefer {
		if _, ok := meta.ThinkingLevelMap[k]; ok {
			return k
		}
	}
	for k := range meta.ThinkingLevelMap {
		return k
	}
	return "high"
}

// upsertCodexTopLevelString sets key = "value" in the preamble (before first [section]).
func upsertCodexTopLevelString(lines []string, key, value string) []string {
	setting := fmt.Sprintf("%s = %q", key, value)
	firstHeader := len(lines)
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			firstHeader = i
			break
		}
	}
	for i := 0; i < firstHeader; i++ {
		if k, ok := lineKey(lines[i]); ok && k == key {
			lines[i] = setting
			return lines
		}
	}
	insertAt := 0
	for i := 0; i < firstHeader; i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "model_provider") {
			insertAt = i + 1
			break
		}
	}
	out := append([]string{}, lines[:insertAt]...)
	out = append(out, setting)
	out = append(out, lines[insertAt:]...)
	return out
}

// --- Codex-native models.json (minimal viable catalog) ---

// CodexReasoningLevel is one supported_reasoning_levels entry.
type CodexReasoningLevel struct {
	Effort      string `json:"effort"`
	Description string `json:"description,omitempty"`
}

// CodexModelEntry is a minimal Codex-native catalog row (omit huge instruction blobs).
type CodexModelEntry struct {
	Slug                     string                `json:"slug"`
	DisplayName              string                `json:"display_name"`
	ContextWindow            int64                 `json:"context_window"`
	MaxContextWindow         int64                 `json:"max_context_window"`
	InputModalities          []string              `json:"input_modalities"`
	SupportedReasoningLevels []CodexReasoningLevel `json:"supported_reasoning_levels,omitempty"`
	DefaultReasoningLevel    string                `json:"default_reasoning_level,omitempty"`
	Visibility               string                `json:"visibility,omitempty"`
	SupportedInAPI           bool                  `json:"supported_in_api"`
}

// CodexModelsFile is the on-disk ~/.codex/models.json envelope.
type CodexModelsFile struct {
	Models []CodexModelEntry `json:"models"`
}

// BuildCodexModelsCatalog builds a minimal Codex-native catalog from Provider.Pick (+ Models)
// enriched via Magic registry meta when available.
func BuildCodexModelsCatalog(magicHome string, p catalog.Provider) CodexModelsFile {
	ids := codexCatalogIDs(p)
	resolve := modelfile.MetaResolver(magicHome)
	out := make([]CodexModelEntry, 0, len(ids))
	for _, id := range ids {
		out = append(out, buildCodexModelEntry(id, resolve))
	}
	return CodexModelsFile{Models: out}
}

func codexCatalogIDs(p catalog.Provider) []string {
	seen := map[string]struct{}{}
	var ids []string
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, id := range p.Pick {
		add(id)
	}
	if p.Models != nil {
		// Stable order for Models values.
		roles := make([]string, 0, len(p.Models))
		for r := range p.Models {
			roles = append(roles, r)
		}
		sort.Strings(roles)
		for _, r := range roles {
			add(p.Models[r])
		}
	}
	return ids
}

func buildCodexModelEntry(slug string, resolve func(string) (modelfile.Model, bool)) CodexModelEntry {
	entry := CodexModelEntry{
		Slug:                  slug,
		DisplayName:           slug,
		ContextWindow:         128000,
		MaxContextWindow:      128000,
		InputModalities:       []string{"text"},
		DefaultReasoningLevel: "high",
		SupportedReasoningLevels: []CodexReasoningLevel{
			{Effort: "low", Description: "Fast responses with lighter reasoning"},
			{Effort: "high", Description: "Extra high reasoning depth for complex problems"},
			{Effort: "max", Description: "Maximum reasoning depth for the hardest problems"},
		},
		Visibility:     "list",
		SupportedInAPI: true,
	}
	if resolve == nil {
		return entry
	}
	meta, ok := resolve(slug)
	if !ok {
		return entry
	}
	if meta.Name != "" {
		entry.DisplayName = meta.Name
	}
	if meta.ContextWindow > 0 {
		entry.ContextWindow = meta.ContextWindow
		entry.MaxContextWindow = meta.ContextWindow
	}
	if len(meta.Input) > 0 {
		entry.InputModalities = append([]string{}, meta.Input...)
	}
	if levels := codexReasoningLevelsFromMeta(meta); len(levels) > 0 {
		entry.SupportedReasoningLevels = levels
		// Prefer high if present, else first.
		entry.DefaultReasoningLevel = levels[0].Effort
		for _, l := range levels {
			if l.Effort == "high" {
				entry.DefaultReasoningLevel = "high"
				break
			}
		}
	}
	return entry
}

func codexReasoningLevelsFromMeta(meta modelfile.Model) []CodexReasoningLevel {
	if meta.ThinkingLevelMap == nil || len(meta.ThinkingLevelMap) == 0 {
		return nil
	}
	order := []string{"low", "medium", "high", "xhigh", "max"}
	desc := map[string]string{
		"low":    "Fast responses with lighter reasoning",
		"medium": "Balanced reasoning depth",
		"high":   "Extra high reasoning depth for complex problems",
		"xhigh":  "Extra-high reasoning depth",
		"max":    "Maximum reasoning depth for the hardest problems",
	}
	var levels []CodexReasoningLevel
	seen := map[string]struct{}{}
	for _, k := range order {
		if _, ok := meta.ThinkingLevelMap[k]; ok {
			levels = append(levels, CodexReasoningLevel{Effort: k, Description: desc[k]})
			seen[k] = struct{}{}
		}
	}
	// Any extra keys.
	extras := make([]string, 0)
	for k := range meta.ThinkingLevelMap {
		if _, ok := seen[k]; !ok {
			extras = append(extras, k)
		}
	}
	sort.Strings(extras)
	for _, k := range extras {
		levels = append(levels, CodexReasoningLevel{Effort: k})
	}
	return levels
}
