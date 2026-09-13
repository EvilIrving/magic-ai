# Hooks 投影设计（归属 + 事件模型）

状态：**设计稿（调研完成，未实现）**。调研日期 **2026-09-11**，来源见文末。
目标：确定 hooks 在 Magic 里放哪、各 harness 怎么投、事件怎么对齐。

## 结论

1. **hooks 是独立能力**（见 [supported-harnesses.md](./supported-harnesses.md)），但不是每家都有；不支持就跳过。
2. **归属用 per-harness 文件**：`packs/harness/<id>/hooks/…`（与 custom 层同一目录、同一套写回逻辑）。
   **不做语义层翻译**（理由见 §4）。写入一律**字段级合并**。
3. hooks 分两类：
   - **声明式配置**（JSON/TOML）：Claude Code、Codex、Gemini CLI、Antigravity CLI、Cursor、Copilot CLI、Grok、Qoder CLI。
   - **代码式扩展**（TypeScript/JS）：Pi（extensions）、OpenCode（plugins）、Cline（SDK plugins）。
     这类**天然属于 custom 层**，Magic 只按文件收放，不翻译。

## 各 harness 的 hooks 落点与格式

| harness | 类 | 文件 | 事件 | 形状 |
|---|---|---|---|---|
| **Claude Code** | 声明 | `~/.claude/settings.json` 的 `hooks` | SessionStart/End、Setup、UserPromptSubmit、UserPromptExpansion、PreToolUse、PermissionRequest、PermissionDenied、PostToolUse、PostToolUseFailure、PostToolBatch、Notification、Stop、StopFailure、SubagentStart/Stop、PreCompact/PostCompact… | `{Event:[{matcher,hooks:[{type:"command",command,timeout}]}]}` |
| **Codex** | 声明 | `~/.codex/hooks.json` 或 `config.toml` 的 `[hooks]` | SessionStart/End、UserPromptSubmit、PreToolUse、PermissionRequest、PostToolUse、PreCompact/PostCompact、SubagentStart/Stop、Stop、Interrupt | 同 Claude 形状；支持 `async`、`statusMessage`、`timeout`(秒)、matcher 正则 |
| **Gemini CLI** | 声明 | `~/.gemini/settings.json` 的 `hooks` | SessionStart、SessionEnd、BeforeAgent、AfterAgent、BeforeModel、AfterModel、**BeforeToolSelection**、BeforeTool、AfterTool、**PreCompress**、Notification | `{Event:[{matcher,hooks:[{type:"command",command,timeout,name,description}]}]}`；工具事件 matcher 是**正则**、生命周期是**精确串**；exit 0/2/其它 |
| **Antigravity CLI** | 声明 | `hooks.json`（`.agents/` 或 `~/.gemini/config/`） | **PreToolUse、PostToolUse、PreInvocation、PostInvocation、Stop** | `{hook名:{enabled,PreToolUse:[{matcher,hooks:[{type:"command",command,timeout}]}]}}`；按 hook 名分组；I/O 是 camelCase JSON；`decision: allow/deny/ask/force_ask/deny_unless_prior_grant` |
| **Cursor** | 声明 | `~/.cursor/hooks.json` / `.cursor/hooks.json` | sessionStart/End、preToolUse/postToolUse/postToolUseFailure、subagentStart/Stop、beforeShellExecution/afterShellExecution、beforeMCPExecution/afterMCPExecution、beforeReadFile/afterFileEdit、beforeSubmitPrompt、preCompact、stop、afterAgentResponse/afterAgentThought、workspaceOpen、beforeTabFileRead/afterTabFileEdit | JSON over stdio；**也支持加载 Claude Code 第三方 hooks** |
| **Copilot CLI** | 声明 | `~/.copilot/hooks/*.json`、`.github/hooks/*.json` | sessionStart、sessionEnd、userPromptSubmitted、preToolUse、postToolUse、agentStop、errorOccurred | `{version:1,hooks:{event:[{type:"command",bash,powershell,cwd,timeoutSec,env}]}}`（跨平台双脚本） |
| **Grok** | 声明 | `~/.grok/hooks/*.json` | Claude 同款（Notification、PreToolUse、SessionEnd…） | **Claude 同款 JSON 形状**（本机实测） |
| **Pi** | 代码 | `~/.pi/agent/extensions/*.ts` | `session_start`、`tool_call`、`before_agent_start`、`before_provider_request`、`after_provider_response`、`session_before_compact`/`session_compact`、`session_before_switch`、`session_end`、`project_trust`、`resources_discover`… | TS 扩展，`pi.on(event, handler)`；非声明式 |
| **OpenCode** | 代码 | `~/.config/opencode/plugins/*.{js,ts}` | `tool.execute.before/after`、`session.*`、`file.edited`、`permission.*`、`message.*`、`command.executed`、`tui.*` | JS/TS 插件返回 hooks 对象；也可从 npm 加载 |
| **Cline** | 代码 | SDK plugins（`cline plugin install`） | `beforeRun`、`afterRun`、`beforeModel`、`afterModel`、`beforeTool`、`afterTool`、`onEvent` | `AgentPlugin` 对象（TS）；Cline 另有 executable-per-event 脚本目录 |
| **Qoder CLI** | 声明 | `~/.qoder/settings.json`（`hooks`）+ 项目 `.qoder/settings.json` / `.qoder/settings.local.json` | **23 个**：SessionStart、SessionEnd、UserPromptSubmit、PreToolUse、PostToolUse、PostToolUseFailure、PermissionRequest、PermissionDenied、Stop、StopFailure、SubagentStart、SubagentStop、PreCompact、PostCompact、Notification、InstructionsLoaded、ConfigChange、CwdChanged、FileChanged、WorktreeCreate、WorktreeRemove、Elicitation、ElicitationResult | `{Event:[{matcher,hooks:[…]}]}`（**Claude 同款形状**，命名也几乎一一对应）；hook 类型 `command`/`http`/`prompt`/`agent`；三份配置**合并**（不覆盖）；env `QODER_PROJECT_DIR`；`hookSpecificOutput` 必须带 `hookEventName` |

