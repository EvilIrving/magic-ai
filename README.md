# Magic

**1.0.0** — 一份配置，投到多个 AI coding harness。

- **文档目录：** [docs/README.md](./docs/README.md)
- **产品使用定义（先读这个）：** [docs/usage.md](./docs/usage.md) —— 每条命令做什么、吐什么、生成的文件长什么样
- **对外入门：** [docs/getting-started.md](./docs/getting-started.md)
- **对内约定：** [AGENTS.md](./AGENTS.md)
- **Harness 矩阵：** [docs/harness-matrix.md](./docs/harness-matrix.md)

真相源是 **`~/.magic-ai`**（`MAGIC_HOME`；别名 `MAGIC_VAULT`）。产品概念简单：只维护这一份 skill / MCP / 中转 API，`apply` 按各 harness **自己的格式**写进已经在用的文件。新开 OpenCode / Grok 不必复制粘贴。Pi 的 theme、`AGENTS.md` 这类自定义随库跨机，只还回同一个工具。

手改配置库 / `skills` / `catalog` / `secrets` 是进库的**手段**，不是产品本身。官方登录 / access token / OAuth **不进库**。多机备份的是这份配置库（R2 / git），不是 `~/.pi` 家目录。不做整机备份，不安装软件。布局见 [docs/library-layout.md](./docs/library-layout.md)。

## 用法优先（Skills）

```bash
# 看支持的 harness 适配器（含 Skills 能力）
magic adapters

# 列出全局技能（源 = $MAGIC_HOME/packs/skills）
magic skills list

# 看每个 Agent 的链接 / 缺失 / **冲突** / 失效 / 外部项
magic skills status

# 同步项目技能（源 = <project>/.agents/skills → .claude|.codex|.cursor|.pi|.grok|.opencode|.copilot|.trae/skills）
magic skills sync --scope project
# 可选：同时继承全局库（同名以项目为准）
magic skills sync --scope project --include-global
```

| 作用域 | 源库 | 目标 |
|--------|------|------|
| **global**（默认） | `$MAGIC_HOME/packs/skills` | `~/.claude/skills`、`~/.codex/skills`、`~/.cursor/skills`、`~/.pi/agent/skills`、`~/.grok/skills`、`~/.config/opencode/skills`、`~/.copilot/skills`、`~/.trae/skills` |
| **project** | `<project>/.agents/skills` | `<project>/.claude\|codex\|cursor\|pi\|grok\|opencode\|copilot\|trae/skills` |

> 全局 Skill 真相源**只有** `$MAGIC_HOME/packs/skills`（默认 `~/.magic-ai/packs/skills`）。  
> **不是** `~/.agents/skills`。项目级才用 `<project>/.agents/skills`。  
> 项目同步用 `magic skills sync --scope project`，**不是** `magic apply`。

## 核心概念

| 概念 | 含义 |
|------|------|
| **配置库** | `~/.magic-ai`：唯一要维护的目录。跨工具的 skill / MCP / API 一份；各工具自己的 theme、`AGENTS.md` 另放 `packs/harness/<id>/` |
| **Harness** | 消费这些配置的 AI coding 工具（Pi / Grok / Claude / Codex / OpenCode…）。Magic 不装它们，只按对方格式往配置里写 |
| **Skill（全局）** | `$MAGIC_HOME/packs/skills/<name>/`；用 `magic skills sync` 或 `magic apply` 投影 |
| **Skill（项目）** | `<project>/.agents/skills/`；仅 `magic skills … --scope project` |
| **MCP** | `$MAGIC_HOME/packs/mcp/mcp.yaml`；apply 按各工具 mcp 字段写入 |
| **中转 API** | `providers.yaml`（无密钥）+ `secrets/providers.json`（本地明文 0600；`.enc` 只是云端状态） |
| **sub-agent** | `packs/agents/<name>.md`；apply 写入各工具 `…/agents/`。产品里说的 **agents = sub-agent** |
| **Harness 适配器** | pi / grok / claude / codex / mcpuser / opencode / gemini / copilot / workbuddy / trae。Skills 目标另含 cursor。**不做 windsurf**。矩阵见 [docs/harness-matrix.md](./docs/harness-matrix.md) |

## 构建

```bash
cd go-magic
go test ./...
go build -o magic ./cmd/magic
./magic version   # → 1.0.0
```

## 初始化

```bash
export MAGIC_HOME=~/.magic-ai   # 未设置时默认即此
./magic init
./magic skills init             # 确保 packs/skills + 各 agent skills 目录
```

