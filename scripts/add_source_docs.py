#!/usr/bin/env python3
"""One-off: insert contract-form `Source:` package docs into internal/adapters/*.

See docs/comment-contract.md. Idempotent: skips files whose package doc already
carries a `Source: <kind> |` tag.

Body convention:
  - body starting with ""  -> appended below the file's existing package doc
  - otherwise              -> the body IS the package doc (replaces any existing one)
"""
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
ADR = ROOT / "internal" / "adapters"

DOC = {
    "pi": [
        "// Package pi is the Magic adapter for Pi Agent (config root ~/.pi/agent).",
        "//",
        "// Source: local-measure | ~/.pi/agent/{skills,agents,mcp.json,models.json}",
        "//   on this Mac + pi-coding-agent docs | measured 2026-09-13 | verified",
        "//",
        "// Pi is Magic's primary harness. themes/ and AGENTS.md under this root are",
        "// per-harness user data (AGENTS.md -> 配置库布局): they travel in",
        "// packs/harness/pi/ so another machine restores the same Pi, and are NOT",
        "// translated into another harness's theme format. Never: capture files back",
        "// out of ~/.pi - the library is the only truth source (capture was deleted).",
    ],
    "grok": [
        "// Package grok is the Magic adapter for Grok CLI (config root ~/.grok).",
        "//",
        "// Source: local-measure | ~/.grok/config.toml on this Mac | measured 2026-09-13 | verified",
        "//",
        "// No public format doc exists (xAI-internal CLI): every path and key here is",
        "// evidence from a real install, not from memory. Re-measure before editing.",
    ],
    "claude": [
        "// Package claude is the Magic adapter for Claude Code.",
        "//",
        "// Source: local-measure | ~/.claude/{skills,agents} + ~/.claude.json | measured 2026-09-13 | verified",
        "// Source: vendor-doc | code.claude.com/docs/en/settings | checked 2026-09-13 | verified",
        "//",
        "// Two roots by design: ~/.claude holds per-item dirs (skills, agents) while",
        "// ~/.claude.json holds the user-level MCP map. Magic merges",
        "// packs/harness/claude/claude.json.partial.json into the latter by key",
        "// (field-level merge, docs/apply-design.md).",
    ],
    "codex": [
        "// Package codex is the Magic adapter for Codex CLI (config root ~/.codex).",
        "//",
        "// Source: local-measure | ~/.codex/{config.toml,skills} | measured 2026-09-13 | verified",
        "// Source: vendor-doc | developers.openai.com/codex/config-reference | checked 2026-09-13 | verified",
        "//",
        "// Codex keeps MCP and provider config in TOML, unlike the JSON crowd.",
        "// Third-party models go through Codex's own model_catalog_json + env_key",
        "// mechanism (docs/harness-matrix.md). Never: shell out to a curl|bash",
        "// installer - the DeepSeek official-integration pattern is already in place",
        "// (AGENTS.md -> 继续 registry 工作时 #5).",
    ],
    "opencode": [
        "// Package opencode is the Magic adapter for OpenCode (root ~/.config/opencode).",
        "//",
        "// Source: local-measure | ~/.config/opencode/opencode.json | measured 2026-09-13 | verified",
        "// Source: vendor-doc | opencode.ai/docs/mcp-servers · opencode.ai/docs/providers ·",
        "//   opencode.ai/config.json ($schema) | checked 2026-09-13 | verified",
        "//",
        "// Never: touch ~/.local/share/opencode/auth.json - OpenCode's own credential",
        "// store, outside Magic scope (docs/harness-matrix.md).",
    ],
    "copilot": [
        "// Package copilot is the Magic adapter for GitHub Copilot CLI (root ~/.copilot).",
        "//",
        "// Source: local-measure | ~/.copilot/skills on this Mac | measured 2026-09-13 | verified",
        "// Source: vendor-doc | docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers",
        "//   | checked 2026-09-13 | verified",
        "//",
        "// Never: mirror ~/.copilot/config.json - it carries auth/runtime state",
        "// (docs/harness-profiles.md §3).",
    ],
    "gemini": [
        "// Package gemini is the Magic adapter for Gemini CLI (config root ~/.gemini).",
        "//",
        "// Source: internal | docs/harness-profiles.md §Gemini/Antigravity | checked 2026-09-13 | verified",
        "//",
        "// Superseded: Gemini CLI was replaced by Antigravity CLI (decision D14). This",
        "// package stays only as a transition target for users who still have ~/.gemini;",
        "// new work goes to internal/adapters/antigravity.",
        "// Never: manage .env - Gemini auth frequently lives there, out of scope.",
    ],
    "workbuddy": [
        "// Package workbuddy is the Magic adapter for CodeBuddy / WorkBuddy (Tencent).",
        "//",
        "// Source: vendor-doc | codebuddy.ai/docs/cli/mcp | checked 2026-09-13 | stale?",
        "//   (that page lists .codebuddy/mcp.json and .codebuddy/.mcp.json;",
        "//   docs/harness-profiles.md §10 records settings.json -> mcpServers)",
        "//",
        "// One product, two names, shared ~/.codebuddy root; ~/.workbuddy is probed for",
        "// presence only. Resolve the MCP-file conflict above on a real install before",
        "// editing the workbuddy row in internal/mcp/target.go.",
    ],
    "trae": [
        "// Package trae is the Magic adapter for Trae (ByteDance), global and CN builds.",
        "//",
        "// Source: vendor-doc | docs.trae.ai/ide/skills · docs.trae.cn/ide_skills | checked 2026-09-13 | verified",
        "//   (skills ~/.trae/skills, CN ~/.trae-cn/skills; MCP ~/.trae/mcp.json)",
        "//",
        "// Never: mirror ~/Library/Application Support/Trae/User - IDE UI and login",
        "// state, not user config (docs/harness-profiles.md §8).",
    ],
    "mcpuser": [
        "// Package mcpuser is the MCP-config mirror (~/.config/mcp/mcp.json), not a harness.",
        "//",
        "// Source: internal | docs/mcp-design.md (generic MCP consumer) | verified",
        "//",
        "// mcpuser is deliberately NOT a model provider: no SwitchWriter, and it must",
        "// not grow a stub pretending to be one (AGENTS.md -> 继续 registry 工作时 #6).",
    ],
    "cursor": [
        "// Package cursor is the Magic adapter for Cursor (IDE + CLI).",
        "// Product decision D20: Cursor was promoted from skills-only to a full adapter",
        "// (usage rank #4). Cursor model access is account/UI driven, so there is no",
        "// model SwitchWriter here.",
        "//",
        "// Source: local-measure | ~/.cursor/{mcp.json,skills} | measured 2026-09-13 | verified",
        "// Source: vendor-doc | cursor.com/en/docs/mcp | checked 2026-09-13 | verified",
    ],
    "antigravity": [
        "",
        "// Source: internal | docs/harness-profiles.md §6 Antigravity | checked 2026-09-13 | verified",
        "//   Remote MCP key is `serverUrl`, not `url`/`httpUrl` - the one dialect that",
        "//   deliberately differs from Gemini's. The global MCP file sits outside the",
        "//   CLI config root, in ~/.gemini/config/mcp_config.json.",
    ],
    "cline": [
        "",
        "// Source: vendor-doc | docs.cline.bot/mcp/configuring-mcp-servers",
        "//   (states ~/.cline/mcp.json) | checked 2026-09-13 | verified",
        "//   Skills use the shared ~/.agents/skills dir, not a Cline-owned directory.",
    ],
    "droid": [
        "",
        "// Source: internal | docs/supported-harnesses.md §调研来源 (Factory Droid) | checked 2026-09-13 | stale?",
        "//   factory.ai/docs does not publish these paths - confirm on a real install.",
        "//   Custom agents live in `droids/`, and the root is ~/.factory, not ~/.droid.",
    ],
    "iflow": [
        "",
        "// Source: vendor-doc | github.com/iflow-ai/iflow-cli docs_en/configuration/settings.md",
        "//   (~/.iflow/settings.json, mcpServers, baseUrl) | checked 2026-09-13 | verified",
        "//   iFlow is low volume (npm ~161 weekly downloads, docs idle since 2026-03):",
        "//   kept as a cheap target, not a v1 priority (docs/supported-harnesses.md).",
    ],
    "kilo": [
        "",
        "// Source: vendor-doc | kilo.ai/docs | checked 2026-09-13 | stale?",
        "//   Docs confirm the OpenCode `mcp` dialect; the config filename has varied by",
        "//   version, so kiloPath probes several candidates instead of guessing one.",
    ],
    "kimicode": [
        "",
        "// Source: vendor-doc | kimi.com/code/docs/en/kimi-code-cli/customization/mcp.html",
        "//   | checked 2026-09-13 | verified",
        "//   Agent files are Claude/OpenCode-compatible; skills go to the shared",
        "//   ~/.agents/skills dir while agents use ~/.kimi-code/agents.",
    ],
    "kiro": [
        "",
        "// Source: vendor-doc | kiro.dev/docs/mcp (documents mcpServers; the page gives no",
        "//   file path) | checked 2026-09-13 | path stale?",
    ],
    "openclaw": [
        "",
        "// Source: internal | docs/supported-harnesses.md §第三轮调研 (OpenClaw) | checked 2026-09-13 | stale?",
        "//   openclaw.json is JSON5-ish: see LenientJSON in internal/mcp/target.go.",
        "//   Never: rewrite that file with a strict json.Marshal - it would drop user",
        "//   comments and break OpenClaw's own parse.",
    ],
    "qoder": [
        "",
        "// Source: vendor-doc | docs.qoder.com/cli/settings-reference · /cli/mcp-servers | checked 2026-09-13 | verified",
        "//   Hooks are Claude-shaped, so the Claude hook writer is reusable here.",
    ],
    "qwencode": [
        "",
        "// Source: internal | docs/supported-harnesses.md §调研来源 (Qwen Code) | checked 2026-09-13 | stale?",
        "//   Gemini CLI lineage (17 hook events, settings.json mcpServers); confirm the",
        "//   directory against QwenLM/qwen-code docs before editing.",
    ],
    "zcode": [
        "",
        "// Source: internal | docs/supported-harnesses.md §调研来源 (ZCode / Z.ai) | checked 2026-09-13 | stale?",
        "//   Nested `mcp.servers` key, unlike every flat `mcpServers` neighbour.",
    ],
    "zed": [
        "",
        "// Source: internal | docs/supported-harnesses.md §调研来源 (Zed) | checked 2026-09-13 | stale?",
        "//   zed.dev MCP doc URLs currently 404. Zed names MCP `context_servers` and has",
        "//   no native HTTP transport (stdio only).",
    ],
    "stub": [
        "// Package stub holds unimplemented harness placeholders.",
        "//",
        "// Source: internal | docs/supported-harnesses.md (capability matrix) | verified",
        "// A stub reports Implemented()=false so `magic status` shows n/a instead of",
        "// pretending a projection exists. Never: let a stub write files.",
    ],
    "dimagent": [
        "",
        "// Source: vendor-doc | dimagent.com/en/docs/config | checked 2026-09-13 | stale?",
        "//   The skills directory is unconfirmed, so SkillsDir returns empty rather",
        "//   than inventing a path.",
    ],
    "hermes": [
        "",
        "// Source: upstream | hermes_cli/config.py + website/docs/reference/mcp-config-reference.md",
        "//   | checked 2026-09-13 | verified",
        "//   YAML target (FormatYAML, snake_case mcp_servers).",
    ],
    "mimocode": [
        "",
        "// Source: upstream | mimocode packages/shared/src/global.ts | checked 2026-09-13 | verified",
        "//   OpenCode dialect; honours an env-var config root before falling back.",
    ],
    "omp": [
        "",
        "// Source: upstream | omp packages/utils/src/dirs.ts, docs/settings.md, docs/mcp-config.md",
        "//   | checked 2026-09-13 | verified",
        "//   Oh My Pi reuses the ~/.<name>/agent layout, which is why root resolution",
        "//   probes a sibling name instead of hardcoding one directory.",
    ],
}

