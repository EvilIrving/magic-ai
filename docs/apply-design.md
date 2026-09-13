# Apply 设计：共有的原数据 + 各 harness 定制

状态：**主要行为已实现**。providers / skills / sub-agents 的投影、`packs/harness/<id>/` 定制写回、字段级合并、`magic add path` 都已落代码。**未实现**：MCP 投影（见 `mcp-design.md`）、provider 三态/冲突检测。

## 目标（一句话）

用户在 **一台机器上维护 `~/.magic-ai` 一份**，`magic apply` 把它按各 harness 自己的格式
写进它们已经在用的文件；换机器只需 `sync pull` + `apply`，不必在每个框架里重复配置。
增删一个模型、换一个中转、加一个 skill/MCP，只改 Magic 一处。

## 关键分层：共有 vs 定制 vs 排除

| 层 | 内容 | 处理方式 |
|---|---|---|
| **共有原数据** | model providers（`base_url` / `protocol` / `pick` / key）、skills、MCP、sub-agents | 库一份；apply 时按各 harness 格式**投影**（格式不同，数据同一份） |
| **harness 定制** | Pi 的 `extensions/` `themes/`、Codex 的 `memories/`、Claude 的 statusline 等 | 按文件放 `packs/harness/<id>/`，apply 时**只写回该 harness**，不译成别的工具 |
| **排除** | 会话 / 运行态 / 缓存 / 登录态（`auth.json`） | 永不同步、永不投影 |

「共有原数据」是不变的：providers 就是 providers。变的只是**落到每个 harness 的格式与文件位置**。

## 共有原数据 → 各 harness 的映射

| harness | 目标文件 | 从库来的数据 | 登录/切换 |
|---|---|---|---|
| Claude Code | `~/.claude/settings.json` 的 `env` | `ANTHROPIC_BASE_URL` / token / 角色模型 | provider switch 写 token |
| Pi | `~/.pi/agent/models.json` 的 `providers.magic-*` | provider + pick（+ registry meta） | `auth.json` 不动；key 直接写进 models.json |
| Codex | `~/.codex/config.toml` + `~/.codex/models.json` | model provider 段 + 第三方 catalog | `auth.json` 不保留；switch 管中转/官方 |
| Grok | `~/.grok/config.toml` | managed catalog 块 | provider-mode 标记 |
| OpenCode | `~/.config/opencode/opencode.json` | `provider.magic-*` + `model` | — |
| WorkBuddy | `~/.codebuddy/models.json` | `models[]`（`magic-*`） | — |

投影细节见 `docs/harness-matrix.md`；模型元数据来自 `registry/models.json`
（`docs/model-registry-from-pi.md`）。

## 各 harness 的定制逻辑（按真实目录）

### Pi — `~/.pi/agent`

真实目录里用户可自定义的部分：

| 文件/目录 | 类别 | 处理 |
|---|---|---|
| `models.json` | 共有投影 + 用户自有 provider | Magic 只 upsert `providers.magic-<name>`，其它 provider 保留 |
| `settings.json` | 定制 | 按文件同步 |
| `AGENTS.md` | 定制 | 按文件同步 |
| `themes/**` | 定制（Pi 专有） | 按文件同步 |
| `extensions/**`（如 `short-path-footer.ts`） | 定制（Pi 专有脚本） | 按文件同步 |
| `mcp.json` | 共有（MCP 层） | 由 `packs/mcp` 投影（见 MCP 设计） |
| `keybindings.json` / `trust.json` / `SYSTEM.md` / `APPEND_SYSTEM.md` | 定制（可选） | 若在 `packs/harness/pi/` 下则写回 |
| `auth.json` | 登录 | **不碰** |
| `sessions/**` `subagent-sessions/**` | 运行态 | **不同步** |
| `npm/**` `bin/**` `pi-lab/**` | 运行态 | **不同步** |
| `mcp-cache.json` `models-store.json` | 缓存 | **不同步** |

Pi 的定制逻辑要点：`extensions/` 是 Pi 独有的扩展脚本，不属于"共有原数据"，必须独立处理
（按文件收放，不投影到别的 harness）。

### Codex — `~/.codex`

| 文件/目录 | 类别 | 处理 |
|---|---|---|
| `config.toml` | 定制（用户主配置） | 同步；provider 段由 switch/apply 写 |
| `models.json` | 共有投影（第三方 catalog） | 第三方时写；官方时删 |
| `AGENTS.md` | 定制 | 同步 |
| `memories/**` | 定制（可存） | 同步 |
| `rules/**` `skills/**` | 定制 / 共有 | 按文件同步（skills 走 skills 层） |
| `auth.json` | 登录 | **不保留**；官方↔中转由 provider switch 管 |
| 其余（大量） | 运行态 | **不同步**：`*.sqlite*` `sessions/**` `archived_sessions/**` `log/**` `logs_*.sqlite*` `cache/**` `attachments/**` `generated_images/**` `history.jsonl` `session_index.jsonl` `*.jsonl` `installation_id` `version.json` `chrome-native-hosts-v2.json` `plugins/**` `process_manager/**` 等 |

Codex 平台可定制的面很小：`config.toml` + `AGENTS.md` + `memories/`，加 skills/agents。
登录文件 `auth.json` Magic 不保留（因为 provider switch 会重写它）。

### Claude — `~/.claude` + `~/.claude.json`

