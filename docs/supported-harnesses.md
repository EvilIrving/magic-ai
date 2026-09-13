# 支持的 harness（目标清单）

状态：**清单（已定）**。目标 harness 与能力支持矩阵；已锁定的产品决定见 [decisions.md](./decisions.md)。

## 名单来源（provenance）

harness / skills 路径这份清单的**来源**必须记住，以便复核与刷新：

| 项 | 值 |
|---|---|
| 来源仓库 | `vercel-labs/skills` （GitHub，MIT） |
| URL | https://github.com/vercel-labs/skills |
| 章节 | README → **Supported Agents** 表（`--agent` / Project Path / Global Path） |
| 快照日期 | **2026-09-11**（抓取日期；仓库会变，重核时更新此日期） |
| 相关 | 目录站 https://skills.sh · 协议规范 https://agentskills.io |
| 用途 | **直接采纳为 L2 skills 路径表**——固定配置，拿来就用，**不做运行时探测**。以后 vercel 名单新增，就加一行 |

刷新方式：vercel README 新增 agent 时，把**新增行**加到本文件 L2 表即可。
（代码里同样注明来源与快照日期。）**没有「真相源」之争：这是公开表格，直接采纳。**

另一参考（对比用，非名单来源）：`winder.ai` 的 AI Agent Harness Comparison（2026-08），只用于理解各 harness 定位。

## 先分清两条轴

| 轴 | 是什么 | Magic 现状 |
|---|---|---|
| **model provider** | `providers.yaml` 里配哪家模型/中转 | 内置模板 34 家 + 任意手加；**不设上限** |
| **harness** | 把配置投到哪个工具 | 固定适配器名单；**这是受限的那一栏** |

Cherry Studio 内置几十家的是 **provider**（它是聊天客户端，不管 harness）。Magic 的 provider 覆盖
已经够宽，真正要扩的是 harness。

## 能力分档（决定成本）

| 档 | 内容 | 成本 | 覆盖手段 |
|---|---|---|---|
| **L1 完整 adapter** | skills + agents + MCP + **provider/模型配置投影** | 高（每个工具格式不同，需定制） | 逐个实现 |
| **L2 skills-only** | 仅把 `packs/skills` 链到该工具 skills 目录 | 极低（一行目录映射） | 一张表覆盖几十家 |
| **L3 SwitchWriter** | official ↔ 中转切换 | 高 | 逐个实现 |
| **不做** | 明确排除 | — | — |

**关键结论**：80% 覆盖率不必写 80 个 adapter。**用 L2 目录表覆盖长尾，用 L1 覆盖主流。**

---

## 现状（已注册 10 个 adapter）

| ID | 名称 | L2 skills | agents | MCP | models 配置 | L3 writer |
|---|---|---|---|---|---|---|
| `pi` | Pi Agent | ✅ | ✅ | | ✅ | ✅ |
| `claude` | Claude Code | ✅ | ✅ | | ✅(env) | ✅ |
| `codex` | Codex CLI | ✅ | ✅ | | ✅ | ✅ |
| `grok` | Grok CLI | ✅ | ✅ | | ✅ | ✅ |
| `opencode` | OpenCode | ✅ | ✅ | ✅ | ✅ | ✅ |
| `workbuddy` | WorkBuddy / CodeBuddy | | | | ✅ | ✅ |
| `gemini` | Gemini CLI | | | ✅ | | ❌ stub |
| `copilot` | GitHub Copilot CLI | ✅ | ✅ | ✅ | | ❌ stub |
| `trae` | Trae | ✅ | ✅ | ✅ | | ❌ stub |
| `mcpuser` | MCP 配置镜像（不是 harness） | | | ✅ | | n/a |

另有 `cursor` 在 skills 目标里但无产品 adapter。**不做 windsurf**（产品濒临停运 / 已被收购，2026；不要因 vercel 名单里有它而加回）。

