# 全量 harness 支持执行方案（排行榜 29 家）

状态：**调研完成 → 方案待执行**。本文是 2026-09-13 第二轮全量调研的产物，覆盖用户给出的
排行榜名单（第 2–30 名，共 29 家；第 1 名未提供）。目的：**每一家都要能同步它自己的配置**
（skills / sub-agents / MCP / hooks / model-provider），而不是只做 skills 目录映射。

- 产品决定：见 [decisions.md](./decisions.md)（本文不重复列决定）。
- 能力矩阵：见 [harness-matrix.md](./harness-matrix.md)；名单来源：见 [supported-harnesses.md](./supported-harnesses.md)。
- MCP 专项：见 [mcp-design.md](./mcp-design.md)；hooks 专项：见 [hooks-design.md](./hooks-design.md)。

---

## 0. 名单映射（用户编号 → harness → 本仓库 id / 状态）

| # | 用户名单 | 本仓库 id | 类型 | 现状 | 本轮结论 |
|---|---|---|---|---|---|
| 1 | （名单未给出） | — | — | — | 待用户补充 |
| 2 | Codex | `codex` | 开源 Apache-2.0 | P0 完整 | 保持；校验 MCP/hooks |
| 3 | Claude Code | `claude` | 闭源（source-available） | P0 完整 | 保持 |
| 4 | Cursor | `cursor` | 闭源 | P1 完整 | 保持 |
| 5 | pi | `pi` | 开源（badlogic/pi-mono） | P0 完整 | 保持 |
| 6 | Grok | `grok` | 开源（xai-org/grok-build, Apache-2.0） | P0 完整 | 保持 |
| 7 | ZCode | `zcode` | 闭源（Z.ai） | P1 完整 | ✅ 路径已由官方文档证实；补 skills/hooks/subagents |
| 8 | Oh My Pi | `omp` | 开源（can1357/oh-my-pi） | **缺** | **新增完整 adapter** |
| 9 | DimAgent | `dimagent` | 闭源（npm `dimagent`） | **缺** | **新增（MCP + models）** |
| 10 | Antigravity | `antigravity-cli` | 闭源（Google） | P1 完整 | 官方路径已证实 = 现状路径 |
| 11 | Hermes | `hermes` | 开源 MIT（NousResearch） | **缺** | **新增完整 adapter（YAML）** |
| 12 | OpenCode | `opencode` | 开源（anomalyco/opencode） | P0 完整 | 保持 |
| 13 | Kimi Code | `kimi-code` | 开源 MIT（MoonshotAI） | P1 完整 | 补 config.toml models / skills 本机路径 |
| 14 | Copilot CLI | `copilot` | 闭源（GitHub） | P0 完整 | 保持 |
| 15 | DeepSeek Harness | `dsh` | 开源 MIT（deepseek-ai） | **缺** | **新增（settings + cordis 插件 MCP）** |
| 16 | OpenClaw | `openclaw` | 开源 MIT | **缺** | **新增（JSON5 + skills）** |
| 17 | WorkBuddy | `workbuddy` | 闭源（腾讯 = CodeBuddy） | P0 完整 | 保持 |
| 18 | Kiro | `kiro` | 闭源（AWS） | P1 完整 | 保持 |
| 19 | MiMoCode | `mimocode` | 开源 MIT（Xiaomi，OpenCode fork） | **缺** | **新增（OpenCode 方言）** |
| 20 | Alma | `alma` | 闭源桌面客户端 | **缺** | **UI-only，需决策**（见 §5） |
| 21 | Trae CLI | `trae`（IDE 路径） | 闭源（字节）/ Agent CLI 开源 | P0 部分 | 保持 IDE 路径；Trae Agent CLI 另议 |
| 22 | Gemini CLI | `gemini` | 开源 Apache-2.0 | P0 过渡 | 保持过渡（D14） |
| 23 | Qoder | `qoder` | 闭源（阿里） | P1 完整 | 保持 |
| 24 | CraftAgent | `craft` | 开源（craft-ai-agents/craft-agents-oss） | **缺** | **新增待定（workspace 源结构）** |
| 25 | Qoder CN | `qoder-cn` | 闭源（阿里 = 通义灵码） | P1 完整 | 保持 |
| 26 | Droid | `droid` | 闭源（Factory） | P1 完整 | 保持 |
| 27 | Cline | `cline` | 开源 Apache-2.0 | P1 完整 | 保持；MCP 路径已证实 |
| 28 | Cola | — | 未确认 | **缺** | **身份待确认，见 §5** |
| 29 | MiniMax Code | `mmx` | 开源（MiniMax-AI/cli） | **缺** | **新增（config + skills L2）** |
| 30 | Roo Code | — | 开源，**已停运** | 不做（D13） | 2026-05-15 归档；见 §5 |

---

## 1. 调研方法

三家来源交叉，**不靠单一搜索**：

