package mcp

import "strings"

// Wire formats.
const (
	FormatJSON = "json"
	FormatTOML = "toml"
	FormatYAML = "yaml"
)

// Translation dialects. One per MCP-capable harness family.
//
// Source: mixed | each dialect is the *target tool's* own field names |
//
//	checked 2026-09-13 | see per-style tags in Translate below and per-row tags in target.go
//
// Never: "clean up" two dialects that look redundant (e.g. StyleTrae /
// StyleWorkBuddy / StyleGeneric, or StyleCursor reused for droid). They stay
// separate because each one is someone else's file contract; merging them is how a
// refactor silently writes a key the target tool will ignore. Add a new Style
// constant only when a harness disagrees with every existing dialect.
const (
	StyleClaude      = "claude"
	StyleCodex       = "codex"
	StyleGrok        = "grok"
	StyleOpenCode    = "opencode"
	StylePi          = "pi"
	StyleGemini      = "gemini"
	StyleCopilot     = "copilot"
	StyleWorkBuddy   = "workbuddy"
	StyleTrae        = "trae"
	StyleGeneric     = "generic"
	StyleCursor      = "cursor"
	StyleAntigravity = "antigravity"
	StyleKimi        = "kimi"
	StyleQoder       = "qoder"
	StyleCline       = "cline"
	StyleKiro        = "kiro"
	StyleZCode       = "zcode"
	StyleZed         = "zed"
	StyleOMP         = "omp"
	StyleHermes      = "hermes"
	StyleDimAgent    = "dimagent"
	StyleOpenClaw    = "openclaw"
)