创建：`magic.yaml`、`providers.yaml`、`packs/skills`、`packs/mcp`、`packs/agents`、`packs/harness`、`secrets/`、`README.md`。

## 常用命令

```bash
magic status
magic adapters
magic apply [--dry-run] [--home "$HOME"]   # 配置库 → 各 harness 原生文件（skill / MCP / API）

magic skills sync [skill...] [--scope global|project|all] [--target claude,pi] [--dry-run]
magic skills status [skill...]
magic skills list [--include-global]
magic skills doctor [skill...]
magic skills clean [--orphaned]
magic skills init
magic skills import PATH [--copy] [--name NAME] [--sync]
magic skills adopt --from AGENT [--sync]
magic skills enable|disable SKILL...

magic agents          # 列 packs/agents 与 Caps.Agents 适配器
magic catalog         # 列 providers.yaml
magic catalog pick <provider> <model-id>   # 追加 pick（可选 models_meta）
magic copy deepseek              # providers.yaml → stdout (key from local secrets/providers.json)
magic copy deepseek --reveal-key # full api key (default redacts middle)
magic copy shu --protocol anthropic-messages   # pick endpoint by protocol
magic copy shu --endpoint openai-responses     # pick endpoint by id
magic secrets set DEEPSEEK_API_KEY --value 'sk-…'   # store key (local plaintext, no passphrase)
magic provider use official|company|<name> [--harness claude|codex|grok|pi|opencode|workbuddy]
magic provider test company
magic secrets list               # key names only
magic registry sync|search|show   # models.dev 规范化目录（+ OpenRouter 可选）
magic sync status|push|pull   # 需 MAGIC_R2_*（配置库 R2，不是 skills sync）
```

### Model registry（models.dev → models.json）

元数据真相源是 **models.dev**；OpenRouter 为可选补充。同步后写入 Pi 形 `$MAGIC_HOME/registry/models.json`（含 `thinkingLevelMap` / cost / api / baseUrl）。设计见 `docs/model-registry-from-pi.md`。

```bash
magic registry sync --source models.dev   # GET api.json → models.dev.json + normalize → models.json
magic registry sync --source all          # models.dev + OpenRouter slim + normalize（OR enrichment）
magic registry sync                       # 默认仍 openrouter slim；若已有 models.dev.json 会再 normalize
magic registry search opus                # 优先搜 models.json（否则回退 openrouter-models.json）
magic registry show gpt-5-nano            # 含 thinkingLevelMap 等
magic catalog pick deepseek deepseek-v4-flash   # pick + models_meta（优先规范化目录）
```

磁盘：

```
$MAGIC_HOME/registry/
  models.dev.json          # 原始 models.dev dump
  openrouter-models.json   # 既有 slim OpenRouter 缓存
  models.json              # 规范化 Magic catalog
```

### providers.yaml（endpoints + active）

官方或多协议自定义（同一 `key_env` + `pick`，多条 `endpoints[]`）。中转站写在用户文件里，不进内置模板。

```yaml
active: deepseek   # optional; override with MAGIC_ACTIVE_PROVIDER
providers:
  deepseek:
    key_env: DEEPSEEK_API_KEY
    pick: [deepseek-flash, deepseek-v4-pro]
    endpoints:
      - id: anthropic
        base_url: https://api.deepseek.com/anthropic
        protocol: anthropic-messages
      - id: openai-chat
        base_url: https://api.deepseek.com/v1
        protocol: openai-chat-completions
      - id: openai-responses
        base_url: https://api.deepseek.com/v1
        protocol: openai-responses
```

- 协议常量：`openai-chat-completions`、`openai-responses`、`anthropic-messages`（别名如 `openai` / `anthropic` 可接受）。
- 兼容：若 `endpoints` 为空，顶层 `base_url` + `protocol` 合成一条 endpoint。
- `active` / `MAGIC_ACTIVE_PROVIDER`：`magic apply` / `magic provider use company` 选用哪个第三方；多候选且未指定则跳过并列出。`company` **只是** active 别名。

### Copy + provider secrets

`magic copy <provider>` 从 `$MAGIC_HOME/providers.yaml` 抽出该 provider 的非密钥字段到 stdout（clipboard-friendly）。