1. **官方文档**（配置页 / MCP 页 / hooks 页）——闭源工具的唯一权威来源。
2. **源码**（`gh api .../git/trees` + `raw.githubusercontent.com` + GitHub code search）——开源工具
   的实际路径常量、schema、合并语义。优先级高于二手博客。
3. **二手榜单**（awesome-cli-coding-agents 2026-09-13、vercel-labs/skills 快照）——只用于发现与排名，
   不作为路径依据。

本文每条路径都标注了来源。凡未取到一手证据的，标 **❓待验证**，**不按猜测写代码**。

---

## 2. 逐家 profile（可同步面）

字段含义：`配置根` / `skills` / `sub-agents` / `MCP 文件+键` / `hooks` / `model-provider`。
`env` 指可覆盖根目录的环境变量（Magic 解析时应尊重，与现有 `KIMI_CODE_HOME` 处理一致）。

### 2.1 已完整支持（保持，逐条复核）

| harness | 配置根 | skills | agents | MCP | hooks | models | 证据 |
|---|---|---|---|---|---|---|---|
| claude | `~/.claude/` + `~/.claude.json` | `~/.claude/skills` | `~/.claude/agents` | `~/.claude.json` `mcpServers` | `settings.json` hooks | `settings.json` env | 现有实现 + 官方文档 |
| codex | `~/.codex/` | `~/.codex/skills` | — | `config.toml` `[mcp_servers.*]` | `hooks.json` | `config.toml`+`models.json` `env_key` | 现有实现 |
| cursor | `~/.cursor/` | `~/.cursor/skills` | — | `~/.cursor/mcp.json` `mcpServers` | `.cursor/hooks.json` | 账号/UI | 现有实现 |
| pi | `~/.pi/agent/` | `~/.pi/agent/skills` | `~/.pi/agent/agents` | `mcp.json` `mcpServers` | `extensions/` | `models.json` | 源码 `packages/coding-agent/src/config.ts`（`CONFIG_DIR_NAME=".pi"`） |
| grok | `~/.grok/`（`GROK_HOME`） | `~/.grok/skills` | `~/.grok/agents` | `config.toml` `[mcp_servers.*]` | `hooks/*.json` | managed catalog | 源码 `crates/codegen/xai-dirs/src/lib.rs`（`$GROK_HOME` else `<home>/.grok`） |
| opencode | `~/.config/opencode/` | `…/skills` | `…/agents` | `opencode.json` `mcp` | `plugins/` | `provider.*` | 现有实现 |
| copilot | `~/.copilot/` | `~/.copilot/skills` | `~/.copilot/agents` | `mcp-config.json` `mcpServers` | `hooks/` | BYOK | 现有实现 |
| workbuddy | `~/.codebuddy/`（Detect 亦认 `.workbuddy`） | `.codebuddy/skills` | `agents` | `settings.json` `mcpServers` | `settings.json` hooks | `models.json` | 现有实现 |
| trae | `~/.trae/`（国内 `~/.trae-cn/`） | `~/.trae/skills` | `~/.trae/agents` | `~/.trae/mcp.json` `mcpServers` | ❓ | UI | 现有实现 + Trae 文档 |
| gemini | `~/.gemini/` | — | — | `settings.json` `mcpServers` | `hooks` | ❓ | 现有实现（D14 过渡） |
| qoder / qoder-cn | `~/.qoder/` / `~/.qoder-cn/` | `~/.qoder/skills` | `agents` | `settings.json` `mcpServers` | hooks（20+） | custom models | 现有实现 + Qoder CLI 文档 |
| kiro | `~/.kiro/` | `~/.kiro/skills` | `agents` | `~/.kiro/settings/mcp.json` `mcpServers` | hooks | ❓ | 官方 issue/builder.aws 证实 `~/.kiro/settings/mcp.json` |
| droid | `~/.factory/` | `~/.factory/skills` | `agents` | `~/.factory/mcp.json` `mcpServers` | hooks | `settings.json` | Factory 文档 + issue #621 证实 `~/.factory/mcp.json` |
| cline | `~/.cline/` | `~/.agents/skills` | — | `~/.cline/mcp.json` `mcpServers` | 插件（SDK） | `~/.cline/data/settings` | Cline 官方文档（CLI: `~/.cline/mcp.json`） |
| qwen-code | `~/.qwen/` | `~/.qwen/skills` | agents | `settings.json` `mcpServers` | hooks | `settings.json` | 现有实现 |
| iflow-cli | `~/.iflow/` | `~/.iflow/skills` | — | `settings.json` `mcpServers` | ❓ | `apiKey`/`baseUrl` | 现有实现 |
| kilo | `~/.config/kilo/` | `~/.kilo/skills` | agents | `config.json` `mcp` | plugins | `provider.<n>.models` | 现有实现 |
| zed | `~/.config/zed/` | `~/.agents/skills` | — | `settings.json` `context_servers` | ❓ | ❓ | 现有实现 |
| mcpuser | `~/.config/mcp/` | — | — | `mcp.json` `mcpServers` | — | — | 现有实现（非 harness） |

### 2.2 本轮新增：可完整支持（一手证据齐全）

