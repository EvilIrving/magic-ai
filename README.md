# Magic

**1.0.0** — AI harness 配置管理器（CLI 引擎）。

- **文档目录：** [docs/README.md](./docs/README.md)
- **对外入门：** [docs/getting-started.md](./docs/getting-started.md)
- **对内约定：** [AGENTS.md](./AGENTS.md)
- **Harness 矩阵：** [docs/harness-matrix.md](./docs/harness-matrix.md)

`magic` 管理 AI coding harness（Pi / Grok / Claude / Codex / OpenCode / Gemini / Copilot / WorkBuddy / Trae / MCP user）的**用户级配置**。  
真相源与工作目录是本地 **配置库 `~/.magic-ai`**（`MAGIC_HOME` 覆盖；别名 `MAGIC_VAULT`）。  
不做整机备份，不安装软件。

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
| **配置库** | `~/.magic-ai`：Skill、sub-agent、镜像配置、providers、secrets 的唯一真相源 |
| **Skill（全局）** | `$MAGIC_HOME/packs/skills/<name>/`；用 `magic skills sync` 或 `magic apply` 投影 |
| **Skill（项目）** | `<project>/.agents/skills/`；仅 `magic skills … --scope project` |
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

创建：`magic.yaml`、`providers.yaml`（示例）、`packs/skills`、`packs/agents`、`packs/harness`、`secrets/`、`capture.overlay.yaml`、`README.md`。

## 常用命令

```bash
magic status
magic adapters
magic capture [--dry-run] [--only pi] [--home /path/fixture]
magic capture approve pi my-extra.toml
magic apply [--dry-run] [--home "$HOME"]   # 镜像恢复 + 全局 packs/skills + sub-agents

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
magic copy deepseek              # providers.yaml → stdout (key from secrets/providers.json.enc)
magic copy deepseek --reveal-key # full api key (default redacts middle)
magic copy shu --protocol anthropic-messages   # pick endpoint by protocol
magic copy shu --endpoint openai-responses     # pick endpoint by id
magic secrets set DEEPSEEK_API_KEY --value 'sk-…'   # store key (needs MAGIC_PASSPHRASE)
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

多协议 relay（同一 `key_env` + `pick`，多条 `endpoints[]`）：

```yaml
active: shu   # optional; override with MAGIC_ACTIVE_PROVIDER
providers:
  shu:
    key_env: SHU_API_KEY
    pick: [gpt-5.6-sol]
    endpoints:
      - id: openai-responses
        base_url: https://api.shu.cool/v1
        protocol: openai-responses
      - id: openai-chat
        base_url: https://api.shu.cool/v1
        protocol: openai-chat-completions
      - id: anthropic
        base_url: https://api.shu.cool
        protocol: anthropic-messages