TAGGED = re.compile(r"Source:\s+(upstream|vendor-doc|local-measure|internal|vendor-api)")


def pkg_index(lines):
    for i, ln in enumerate(lines):
        if ln.startswith("package "):
            return i
    return None


def main():
    touched, skipped = [], []
    for path in sorted(ADR.glob("*/*.go")):
        if path.name.endswith("_test.go"):
            continue
        stem = path.parent.name
        body = DOC.get(stem)
        if body is None:
            skipped.append(f"{path.relative_to(ROOT)} (no mapping)")
            continue
        lines = path.read_text(encoding="utf-8").split("\n")
        pk = pkg_index(lines)
        if pk is None:
            skipped.append(f"{path.relative_to(ROOT)} (no package line)")
            continue
        start = pk
        while start > 0 and lines[start - 1].startswith("//"):
            start -= 1
        doc = lines[start:pk]
        if any(TAGGED.search(l) for l in doc):
            skipped.append(f"{path.relative_to(ROOT)} (already tagged)")
            continue
        if body[0] == "":  # append under the file's existing package doc
            assert doc, f"{path}: append-style body but no package doc"
            out = lines[:pk] + body + lines[pk:]
        else:  # the body IS the package doc (replaces any existing one)
            # No blank line: a package doc must sit directly above `package`.
            out = lines[:start] + body + lines[pk:]
        path.write_text("\n".join(out), encoding="utf-8")
        touched.append(str(path.relative_to(ROOT)))
    for t in touched:
        print("updated  " + t)
    for s in skipped:
        print("skipped  " + s)
    print(f"\n{len(touched)} updated, {len(skipped)} skipped")
    return 0


if __name__ == "__main__":
    sys.exit(main())
