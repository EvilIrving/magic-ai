package catalog

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Protocol string constants used in providers.yaml endpoints.
// Canonical forms:
//
//	openai-chat-completions — OpenAI Chat Completions API
//	openai-responses        — OpenAI Responses API
//	anthropic-messages      — Anthropic Messages API
//
// Aliases accepted by NormalizeProtocol:
//
//	openai, openai-completions, openai-chat, chat-completions → openai-chat-completions
//	responses                                                 → openai-responses
//	anthropic, messages                                       → anthropic-messages
const (
	ProtocolOpenAIChatCompletions = "openai-chat-completions"
	ProtocolOpenAIResponses       = "openai-responses"
	ProtocolAnthropicMessages     = "anthropic-messages"
)

// ExampleProvidersYAML is written on init when providers.yaml is missing.
const ExampleProvidersYAML = `# providers.yaml — user provider bindings + model picks (non-secret)
# API keys live in encrypted $MAGIC_HOME/secrets/providers.json.enc,
# indexed by key_env (logical name). Never store secret values here.
# magic copy resolves keys via MAGIC_PASSPHRASE — not from process env.
#
# active: default third-party provider id (magic apply + magic provider use company).
# Override with env MAGIC_ACTIVE_PROVIDER. "company" is only this alias — not a vendor id.
#
# Preferred shape: endpoints[] with id / base_url / protocol.
# Backward compat: if endpoints empty, top-level base_url+protocol is one endpoint.

# active: shu

providers:
  deepseek:
    base_url: https://api.deepseek.com/v1
    protocol: openai
    key_env: DEEPSEEK_API_KEY
    # key value never stored here — set with: magic secrets set DEEPSEEK_API_KEY --value …
    pick:
      - deepseek-chat
      - deepseek-reasoner

  # Multi-protocol relay example (same key_env + pick, several endpoints):
  # shu:
  #   key_env: SHU_API_KEY
  #   pick: [gpt-5.6-sol]
  #   endpoints:
  #     - id: openai-responses
  #       base_url: https://api.shu.cool/v1
  #       protocol: openai-responses
  #     - id: openai-chat
  #       base_url: https://api.shu.cool/v1
  #       protocol: openai-chat-completions
  #     - id: anthropic
  #       base_url: https://api.shu.cool
  #       protocol: anthropic-messages
`

// Endpoint is one protocol-specific base URL under a provider.
type Endpoint struct {
	ID       string `yaml:"id,omitempty"`
	BaseURL  string `yaml:"base_url"`
	Protocol string `yaml:"protocol"`
}

// Provider is one user-configured API binding (possibly multi-endpoint).
type Provider struct {
	BaseURL   string            `yaml:"base_url,omitempty"`
	Protocol  string            `yaml:"protocol,omitempty"`
	KeyEnv    string            `yaml:"key_env"`
	Pick      []string          `yaml:"pick"`
	Endpoints []Endpoint        `yaml:"endpoints,omitempty"`
	// Models is optional role→model map for Claude Code projection
	// (haiku/sonnet/opus/default/reasoning). Unused by other harnesses.
	Models map[string]string `yaml:"models,omitempty"`
}

// ModelMeta is optional registry-sourced metadata for a model id.
type ModelMeta struct {
	Name          string `yaml:"name,omitempty"`
	ContextLength int64  `yaml:"context_length,omitempty"`
	Source        string `yaml:"source,omitempty"`
}

// File is the providers.yaml root.
type File struct {
	Active     string               `yaml:"active,omitempty"`
	Providers  map[string]Provider  `yaml:"providers"`
	ModelsMeta map[string]ModelMeta `yaml:"models_meta,omitempty"`
}

// NormalizeProtocol maps aliases to canonical protocol strings.
func NormalizeProtocol(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "openai", "openai-completions", "openai-chat", "openai-chat-completions", "chat-completions":
		return ProtocolOpenAIChatCompletions
	case "openai-responses", "responses":
		return ProtocolOpenAIResponses
	case "anthropic", "anthropic-messages", "messages":
		return ProtocolAnthropicMessages
	default:
		return strings.ToLower(strings.TrimSpace(s))
	}
}

// ProtocolsMatch reports whether a and b refer to the same canonical protocol.
func ProtocolsMatch(a, b string) bool {
	na, nb := NormalizeProtocol(a), NormalizeProtocol(b)
	if na == "" || nb == "" {
		return false
	}
	return na == nb
}

// NormalizeEndpoints returns provider endpoints. If Endpoints is non-empty those
// are used (protocols normalized). Otherwise a single endpoint is synthesized
// from top-level BaseURL+Protocol when either is set.
func NormalizeEndpoints(p Provider) []Endpoint {
	if len(p.Endpoints) > 0 {
		out := make([]Endpoint, 0, len(p.Endpoints))
		for i, ep := range p.Endpoints {
			id := strings.TrimSpace(ep.ID)
			if id == "" {
				id = fmt.Sprintf("ep-%d", i+1)
			}
			out = append(out, Endpoint{
				ID:       id,
				BaseURL:  strings.TrimSpace(ep.BaseURL),
				Protocol: NormalizeProtocol(ep.Protocol),
			})
		}
		return out
	}
	base := strings.TrimSpace(p.BaseURL)
	proto := NormalizeProtocol(p.Protocol)
	if base == "" && proto == "" {
		return nil
	}
	id := proto
	if id == "" {
		id = "default"
	}
	return []Endpoint{{ID: id, BaseURL: base, Protocol: proto}}
}