#### `omp` — Oh My Pi（开源，can1357/oh-my-pi，Pi fork）

| 项 | 值 | 证据 |
|---|---|---|
| 配置根 | `~/.omp/`（`PI_CONFIG_DIR`，默认 `.omp`） | `packages/utils/src/dirs.ts`（`APP_NAME="omp"`，`CONFIG_DIR_NAME=".omp"`） |
| agent 目录 | `~/.omp/agent/`（`PI_CODING_AGENT_DIR`）；profile 时 `~/.omp/profiles/<name>/agent` | 同上 + `docs/settings.md` |
| 主设置 | `~/.omp/agent/config.yml`（`config.yaml` 兼容）；`models.yml` | `docs/settings.md` |
| skills | `~/.omp/agent/skills/<n>/SKILL.md`；项目 `.omp/skills/` | `src/discovery/builtin.ts`（user scan `getAgentDir()/skills`） |
| MCP | `~/.omp/agent/mcp.json`，键 `mcpServers`；兼容 `.omp/.mcp.json` | `docs/mcp-config.md` |
| hooks/extensions | `~/.omp/agent/extensions/`、`hooks/<type>/*.ts`（代码式） | `docs/hooks.md` + `discovery/builtin.ts` |
| 自定义文件 | `RULES.md`、`SYSTEM.md`、`AGENTS.md`、`AGENTS.md`(project) | `docs/settings.md` / `discovery/builtin.ts` |
| models | `config.yml` + `models.yml` + `.env`；多 provider | `docs/settings.md` / `docs/providers.md` |
| 导入 | 会读 `.claude/`、`.codex/`、`.cursor/`、`.gemini/`、`opencode.json`、`.agents/` | `docs/mcp-config.md` |

结论 caps：`Skills ✅ · Agents ✅（SYSTEM/RULES + 兼容 `.pi`/Claude agents）· MCP ✅ · Hooks ✅（代码式）· Models ✅`。
MCP dialect = 通用 `mcpServers` + `disabled`（与 Pi 同族）；remote 支持（`url`/`headers`）。

#### `hermes` — Hermes Agent（开源 MIT，NousResearch）

| 项 | 值 | 证据 |
|---|---|---|
| 配置根 | `~/.hermes/`（`HERMES_HOME`） | `hermes_cli/config.py`、`hermes_cli/config_home.py` |
| 主设置 | `~/.hermes/config.yaml`；`~/.hermes/.env` | `hermes_cli/config.py:471` |
| skills | `~/.hermes/skills/`（agentskills.io 标准） | `_HERMES_HOME_SUBDIRS` 含 `skills`；vercel 表 |
| MCP | `config.yaml` 键 `mcp_servers`（stdio `command/args/env`；remote `url/headers/ssl_verify`） | `website/docs/reference/mcp-config-reference.md` |
| hooks | `~/.hermes/hooks/`（目录骨架存在） | `_HERMES_HOME_SUBDIRS` |
| sub-agents | 运行期 spawn（无独立声明目录） | README（“Spawn isolated subagents”） |
| models | `config.yaml` providers + `hermes model` | `docs/user-guide/configuring-models` |
| 备注 | `agentskills.io` 兼容；`claude-code`/`codex` 可作子 harness | README |

结论 caps：`Skills ✅ · MCP ✅（YAML）· Hooks ✅ · Models ✅ · Agents —（运行期，无声明文件）`。
**需要 YAML 格式写回**（现有 MCP 只支持 JSON/TOML）。

#### `mimocode` — MiMo Code（开源 MIT，Xiaomi，OpenCode fork）

| 项 | 值 | 证据 |
|---|---|---|
| 配置根 | `$MIMOCODE_HOME/config` 或 `$XDG_CONFIG_HOME/mimocode`（默认 `~/.config/mimocode`） | `packages/shared/src/global.ts`（`APP="mimocode"`） |
| 主设置 | `~/.config/mimocode/mimocode.json` / `.jsonc` | `packages/opencode/src/config/config.ts:542,642` |
| MCP | 同文件键 `mcp`（OpenCode 方言：`type: local/remote`、`command[]`、`environment`） | fork 自 OpenCode；`packages/opencode/src/config/config.ts` |
| skills | 项目 `.mimocode/skills/`；也读 `.claude/commands` | `config/paths.ts`（`.mimocode`）、repo `.mimocode/skills` |
| workflows | `.mimocode/workflows/*.js` | README |
| models | 同 `mimocode.json`（OpenCode provider 结构） | README“supports any mainstream LLM provider” |
| 备注 | 直接复用 OpenCode 方言（`StyleOpenCode`） | README“fork from OpenCode” |

结论 caps：`Skills ✅ · MCP ✅（OpenCode 方言）· Models ✅ · Agents（subagent 运行期）· Hooks ❓`。
MCP target 一行即可（`~/.config/mimocode/mimocode.json`，`mcp`，StyleOpenCode）。

#### `dimagent` — DimAgent（闭源 npm `dimagent`，ACP）

