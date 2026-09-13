# Magic NOTES

> 历史变更日志（**不是**需求真相源）。**2026-09-10 及更早的条目里，capture / 本地加密 secrets /
> `$KEY_ENV` 占位已被推翻**。当前口径：本地 `secrets/providers.json` 明文；`.enc` **只是云端状态**；
> Pi / WorkBuddy / OpenCode / Claude 投影写真值 key。见 `docs/decisions.md` D2/D3、`docs/secrets-and-sync.md`。
> 下面旧条目里的 `providers.json.enc` / `MAGIC_PASSPHRASE` for copy 是当时实现，不是现行产品。

## 排行榜全量调研 + 第三批 adapter (2026-09-13)

- 用户榜单第 2–30 名逐家调研（官方文档 + 源码），产出 `docs/harness-rollout-plan.md`（含两轮自审、能力矩阵、实施批次）。
- 新增 adapter：`omp`（Oh My Pi，`~/.omp/agent`）、`hermes`（`~/.hermes/config.yaml`）、`mimocode`（MiMo Code，`~/.config/mimocode`）、`dimagent`（`~/.dim/v2`）、`openclaw`（`~/.openclaw`）。
- MCP：新增 `FormatYAML`（`internal/mcp/yaml.go`，`yaml.Node` 级合并保留注释/其它键）+ 4 个 dialect：`omp`、`hermes`、`dimagent`（sse→n/a）、`openclaw`（嵌套 `mcp.servers`）。
- `DefaultAgents` 增 `omp/hermes/mimocode/openclaw`；skills 表增 `omp`/`oh-my-pi`。
- 决定 D21–D25：Roo Code 停运不做；Alma/MiniMax-MCP/Lingma UI-only；DSH MCP 暂缓；Cola 身份待确认。
- 修复工作区里 `internal/provider/switch_test.go` 的编译错误（`&catalogFileForTest(t)` → `catalogFileForTest(t)`，untracked WIP 笔误）。
- 测试：`internal/mcp` 新 dialect + YAML/嵌套键用例；`go test ./...` 全绿；临时 HOME 下 `magic apply` 端到端验证 5 个新 target。

## provider switch 语义修正：official vs none (2026-09-13)

- 读官方文档后修正（Pi https://pi.dev/docs/latest/providers，OpenCode /docs/providers + models.dev）。
- `SwitchWriter` 新增 `SupportsOfficial() bool`：claude/codex/grok/workbuddy = true；**pi/opencode = false**（多 provider 聚合器）。
- 新目标 `none`（别名 off/clear）→ 对所有 harness 都表示「清掉 Magic 写进配置的条目」。
  `provider use official --harness pi|opencode` 报错并提示用 `none`；无 `--harness` 时对这两家记 `n/a` 而非失败。
- 顺便核实 OpenCode 字段：官方 JSON Schema 确认 `provider.<id>.{npm,name,whitelist,blacklist,options.{apiKey,baseURL,timeout},models}` 均合法（之前的 `whitelist` 写法没问题）。
- 决定 D21；AGENTS/usage/README/decisions 已同步。测试：`internal/provider/switch_test.go`。

## Provider 补齐 + MCP 第二轮调研/实现 (2026-09-13)

- 内置模板 `providers.yaml` +5：`google`(Gemini OpenAI 兼容)、`mistral`、`meta`(Llama API)、`upstage`、`hunyuan`(腾讯)，按模型用量榜补。
- MCP 新增 7 个 adapter + target：kiro、droid、zcode(嵌套 `mcp.servers`)、qwen-code、iflow-cli、kilo、zed。
- 新 dialect：kiro(`disabled`)、zcode(`enable`)、zed(stdio only)；target.TopKey 支持点号嵌套键（读写/检查均已）。
- 调研文档：`docs/mcp-design.md` 新增「第二轮调研（2026-09-13）」，列出全部 harness 的 MCP 格式 + 状态。
  未实现（需新 writer）：Hermes/Continue/Goose（YAML）、Mistral Vibe（TOML 数组表）、OpenClaw（JSON5）、Crush（待实测）、MiniMax/Lingma/Amp/Cortex（UI/CLI only）。
- 测试：`internal/registry`（P1+调研批次）、`internal/mcp`（新 dialect/target/嵌套键）；`go test ./...` 全绿。

## 用量驱动的 v1 覆盖：P1 补齐 (2026-09-13)

- 依据 OpenRouter app 榜（2026-09）定 v1 范围（D20，修订 D9 的「全做」）。
- 新增 adapter：`cursor`（升为完整，含 MCP/hooks）、`antigravity-cli`、`kimi-code`、`qoder`、`qoder-cn`、`cline`（+ `internal/adapters/*`，注册到 `registry`、写入 `magic.yaml` 默认 enabled）。
- MCP `translate.go` 新增 dialect：cursor / antigravity(`serverUrl`) / kimi(`transport:sse`) / qoder / cline(`streamableHttp`)；`target.go` 落点从 10 → 16。
- `DefaultAgents` 改为按用量排序的 14 家；skills 路径表增 `kimi-code` 别名。
- P2（droid/kiro/minimax-code/zcode/hermes/openclaw/deepseek-harness/oh-my-pi/dimagent/alma/craftagent/cola）与长尾~50 家**推迟**；skill 路径已在表、`--target all` 可达。
- 测试：`internal/mcp` 新 dialect + target，`internal/registry` v1 core 完整性；`go test ./...` 13 包全绿。