> 能力不对等：不要求每个 harness 所有能力齐全。支持什么投什么，不支持就跳过。
> 能力支持矩阵见本文「Tier 与能力支持」；MCP / hooks 专项见 [mcp-design.md](./mcp-design.md) / [hooks-design.md](./hooks-design.md)。

---

## 覆盖集（13 家，本轮全做）

选型依据 = JetBrains DES 2026 **+ GitHub stars / npm 下载量交叉复核**（见 [harness-profiles.md](./harness-profiles.md) 文末「声量复核」）。
官方调研有自家盘子加成，且低估了 opencode（官方 7% vs **实测 206k stars 全类第一**）。

| # | harness | 声量依据 | 配置根 | 能力面 |
|---|---|---|---|---|
| 1 | **Claude Code** | 145k★ · npm 855万/周 | `~/.claude/` + `~/.claude.json` | skills·agents·MCP·models(env)·hooks(20+)·custom |
| 2 | **Codex** | 123k★ · npm **1320万/周** | `~/.codex/` | skills·MCP(TOML)·models(env_key)·hooks(12) |
| 3 | **GitHub Copilot CLI** | 4.7M 付费 | `~/.copilot/`、`.github/` | skills·agents·MCP·models(BYOK)·hooks(7) |
| 4 | **Cursor** | $2B ARR · 1M+ 付费 | `~/.cursor/`、`.cursor/` | skills·MCP·hooks(20+)·plugins |
| 5 | **OpenCode** | **206k★ 全类第一** · npm 131万/周 | `~/.config/opencode/` | skills·agents·MCP·models·plugins |
| 6 | **Google Antigravity** | gemini-cli 107k★（其前身） | `~/.gemini/antigravity-cli/` | skills·MCP(`serverUrl`)·hooks(5) |
| 7 | **Trae / Trae CN** | **MAU >100万** | `~/.trae/`、`~/.trae-cn/` | skills·MCP·models(UI)·rules |
| 8 | **Qoder / Qoder CN**（= 通义灵码） | 插件下载 **>1500万** | `~/.qoder/`、`~/.qoder-cn/`、`~/.lingma/` | skills·agents·MCP·models·hooks(23) |
| 9 | **CodeBuddy / WorkBuddy** | 腾讯 · 国内主流 | `~/.codebuddy/` | skills·agents·MCP·models·hooks |
| 10 | **Kimi Code** | 11k★ · 活跃 | `~/.kimi-code/` | skills·agents·MCP·hooks(TOML 20) |
| 11 | **Qwen Code** | 28k★ · 活跃 | `~/.qwen/` | skills·agents·MCP·models·hooks(17) |
| 12 | **Pi** | 自有 | `~/.pi/agent/` | 全能力 |
| 13 | **Grok CLI** | 既有 | `~/.grok/` | 全能力 |

> **出局**：Comate（已并入百度搭子）· CodeGeeX（repo 停更 2 年）· iFlow（npm 周下载 161）。
> **降级**：JetBrains AI / Junie —— 只能写 `.junie/AGENTS.md` 这类文件，MCP/skills 在 IDE 设置里。
> **边界观察**（声量够，本轮不做）：Cline 68k · Goose 54k · OpenHands 87k · Continue 36k · Crush 28k · Kilo 27k · Zed 90k。
> **通义灵码 = Qoder CN**；**Gemini CLI → 由 Antigravity CLI 取代**（2026-06-18）；`mcpuser` 是通用 MCP 落点，不是 harness。

每家一个完整 profile 见 [harness-profiles.md](./harness-profiles.md)。

## 目标 Tier B：L2 skills-only 长尾（低成本，一次覆盖几十家）

这是 `vercel-labs/skills`（README → Supported Agents，快照 **2026-09-11**；来源见文首「名单来源」）
的完整 skills 目标表（全局路径）。**直接采纳为固定配置，不逐次探测**；以后新增 agent 就加一行。
`magic skills sync` 把目录映射补全，一次性覆盖这些 agent：

