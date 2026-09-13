# Harness 深度档案（覆盖集）

状态：**调研稿**。目标：把要覆盖的十来家**每家都扒到底**——配置根、skills、agents、MCP、models、hooks、custom、以及 Magic 该往哪写。
选型依据见文末「为什么是这几家」。

符号：**✅ 已核实**（官方文档/本机实测） · **❓ 未确认** · **— 不支持**

---

## 一、国外

### 1. Claude Code ｜ 39%（全球第一）

| 项 | 内容 |
|---|---|
| 配置根 | `~/.claude/` + 用户级总配置 `~/.claude.json` |
| skills | `~/.claude/skills/<name>/SKILL.md`；项目 `.claude/skills/` ✅ |
| agents | `~/.claude/agents/*.md`（frontmatter: `name` / `description` / `tools` / `model`）；项目 `.claude/agents/` ✅ |
| MCP | `~/.claude.json` 的 `mcpServers`（用户）；项目 `.mcp.json`；remote `{type:"http"\|"sse", url, headers}`（**有 `url` 就必须带 `type`**）✅ |
| models / provider | `~/.claude/settings.json` 的 `env`：`ANTHROPIC_BASE_URL` / `ANTHROPIC_AUTH_TOKEN`(主) / `ANTHROPIC_API_KEY` / `ANTHROPIC_MODEL` / `ANTHROPIC_DEFAULT_*` / `CLAUDE_CODE_SUBAGENT_MODEL` ✅ |
| hooks | `~/.claude/settings.json` 的 `hooks`，**20+ 事件**（SessionStart/End、UserPromptSubmit、PreToolUse、PostToolUse…）✅ |
| custom | `CLAUDE.md`、`settings.local.json`、`statusline.sh`、`plugins/` |
| 排除 | `sessions/` `projects/` `cache/` `daemon*` `history.jsonl` `provider-switch/` `teams/` |
| **Magic 写** | settings.json `env`（真 token）+ skills + agents + MCP + hooks |

### 2. Codex CLI ｜ 16%（一年涨 5 倍）

| 项 | 内容 |
|---|---|
| 配置根 | `~/.codex/` |
| skills | `~/.codex/skills/`；项目 `.codex/skills/` ✅ |
| agents | `~/.codex/agents/*.md` ❓（Codex 的 subagent 在 `config.toml` 里） |
| MCP | `~/.codex/config.toml` 的 `[mcp_servers.<name>]`（TOML）；remote 直接用 `url`（streamable http），**SSE 不支持**（需代理）✅ |
| models / provider | `config.toml` 的 provider 段 + `~/.codex/models.json` + `model_catalog_json` + **`env_key`**（Codex 自己的机制，指向环境变量名）✅ |
| hooks | `~/.codex/hooks.json`，**12 事件**（SessionStart/End、PreToolUse、PermissionRequest、PostToolUse、Pre/PostCompact、SubagentStart/Stop、Stop、Interrupt…）✅ |
| custom | `AGENTS.md`、`memories/**`、`rules/**` |
| 排除 | `auth.json`（**不保留**，switch 会重写）、`*.sqlite*`、`sessions/`、`log/`、`cache/`、`history.jsonl` |
| **Magic 写** | config.toml provider 段 + models.json（第三方时）+ hooks + skills + MCP |

### 3. GitHub Copilot CLI ｜ 21%

| 项 | 内容 |
|---|---|
| 配置根 | `~/.copilot/`（用户）；项目 `.github/` |
| skills | `~/.copilot/skills/`；项目 `.github/skills/` ✅ |
| agents | **`~/.copilot/agents/*.md`（全局）/ `.github/agents/*.md`（项目）** ✅ 官方专门有「Create custom agents」文档 |
| MCP | `~/.copilot/mcp-config.json`，`mcpServers`；本地 `{type:"local", command,args,env, tools,timeout}`，remote `{type:"http"\|"sse", url, headers}` ✅ |
| models / provider | **支持 BYOK 自定义模型**（2026-04 起）✅；官方登录为主 |
| hooks | `~/.copilot/hooks/*.json`、`.github/hooks/*.json`，**7 事件**（sessionStart/End、userPromptSubmitted、preToolUse、postToolUse、agentStop、errorOccurred）✅ |
| custom | `copilot-instructions.md`、`instructions/**`、`extensions/**`、`lsp-config.json` |
| 排除 | `config.json`（含 auth）、logs、session-state、sqlite、`mcp-oauth` |
| **Magic 写** | agents + skills + MCP + hooks |

