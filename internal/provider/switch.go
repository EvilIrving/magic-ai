package provider

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"magic/internal/catalog"
)

// Context carries paths and options for provider switch / status / test.
type Context struct {
	MagicHome string
	UserHome  string
	HTTP      *http.Client // nil → DefaultClient (tests inject httptest)
	Catalog   *catalog.File // optional preloaded; else LoadCatalog(MagicHome)
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

// SwitchWriter projects library selection into one harness's on-disk dialect.
// official → clear Magic-managed overrides / restore official path.
// provider → project selected provider's matching endpoint (+ secret when needed).
// Adding a harness = one SwitchWriter, not another shell script.
type SwitchWriter interface {
	Harness() string
	Implemented() bool
	// Apply writes official or third-party projection from a resolved ApplyRequest.
	Apply(ctx Context, req ApplyRequest) (msg string, err error)
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
	// gemini/copilot/trae: Adapter capture/apply only (no stable third-party dialect yet).
	// mcpuser is MCP config, not a model provider.
	for _, id := range []string{"gemini", "copilot", "trae"} {
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
		HarnessOpenCode, HarnessWorkBuddy, "gemini", "copilot", "trae",
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
	if rt.Mode == ModeOfficial {
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

// Use is the product entry: resolve target against library, then Apply on writer.
func Use(ctx Context, harness, rawTarget string) (string, error) {
	w := Writer(harness)
	if w == nil {
		return "", fmt.Errorf("unknown harness %q", harness)
	}
	if !w.Implemented() {
		return "", ErrNotImplemented(w.Harness())
	}
	h := NormalizeHarness(harness)
	needKey := h == HarnessClaude || h == HarnessOpenCode
	// Codex/Grok/WorkBuddy require key present for third-party but write env_key refs, not plaintext.
	if h == HarnessCodex || h == HarnessGrok || h == HarnessWorkBuddy {
		f, err := ctx.File()
		if err != nil {
			return "", err
		}
		rt, err := ResolveUseTarget(f, rawTarget)
		if err != nil {
			return "", err
		}
		if rt.Mode == ModeProvider {
			keyEnv := strings.TrimSpace(rt.Provider.KeyEnv)
			if keyEnv == "" {
				keyEnv = rt.Name
			}
			if _, err := ResolveSecret(ctx.SecretsDir(), keyEnv); err != nil {
				return "", err
			}
		}
	}
	req, err := BuildApplyRequest(ctx, harness, rawTarget, needKey)
	if err != nil {
		return "", err
	}
	return w.Apply(ctx, req)
}

// ErrNotImplemented is returned by stub writers.
func ErrNotImplemented(harness string) error {
	return fmt.Errorf("provider switch for harness %q is not implemented yet — Claude/Codex/Grok/Pi/OpenCode/WorkBuddy are available; add a SwitchWriter for %s", harness, harness)
}
