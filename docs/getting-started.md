# Magic 入门

Magic（仓库 [magic-ai](https://github.com/EvilIrving/magic-ai)）是给 **Claude Code、Codex、Cursor、Grok、OpenCode、Pi** 用的 Go CLI：一份本地配置库同步 **skills、MCP、中转 API**。你只维护 `~/.magic-ai`。`apply` 按各工具自己的格式写进已有配置，不必每个 Agent 再复制粘贴。

多机备份的是这份配置库（R2 或 git 私有库），不是 `~/.pi` / `~/.grok` 家目录。

产品概念不复杂，难的是 **按各 harness 自己的格式写进去**（skill 目录、MCP 字段、中转协议都不一样）。

两类内容：

| 层 | 例子 | 怎么处理 |
|---|---|---|
| 跨 harness | skill、MCP、中转 API / 模型 | 库里一份；apply 译成对方格式写进已有文件 |
| 某工具自己的自定义 | Pi `themes/**`、`AGENTS.md` | 随库跨机，只还回同一个 harness |

手改 `~/.magic-ai` 或用 CLI 命令是进库的手段。官方登录 / access token **不进库**。

**不做：** 装软件、同步会话/缓存、绑定 Raycast/Alfred、把 harness 家目录整树当产品。

## 安装（源码）

```bash
git clone https://github.com/EvilIrving/magic-ai.git
cd magic-ai
go test ./...
go build -o magic ./cmd/magic
./magic version
```

把 `magic` 放进 `PATH`，或在仓库里用 `./magic`。

## 五分钟

```bash
export MAGIC_HOME=~/.magic-ai   # 可选；这就是默认值
magic init
magic adapters                  # 已实现的 harness
magic status

magic skills init
magic skills list
magic skills sync                 # 全局：packs/skills → 各工具 skills 目录
magic skills sync --scope project # 项目：<repo>/.agents/skills → .claude|…/skills

magic apply --dry-run
magic apply               # 出口：配置库 → 各 harness
```

## 心智模型

| 动作 | 方向 |
|------|------|
| 手改 / `skills` / `catalog` / `secrets` | 进 **`~/.magic-ai`**（skills / MCP / API） |
| `apply` | **配置库** → 各 harness 原生文件 |
| `sync push` | 配置库 → Cloudflare R2 桶 `magic-ai`（前缀 `vault/`） |
| `sync pull` | R2 → 配置库 |

- **全局 Skill 真相：** `$MAGIC_HOME/packs/skills`（不是 `~/.agents/skills`）。
- **项目 Skill：** `<repo>/.agents/skills`，只用 `magic skills sync --scope project`，**不是** `magic apply`。
- **sub-agent：** `$MAGIC_HOME/packs/agents/*.md`，apply 写入各工具 `…/agents/`。
- **MCP：** `$MAGIC_HOME/packs/mcp/mcp.yaml`。
- **Providers：** `$MAGIC_HOME/providers.yaml` + 本地明文 `$MAGIC_HOME/secrets/providers.json`（0600）。本地无 `.enc`。加密只在 `sync push`（云端对象为 `.enc`）；口令不进库。
- **per-harness 自定义：** `packs/harness/<id>/`，如 Pi `themes/**`、`AGENTS.md`，随库跨机，只还回该 harness。
- **布局：** [library-layout.md](./library-layout.md)。

## 支持的 harness

Harness = 要往里面写配置的 AI coding 工具。用户配的是 `~/.magic-ai`，不是这些家目录。

| 适配器 | apply 投影（API / 模型） | Skills 同步 | MCP |
|--------|--------------------------|-------------|-----|
| pi / grok / claude / codex | 是（Grok/Codex 日常用 `provider use`） | 是 | 是 |
| opencode | 是 | 是（`~/.config/opencode/skills`） | 是 |
| workbuddy | 是（`~/.codebuddy/models.json`） | — | 是 |
| gemini | 否（OAuth / `.env`） | — | 是（过渡，新目标 antigravity-cli） |
| copilot | 否（GitHub 账号） | 是（`~/.copilot/skills`） | 是 |
| trae | 否 | 是 | 是 |
| cursor / antigravity-cli / kimi-code / qoder / qoder-cn / cline | 否（账号 / UI） | 是 | 是 |

覆盖范围按 OpenRouter app 用量分档，v1 核心 + 推迟长尾见
[supported-harnesses.md](./supported-harnesses.md)（D20）。
| mcpuser | 否（不是模型 provider） | — | 目标是 `~/.config/mcp/mcp.json` |

Skills 目标含 `cursor` 等全量路径表；v1 核心 harness 见 [supported-harnesses.md](./supported-harnesses.md)（用量分档，D20）。**不做 windsurf。**

路径细节见 [harness-matrix.md](./harness-matrix.md)。

## Provider 切换

方言已实现：**Claude、Codex、Grok、Pi、OpenCode、WorkBuddy**。

```bash
magic provider seed
magic secrets set SHU_API_KEY --value '…'   # 本地明文，无需口令
magic provider use official
magic provider use company                  # = providers.active
magic provider use shu --harness opencode
magic provider test company
```

`magic apply` 在探测到对应工具时，还会把 active provider 投影到 **Claude、Pi、OpenCode、WorkBuddy**。Grok / Codex 用 `magic provider use`。

## 多机（R2）

需要环境变量（S3 兼容 API，不是 Wrangler 登录态）：

```bash
export MAGIC_R2_BUCKET=magic-ai
export MAGIC_R2_ACCESS_KEY_ID=…
export MAGIC_R2_SECRET_ACCESS_KEY=…
export MAGIC_R2_ENDPOINT=https://<ACCOUNT_ID>.r2.cloudflarestorage.com
magic sync status
magic sync push
magic sync pull
```

- 上传整个配置库到前缀 `vault/`；**只把含密钥的文件加密成 `.enc`**，其余明文。跳过 `backups/`。
- **口令只在 push/pull 需要**，不进 R2、不进配置库；pull 解密 `.enc` 后落回本地明文（无 `.enc`）。
- 冲突检测、变更后自动 push **尚未实现**；当手动备份/同步通道用。
- 本机密钥位置见 local-environment skill 的 `secrets.md`（不要写进本仓库）。

## 安全

- `magic apply` 把库写进 harness 已有文件；无交互确认。冲突要比模型/skill 记录，不是整份 json 当黑盒（检测未做）。
- Skills 同步 **永不删除真实目录**；冲突见 `magic skills status` / `doctor`。
- 中转 API key 走 `magic secrets`（本地明文 `secrets/providers.json`）。`.enc` 是云端状态，不是本地文件。不要把 access token / OAuth 登录态放进配置库。

## 接下来读

| 读者 | 文档 |
|------|------|
| 日常命令 | [README.md](../README.md) |
| 配置库布局 | [library-layout.md](./library-layout.md) |
| 文档目录 | [README.md](./README.md) |
| 协作者约定 | [AGENTS.md](../AGENTS.md) |
| Harness 路径 | [harness-matrix.md](./harness-matrix.md) |

**1.0.0** 是 CLI 引擎正式版。延期：自研菜单栏/热键、`magic login` 的 TUI、R2 冲突 UI、registry aliases/patches。