### 4. Cursor ｜ 12%（中国掉得最狠：28%→16%）

| 项 | 内容 |
|---|---|
| 配置根 | `~/.cursor/`（用户）；项目 `.cursor/` |
| skills | `~/.cursor/skills/` ❓；也认 `.agents/skills/` |
| agents | 通过 **Agent Plugins** 打包（rules/skills/agents/commands/MCP/hooks）；目录 ❓ |
| MCP | `~/.cursor/mcp.json`、项目 `.cursor/mcp.json`；`mcpServers`；remote **直接 `url`，不需要 `type`**；也支持 `{url, auth:{CLIENT_ID, CLIENT_SECRET, scopes}}` ✅ |
| models / provider | — 无第三方 provider 配置文件（纯官方登录）❓ |
| hooks | **`~/.cursor/hooks.json` / 项目 `.cursor/hooks.json`**，**20+ 事件**（sessionStart/End、preToolUse、postToolUse、postToolUseFailure、subagentStart/Stop、beforeShellExecution、beforeMCPExecution、beforeReadFile、afterFileEdit、beforeSubmitPrompt、preCompact、stop、afterAgentResponse、beforeTabFileRead…）；支持 `type:"prompt"`（LLM 判定）；exit 0/2 ✅ **并且能直接加载 Claude Code 的 hooks** |
| custom | `rules`、`commands`、`AGENTS.md` |
| **Magic 写** | hooks.json + MCP + skills（+ plugins） |

### 5. OpenCode ｜ 7%（开源，42% 心智占有率）

| 项 | 内容 |
|---|---|
| 配置根 | `~/.config/opencode/`；项目 `<repo>/opencode.json(c)`、`.opencode/` |
| skills | `~/.config/opencode/skills/`；项目 `.opencode/skills` ✅ |
| agents | `~/.config/opencode/agents/`（markdown + frontmatter）✅ |
| MCP | `opencode.json` 的 **`mcp`** 键（不是 `mcpServers`）；local `{type:"local", command:[数组], enabled}`，remote `{type:"remote", url, headers, enabled}` ✅ |
| models / provider | `opencode.json` 的 `provider.magic-<name>.options.{baseURL,apiKey}` + `model: magic-<name>/<pick>` ✅ 写**明文 apiKey** |
| hooks | 代码式 **plugins**（`~/.config/opencode/plugins/*.{js,ts}`），事件 `tool.execute.before/after`、`session.*`、`file.edited`、`permission.*` ✅ |
| custom | `commands/` `themes/` `modes/` `tools/` `tui.json` |
| 排除 | `~/.local/share/opencode/auth.json` |
| **Magic 写** | provider + model + agents + skills + MCP + plugins |

### 6. Google Antigravity（CLI `agy` + IDE）｜ 6%（印度 15%）

> **取代 Gemini CLI**（2026-06-18 官方）。`gemini` 适配器保留作过渡。

