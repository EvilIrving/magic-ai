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

Magic 不只同步 skill：覆盖 skill、MCP、API，外加这类 per-harness 自定义配置的跨机备份。

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
- MCP：`magic mcp list|status|init`；库 `packs/mcp/mcp.yaml` → `magic apply` 译成各 harness 原生 MCP 配置（实现 `internal/mcp`）
- Skills：`magic skills list|status|doctor|sync|…`，支持 `--scope project`；`--target all` 覆盖 vercel-labs/skills 全量路径表（`internal/skills/paths.go`，~70 家，默认只写 8 家有 adapter 的）
- 配置库 → harness：`magic apply`（达成目的的出口）
- 多机备份配置库：`magic sync push|pull|status`（需 `MAGIC_R2_*`；git 私有库为备选）
- 进配置库：手改、`skills` / `catalog` / `secrets`（结果要能变成可投影的 Magic 数据）
- sub-agent：`packs/agents/<name>.md` + `magic agents` / apply 投影
- Provider switch：`magic provider list|status|use|test|backup|env|seed|add`（`add` = 内置模板写入 `providers.yaml`；读取时 `provider.LoadCatalog` 已把内置模板叠在用户文件下、用户优先。`company` = active 别名；SwitchWriter 在 `internal/provider`）
- Provider 接入（登录拉模型）：`magic login [provider] [--api-key …] [--pick a,b] [--yes] [--no-live]`（Bubble Tea TUI；实现 `internal/login` + `internal/provider/modelfetch.go`）
- `copy` / `catalog`：只读 `providers.yaml`（支持 `endpoints[]` + 根级 `active`）；`copy` 用 `key_env` 从本地明文 `secrets/providers.json` 取 key（**不**需要口令，**不**读 process env）；`--protocol` / `--endpoint` 选 endpoint；`magic secrets set|list`；`registry sync|search|show` = models.dev 规范化 `models.json`（回退 OpenRouter slim）
- `apply` / `provider use` 投影：Claude settings env（AUTH_TOKEN 真值）、Pi / WorkBuddy / OpenCode harness 配置内**明文 apiKey 真值**（开箱即用，**不走 `$KEY_ENV`**）；Codex `models.json` + `model_catalog_json` + `env_key`（Codex 自身机制）。Grok/Codex 日常入口仍是 `magic provider use`。见 `docs/harness-matrix.md`
- **Apply 元数据 enrichment**：每个 pick 经 `modelfile.ResolvePickMeta` / `FindFuzzy` 查 `$MAGIC_HOME/registry/models.json`，投影到 Implemented harness（Pi 模型行、Grok context/efforts、Claude 角色/`CLAUDE_CODE_*`、Codex models.json + reasoning effort）。缺 meta → 默认值，不失败。先 `magic registry sync --source models.dev`。
- `magic guide` / `status --json` / `adapters --json`：`cmd/magic/main.go`
- Capture：**已删除**（产品方向不符）：用户只维护 Magic 一份配置，不需要反向从 harness 收集。不要再加回。

## 实现位置

- 主实现：本仓库 Go module（`go-magic`）
- 内置 provider 模板：`internal/provider/builtin/providers.yaml`（go:embed，默认目录；见该目录 `README.md`）
- Provider 登录：`internal/login`（Bubble Tea TUI）+ `internal/provider/modelfetch.go`（`GET /models` 解析）+ `cmd/magic/login.go`
- MCP 投影：`internal/mcp`（`packs/mcp/mcp.yaml` 规范形 → 各家 `mcpServers`/`mcp`/`[mcp_servers]`；JSON+TOML）；hooks 走 `packs/harness/<id>/hooks/…` 字段级合并（B 方案）
- 产品/用户文档：`docs/`（见 `docs/README.md`）
- 原 python-magic 需求说明书若不在工作区，以本文件 + `docs/` 为准

## Provider switch 架构

1. Library: `providers.yaml` + `secrets/providers.json`（本地明文）+ `magic.yaml`
2. Semantics: `provider | official | none`；`company` → active only。`official` = 一方原生登录路径（仅 Claude/Codex/Grok/WorkBuddy）；`none` = 清掉 Magic 写的条目（Pi/OpenCode 这类多 provider 聚合器只有 `none`）
3. `internal/provider.SwitchWriter` per harness（Claude/Codex/Grok/Pi/OpenCode/WorkBuddy）；gemini/copilot/trae share stub interface（仅 apply）
4. CLI thin over (1)+(3)；`test` HTTP harness-agnostic

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
4. Adapter 已实现：pi / grok / claude / codex / mcpuser / opencode / gemini / copilot / workbuddy / trae + **v1 补齐 cursor / antigravity-cli / kimi-code / qoder / qoder-cn / cline**（D20，按 OpenRouter apps 用量分档）+ **调研批次 kiro / droid / zcode / qwen-code / iflow-cli / kilo / zed**。**不做 windsurf / roo**（D13，停运）。`workbuddy` = CodeBuddy。⚠️ **Gemini CLI 已被 Antigravity CLI 取代**；`gemini` 保留过渡，新目标 `antigravity-cli`（`agy`）。SwitchWriter：Claude/Codex/Grok/Pi/OpenCode/WorkBuddy。目标 harness 与能力矩阵见 `docs/supported-harnesses.md`（含「用量驱动的 v1 覆盖范围」与 P2/长尾）；MCP / hooks 专项见 `docs/mcp-design.md` / `docs/hooks-design.md`；harness/skills 路径名单来源 = `vercel-labs/skills` README → Supported Agents（快照 2026-09-11，**直接采纳为固定配置**）。
5. DeepSeek 官方集成模式见 `docs/deepseek-official-integrations.md`；Claude/Codex SwitchWriter **已按该模式落地**（勿再 shell 出 curl|bash installer）。
6. `mcpuser` = MCP 配置镜像，**不是** model provider；不要给它加 SwitchWriter stub 冒充 provider。

