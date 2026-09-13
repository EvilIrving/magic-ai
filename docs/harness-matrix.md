# Harness 配置矩阵（对内）

更新：2026-09-11。各 harness **真实磁盘路径** 与 Magic 两条投影路径的对照。新增适配器先 Adapter，再（可选）SwitchWriter。
已锁定的产品决定见 [decisions.md](./decisions.md)。

## 两条路径

目的是 **配置库里的一份 skill / MCP / API → 各 harness 原生文件**。进库靠手改 / `skills` / `catalog` / `secrets`（capture 已删）；apply / SwitchWriter 是出口。

| 路径 | 命令 | 写什么 | 不写什么 |
|------|------|--------|----------|
| **进配置库** | 手改；`skills` / `catalog` / `secrets` | skill / MCP / API 变成配置库里的 Magic 数据 | 会话、缓存、`npm/`、OAuth 登录态 |
| **Apply 投影** | `magic apply`；`magic skills sync` | 配置库 → 各 harness 已有文件 | 同上 |
| **Provider SwitchWriter** | `magic provider use`；apply 末尾部分 harness | `providers.yaml` → 方言（base URL / model / 真值 key；Codex 用原生 `env_key`） | UI 状态、MCP 进程 |

`mcpuser` 只投 MCP 配置，**不是**模型 provider，没有 SwitchWriter。

**不做：** windsurf（产品濒临停运 / 已被收购）、roo（2026-05 停运）。

---

## 实现对照

| ID | RelHome | Skills | Agents | SwitchWriter | apply 末尾投影 | 备注 |
|----|---------|--------|--------|--------------|----------------|------|
| pi | `.pi/agent` | yes | yes | 是 | 是 | `models.json` 明文 apiKey 真值 |
| grok | `.grok` | yes | yes | 是 | 否（用 `provider use`） | managed catalog markers |
| claude | `.claude` + `~/.claude.json` partial | yes | yes | 是 | 是 | AUTH_TOKEN 主 |
| codex | `.codex` | yes | yes | 是 | 否（用 `provider use`） | `models.json` + `env_key` |
| mcpuser | `.config/mcp` | — | — | n/a | n/a | 只 `mcp.json` |
| opencode | `.config/opencode` | yes | yes | 是 | 是 | 不收 `~/.local/share/opencode/auth.json` |
| gemini | `.gemini` | — | — | 否 | 否 | 不碰 `.env` / history |
| copilot | `.copilot` | yes | yes | 否 | 否 | 不碰 `config.json`（含 auth） |
| workbuddy | `.codebuddy` | — | — | 是 | 是 | Detect 亦认 `.workbuddy` |
| trae | `.trae` | yes | yes | 否 | 否 | 不碰 Application Support |
| cursor | `.cursor` | yes | — | 否 | 否 | skills/MCP/hooks；无 model writer（D20） |
| antigravity-cli | `.gemini/antigravity-cli` | yes | — | 否 | 否 | MCP 在 `~/.gemini/config/mcp_config.json` |
| kimi-code | `.kimi-code` | yes | yes | 否 | 否 | MCP `~/.kimi-code/mcp.json` |
| qoder / qoder-cn | `.qoder` / `.qoder-cn` | yes | yes | 否 | 否 | settings.json 含 mcp |
| cline | `.cline` | yes | — | 否 | 否 | MCP `~/.cline/mcp.json`；skills `.agents` |
| kiro | `.kiro` | yes | yes | 否 | 否 | MCP `~/.kiro/settings/mcp.json` |
| droid | `.factory` | yes | yes | 否 | 否 | MCP `~/.factory/mcp.json` |
| zcode | `.zcode` | yes | — | 否 | 否 | MCP `~/.zcode/cli/config.json` `mcp.servers` |
| omp | `.omp/agent` | yes | yes | 否 | 否 | MCP `mcp.json`；config.yml/models.yml |
| hermes | `.hermes` | yes | — | 否 | 否 | MCP `config.yaml` `mcp_servers`（YAML） |
| mimocode | `.config/mimocode` | yes | yes | 否 | 否 | OpenCode 方言；`mimocode.json` `mcp` |
| dimagent | `.dim/v2` | — | — | 否 | 否 | MCP `mcp.json`；skills 目录未证实 |
| openclaw | `.openclaw` | yes | — | 否 | 否 | MCP `openclaw.json` `mcp.servers`（JSON5） |

Skills 同步目标（`DefaultAgents`）：`claude, codex, cursor, pi, grok, opencode, copilot, trae, antigravity-cli, qoder, qoder-cn, cline, kimi-code, workbuddy, omp, hermes, mimocode, openclaw`。

