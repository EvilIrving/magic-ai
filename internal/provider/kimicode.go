package provider

import (
	"fmt"
	"path/filepath"
	"strings"

	"magic/internal/adapters/kimicode"
	"magic/internal/catalog"
	"magic/internal/modelfile"
)

// KimiCodeWriter projects the resolved selection into Kimi Code CLI's
// ~/.kimi-code/config.toml ($KIMI_CODE_HOME): one Magic-managed provider table,
// one model table per pick, and the root default_model pointer.
type KimiCodeWriter struct{}

func (KimiCodeWriter) Harness() string        { return HarnessKimiCode }
func (KimiCodeWriter) Implemented() bool      { return true }
func (KimiCodeWriter) SupportsOfficial() bool { return false } // Kimi Code's first party is /login OAuth (managed:kimi-code), not a file Magic may write
func (KimiCodeWriter) BackupPrefix() string   { return "config" }
func (KimiCodeWriter) BackupExt() string      { return ".toml" }

// ConfigPath reuses the adapter's root (it honors KIMI_CODE_HOME) so provider
// projection and skills/MCP projection can never disagree about the directory.
func (KimiCodeWriter) ConfigPath(userHome string) string {
	return filepath.Join(kimicode.New().MirrorRoot(userHome), "config.toml")
}

// Source: vendor-doc | moonshotai.github.io/kimi-code/en/configuration/config-files
//   | checked 2026-11-14 | verified
//   Config lives at ~/.kimi-code/config.toml ($KIMI_CODE_HOME/config.toml);
//   TOML field names are snake_case and a key containing "." must be quoted.
//
// Source: vendor-doc | moonshotai.github.io/kimi-code/en/configuration/providers
//   | checked 2026-11-14 | verified
//   [providers.<id>] takes type/base_url/api_key (or api_key_env — exactly one),
//   [models.<alias>] takes provider/model/max_context_size/display_name, and the
//   root default_model names an alias. Provider types: kimi | anthropic | openai
//   | openai_responses | bedrock | google-genai | vertexai.
//
// Source: upstream | @moonshot-ai/kimi-code 2.6.2 dist/main.mjs
//   (ProviderConfigSchema, ModelBaseSchema, ALLOWED_PROVIDER_TYPES)
//   | checked 2026-11-14 | verified
//   Confirms the same field names from the parser side and that an unknown
//   [providers.*] id is accepted (managed:kimi-code is just an id).
//
// Magic deviation: Magic's tables are keyed magic-<providerName> and the model
// alias is magic-<providerName>/<pick>, so `provider use none` can remove
// exactly what Magic wrote and leave the user's own providers and models alone.

const kimiDefaultModelKey = "default_model"

// Files writes the provider table, the model tables and the default_model
// pointer; `none` (req.Official) removes Magic's tables and restores the
// default_model Magic displaced (or drops the key when it was not there).
func (w KimiCodeWriter) Files(ctx Context, req ApplyRequest) ([]FileWrite, error) {
	path := w.ConfigPath(ctx.UserHome)
	statePath := ProjectionStatePath(filepath.Dir(path))
	raw, err := readTarget(ctx, path)
	if err != nil {
		return nil, err
	}
	st, err := loadProjectionState(ctx, statePath)
	if err != nil {
		return nil, err
	}
	original := splitLines(string(raw))
	ownsTables := tomlHasTableMatching(original, isMagicKimiTable)
	lines := tomlDropTables(original, isMagicKimiTable)

	if req.Official {
		if !ownsTables && !st.touched(kimiDefaultModelKey) {
			// Nothing Magic wrote is here: leave the user's file alone.
			return nil, nil
		}
		if st.touched(kimiDefaultModelKey) {
			if current, _ := tomlRootValue(lines, kimiDefaultModelKey); st.stillMagic(kimiDefaultModelKey, current) {
				if prior, ok := st.prior(kimiDefaultModelKey); ok {
					lines = tomlSetRootValue(lines, kimiDefaultModelKey, prior)
				} else {
					lines = tomlDropRootValue(lines, kimiDefaultModelKey)
				}
			}
		}
		return []FileWrite{
			{Path: path, Data: []byte(joinLines(lines)), Mode: 0o600, Role: RoleProviderBlock},
			clearFile(statePath),
		}, nil
	}

	picks := kimiPicks(req)
	if len(picks) == 0 {
		return nil, fmt.Errorf("provider %q has no pick (models) to project into %s", req.Name, path)
	}
	alias := kimiModelAlias(req.Name, picks[0])
	if current, present := tomlRootValue(lines, kimiDefaultModelKey); !st.touched(kimiDefaultModelKey) {
		st.remember(kimiDefaultModelKey, current, present, alias)
	} else {
		st.Written[kimiDefaultModelKey] = alias
	}
	lines = tomlAppendBlock(lines, kimiProviderBlock(req))
	resolve := modelfile.MetaResolver(ctx.MagicHome)
	for _, pick := range picks {
		lines = tomlAppendBlock(lines, kimiModelBlock(req, pick, resolve))
	}
	lines = tomlSetRootValue(lines, kimiDefaultModelKey, alias)

	stateWrite, err := st.file(statePath)
	if err != nil {
		return nil, err
	}
	return []FileWrite{
		{Path: path, Data: []byte(joinLines(lines)), Mode: 0o600, Role: RoleProviderBlock},
		stateWrite,
	}, nil
}