```
aider-desk        ~/.aider-desk/skills/
amp/replit/universal  ~/.config/agents/skills/
antigravity       ~/.gemini/antigravity/skills/
antigravity-cli   ~/.gemini/antigravity-cli/skills/
astrbot           ~/.astrbot/data/skills/
autohand-code     ~/.autohand/skills/
augment           ~/.augment/skills/
bob               ~/.bob/skills/
claude-code       ~/.claude/skills/
openclaw          ~/.openclaw/skills/
cline/dexto/kimi-code-cli/loaf/sarvam-code/warp/zed  ~/.agents/skills/
codearts-agent    ~/.codeartsdoer/skills/
codebuddy         ~/.codebuddy/skills/
codemaker         ~/.codemaker/skills/
codestudio        ~/.codestudio/skills/
codex             ~/.codex/skills/
command-code      ~/.commandcode/skills/
continue          ~/.continue/skills/
cortex            ~/.snowflake/cortex/skills/
crush             ~/.config/crush/skills/
cursor            ~/.cursor/skills/
deepagents        ~/.deepagents/agent/skills/
devin             ~/.config/devin/skills/
droid             ~/.factory/skills/
firebender        ~/.firebender/skills/
forgecode         ~/.forge/skills/
fx                ~/.fx/skills/
gemini-cli        ~/.gemini/skills/
github-copilot    ~/.copilot/skills/
goose             ~/.config/goose/skills/
grok              ~/.grok/skills/
hermes-agent      ~/.hermes/skills/
inference-sh      ~/.inferencesh/skills/
jazz              ~/.jazz/skills/
junie             ~/.junie/skills/
iflow-cli         ~/.iflow/skills/
kilo              ~/.kilo/skills/
kimchi            ~/.config/kimchi/harness/skills/
kiro-cli          ~/.kiro/skills/
kode              ~/.kode/skills/
lingma            ~/.lingma/skills/
mcpjam            ~/.mcpjam/skills/
minimax-code      ~/.minimax/skills/
mistral-vibe      ~/.vibe/skills/
moxby             ~/.moxby/skills/
mux               ~/.mux/skills/
opencode          ~/.config/opencode/skills/
openhands         ~/.openhands/skills/
ona               ~/.ona/skills/
pi                ~/.pi/agent/skills/
posit-assistant   ~/.posit/assistant/skills/
qoder             ~/.qoder/skills/
qoder-cn          ~/.qoder-cn/skills/
qwen-code         ~/.qwen/skills/
reasonix          ~/.reasonix/skills/
rovodev           ~/.rovodev/skills/
roo               ~/.roo/skills/   ← 不做（2026-05 停运）
tabnine-cli       ~/.tabnine/agent/skills/
terramind         ~/.terramind/skills/
tinycloud         ~/.tinycloud/skills/
trae              ~/.trae/skills/
trae-cn           ~/.trae-cn/skills/
windsurf          ~/.codeium/windsurf/skills/   ← 不做（产品濒临停运）
zcode             ~/.zcode/skills/
zencoder/zenflow  ~/.zencoder/skills/
neovate           ~/.neovate/skills/
pochi             ~/.pochi/skills/
adal              ~/.adal/skills/
```

（`eve` / `promptscript` 是 project-only，无全局路径。）

## Tier 与能力支持（能力不对等）

**不追求每家能力完全一样。** 某 harness 支持某能力就投，不支持就**跳过：不报错、不造空文件**
（日志写 `[harness] hooks: n/a`）。「不支持」是**调研结论**，写进下面的矩阵，不是猜的。

符号：**✅ 已实现** · **○ 工具支持、Magic 未做** · **❓ 待调研** · **— 不支持 / 不适用**