| 项 | 内容 |
|---|---|
| 配置根 | `~/.gemini/antigravity-cli/`（CLI）；`~/.gemini/antigravity/`（App） |
| 设置 | `~/.gemini/antigravity-cli/settings.json`；快捷键 `keybindings.json` ✅ |
| skills | **`~/.gemini/antigravity-cli/skills/`（全局）**、**`.agents/skills/`（工作区）** ✅（Gemini CLI 是 `~/.gemini/skills/` + `.gemini/skills/`） |
| agents | subagents（内置，`/agents`）；自定义 ❓ |
| MCP | **独立文件**：全局 `~/.gemini/config/mcp_config.json`、工作区 `.agents/mcp_config.json`；`mcpServers`；remote 键是 **`serverUrl`**（不是 `url`/`httpUrl`）✅ |
| models | models 页（`/model`）；配置文件里可覆盖 ❓ |
| hooks | **`hooks.json`**（`.agents/` 或 `~/.gemini/config/`）；**5 事件**：PreToolUse、PostToolUse、PreInvocation、PostInvocation、Stop；按 hook 名分组；I/O 是 camelCase JSON；`decision: allow/deny/ask/force_ask/deny_unless_prior_grant` ✅ |
| custom | `GEMINI.md`、`AGENTS.md`、plugins、statusline、title |
| **Magic 写** | skills + MCP（`serverUrl`！）+ hooks |

### 7. JetBrains AI / Junie ｜ 9%（**降级**：能写的文件极少）

| 项 | 内容 |
|---|---|
| 配置根 | IDE 设置（`~/Library/Application Support/JetBrains/<IDE><ver>/`）；项目 `.junie/`、`.aiassistant/` ❓ |
| skills | ❓ |
| agents | Junie（内置 agent）+ MCP 代理 |
| MCP | **IDE 设置里配**（Settings | Tools | AI Assistant | MCP）；也认 IDE MCP server JSON ✅ |
| models | IDE 设置里选 |
| hooks | ❓ |
| custom | **`.junie/AGENTS.md`（推荐位置）**；自定义路径在 Settings | Tools | Junie | 项目设置 ✅ |
| **Magic 写** | 只能写 `.junie/AGENTS.md` 这类文件；MCP/skills 走 IDE 设置 → **能力受限** |

---

## 二、国内

### 8. Trae / Trae CN ｜ 41.2%（国内第一）

| 项 | 内容 |
|---|---|
| 配置根 | `~/.trae/`（国际版）、`~/.trae-cn/`（国内版）；项目 `.trae/` |
| skills | **全局 `~/.trae/skills`（国内版 `~/.trae-cn/skills`）、项目 `.trae/skills/`** ✅（官方中文文档）；**也支持标准 `.agents/skills/`**（设置里开关；重名时 `.trae/skills/` 优先） |
| agents | 自定义智能体（UI 里配）；文件落点 ❓ |
| MCP | **`~/.trae/mcp.json`（国内版 `~/.trae-cn/mcp.json`）**、项目根 `mcp.json`；`mcpServers` ✅；也可在 IDE UI 里配 |
| models | UI 里加自定义模型（也支持「自定义模型服务商」）✅ |
| hooks | ❓ |
| custom | `.trae/skill-config.json`（禁用状态）、`rules/`、`sandbox.json`、`settings.json`、`user_rules.md` |
| 排除 | `~/Library/Application Support/Trae/User/` |
| **Magic 写** | skills + MCP（+ 可能的 rules） |

### 9. Qoder / Qoder CN（含通义灵码 Lingma）｜ 阿里

> **通义灵码 = Qoder CN**（阿里同一产品线，正在并入）。

| 项 | 内容 |
|---|---|
| 配置根 | `~/.qoder/`（国际）、`~/.qoder-cn/`、`~/.lingma/`（灵码）；项目 `.qoder/` |
| skills | `.qoder/skills/`（项目）；用户级 skills 目录；灵码 `~/.lingma/skills/` + 支持 `.agents/skills/` ✅ |
| agents | **Subagent**（官方扩展一览里有）✅ |
| MCP | `~/.qoder/settings.json` 的 `mcpServers`（stdio/sse/http/ws）✅ |
| models | **Custom Models**（官方有专门文档）✅；iFlow 式 `apiKey`/`baseUrl`/`modelName` 落在 settings.json ❓ |
| hooks | **`~/.qoder/settings.json` 的 `hooks`**；项目 `.qoder/settings.json`、`.qoder/settings.local.json`（**三份合并**）；**23 事件**：SessionStart/End、UserPromptSubmit、PreToolUse、PostToolUse、PostToolUseFailure、PermissionRequest、PermissionDenied、Stop、StopFailure、SubagentStart/Stop、Pre/PostCompact、Notification、InstructionsLoaded、ConfigChange、CwdChanged、FileChanged、WorktreeCreate/Remove、Elicitation、ElicitationResult；类型 `command`/`http`/`prompt`/`agent` ✅ **Claude 同款形状** |
| custom | `plugins/`、`commands/`、`memory/`、`knowledge-base/`、`output-styles` |
| **Magic 写** | skills + MCP + hooks（形状同 Claude，可复用 Claude 的 hook writer）+ models |