| **Qwen Code** | 声明 | `~/.qwen/settings.json`（`hooks`）+ 项目 settings | **17 个**：PreToolUse、PostToolUse、PostToolUseFailure、UserPromptSubmit、SessionStart、SessionEnd、SessionDelete、MessageDisplay、Stop、SubagentStart、SubagentStop、PreCompact、Notification、PermissionRequest、PermissionDenied、TodoCreated、TodoCompleted | `{Event:[{matcher,hooks:[…]}]}`；类型 `command` / `http` / `function` / `prompt`；`disableAllHooks` 总开关；HTTP 钩子带 SSRF / 私网防护 |
| **Kimi Code** | 声明 | `~/.kimi-code/config.toml` 的 `[[hooks]]` | **20 个**：UserPromptSubmit、UserPromptQueued、PreToolUse、Stop、TurnStarted、PostToolUse、PostToolUseFailure、PermissionRequest、PermissionResult、SessionStart、SessionEnd、SessionHeartbeat、SubagentStart、SubagentStop、TaskStarted、StopFailure、Interrupt、PreCompact、PostCompact、Notification | TOML 数组 `{event,matcher,command,timeout}`；exit 0/2；**只有 PreToolUse / Stop / UserPromptSubmit 可阻断**，其余是观察型；fail-open |
| **Droid（Factory）** | 声明 | 见 `/docs/harness/hooks` | 待抄 | Claude 同套（`from` 块可阻断） |
| **Kiro CLI** | 声明 | 见 `/docs/hooks/` | 待抄 | — |
| **Goose** | 声明 | 按 **Open Plugins hooks spec** | 待抄 | — |
| **Zed** | 声明 | lifecycle hooks（v1.8 引入 bootstrap hook） | 待抄 | — |
| **Continue** | 声明 | 见最近 CLI 的 hooks 特性 | 待抄 | — |
| **Kilo** | — | **无原生 hooks**（仍是 feature request） | — | — |

## 事件对齐表（canonical ↔ 各家）

Magic 内部用一套规范事件名，只在**声明式**几家之间映射；代码式的不映射（按文件放）。

> ⚠️ **Gemini CLI 已被 Antigravity CLI 取代**（2026-06-18；官方博客 + 迁移文档）。Magic 目标改指
> `agy`（Antigravity CLI），`gemini` 保留作过渡兼容。两者事件名不同（见下）。

