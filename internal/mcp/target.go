package mcp

import (
	"os"
	"path/filepath"

	"magic/internal/appdir"
)

// Target describes where and how one harness stores MCP servers.
type Target struct {
	Harness string
	// Style selects the translation dialect (see translate.go).
	Style string
	// Format is "json" or "toml".
	Format string
	// TopKey is the JSON object key or the TOML section name.
	TopKey string
	// LenientJSON skips (instead of failing) when the file is not plain
	// JSON/JSONC — e.g. OpenClaw's JSON5-ish config. The file is left untouched.
	LenientJSON bool
	// Path resolves the config file for a user home.
	Path func(userHome string) string
}

// target table — one row per MCP-capable harness (docs/mcp-design.md).
//
// Source: mixed | see per-row tags below | checked 2026-09-13
//
//	Row tags use the format from docs/comment-contract.md. A row without its own
//	tag inherits the default below: research record in docs/harness-profiles.md
//	(official docs / local measurement, verification marker kept there as ✅/❓).
//	Never: treat a row as "just a path". Path + Format + TopKey + Style are the target
//	harness's own file contract; getting one wrong writes an invalid config into a user's
//	installed tool, and Magic has no rollback for that. Verify a changed row against the
//	evidence its tag cites (local-measure = look at the real file; vendor-doc = read that
//	page), then update the row's `checked` date.
var targets = map[string]Target{
	"claude": {
		// Source: local-measure | ~/.claude.json (mcpServers) | measured 2026-09-13 | verified
		// Source: vendor-doc | code.claude.com/docs/en/settings | checked 2026-09-13 | verified
		Harness: "claude", Style: StyleClaude, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".claude.json") },
	},
	"codex": {
		// Source: vendor-doc | developers.openai.com/codex/config-reference ([mcp_servers.<name>]) | checked 2026-09-13 | verified
		// Source: local-measure | ~/.codex/config.toml | measured 2026-09-13 | verified
		Harness: "codex", Style: StyleCodex, Format: FormatTOML, TopKey: "mcp_servers",
		Path: func(h string) string { return filepath.Join(h, ".codex", "config.toml") },
	},
	"grok": {
		// Source: local-measure | ~/.grok/config.toml [mcp_servers] | measured 2026-09-13 | verified
		// No public format doc; xAI-internal CLI. Confirm against a real install before editing.
		Harness: "grok", Style: StyleGrok, Format: FormatTOML, TopKey: "mcp_servers",
		Path: func(h string) string { return filepath.Join(h, ".grok", "config.toml") },
	},
	"opencode": {
		// Source: vendor-doc | opencode.ai/docs/mcp-servers (key `mcp`, not `mcpServers`) | checked 2026-09-13 | verified
		// Source: vendor-doc | opencode.ai/config.json ($schema) | checked 2026-09-13 | verified
		// Source: upstream | opencode reads XDG_CONFIG_HOME for its config root (see internal/adapters/opencode) | checked 2026-11-14 | verified
		Harness: "opencode", Style: StyleOpenCode, Format: FormatJSON, TopKey: "mcp",
		Path: openCodePath,
	},
	"pi": {
		// Source: local-measure | ~/.pi/agent/mcp.json | measured 2026-09-13 | verified
		// Source: upstream | pi-coding-agent docs (mcp.json is read by pi-mcp-adapter, not Pi core) | verified
		Harness: "pi", Style: StylePi, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".pi", "agent", "mcp.json") },
	},
	"gemini": {
		// Source: internal | docs/harness-profiles.md §Gemini/Antigravity | checked 2026-09-13 | verified
		// Superseded by antigravity-cli; kept as a transition target only.
		Harness: "gemini", Style: StyleGemini, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".gemini", "settings.json") },
	},
	"copilot": {
		// Source: vendor-doc | docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers | checked 2026-09-13 | verified
		// Source: local-measure | ~/.copilot present on this Mac | measured 2026-09-13
		Harness: "copilot", Style: StyleCopilot, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".copilot", "mcp-config.json") },
	},
	"workbuddy": {
		// Source: vendor-doc | codebuddy.ai/docs/cli/mcp (page lists `.codebuddy/mcp.json`
		//   and `.codebuddy/.mcp.json`) | checked 2026-09-13 | stale?
		// Conflict to resolve before changing this row: docs/harness-profiles.md §10 records
		// `~/.codebuddy/settings.json` → `mcpServers` (Claude-shaped), the page above shows a
		// dedicated mcp.json. Both may be valid at different scopes. Verify on a real install.
		Harness: "workbuddy", Style: StyleWorkBuddy, Format: FormatJSON, TopKey: "mcpServers",
		Path: workBuddyPath,
	},
	"trae": {
		// Source: vendor-doc | docs.trae.ai/ide/skills · docs.trae.cn/ide_skills | checked 2026-09-13 | verified
		//   (user MCP file = ~/.trae/mcp.json, CN = ~/.trae-cn/mcp.json)
		// Magic deviation: only the user-level file is written; Trae IDE UI config and
		// Application Support/Trae/User are off-limits (login/UI state).
		Harness: "trae", Style: StyleTrae, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".trae", "mcp.json") },
	},
	"mcpuser": {
		// Source: internal | generic MCP config spec consumed by several tools (docs/mcp-design.md).
		// Not a harness and not a model provider — see AGENTS.md; no SwitchWriter.
		Harness: "mcpuser", Style: StyleGeneric, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".config", "mcp", "mcp.json") },
	},
	"cursor": {
		// Source: vendor-doc | cursor.com/en/docs/mcp | checked 2026-09-13 | verified
		// Source: local-measure | ~/.cursor/mcp.json | measured 2026-09-13 | verified
		Harness: "cursor", Style: StyleCursor, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".cursor", "mcp.json") },
	},
	"antigravity-cli": {
		// Source: internal | docs/harness-profiles.md §6 Antigravity (`serverUrl`, separate global file) | verified
		Harness: "antigravity-cli", Style: StyleAntigravity, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".gemini", "config", "mcp_config.json") },
	},
	"kimi-code": {
		// Source: vendor-doc | kimi.com/code/docs/en/kimi-code-cli/customization/mcp.html
		//   (`mcp.json`, `mcpServers`, `enabledTools`) | checked 2026-09-13 | verified
		Harness: "kimi-code", Style: StyleKimi, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".kimi-code", "mcp.json") },
	},
	"qoder": {
		// Source: vendor-doc | docs.qoder.com/cli/settings-reference · /cli/mcp-servers | checked 2026-09-13 | verified
		Harness: "qoder", Style: StyleQoder, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".qoder", "settings.json") },
	},
	"qoder-cn": {
		// Source: internal | docs/harness-profiles.md §9 Qoder CN / 通义灵码, same file shape as qoder
		Harness: "qoder-cn", Style: StyleQoder, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".qoder-cn", "settings.json") },
	},
	"cline": {
		// Source: vendor-doc | docs.cline.bot/mcp/configuring-mcp-servers (states ~/.cline/mcp.json)
		//   | checked 2026-09-13 | verified
		Harness: "cline", Style: StyleCline, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".cline", "mcp.json") },
	},
	"kiro": {
		// Source: vendor-doc | kiro.dev/docs/mcp (confirms `mcpServers` + `disabled`; the page does
		//   NOT state the file path) | checked 2026-09-13
		// Path below is still `stale?` — confirm ~/.kiro/settings/mcp.json on a real install.
		Harness: "kiro", Style: StyleKiro, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".kiro", "settings", "mcp.json") },
	},
	"droid": {
		// Source: internal | docs/supported-harnesses.md §调研来源 (Droid/Factory) | checked 2026-09-13 | stale?
		// Shares the Cursor dialect: `{url}` without a required `type`.
		Harness: "droid", Style: StyleCursor, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".factory", "mcp.json") },
	},
	"zcode": {
		// Source: internal | docs/supported-harnesses.md §调研来源 (ZCode) | checked 2026-09-13 | stale?
		// Nested key: `mcp.servers`, not a flat `mcpServers`.
		Harness: "zcode", Style: StyleZCode, Format: FormatJSON, TopKey: "mcp.servers",
		Path: func(h string) string { return filepath.Join(h, ".zcode", "cli", "config.json") },
	},
	"qwen-code": {
		// Source: internal | docs/supported-harnesses.md §调研来源 (Qwen Code) | checked 2026-09-13 | stale?
		// Gemini CLI lineage, so it reuses the Gemini dialect on purpose; confirm
		// ~/.qwen/settings.json against QwenLM/qwen-code docs before editing.
		Harness: "qwen-code", Style: StyleGemini, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".qwen", "settings.json") },
	},
	"iflow-cli": {
		// Source: vendor-doc | github.com/iflow-ai/iflow-cli docs_en/configuration/settings.md
		//   (`~/.iflow/settings.json`, `mcpServers`, `baseUrl`) | checked 2026-09-13 | verified
		Harness: "iflow-cli", Style: StyleGeneric, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".iflow", "settings.json") },
	},
	"kilo": {
		// Source: vendor-doc | kilo.ai/docs | checked 2026-09-13 | stale?
		// Docs confirm Kilo speaks the OpenCode `mcp` dialect; the exact config filename
		// varies across versions, hence kiloPath probes several candidates.
		// Source: vendor-blog | Kilo relocated its config to XDG_CONFIG_HOME (see internal/adapters/kilo) | checked 2026-11-14 | verified
		Harness: "kilo", Style: StyleOpenCode, Format: FormatJSON, TopKey: "mcp",
		Path: kiloPath,
	},
	"zed": {
		// Source: internal | docs/supported-harnesses.md §调研来源 (Zed) | checked 2026-09-13 | stale?
		// zed.dev MCP doc URLs currently 404 — do not cite one until a live page is found.
		// TopKey `context_servers` is Zed's own name; stdio only (no native HTTP transport).
		// Source: vendor-doc | zed.dev/docs/configuring-zed: Linux reads $XDG_CONFIG_HOME/zed/settings.json (see internal/adapters/zed) | checked 2026-11-14 | verified
		Harness: "zed", Style: StyleZed, Format: FormatJSON, TopKey: "context_servers",
		Path: zedPath,
	},
	"omp": {
		// Source: upstream | omp repo packages/utils/src/dirs.ts, docs/settings.md, docs/mcp-config.md
		//   (already recorded in internal/adapters/omp) | checked 2026-09-13 | verified
		Harness: "omp", Style: StyleOMP, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".omp", "agent", "mcp.json") },
	},
	"hermes": {
		// Source: upstream | hermes_cli/config.py (YAML config, snake_case `mcp_servers`)
		//   see internal/adapters/hermes | checked 2026-09-13 | verified
		// Only YAML target in this table (see yaml.go).
		Harness: "hermes", Style: StyleHermes, Format: FormatYAML, TopKey: "mcp_servers",
		Path: func(h string) string { return filepath.Join(h, ".hermes", "config.yaml") },
	},
	"mimocode": {
		// Source: upstream | mimocode packages/shared/src/global.ts; OpenCode dialect
		//   see internal/adapters/mimocode | checked 2026-09-13 | verified
		// Source: upstream | resolveMimocodeHome(): $MIMOCODE_HOME/config outranks $XDG_CONFIG_HOME/mimocode
		//   (see internal/adapters/mimocode) | checked 2026-11-14 | verified
		Harness: "mimocode", Style: StyleOpenCode, Format: FormatJSON, TopKey: "mcp",
		Path: mimoCodePath,
	},
	"dimagent": {
		// Source: vendor-doc | dimagent.com/en/docs/config (see internal/adapters/dimagent) | checked 2026-09-13 | verified
		Harness: "dimagent", Style: StyleDimAgent, Format: FormatJSON, TopKey: "mcpServers",
		Path: func(h string) string { return filepath.Join(h, ".dim", "v2", "mcp.json") },
	},
	"openclaw": {
		// Source: internal | docs/supported-harnesses.md §第三轮调研 (OpenClaw) | checked 2026-09-13 | stale?
		// LenientJSON: openclaw.json is JSON5-ish (comments/trailing commas), so strict
		// encoding would destroy user content. Never: rewrite it with a plain json.Marshal.
		Harness: "openclaw", Style: StyleOpenClaw, Format: FormatJSON, TopKey: "mcp.servers",
		LenientJSON: true,
		Path:        func(h string) string { return filepath.Join(h, ".openclaw", "openclaw.json") },
	},
}