| harness | skills | agents | mcp | models | custom | hooks | switch |
|---|---|---|---|---|---|---|---|
| claude | ✅ | ✅ | ○ | ✅ | ✅ | ○(20+) | ✅ |
| copilot | ✅ | ○ | ✅ | ○(BYOK) | ✅ | ○(7) | — |
| codex | ✅ | ○ | ○ | ✅ | ✅ | ○(12) | ✅ |
| cursor | ○ | ○ | ○ | — | ○ | ○(20+) | — |
| opencode | ✅ | ○ | ✅ | ✅ | ✅ | ○(plugins) | ✅ |
| antigravity-cli | ○ | ○ | ○ | ❓ | ○ | ○(5) | — |
| trae | ○ | ○ | ○ | ○(UI) | ○ | ❓ | — |
| qoder | ○ | ○ | ○ | ○ | ○ | ○(23) | — |
| workbuddy (= CodeBuddy) | ○ | ○ | ○ | ✅ | ○ | ○ | ✅ |
| kimi-code | ○ | ○ | ○ | ○(config.toml) | ○ | ○(20) | — |
| qwen-code | ○ | ○ | ○ | ○(settings.json) | ○ | ○(17) | ○(settings.json) |
| pi | ✅ | ✅ | ○ | ✅ | ✅ | ○(extensions) | ✅ |
| grok | ✅ | ✅ | ❓ | ✅ | ✅ | ✅ | ✅ |
| gemini（过渡） | ○ | — | ✅ | ❓ | ✅ | ○(11) | — |
| mcpuser | — | — | ✅ | — | — | — | — |

> `✅ 已实现` · `○ 工具支持、Magic 未做` · `❓ 待调研` · `— 不支持/不适用` · `● 只能写文件、走不了配置`


> 目标表，不是现状。当前代码 `adapter.Caps` 只有 `Skills/Agents/MCP/Models/Themes`，
> 缺 `hooks`/`custom` 显式表达，且部分值偏低（claude 的 mcp、gemini 的 skills）。
>
> **2026-09-11 调研补充**（来源见各行左栏对应工具的官方文档）：
> - **copilot**：支持 BYOK 自定义模型（2026-04）→ `models` 从 `—` 改 `○`。
> - **qoder**（Qoder CLI）：`~/.qoder/settings.json` 有 `mcpServers`；官方文档列了 Subagent / MCP / Skills / Plugins / **Hooks** / Commands / **Custom Models** → 能力很全。
> - **iflow-cli**：`~/.iflow/settings.json` 的 `apiKey` / `baseUrl` / `modelName` → `models` 和 `switch` 都能做；支持 skills / mcp。
> - **kimi-code**：`~/.kimi-code/mcp.json`（用户）+ `.kimi-code/mcp.json`（项目）→ `mcp` 可做；skills 有。
> - **workbuddy / CodeBuddy**：**共用 `~/.codebuddy/`** —— 用户级 `settings.json`（含 **hooks / mcp / plugins**）、`models.json`（自定义模型）；项目级 `.codebuddy/`。插件包含 Skills / Agents / Hooks / MCP / LSP。
> - **gemini**：⚠️ **Gemini CLI 已于 2026-06-18 被 Antigravity CLI 取代**（官方）；保留作过渡，
>   新目标指向 `antigravity-cli`（`agy`）。hooks 事件已查齐（11 个）。
> - **antigravity-cli**：配置 `~/.gemini/antigravity-cli/settings.json`；skills 全局
>   `~/.gemini/antigravity-cli/skills/`、工作区 `.agents/skills/`；MCP `~/.gemini/config/mcp_config.json`（`serverUrl`）；
>   hooks `hooks.json`（PreToolUse/PostToolUse/PreInvocation/PostInvocation/Stop）。
> - **trae**：skills 用户级 `~/.trae/skills/`（国内版 **`~/.trae-cn/skills`**，官方文档确认），项目级 `.trae/skills/`；**MCP 用户级文件 = `~/.trae/mcp.json`（国内版 `~/.trae-cn/mcp.json`）**。
>   另支持标准目录 **`.agents/skills/`**（设置里开关）；重名时 `.trae/skills/` 优先；禁用状态记在项目 `.trae/skill-config.json`。
> - **qoder**：hooks 事件已抄齐（Claude 同款形状，20+ 事件）。
> - **kilo**：有 MCP 与 CLI；**配置 `~/.config/kilo/config.json`**，自定义模型在 `provider.<name>.models`。
> - **lingma（通义灵码）**：**就是 Qoder CN**（阿里同一产品线）。skills `~/.lingma/skills/` + 支持行业标准 `.agents/skills/`；MCP 走 **IDE UI 配置，无独立配置文件**。
> - **miniMax-code**：官方推荐用 `npx skills add MiniMax-AI/cli -g` → skills 是 L2 层。
> - **roo**：2026-05 停运 → **不做**（不因 vercel 名单里有它而加回）。