```

- 协议常量：`openai-chat-completions`、`openai-responses`、`anthropic-messages`（别名如 `openai` / `anthropic` 可接受）。
- 兼容：若 `endpoints` 为空，顶层 `base_url` + `protocol` 合成一条 endpoint。
- `active` / `MAGIC_ACTIVE_PROVIDER`：`magic apply` / `magic provider use company` 选用哪个第三方；多候选且未指定则跳过并列出。`company` **只是** active 别名。

### Copy + provider secrets

`magic copy <provider>` 从 `$MAGIC_HOME/providers.yaml` 抽出该 provider 的非密钥字段到 stdout（clipboard-friendly）。

- `key_env` 是逻辑索引名（如 `DEEPSEEK_API_KEY`），**不是** process env 变量名；真值存在加密的 `$MAGIC_HOME/secrets/providers.json.enc`。
- 密钥解析（copy）：用 `MAGIC_PASSPHRASE` 解密 `secrets/providers.json.enc` → 按 `key_env` 查找（fallback：provider 名）→ 有则打印（默认 redact 中间：首 4…末 4；`--reveal-key` 全文）→ 无则占位符 `<DEEPSEEK_API_KEY>` + 一行缺失提示。
- **不**读 OS / process env 取 copy key（无 env override）。
- Endpoint 选择：默认第一条；`--protocol <name>` 或 `--endpoint <id>` 指定。
- Adapter reinject 仍用 `secrets/<adapterID>.json.enc`（capture/apply）；与 provider map 并存。
- 写入：`magic secrets set <key_env> --value …`（或 stdin）；`magic secrets list` 只列名。
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

- 上传对象：整个 `$MAGIC_HOME`（前缀 `vault/`），**含**加密的 `secrets/*.enc`（adapter reinject + `providers.json.enc`）。
- **不**上传：`backups/`。
- **`MAGIC_PASSPHRASE` 只留本机**（环境变量 / 本地约定），**不**进 R2、不进配置库明文。对端 pull 后用同一口令解密。
- 自动 push 约定：**库变更后**（如 `capture` 成功）可挂自动 push；**禁止**在 `apply` 后自动 push。当前 CLI 以手动 `magic sync push|pull` 为主（自动挂接未接上时勿假装已有）。R2 冲突 UI 未做。

## Capture

按适配器 builtin allow/deny + `capture.overlay.yaml` 分类，allow 写入 `packs/harness/<id>/`。  
密钥字段红acted 为 `${MAGIC_SECRET:…}`，有 `MAGIC_PASSPHRASE` 时密文进 `secrets/<id>.json.enc`。

Claude：`~/.claude.json` **字段级抽取**（至少 `mcpServers` 等配置键）→  
`packs/harness/claude/claude.json.partial.json`，不整文件镜像庞大 runtime/project 状态。  

**Local config refs（通用）：** allow 的 `settings.json` / `settings.local.json` 会解析本地路径引用并 force-allow 存在的文件（`settings.ref`）；Codex 的 `config.toml` 保守抽取 harness 根下路径（`config.ref`，忽略 URL / 裸二进制）。库内 settings 会把 `$HOME/<adapter RelHome>` 折成 `~/…` 以便跨机。  

**Allow 扩展：** Codex 另收 `hooks.json`、`themes/**`；Grok 另收 `hooks/**`、`commands/**`、`disabled-hooks`。

**Pi：** `themes/**` 在 allow；`models-store.json` 在 deny；`memory` 等未列项按 unknown **默认不收**（可用 `capture.overlay.yaml` / `magic capture approve` 显式纳入）。

**OpenCode：** `~/.config/opencode`（json/jsonc、skills/agents/themes…）；**不**收 `~/.local/share/opencode/auth.json`。  
**Gemini：** `~/.gemini` 的 `settings.json` / `GEMINI.md` / `commands/**`；deny `.env` 与 history。  
**Copilot：** `~/.copilot` 用户可编辑项；**deny** `config.json`（含登录态）。  
**WorkBuddy：** `~/.codebuddy/models.json`（Detect 亦认 `~/.workbuddy`）。  
**Trae：** 只收 `~/.trae`（skills/mcp/sandbox 等），不镜像 Application Support。  
**mcpuser：** `~/.config/mcp/mcp.json`。

## Apply（配置库投影）

1. 恢复镜像树，并按 secrets **reinject** 占位符  
2. **Skills（仅全局）**：`$MAGIC_HOME/packs/skills` → 各 Caps.Skills harness 目录（symlink → junction → copy）。**永不删除真实目录**  
3. **Sub-agents**：`packs/agents/*.md` → 各 Caps.Agents 目录（claude / codex / grok / pi / opencode / copilot / trae）
4. **Providers**（`magic apply` 末尾；`providers.yaml` 存在且适配器 Detect 通过时）。**Claude / Pi / OpenCode / WorkBuddy** 会投影 active provider。Grok / Codex 走 `magic provider use`：
   - **Claude Code**：非破坏性 upsert `~/.claude/settings.json` 的 `env`：`ANTHROPIC_BASE_URL` + **`ANTHROPIC_AUTH_TOKEN`（主）** + `ANTHROPIC_API_KEY`（同值兼容）；角色 env（`ANTHROPIC_MODEL` / `ANTHROPIC_DEFAULT_*` / `CLAUDE_CODE_SUBAGENT_MODEL`）；可选 `CLAUDE_CODE_EFFORT_LEVEL` / `CLAUDE_CODE_AUTO_COMPACT_WINDOW`（来自 registry meta）。选 `anthropic-messages` endpoint。官方清除全部 Magic 托管 `ANTHROPIC_*` / `CLAUDE_CODE_*`。
   - **Pi**：merge `~/.pi/agent/models.json` 中 `providers.magic-<name>`（`baseUrl` / `api` / `models` from pick；`apiKey` 写成 `$KEY_ENV`，**不**写真值）；协议优先 `openai-responses` → `openai-chat-completions`。另写 sidecar `models.magic.json`。每个 pick 会从 `$MAGIC_HOME/registry/models.json` fuzzy 查找并投影 `contextWindow` / `maxTokens` / `thinkingLevelMap` / `cost` / `reasoning` / `input`（缺 meta 时仅写 id+name，不失败）。
   - **OpenCode**：upsert `~/.config/opencode/opencode.json` 的 `provider.magic-<name>`（`options.baseURL` + `apiKey` 明文）与 `model`。官方清除 `magic-*`。
   - **WorkBuddy**：merge `~/.codebuddy/models.json`（`apiKey: ${KEY_ENV}`，id 前缀 `magic-`）。官方去掉 Magic 模型行。
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
./magic capture --dry-run --home testdata/demo-home --only pi
```

## 版本

`VERSION` / `magic version` = **1.0.0**（formal）。

## 非目标 / 不做

- 整机备份 / 寄修还原 / 钥匙串登录态搬家
- 安装软件；同步会话 / 缓存 / marketplace
- 绑定 Raycast / Alfred / tinycast（跟手壳若做，必须自研）
- Cherry Studio 集成（已取消）
- 把本仓库 / 项目拷到 `Documents` 当「安装」
- `aliases.json` / `patches.json` / registry `meta.json` etags（仍 **FUTURE**）
- TUI、完整 R2 冲突 UI、enterprise denylist UI、Origin push（延期）
- Windsurf 适配器（产品跳过）

**后续产品 UX（不在 1.0.0 formal CLI 范围）**：Magic **自有** menubar/tray + 全局热键命令面板。本版只交付 CLI/引擎。
