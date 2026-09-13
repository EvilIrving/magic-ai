# Magic NOTES

> 历史变更日志（**不是**需求真相源）。**2026-09-10 及更早的条目里，capture / 本地加密 secrets /
> `$KEY_ENV` 占位已被推翻**。当前口径：本地 `secrets/providers.json` 明文；`.enc` **只是云端状态**；
> Pi / WorkBuddy / OpenCode / Claude 投影写真值 key。见 `docs/decisions.md` D2/D3、`docs/secrets-and-sync.md`。
> 下面旧条目里的 `providers.json.enc` / `MAGIC_PASSPHRASE` for copy 是当时实现，不是现行产品。

## 口径统一 (2026-09-13)

- D2：本地明文 `secrets/providers.json`；`.enc` 只表示云端对象（push 加密上传、pull 解密落回明文）。
- D3：Pi / WorkBuddy / OpenCode / Claude 写真值 key。实现差距只列 `docs/open-questions.md` C 表。

## Review fixes (2026-09-10)

- Redact: jsonc comments, `$KEY_ENV` skip, `key`/`access`/`refresh`/`*_token`; Pi deny `auth.json`; Claude partial redacts + reinjects.
- Apply: secret files 0600; skills skip unmanaged links; honor `adapters.enabled`; Claude apply uses `BuildClaudeEnvFromProvider`.
- R2: ListObjectsV2 pagination; reject `vault/../` escapes.
- WorkBuddy: union `availableModels`; Detect only `.codebuddy`.
- OpenCode SwitchWriter prefers existing jsonc and parses comments.

## Docs pass (2026-09-10)

- `docs/README.md` 文档目录；`getting-started.md` 改为中文对外入门（含当前 harness / R2 `magic-ai`）。
- `harness-matrix.md` 从调研草稿改为实现对照（不再写「建议/第二步」）。
- README / AGENTS 对齐：apply 投影 OpenCode+WorkBuddy、R2 桶约定、去掉失效的 `python-magic` / `/workspace/magic` 路径。

## Harness coverage (2026-09-10)

- Adapters implemented: gemini, copilot, workbuddy, trae (plus existing pi/grok/claude/codex/mcpuser/opencode).
- SwitchWriter: OpenCode (`opencode.json` provider.magic-*) + WorkBuddy (`~/.codebuddy/models.json` `$KEY_ENV`).
- **Dropped windsurf** (no product adapter / no skills target).
- Skills sync targets: + copilot, trae (global `~/.copilot/skills`, `~/.trae/skills`).
- `magic apply` also projects active provider into OpenCode / WorkBuddy when detected.

## Stub research + OpenCode adapter + docs (2026-09-10)

- Added `docs/harness-matrix.md` (real on-disk configs for stub harnesses + apply/SwitchWriter plan).
- Added `docs/getting-started.md` (external onboarding).
- OpenCode **Adapter implemented** (`~/.config/opencode`); skills sync target `opencode` added; auth.json under `~/.local/share/opencode` not captured by default.
- Removed `mcpuser` from provider SwitchWriter stub list (MCP ≠ model provider).
- R2：已单独建桶 `magic-ai`（前缀 `vault/`）。本机 `MAGIC_R2_*` 写在 local-environment `secrets.md`，不进本仓库。

## 1.0.0 formal (2026-09-09)

- Removed misleading `agents` adapter ("Central Agents"). Product language: **agents = sub-agents** under `packs/agents/`.
- Skills source of truth: `$MAGIC_HOME/packs/skills` (default `~/.magic-ai`).
- Default work dir remains `~/.magic-ai` (`MAGIC_HOME` / `MAGIC_VAULT`).
- Sub-agent projection: `packs/agents/<name>.md` → Caps.Agents harness dirs.
- Secrets: capture redact + apply reinject (passphrase); roundtrip tests.
- Skills apply: symlink → (Windows) junction → copy; never delete real dirs.
- Claude `~/.claude.json`: field-level extract → `claude.json.partial.json`.
- `magic catalog` / `magic copy <provider>` from `providers.yaml`.
- Implemented adapters: pi, grok, claude, codex, mcpuser, opencode, gemini, copilot, workbuddy, trae. No windsurf. No cursor product adapter (skills target only).

- Future UX (deferred): native Magic menubar/tray + global hotkey palette — not Raycast/Alfred.

## Skills UX (skillmesh port)
- `magic skills` mirrors skillmesh: sync/status/list/doctor/clean/init/import/adopt/enable/disable.
- Global source: `$MAGIC_HOME/packs/skills` only (never ~/.agents/skills as Magic global central).
- Project source: `<project>/.agents/skills`; sync via `magic skills sync --scope project`.
- Reference script: `third_party/skillmesh.sh`.

## OpenRouter registry + providers copy (2026-09-09)

- `magic registry sync|search|show` — OpenRouter models cache under `$MAGIC_HOME/registry/`.
- `magic copy <provider>` — providers.yaml only; key from `secrets/providers.json.enc` via `key_env` + `MAGIC_PASSPHRASE` (redact / `--reveal-key`). No process env for copy keys.
- `magic secrets set|list` — manage provider key map in `secrets/providers.json.enc`.
- Removed harness product paths: `copy pi/...`, `copy grok/...`, `--harness` / `--file` / `--auth-file`; deleted `internal/copyfmt` + `testdata/copy-fixtures/`.
- `magic catalog pick <provider> <model-id>` appends pick + optional `models_meta`.
- Adapter `secrets/<id>.json.enc` reinject remains apply-only; provider map is separate.

## Multi-endpoint providers → Claude / Pi (2026-09-09)

