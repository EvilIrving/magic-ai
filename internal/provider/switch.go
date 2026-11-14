package provider

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"magic/internal/catalog"
)

// Context carries paths and options for provider switch / status / test.
type Context struct {
	MagicHome string
	UserHome  string
	HTTP      *http.Client  // nil → DefaultClient (tests inject httptest)
	Catalog   *catalog.File // optional preloaded; else LoadCatalog(MagicHome)
	// BackupDir is the apply backup set ApplyFiles records into. Empty → the
	// projection is written without a manifest entry (plan, `provider use`).
	BackupDir string
	// ReadFile is how a projection reads the content it merges into. nil → the
	// disk. plan sets it to the bytes its layers have produced so far, so a
	// provider projection stacks on top of the earlier layers instead of
	// re-reading the file.
	ReadFile func(path string) ([]byte, error)
}

// SecretsDir is $MAGIC_HOME/secrets.
func (c Context) SecretsDir() string {
	return filepath.Join(c.MagicHome, "secrets")
}

// ProvidersPath is $MAGIC_HOME/providers.yaml.
func (c Context) ProvidersPath() string {
	return filepath.Join(c.MagicHome, "providers.yaml")
}

// File returns the providers catalog (cached on Context when set).
func (c *Context) File() (*catalog.File, error) {
	if c.Catalog != nil {
		return c.Catalog, nil
	}
	f, err := LoadCatalog(c.MagicHome)
	if err != nil {
		return nil, err
	}
	c.Catalog = f
	return f, nil
}

// FileWrite is one file a projection writes, or removes, without touching the
// disk. Files returns them, so plan and apply compute the same bytes.
type FileWrite struct {
	Path string
	Data []byte
	Mode os.FileMode
	// Role names what the file is in this harness's dialect: settings-env (one
	// env block inside the config), models (model catalog), provider-block (the
	// provider section of the config itself) or sidecar (a Magic-managed helper
	// file). plan labels the layer with it.
	Role string
	// Remove deletes Path instead of writing it; Data is nil. Codex official
	// mode clears the Magic-managed model catalog this way, so the harness stops
	// pointing at a catalog Magic no longer maintains.
	Remove bool
}

// FileWrite roles: what a projection file is in its harness's dialect.
const (
	// RoleSettingsEnv is one env block inside a harness config (Claude settings.json).
	RoleSettingsEnv = "settings-env"
	// RoleModels is a model catalog the harness reads (Pi/Codex/WorkBuddy models.json).
	RoleModels = "models"
	// RoleProviderBlock is the provider section of the harness's own config file.
	RoleProviderBlock = "provider-block"
	// RoleSidecar is a Magic-managed helper file next to the config.
	RoleSidecar = "sidecar"
)

// SwitchWriter projects library selection into one harness's on-disk dialect.
// official → clear Magic-managed overrides / restore the first-party path;
//
//	only valid where SupportsOfficial() is true.
//
// none     → always just clear Magic-managed entries (multi-provider harnesses).
// provider → project the selected provider's matching endpoint (+ secret when needed).
// Adding a harness = one SwitchWriter, not another shell script.
type SwitchWriter interface {
	Harness() string
	Implemented() bool
	// SupportsOfficial reports whether the harness has a first-party provider
	// path. Pi / OpenCode are multi-provider aggregators and have none.
	SupportsOfficial() bool
	// Files is a pure function: it computes the files this projection writes
	// (or removes) for official / none / third-party, without touching the disk.
	// ApplyFiles does the writing, so apply and plan share one computation.
	Files(ctx Context, req ApplyRequest) ([]FileWrite, error)
	Status(ctx Context) string
	List(ctx Context) string
	ConfigPath(userHome string) string
	BackupPrefix() string
	BackupExt() string
}

var writers = map[string]SwitchWriter{}

func init() {
	Register(ClaudeWriter{})
	Register(CodexWriter{})
	Register(GrokWriter{})
	Register(PiWriter{})
	Register(OpenCodeWriter{})
	Register(WorkBuddyWriter{})
	Register(KimiCodeWriter{})
	Register(VibeWriter{})
	Register(HermesWriter{})
	Register(OmpWriter{})
	Register(DroidWriter{})
	Register(QoderWriter{})
	Register(MmxWriter{})
	// gemini/copilot/trae: Adapter apply only (no stable third-party dialect yet).
	// cline/zcode: a verified dialect Magic cannot write purely — see
	// internal/provider/stubdialects.go for what each one is missing.
	// mcpuser is MCP config, not a model provider.
	for _, id := range []string{"gemini", "copilot", "trae", "cline", "zcode"} {
		Register(StubWriter{ID: id})
	}
}

// Register adds or replaces a SwitchWriter.
func Register(w SwitchWriter) {
	if w == nil {
		return
	}
	writers[NormalizeHarness(w.Harness())] = w
}

