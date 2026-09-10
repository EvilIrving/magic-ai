# Magic 入门

Magic 是 **AI coding harness 的用户级配置管理器**（Go CLI）。本地 **配置库**（默认 `~/.magic-ai`）是真相源：Skills、sub-agent、providers、密钥和部分 harness 配置从这里收进、再投影回各工具。

**不做：** 装软件、整机备份、同步会话/缓存、绑定 Raycast/Alfred。

## 安装（源码）

```bash
git clone <this-repo> && cd go-magic
go test ./...
go build -o magic ./cmd/magic
./magic version   # 1.0.0
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

magic capture --dry-run
magic capture
magic apply --dry-run
magic apply
```

## 心智模型

| 动作 | 方向 |
|------|------|
| `capture` | 本机 → **配置库**（库变了） |
| `apply` | **配置库** → 本机（库不变） |
| `sync push` | 配置库 → Cloudflare R2 桶 `magic-ai`（前缀 `vault/`） |
| `sync pull` | R2 → 配置库 |

- **全局 Skill 真相：** `$MAGIC_HOME/packs/skills`（不是 `~/.agents/skills`）。
- **项目 Skill：** `<repo>/.agents/skills`，只用 `magic skills sync --scope project`，**不是** `magic apply`。
- **sub-agent：** `$MAGIC_HOME/packs/agents/*.md`，apply 写入各工具 `…/agents/`。
- **Providers：** `$MAGIC_HOME/providers.yaml` + 加密 `$MAGIC_HOME/secrets/providers.json.enc`（需 `MAGIC_PASSPHRASE`）。

## 支持的 harness

| 适配器 | capture / apply | Skills 同步 | 官方 ↔ 中转（`provider use`） |
|--------|-----------------|-------------|-------------------------------|
| pi / grok / claude / codex | 是 | 是 | 是 |
| opencode | 是 | 是（`~/.config/opencode/skills`） | 是 |
| workbuddy | 是 | — | 是（`~/.codebuddy/models.json`） |
| gemini | 是 | — | 否（OAuth / `.env`） |
| copilot | 是 | 是（`~/.copilot/skills`） | 否（GitHub 账号） |
| trae | 是（只收 `~/.trae`） | 是 | 否 |
| mcpuser | 是（`~/.config/mcp/mcp.json`） | — | 否（不是模型 provider） |

Skills 目标另含 **cursor**（无产品适配器）。**不做 windsurf。**

路径细节见 [harness-matrix.md](./harness-matrix.md)。

## Provider 切换

方言已实现：**Claude、Codex、Grok、Pi、OpenCode、WorkBuddy**。

```bash
magic provider seed
magic secrets set SHU_API_KEY --value '…'   # 需要 MAGIC_PASSPHRASE
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

- 上传整个配置库到前缀 `vault/`（含 `secrets/*.enc`），跳过 `backups/`。
- **`MAGIC_PASSPHRASE` 只留本机**，不进 R2。
- 冲突检测、capture 后自动 push **尚未实现**；当手动备份/同步通道用。
- 本机密钥位置见 local-environment skill 的 `secrets.md`（不要写进本仓库）。

## 安全

- `magic apply` **备份后覆盖**已镜像的本机文件，无交互确认。
- Skills 同步 **永不删除真实目录**；冲突见 `magic skills status` / `doctor`。
- 密钥走 `magic secrets` + capture redact / apply reinject，不要把明文 key 放进配置库。

## 接下来读

| 读者 | 文档 |
|------|------|
| 日常命令 | [README.md](../README.md) |
| 文档目录 | [README.md](./README.md) |
| 协作者约定 | [AGENTS.md](../AGENTS.md) |
| Harness 路径 | [harness-matrix.md](./harness-matrix.md) |

**1.0.0** 是 CLI 引擎正式版。延期：自研菜单栏/热键、TUI、R2 冲突 UI、registry aliases/patches。
