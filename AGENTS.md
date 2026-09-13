# AGENTS.md — Magic

给在本仓库工作的 Agent / 协作者的产品与实现约定。

## 产品是什么

Magic 是 **一份配置、投到多个 AI coding harness** 的 Go CLI。

用户只维护配置库 `~/.magic-ai`（`MAGIC_HOME`）：skills、MCP、API / provider / 模型。`apply` 把这份内容写进各 harness **已经在用的**配置文件。新开一个工具（OpenCode、Grok、Claude、Pi…）不必把 skill / MCP / API 再复制粘贴一遍。

**Harness** = 消费这些配置的 AI coding 工具，不是 Magic 自己。例：Claude Code、Codex、Grok、Pi、OpenCode、Cursor、Copilot。Magic 不安装它们，只往它们的配置里写。

多机：备份的是 **Magic 配置库**（Cloudflare R2、GitHub 私有库等），不是 `~/.pi` / `~/.grok` 家目录。另一台 Mac 或 Windows 拉下配置库再 apply。

产品概念简单，实现里难的是 **按各 harness 自己的格式写进去**。Skill / MCP 调研过很多次：每个工具的目录、manifest、MCP 字段都不一样，必须遵照对方格式写，不能塞一份通用 json 了事。

配置库里有两类内容：

| 层 | 是什么 | 怎么处理 |
|---|---|---|
| **跨 harness** | skill、MCP、API / 模型 | 库里一份；apply 时译成 Claude / Pi / Grok / OpenCode… 各自的格式写进已有文件 |
| **某 harness 自己的自定义** | 如 Pi 的 `themes/**`、`AGENTS.md` | 属于用户自定义配置，收进库是为了换电脑时还给**同一个** harness。不必译成别的工具的 theme |

前代 **skillmesh** 只做 Skill 同步。Magic 覆盖 skill、MCP、API，外加这类 per-harness 自定义配置的跨机备份。

**目的 vs 手段：** 目的是上面两类都进配置库，再能写回 / 同步。进库只能靠手改 / `skills` / `catalog` / `secrets`（capture 已删）。跨 harness 那层，Pi 的 5 个模型必须能写成 Grok / OpenCode 的格式，只把 `models.json` 原样拷回 Pi 不够。per-harness 自定义那层，把 Pi theme / `AGENTS.md` 按文件放进 `packs/harness/<id>/` 再放回去，就是对的手段。

**不做：** 装软件、整机备份、同步会话/缓存、绑定 Raycast / Alfred / tinycast、Cherry Studio、把项目拷到 Documents。

**配置库布局（已定死）：** `~/.magic-ai` 见用户目录 `README.md` 与 [docs/library-layout.md](./docs/library-layout.md)。`providers.yaml` + `secrets/providers.json`（本地明文，无 `.enc`）+ `packs/skills` + `packs/mcp` + `packs/harness/<id>`。官方 access token 不进库。**`.enc` 只表示云端状态**：`sync push` 只加密含密钥的文件并上传为 `.enc`（见 [docs/secrets-and-sync.md](./docs/secrets-and-sync.md)）。**无迁移入口**（开发阶段、零用户）。

**产品决定（已锁定）见 [docs/decisions.md](./docs/decisions.md)**——那里是唯一定义处，本文件不重复列。

**文档分层：** 目录 `docs/README.md`；对外入门 `docs/getting-started.md`；对内契约以本文件为准；harness 路径见 `docs/harness-matrix.md`。`NOTES.md` 是变更日志，不是需求真相源。

## 真相源与数据方向

| 动作 | 方向 |
|---|---|
| 用户配置（手改 / `skills` / `catalog` / `secrets` / `provider`） | 用户 → **配置库**（唯一要维护的目录） |
| `apply` | **配置库** → 各 harness 原生文件（注入/投影；库不变） |
| `sync push` | **配置库** → Cloudflare R2（或 git 私有库） |
| `sync pull` | R2 / git → **配置库** |

- 配置库才是真相。各 harness 家目录是投影目标，不是备份源。
- Skill **全局**真相：`$MAGIC_HOME/packs/skills`（默认 `~/.magic-ai/packs/skills`）。**不要**把 `~/.agents/skills` 当 Magic 全局中央仓，不要写迁移逻辑。
- Skill **项目**同步：`magic skills sync --scope project`（源：`<repo>/.agents/skills`）。未传 `--project` 时自动 git 根 / cwd。这与 `magic apply` **不是**同一条路径。
- MCP / API 只维护配置库里的一份，apply 时按目标 harness 的格式写进对应文件（不是拷一份通用文件到所有工具）。
- per-harness 自定义（Pi themes、`AGENTS.md` 等）按该工具的路径收/放，给跨机用；不投影到没有这种文件的 harness。
- **不要**在 apply 后自动 push。当前入口是手动 `magic sync push|pull`；自动挂接未接上时勿假装已有。
- 官方多机通道：R2。本地全明文（密钥在 `secrets/providers.json`）。**`.enc` 只是云端状态**：`sync push` 把含密钥的文件加密后以 `.enc` 上传，其余明文；`sync pull` 解密后落回本地明文（无 `.enc`）。口令仅 push/pull 时需要，**不进库、不同步**。git 私有库是同等备份通道。冲突检测需求有、当前实现未做（勿假装已有）。
- 进库手段：手改 / `skills` / `catalog` / `secrets`（capture 已删）。跨 harness 层：结果必须能按各工具格式写出。per-harness 自定义层：按该工具文件放进 `packs/harness/<id>/`。