| canonical | Claude Code | Codex | Gemini CLI | Antigravity CLI | Cursor | Copilot CLI | Grok | Qoder |
|---|---|---|---|---|---|---|---|---|
| `session_start` | SessionStart | SessionStart | SessionStart | — | sessionStart | sessionStart | SessionStart | SessionStart |
| `session_end` | SessionEnd | SessionEnd | SessionEnd | — | sessionEnd | sessionEnd | SessionEnd | SessionEnd |
| `user_prompt` | UserPromptSubmit | UserPromptSubmit | BeforeAgent | PreInvocation | beforeSubmitPrompt | userPromptSubmitted | UserPromptSubmit | UserPromptSubmit |
| `pre_tool` | PreToolUse | PreToolUse | BeforeTool | PreToolUse | preToolUse | preToolUse | PreToolUse | PreToolUse |
| `post_tool` | PostToolUse | PostToolUse | AfterTool | PostToolUse | postToolUse | postToolUse | PostToolUse | PostToolUse |
| `pre_compact` | PreCompact | PreCompact | PreCompress | — | preCompact | — | PreCompact | PreCompact |
| `stop` | Stop | Stop | AfterAgent | Stop | stop | agentStop | Stop | Stop |
| `subagent_start` | SubagentStart | SubagentStart | — | — | subagentStart | — | SubagentStart | SubagentStart |
| `notification` | Notification | — | Notification | — | — | — | Notification | Notification |

`—` = 该家无对应事件。Gemini 的 `BeforeModel`/`AfterModel`/`BeforeToolSelection`、Antigravity 的
`PostInvocation`、Qoder 的 `PermissionRequest`/`Elicitation` 等**无 canonical 对应**，用时直接写各家名。

## 归属方案对比

| 方案 | 说明 | 取舍 |
|---|---|---|
| **A. 语义层** `packs/hooks/<name>` | 一份 hook 描述按 harness 翻译（像 MCP） | 跨家复用；但事件名/生命周期差异大，映射易错、难验证 |
| **B. per-harness 文件** `packs/harness/<id>/hooks/…` | 原样写回该 harness | 简单、不猜翻译；换家要重写。**与 custom 层同一目录** |

**选 B。** 理由：

1. 事件模型差异大（Claude 20+ 事件、Codex 12、Copilot 7），语义层必然丢信息。
2. 一半 harness 的 hooks 是**代码**（Pi/OpenCode/Cline），本就无法翻译，只能按文件放。
3. B 复用已有的 `packs/harness/<id>/` 写回逻辑，**零新增机制**。
4. 将来若证实声明式子集能无损对齐，再考虑给声明式几家加语义层（A），不影响 B。

## apply 伪代码（B 方案）

```
for harness in enabled ∩ detected:
    if harness has no hooks capability:   log "[h] hooks: n/a"; continue
    src = packs/harness/<id>/hooks/       # 声明式的 JSON，或代码式的 ts/js
    if src missing:                        continue
    mergeTree(src, harnessHooksDir)        # 字段级：同键/同名项写，其余保留
```

- **写入一律字段级合并，不整文件覆盖**（已定）。用户本机可能有一份独特 hooks 配置，
  只能加/改库里有的事件，不能把用户其它 hook/键抹掉。
- 声明式：`packs/harness/claude/hooks/*.json` → 合并进 `~/.claude/settings.json` 的 `hooks`
  （或直接放 `~/.codex/hooks.json` 这类独立文件）。
- 代码式：`packs/harness/pi/extensions/*.ts`、`packs/harness/opencode/plugins/*.ts` 原样写回。
- 不支持 → `n/a`，不报错、不造空文件。

## 未定

见 [open-questions.md](./open-questions.md) A5 / A6 / A8。（写入方式已定：**字段级合并**，见 [decisions.md](./decisions.md) D5。）

## 调研来源（2026-09-11）

| harness | 来源 |
|---|---|
| Claude Code | https://code.claude.com/docs/en/hooks |
| Codex | https://learn.chatgpt.com/docs/hooks |
| Gemini CLI | https://geminicli.com/docs/hooks/ · /docs/hooks/reference/ |
| Antigravity CLI | https://antigravity.google/docs/hooks/ · https://antigravity.google/docs/cli/gcli-migration/ |
| Qoder CLI | https://docs.qoder.com/zh/cli/hooks · /cli/hooks-reference |
| Qwen Code | https://github.com/QwenLM/qwen-code/blob/main/docs/users/features/hooks.md |
| Kimi Code | https://www.kimi.com/code/docs/kimi-code-cli/customization/hooks.html |
| Droid | https://docs.factory.ai/harness/hooks |
| Kiro CLI | https://kiro.dev/docs/hooks/ |
| Cursor | https://cursor.com/docs/hooks |
| Copilot CLI | https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/use-hooks |
| Cline | https://docs.cline.bot/sdk/plugins |
| OpenCode | https://opencode.ai/docs/plugins/ |
| Pi | 本机 `@earendil-works/pi-coding-agent/docs/extensions.md` + `~/.grok/hooks/*.json` 实测 |
| Grok | 本机 `~/.grok/hooks/cmux-session.json` 实测（Claude 同款形状） |

