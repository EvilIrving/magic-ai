# AGENTS.md — Magic

给在本仓库工作的 Agent / 协作者的产品与实现约定。

## 产品是什么

Magic 是 **AI harness 配置管理器**（Go CLI，默认配置库 `~/.magic-ai`）。  
前代 **skillmesh** 只做 Skill 同步；Magic 是产品化升级：Skill 能力至少持平 skillmesh，并覆盖 sub-agent、capture/apply、密钥、R2 多机、跟手壳等。

**不做：** 装软件、整机备份、同步会话/缓存、绑定 Raycast/Alfred/tinycast、Cherry Studio、把项目拷到 Documents。

**文档分层：** 目录 `docs/README.md`；对外入门 `docs/getting-started.md`；对内契约以本文件为准；harness 路径见 `docs/harness-matrix.md`。`NOTES.md` 是变更日志，不是需求真相源。

## 真相源与数据方向

| 动作 | 方向 |
|---|---|
| `capture` | 本机 → **配置库**（库变了） |
| `apply` | **配置库** → 本机（库不变） |
| `sync push` | **配置库** → Cloudflare R2 |
| `sync pull` | R2 → **配置库** |

- Skill **全局**真相：`$MAGIC_HOME/packs/skills`（默认 `~/.magic-ai/packs/skills`）。**不要**把 `~/.agents/skills` 当 Magic 全局中央仓，不要写迁移逻辑。
- Skill **项目**同步：`magic skills sync --scope project`（源：`<repo>/.agents/skills`）。未传 `--project` 时自动 git 根 / cwd。这与 `magic apply` **不是**同一条路径。
- **自动 push 挂在库变更后**（如 capture），**不要**在 apply 后自动 push。当前入口是手动 `magic sync push|pull`；自动挂接未接上时勿假装已有。
- 官方多机通道：R2。`sync push` 上传整个配置库（含 `secrets/*.enc`）；**`MAGIC_PASSPHRASE` 仅本机**，不同步。git/iCloud 仅备选。冲突检测需求有、当前实现未做（勿假装已有）。

## 命令地图（使用视角）

- 适配器：`magic adapters` / `magic status`
- Skills（skillmesh 级 UX）：`magic skills list|status|doctor|sync|…`，支持 `--scope project`
- 配置库收/放：`magic capture` / `magic apply`
- 多机：`magic sync push|pull|status`（需 `MAGIC_R2_*`）
- sub-agent：`packs/agents/<name>.md` + `magic agents` / apply 投影
- Provider switch：`magic provider list|status|use|test|backup|env|seed`（`company` = active 别名；SwitchWriter 在 `internal/provider`）
- `copy` / `catalog`：只读 `providers.yaml`（支持 `endpoints[]` + 根级 `active`）；`copy` 用 `key_env` 从 `secrets/providers.json.enc` 取 key（需 `MAGIC_PASSPHRASE`；**不**读 process env）；`--protocol` / `--endpoint` 选 endpoint；`magic secrets set|list`；adapter secrets reinject 仍仅 apply；`registry sync|search|show` = models.dev 规范化 `models.json`（回退 OpenRouter slim）
- `apply` / `provider use` 投影：Claude settings env（AUTH_TOKEN 主）；Pi / WorkBuddy `$KEY_ENV`；OpenCode `opencode.json` 明文 apiKey；Codex `models.json` + `model_catalog_json`（仅第三方）。Grok/Codex 日常入口仍是 `magic provider use`。见 `docs/harness-matrix.md`
- **Apply 元数据 enrichment**：每个 pick 经 `modelfile.ResolvePickMeta` / `FindFuzzy` 查 `$MAGIC_HOME/registry/models.json`，投影到 Implemented harness（Pi 模型行、Grok context/efforts、Claude 角色/`CLAUDE_CODE_*`、Codex models.json + reasoning effort）。缺 meta → 默认值，不失败。先 `magic registry sync --source models.dev`。
- Config local refs：allow 的 `settings.json` / `settings.local.json` 跟随本地路径引用（`settings.ref`）；Codex `config.toml` 保守抽取 harness 内路径（`config.ref`）；capture 时把 `$HOME/<RelHome>` 折成 `~/…`。见 `internal/core/settings_refs.go`
- Capture allow 要点：Codex `hooks.json`/`themes/**`；Grok `hooks/**`/`commands/**`/`disabled-hooks`；Pi `themes/**` yes，`models-store.json` / `auth.json` deny；OpenCode 不收 auth.json；Copilot deny `config.json`；Gemini deny `.env`；Trae 只收 `~/.trae`。未列项默认不收