// PickEndpoint returns the first endpoint whose protocol matches any of wantProtocols
// (aliases accepted). Empty wantProtocols → first normalized endpoint.
func PickEndpoint(p Provider, wantProtocols ...string) (Endpoint, bool) {
	eps := NormalizeEndpoints(p)
	if len(eps) == 0 {
		return Endpoint{}, false
	}
	if len(wantProtocols) == 0 {
		return eps[0], true
	}
	for _, want := range wantProtocols {
		want = strings.TrimSpace(want)
		if want == "" {
			continue
		}
		for _, ep := range eps {
			if ProtocolsMatch(ep.Protocol, want) {
				return ep, true
			}
		}
	}
	return Endpoint{}, false
}

// PickEndpointByID returns the endpoint with the given id (case-sensitive trim).
func PickEndpointByID(p Provider, id string) (Endpoint, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Endpoint{}, false
	}
	for _, ep := range NormalizeEndpoints(p) {
		if ep.ID == id {
			return ep, true
		}
	}
	return Endpoint{}, false
}

// ResolveCopyEndpoint selects an endpoint for magic copy.
// Priority: endpointID → protocol → first normalized endpoint.
func ResolveCopyEndpoint(p Provider, endpointID, protocol string) (Endpoint, error) {
	if id := strings.TrimSpace(endpointID); id != "" {
		ep, ok := PickEndpointByID(p, id)
		if !ok {
			return Endpoint{}, fmt.Errorf("endpoint id %q not found", id)
		}
		return ep, nil
	}
	if proto := strings.TrimSpace(protocol); proto != "" {
		ep, ok := PickEndpoint(p, proto)
		if !ok {
			return Endpoint{}, fmt.Errorf("no endpoint with protocol %q", proto)
		}
		return ep, nil
	}
	ep, ok := PickEndpoint(p)
	if !ok {
		return Endpoint{}, fmt.Errorf("provider has no endpoints (set endpoints[] or base_url/protocol)")
	}
	return ep, nil
}

// ActiveProviderName returns the configured active provider hint.
// Order: MAGIC_ACTIVE_PROVIDER env, then File.Active.
func ActiveProviderName(f *File) string {
	if v := strings.TrimSpace(os.Getenv("MAGIC_ACTIVE_PROVIDER")); v != "" {
		return v
	}
	if f != nil {
		return strings.TrimSpace(f.Active)
	}
	return ""
}

// ResolveProviderForProtocols picks a provider that has a matching endpoint.
// If hint (active) is set, that provider must have a matching endpoint.
// Else if exactly one provider matches, use it; if multiple, return ok=false with a message.
func ResolveProviderForProtocols(f *File, hint string, wantProtocols ...string) (name string, p Provider, ep Endpoint, ok bool, msg string) {
	if f == nil || len(f.Providers) == 0 {
		return "", Provider{}, Endpoint{}, false, "no providers in providers.yaml"
	}
	hint = strings.TrimSpace(hint)
	if hint != "" {
		prov, found := f.Providers[hint]
		if !found {
			return "", Provider{}, Endpoint{}, false, fmt.Sprintf("active provider %q not found in providers.yaml", hint)
		}
		ep, matched := PickEndpoint(prov, wantProtocols...)
		if !matched {
			return "", Provider{}, Endpoint{}, false, fmt.Sprintf("active provider %q has no endpoint matching %v", hint, wantProtocols)
		}
		return hint, prov, ep, true, ""
	}

	type hit struct {
		name string
		p    Provider
		ep   Endpoint
	}
	var hits []hit
	for n, prov := range f.Providers {
		if ep, matched := PickEndpoint(prov, wantProtocols...); matched {
			hits = append(hits, hit{n, prov, ep})
		}
	}
	if len(hits) == 0 {
		return "", Provider{}, Endpoint{}, false, fmt.Sprintf("no provider has endpoint matching %v", wantProtocols)
	}
	if len(hits) > 1 {
		names := make([]string, len(hits))
		for i, h := range hits {
			names[i] = h.name
		}
		return "", Provider{}, Endpoint{}, false, fmt.Sprintf("multiple providers match %v: %s — set providers.active or MAGIC_ACTIVE_PROVIDER", wantProtocols, strings.Join(names, ", "))
	}
	return hits[0].name, hits[0].p, hits[0].ep, true, ""
}