- `key_env` 是逻辑索引名（如 `DEEPSEEK_API_KEY`），**不是** process env 变量名；真值存在本地密钥文件。
- **本地明文**：`$MAGIC_HOME/secrets/providers.json`（0600）。本地无 `.enc`，无需口令。
- **`.enc` 是云端状态**：`sync push` 把含密钥的文件加密后上传为 `<rel>.enc`，其余明文；`sync pull` 解密后落回本地明文。口令仅 push/pull 需要。
- 密钥解析（copy）：按 key_env 在本地明文库里查找（fallback：provider 名）→ 有则打印（默认 redact 中间：首 4…末 4；`--reveal-key` 全文）→ 无则占位符 `<DEEPSEEK_API_KEY>` + 一行缺失提示。
- **不**读 OS / process env 取 copy key（无 env override）。
- Endpoint 选择：默认第一条；`--protocol <name>` 或 `--endpoint <id>` 指定。
- 写入：`magic secrets set <key_env> --value …`（或 stdin）；`magic secrets list` 只列名。皆不需口令。
- 不再支持 `magic copy pi/...` / `grok/...` / `--harness`。


## Provider switch（official ↔ 中转）

`claudes` / `gpts` / `groks` / pi 探测脚本已由 Magic 统一入口替代（参考副本在 `third_party/prompts-can-providers/`，仅作 golden，不再作为日常入口）。

```bash
magic provider seed                          # 写入默认 providers.yaml 档案（含 shu 等）
magic secrets set SHU_API_KEY --value '…'    # 或 magic provider env set …
magic provider list
magic provider status [--harness claude|codex|grok|pi|opencode|workbuddy]
magic provider use official [--harness …]    # 清除 Magic 托管覆盖 / 官方路径
magic provider use company [--harness …]     # = providers.active（或 MAGIC_ACTIVE_PROVIDER）
magic provider use shu --harness opencode    # 具名第三方
magic provider test company                  # HTTP GET …/models（与 harness 无关）
magic provider backup list|restore|clear --harness claude
```

薄别名：`magic use`、`magic test`。

语义：`official | <providerName>`；**company 只是 active 第三方的别名**，不把某一厂商写进核心类型。新增 harness = 实现一个 `SwitchWriter`，不是再写 shell。


## Sync（Cloudflare R2）

官方多机通道：`magic sync push|pull|status`（需 `MAGIC_R2_*`）。约定桶名 **`magic-ai`**，对象前缀 `vault/`（与 `light-stats` 等下载桶分开）。

必填：`MAGIC_R2_BUCKET`、`MAGIC_R2_ACCESS_KEY_ID`（或 `AWS_ACCESS_KEY_ID`）、`MAGIC_R2_SECRET_ACCESS_KEY`（或 `AWS_SECRET_ACCESS_KEY`）。建议：`MAGIC_R2_ENDPOINT=https://<ACCOUNT_ID>.r2.cloudflarestorage.com`。可选：`MAGIC_R2_REGION`（默认 `auto`）。这是 R2 **S3 API** 凭证，不是随便一个 Cloudflare API Token。本机值放 local-environment skill，**不要**写进本仓库。

- 上传对象：整个 `$MAGIC_HOME`（前缀 `vault/`）。**只把含密钥的文件加密成 `.enc`**，其余明文。
- **不**上传：`backups/`。
- **口令只在 push/pull 需要**（环境变量 / 本地约定），**不**进 R2、不进配置库。对端 pull 解密 `.enc`，落盘为本地明文（无 `.enc`）。
- 自动 push 约定：**库变更后**可挂自动 push；**禁止**在 `apply` 后自动 push。当前 CLI 以手动 `magic sync push|pull` 为主（自动挂接未接上时勿假装已有）。R2 冲突 UI 未做。

## Apply（配置库投影）

产品出口：把库里的一份 skill / MCP / API **按对方格式写进**各 harness 已有文件。写回一律**字段级合并**（结构化文件按键、一文件一项按项），不整文件覆盖；冲突要比记录（A 机 5 个模型 vs B 机 3 个），不是整份 json 当黑盒。

