# Magic

**English** · [中文](./README.zh-CN.md)

**One config library for your AI coding tools.**

Claude Code, Codex, Grok, OpenCode, Pi, Copilot and Trae each want their own copy of your skills, MCP servers and API keys. Magic keeps one copy in `~/.magic-ai` and writes it into the config files those tools already use, in each tool's own format.

```
~/.magic-ai/                          the only place you edit
      │  magic apply · magic skills sync
      ▼
~/.claude/  ~/.codex/  ~/.grok/  ~/.pi/  ~/.config/opencode/  …
```

Magic does not install those tools and does not replace them. It only writes into them.

## One copy in, many tools out

| You maintain | Magic writes |
|---|---|
| skills in `packs/skills/` | `~/.claude/skills`, `~/.codex/skills`, `~/.cursor/skills`, `~/.pi/agent/skills`, `~/.grok/skills`, `~/.config/opencode/skills`, `~/.copilot/skills`, `~/.trae/skills` |
| `providers.yaml` + keys | Claude `settings.json` token, Pi / OpenCode `apiKey`, Codex / Grok provider config |
| sub-agents in `packs/agents/*.md` | each tool's `agents/` directory |
| one tool's own files (Pi theme, `AGENTS.md`) | only back to that tool, so a new machine feels the same |

Keys are plaintext locally in `~/.magic-ai/secrets/providers.json` (0600). On cloud push, only the files that contain secrets are encrypted (`.enc`).

**Status:** skills, provider/API projection, sub-agents and per-tool files work today. MCP projection is designed ([docs/mcp-design.md](./docs/mcp-design.md)) but is not wired into `apply` yet.

## Install

Requires Go 1.24+.

```bash
git clone https://github.com/EvilIrving/magic-ai.git
cd magic-ai
make build          # or: go build -o magic ./cmd/magic
```

Put `magic` on your `PATH`.

## Quick start

```bash
magic init
magic secrets set DEEPSEEK_API_KEY --value 'sk-…'   # use your own key name
# edit ~/.magic-ai/providers.yaml: set `active` and `pick`
magic apply
magic skills sync
```

Open Claude, Pi or OpenCode: the config is already there. To go back to an official subscription:

```bash
magic provider use official
```

Grok and Codex switch providers with `magic provider use <name>`, not at the end of `apply`.

Project-scoped skills are read from `<repo>/.agents/skills`:

```bash
magic skills sync --scope project
```

## Commands

| Command | What it does |
|---|---|
| `magic status` | library state + which harnesses are installed here |
| `magic adapters` | supported harnesses and their skills / sub-agent / MCP capabilities |
| `magic apply` | write the library into each harness (`--dry-run`, `--only pi`) |
| `magic skills sync` | global skills; add `--scope project` for a repo |
| `magic skills status` | per-harness linked / missing / conflict / stale (read-only) |
| `magic agents` | sub-agents in the library, and which harnesses support them |
| `magic provider use official\|<name>` | switch a harness between official and a third-party provider |
| `magic secrets set\|list` | provider keys (`set` writes `secrets/providers.json`) |
| `magic add <path>` | copy a harness-local file into the library |
| `magic catalog` · `magic copy` | list `providers.yaml` · print one provider config (key from local secrets) |
| `magic registry sync\|search\|show` | models.dev catalog used for model metadata |
| `magic sync push\|pull\|status` | back up `~/.magic-ai` to Cloudflare R2 (needs `MAGIC_R2_*`) |

`docs/usage.md` shows each command as input, terminal output and resulting files. Start there:
[docs/usage.md](./docs/usage.md).

## Supported harnesses

Adapters: `claude`, `codex`, `grok`, `pi`, `opencode`, `workbuddy` (CodeBuddy), `gemini`,
`copilot`, `trae`, plus `mcpuser` for MCP config files. `cursor` is a skills target only.
Paths and capabilities per harness: [docs/harness-matrix.md](./docs/harness-matrix.md).

## Library layout

```
~/.magic-ai/
  magic.yaml                  which harnesses are enabled
  providers.yaml              relays/providers + model picks (no keys)
  secrets/providers.json      plaintext keys, 0600
  packs/skills/<name>/        cross-tool skills (global source of truth)
  packs/mcp/                  mcp.yaml (added by you)
  packs/agents/<name>.md      sub-agents
  packs/harness/<id>/         one harness's own files, written back to it only
  registry/                   models.dev cache, not user-edited
  backups/                    apply backups, not synced
```

Details: [docs/library-layout.md](./docs/library-layout.md).

## What Magic doesn't do

- install Claude / Pi / Grok
- sync sessions, caches or login state
- act as a skill marketplace, or a chat client like Cherry Studio
- bundle relay providers (SHU and friends) — those live in your own `providers.yaml`

## Docs

| | |
|---|---|
| [docs/usage.md](./docs/usage.md) | what each command does (read this first) |
| [docs/getting-started.md](./docs/getting-started.md) | install and first five minutes |
| [docs/harness-matrix.md](./docs/harness-matrix.md) | paths and capabilities per harness |
| [docs/library-layout.md](./docs/library-layout.md) | the `~/.magic-ai` directory |
| [docs/README.md](./docs/README.md) | full documentation index |

## License

MIT. See [LICENSE](./LICENSE).

Version 1.0.0 (`magic version`).