### 10. 腾讯 CodeBuddy / WorkBuddy ｜ 国内主流

| 项 | 内容 |
|---|---|
| 配置根 | **`~/.codebuddy/`**（用户级全局，**WorkBuddy 共用这同一目录**）；项目 `.codebuddy/` ✅ |
| skills | `~/.codebuddy/skills/`；插件可打包 Skills ✅ |
| agents | 插件含 Agents；`.codebuddy/agents/` ❓ |
| MCP | **`~/.codebuddy/settings.json` 的 `mcpServers`**（stdio/sse/http）；作用域 local > project > user；工具权限 `mcp__server__tool` ✅ |
| models | **`~/.codebuddy/models.json`** 的 `models[]`；SwitchWriter 写 `id: magic-*` + 明文 `apiKey` ✅ |
| hooks | **`~/.codebuddy/settings.json` 的 hooks**（Claude 兼容 command hook + HTTP 权限 hook）✅ |
| custom | 插件含 Skills / Agents / Hooks / MCP / **LSP**；`settings.json` 是官方配置机制 |
| **Magic 写** | models.json + settings.json(hooks/mcp) + skills + agents |

### 11. Kimi Code（月之暗面）

| 项 | 内容 |
|---|---|
| 配置根 | `~/.kimi-code/`（随 `KIMI_CODE_HOME` 移动）；项目 `.kimi-code/` |
| skills | `~/.kimi-code/skills/`；项目 `.kimi-code/skills/` ❓ |
| agents | **`~/.kimi-code/agents/`、`~/.agents/agents/`（用户级）；项目 `.kimi-code/agents/`、`.agents/agents/`**；扫描优先级 显式 > 项目 > 额外 > 用户 > Plugin > 内置 ✅ |
| MCP | `~/.kimi-code/mcp.json`、项目 `.kimi-code/mcp.json`；`mcpServers`；stdio `{command,args,env,cwd}`，remote `{url}` 或 `{transport:"sse",url}`；`enabled` / `enabledTools` ✅ |
| models | `config.toml` ✅ |
| hooks | **`~/.kimi-code/config.toml` 的 `[[hooks]]`**（TOML）；**20 事件**；字段只有 `event`/`matcher`/`command`/`timeout`；**只有 PreToolUse / Stop / UserPromptSubmit 能阻断**；fail-open ✅ |
| custom | `SYSTEM.md`（覆盖 main agent 系统提示词）、`AGENTS.md`、plugins |
| **Magic 写** | agents + MCP + hooks(TOML) + skills |

> **agent 文件跟 Claude Code / OpenCode 互通** —— 只含 `description` + 正文的最小文件可跨工具直接加载。

### 12. Qwen Code（阿里开源）

| 项 | 内容 |
|---|---|
| 配置根 | `~/.qwen/`；项目 `.qwen/`（桌面版与 CLI **共用同一份 settings.json**）✅ |
| skills | ❓（Gemini CLI 血统，大概率 `.qwen/skills/`） |
| agents | **Subagents**（官方有专门文档）✅；目录 ❓ |
| MCP | `~/.qwen/settings.json` 的 `mcpServers` ✅ |
| models | `settings.json` ✅ |
| hooks | **`settings.json` 的 `hooks`**；**17 事件**：PreToolUse、PostToolUse、PostToolUseFailure、UserPromptSubmit、SessionStart、SessionEnd、SessionDelete、MessageDisplay、Stop、SubagentStart、SubagentStop、PreCompact、Notification、PermissionRequest、PermissionDenied、TodoCreated、TodoCompleted；类型 `command` / `http` / `function` / `prompt`；`disableAllHooks` 总开关；HTTP 钩子带 SSRF/私网防护 ✅ |
| custom | `memory`、`GEMINI.md` 式上下文 |
| **Magic 写** | skills + MCP + hooks + models |