// isMagicKimiTable matches the tables Magic owns: providers.magic-<name> (and
// its nested sub-tables) and models.magic-<name>/<pick>.
func isMagicKimiTable(path string) bool {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return false
	}
	switch parts[0] {
	case "providers", "models":
		return strings.HasPrefix(parts[1], "magic-")
	}
	return false
}

func kimiProviderBlock(req ApplyRequest) []string {
	providerID := catalog.MagicPiProviderID(req.Name)
	block := []string{
		fmt.Sprintf("[providers.%s]", tomlTableKey(providerID)),
		fmt.Sprintf("type = %q", kimiProviderType(req.Endpoint.Protocol)),
	}
	if base := strings.TrimSpace(req.Endpoint.BaseURL); base != "" {
		block = append(block, fmt.Sprintf("base_url = %q", strings.TrimRight(base, "/")))
	}
	if key := strings.TrimSpace(req.APIKey); key != "" {
		block = append(block, fmt.Sprintf("api_key = %q", key))
	} else {
		// Kimi Code wants exactly one of api_key / api_key_env; the library's
		// env var name is the entry point when Magic resolved no secret.
		block = append(block, fmt.Sprintf("api_key_env = %q", providerKeyEnv(req)))
	}
	return block
}

// kimiModelBlock is one [models.<alias>] table, enriched from registry meta when
// the pick is known there.
func kimiModelBlock(req ApplyRequest, pick string, resolve func(string) (modelfile.Model, bool)) []string {
	alias := kimiModelAlias(req.Name, pick)
	block := []string{
		fmt.Sprintf("[models.%s]", tomlTableKey(alias)),
		fmt.Sprintf("provider = %q", catalog.MagicPiProviderID(req.Name)),
		fmt.Sprintf("model = %q", pick),
	}
	display := pick
	var contextSize int64
	if meta, ok := resolve(pick); ok {
		if name := strings.TrimSpace(meta.Name); name != "" {
			display = name
		}
		contextSize = meta.ContextWindow
	}
	if display != pick {
		block = append(block, fmt.Sprintf("display_name = %q", display))
	}
	if contextSize > 0 {
		block = append(block, fmt.Sprintf("max_context_size = %d", contextSize))
	}
	return block
}

// kimiModelAlias is the models key Magic writes: stable, provider-scoped, and
// impossible to confuse with a user alias because of the magic- prefix.
func kimiModelAlias(providerName, pick string) string {
	return catalog.MagicPiProviderID(providerName) + "/" + pick
}

func kimiPicks(req ApplyRequest) []string {
	var picks []string
	for _, pick := range req.Provider.Pick {
		if p := strings.TrimSpace(pick); p != "" {
			picks = append(picks, p)
		}
	}
	if len(picks) == 0 {
		if def := strings.TrimSpace(req.Provider.Models["default"]); def != "" {
			picks = []string{def}
		}
	}
	return picks
}

// providerKeyEnv is the env var name the library binds the secret to.
func providerKeyEnv(req ApplyRequest) string {
	if keyEnv := strings.TrimSpace(req.Provider.KeyEnv); keyEnv != "" {
		return keyEnv
	}
	return req.Name
}

// kimiProviderType maps a Magic endpoint protocol onto Kimi Code's provider
// `type`, from the documented type table (see the Source block above).
func kimiProviderType(protocol string) string {
	switch catalog.NormalizeProtocol(protocol) {
	case catalog.ProtocolAnthropicMessages:
		return "anthropic"
	case catalog.ProtocolOpenAIResponses:
		return "openai_responses"
	default:
		return "openai"
	}
}

func (w KimiCodeWriter) Status(ctx Context) string {
	path := w.ConfigPath(ctx.UserHome)
	text, _ := readFileString(path)
	lines := splitLines(text)
	def, _ := tomlRootValue(lines, kimiDefaultModelKey)
	var magic []string
	for _, line := range lines {
		if table, ok := tomlTablePath(line); ok && isMagicKimiTable(table) {
			magic = append(magic, table)
		}
	}
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("===== Kimi Code provider status =====\n")
	fmt.Fprintf(&b, "Config           : %s\n", path)
	fmt.Fprintf(&b, "default_model    : %s\n", orNone(def))
	fmt.Fprintf(&b, "Magic tables     : %v\n", magic)
	if f != nil {
		fmt.Fprintf(&b, "Library active   : %s\n", catalog.ActiveProviderName(f))
	}
	return b.String()
}

func (w KimiCodeWriter) List(ctx Context) string {
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("  none           clear Magic-managed providers in config.toml (Kimi Code's first party is /login)\n")
	if f == nil {
		return b.String()
	}
	libActive := catalog.ActiveProviderName(f)
	for name, p := range f.Providers {
		if _, ok := catalog.PickEndpoint(p, kimiProtocolPreference...); !ok {
			continue
		}
		tag := ""
		if name == libActive {
			tag = " [active→company]"
		}
		fmt.Fprintf(&b, "  %-14s → providers.\"%s\"%s\n", name, catalog.MagicPiProviderID(name), tag)
	}
	return b.String()
}

// kimiProtocolPreference is the endpoint order Kimi Code is projected with.
// OpenAI Chat Completions first: the vendor documents that third-party
// reasoning providers (DeepSeek/Qwen/One API) work out of the box on it.
var kimiProtocolPreference = []string{
	catalog.ProtocolOpenAIChatCompletions,
	catalog.ProtocolOpenAIResponses,
	catalog.ProtocolAnthropicMessages,
}

func orNone(v string) string {
	if strings.TrimSpace(v) == "" {
		return "(unset)"
	}
	return v
}