| 项 | 值 | 证据 |
|---|---|---|
| 配置根 | `~/.dim/v2/`（`DIMCODE_HOME`） | 官方 docs `/docs/config` |
| MCP | `~/.dim/v2/mcp.json`，键 `mcpServers`；项目 `.mcp.json` | 同上；entry 支持 `name/command/args/env/cwd/enabled`，remote `type:http`+`url`+`headers`/`auth` |
| skills/hooks/plugins | TUI `/plugins`、`/mcp`；声明路径 ❓ | 官网（Plugins/MCP/Skills/Hooks/subagents 提及） |
| models | `~/.dim/v2/credentials*` + `/connect`；env `DIMCODE_MODEL` 等 | 同上 |
| remote | 仅 `http`（`sse` 会 warning 并跳过） | 同上 |

结论 caps：`MCP ✅ · Skills ❓ · Models ✅（credentials；写回未定）`。
MCP 需要 **跳过 sse**，并支持 `enabled`、remote `type:http`。

#### `openclaw` — OpenClaw（开源 MIT）

| 项 | 值 | 证据 |
|---|---|---|
| 配置根 | `~/.openclaw/`（`OPENCLAW_HOME`/`OPENCLAW_STATE_DIR`/`OPENCLAW_CONFIG_PATH`） | `docs/start/setup.md`、`docs/nodes/mcp-and-skills.md` |
| 主设置 | `~/.openclaw/openclaw.json`（JSON/JSON5-ish） | 同上 |
| MCP | 同文件键 `mcp.servers`（`command/args/cwd` 或 `url/transport/headers`、`toolFilter`）；节点 `nodeHost.mcp.servers` | `docs/tools/mcp.md`、`docs/nodes/mcp-and-skills.md` |
| skills | `~/.openclaw/skills/<n>/SKILL.md`（`OPENCLAW_STATE_DIR/skills` 优先） | 同上 + vercel 表 |
| workspace | `~/.openclaw/workspace/`（prompts/memories） | `docs/start/setup.md` |
| models | 多 provider（`docs/providers/*`） | 文档目录 |

结论 caps：`Skills ✅ · MCP ✅（JSON5，嵌套键 mcp.servers）· Models ✅（provider 块 ❓ 键名）`。
需要 **JSON5/JSON 嵌套写回**（现有 `mergeJSONConfig` 已支持 dotted key `mcp.servers`）；
JSON5 读取依赖 `secrets.StripJSONC`（只处理注释/尾逗号，不处理无引号键）→ 需评估。

#### `dsh` — DeepSeek Harness（开源 MIT）

| 项 | 值 | 证据 |
|---|---|---|
| 配置根 | `~/.dsh/`（`DSH_HOME`） | `packages/util/home-paths/src/index.ts`（`DSH_HOME_DIR_NAME=".dsh"`） |
| settings | `~/.dsh/settings.yaml`（YAML/JSON） | `packages/settings/settings-file/src/index.ts:23-25` |
| MCP | 通过 `@deepseek-ai/dsh-mcp-client` 插件行，写在 cordis 叠加层 `~/.dsh/cordis.patch.yml`（或 `~/.dsh/profiles/<n>/cordis.patch.yml`） | `docs/user/guide/mcp-memory.md` |
| skills | plugin 化（`packages/skill/skill-filesystem`） | 源码树 |
| models | `settings.yaml` + 插件配置（provider/model） | `docs/config-catalog.md` |
| 备注 | “everything is a plugin”，cordis YAML 树结构，**不是扁平配置** | README |

结论 caps：`MCP ✅（经 cordis 插件，YAML patch）· Skills ❓ · Models ❓`。
风险：MCP 投影要写 cordis `insert` 节点，**与现有 MCP 抽象不兼容**；建议列 P3，先做 `settings.yaml` 自定义层。

#### `mmx` — MiniMax Code CLI（开源，MiniMax-AI/cli）

| 项 | 值 | 证据 |
|---|---|---|
| 配置根 | `~/.mmx/`（`MMX_CONFIG_DIR`） | `src/config/paths.ts`（`CONFIG_DIR_NAME=".mmx"`） |
| 主设置 | `~/.mmx/config.json`（region、默认模型、api_key、OAuth token） | README + `src/config/schema.ts` |
| skills | `npx skills add MiniMax-AI/cli -y -g` → 走 agentskills（L2） | README |
| MCP | ❓（CLI 面向模型调用，未见 MCP host 文档） | — |
| models | `mmx config set --key default-text-model` | README |

结论 caps：`Models ✅（config.json）· Skills ✅（L2 目录）· MCP —（❓）`。

### 2.3 本轮新增：部分/待定