---

## 三、Magic 自有 / 既有（不在调查里，但必须有）

### 13. Pi ｜ Magic 自己的主力

| 项 | 内容 |
|---|---|
| 配置根 | `~/.pi/agent/` |
| skills | `~/.pi/agent/skills/` ✅ |
| agents | `~/.pi/agent/agents/` ✅ |
| MCP | `~/.pi/agent/mcp.json`（`mcpServers`，stdio）；remote 靠第三方 `pi-mcp-adapter`；还读 `~/.config/mcp/mcp.json`、`.mcp.json`、`~/.agents/mcp.json` ✅ |
| models | **`~/.pi/agent/models.json`** 的 `providers.magic-<name>`，明文 `apiKey`；配 registry meta（context/thinkingLevelMap）✅ |
| hooks | 代码式 **extensions**（`~/.pi/agent/extensions/*.ts`）；事件 `session_start`、`tool_call`、`before_agent_start`、`before_provider_request`、`after_provider_response`、`session_before_compact/compact`、`session_end`… ✅ |
| custom | `themes/**`、`AGENTS.md`、`keybindings.json`、`settings.json` |
| 排除 | `auth.json`、`sessions/**`、`npm/**`、`mcp-cache.json` |
| **Magic 写** | models.json（真 key）+ skills + agents + MCP + extensions/themes |

### 14. Grok CLI ｜ 既有

| 项 | 内容 |
|---|---|
| 配置根 | `~/.grok/` |
| skills | `~/.grok/skills/` ✅ |
| agents | `~/.grok/agents/` ✅ |
| MCP | `~/.grok/config.toml` 的 `[mcp_servers.<name>]`；`${VAR}` 展开；**也兼容读** `~/.claude.json`、`.cursor/mcp.json`、`.mcp.json` ✅ |
| models | `config.toml` managed catalog 块 + provider-mode 标记 ✅ |
| hooks | `~/.grok/hooks/*.json`，**Claude 同款形状**（本机实测）✅ |
| custom | `commands/**`、`AGENTS.md`、`disabled-hooks` |
| **Magic 写** | MCP + skills + agents + hooks + provider 块 |

---

## 附：通用 MCP 落点（不是 harness）

- **mcpuser**：`~/.config/mcp/mcp.json`。一份通用 MCP，给没自己 MCP 文件的工具用。**没有 SwitchWriter**。

---

## 长尾：暂不做（未来再说）

有数据但不进这一轮：Cline、Kilo、Goose、Crush、Amp、Droid、Kiro、Continue、Zed、OpenHands、Zencoder、Mistral Vibe、Cortex、Pochi、MiniMax Code、iFlow、Kimi Code、Qwen Code、Warp。

**已定不做**：windsurf、roo（停运）、Tabnine / Rovo Dev / Junie / Posit / Neovate（没人用）。

**声量淘汰**（2026-09 硬数据）：CodeGeeX（repo 停更 2 年）· iFlow（npm 周下载 161、停更）· Comate（已并入百度搭子）。

> 长尾不是废弃——配置路径已经查好了（见 `supported-harnesses.md`「第二轮调研」表），
> 以后有空照这张表加就行，不需要重新调研。

---

## 为什么是这几家 —— 声量复核

官方调研有自家盘子加成（JetBrains 会给自家 Junie / AI Assistant 记一笔，且「最受喜爱」是自报），
所以又拿**硬数据**（GitHub stars、npm 周下载、国内下载量/MAU）交叉验了一遍。