// TargetFor returns the MCP target for a harness id.
func TargetFor(harness string) (Target, bool) {
	t, ok := targets[harness]
	return t, ok
}

// Targets returns all targets sorted by harness.
func Targets() []Target {
	out := make([]Target, 0, len(targets))
	for _, t := range targets {
		out = append(out, t)
	}
	sortTargets(out)
	return out
}

func kiloPath(home string) string {
	dir := filepath.Join(appdir.ConfigHome(home), "kilo")
	for _, name := range []string{"kilo.json", "kilo.jsonc", "config.json"} {
		p := filepath.Join(dir, name)
		if fileExists(p) {
			return p
		}
	}
	return filepath.Join(dir, "kilo.json")
}

func openCodePath(home string) string {
	dir := filepath.Join(appdir.ConfigHome(home), "opencode")
	json := filepath.Join(dir, "opencode.json")
	jsonc := filepath.Join(dir, "opencode.jsonc")
	if fileExists(json) {
		return json
	}
	if fileExists(jsonc) {
		return jsonc
	}
	return json
}

func zedPath(home string) string {
	return filepath.Join(appdir.ConfigHome(home), "zed", "settings.json")
}

// mimoCodePath repeats the adapter's precedence on purpose: an imported adapter
// would make this leaf table depend on the dialect packages.
func mimoCodePath(home string) string {
	if v := os.Getenv("MIMOCODE_HOME"); v != "" {
		return filepath.Join(filepath.Clean(v), "config", "mimocode.json")
	}
	return filepath.Join(appdir.ConfigHome(home), "mimocode", "mimocode.json")
}

func workBuddyPath(home string) string {
	cb := filepath.Join(home, ".codebuddy", "settings.json")
	if st, err := os.Stat(filepath.Dir(cb)); err == nil && st.IsDir() {
		return cb
	}
	if fileExists(cb) {
		return cb
	}
	wb := filepath.Join(home, ".workbuddy", "settings.json")
	if st, err := os.Stat(filepath.Dir(wb)); err == nil && st.IsDir() {
		return wb
	}
	return cb
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func sortTargets(ts []Target) {
	for i := 1; i < len(ts); i++ {
		for j := i; j > 0 && ts[j-1].Harness > ts[j].Harness; j-- {
			ts[j-1], ts[j] = ts[j], ts[j-1]
		}
	}
}