| harness | 可同步面 | 缺什么 | 处置 |
|---|---|---|---|
| `craft`（Craft Agents OSS，开源） | 配置根 `~/.craft-agent/`（`CRAFT_CONFIG_DIR`）；`config.json`；workspace `workspaces/<id>/`（`config.json`、`skills/`、`sources/`） | `sources/` 每源一个 `config.json`（MCP/API/local），**无单一 mcpServers 映射** | 先只做 skills/自定义镜像，MCP 投影 P3 |
| `zcode` | 用户 `~/.zcode/cli/config.json` `mcp.servers`；skills `~/.zcode/skills/<n>/SKILL.md`；项目 `<root>/.zcode/config.json`；`.agents/mcp.json` 兼容 | hooks / subagents / commands 文件键 ❓ | 补 skills（已实现）；hooks 待查 |
| `trae`（Agent CLI，开源） | 与 Trae IDE 不同产品；`bytedance/trae-agent` | 配置文件名 ❓ | 另立 id（如 `trae-agent`），P3；`trae` 仍指 IDE/CLI 共用 `~/.trae` |

### 2.4 UI-only / 无配置投影面（本轮不投）

| harness | 结论 | 证据 |
|---|---|---|
| `alma`（闭源桌面） | 供应商编排客户端（填 Base URL + API Key + Fetch models），无 skills/MCP 配置文件文档 | alma.now、docs.ikuncode.cc/zh/apps/alma |
| `cola` | 未确认；`cola-io` 仅有 `codex-acp`（ACP 桥）与 `koffee`，非可配置 harness | GitHub org `cola-io` |
| `roo`（Roo Code） | **2026-05-15 归档停运**，团队转向 Roomote；社区 fork ZooCode | thenewstack / kilo / GH README |

---

## 3. 能力矩阵（本轮 29 家汇总）

符号：**✅ 已实现** · **○ 工具支持、Magic 未做** · **❓ 待调研** · **— 不支持/不适用**

| # | harness | skills | agents | mcp | models | hooks | custom | switch |
|---|---|---|---|---|---|---|---|---|
| 2 | codex | ✅ | ○ | ✅ | ✅ | ○ | ✅ | ✅ |
| 3 | claude | ✅ | ✅ | ✅ | ✅ | ○ | ✅ | ✅ |
| 4 | cursor | ✅ | — | ✅ | — | ✅ | ○ | — |
| 5 | pi | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 6 | grok | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 7 | zcode | ✅ | ○ | ✅ | ○ | ❓ | ○ | — |
| 8 | **omp** | **新** | **新** | **新** | **新** | **新** | **新** | ❓ |
| 9 | **dimagent** | ❓ | ❓ | **新** | ○ | ❓ | ○ | — |
| 10 | antigravity-cli | ✅ | — | ✅ | ❓ | ✅ | ✅ | — |
| 11 | **hermes** | **新** | — | **新** | **新** | **新** | **新** | ❓ |
| 12 | opencode | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 13 | kimi-code | ✅ | ✅ | ✅ | ○ | ✅ | ✅ | ○ |
| 14 | copilot | ✅ | ○ | ✅ | ○ | ○ | ✅ | — |
| 15 | **dsh** | ❓ | ❓ | ○ | ❓ | ❓ | **新** | ❓ |
| 16 | **openclaw** | **新** | ○ | **新** | ❓ | ❓ | **新** | ❓ |
| 17 | workbuddy | ✅ | ○ | ✅ | ✅ | ○ | ✅ | ✅ |
| 18 | kiro | ✅ | ○ | ✅ | ❓ | ○ | ○ | — |
| 19 | **mimocode** | **新** | ○ | **新** | **新** | ❓ | **新** | ○ |
| 20 | alma | — | — | — | — | — | — | — |
| 21 | trae | ✅ | ✅ | ✅ | ○ | ❓ | ✅ | — |
| 22 | gemini | — | — | ✅ | ❓ | ○ | ✅ | — |
| 23 | qoder | ✅ | ✅ | ✅ | ○ | ○ | ○ | — |
| 24 | **craft** | ○ | — | ○ | ❓ | ❓ | ○ | — |
| 25 | qoder-cn | ✅ | ✅ | ✅ | ○ | ○ | ○ | — |
| 26 | droid | ✅ | ✅ | ✅ | ❓ | ○ | ○ | — |
| 27 | cline | ✅ | — | ✅ | ❓ | ○ | ○ | — |
| 28 | cola | — | — | — | — | — | — | — |
| 29 | **mmx** | ✅(L2) | — | — | **新** | — | ○ | ○ |
| 30 | roo | — | — | — | — | — | — | — |

---

## 4. 实施计划（按代码改动面分批）

> 原则：**先补一手证据齐全、改动面小、收益高的**；每个批次 `go build` + `go test ./...` 通过后再进下一批。
> 所有写回遵循 D5 字段级合并、D10 能力不对等不报错、D18/D19 的 MCP/hooks 规范。

### 批次 1（新增 4 个 adapter + MCP target）