- `providers.yaml`: `endpoints[]` (`id`/`base_url`/`protocol`) + optional root `active`; legacy top-level `base_url`+`protocol` still synthesizes one endpoint.
- Protocols: `openai-chat-completions`, `openai-responses`, `anthropic-messages` (+ aliases).
- `magic copy --protocol` / `--endpoint` selects which base_url/protocol to print; keys still secrets-only.
- `magic apply` projects active provider: Claude `~/.claude/settings.json` env (plaintext token, 0600); Pi `~/.pi/agent/models.json` with `apiKey: $KEY_ENV` (+ `models.magic.json` sidecar). Grok/Codex also covered by `magic provider use` SwitchWriters.

## Provider switch: official ↔ third-party (2026-09-09)

- Product loop for every Implemented harness: `official | <providerName>`; `company` is **only** an alias for `providers.active` / `MAGIC_ACTIVE_PROVIDER` (never a core type / never hardcodes shu).
- Layers: library (`providers.yaml` + `secrets/providers.json.enc`) → SwitchWriter dialect → thin CLI `magic provider *`.
- Implemented writers: Claude (`settings.json`), Codex (`config.toml` model_providers.<name>), Grok (`# magic-managed-catalog-*`), Pi (`models.json` magic-<name>).
- Stubs (opencode/gemini/…): same SwitchWriter interface, return not-implemented.
- Seed profiles (shu / shu-deepseek / shu-grok / deepseek / kimi / minimax) via `magic provider seed` — vendor URLs live in library seed data.
- Keys: `magic secrets set` / `magic provider env set` → `secrets/providers.json.enc` (MAGIC_PASSPHRASE).
- Backups: `$MAGIC_HOME/backups/<harness>/` limit 3.
- `magic provider test` is harness-agnostic HTTP against provider endpoint (httptest in unit tests).
- Reference goldens only: `third_party/prompts-can-providers/{claudes,gpts,groks,pi}` — scripts superseded as daily entrypoints.
- Aliases: `magic use`, `magic test`.

## Capture allow + config refs (2026-09-09)

- Codex Allow: +`hooks.json`, +`themes/**` (deny sessions/sqlite/cache unchanged).
- Grok Allow: +`hooks/**`, +`commands/**`, +`disabled-hooks`.
- `settings_refs.go`: generic harness-root settings fold + ref follow; Codex `config.toml` → `config.ref`.

## Model registry from Pi (design 2026-09-09)

- Design: `docs/model-registry-from-pi.md` — models.dev primary, OpenRouter optional enrichment.
- Stub (earlier): raw `models.dev.json` download via `internal/modelsdev`.

## Model registry normalize (2026-09-09)

- `internal/modelfile` writes Pi-shaped `$MAGIC_HOME/registry/models.json` after sync.
- `thinkingLevelMap` from models.dev `reasoning_options` effort values (Pi `getEffortThinkingLevelMap`).
- ProviderDefaults table for api/baseUrl (openai→responses, anthropic→messages, deepseek→completions, …); unknown → unsupported + provider.api URL.
- CLI: `sync --source models.dev|all` always normalize; `openrouter` re-normalizes if models.dev.json exists.
- `registry search|show` + `catalog pick` prefer models.json (fuzzy: exact, strip vendor prefix, collapse .-_).
- Apply / `provider use` enrich Implemented harness writers from `registry/models.json` (Pi/Grok/Claude/Codex); sync `--source models.dev` first for best results. Missing meta → defaults, never fail apply.
- Include-all models; `toolCall` field kept. Costs are $/MTok as in models.dev (OR enrichment ×1e6).
- Still out of scope: aliases.json, patches.json, meta etags, Pi compat quirks.
- Tests: `testdata/registry/models.dev.fixture.json` (no live network).

## DeepSeek-aligned Claude / Codex projection (2026-09-09)

- Reference: `docs/deepseek-official-integrations.md` + `third_party/deepseek-integrations/codex-deepseek-setup.sh`.
- **Claude** third-party: `ANTHROPIC_AUTH_TOKEN` primary (+ `ANTHROPIC_API_KEY` same value); role envs incl. `CLAUDE_CODE_SUBAGENT_MODEL`; optional effort/auto-compact from registry meta; official clears `ANTHROPIC_*` + `CLAUDE_CODE_*`.
- **Codex** third-party: writes Codex-native `~/.codex/models.json` (minimal schema, no instruction blobs) + sets `model_catalog_json` **only** for third-party; `env_key` never plaintext bearer. Official removes catalog_json + deletes Magic `models.json`.
- Seed `deepseek`: anthropic `https://api.deepseek.com/anthropic`, responses `https://api.deepseek.com/`, Models pro/flash (+ subagent).

## Docs + sync/capture policy snapshot (2026-09-09)

- README / AGENTS / NOTES / `docs/*` / `python-magic/docs/需求.md` 对齐当前 CLI（`go run ./cmd/magic --help` 核对）。
- **copy**：secrets-only（`providers.json.enc` + `MAGIC_PASSPHRASE`）；**无** OS env override；无 `copy pi/...` / `grok/...`。
- **R2 sync**：可同步加密 `secrets/*.enc`；passphrase 本机；push 跳过 `backups/`。自动 push 约定挂库变更（capture），**不**挂 apply；当前以手动 `magic sync push` 为主。
- **Capture**：settings.json ref-follow；Codex/Grok hooks in allow；Pi themes yes，`models-store.json` deny，memory 未 allow → 默认不收。
- **Codex**：第三方写 `~/.codex/models.json` + `model_catalog_json`；官方移除 catalog、**不用** `model_catalog_json`；无 plaintext bearer。
- **非目标重申**：整机还原、Raycast/Alfred、Cherry Studio（取消）、拷项目到 Documents。
- **FUTURE**：`aliases.json`、`patches.json`、registry `meta.json` etags、跟手壳、R2 冲突 UI。