1. 把配置库内容按各 harness 格式写入目标文件（本地密钥明文，无 reinject）  
2. **Skills（仅全局）**：`$MAGIC_HOME/packs/skills` → 各 Caps.Skills harness 目录（symlink → junction → copy）。**永不删除真实目录**  
3. **MCP**：`packs/mcp/mcp.yaml` → 各工具 mcp 字段  
4. **Sub-agents**：`packs/agents/*.md` → 各 Caps.Agents 目录（claude / codex / grok / pi / opencode / copilot / trae）
5. **per-harness 自定义**：`packs/harness/<id>/` 只还回该工具  
6. **Providers**（`magic apply` 末尾；`providers.yaml` 存在且适配器 Detect 通过时）。**Claude / Pi / OpenCode / WorkBuddy** 会投影 active provider。Grok / Codex 走 `magic provider use`：
   - **Claude Code**：非破坏性 upsert `~/.claude/settings.json` 的 `env`：`ANTHROPIC_BASE_URL` + **`ANTHROPIC_AUTH_TOKEN`（主）** + `ANTHROPIC_API_KEY`（同值兼容）；角色 env（`ANTHROPIC_MODEL` / `ANTHROPIC_DEFAULT_*` / `CLAUDE_CODE_SUBAGENT_MODEL`）；可选 `CLAUDE_CODE_EFFORT_LEVEL` / `CLAUDE_CODE_AUTO_COMPACT_WINDOW`（来自 registry meta）。选 `anthropic-messages` endpoint。官方清除全部 Magic 托管 `ANTHROPIC_*` / `CLAUDE_CODE_*`。
   - **Pi**：merge `~/.pi/agent/models.json` 中 `providers.magic-<name>`（`baseUrl` / `api` / `models` from pick；`apiKey` 写**真值**，开箱即用，不走 `$KEY_ENV`）。协议优先 `openai-responses` → `openai-chat-completions`。另写 sidecar `models.magic.json`。每个 pick 会从 `$MAGIC_HOME/registry/models.json` fuzzy 查找并投影 `contextWindow` / `maxTokens` / `thinkingLevelMap` / `cost` / `reasoning` / `input`（缺 meta 时仅写 id+name，不失败）。
   - **OpenCode**：upsert `~/.config/opencode/opencode.json` 的 `provider.magic-<name>`（`options.baseURL` + `apiKey` 真值）与 `model`。官方清除 `magic-*`。
   - **WorkBuddy**：merge `~/.codebuddy/models.json`（`apiKey` 写**真值**，与 Pi 同规则；id 前缀 `magic-`）。官方去掉 Magic 模型行。
   - Active provider：`MAGIC_ACTIVE_PROVIDER` → `providers.active` → 唯一匹配者；多匹配则跳过并提示。

**Apply / `provider use` 元数据：** Pi / Grok / Claude / Codex / WorkBuddy 投影 pick 时读取 `registry/models.json`。最佳效果请先：

```bash
magic registry sync --source models.dev
```

- **Grok**：`context_window` / `max_completion_tokens` / `name` / `reasoning_efforts` 来自 meta（缺省 256000 / 128000 / 固定 effort 列表）。
- **Claude**：AUTH_TOKEN（+ API_KEY 兼容）+ 角色 / `CLAUDE_CODE_*` extras（effort / auto-compact 来自 meta）。
- **Codex（第三方）**：写 Codex-native `~/.codex/models.json`（最小字段：slug / display_name / context_window / max_context_window / input_modalities / supported_reasoning_levels / default_reasoning_level / visibility / supported_in_api）+ `config.toml` 的 `model` / `model_provider` / `model_reasoning_effort` / **`model_catalog_json`** + `[model_providers.<name>]`（`env_key`，**不**写真值 token）。**官方**移除 `model_catalog_json` 并删除 Magic 托管 `models.json`。见 `docs/deepseek-official-integrations.md`。

项目级 Skill 请用 `magic skills sync --scope project`。

## Sub-agent 约定

```text
packs/agents/<name>.md          # 推荐
packs/agents/<name>/AGENT.md    # 或目录整树
```

详见 `packs/agents/README.md`（init 时生成）。

## Demo

```bash
./scripts/demo.sh
# 或：
export MAGIC_HOME=/tmp/magic-lib
./magic init
```

## 版本

`VERSION` / `magic version` = **1.0.0**。

含义：**CLI 引擎与配置库 + apply 投影**（skill / MCP / API → 各 harness）。
文档中标「规划中」的（`magic login`、`magic guide`、桌面端、签名发行）**不属于 1.0.0**。

## 非目标 / 不做

- 整机备份 / 寄修还原 / 钥匙串登录态搬家
- 安装软件；同步会话 / 缓存 / marketplace
- 绑定 Raycast / Alfred / tinycast（跟手壳若做，必须自研）
- Cherry Studio 集成（已取消）
- 把本仓库 / 项目拷到 `Documents` 当「安装」
- `aliases.json` / `patches.json` / registry `meta.json` etags（仍 **FUTURE**）
- TUI、完整 R2 冲突 UI、enterprise denylist UI、Origin push（延期）
- Windsurf 适配器（产品跳过）

**后续产品 UX（不在 1.0.0 CLI 范围）**：Magic **自有** menubar/tray + 全局热键命令面板。本版只交付 CLI/引擎。