---

## 各 harness 路径

### pi / grok / claude / codex

apply 写各 harness 已有文件；Magic 不反向收录（capture 已删）。Claude 额外把 `packs/harness/claude/claude.json.partial.json` 合并进 `~/.claude.json`。

### opencode

| 项 | 路径 |
|----|------|
| 全局配置 | `~/.config/opencode/opencode.json`（或 `.jsonc`） |
| TUI | `tui.json` |
| 扩展 | `agents/` `commands/` `skills/` `themes/` `plugins/` `modes/` `tools/` |
| 凭证 | `~/.local/share/opencode/auth.json`（不碰） |
| 项目 | `<repo>/opencode.json`、`<repo>/.opencode/`（项目 skills：`.opencode/skills`） |

SwitchWriter：`provider.magic-<name>.options.{baseURL,apiKey}` + `model: magic-<name>/<pick>`；official 清除 `magic-*`。第三方 **写明文 apiKey**（与 Claude 同类，需 secrets）。

### gemini

> ⚠️ **Gemini CLI 已被 Antigravity CLI 取代**（2026-06-18，官方博客 + 迁移文档）。
> 新目标应为 `antigravity-cli`（`agy`，配置在 `~/.gemini/antigravity-cli/`，MCP 在 `~/.gemini/config/mcp_config.json`，
> remote 键是 `serverUrl`）。`gemini` adapter 保留作过渡兼容，不再新增能力。

写：`settings.json`、`GEMINI.md`、`commands/**`。不碰：`history/**`、`tmp/**`、`.env`、`installation_id`、`projects.json`。无 SwitchWriter。

### copilot（GitHub Copilot CLI）

写：`settings.json`、`copilot-instructions.md`、`instructions/**`、`skills/**`、`agents/**`、`hooks/**`、`extensions/**`、`mcp-config.json`、`lsp-config.json`。不碰：`config.json`（含 auth）、logs、session-state、sqlite、mcp-oauth/secrets、ide、installed-plugins。无 SwitchWriter。

### workbuddy（WorkBuddy = CodeBuddy）

**同一产品，共用 `~/.codebuddy/`**（WorkBuddy 文档直接引用 `~/.codebuddy/models.json`）。
目录：用户级 `~/.codebuddy/settings.json`（含 hooks / mcp / plugins）+ `models.json`；项目级 `.codebuddy/`。
Apply 主路径：`~/.codebuddy/models.json`。
SwitchWriter 写 `models[]`，`id` 前缀 `magic-`，`apiKey` 写**真值**；official 去掉 Magic 条目。

### trae

只投 `~/.trae`：`skills/**`、`agents/**`、`rules/**`、`mcp.json`、`sandbox.json`、`settings.json`、`user_rules.md`。不碰 `~/Library/Application Support/Trae/User/`。无 SwitchWriter。

### mcpuser

`~/.config/mcp/mcp.json`。通用用户 MCP 走这里（与各 harness 自有的 MCP 文件分开）。

---

## SwitchWriter 方言摘要

| Harness | 第三方写入 | Key | official |
|---------|------------|-----|----------|
| Claude | `~/.claude/settings.json` env | 明文 AUTH_TOKEN（+ API_KEY 兼容） | 清 Magic 托管 `ANTHROPIC_*` / `CLAUDE_CODE_*` |
| Pi | `models.json` `providers.magic-*` | 明文 apiKey 真值 | 删 Magic provider |
| Codex | `config.toml` + `models.json` | `env_key`（Codex 原生；不写真值 token） | 去 catalog、删 Magic models.json |
| Grok | managed catalog markers | env_key 类 | 清 Magic 块 |
| OpenCode | `opencode.json` `provider.magic-*` | 明文 `options.apiKey` 真值 | 清 `magic-*` provider / model |
| WorkBuddy | `~/.codebuddy/models.json` | 明文 apiKey 真值 | 去掉 `magic-*` 模型行 |

Pi / WorkBuddy / OpenCode / Claude 都写**真值** key，不用 `$KEY_ENV`。Codex 用其原生 `env_key`。
含密钥的 harness 文件本地同样明文；云端以 `.enc` 存。见 [secrets-and-sync.md](./secrets-and-sync.md)（D2 / D3）。

---

## R2

Magic 官方多机桶名约定：**`magic-ai`**，对象前缀 `vault/`。环境变量 `MAGIC_R2_BUCKET` / `MAGIC_R2_ACCESS_KEY_ID` / `MAGIC_R2_SECRET_ACCESS_KEY` / `MAGIC_R2_ENDPOINT`。本机值只放 local-environment `secrets.md`，不进本仓库。