## Provider 接入（`magic login`，已实现）

手段：`magic login <provider>` → 拿 key（参数 / 已存 / TUI 输入）→ 拉该 provider 官方 `GET /models`（`internal/provider/modelfetch.go`）→ TUI 多选 → 确认后写 `providers.yaml` + 本地明文密钥。与 Pi `/login` + 选模型、Cherry Studio「获取模型列表」同一件事。

- 设计真相源：`docs/provider-onboarding.md`；决定见 `docs/decisions.md` D16/D17。实现：`internal/login`（Bubble Tea TUI：key / 多选 / 单选 / 确认）+ `cmd/magic/login.go`。
- 回退顺序（输出须说明）：live `/models` → 内置模板 `pick` → `registry/models.json` → 手动输入。非交互：`--api-key` + `--pick` + `--yes`（可选 `--no-live`）。不写 `models_meta`。
- 内置模板 `internal/provider/builtin/providers.yaml`（25 家官方骨架；中转不内置）；`magic provider add` 把模板写进用户文件。
- 密钥：本地明文 `secrets/providers.json`；`.enc` 只是云端状态（D2）。
- 已有同名 provider：`key_env` / `endpoints` / `models` 用户优先，`pick` 替换为本次勾选。
- 不要把它和 `magic provider use`（切换已配置 provider）或 official OAuth 登录混为一谈。TUI 库 = **Bubble Tea**（`bubbletea` + `bubbles` + `lipgloss`）。

**已定的投影规则（写代码时遵守）**：

- Pi / WorkBuddy / OpenCode 的 harness 配置里写**明文 key 真值**，不用 `$KEY_ENV` 占位。Claude 同理（AUTH_TOKEN）。Codex 用其自身 `env_key` 机制。
- 这些 harness 文件含明文 key。capture 已删，不存在“回流配置库”路径；本地密钥同在 `secrets/providers.json`。云端以 `.enc` 存。

C 表（[docs/open-questions.md](./docs/open-questions.md)）里 C1–C10 已全部落代码（本地明文 secrets/encrypt-on-push、真值 key、删 reinject、字段级合并、`magic provider add`、`status` 只列已装、`magic guide`、`magic login`、MCP 投影含 Codex/Grok TOML）。

## 注释与来源（写代码时遵守）

详规则见 [docs/comment-contract.md](./docs/comment-contract.md)。本节是入口约定。

注释是交付物的一部分，承担三种职责：**来源**（provenance）、**逻辑解释**（why，不是 what）、**开发约束**（never / invariant）。密度与写法由改动者判断，但**第一条在“外部事实定义点”强制**——Magic 难的不是逻辑，是各 harness 的磁盘路径、MCP 字段名、provider base URL、hook 事件名这些**别人的产品行为**；下一个人改到这里时必须当场看到它照什么写的，而不是去翻文档。

固定格式（`rg "Source:"` 可一次列出全仓外来事实）：

```go
// Source: <kind> | <locator> | checked <YYYY-MM-DD> | <confidence>
// kind: upstream | vendor-doc | vendor-api | local-measure | internal
// confidence: verified | measured | inferred | stale?（inferred / stale? 必须写确认方式）
```

- 一张表 / 一组常量放**一条**；行内来源与整块不同才单独标。
- **不写没核实过的 URL**；不知道来源就标 `inferred` / `stale?` 并给出确认动作。假 URL 比没有 URL 更坏。
- 偏离对方格式标 `Magic deviation:`；禁令写 `Never:`，不变性质写 `Invariant:`，未做能力写 `TODO(FUTURE):`，全部落在会被改到的那个函数 / 那张表上，而不是只进文档。
- 改一个带 `Source:` 的值：同步更新 locator + `checked` 日期，或保留原值并说明理由；不许悄悄改掉。
- 分工：代码 = 事实 + 来源 + 不可破坏的性质；`docs/` = 理由、对比、覆盖面、决定过程。两边不一致当 bug 处理。
- 报告工具：`./scripts/provenance.sh`（定义点覆盖情况，**不是** CI 门禁）。

## 改代码时注意

1. `go test ./...` 与 `go build -o magic ./cmd/magic` 保持通过。
2. apply 的产品行为是把配置库里的 skill / MCP / API **写进** harness 已有文件。**写回一律字段级合并（结构化文件按键、一文件一项按项），不整文件覆盖**。冲突（A 机 5 个模型 vs B 机 3 个）比的是记录，不是整份 json 当黑盒。当前冲突检测未做（勿假装已有）。收外部文件进库用 `magic add path`。
3. skills 受管链接：永不删除真实目录；冲突 = 真目录或非受管链接。
4. Cursor 已按 D20 升为完整 adapter（skills/MCP/hooks；无 model SwitchWriter）。**不做 windsurf、roo**（均停运）。P2/长尾 harness 推迟（D20）。
5. 跟手入口：自研菜单栏/热键，不绑第三方启动器（尚未实现）。
6. 新写 / 改动**外部事实定义点**（adapter 路径、MCP 方言、provider 键名与 base URL、registry 接口、sync endpoint）必须带 `Source:` 注释，约束见「注释与来源」与 [docs/comment-contract.md](./docs/comment-contract.md)。开发过程中形成但 `docs/` 还没写的关键约束、踩坑、非显然规则，就地写注释补上，别只留在会话里。

## 文档索引

见 [docs/README.md](./docs/README.md)。
