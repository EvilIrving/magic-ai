package provider

import (
	"fmt"
	"path/filepath"
	"strings"

	"magic/internal/adapters/vibe"
	"magic/internal/catalog"
)

// VibeWriter projects the resolved selection into Mistral Vibe's
// $VIBE_HOME/config.toml (~/.vibe): one [[providers]] entry, one [[models]]
// entry per pick, and the root active_model pointer.
type VibeWriter struct{}

func (VibeWriter) Harness() string        { return HarnessVibe }
func (VibeWriter) Implemented() bool      { return true }
func (VibeWriter) SupportsOfficial() bool { return false } // Vibe's first party is the Mistral account/keyring path, not a file Magic may write
func (VibeWriter) BackupPrefix() string   { return "config" }
func (VibeWriter) BackupExt() string      { return ".toml" }

// ConfigPath reuses the adapter's root (it honors VIBE_HOME, including the
// tilde form upstream expands) so the two projections share one path table.
func (VibeWriter) ConfigPath(userHome string) string {
	return filepath.Join(vibe.New().MirrorRoot(userHome), "config.toml")
}

// Source: upstream | mistralai/mistral-vibe vibe/core/config/models.py
//   (ProviderConfig: name, api_base, api_key_env_var, api_style, backend, extra_headers;
//   ModelConfig: name, provider, alias, display_name, max_context_length …)
//   | checked 2026-11-14 | verified
//
// Source: vendor-doc | mistralai/mistral-vibe vibe/plugins/builtins/vibe/skills/vibe/SKILL.md
//   §Providers ([[providers]] name/api_base/api_key_env_var/backend) and §Models
//   ([[models]] name/provider/alias) | checked 2026-11-14 | verified
//
// Source: upstream | vibe/core/config/vibe_schema.py (the user TOML layer is
//   ~/.vibe/config.toml; active_model names a [[models]] alias; models merge
//   per alias, so a Magic entry never replaces a user entry) | checked 2026-11-14 | verified
//
// Source: upstream | vibe/app_server/_provider_credentials.py (_BEARER_STYLES =
//   openai, openai-responses, reasoning; api_style "anthropic" selects the
//   Anthropic wire) | checked 2026-11-14 | verified
//
// Magic deviation: Vibe's ProviderConfig has no api_key field — a provider is
// keyed by api_key_env_var (env var or system keyring). Magic therefore writes
// the *name* of the key's env var, like Codex's env_key, and never the value.
//
// Never: write $VIBE_HOME/.env — Vibe's credential file (see the adapter's
// Source block); the env var reference stays a reference.

const (
	vibeManagedStart = "# magic-managed-start"
	vibeManagedEnd   = "# magic-managed-end"
	vibeActiveModel  = "active_model"
)

// Files writes Magic's managed block plus active_model; `none` removes the
// block and restores the active_model Magic displaced.
func (w VibeWriter) Files(ctx Context, req ApplyRequest) ([]FileWrite, error) {
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
	ownsBlock := strings.Contains(string(raw), vibeManagedStart)
	lines := tomlDropCommentBlock(original, vibeManagedStart, vibeManagedEnd)

	if req.Official {
		if !ownsBlock && !st.touched(vibeActiveModel) {
			// Nothing Magic wrote is here: leave the user's file alone.
			return nil, nil
		}
		if st.touched(vibeActiveModel) {
			if current, _ := tomlRootValue(lines, vibeActiveModel); st.stillMagic(vibeActiveModel, current) {
				if prior, ok := st.prior(vibeActiveModel); ok {
					lines = tomlSetRootValue(lines, vibeActiveModel, prior)
				} else {
					lines = tomlDropRootValue(lines, vibeActiveModel)
				}
			}
		}
		return []FileWrite{
			{Path: path, Data: []byte(joinLines(lines)), Mode: 0o600, Role: RoleProviderBlock},
			clearFile(statePath),
		}, nil
	}

	picks := vibePicks(req)
	if len(picks) == 0 {
		return nil, fmt.Errorf("provider %q has no pick (models) to project into %s", req.Name, path)
	}
	alias := vibeModelAlias(req.Name, picks[0])
	if current, present := tomlRootValue(lines, vibeActiveModel); !st.touched(vibeActiveModel) {
		st.remember(vibeActiveModel, current, present, alias)
	} else {
		st.Written[vibeActiveModel] = alias
	}
	lines = tomlAppendBlock(lines, vibeManagedBlock(req, picks))
	lines = tomlSetRootValue(lines, vibeActiveModel, alias)

	stateWrite, err := st.file(statePath)
	if err != nil {
		return nil, err
	}
	return []FileWrite{
		{Path: path, Data: []byte(joinLines(lines)), Mode: 0o600, Role: RoleProviderBlock},
		stateWrite,
	}, nil
}

