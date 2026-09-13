# MCP 投影设计（多 harness）

状态：**设计稿（调研完成，未实现）**。调研日期 **2026-09-11**，来源见文末。
目标：库里一份 `packs/mcp/mcp.yaml`，按各 harness 格式写进它们的 MCP 配置。

## 结论

MCP 是七能力之一（见 [supported-harnesses.md](./supported-harnesses.md)）。各家**字段名、文件、传输表示都不同**，
必须逐家翻译。但**多数家共用 `mcpServers` + `{command,args,env}` / `{url,headers}` 形状**，
所以可以定一个 Magic 规范形，再为少数例外写规则。

**不支持 MCP 或无法写配置的 harness 跳过，不报错**（能力不对等）。

## Magic 规范形（canonical）

库里只维护这一份，翻译到各家：

```yaml
# ~/.magic-ai/packs/mcp/mcp.yaml
mcp:
  context7:
    transport: stdio
    command: npx
    args: ["-y", "@upstash/context7-mcp@latest"]
    env:
      API_KEY: "..."          # 本地明文；值可写 ${ENV} 由各 harness 自己展开
    enabled: true

  linear:
    transport: http           # http | sse
    url: https://mcp.linear.app/mcp
    headers:
      Authorization: "Bearer ..."
    enabled: true
```

字段：`transport`(stdio|http|sse)、stdio 用 `command/args/env/cwd`、remote 用 `url/headers`、`enabled`。

## 各家落点与格式

| harness | 文件 | 顶层键 | stdio | remote | 备注 |
|---|---|---|---|---|---|
| **Claude Code** | `~/.claude.json`（用户）/ `.mcp.json`（项目） | `mcpServers` | `{command,args,env}` | `{type:"http"\|"sse", url, headers}` | 有 `url` 就必须有 `type`；`streamable-http` 是 `http` 的别名 |
| **Codex** | `~/.codex/config.toml` | `[mcp_servers.<name>]` | `command,args,env` | `url`（streamable http） | TOML；`enabled`；SSE 不支持（需代理） |
| **Gemini CLI** | `~/.gemini/settings.json` | `mcpServers` | `{command,args,env,cwd,timeout,trust}` | SSE `{url,headers}`；HTTP **`{httpUrl,headers}`** | ⚠️ **已被 Antigravity CLI 取代**（2026-06-18）；保留作过渡 |
| **Antigravity CLI** | `~/.gemini/config/mcp_config.json`（全局） / `.agents/mcp_config.json`（工作区） | `mcpServers` | `{command,args,env}` | **`{serverUrl,env}`** —— remote 键是 **`serverUrl`**（不是 `url`/`httpUrl`） | MCP 拆到**独立文件**，不再内联进 settings |
| **Trae** | `~/.trae/mcp.json`（国内版 `~/.trae-cn/mcp.json`）/ 项目根 `mcp.json` | `mcpServers` | `{command,args,env}` | `{url,headers}` | 基于 VS Code 的 IDE，也可在 UI 里配 |
| **Cursor** | `~/.cursor/mcp.json` / `.cursor/mcp.json` | `mcpServers` | `{command,args,env}` | `{url,headers}`；或 `{url,auth:{CLIENT_ID,CLIENT_SECRET,scopes}}` | remote 不需要 `type` |
| **OpenCode** | `~/.config/opencode/opencode.json` | **`mcp`** | `{type:"local", command:[...], enabled}` | `{type:"remote", url, headers, enabled}` | 键名是 `mcp`；local 的 `command` 是**数组** |
| **Copilot CLI** | `~/.copilot/mcp-config.json` | `mcpServers` | `{type:"local", command,args,env, tools,timeout}` | `{type:"http"\|"sse", url,headers, tools,timeout}` | `tools` 白名单 |
| **VS Code** | `.vscode/mcp.json` / 用户 profile `mcp.json` | **`servers`** | `{type:"stdio", command,args,env}` | `{type:"http", url,headers}` | 键名是 `servers`；有 `inputs` 机制藏密钥 |
| **Cline** | `~/.cline/mcp.json` / 扩展 globalStorage `cline_mcp_settings.json` | `mcpServers` | `{command,args,env, disabled,autoApprove}` | `{type:"streamableHttp"\|"sse", url,headers}` | transport 名不同；`autoApprove` |
| **Grok** | `~/.grok/config.toml` | `[mcp_servers.<name>]` | `command,args,env, startup_timeout_sec,tool_timeout_sec` | `url,headers` | `${VAR}` 展开；**也兼容读** `~/.claude.json`、`.cursor/mcp.json`、`.mcp.json` |
| **Pi** | `~/.pi/agent/mcp.json` | `mcpServers` | `{command,args,env, disabled}` | ❓（remote 靠 `pi-mcp-adapter` 第三方包） | Pi 还读 `~/.config/mcp/mcp.json`、`.mcp.json`、`~/.agents/mcp.json` |
| **Qoder CLI** | `~/.qoder/settings.json` | `mcpServers` | `{command,args,env}` | `{type:"http"\|"sse"\|"ws", url, headers}` | 作用域 user/project/local；`qoder mcp add` |
| **Kimi Code CLI** | `~/.kimi-code/mcp.json`（或 `$KIMI_CODE_HOME`） | `mcpServers` | `{command,args,env,cwd}` | `{url}`（HTTP）或 `{transport:"sse",url}`；`headers` / `bearerTokenEnvVar` | 项目级 `.kimi-code/mcp.json` 覆盖；`enabled`/`enabledTools` |
| **CodeBuddy (= WorkBuddy)** | `~/.codebuddy/settings.json`（用户）/ `.codebuddy/settings.json`（项目） | `mcpServers` | `{command,args,env}` | `{url,headers}` | 与 hooks / plugins 同一 settings；工具权限 `mcp__server__tool` |
| **Kilo** | `~/.config/kilo/config.json`（MCP 另读 `.kilocode/mcp.json`，**深合并**） | **`mcp`** | `{command,args,env}` | `{url,headers}` | CLI 基于 OpenCode（`kilo mcp add` 会写 `opencode.jsonc`） |
| **iFlow CLI** | `~/.iflow/settings.json` | `mcpServers` | `{command,args,env}` | `{url,headers}` | 有 JSON Schema；支持工具白名单 |