// Translate converts one canonical server into a harness entry.
// ok=false with a reason means the harness cannot express this server
// (logged as n/a, never written).
//
// Source: mixed | per-case `Source:` tags below | checked 2026-09-13
//
// Invariant: the input Server is Magic's canonical shape (mcp.go); this function
// only ever *lowers* it to a vendor dialect. Never add a Magic-only field to the
// output entry — the harness will not understand it. Vendor-specific extras belong
// in the library file and pass through Server, not here.
func Translate(s Server, style string) (entry map[string]any, ok bool, reason string) {
	remote := s.IsRemote()
	enabled := s.IsEnabled()

	switch style {
	case StyleClaude:
		// Source: vendor-doc | code.claude.com/docs/en/settings (remote needs `type` + `url`)
		//   + local-measure ~/.claude.json | checked 2026-09-13 | verified
		if remote {
			t := "http"
			if s.TransportKind() == TransportSSE {
				t = "sse"
			}
			e := map[string]any{"type": t, "url": s.URL}
			withHeaders(e, s.Headers)
			return e, true, ""
		}
		return stdioEntry(s, false), true, ""

	case StyleGemini:
		// Source: internal | docs/harness-profiles.md §Gemini/Antigravity (`httpUrl` for streamable
		//   HTTP, `url` for SSE) | checked 2026-09-13 | verified. Also reused by qwen-code (Gemini lineage).
		if remote {
			e := map[string]any{}
			if s.TransportKind() == TransportSSE {
				e["url"] = s.URL
			} else {
				e["httpUrl"] = s.URL
			}
			withHeaders(e, s.Headers)
			return e, true, ""
		}
		return stdioEntry(s, true), true, ""

	case StyleCodex:
		// Source: vendor-doc | developers.openai.com/codex/config-reference ([mcp_servers.<name>],
		//   `url` + `http_headers`, no `type`) | checked 2026-09-13 | verified
		if remote {
			if s.TransportKind() == TransportSSE {
				return nil, false, "Codex does not support SSE remote MCP (use an HTTP proxy)"
			}
			e := map[string]any{"url": s.URL}
			if len(s.Headers) > 0 {
				e["http_headers"] = toAnyMap(s.Headers)
			}
			e["enabled"] = enabled
			return e, true, ""
		}
		e := stdioEntry(s, true)
		e["enabled"] = enabled
		return e, true, ""

	case StyleGrok:
		// Source: local-measure | ~/.grok/config.toml [mcp_servers] | measured 2026-09-13 | verified
		// No public format doc (xAI-internal CLI) — confirm on a real install, not from memory.
		if remote {
			e := map[string]any{"url": s.URL}
			withHeaders(e, s.Headers)
			return e, true, ""
		}
		return stdioEntry(s, true), true, ""

	case StyleOpenCode:
		// Source: vendor-doc | opencode.ai/docs/mcp-servers (type "local"|"remote", `command` is
		//   an ARRAY, `environment`, `enabled`) | checked 2026-09-13 | verified
		e := map[string]any{"enabled": enabled}
		if remote {
			e["type"] = "remote"
			e["url"] = s.URL
			withHeaders(e, s.Headers)
			return e, true, ""
		}
		e["type"] = "local"
		cmd := make([]any, 0, len(s.Args)+1)
		if strings.TrimSpace(s.Command) != "" {
			cmd = append(cmd, s.Command)
		}
		for _, a := range s.Args {
			cmd = append(cmd, a)
		}
		if len(cmd) > 0 {
			e["command"] = cmd
		}
		if len(s.Env) > 0 {
			e["environment"] = toAnyMap(s.Env)
		}
		return e, true, ""

	case StylePi:
		// Source: upstream | github.com/nicobailon/pi-mcp-adapter (stdio `mcpServers` + `disabled`)
		//   + local-measure ~/.pi/agent/mcp.json | checked 2026-09-13 | verified
		if remote {
			return nil, false, "Pi remote MCP needs the third-party pi-mcp-adapter"
		}
		e := stdioEntry(s, true)
		e["disabled"] = !enabled
		return e, true, ""

	case StyleCopilot:
		// Source: vendor-doc | docs.github.com/.../customize-copilot/add-mcp-servers
		//   (`type:"local"` for stdio, `type:"http"|"sse"` for remote) | checked 2026-09-13 | verified
		if remote {
			t := "http"
			if s.TransportKind() == TransportSSE {
				t = "sse"
			}
			e := map[string]any{"type": t, "url": s.URL}
			withHeaders(e, s.Headers)
			return e, true, ""
		}
		e := stdioEntry(s, true)
		e["type"] = "local"
		return e, true, ""

	case StyleTrae, StyleWorkBuddy, StyleGeneric:
		// Source: vendor-doc | docs.trae.ai/ide/skills · codebuddy.ai/docs/cli/mcp | checked 2026-09-13 | verified
		// Claude-ish `{command,args,env}` / `{url,headers}` with no `type` discriminator.
		// StyleGeneric (mcpuser / iflow-cli) is the plain MCP shape, not a placeholder for "unknown";
		// a new harness must be matched against a real dialect instead of defaulting here.
		if remote {
			e := map[string]any{"url": s.URL}
			withHeaders(e, s.Headers)
			return e, true, ""
		}
		return stdioEntry(s, true), true, ""

	case StyleCursor:
		// Source: vendor-doc | cursor.com/en/docs/mcp (remote = bare `url`, `type` NOT required)
		//   + local-measure ~/.cursor/mcp.json | checked 2026-09-13 | verified
		if remote {
			e := map[string]any{"url": s.URL}
			withHeaders(e, s.Headers)
			return e, true, ""
		}
		return stdioEntry(s, false), true, ""

	case StyleAntigravity:
		// Source: internal | docs/harness-profiles.md §6 Antigravity — remote key is **`serverUrl`**,
		//   not `url`/`httpUrl` | checked 2026-09-13 | verified
		if remote {
			e := map[string]any{"serverUrl": s.URL}
			withHeaders(e, s.Headers)
			return e, true, ""
		}
		return stdioEntry(s, false), true, ""

	case StyleKimi:
		// Source: vendor-doc | kimi.com/code/docs/en/kimi-code-cli/customization/mcp.html
		//   (`{url}` or `{transport:"sse", url}`, `enabledTools`) | checked 2026-09-13 | verified
		if remote {
			e := map[string]any{"url": s.URL}
			if s.TransportKind() == TransportSSE {
				e["transport"] = "sse"
			}
			withHeaders(e, s.Headers)
			return e, true, ""
		}
		return stdioEntry(s, true), true, ""

	case StyleQoder:
		// Source: vendor-doc | docs.qoder.com/cli/mcp-servers · /cli/settings-reference
		//   (Claude-compatible `type: http|sse`) | checked 2026-09-13 | verified
		if remote {
			t := "http"
			if s.TransportKind() == TransportSSE {
				t = "sse"
			}
			e := map[string]any{"type": t, "url": s.URL}
			withHeaders(e, s.Headers)
			return e, true, ""
		}
		return stdioEntry(s, false), true, ""

	case StyleCline:
		// Source: vendor-doc | docs.cline.bot/mcp/configuring-mcp-servers — Cline uses
		//   `streamableHttp`, a transport name no other dialect uses | checked 2026-09-13 | verified
		if remote {
			t := "streamableHttp"
			if s.TransportKind() == TransportSSE {
				t = "sse"
			}
			e := map[string]any{"type": t, "url": s.URL}
			withHeaders(e, s.Headers)
			return e, true, ""
		}
		return stdioEntry(s, false), true, ""

	case StyleKiro:
		// Source: vendor-doc | kiro.dev/docs/mcp (confirms `mcpServers` + `disabled`) |
		//   checked 2026-09-13 | verified
		// `disabled` (inverted) rather than `enabled` — keep the inversion, Kiro reads it this way.
		if remote {
			e := map[string]any{"url": s.URL}
			withHeaders(e, s.Headers)
			e["disabled"] = !enabled
			return e, true, ""
		}
		e := stdioEntry(s, false)
		e["disabled"] = !enabled
		return e, true, ""

	case StyleZCode:
		// Source: internal | docs/supported-harnesses.md §调研来源 (ZCode) — `enable`, not `enabled`
		//   | checked 2026-09-13 | stale?
		e := map[string]any{"enable": enabled}
		if remote {
			t := "http"
			if s.TransportKind() == TransportSSE {
				t = "sse"
			}
			e["type"] = t
			e["url"] = s.URL
			withHeaders(e, s.Headers)
			return e, true, ""
		}
		for k, v := range stdioEntry(s, false) {
			e[k] = v
		}
		return e, true, ""

	case StyleZed:
		// Source: internal | docs/supported-harnesses.md §调研来源 (Zed) | checked 2026-09-13 | stale?
		// stdio only — Zed has no native HTTP MCP transport. zed.dev MCP doc URLs currently 404;
		// find a live page (or a real ~/.config/zed/settings.json) before changing this case.
		if remote {
			return nil, false, "Zed has no native HTTP MCP transport (use the mcp-remote bridge)"
		}
		return stdioEntry(s, false), true, ""

	case StyleOMP:
		// Source: upstream | omp docs/mcp-config.md (`mcpServers` + `enabled`, stdio and remote)
		//   see internal/adapters/omp | checked 2026-09-13 | verified
		if remote {
			e := map[string]any{"url": s.URL, "enabled": enabled}
			withHeaders(e, s.Headers)
			if s.TransportKind() == TransportSSE {
				e["type"] = "sse"
			}
			return e, true, ""
		}
		e := stdioEntry(s, true)
		e["enabled"] = enabled
		return e, true, ""

	case StyleHermes:
		// Source: upstream | hermes website/docs/reference/mcp-config-reference.md
		//   mcp_servers.{command,args,env,url,headers,enabled} | checked 2026-09-13 | verified
		// Only dialect written into YAML (FormatYAML) — see yaml.go.
		if remote {
			e := map[string]any{"url": s.URL, "enabled": enabled}
			withHeaders(e, s.Headers)
			return e, true, ""
		}
		e := stdioEntry(s, true)
		e["enabled"] = enabled
		return e, true, ""

	case StyleDimAgent:
		// Source: vendor-doc | dimagent.com/en/docs/config | checked 2026-09-13 | stale?
		// stdio {command,args,env,enabled}; remote {type:"http",url,headers,enabled}.
		// SSE is rejected with a warning by the CLI, so skip it rather than writing a dead entry.
		if remote {
			kind := s.TransportKind()
			if kind == TransportSSE {
				return nil, false, "DimAgent only supports http remote MCP (sse is skipped)"
			}
			e := map[string]any{"type": "http", "url": s.URL, "enabled": enabled}
			withHeaders(e, s.Headers)
			return e, true, ""
		}
		e := stdioEntry(s, true)
		e["enabled"] = enabled
		return e, true, ""

	case StyleOpenClaw:
		// Source: internal | docs/supported-harnesses.md §第三轮调研 (OpenClaw) — `mcp.servers.<name>`
		//   with command/args or url/transport (`streamable-http` is hyphenated here, unlike
		//   Cline's `streamableHttp`) | checked 2026-09-13 | stale?
		if remote {
			t := "streamable-http"
			if s.TransportKind() == TransportSSE {
				t = "sse"
			}
			e := map[string]any{"url": s.URL, "transport": t}
			withHeaders(e, s.Headers)
			return e, true, ""
		}
		return stdioEntry(s, true), true, ""

	default:
		// Unknown dialect: refuse, never fall back to a generic shape. Writing the
		// MCP-config-of-the-month into an unrecognised harness's file is worse than
		// skipping it (see the Style block comment above).
		return nil, false, "unknown MCP dialect " + style
	}
}

// stdioEntry builds a {command,args,env[,cwd]} entry.
func stdioEntry(s Server, withCwd bool) map[string]any {
	e := map[string]any{}
	if strings.TrimSpace(s.Command) != "" {
		e["command"] = s.Command
	}
	if len(s.Args) > 0 {
		e["args"] = toAnySlice(s.Args)
	}
	if len(s.Env) > 0 {
		e["env"] = toAnyMap(s.Env)
	}
	if withCwd && strings.TrimSpace(s.Cwd) != "" {
		e["cwd"] = s.Cwd
	}
	return e
}

func withHeaders(e map[string]any, headers map[string]string) {
	if len(headers) > 0 {
		e["headers"] = toAnyMap(headers)
	}
}

func toAnyMap(m map[string]string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func toAnySlice(in []string) []any {
	out := make([]any, 0, len(in))
	for _, s := range in {
		out = append(out, s)
	}
	return out
}