func vibeManagedBlock(req ApplyRequest, picks []string) []string {
	providerName := catalog.MagicPiProviderID(req.Name)
	lines := []string{
		vibeManagedStart,
		fmt.Sprintf("# magic provider=%s endpoint=%s protocol=%s", req.Name, req.Endpoint.ID, req.Endpoint.Protocol),
		"[[providers]]",
		fmt.Sprintf("name = %q", providerName),
	}
	if base := strings.TrimSpace(req.Endpoint.BaseURL); base != "" {
		lines = append(lines, fmt.Sprintf("api_base = %q", strings.TrimRight(base, "/")))
	}
	lines = append(lines,
		fmt.Sprintf("api_key_env_var = %q", providerKeyEnv(req)),
		fmt.Sprintf("api_style = %q", vibeAPIStyle(req.Endpoint.Protocol)),
	)
	for _, pick := range picks {
		lines = append(lines,
			"",
			"[[models]]",
			fmt.Sprintf("name = %q", pick),
			fmt.Sprintf("provider = %q", providerName),
			fmt.Sprintf("alias = %q", vibeModelAlias(req.Name, pick)),
		)
	}
	return append(lines, vibeManagedEnd)
}

func vibeModelAlias(providerName, pick string) string {
	return catalog.MagicPiProviderID(providerName) + "/" + pick
}

func vibePicks(req ApplyRequest) []string {
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

// vibeAPIStyle maps a Magic endpoint protocol onto Vibe's api_style value.
func vibeAPIStyle(protocol string) string {
	switch catalog.NormalizeProtocol(protocol) {
	case catalog.ProtocolAnthropicMessages:
		return "anthropic"
	case catalog.ProtocolOpenAIResponses:
		return "openai-responses"
	default:
		return "openai"
	}
}

func (w VibeWriter) Status(ctx Context) string {
	path := w.ConfigPath(ctx.UserHome)
	text, _ := readFileString(path)
	active, _ := tomlRootValue(splitLines(text), vibeActiveModel)
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("===== Mistral Vibe provider status =====\n")
	fmt.Fprintf(&b, "Config           : %s\n", path)
	fmt.Fprintf(&b, "active_model     : %s\n", orNone(active))
	fmt.Fprintf(&b, "Magic block      : %s\n", map[bool]string{true: "present", false: "absent"}[strings.Contains(text, vibeManagedStart)])
	if f != nil {
		fmt.Fprintf(&b, "Library active   : %s\n", catalog.ActiveProviderName(f))
	}
	return b.String()
}

func (w VibeWriter) List(ctx Context) string {
	f, _ := ctx.File()
	var b strings.Builder
	b.WriteString("  none           clear the Magic-managed providers/models block in config.toml (Vibe's first party is the Mistral account path)\n")
	if f == nil {
		return b.String()
	}
	libActive := catalog.ActiveProviderName(f)
	for name, p := range f.Providers {
		if _, ok := catalog.PickEndpoint(p, vibeProtocolPreference...); !ok {
			continue
		}
		tag := ""
		if name == libActive {
			tag = " [active→company]"
		}
		fmt.Fprintf(&b, "  %-14s → [[providers]] %s%s\n", name, catalog.MagicPiProviderID(name), tag)
	}
	return b.String()
}

// vibeProtocolPreference is the endpoint order Vibe is projected with:
// api_style openai | anthropic | openai-responses are all first class.
var vibeProtocolPreference = []string{
	catalog.ProtocolOpenAIChatCompletions,
	catalog.ProtocolAnthropicMessages,
	catalog.ProtocolOpenAIResponses,
}