| 文件 | 类别 | 处理 |
|---|---|---|
| `settings.json` | 共有投影（`env`）+ 定制（其它键） | Magic 只 upsert `ANTHROPIC_*` / `CLAUDE_CODE_*`，其余键保留 |
| `settings.local.json` | 定制 | 按文件同步 |
| `CLAUDE.md` | 定制 | 同步 |
| `skills/**` `agents/**` | 共有 | 由 skills/agents 层投影 |
| `statusline.sh` 及自定义脚本 | 定制 | 按文件同步（`packs/harness/claude/`） |
| `~/.claude.json` | 配置键（`mcpServers` 等） | 字段级 partial 合并（`packs/harness/claude/claude.json.partial.json`） |
| `sessions/**` `projects/**` `cache/**` `daemon*` `history.jsonl` `plugins/**` `provider-switch/**` `teams/**` `backups/**` | 运行态 | **不同步** |

### Grok / OpenCode / WorkBuddy

- Grok `~/.grok`：定制 = `config.toml`（`AGENTS.md`、`hooks/**`、`commands/**`、`disabled-hooks`）；运行态排除。
- OpenCode `~/.config/opencode`：定制 = `opencode.json(c)`、`agents/` `commands/` `themes/` `plugins/` `modes/` `tools/`；凭证 `~/.local/share/opencode/auth.json` 不碰。
- WorkBuddy `~/.codebuddy`（Detect 亦认 `~/.workbuddy`）：`models.json`。
- Gemini/Copilot/Trae 是 apply-only 适配器（无第三方 dialect writer）。

## harness 定制的存放与匹配

```
~/.magic-ai/packs/harness/<id>/<rel>   →   <harness 根>/<rel>
```

例：

```
packs/harness/pi/extensions/short-path-footer.ts → ~/.pi/agent/extensions/short-path-footer.ts
packs/harness/pi/themes/dark.json                → ~/.pi/agent/themes/dark.json
packs/harness/codex/memories/index.json          → ~/.codex/memories/index.json
packs/harness/claude/statusline.sh               → ~/.claude/statusline.sh
```

- 只写回**对应** harness，不投影到没有这种文件的工具（Pi theme 不会变成 Grok theme）。
- **写回是字段级合并，不整文件覆盖**（已定）：

  | 目标 | 规则 |
  |---|---|
  | 结构化文件（`settings.json` / `opencode.json` / `config.toml` / `*.partial.json`） | **字段级合并**：只写库里有的键/记录，其余键原样保留 |
  | 一文件一项（`extensions/*.ts`、`themes/*.json`、`commands/*.md`、`hooks/*`、`memories/*`） | **按项合并**：同名项写，其它文件不删 |
  | 整份语义的文件（`AGENTS.md` 这类） | 覆盖（库里没有则不碰） |

  绝不删 harness 里用户自己的其它文件/键。

## 把任意文件收进库：`magic add path`

用户机器上可能有一份 harness 独有配置（Pi 的 theme、Codex 的 `memories/`、一个自定义 statusline）。
`magic add path` 把它带进库，随 `sync` 走到其它机器，再由 `apply` 写回：

```
magic add path <外部文件> [--as <harness>/<库内相对路径>]

# 例：
magic add path ~/.pi/agent/themes/foo.json
#   → packs/harness/pi/themes/foo.json
magic add path ~/.claude/statusline.sh
#   → packs/harness/claude/statusline.sh
```

- 目标是 `packs/harness/<id>/<rel>`；按 harness 根自动换算（`~/.pi/agent` → `packs/harness/pi`）。
- **复制**（不是移动），原文件不动；之后 `apply` 按上面的合并规则写回。
- 这是给人的显式动作（capture 已删，不会自动回流）。

## apply 伪代码

```
apply(magicHome, userHome, only):
  providers   = load providers.yaml            # 共有
  secrets     = load secrets/providers.json    # 本地明文
  skills      = scan packs/skills
  mcp         = load packs/mcp/mcp.yaml
  agents      = scan packs/agents
  custom      = scan packs/harness/<id>

  for each harness in enabled ∩ detected:
    backupDir = new backup under magicHome/backups

    # 1) 共有原数据 → 该 harness 的格式（全部字段级合并，不整文件覆盖）
    switch harness:
      claude:   upsert ~/.claude/settings.json env(ANTHROPIC_*, 角色)   # 真 token
      pi:       merge ~/.pi/agent/models.json providers.magic-<active>  # 真 key
      codex:    merge config.toml provider 段 + models.json (第三方)
      grok:     merge config.toml managed 块 + provider-mode
      opencode: merge opencode.json provider.magic-<active> + model
      workbuddy:merge ~/.codebuddy/models.json magic-* 行

    # 2) 共有 skills / MCP / sub-agents
    sync skills → harness skills dir
    project mcp → merge harness mcp 文件（字段级）
    project agents → harness agents dir

    # 3) harness 定制：字段级合并写回（同名键/项覆盖，其余保留）
    mergeTree(packs/harness/<id>, harnessRoot)

    # 4) 永不触碰：sessions / runtime / cache / auth.json
  return log
```

## 与已删功能的关系

- **capture 已删除**（产品方向相反）。因此 `packs/harness/**` 只能手放或用 `magic add path` 收进来。
- 路径已定：harness 定制层 = **`packs/harness/<id>/`**。覆盖规则已定：**字段级合并**。

## 已定 / 仍开放

**已定**（见 [decisions.md](./decisions.md) D5/D6/D7）：写回**字段级合并**；harness 定制层 = `packs/harness/<id>/`；收文件进库用 `magic add path`。

**仍开放：**

1. Pi `models.json` 里 Magic provider（`magic-*`）与用户自有 provider 的命名/合并边界。
2. MCP 多协议映射见 [mcp-design.md](./mcp-design.md)。