## 翻译规则（规范形 → 各家）

三处最容易踩坑，按 harness 分派：

1. **顶层键名**：`mcpServers`（Claude/Codex 之外的多数）/ `servers`（VS Code）/ `mcp`（OpenCode）/
   `[mcp_servers]` TOML（Codex、Grok）。
2. **remote 传输字段**：
   - `type:"http"`：Claude Code、Copilot CLI、VS Code
   - `httpUrl`：Gemini CLI 的 HTTP（`url` 留给 SSE）
   - 无 `type`、直接 `url`：Cursor、Codex、Grok
   - `type:"streamableHttp"`：Cline
   - `type:"remote"`：OpenCode
3. **文件格式**：JSON（多数）/ TOML（Codex、Grok）。

其余同名直译：`command/args/env/url/headers`。`enabled` 各家叫法不一
（Codex/OpenCode 是 `enabled`；Cline/Pi 是 `disabled`），翻译时反转。

### 伪代码

```
for harness in enabled ∩ detected:
    if harness has no mcp capability:  log "[h] mcp: n/a"; continue
    cfg = readMCPConfig(harness)
    for name, server in canonicalMCP:
        entry = translate(server, harness.quirks)     # 上面的 1/2/3
        cfg[harness.key][name] = entry                # mcpServers | servers | mcp | [mcp_servers]
    if harness == codex|grok: writeTOML(cfg) else: writeJSON(cfg)
```

### 合并规则

- 只 upsert Magic 管理的条目；用户自己加的 server **保留**。
- 需要标记「哪些是 Magic 管的」：用名字前缀 `magic-`（与 provider 一致）或单独记录。
  **待定**（见开放问题）。
- 不删除用户条目。

## 不支持 / 特殊情况

- **Pi**：remote 支持待确认；当前实测只有 stdio。
- **Codex**：不支持 SSE remote（要代理）。
- **VS Code**：推荐用 `inputs` 而非明文，翻译时可留空值让用户填。
- **Grok**：会**自动加载** Claude / Cursor / 项目 `.mcp.json`。若同时给 Grok 和 Claude 都写，
  会出现重复来源；需要决定是否只写 Grok 的 `config.toml`（推荐）。

## 与 `mcpuser` 适配器的关系

`mcpuser`（`~/.config/mcp/mcp.json`）是通用 MCP 落点，不是 harness。它可作为**一个通用落点**
（一份通用 MCP），但不要和「模型 provider」混（`mcpuser` 不是 provider，没有 SwitchWriter）。

## 密钥处理（已定）

MCP 的密钥**就在 MCP 配置里**（`env` / `headers`）——这是 MCP 自己的事，与 Magic 产品无关。

- Magic 可以把它们**存进库**（方便跨机），也可以不管；都不是产品要求。
- **写回 harness 时直接写明文**：不用 `${ENV}` / `$VAR` 这类间接（各家展开语法不一、Claude 不展开）。
  用户要的是开箱即用，就得是明文真值。与 Pi/WorkBuddy 的模型 key 同一策略。
- 含密钥的 MCP 文件本地明文；`sync push` 时按密钥文件加密成云端 `.enc`（见 `secrets-and-sync.md`）。

## 未定

见 [open-questions.md](./open-questions.md) A7；Codex/Grok 的 TOML 写入复用见 C10。

密钥写明文、Pi remote MCP（第三方适配器）、project 级暂不做 —— 见 [decisions.md](./decisions.md) D8。

## 调研来源（2026-09-11）

| harness | 来源 |
|---|---|
| Claude Code | https://code.claude.com/docs/en/mcp |
| Codex | https://learn.chatgpt.com/docs/config-file/config-reference |
| Gemini CLI | https://google-gemini.github.io/gemini-cli/docs/tools/mcp-server.html |
| Antigravity CLI | https://antigravity.google/docs/cli/gcli-migration/ · /docs/mcp/ |
| Trae | https://docs.trae.ai/ide/mcp-server-install-links |
| CodeBuddy | https://www.codebuddy.ai/docs/zh/cli/settings · /cli/codebuddy-dir |
| Kilo | https://kilo.ai/docs/code-with-ai/agents/custom-models |
| Cursor | https://cursor.com/docs/mcp |
| OpenCode | https://opencode.ai/docs/config/ |
| Copilot CLI | https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers |
| VS Code | https://code.visualstudio.com/docs/agent-customization/mcp-servers |
| Cline | https://docs.cline.bot/mcp/mcp-overview |
| Grok | https://docs.x.ai/build/features/mcp-servers |
| Pi | 本机 `~/.pi/agent/mcp.json` 实测 · 适配器 https://github.com/nicobailon/pi-mcp-adapter |
| Qoder CLI | https://docs.qoder.com/cli/mcp-servers |
| Kimi Code CLI | https://www.kimi.com/code/docs/en/kimi-code-cli/customization/mcp.html |
| CodeBuddy CLI | https://www.codebuddy.ai/docs/cli/mcp |