### 调研来源（2026-09-11）

- Trae skills：https://docs.trae.ai/ide/skills · https://docs.trae.cn/ide_skills（中文正文含具体路径）
- Qoder CLI settings/MCP/hooks：https://docs.qoder.com/cli/settings-reference · /cli/mcp-servers · /cli/hooks
- iFlow settings：https://github.com/iflow-ai/iflow-cli/blob/main/docs_en/configuration/settings.md
- Kimi Code MCP：https://www.kimi.com/code/docs/en/kimi-code-cli/customization/mcp.html
- CodeBuddy MCP：https://www.codebuddy.ai/docs/cli/mcp
- MiniMax CLI：https://platform.minimax.io/docs/token-plan/minimax-cli
- Kilo CLI/config：https://kilo.ai/docs · https://github.com/Kilo-Org/kilocode/issues/3034
- Copilot BYOK：https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/use-byok-models
- Pi MCP 适配器：https://github.com/nicobailon/pi-mcp-adapter

### 各能力从哪来、投到哪

| 能力 | 库路径 | 投到 |
|---|---|---|
| `skills` | `packs/skills/<name>/` | 各 harness skills 目录 |
| `agents` | `packs/agents/<name>.md` | 各 harness agents 目录 |
| `mcp` | `packs/mcp/mcp.yaml` | 各 harness MCP 配置（见 mcp-design.md）|
| `models` | `providers.yaml` + `secrets/providers.json` | 各 harness provider 配置 |
| `custom` | `packs/harness/<id>/<rel>` | 按文件写回该 harness |
| `hooks` | `packs/harness/<id>/hooks/` | 各 harness hook 配置（见 hooks-design.md）|
| `switch` | `providers.yaml` 的 `active` | 各 harness 登录/provider 文件 |

### 不做

- **windsurf**：产品濒临停运 / 已被收购（2026），不做；不要因为第三方名单里有它而加回。
- **roo（Roo Code）**：2026-05 停运，不做；从 Tier A / 矩阵 / L2 均已移出。
- **Tabnine CLI / Rovo Dev / Junie / Posit Assistant / Neovate**：没人用，**不做**（不再调研）。
- 不支持的能力**不硬做**：比如某些 CLI 不允许第三方 provider 介入，`switch` 标 `—`。

### 待调研

**覆盖集内还有 `❓`**：JetBrains AI / Junie 的 skills·hooks（IDE 集成，面窄）；Trae / Comfy 的 hooks；
Antigravity 的 agents·models；Comate 的 skills·hooks·用户级路径；Cursor 的 agents 目录、models 配置。

**长尾（21 家）暂不做**，但配置路径已查好，记录在 `harness-profiles.md` 文末；以后照表加，不用重查。

（目前无。）

其余（不做 windsurf / roo、L2 表直接采纳 vercel 表、status 只列已装、无优先级）见 [decisions.md](./decisions.md)。