| 任务 | 文件 | 内容 |
|---|---|---|
| 1.1 `omp` adapter | `internal/adapters/omp/omp.go` | root `~/.omp/agent`；skills `agent/skills`；MCP `agent/mcp.json`；detect `~/.omp` |
| 1.2 `omp` MCP | `internal/mcp/target.go`、`translate.go` | 新 style `StyleOMP`（= Pi 族 + remote `url` + `disabled`）；`~/.omp/agent/mcp.json` `mcpServers` |
| 1.3 `hermes` adapter | `internal/adapters/hermes/hermes.go` | root `~/.hermes`；skills `skills/`；MCP 走 config.yaml |
| 1.4 `hermes` MCP（YAML） | `internal/mcp/target.go`、`apply.go`、新 `yaml.go` | `FormatYAML`；写 `config.yaml` 的 `mcp_servers`（stdio `command/args/env`；remote `url/headers`） |
| 1.5 `mimocode` adapter | `internal/adapters/mimocode/mimocode.go` | root `~/.config/mimocode`（`MIMOCODE_HOME`）；skills `skills/`；MCP `mimocode.json` |
| 1.6 `mimocode` MCP | `internal/mcp/target.go` | `StyleOpenCode`，path `~/.config/mimocode/mimocode.json`，key `mcp`，jsonc 读取 |
| 1.7 `dimagent` adapter | `internal/adapters/dimagent/dimagent.go` | root `~/.dim/v2`（`DIMCODE_HOME`）；MCP `mcp.json` |
| 1.8 `dimagent` MCP | `internal/mcp/target.go`、`translate.go` | 新 style `StyleDimAgent`：stdio `{command,args,env,enabled}`；remote `{type:"http",url,headers,enabled}`；**sse → n/a** |
| 1.9 注册 | `internal/registry/registry.go` | 追加 4 个 adapter |
| 1.10 skills 路径 | `internal/skills/paths.go` | 补 `omp`(?) / `dimagent`(?) / `mimocode`(?) / `openclaw`(已有) / `hermes-agent`(已有) / `mmx`(?)；只加有一手路径的 |

### 批次 2（OpenClaw + 现有 adapter 补全）

| 任务 | 文件 | 内容 |
|---|---|---|
| 2.1 `openclaw` adapter | `internal/adapters/openclaw/openclaw.go` | root `~/.openclaw`；skills `skills/` |
| 2.2 `openclaw` MCP | `internal/mcp/target.go`、`translate.go` | style `StyleOpenClaw`；path `~/.openclaw/openclaw.json`，嵌套键 `mcp.servers`；remote `{url,transport,headers}` |
| 2.3 JSON5 读取加固 | `internal/secrets/store.go` 或新 helper | 现有 `StripJSONC` 需支持无引号键 / 单引号（OpenClaw `openclaw.json` 为 JSON5-ish）；失败则**不写、报 n/a**，不破坏文件 |
| 2.4 `mmx` adapter | `internal/adapters/mmx/mmx.go` | root `~/.mmx`；models `config.json`；skills L2 `~/.agents/skills` |
| 2.5 `kimi-code` skills 本机路径 | `internal/adapters/kimicode/kimicode.go` | 官方文档：Kimi 专属 user skills = `$KIMI_CODE_HOME/skills`（通用 `.agents/skills` 是共享层）；改为写 `skills/` |
| 2.6 `zcode` skills | 已实现，校验 `~/.zcode/skills` | 官方文档证实 |
| 2.7 `cline` MCP 路径 | 已实现 `~/.cline/mcp.json` | 官方文档证实 |

### 批次 3（复杂/待证据，先文档不写代码）

- `dsh`：cordis patch YAML（`~/.dsh/cordis.patch.yml`）里 `insert` MCP 插件行。需要新 writer + 与现有 `Target` 抽象解耦；
  先只做 `settings.yaml` 的 custom 镜像，MCP 单独设计（见 §6）。
- `craft`：workspace `sources/` 结构未取全；只做 `~/.craft-agent/` 的 skills/自定义镜像。
- `trae-agent`（开源 CLI）：另立 id，待取配置文件名。
- `zcode` hooks / `kiro` models / `droid` models / `cline` models：待查后补矩阵。

---

## 5. 需用户决定 / 明确不做

> **2026-09-13 用户确认**：本节 1–5 全部按建议执行，**本轮不动**。Roo 不做、Alma/Cola 不投、
> DSH MCP 暂缓、Trae 保持 IDE 路径；榜单第 1 名缺失不影响执行。除非用户后续主动提出，
> 这些不再列为待办。

1. **Roo Code（#30）**：已确认 2026-05-15 归档停运。**建议不做**（维持 D13）。如需照顾存量用户，
   可加 **ZooCode**（社区 fork）作为替代——请确认。
2. **Alma（#20）**：闭源桌面客户端，只有供应商配置（Base URL/Key/模型）在 GUI 内，无 skills/MCP 配置文件文档。
   **建议**：标 `UI-only`，Magic 不投影；provider 覆盖已由 `providers.yaml` 承担。
3. **Cola（#28）**：未找到与“AI 编程 harness”匹配的同名产品（`cola-io/codex-acp` 是 ACP 桥）。
   **请提供官网/仓库链接**，否则标 `未确认`。