// Writer returns the SwitchWriter for harness, or nil.
func Writer(harness string) SwitchWriter {
	return writers[NormalizeHarness(harness)]
}

// Writers returns registered writers (implemented first).
func Writers() []SwitchWriter {
	order := []string{
		HarnessClaude, HarnessCodex, HarnessGrok, HarnessPi,
		HarnessOpenCode, HarnessWorkBuddy, HarnessKimiCode, HarnessVibe,
		HarnessHermes, HarnessOmp, HarnessDroid, HarnessQoder, HarnessMmx,
		"gemini", "copilot", "trae", "cline", "zcode",
	}
	seen := map[string]bool{}
	var out []SwitchWriter
	for _, id := range order {
		if w, ok := writers[id]; ok {
			out = append(out, w)
			seen[id] = true
		}
	}
	for id, w := range writers {
		if !seen[id] {
			out = append(out, w)
		}
	}
	return out
}

// BuildApplyRequest resolves target + loads secret for Claude-style inline keys.
// needKey: if true and ModeProvider, API key must exist in Magic secrets.
func BuildApplyRequest(ctx Context, harness, rawTarget string, needKey bool) (ApplyRequest, error) {
	f, err := ctx.File()
	if err != nil {
		return ApplyRequest{}, err
	}
	rt, err := ResolveUseTarget(f, rawTarget)
	if err != nil {
		return ApplyRequest{}, err
	}
	if rt.Mode == ModeOfficial || rt.Mode == ModeClear {
		return ApplyRequest{Official: true}, nil
	}
	ep, err := PreferredEndpoint(rt.Provider, harness)
	if err != nil {
		return ApplyRequest{}, fmt.Errorf("provider %q: %w", rt.Name, err)
	}
	req := ApplyRequest{
		Official: false,
		Name:     rt.Name,
		Provider: rt.Provider,
		Endpoint: ep,
	}
	if needKey {
		keyEnv := strings.TrimSpace(rt.Provider.KeyEnv)
		if keyEnv == "" {
			keyEnv = rt.Name
		}
		key, err := ResolveSecret(ctx.SecretsDir(), keyEnv)
		if err != nil {
			return ApplyRequest{}, err
		}
		req.APIKey = key
	}
	return req, nil
}

// Use is the product entry: resolve target against library, then write it.
func Use(ctx Context, harness, rawTarget string) (string, error) {
	w := Writer(harness)
	if w == nil {
		return "", fmt.Errorf("unknown harness %q", harness)
	}
	if !w.Implemented() {
		return "", ErrNotImplemented(w.Harness())
	}
	h := NormalizeHarness(harness)

	f, err := ctx.File()
	if err != nil {
		return "", err
	}
	rt, err := ResolveUseTarget(f, rawTarget)
	if err != nil {
		return "", err
	}
	if rt.Mode == ModeOfficial && !w.SupportsOfficial() {
		return "", fmt.Errorf("harness %q has no official provider path (it is multi-provider); use `magic provider use none` to clear the Magic-managed entries", h)
	}

	// Claude / Pi / WorkBuddy / OpenCode / Kimi Code / Hermes / Oh My Pi / Droid /
	// Qoder / MiniMax Code write the real key value (D3).
	needKey := h == HarnessClaude || h == HarnessOpenCode || h == HarnessPi || h == HarnessWorkBuddy ||
		h == HarnessKimiCode || h == HarnessHermes || h == HarnessOmp || h == HarnessDroid ||
		h == HarnessQoder || h == HarnessMmx
	// Codex / Grok / Vibe keep their native env-var reference but still require
	// the key present, so the reference never dangles.
	if (h == HarnessCodex || h == HarnessGrok || h == HarnessVibe) && rt.Mode == ModeProvider {
		keyEnv := strings.TrimSpace(rt.Provider.KeyEnv)
		if keyEnv == "" {
			keyEnv = rt.Name
		}
		if _, err := ResolveSecret(ctx.SecretsDir(), keyEnv); err != nil {
			return "", err
		}
	}
	req, err := BuildApplyRequest(ctx, harness, rawTarget, needKey)
	if err != nil {
		return "", err
	}
	msg, err := ApplyFiles(ctx, w, req)
	if err != nil {
		return "", err
	}
	// Pi records the endpoint it may probe; that note is a side effect of
	// `provider use`, not part of the projection Files computes.
	if h == HarnessPi {
		_ = writeProbeSummary(ctx, req)
	}
	return msg, nil
}

// ErrNotImplemented is returned by stub writers.
func ErrNotImplemented(harness string) error {
	return fmt.Errorf("provider switch for harness %q is not implemented yet — Claude/Codex/Grok/Pi/OpenCode/WorkBuddy/KimiCode/Vibe/Hermes/OhMyPi/Droid/Qoder/MiniMaxCode are available; add a SwitchWriter for %s (see internal/provider/stubdialects.go for what a stub is missing)", harness, harness)
}