## 实现位置

- 主实现：本仓库 Go module（`go-magic`）
- 产品/用户文档：`docs/`（见 `docs/README.md`）
- 原 python-magic 需求说明书若不在工作区，以本文件 + `docs/` 为准
- skillmesh 参考副本：`third_party/skillmesh.sh`

## Provider switch 架构

1. Library: `providers.yaml` + `secrets/providers.json.enc` + `magic.yaml`
2. Semantics: `official | <providerName>`；`company` → active only
3. `internal/provider.SwitchWriter` per harness（Claude/Codex/Grok/Pi/OpenCode/WorkBuddy）；gemini/copilot/trae share stub interface（仅 capture/apply）
4. CLI thin over (1)+(3)；`test` HTTP harness-agnostic
5. `third_party/prompts-can-providers/*` = reference goldens only（scripts retired as entry）

## Registry / copy / providers 投影

- `internal/modelsdev` — raw models.dev Sync + types + `EffortThinkingLevelMap` + `KnownProviders`
- `internal/modelfile` — normalize → `$MAGIC_HOME/registry/models.json`；Search / FindFuzzy / `ResolvePickMeta`；apply/provider-use 元数据源
- `internal/openrouter` — slim OpenRouter adapter（`$MAGIC_HOME/registry/openrouter-models.json`）；`--source all` 时可 enrichment
- `docs/model-registry-from-pi.md` — 设计 + ProviderDefaults 表
- `internal/catalog` — `providers.yaml`（`Endpoint` / `NormalizeEndpoints` / `PickEndpoint` / `active`）+ `CopyText`/`CopyTextEndpoint`；`project.go` 写 Claude settings / Pi models
- `internal/provider` — SwitchWriter + resolve/seed/test/backup
- `internal/core/providers_apply.go` — `ApplyProviders`（apply 末尾：Claude 真值、Pi `$KEY_ENV`、OpenCode / WorkBuddy 走 SwitchWriter）
- `internal/secrets` — `providers.json.enc` map（`Load/Set/GetProviderSecret`）；adapter `StoreSecrets`/`LoadSecrets` 不变
- copy 在 `cmd/magic/copy.go` 用 `MAGIC_PASSPHRASE` 解析 key；**不**读 process env；**不**读 harness mirrors；`--protocol` / `--endpoint`

## 继续 registry 工作时

1. 改 normalize / thinkingLevelMap：看 `docs/model-registry-from-pi.md` + Pi `packages/ai/scripts/models-dev-reasoning-options.ts`。
2. 单测用 `testdata/registry/models.dev.fixture.json`，**禁止**单测打真网。
3. 别名 `aliases.json` / `patches.json` / `meta.json` etags 尚未做（**FUTURE**）；不要假装已有。Apply enrichment（Pi/Grok/Claude/Codex 读 `models.json`）**已实现**。
4. Adapter **全部已实现**：pi / grok / claude / codex / mcpuser / opencode / gemini / copilot / workbuddy / trae。**不做 windsurf**。SwitchWriter：Claude/Codex/Grok/Pi/OpenCode/WorkBuddy。Skills 目标另含 cursor。详见 `docs/harness-matrix.md`。
5. DeepSeek 官方集成模式见 `docs/deepseek-official-integrations.md`；Claude/Codex SwitchWriter **已按该模式落地**（勿再 shell 出 curl|bash installer）。
6. `mcpuser` = MCP 配置镜像，**不是** model provider；不要给它加 SwitchWriter stub 冒充 provider。

## 改代码时注意

1. `go test ./...` 与 `go build -o magic ./cmd/magic` 保持通过。
2. apply 对已 capture 的镜像文件会备份后覆盖本机；无冲突询问。
3. skills 受管链接：永不删除真实目录；冲突 = 真目录或非受管链接。
4. Cursor 产品适配器按产品决定跳过；skills 目标里仍可有 cursor（与 skillmesh 一致）。**不做 windsurf。**
5. 跟手入口：自研菜单栏/热键，不绑第三方启动器（尚未实现）。

## 文档索引

见 [docs/README.md](./docs/README.md)。