4. **DeepSeek Harness MCP 语义**：DSH 用 cordis 插件树而非扁平 MCP 配置。是否值得为它单独实现
   `cordis.patch.yml` 写入器（成本高、产品处于 developer preview、可能破坏兼容）？
   **建议**：P3 暂缓，先做 settings/自定义层。
5. **Trae**：`trae` 目前指 Trae IDE 的 `~/.trae`。榜单写“Trae CLI”，若指开源的 `bytedance/trae-agent`，
   则需**新增**一个独立 adapter。请确认。

---

## 6. 验证方案

| 层级 | 手段 |
|---|---|
| 单元 | `internal/mcp/translate_test.go` 增补 4–6 个新 style 的 canonical→entry 断言（含 remote/sse/disabled 边界） |
| 单元 | YAML 写回：新增 `internal/mcp/yaml_test.go`，断言不覆盖用户其它键 |
| 单元 | 新 adapter：`Detect`/`SkillsDir`/`MirrorRoot` 表驱动测试（参考现有 `*_test.go`） |
| 集成 | 临时 HOME 下跑 `magic apply --dry-run`，核对路径与键；再跑真实 apply 后 diff 原文件，确认字段级合并 |
| 回归 | `go build -o magic ./cmd/magic` + `go test ./...`（AGENTS.md 硬性要求） |
| 文档 | 更新 `docs/supported-harnesses.md`、`docs/harness-matrix.md`、`docs/decisions.md`（新增 D21 批次范围） |

**不做**：不打真实网络（registry 单测用 fixture，AGENTS.md 要求）；不写会破坏用户文件的正则式 JSON5 解析。

---

## 7. 风险

| 风险 | 影响 | 缓解 |
|---|---|---|
| 闭源工具路径随版本变化 | 写错位置 | 每个 target 标注来源与日期；`Detect` 失败即跳过；不造空文件 |
| JSON5 解析不完整（OpenClaw） | 破坏用户配置 | 解析失败即 `n/a` 跳过 + 日志，**绝不**重写整文件 |
| YAML 写回丢注释 | 用户配置可读性 | 用 `yaml.v3` 的 Node 级合并（保留注释）或选择专用文件（Hermes `mcp_servers` 独立键） |
| DSH developer preview 破坏性变更 | 适配器快速失效 | 明确标 preview，P3 暂缓 |
| 名单第 1 名缺失 | 覆盖不全 | 待用户补充 |

---

## 8. 自审记录

### 第一轮自审（方案完成时）

发现并修正：

1. **Antigravity 路径歧义**：二手来源（learn.arm.com）称 `~/.gemini/antigravity-cli/mcp_config.json`，
   与现有实现 `~/.gemini/config/mcp_config.json` 冲突。以 **antigravity.google 官方文档**为准
   （IDE/CLI 均写明 `~/.gemini/config/mcp_config.json` + workspace `.agents/mcp_config.json`），
   现有实现**正确**，不改。→ 已在 §2.1 记录证据。
2. **Cline MCP 路径**：旧 issue 称 `~/.cline/data/settings/cline_mcp_settings.json`（那是 VS Code 扩展的
   globalStorage，不是 CLI）。官方文档明确 **CLI = `~/.cline/mcp.json`**，现有实现正确。
3. **Kimi Code skills 路径**：现实现写 `~/.agents/skills`（共享层）。官方 docs 区分：通用 Skills 在
   `~/.agents/skills`，**Kimi 专属** user skills 在 `$KIMI_CODE_HOME/skills`。→ 批次 2 修正为 `skills/`（保留兼容）。
4. **Hermes 需要 YAML**：现有 MCP 抽象只有 JSON/TOML，方案里补了 `FormatYAML` 任务，否则无法落地。
5. **Droid/Kiro 路径**：分别由 issue #621 / kiro issue #5047 与 builder.aws 证实，现有实现正确。

### 第二轮自审（对照源码/文档逐条重核）

方法：对每个“新增”结论回查一手文件；对每个“保持”结论抽 1 条路径核对。

| 结论 | 复核动作 | 结果 |
|---|---|---|
| omp 根 `~/.omp/agent` | 重读 `packages/utils/src/dirs.ts` + `docs/settings.md` 表格 | ✅ 一致 |
| omp MCP `~/.omp/agent/mcp.json` | 重读 `docs/mcp-config.md` “Preferred config locations” | ✅ 一致（另有 `.omp/.mcp.json` 兼容） |
| hermes `config.yaml` + `mcp_servers` | 重读 `config.py:471` + `mcp-config-reference.md` | ✅ 一致 |
| mimocode `~/.config/mimocode/mimocode.json` | 重读 `packages/shared/src/global.ts` + `config.ts:542` | ✅ 一致 |
| dimagent `~/.dim/v2/mcp.json` | 重读官方 `/docs/config` 目录树 + mcp.json 示例 | ✅ 一致 |
| openclaw `~/.openclaw/openclaw.json` `mcp.servers` | 重读 `docs/tools/mcp.md` + `docs/nodes/mcp-and-skills.md` | ✅ 一致 |
| dsh `~/.dsh/settings.yaml` | 重读 `settings-file/src/index.ts:23-25` + `home-paths` | ✅ 一致 |
| mmx `~/.mmx/config.json` | 重读 `src/config/paths.ts` + README | ✅ 一致 |
| craft `~/.craft-agent/` | 重读 `config/paths.ts` + README §Configuration | ✅ 根一致；MCP 结构仍 ❓ |
| roo 停运 | 三方来源（TheNewStack / kilo / GH README） | ✅ 一致，2026-05-15 |
| zcode `mcp.servers` | 重读官方 MCP 文档“Configuration Files and Default Load Paths” | ✅ 一致；skills `~/.zcode/skills` 亦证实 |

