# DeepSeek official agent integrations (reference)

Sources (2026-09-09):
- https://api-docs.deepseek.com/zh-cn/quick_start/agent_integrations/claude_code
- https://api-docs.deepseek.com/zh-cn/quick_start/agent_integrations/codex
- Setup script: https://cdn.deepseek.com/api-docs/codex-deepseek-setup.sh

## Claude Code
Third-party via Anthropic-compatible endpoint:
- `ANTHROPIC_BASE_URL=https://api.deepseek.com/anthropic`  (**note `/anthropic` suffix** = anthropic-messages dialect)
- `ANTHROPIC_AUTH_TOKEN` (docs use AUTH_TOKEN, not API_KEY) — Magic sets AUTH_TOKEN primarily and also sets `ANTHROPIC_API_KEY` to the same value for broader Claude Code compatibility
- Role envs: `ANTHROPIC_MODEL`, `ANTHROPIC_DEFAULT_{OPUS,SONNET,HAIKU}_MODEL`, `CLAUDE_CODE_SUBAGENT_MODEL`
- Extras: `CLAUDE_CODE_EFFORT_LEVEL`, `CLAUDE_CODE_AUTO_COMPACT_WINDOW` (Magic derives from registry meta when present: highest thinking level + ~75% of contextWindow)
- DeepSeek maps claude-opus* → v4-pro, claude-haiku/sonnet* → v4-flash server-side
- Seed `deepseek` Models: opus/sonnet/default → `deepseek-v4-pro`; haiku/subagent → `deepseek-v4-flash`

Official Apply clears all Magic-managed `ANTHROPIC_*` and `CLAUDE_CODE_*` keys.

## Codex
Third-party needs **both**:
1. `~/.codex/models.json` — Codex-native catalog. Magic writes a **minimal viable** schema (no giant `model_messages` / `base_instructions` blobs):
   - required-ish: `slug`, `display_name`, `context_window`, `max_context_window`, `input_modalities`, `supported_reasoning_levels`, `default_reasoning_level`, `visibility`, `supported_in_api`
   - values enriched from `$MAGIC_HOME/registry/models.json` when available; else safe defaults (128k context, text modality, low/high/max efforts)
2. `config.toml`: `model_provider`, `model`, `model_reasoning_effort`, **`model_catalog_json`** → **only for third-party**
3. `[model_providers.<name>]` with `wire_api` (from endpoint protocol, usually `responses`), `base_url`, `env_key` — **never** plaintext bearer tokens

Official Apply:
- remove `model_catalog_json`
- delete Magic-managed `~/.codex/models.json`
- switch `model_provider` → `openai`
- do not delete user MCP / projects / plugins

Official restore = remove deepseek provider + catalog (script menu item 9). Confirms: **official does not use model_catalog_json**.

Magic should treat DeepSeek (and similar) provider profiles as library data that projects these dialect fields; do not shell out to their curl|bash installer.

## Magic adapters (implemented 2026-09-09)

These patterns are **implemented** in Magic dialect writers — do not shell out to DeepSeek’s curl|bash installer for daily use:

| Pattern | Magic entry |
|---------|-------------|
| Claude AUTH_TOKEN + `/anthropic` base + role / `CLAUDE_CODE_*` | `internal/provider/claude.go` + `internal/core/providers_apply.go` via `magic provider use` / `magic apply` |
| Codex `models.json` + `model_catalog_json` (third-party only) + `env_key` | `internal/provider/codex.go` |
| Seed DeepSeek endpoints (`…/anthropic` + responses root) + Models pro/flash | `magic provider seed` → `providers.yaml` |

`official` clears Magic-managed Claude env keys / Codex catalog as described above. Same shape applies to other Anthropic-compatible / Codex-compatible relays in the library (e.g. seeded `shu`).