### GitHub stars（2026-09-11 实测）

| 项目 | stars | 最近提交 |
|---|---|---|
| **anomalyco/opencode** | **206,540** | 2026-09-11 |
| anthropics/claude-code | 144,696 | 2026-09-10 |
| openai/codex | 123,166 | 2026-09-11 |
| google-gemini/gemini-cli | 106,910 | 2026-09-11 |
| zed-industries/zed | 90,074 | 2026-09-11 |
| OpenHands/OpenHands | 87,294 | 2026-09-11 |
| cline/cline | 67,813 | 2026-09-11 |
| aaif-goose/goose | 54,109 | 2026-09-11 |
| continuedev/continue | 35,869 | 2026-09-08 |
| charmbracelet/crush | 28,004 | 2026-09-11 |
| QwenLM/qwen-code | 27,762 | 2026-09-11 |
| Kilo-Org/kilocode | 27,251 | 2026-09-10 |
| RooCodeInc/Roo-Code | 24,303 | **2026-05-15（停）** |
| MoonshotAI/kimi-cli | 11,339 | 2026-09-01 |
| zai-org/CodeGeeX | 8,804 | **2024-08-13（死）** |
| iflow-ai/iflow-cli | 5,096 | **2026-03-20（停）** |

### npm 周下载（2026-09 实测）

| 包 | 周下载 |
|---|---|
| @openai/codex | **13,156,260** |
| @anthropic-ai/claude-code | 8,554,147 |
| opencode-ai | 1,310,980 |
| @google/gemini-cli | 237,533 |
| @qwen-code/qwen-code | 36,604 |
| @charmland/crush | 3,957 |
| @iflow-ai/iflow-cli | **161** |

### 国内（闭源，看下载量/MAU）

- **通义灵码**：插件下载量 **>1500 万**，生成代码 >30 亿行（官网/报道）
- **Trae**：**MAU >100 万**（2025-06，字节技术副总裁）
- **文心快码 Comate**：**已并入「百度搭子」**，成为其核心代码能力模块（新闻，2026-09）→ 作为独立 harness 已不存在
- **CodeGeeX**：repo 2024-08 后无提交 → 事实上停更

### 结论：官方调研错在哪

| 官方说 | 硬数据 | 判断 |
|---|---|---|
| opencode 只有 7% | **206k stars，全类第一；npm 131 万/周** | **被严重低估**（JetBrains 样本偏 IDE 用户，漏掉 CLI/开源党） |
| Claude Code 39% > Codex 16% | 但 Codex npm **1320 万 > Claude Code 855 万** | 两家**同级**，不是 2.4 倍差距 |
| JetBrains AI/Junie 9% | 自家产品、自家用户投票 | **有自家盘子加成**；且它主要活在 IDE 设置里，能写的文件极少 |
| 国内「文心快码 12.3%」 | **已并入百度搭子** | 名单要换 |
| 国内「CodeGeeX」 | **repo 停更 2 年** | 名单要换 |

### 调整后的名单（13 家）

- **国外 6**：Claude Code · Codex · GitHub Copilot CLI · Cursor · **OpenCode**（升位）· Antigravity
- **国内 5**：Trae · Qoder（= 通义灵码）· CodeBuddy / WorkBuddy · **Kimi Code** · **Qwen Code**
  - 出：~~Comate~~（并入搭子）、~~CodeGeeX~~（停更）、~~iFlow~~（周下载 161，停更）
- **自有 2**：Pi · Grok

**边界观察**（声量够，本轮不做，随时可加）：Cline 68k · Goose 54k · OpenHands 87k · Continue 36k · Crush 28k · Kilo 27k · Zed 90k。
**明确降级**：JetBrains AI / Junie —— 能写的只有 `.junie/AGENTS.md` 这类文件，MCP/skills 都在 IDE 设置里，Magic 插不进去。

**依据来源**：JetBrains DES 2026（n>15,000）；GitHub API / npm registry 2026-09-11 实测；国内份额与新闻见下方原表。