第二轮**未发现新的路径错误**；新增标记的 ❓ 项（craft MCP 结构、dsh 插件 schema、alma/cola 身份）已在
§5/§2.3 明列，不作为实现依据。

**结论**：批次 1/2 的一手证据充分，可开始开发；批次 3 与 §5 决策项在实现前保持挂起。

---

## 9. 实施进度（2026-09-13）

### 已完成（批次 1 + OpenClaw）

| 项 | 文件 | 状态 |
|---|---|---|
| `omp` adapter | `internal/adapters/omp/omp.go` | ✅ |
| `hermes` adapter | `internal/adapters/hermes/hermes.go` | ✅ |
| `mimocode` adapter | `internal/adapters/mimocode/mimocode.go` | ✅ |
| `dimagent` adapter | `internal/adapters/dimagent/dimagent.go` | ✅ |
| `openclaw` adapter | `internal/adapters/openclaw/openclaw.go` | ✅ |
| MCP `FormatYAML` writer | `internal/mcp/yaml.go` | ✅（`yaml.Node` 合并，保留注释） |
| MCP dialect `omp`/`hermes`/`dimagent`/`openclaw` | `internal/mcp/translate.go` | ✅ |
| MCP targets | `internal/mcp/target.go` | ✅ 5 行 |
| registry 注册 | `internal/registry/registry.go` | ✅ 28 个 adapter |
| skills 路径 | `internal/skills/paths.go` | ✅ `omp`/`oh-my-pi`；`DefaultAgents` +4 |
| 测试 | `internal/mcp/{translate,apply}_test.go` | ✅ 新 dialect / YAML / 嵌套键 / sse 跳过 |
| 端到端 | 临时 HOME `magic apply` | ✅ 5 个 target 写入正确 |
| `go build` + `go test ./...` | — | ✅ 全绿 |

### 未做（下一步）

- 批次 2：`mmx` 适配器（目前无 MCP，收益低，暂缓）、`zcode` hooks、`kiro`/`droid`/`cline` models。
- 批次 3：`dsh` cordis MCP（D24）、`craft` sources 结构（❓）、`trae-agent`。
- 决策项：§5 的 Roo / Alma / Cola / Trae 待用户确认。

### 实施阶段自审（对着代码复查）

1. **dimagent 的 `sse` 处理**：`Translate` 返回 `ok=false` + reason，`Project` 记 `n/a` 并跳过，
   但**同一次 apply 里其它 server 照常写入**——已验证（`TestProjectDimAgentSkipsSSE`）。
2. **hermes YAML 合并**：只替换 `mcp_servers` 下受管条目，`model`/`telemetry` 等键与注释保留——
   已验证（`TestProjectYAMLPreservesConfigAndComments`）。
3. **openclaw 的 `enabled`**：OpenClaw 无确认的开关字段，故**未**把 `StyleOpenClaw` 加入
   `styleSupportsEnabled`；`enabled: false` 的 server 会记 `n/a` 跳过，不会写出语义不明的字段。
4. **env 覆盖与 MCP 路径的一致性**：adapter 的 `Detect`/`SkillsDir` 尊重 `PI_CODING_AGENT_DIR` /
   `HERMES_HOME` / `MIMOCODE_HOME` / `DIMCODE_HOME`，但 MCP target 的 `Path` 仍用默认路径
   （与既有 `kimi-code` 的 `KIMI_CODE_HOME` 同样是已知限制）。→ 记为 **FUTURE**：`Target.Path` 增加
   env 感知（`func(home string) string` 需要能读环境或用 resolver）。
5. **`kimi-code` skills 路径**：官方区分「通用 `.agents/skills`（跨工具共享）」与「`$KIMI_CODE_HOME/skills`
   （Kimi 专属）」。Magic 保持写 `.agents/skills`（跨工具共享标准，cline/zed 同路径），**不改**——
   这是有意选择，不是遗漏。

### 实施阶段自审补充

6. **OpenClaw JSON5 加固**：`Target` 新增 `LenientJSON` 字段；`internal/mcp/apply.go` 在写入前用
   `jsonConfigParseable` 探测，无法按 JSON/JSONC 解析时记 `n/a (config is not plain JSON; left untouched)`
   并**跳过**，不改文件（`TestProjectOpenClawLeavesJSON5Untouched`）。避免破坏带注释/无引号键的用户配置。