## 命令地图（使用视角）

- 适配器：`magic adapters` / `magic status`
- Skills（skillmesh 级 UX）：`magic skills list|status|doctor|sync|…`，支持 `--scope project`
- 配置库 → harness：`magic apply`（达成目的的出口）
- 多机备份配置库：`magic sync push|pull|status`（需 `MAGIC_R2_*`；git 私有库为备选）
- 进配置库：手改、`skills` / `catalog` / `secrets`（结果要能变成可投影的 Magic 数据）
- sub-agent：`packs/agents/<name>.md` + `magic agents` / apply 投影
- Provider switch：`magic provider list|status|use|test|backup|env|seed`（`company` = active 别名；SwitchWriter 在 `internal/provider`）
- `copy` / `catalog`：只读 `providers.yaml`（支持 `endpoints[]` + 根级 `active`）；`copy` 用 `key_env` 从本地明文 `secrets/providers.json` 取 key（**不**需要口令，**不**读 process env）；`--protocol` / `--endpoint` 选 endpoint；`magic secrets set|list`；`registry sync|search|show` = models.dev 规范化 `models.json`（回退 OpenRouter slim）
- `apply` / `provider use` 投影：Claude settings env（AUTH_TOKEN 真值）、Pi / WorkBuddy / OpenCode harness 配置内**明文 apiKey 真值**（开箱即用，**不走 `$KEY_ENV`**）；Codex `models.json` + `model_catalog_json` + `env_key`（Codex 自身机制）。Grok/Codex 日常入口仍是 `magic provider use`。见 `docs/harness-matrix.md`
- **Apply 元数据 enrichment**：每个 pick 经 `modelfile.ResolvePickMeta` / `FindFuzzy` 查 `$MAGIC_HOME/registry/models.json`，投影到 Implemented harness（Pi 模型行、Grok context/efforts、Claude 角色/`CLAUDE_CODE_*`、Codex models.json + reasoning effort）。缺 meta → 默认值，不失败。先 `magic registry sync --source models.dev`。
- Capture：**已删除**（产品方向不符）：用户只维护 Magic 一份配置，不需要反向从 harness 收集。不要再加回。

## 实现位置

- 主实现：本仓库 Go module（`go-magic`）
- 内置 provider 模板：`internal/provider/builtin/providers.yaml`（go:embed，默认目录；见该目录 `README.md`）
- 产品/用户文档：`docs/`（见 `docs/README.md`）
- 原 python-magic 需求说明书若不在工作区，以本文件 + `docs/` 为准
- skillmesh 参考副本：`third_party/skillmesh.sh`

## Provider switch 架构

1. Library: `providers.yaml` + `secrets/providers.json`（本地明文）+ `magic.yaml`
2. Semantics: `official | <providerName>`；`company` → active only
3. `internal/provider.SwitchWriter` per harness（Claude/Codex/Grok/Pi/OpenCode/WorkBuddy）；gemini/copilot/trae share stub interface（仅 apply）
4. CLI thin over (1)+(3)；`test` HTTP harness-agnostic
5. `third_party/prompts-can-providers/*` = reference goldens only（scripts retired as entry）

## Registry / copy / providers 投影

- `internal/modelsdev` — raw models.dev Sync + types + `EffortThinkingLevelMap` + `KnownProviders`
- `internal/modelfile` — normalize → `$MAGIC_HOME/registry/models.json`；Search / FindFuzzy / `ResolvePickMeta`；apply/provider-use 元数据源
- `internal/openrouter` — slim OpenRouter adapter（`$MAGIC_HOME/registry/openrouter-models.json`）；`--source all` 时可 enrichment
- `docs/model-registry-from-pi.md` — 设计 + ProviderDefaults 表
- `internal/catalog` — `providers.yaml`（`Endpoint` / `NormalizeEndpoints` / `PickEndpoint` / `active`）+ `CopyText`/`CopyTextEndpoint`；`project.go` 写 Claude settings / Pi models
- `internal/provider` — SwitchWriter + resolve/seed/test/backup
- `internal/core/providers_apply.go` — `ApplyProviders`（apply 末尾：Claude / Pi / OpenCode / WorkBuddy 投影；写真值 key，不走 `$KEY_ENV`）
- `internal/provider/builtin` — 内置 provider 模板（go:embed `providers.yaml`）：默认 `name`/`key_env`/`endpoints[]`/`pick`；合并进用户 `providers.yaml` 时用户优先（endpoints/pick 整段替换）
- `internal/secrets` — 本地**明文** `secrets/providers.json`（0600）`Load/Set/GetProviderSecret`。`.enc` 只作为云端对象名，在 `sync push/pull` 加解密（见 [docs/secrets-and-sync.md](./docs/secrets-and-sync.md)）
- copy 在 `cmd/magic/copy.go` 读本地明文 `secrets/providers.json` 解析 key；**不**读 process env；**不**读 harness mirrors；`--protocol` / `--endpoint`