## L2 skills 长尾表接入 (2026-09-13)

- `internal/skills/paths.go`：新增 `globalSkills` 表（vercel-labs/skills 全量，~74 家）、`KnownAgent` / `AllAgents`；`GlobalTargetDir` 改为查表；`AgentDotDir`（project）扩到产品集。
- `--target all` → `AllAgents()`；默认仍 8 家（避免给未安装工具造目录）。`windsurf`/`roo` 剔除（D13）。
- 测试：`internal/skills/paths_test.go`。

## MCP + hooks 投影落地 (2026-09-13)

- 新增 `internal/mcp`：规范形 `packs/mcp/mcp.yaml` → 各 harness 原生格式（JSON + TOML）。
  `translate.go`（claude/codex/grok/opencode/pi/gemini/copilot/workbuddy/trae/generic）、
  `toml.go`（`[mcp_servers.<name>]` 字段级 upsert）、`target.go`（10 个 harness 落点）、
  `apply.go`（merge，保留用户条目）、`inspect.go`。
- `magic apply` 新增 `## mcp projection`；新增 `magic mcp list|status|init`。
- `adapter.Caps` 新增 `Hooks`；claude/codex/grok/pi/workbuddy 等补上 `MCP:true`；`magic adapters` 加 HOOKS 列。
- hooks 按 B 方案（D19）：`packs/harness/<id>/hooks/…` 字段级合并（`merge.go` 白名单加 `hooks.json`/`mcp.json`/`mcp-config.json`）。
- 决定 D18（MCP：无前缀、按库名字管理、明文密钥） / D19（hooks：B 方案、路径不折 `~`、不写 trust）。A5–A8 已定，C10 已定。
- 测试：`internal/mcp` 翻译/TOML/投影合并；`go test ./...` 13 包全绿。

## `magic login` 落地（Bubble Tea TUI）(2026-09-13)

- 新增 `internal/provider/modelfetch.go`：`ModelsURLFor` / `FetchModels` / `ParseModelsResponse`（OpenAI `data[]`、Anthropic `display_name`、裸数组/字符串；去重排序）。
- 新增 `internal/login`：`LookupProvider` / `Resolve`（三级回退 live → 内置 pick → `registry/models.json` → 手动；`DefaultSelection`）/ `Apply`（用户优先合并，`pick` 替换）/ Bubble Tea TUI（masked key 输入、单选/多选、确认）。
- 新增 `cmd/magic/login.go`：`magic login [provider] [--api-key] [--pick] [--yes] [--no-live]`；非 TTY 强制 `--pick` + `--yes`，TTY 检测用 `go-isatty`。
- 依赖：`charmbracelet/bubbletea` + `bubbles` + `lipgloss`。
- 决定 D17（TUI 库、确认后写、回退顺序、不写 `models_meta`）；A3 已定。
- 测试：`internal/login`（13 个，含 fake `/models` httptest、TUI model 直接驱动）、`internal/provider` 解析。`go test ./...` 全绿。

## C 表落地：D2/D3/D5 实现跟上 (2026-09-13)

- **C1** `internal/secrets` 改为本地恒 `secrets/providers.json` 明文 0600；删 `MAGIC_SECRETS_MODE` / 双模式 / 本地 `.enc`。`sync push` 按「路径在 `secrets/` 或内容含真值」加密后上传 `<rel>.enc`；`sync pull` 解密 `.enc` 落回明文、无 `.enc`。口令仅 push/pull 需要。
- **C2** Pi / WorkBuddy 写**真值** apiKey（`catalog.BuildPiProviderEntry` 收真值；`provider.Use` 对 pi/workbuddy 也 needKey）。无 key 则 skip + 日志，不写空值。
- **C3** 删 adapter 级 `secrets/<id>.json.enc` reinject（`LoadSecrets` / `ReinjectBytes` / `${MAGIC_SECRET:}`）；apply 的 harness 定制文件本地即最终内容。
- **C7** `core/merge.go`：apply 写回结构化文件字段级合并（JSON deep merge / TOML section upsert，`[[array]]` 整段替换）；AGENTS.md 等整份覆盖；一文件一项自然不删其它项。
- **C4** `provider/builtin` 增 `Merge` / `MergeProvider` / `AddSelection`；`provider.LoadCatalog` 读取时叠加内置模板（用户优先）；新增 `magic provider add [ids…|--all|--list]`。
- **C8** `magic status` 只列已安装 harness（D12）。
- **C9** 新增 `magic guide`、`status --json`、`adapters --json`。桌面端 / 签名发行仍不在库内。
- 测试：`go test ./...` 全绿；新增 merge / builtin / sync / secrets 单测。

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

## Skills UX
- `magic skills`: sync/status/list/doctor/clean/init/import/adopt/enable/disable.
- Global source: `$MAGIC_HOME/packs/skills` only (never ~/.agents/skills as Magic global central).
- Project source: `<project>/.agents/skills`; sync via `magic skills sync --scope project`.

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

- Reference: `docs/deepseek-official-integrations.md`.
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
