# Harness 配置矩阵（对内）

更新：2026-09-10。各 harness **真实磁盘路径** 与 Magic 两条投影路径的对照。新增适配器先 Adapter，再（可选）SwitchWriter。

## 两条路径

| 路径 | 命令 | 写什么 | 不写什么 |
|------|------|--------|----------|
| **Adapter capture/apply** | `magic capture` / `magic apply` | allow 列表 → `packs/harness/<id>/`；全局 skills / sub-agents | 会话、缓存、OAuth 登录态（除非 overlay 显式收） |
| **Provider SwitchWriter** | `magic provider use`；apply 末尾部分 harness | `providers.yaml` → 方言（base URL / model / key 引用或 Claude 明文 token） | UI 状态、MCP 进程 |

`mcpuser` 只镜像 MCP 配置，**不是**模型 provider，没有 SwitchWriter。

**不做：** windsurf。

---

## 实现对照

| ID | RelHome | Skills | Agents | SwitchWriter | apply 末尾投影 | 备注 |
|----|---------|--------|--------|--------------|----------------|------|
| pi | `.pi/agent` | yes | yes | 是 | 是 | `models.json` `$KEY_ENV` |
| grok | `.grok` | yes | yes | 是 | 否（用 `provider use`） | managed catalog markers |
| claude | `.claude` + `~/.claude.json` partial | yes | yes | 是 | 是 | AUTH_TOKEN 主 |
| codex | `.codex` | yes | yes | 是 | 否（用 `provider use`） | `models.json` + `env_key` |
| mcpuser | `.config/mcp` | — | — | n/a | n/a | 只 `mcp.json` |
| opencode | `.config/opencode` | yes | yes | 是 | 是 | 不收 `~/.local/share/opencode/auth.json` |
| gemini | `.gemini` | — | — | 否 | 否 | deny `.env` / history |
| copilot | `.copilot` | yes | yes | 否 | 否 | deny `config.json` |
| workbuddy | `.codebuddy` | — | — | 是 | 是 | Detect 亦认 `.workbuddy` |
| trae | `.trae` | yes | yes | 否 | 否 | 不镜像 Application Support |

Skills 同步目标（`DefaultAgents`）：`claude, codex, cursor, pi, grok, opencode, copilot, trae`。cursor **仅** skills，无产品适配器。

---

## 各 harness 路径

### pi / grok / claude / codex

与既有 Adapter `CapturePolicy` 一致。Claude `~/.claude.json` 字段级抽取 `mcpServers` 等 → `claude.json.partial.json`。Codex extra allow：`hooks.json`、`themes/**`。Grok extra：`hooks/**`、`commands/**`、`disabled-hooks`。Pi：`themes/**` allow，`models-store.json` deny。

### opencode

| 项 | 路径 |
|----|------|
| 全局配置 | `~/.config/opencode/opencode.json`（或 `.jsonc`） |
| TUI | `tui.json` |
| 扩展 | `agents/` `commands/` `skills/` `themes/` `plugins/` `modes/` `tools/` |
| 凭证 | `~/.local/share/opencode/auth.json`（**不 capture**） |
| 项目 | `<repo>/opencode.json`、`<repo>/.opencode/`（项目 skills：`.opencode/skills`） |

SwitchWriter：`provider.magic-<name>.options.{baseURL,apiKey}` + `model: magic-<name>/<pick>`；official 清除 `magic-*`。第三方 **写明文 apiKey**（与 Claude 同类，需 secrets）。

### gemini

Allow：`settings.json`、`GEMINI.md`、`commands/**`。Deny：`history/**`、`tmp/**`、`.env`、`installation_id`、`projects.json`。无 SwitchWriter。

### copilot（GitHub Copilot CLI）

Allow：`settings.json`、`copilot-instructions.md`、`instructions/**`、`skills/**`、`agents/**`、`hooks/**`、`extensions/**`、`mcp-config.json`、`lsp-config.json`。Deny：`config.json`（含 auth）、logs、session-state、sqlite、mcp-oauth/secrets、ide、installed-plugins。无 SwitchWriter。

### workbuddy（WorkBuddy / CodeBuddy）

Capture / apply 主路径：`~/.codebuddy/models.json`。Detect：`.codebuddy` 或 `.workbuddy`。SwitchWriter 写 `models[]`，`id` 前缀 `magic-`，`apiKey: ${KEY_ENV}`（不写真值）；official 去掉 Magic 条目。

### trae

只镜像 `~/.trae`：`skills/**`、`agents/**`、`rules/**`、`mcp.json`、`sandbox.json`、`settings.json`、`user_rules.md`。不收 `~/Library/Application Support/Trae/User/`。无 SwitchWriter。

### mcpuser

`~/.config/mcp/mcp.json`。与 Windsurf 等 IDE 自有 MCP 文件分开；通用用户 MCP 走这里。

---

## SwitchWriter 方言摘要

| Harness | 第三方写入 | Key | official |
|---------|------------|-----|----------|
| Claude | `~/.claude/settings.json` env | 明文 AUTH_TOKEN（+ API_KEY 兼容） | 清 Magic 托管 `ANTHROPIC_*` / `CLAUDE_CODE_*` |
| Pi | `models.json` `providers.magic-*` | `$KEY_ENV` | 删 Magic provider |
| Codex | `config.toml` + `models.json` | `env_key` | 去 catalog、删 Magic models.json |
| Grok | managed catalog markers | env_key 类 | 清 Magic 块 |
| OpenCode | `opencode.json` `provider.magic-*` | 明文 `options.apiKey` | 清 `magic-*` provider / model |
| WorkBuddy | `~/.codebuddy/models.json` | `${KEY_ENV}` | 去掉 `magic-*` 模型行 |

---

## R2

Magic 官方多机桶名约定：**`magic-ai`**，对象前缀 `vault/`。环境变量 `MAGIC_R2_BUCKET` / `MAGIC_R2_ACCESS_KEY_ID` / `MAGIC_R2_SECRET_ACCESS_KEY` / `MAGIC_R2_ENDPOINT`。本机值只放 local-environment `secrets.md`，不进本仓库。