## 继续 registry 工作时

1. 改 normalize / thinkingLevelMap：看 `docs/model-registry-from-pi.md` + Pi `packages/ai/scripts/models-dev-reasoning-options.ts`。
2. 单测用 `testdata/registry/models.dev.fixture.json`，**禁止**单测打真网。
3. 别名 `aliases.json` / `patches.json` / `meta.json` etags 尚未做（**FUTURE**）；不要假装已有。Apply enrichment（Pi/Grok/Claude/Codex 读 `models.json`）**已实现**。
4. Adapter **全部已实现**：pi / grok / claude / codex / mcpuser / opencode / gemini / copilot / workbuddy / trae。**不做 windsurf**（产品濒临停运）。`workbuddy` = CodeBuddy（同一产品，目录待核）。⚠️ **Gemini CLI 已被 Antigravity CLI 取代**（2026-06-18）；`gemini` 保留过渡，新目标 `antigravity-cli`（`agy`）。SwitchWriter：Claude/Codex/Grok/Pi/OpenCode/WorkBuddy。Skills 目标另含 cursor。目标 harness 与能力矩阵见 `docs/supported-harnesses.md`（Tier L1/L2/L3 + 能力支持矩阵，**无优先级**）；MCP / hooks 专项见 `docs/mcp-design.md` / `docs/hooks-design.md`；harness/skills 路径名单来源 = `vercel-labs/skills` README → Supported Agents（快照 2026-09-11，**直接采纳为固定配置**）。
5. DeepSeek 官方集成模式见 `docs/deepseek-official-integrations.md`；Claude/Codex SwitchWriter **已按该模式落地**（勿再 shell 出 curl|bash installer）。
6. `mcpuser` = MCP 配置镜像，**不是** model provider；不要给它加 SwitchWriter stub 冒充 provider。

## Provider 接入（规划中，产品未定）

目标手段：`magic login <provider>` → 拿 key → 拉该 provider 官方 `GET /models` → 用户勾选 → 写 `providers.yaml` + 密钥。与 Pi `/login` + 选模型、Cherry Studio「获取模型列表」同一件事。

- 设计真相源：`docs/provider-onboarding.md`。**未实现**；产品定下来之前不写这部分代码，也不要把它写进 README / 命令帮助冒充已有。
- 已实现的部分：内置模板 `internal/provider/builtin/providers.yaml`（25 家官方骨架；中转不内置）。
- 密钥：本地明文 `secrets/providers.json`；`.enc` 只是云端状态（D2）。
- apply 分层（共有原数据投影 + `packs/harness/<id>/` 定制）设计见 [docs/apply-design.md](./docs/apply-design.md)。
- 不要把它和 `magic provider use`（切换已配置 provider）或 official OAuth 登录混为一谈。

**已定的投影规则（写代码时遵守）**：

- Pi / WorkBuddy / OpenCode 的 harness 配置里写**明文 key 真值**，不用 `$KEY_ENV` 占位。Claude 同理（AUTH_TOKEN）。Codex 用其自身 `env_key` 机制。
- 这些 harness 文件含明文 key。capture 已删，不存在“回流配置库”路径；本地密钥同在 `secrets/providers.json`。云端以 `.enc` 存。

代码未跟的项（C1 本地双模式、C2 Pi/WorkBuddy 仍写 `$KEY_ENV`、C3 reinject）只列在 [docs/open-questions.md](./docs/open-questions.md) 的 C 表，**不要写进产品句子**。

## 改代码时注意

1. `go test ./...` 与 `go build -o magic ./cmd/magic` 保持通过。
2. apply 的产品行为是把配置库里的 skill / MCP / API **写进** harness 已有文件。**写回一律字段级合并（结构化文件按键、一文件一项按项），不整文件覆盖**。冲突（A 机 5 个模型 vs B 机 3 个）比的是记录，不是整份 json 当黑盒。当前冲突检测未做（勿假装已有）。收外部文件进库用 `magic add path`。
3. skills 受管链接：永不删除真实目录；冲突 = 真目录或非受管链接。
4. Cursor 产品适配器按产品决定跳过；skills 目标里仍可有 cursor（与 skillmesh 一致）。**不做 windsurf、roo**（均停运）。
5. 跟手入口：自研菜单栏/热键，不绑第三方启动器（尚未实现）。

## 文档索引

见 [docs/README.md](./docs/README.md)。