// ProtocolToPiAPI maps Magic protocol → Pi models.json "api" dialect.
func ProtocolToPiAPI(protocol string) string {
	switch NormalizeProtocol(protocol) {
	case ProtocolOpenAIResponses:
		return "openai-responses"
	case ProtocolOpenAIChatCompletions:
		return "openai-completions"
	case ProtocolAnthropicMessages:
		return "anthropic-messages"
	default:
		return NormalizeProtocol(protocol)
	}
}

// Load reads providers.yaml; missing → empty.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{Providers: map[string]Provider{}}, nil
		}
		return nil, err
	}
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.Providers == nil {
		f.Providers = map[string]Provider{}
	}
	return &f, nil
}

// Save writes providers.yaml.
func Save(path string, f *File) error {
	if f == nil {
		f = &File{Providers: map[string]Provider{}}
	}
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// AddPick appends modelID to provider's pick if not already present.
// Returns true if the pick list changed.
func (f *File) AddPick(provider, modelID string) (bool, error) {
	if f.Providers == nil {
		f.Providers = map[string]Provider{}
	}
	p, ok := f.Providers[provider]
	if !ok {
		return false, fmt.Errorf("provider %q not found in providers.yaml", provider)
	}
	for _, m := range p.Pick {
		if m == modelID {
			return false, nil
		}
	}
	p.Pick = append(p.Pick, modelID)
	f.Providers[provider] = p
	return true, nil
}

// SetModelMeta stores optional metadata for a model id.
func (f *File) SetModelMeta(modelID string, meta ModelMeta) {
	if f.ModelsMeta == nil {
		f.ModelsMeta = map[string]ModelMeta{}
	}
	f.ModelsMeta[modelID] = meta
}

// FormatList prints catalog for CLI.
func FormatList(f *File) string {
	if f == nil || len(f.Providers) == 0 {
		return "(no providers in providers.yaml — run magic init or add entries)\n"
	}
	var b strings.Builder
	if a := ActiveProviderName(f); a != "" {
		fmt.Fprintf(&b, "active: %s\n", a)
	}
	b.WriteString("PROVIDER\tENDPOINT\tBASE_URL\tPROTOCOL\tKEY_ENV\tPICK\n")
	for name, p := range f.Providers {
		pick := strings.Join(p.Pick, ",")
		eps := NormalizeEndpoints(p)
		if len(eps) == 0 {
			fmt.Fprintf(&b, "%s\t-\t%s\t%s\t%s\t%s\n", name, p.BaseURL, p.Protocol, p.KeyEnv, pick)
			continue
		}
		for _, ep := range eps {
			fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\t%s\n", name, ep.ID, ep.BaseURL, ep.Protocol, p.KeyEnv, pick)
		}
	}
	return b.String()
}

// RedactKey shows first 4 … last 4 of a key (default). Short keys → ****.
func RedactKey(key string) string {
	if key == "" {
		return ""
	}
	r := []rune(key)
	if len(r) < 8 {
		return "****"
	}
	return string(r[:4]) + "…" + string(r[len(r)-4:])
}

// CopyText returns provider fields + api_key for clipboard/stdout using the
// first normalized endpoint. Prefer CopyTextEndpoint when a specific endpoint
// was selected (--protocol / --endpoint).
// apiKey is the already-resolved secret (from secrets/providers.json.enc);
// empty means missing → placeholder <KEY_ENV> plus a one-line note.
// Formatting is pure: no env or secrets I/O here.
func CopyText(name string, p Provider, apiKey string, revealKey bool) string {
	ep, ok := PickEndpoint(p)
	if !ok {
		ep = Endpoint{BaseURL: p.BaseURL, Protocol: NormalizeProtocol(p.Protocol)}
	}
	return CopyTextEndpoint(name, p, ep, apiKey, revealKey)
}

// CopyTextEndpoint formats copy output for a specific endpoint.
func CopyTextEndpoint(name string, p Provider, ep Endpoint, apiKey string, revealKey bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "provider: %s\n", name)
	if ep.ID != "" {
		fmt.Fprintf(&b, "endpoint: %s\n", ep.ID)
	}
	fmt.Fprintf(&b, "base_url: %s\n", ep.BaseURL)
	fmt.Fprintf(&b, "protocol: %s\n", ep.Protocol)
	if p.KeyEnv != "" {
		fmt.Fprintf(&b, "key_env: %s\n", p.KeyEnv)
	}

	keyPh := p.KeyEnv
	if keyPh == "" {
		keyPh = "YOUR_API_KEY"
	}
	if apiKey != "" {
		if revealKey {
			fmt.Fprintf(&b, "api_key: %s\n", apiKey)
		} else {
			fmt.Fprintf(&b, "api_key: %s\n", RedactKey(apiKey))
		}
	} else {
		fmt.Fprintf(&b, "api_key: <%s>\n", keyPh)
		fmt.Fprintf(&b, "# note: secret missing — set with: magic secrets set %s --value …\n", keyPh)
	}

	if len(p.Pick) > 0 {
		b.WriteString("models:\n")
		for _, m := range p.Pick {
			fmt.Fprintf(&b, "  - %s\n", m)
		}
	}
	return b.String()
}
