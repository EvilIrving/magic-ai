# 产品使用定义（从使用角度）

这份文档不用架构词汇描述产品，只用一件事定义它：**我敲什么命令，它做了什么，最后
机器上哪些文件变成了什么样。** 读这份文档就能判断产品对不对。凡是「以后会怎样」的
都标了状态，不许当成已有能力。

状态标记：**✅ 已实现** · **🧪 规划中（未实现）**

产品决定见 [decisions.md](./decisions.md)。密钥口径见 [secrets-and-sync.md](./secrets-and-sync.md)。

---

## 心智模型：一个库，投到多个工具

```
~/.magic-ai/           ← 你唯一要维护的目录（真相源）
      │  magic apply
      ▼
~/.claude/  ~/.pi/  ~/.codex/  ~/.grok/  ~/.config/opencode/  ~/.codebuddy/
```

- 库里的**跨工具内容**（skill / MCP / 中转 API）：一份，apply 时译成各工具自己的格式。
- 库里的**某工具自定义**（Pi theme、`AGENTS.md`）：原样放回那个工具。
- 方向：你手改 → 库；`apply` 库 → 工具；`sync push` 库 → R2。

---

## 1. 装上以后第一次

```bash
magic init          # 在 ~/.magic-ai 建骨架（幂等）
magic adapters      # 看支持哪些 harness
magic status        # 看库状态 + 本机已装（探测到）的 harness
```

`magic adapters` ✅ 输出（Caps 以命令为准）：

```
ID               NAME                IMPLEMENTED  SKILLS  SUB-AGENT  MCP   HOOKS
pi               Pi Agent            true         true    true       true  true
grok             Grok                true         true    true       true  true
claude           Claude Code         true         true    true       true  true
codex            Codex               true         true    true       true  true
mcpuser          MCP User Config     true         false   false      true  false
opencode         OpenCode            true         true    true       true  true
gemini           Gemini CLI          true         false   false      true  true
copilot          GitHub Copilot CLI  true         true    true       true  true
workbuddy        WorkBuddy           true         false   false      true  false
trae             Trae                true         true    true       true  false
cursor           Cursor              true         true    false      true  true
antigravity-cli  Antigravity CLI     true         true    false      true  true
kimi-code        Kimi Code CLI       true         true    true       true  true
qoder            Qoder               true         true    false      true  true
qoder-cn         Qoder CN            true         true    false      true  true
cline            Cline               true         true    false      true  true
```

`magic adapters --json` 输出同样的字段。**不做 windsurf / roo**（D13）。
P2（droid/kiro/minimax-code/zcode/hermes/openclaw…）与长尾推迟（D20）；skills 路径已在表里，
`magic skills sync --target all` 可达。

`magic status` 产品行为（D12）：探测是全量的，输出**只给本机已装的**；未装的不列（可用 `magic adapters` 看全量）。

产品输出长这样：

```
Magic 1.0.0
配置库 (MAGIC_HOME): /Users/you/.magic-ai
  initialized: yes
user home: /Users/you

已安装 harness（本机探测到的；未装的不列）:
  claude     implemented   skills sub-agent models switch
  pi         implemented   skills sub-agent models switch
  gemini     apply-only    mcp
```

`magic init` 生成 ✅：

```
~/.magic-ai/
  README.md
  magic.yaml                # 哪些 harness 启用
  providers.yaml            # 中转 + 模型 pick（无密钥）
  secrets/                  # 密钥目录；key 落 secrets/providers.json（明文，见 §8）
  packs/skills/             # 跨工具 skill（全局真相源）
  packs/mcp/                # MCP 规范形（你添加 mcp.yaml）
  packs/agents/             # sub-agent
  packs/harness/<id>/       # 某工具自定义（Pi theme 等）
  registry/                 # models.dev 缓存
  backups/                  # apply 前自动备份（不同步）
```

`init` 建的是目录骨架。`packs/mcp/mcp.yaml`、`secrets/providers.json` 要你自己加内容，
不会预填密钥。

---

## 2. 加一个 provider（核心流程）

### 2a. 交互添加 `magic login` ✅

```
magic login deepseek
  API key: ****
  → 拉 https://api.deepseek.com/models
  → 列出模型让你勾（space 勾选 / a 全选 / enter 确认）
  → 确认后写 providers.yaml + secrets/providers.json
```

终端示意：

```
$ magic login deepseek
Provider: deepseek (key_env=DEEPSEEK_API_KEY)
API key for deepseek
  > •••••••••
Models for deepseek (live)
  3 selected · space toggle · a all · enter confirm · esc cancel
○ deepseek-chat
◉ deepseek-flash
◉ deepseek-v4-pro
…
Write to the config library?
provider : deepseek
key_env  : DEEPSEEK_API_KEY
models   : deepseek-flash, deepseek-v4-pro
```

回退（`/models` 取不到时，输出会说明用了哪种）：内置模板 `pick` → `registry/models.json` → 手动输入。

非交互（脚本 / CI）：

```
magic login deepseek --api-key sk-… --pick deepseek-flash,deepseek-v4-pro --yes
magic login                       # 不带参数：从内置模板列出 provider 让你选（需 TTY）
magic login deepseek --no-live    # 跳过 GET /models，直接用回退
```

已有同名 provider 时：`key_env` / `endpoints` / `models` 用户优先，`pick` 替换为本次勾选。
详见 [provider-onboarding.md](./provider-onboarding.md)；决定见 [decisions.md](./decisions.md) D17。

### 2b. 现在就能用的手动方式 ✅

内置模板有 25 家官方 provider 骨架（`key_env` / endpoints / 默认 pick）。中转不内置，写在用户的 `providers.yaml`。
用 `magic provider add` 把内置 provider 写进 `providers.yaml`（已存在的不覆盖，用户优先）；
`magic provider seed` 则把内置 seed profiles 里缺的补上：

```bash
magic provider add --list          # 看内置 provider id
magic provider add deepseek zai    # 写进 ~/.magic-ai/providers.yaml
```
      - {id: anthropic,         base_url: https://api.deepseek.com/anthropic, protocol: anthropic-messages}
      - {id: openai-chat,       base_url: https://api.deepseek.com/v1,        protocol: openai-chat-completions}
      - {id: openai-responses,  base_url: https://api.deepseek.com/v1,        protocol: openai-responses}
```

然后存 key ✅：

```bash
magic secrets set DEEPSEEK_API_KEY --value 'sk-…'
# 本地就是明文 secrets/providers.json，不需要口令
```

---

## 3. 看库里有什么

`magic catalog` ✅：

```
active: shu
PROVIDER  ENDPOINT          BASE_URL                           PROTOCOL                KEY_ENV           PICK
deepseek  anthropic         https://api.deepseek.com/anthropic  anthropic-messages      DEEPSEEK_API_KEY  deepseek-flash,deepseek-v4-pro
deepseek  openai-chat       https://api.deepseek.com/v1         openai-chat-completions DEEPSEEK_API_KEY  deepseek-flash,deepseek-v4-pro
shu       openai-responses  https://api.shu.cool/v1             openai-responses        SHU_API_KEY       gpt-6-astra,gpt-5.6-sol,…
```

`magic provider list` ✅ 除了库里的 provider，还列出每个 harness 的切换目标。

---

## 4. `magic copy` —— 把一个 provider 拷成可粘贴的文本

**什么时候用**：想手动把某个中转贴进别的工具，或临时看 key 解析对不对。

`magic copy deepseek` ✅ 输出（默认 redact key 中间）：

```
provider: deepseek
endpoint: anthropic
base_url: https://api.deepseek.com/anthropic
protocol: anthropic-messages
key_env: DEEPSEEK_API_KEY
api_key: sk-d…cdef
models:
  - deepseek-flash
  - deepseek-v4-pro
```

`magic copy deepseek --endpoint openai-chat --reveal-key` ✅ 换成另一端点和完整 key。
`--protocol anthropic-messages` 按协议选。

key 从本地明文 `secrets/providers.json` 取，**不读 process env，不需要口令**。
取不到就输出占位符：

```
api_key: <DEEPSEEK_API_KEY>
# note: secret missing — set with: magic secrets set DEEPSEEK_API_KEY --value …
```

---

## 5. `magic apply` —— 把库投到各工具（最重要）

**什么时候用**：库里改完（provider / skill / agent / MCP / 自定义文件），让各工具生效。

```bash
magic apply            # 全部启用且已探测到的 harness
magic apply --dry-run  # 只看会做什么，不落盘
magic apply --only pi  # 只投一个
```

### 终端输出 ✅（形状；具体路径随本机变）

```
backup → /Users/you/.magic-ai/backups/20260913T184000Z
[claude] no library mirror at /Users/you/.magic-ai/packs/harness/claude
[pi] restore AGENTS.md
[pi] restore settings.json

## skills projection
link /Users/you/.magic-ai/packs/skills/anysearch → ~/.claude/skills/anysearch
link /Users/you/.magic-ai/packs/skills/anysearch → ~/.pi/agent/skills/anysearch

## sub-agent projection
project packs/agents/reviewer.md → ~/.claude/agents/reviewer.md

## providers projection
[claude] providers: upsert /Users/you/.claude/settings.json (provider=shu endpoint=anthropic protocol=anthropic-messages)
[pi] providers: merge /Users/you/.pi/agent/models.json provider=magic-shu (endpoint=openai-responses protocol=openai-responses api=openai-responses apiKey=sk-…)
[opencode] providers: OpenCode: provider magic-shu baseURL=https://api.shu.cool/v1 model=magic-shu/gpt-6-astra

## mcp projection
[claude] mcp: merged 2 server(s) → /Users/you/.claude.json
[codex] mcp: merged 2 server(s) → /Users/you/.codex/config.toml
[pi] mcp: n/a linear — Pi remote MCP needs the third-party pi-mcp-adapter
[pi] mcp: merged 1 server(s) → /Users/you/.pi/agent/mcp.json
```

全局 skill 源是 **`$MAGIC_HOME/packs/skills`**，不是 `~/.agents/skills`。
`~/.agents/skills` 只出现在项目级 skill 源（`<repo>/.agents/skills`）和少数 harness 的共用目录。

apply 分两层：

| 层 | 从哪 | 怎么写 |
|---|---|---|
| 共有原数据 | `providers.yaml` + secrets、`packs/skills`、`packs/mcp`、`packs/agents` | 按各 harness 格式投影 |
| harness 定制 | `packs/harness/<id>/` | 只写回该工具 |

写回一律**字段级合并**（D5）：结构化文件按键、一文件一项按项，不整文件覆盖，不删用户自己的键。
hooks 走 `packs/harness/<id>/hooks/…`（B 方案，字段级合并，D19）。
MCP 从 `packs/mcp/mcp.yaml` 投影到各 harness 原生格式（D18）——详见 §11。

### 生成的文件长这样（以 active=shu 为例）

**Claude Code** → `~/.claude/settings.json`（只 upsert `env` 里 Magic 管的键，其它键保留）✅

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "https://api.shu.cool",
    "ANTHROPIC_AUTH_TOKEN": "sk-…",
    "ANTHROPIC_API_KEY": "sk-…",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "gpt-5.6-luna",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "gpt-5.6-terra",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "gpt-5.6-sol",
    "ANTHROPIC_MODEL": "gpt-5.6-sol",
    "CLAUDE_CODE_SUBAGENT_MODEL": "gpt-5.6-luna"
  }
}
```

`ANTHROPIC_AUTH_TOKEN` 是主键，`ANTHROPIC_API_KEY` 同值兜底。角色没显式配时按
`haiku` / 第一条 pick 推导。

**Pi** → `~/.pi/agent/models.json`（merge 进 `providers`，其它 provider 保留）✅
另写 sidecar `~/.pi/agent/models.magic.json`。

```json
{
  "providers": {
    "magic-shu": {
      "baseUrl": "https://api.shu.cool/v1",
      "api": "openai-responses",
      "apiKey": "sk-…",
      "models": [
        { "id": "gpt-5.6-sol", "name": "GPT-5.6 Sol", "reasoning": true,
          "contextWindow": 1050000, "maxTokens": 128000,
          "cost": { "input": 5, "output": 30, "cacheRead": 0.5, "cacheWrite": 6.25 },
          "thinkingLevelMap": { "off": "none", "low": "low", "high": "high", "max": "max" } }
      ]
    }
  }
}
```

`apiKey` 写**真值**（从 `secrets/providers.json` 解析），Pi 拿来直接用，不用 export 环境变量，
不走 `$KEY_ENV`。Claude / OpenCode / WorkBuddy 同一规则。

`models[]` 的 cost / context / thinkingLevelMap 来自 `registry/models.json`（先
`magic registry sync --source models.dev`），缺 meta 就只写 `id` + `name`。

**OpenCode** → `~/.config/opencode/opencode.json`（有 `.jsonc` 就写 `.jsonc`）✅

```json
{
  "$schema": "https://opencode.ai/config.json",
  "provider": {
    "magic-shu": {
      "options": { "baseURL": "https://api.shu.cool/v1", "apiKey": "sk-…" },
      "whitelist": ["gpt-6-astra", "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"]
    }
  },
  "model": "magic-shu/gpt-6-astra"
}
```

`provider.magic-*` 是 Magic 管的；非 `magic-` 前缀不动。切 official 只删 `magic-*`。

**Codex** → `~/.codex/config.toml` + `~/.codex/models.json` ✅
走 `magic provider use`（apply 末尾不投 Codex）。`models.json` 是 Codex 原生形状
（`slug` / `display_name` / `context_window` / `supported_reasoning_levels`），只在第三方时写。
key 用 Codex 原生 `env_key`，不写真值 token。

**Grok** → `~/.grok/config.toml`（Magic 管理的 catalog 块）+ `~/.grok/provider-mode` ✅
同样走 `magic provider use`。`provider-mode` 是单行：`official` 或 provider 名。

**WorkBuddy / CodeBuddy** → `~/.codebuddy/models.json` ✅
同一产品，共用 `~/.codebuddy/`。Magic 管的 model id 带 `magic-` 前缀，`apiKey` 写**真值**。
切 official 只删这些行。

**某工具自定义**（Pi `extensions/`、`themes/`、Codex `memories/` 等）→
`packs/harness/<id>/` 按文件写回该工具。收外部文件进库：
`magic add path <file> [--as <harness>/<rel>]` 🧪（D7；复制，不动原文件）。

### apply 的规则

- **先备份**：每次非 dry-run 在 `backups/<utc>/` 留一份改动前的文件（例如
  `backups/20260913T184000Z/pi/agent/models.json`）。`backups/` 不同步。
- **只碰自己管的键**：Claude 只 upsert Magic 管的 `ANTHROPIC_*` / `CLAUDE_CODE_*`；
  OpenCode / WorkBuddy / Pi 只碰 `magic-*`；Codex / Grok 只碰 Magic-managed 块。
  用户自己的配置不动。
- **缺密钥不失败但有方向**：Claude / Pi / OpenCode / WorkBuddy 都需要真值，缺了就 skip 并提示
  `magic secrets set <KEY_ENV> --value …`。无密钥的官方路径不需要。
- **未安装的 harness 默认跳过**（除非 `--only` 指定）：镜像树、`packs/skills`、`packs/agents`、共享
  `packs/AGENTS.md`、provider、MCP 六处写入都过 `Detect` 这道门，不会给没装的工具创建
  `~/.factory/`、`~/.trae/`、`~/.qwen/` 这类目录（D26）。`magic apply --only <id>` 是显式预装：
  先写该工具的配置，再打开工具；`magic plan` 对未安装的 adapter 显示 skipped（原因「未安装」）。
- **能力不对等**（D10）：某家没有的面（skills / MCP / models）跳过，日志 `n/a`，不造空文件。

---

## 6. `magic provider` —— 日常切换 official ↔ 中转

`magic provider use` 是 Grok / Codex 的日常入口；apply 末尾投 Claude / Pi / OpenCode / WorkBuddy。

```bash
magic provider use shu          # 所有适用的 harness 切到 shu
magic provider use company      # active 别名（= providers.yaml 的 active）
magic provider use official     # 切回一方订阅（claude/codex/grok/workbuddy）
magic provider use official --harness grok
magic provider use none         # 清掉 Magic 写的条目（Pi / OpenCode 用这个）
magic provider status           # 看每个 harness 当前是什么
magic provider test company     # HTTP 打一下 {base}/models，验证 key 通不通
magic provider seed             # 把内置 seed profiles 合并进 providers.yaml（缺的才加）
magic provider add deepseek     # 把内置模板 provider 写进 providers.yaml（用户优先）
magic guide                     # 看一遍上手顺序
```

**`official` 只在一方有原生订阅/登录的 harness 上成立**：Claude（Anthropic）、Codex（OpenAI/ChatGPT）、
Grok（xAI）、WorkBuddy（CodeBuddy 自带模型）。**Pi 与 OpenCode 是多 provider 聚合器，没有单一 official**
（Pi：`/login` 订阅或 API key + `auth.json`；OpenCode：预加载 75+ provider + `auth.json`，Zen 可选）——
对这两家用 **`none`**；传 `official` 会报错并提示用 `none`。`none`/`off`/`clear` 对所有 harness 有效。
（依据各自官方文档，2026-09-13。）

`magic provider use shu --harness grok` ✅ 输出：

```
Grok configured for provider "shu" (endpoint=openai-responses api_backend=…).
Restart Grok for the change to take effect.
```

`magic provider test deepseek` ✅ 成功：`Gateway reachable and accepted the API key (HTTP 200): …`；
失败会明确 401/403（key 不对）还是连不上。

---

## 7. Skill / sub-agent

**全局**真相源是 `$MAGIC_HOME/packs/skills`（**不是** `~/.agents/skills`）：

```bash
magic skills list
magic skills sync                         # 全局源 → DefaultAgents（有 adapter 的 8 家）
magic skills sync --target all            # → vercel-labs/skills 全量 skills 路径表（~70 家）
magic skills sync --target qoder,kimi-code-cli   # 指定长尾 agent
magic skills sync --scope project         # 源 = <repo>/.agents/skills
magic skills sync --scope project --include-global   # 项目为主，再继承全局（同名以项目为准）
magic skills status                       # 每个 agent：linked / missing / conflict / stale
magic skills doctor
magic skills import PATH                  # 把一个本地 skill 目录收进源（默认链接）
magic skills adopt --from AGENT           # 把某 agent 目录里没管的 skill 收编进源
```

`--include-global` 只对 **project scope 的 sync** 有意义，不是 `list` 的开关。

**共享层**：`magic skills sync --shared` **只写**标准目录 `~/.agents/skills` 这一份，不碰任何工具自己的主路径
（那是默认 `magic skills sync` 的活）。技能路径表里把它当**主路径**的有 8 个 id：
`cline, dexto, kimi-code, kimi-code-cli, loaf, sarvam-code, warp, zed`；其中 Magic 有 adapter 的只有
`cline / kimi-code / zed`，所以一份共享层等于让另外 5 家（`dexto / kimi-code-cli / loaf / sarvam-code / warp`）
直接生效，不必为它们各写 adapter。

对「有自己的主路径、只是兼容读共享层」的工具（pi / grok / codex / cursor / trae），共享层优先级更低——grok 读
`~/.claude/skills` 是 Lowest 且可配置，用户主路径的值会盖过共享层。共享层里非 Magic 创建的条目保持原样并
逐个报告（`shared: keep …`）：不删、不覆盖，`--force` 也不作用于它。`--shared` 与 `--scope project`、
`--target` 都互斥：后两者说的都是「某个工具自己的主路径」，和「只写标准目录」不是一件事，给了就报错，
不静默忽略。

`magic skills status` 的每行多两列：`shared +N`（另有 N 个**工具**指向同一目录）/ `shared -`，以及
`primary yes|no`（该目录是不是这个工具自己的主路径；`cline / zed / kimi-code / pi / grok / codex` 为 yes，
`workbuddy` 与表里没有 adapter 的 id 为 no）。别名不算「另一个工具」：`claude-code`（= `claude`）、
`kimi-code-cli`（= `kimi-code`）、`codebuddy`（= `workbuddy`，D15）在 `shared_with` 里都归一到规范 id 且只出现一次，
所以 `claude` 是 `shared -`，`cline` 是 `shared +6`（含 `kimi-code`，不含 `kimi-code-cli`）。
表里没有记录、但 Magic 有 adapter 的 harness（`hermes`、`mimocode`）显示 `no upstream entry`，目录取其 adapter
声明的 skills 路径，不算错误、不影响退出码。`compat reads:` 行列该工具另外还会读的目录——**文档声明，只用于
显示，不参与任何写入决策**。`magic skills status --json` 带同样的 `shared_with` / `primary` / `compat_reads`，
这类 agent 另有 `no_upstream_entry: true`。

行为约定 ✅：**永不删除真实目录**；冲突 = 真目录或非受管链接，报出来不覆盖。
项目级 skill 用 `magic skills sync --scope project`，**不是** `magic apply`。

当前 skills 目标（默认，按用量排序）：`codex, claude, cursor, pi, grok, antigravity-cli, opencode, kimi-code, copilot, workbuddy, trae, qoder, qoder-cn, cline`（均有产品 adapter）。
L2 长尾表（vercel-labs/skills，~70 家）已锁定为产品配置（`internal/skills/paths.go`）：
`--target all` 全部覆盖，`--target <id>` 任选；默认不写长尾，避免给未安装工具造空目录。
（`windsurf` / `roo` 已剔除，D13。）

sub-agent：把 `packs/agents/<name>.md` 放进库，`magic apply` 投到各 Caps.Agents 目录；
`magic agents` ✅ 列出库里的和每个 adapter 的目标路径。

---

## 8. 密钥：本地明文，`.enc` 是云端状态

| 位置 | 形态 |
|---|---|
| **本地** `secrets/providers.json`（0600） | **明文**。无 `.enc`，无需口令 |
| **云端** | 含密钥的文件 → `vault/…/<file>.enc`（密文）。`.enc` **只表示云端状态** |
| 其余文件 | 云端也明文，不加 `.enc` |

```bash
magic secrets set DEEPSEEK_API_KEY --value 'sk-…'
magic secrets list                 # 只列 key 名，不打印值
```

`key_env` 是逻辑索引，**不是** process env 变量名。`copy` / `apply` / `provider` 直接读
本地明文，不读 env、不需口令。加密只发生在 `sync push`；`sync pull` 解密 `.enc` 后落回
本地明文（无 `.enc`）。口令仅 push/pull 时需要，不进库、不同步。

投影：Pi / WorkBuddy / OpenCode / Claude 写 **apiKey / AUTH_TOKEN 真值**。Codex 用原生 `env_key`。

完整口径见 [secrets-and-sync.md](./secrets-and-sync.md)（D2 / D3）。

---

## 9. 多机备份：`magic sync`

```bash
export MAGIC_PASSPHRASE=…          # 口令先有，pull 才能解 .enc
export MAGIC_R2_BUCKET=magic-ai
export MAGIC_R2_ACCESS_KEY_ID=…
export MAGIC_R2_SECRET_ACCESS_KEY=…
export MAGIC_R2_ENDPOINT=https://<ACCOUNT_ID>.r2.cloudflarestorage.com

magic sync status   # 看 R2 上有多少对象
magic sync push     # 含密钥文件 → 加密成 .enc 上传；其余明文。跳过 backups/
magic sync pull     # 下载 .enc → 解密 → 落回本地明文（无 .enc）
```

新机器：**先**设口令和 `MAGIC_R2_*`，再 `magic sync pull`，再 `magic apply`。
git 私有库是备选通道。冲突检测 🧪。

---

## 10. 模型注册表（可选，给 apply 补元数据）

```bash
magic registry sync --source models.dev   # 拉 models.dev → registry/models.json
magic registry search kimi
magic registry show kimi-k3
magic catalog pick deepseek deepseek-v4-pro   # 追加进 pick，并从 registry 补 models_meta
```

apply / `provider use` 会用 `registry/models.json` 给 pick 补 context / cost / thinkingLevelMap；
缺了就写默认值，**不失败**。

---

## 11. MCP 与 hooks 投影 ✅

### MCP —— `packs/mcp/mcp.yaml` → 各家原生文件

```yaml
# ~/.magic-ai/packs/mcp/mcp.yaml
mcp:
  context7:
    transport: stdio
    command: npx
    args: ["-y", "@upstash/context7-mcp@latest"]
    env:
      API_KEY: "sk-…"
  linear:
    transport: http
    url: https://mcp.linear.app/mcp
    headers:
      Authorization: "Bearer …"
```

```bash
magic mcp init      # 写一份示例（不覆盖已有）
magic mcp list      # 库里的 server
magic mcp status    # 各 harness 是否已写入
magic apply         # 翻译并写进各 harness 原生 MCP 配置
```

落点（D18）：

| harness | 文件 | 顶层键 | remote |
|---|---|---|---|
| claude | `~/.claude.json` | `mcpServers` | `{type:"http"\|"sse",url,headers}` |
| codex / grok | `*.toml` | `[mcp_servers.*]` | `url`（codex 不支持 sse；grok 可带 `headers`） |
| opencode | `opencode.json` | `mcp` | `{type:"remote",url,headers}`；local 的 `command` 是数组 |
| pi | `~/.pi/agent/mcp.json` | `mcpServers` | 不支持（需第三方 adapter）→ `n/a` |
| gemini | `settings.json` | `mcpServers` | http→`httpUrl`，sse→`url` |
| copilot | `~/.copilot/mcp-config.json` | `mcpServers` | `{type:"local"\|"http"\|"sse"}` |
| workbuddy | `~/.codebuddy/settings.json` | `mcpServers` | `{url,headers}` |
| trae | `~/.trae/mcp.json` | `mcpServers` | `{url,headers}` |
| mcpuser | `~/.config/mcp/mcp.json` | `mcpServers` | `{url,headers}` |
| cursor | `~/.cursor/mcp.json` | `mcpServers` | `{url,headers}`（无 type） |
| antigravity-cli | `~/.gemini/config/mcp_config.json` | `mcpServers` | `{serverUrl,headers}` |
| kimi-code | `~/.kimi-code/mcp.json` | `mcpServers` | http `{url,headers}`；sse `{transport:"sse",url}` |
| qoder / qoder-cn | `~/.qoder/settings.json` / `~/.qoder-cn/settings.json` | `mcpServers` | `{type:"http"\|"sse",url,headers}` |
| cline | `~/.cline/mcp.json` | `mcpServers` | `{type:"streamableHttp"\|"sse",url,headers}` |

规则：密钥写**明文真值**；Magic 管理的条目 = 名字在 `packs/mcp/mcp.yaml` 里的（**无前缀**）；
用户其它 server **永不删**；不支持就记 `n/a`（如 Pi 的 remote）。

### hooks —— `packs/harness/<id>/hooks/…`

hooks 不做跨家翻译（B 方案，D19）：原样放到 `packs/harness/<id>/` 下，`apply` 按字段级合并写回。

```
packs/harness/claude/settings.json     # {"hooks":{...}} 合并进 ~/.claude/settings.json 的 hooks
packs/harness/codex/hooks.json         # 合并进 ~/.codex/hooks.json
packs/harness/grok/hooks/*.json        # → ~/.grok/hooks/*.json（一文件一项）
packs/harness/copilot/hooks/*.json     # → ~/.copilot/hooks/*.json
packs/harness/pi/extensions/*.ts       # 代码式扩展，原样写回
packs/harness/opencode/plugins/*.ts    # 代码式插件，原样写回
```

路径值**原样写**（不折 `~`）；Codex hooks 不写 trust 元数据。

---

## 12. `magic plan` —— 先看 apply 会写什么 ✅

**什么时候用**：动手之前想知道 apply 会改哪些文件、改哪个键；或者把「这次写入清单」打进 CI 日志。

```bash
magic plan             # 全部启用（magic.yaml 的 enabled）的 harness
magic plan --only pi   # 只算一家
magic plan --json      # 机器可读（字段见 docs/plan.md）
```

**只读**：不写目标文件、不写备份、不动 `magic.yaml`。

### 终端输出 ✅（临时库 + 临时 home 实测）

```
配置库: /Users/you/.magic-ai
目标主目录: /Users/you

ADAPTER  ACTION     REL                MODE  BACKUP  说明
claude   update     _claude.json       0600  yes     claude partial 深合并 → mcp 投影
                                                     add mcpServers.docs.args
                                                     add mcpServers.docs.command
                                                     add mcpServers.linear.args
                                                     add mcpServers.linear.command
                                                     keep projects
claude   update     settings.json      0600  yes     镜像 JSON 合并 → provider 投影 env
                                                     add env.ANTHROPIC_API_KEY
                                                     add env.ANTHROPIC_AUTH_TOKEN
                                                     add env.ANTHROPIC_BASE_URL
                                                     add env.ANTHROPIC_MODEL
                                                     add permissions.allow
                                                     keep statusLine
codex    update     config.toml        0600  yes     镜像 TOML 合并 → mcp 投影
                                                     行 +12 -1
grok     -          -                  -     -       配置库里没有镜像目录 /Users/you/.magic-ai/packs/harness/grok；apply 不投影该 adapter 的 provider（只有 magic provider use 会写）
pi       unchanged  AGENTS.md          0644  -       镜像整文件覆盖（结果与磁盘一致）
pi       create     mcp.json           0600  -       mcp 投影
                                                     add mcpServers.docs.args
                                                     add mcpServers.docs.command
                                                     add mcpServers.docs.disabled
                                                     add mcpServers.linear.args
                                                     add mcpServers.linear.command
                                                     add mcpServers.linear.disabled
pi       create     models.json        0600  -       provider 投影 models.json
                                                     add providers.magic-shu.api
                                                     add providers.magic-shu.apiKey
                                                     add providers.magic-shu.baseUrl
                                                     add providers.magic-shu.models
pi       create     models.magic.json  0600  -       provider 投影 models.magic.json
                                                     add providers.magic-shu.api
                                                     add providers.magic-shu.apiKey
                                                     add providers.magic-shu.baseUrl
                                                     add providers.magic-shu.models

7 个文件：3 新建，3 更新，0 删除，1 不变
```

`ACTION` 四取一（每个文件一条）：

| 值 | 含义 |
|---|---|
| `create` | 目标还不存在，apply 会新建 |
| `update` | 内容会变；`BACKUP yes` = apply 先把它复制到 `backups/<时间戳>/` |
| `delete` | apply 会删掉这个文件（provider 投影用删除表达「清空」）。今天 `magic plan` 里看不到它：唯一的删除来自 codex 官方模式的 provider 投影，而 `apply` 不投影 codex / grok（那条路是 `magic provider use`），所以这个值目前出现在 plan 的 schema 与 `verify` 的 `delete` 漂移里 |
| `unchanged` | 内容与权限都和库一致，apply 不写盘、不动 mtime |

`说明` 列按顺序列出写这个文件的层（`镜像 → provider → mcp`）。JSON 配置给键级差异：
`add`（库有、目标没有）、`overwrite`（值会变）、`keep`（库没定义、合并后保留你的值）；
`config.toml` 给行数（`行 +12 -1`），不做字符级 diff。最后一行是四类计数汇总。

### 库内没有镜像目录时也会报投影

codex 在多数配置库里没有 `packs/harness/codex`，但 MCP 投影照样写 `~/.codex/config.toml`，plan 必须列出这一条：

```
ADAPTER  ACTION  REL          MODE  BACKUP  说明
codex    update  config.toml  0600  yes     mcp 投影
                                            行 +5 -0

1 个文件：0 新建，1 更新，0 删除，0 不变
```

把 `packs/mcp/mcp.yaml` 也去掉，才是「什么都不写」，这时说明列会同时给出两个原因：

```
ADAPTER  ACTION  REL  MODE  BACKUP  说明
codex    -       -    -     -       配置库里没有镜像目录 /Users/you/.magic-ai/packs/harness/codex；apply 不投影该 adapter 的 provider（只有 magic provider use 会写）

0 个文件：0 新建，0 更新，0 删除，0 不变
```

两句缺一不可：只看第一句会以为「往库里放一个 `packs/harness/codex` 就能让 apply 管 codex」——
而 apply 根本不投影 codex / grok 的 provider，那条路要 `magic provider use` 才走得到。

### 生成的文件

无。plan 只读；它给出的 `REL`、`MODE`、`BACKUP` 就是 apply 接下来会用到的值。

---

## 13. `magic verify` —— 检查有没有漂移 ✅

**什么时候用**：apply 之后确认真的落地了；放进 CI 守住「配置库 = 本机配置」。

```bash
magic verify                # 全部启用（magic.yaml 的 enabled）的 harness
magic verify --only claude
magic verify --json
```

**只读**：不写目标文件，也不修漂移（要修是 `magic apply`）。

### 终端输出 ✅：有漂移时退出码 1

```
配置库: /Users/you/.magic-ai
目标主目录: /Users/you

ADAPTER  状态      REL            路径
claude   differ  settings.json  /Users/you/.claude/settings.json

检查 8 个文件，1 处漂移，0 个 adapter 被跳过
Error: 发现 1 处漂移（运行 magic apply 修正，magic plan 查看细节）
```

`状态` 三取一：`missing`（apply 会新建）、`differ`（内容不同）、`delete`（apply 会删除）。
`--json` 给同样的内容：`checked`、`drift[]`、`skipped[]`。没有漂移时退出码 0：

```
ADAPTER  状态  REL  路径

检查 8 个文件，0 处漂移，0 个 adapter 被跳过
```

### 空检查与被跳过的 adapter 也返回非零

verify 只有在真的检查过文件时才敢说「一致」。一个都没查时退出码 1，并列出被跳过的 adapter 及原因：

```
ADAPTER  状态  REL  路径
跳过：codex: 配置库里没有镜像目录 /Users/you/.magic-ai/packs/harness/codex；apply 不投影该 adapter 的 provider（只有 magic provider use 会写）

检查 0 个文件，0 处漂移，1 个 adapter 被跳过
Error: 没有检查任何文件：所有 adapter 都被跳过，verify 不能据此判定一致
```

有漂移又有跳过时先报漂移；只有跳过时退出码 1，并说明「N 个 adapter 被跳过，verify 不能判定它们的状态」。

### 不覆盖的投影如实说明

verify 只覆盖 plan 能预测的四层：镜像、claude partial（`~/.claude.json`）、provider 投影
（claude / pi / opencode / workbuddy）、MCP 投影。下面这些**不在**它的检查范围内，它也不会
声称它们一致：

- skills 与 sub-agents 的目录投影 —— 看 `magic skills status` / `magic agents`；
- codex / grok 的 provider 投影 —— 只有 `magic provider use codex|grok` 会写，不在 apply 的写集里。

### 生成的文件

无。

---

## 14. `magic rollback` —— 回到某次 apply 之前 ✅

**什么时候用**：apply 之后发现不对，想整体退回上一次的状态。

```bash
magic rollback --list                       # 有哪些备份
magic rollback                              # 回到最新一次 apply 之前
magic rollback --to 20261113T194845Z --adapter pi
magic rollback --dry-run                    # 只看会动哪些文件
magic rollback --force
```

**输入**：`backups/<时间戳>/manifest.json`（apply 每次写入时记的清单：目标、动作、备份副本、
`before` / `after` 指纹）+ 同目录下的副本文件。

### 终端输出 ✅：默认不覆盖你手改过的文件

apply 之后又手改过的目标，内容指纹既不是 apply 写的（`after`）也不是 apply 换掉的（`before`），
默认跳过并逐条给原因，其余按清单还原：

```
ADAPTER  动作  REL                          路径                                    字节
pi       删除  .pi/agent/models.json        /Users/you/.pi/agent/models.json        -
pi       删除  .pi/agent/models.magic.json  /Users/you/.pi/agent/models.magic.json  -
pi       删除  mcp.json                     /Users/you/.pi/agent/mcp.json           -
跳过：pi/AGENTS.md: 目标在 apply 之后被改动，跳过（加 --force 覆盖）

跳过 1 个（见上），加 --force 强制按 manifest 执行

恢复 0 个文件，删除 3 个（备份 20261113T194845Z）
```

`--force` 才按清单执行（这一条是 `update`，所以是「恢复」，字节数是回填副本的大小）：

```
ADAPTER  动作  REL                          路径                                    字节
pi       删除  .pi/agent/models.json        /Users/you/.pi/agent/models.json        -
pi       删除  .pi/agent/models.magic.json  /Users/you/.pi/agent/models.magic.json  -
pi       恢复  AGENTS.md                    /Users/you/.pi/agent/AGENTS.md          18
pi       删除  mcp.json                     /Users/you/.pi/agent/mcp.json           -

恢复 1 个文件，删除 3 个（备份 20261113T194845Z）
```

### 路径越界一律拒绝，`--force` 也不执行

清单里的目标在 user home 之外（或备份路径逃出备份集）时，条目归「拒绝」，与可以用 `--force` 的
「跳过」分开报：

```
ADAPTER  动作  REL        路径                            字节
pi       恢复  AGENTS.md  /Users/you/.pi/agent/AGENTS.md  15
拒绝：pi/escape.json: 目标 /etc/magic-demo-escape.json 不在 user home /Users/you 之下，拒绝写入

拒绝 1 个（见上），--force 也不执行

恢复 1 个文件，删除 0 个（备份 20990101T000000Z）
```

这条校验守的是文件系统边界，不是内容，所以 `--force` 不解锁。

### `--list` 区分 apply 备份与 provider 备份

```
时间戳               文件数  ADAPTERS
20261113T194828Z  7    claude, codex, grok, pi
provider 备份，不参与回滚：claude
```

只有 `backups/<时间戳>/`（带 `manifest.json`）能回滚；`backups/<harness>/` 是 provider 层
自己的备份目录，列出来但标明不参与回滚。没有 `manifest.json` 的旧备份按老办法只恢复文件，
并明确标注「旧备份，只恢复文件不删除新建」。

### 生成的文件

按清单还原目标：`update` 条目把 apply 之前的副本放回去，`create` 条目删除 apply 新建的文件。
「字节」列是写回/删除的大小（删除显示 `-`）。

---

## 命令速查（使用视角）

| 我想…… | 命令 | 状态 |
|---|---|---|
| 初始化配置库 | `magic init` | ✅ |
| 看能投哪些工具 | `magic adapters` | ✅ |
| 看本机已装哪些 | `magic status` | ✅（只列已装） |
| 上手顺序 | `magic guide` | ✅ |
| 加 provider 并选模型 | `magic login <provider>` | ✅ |
| 手动加 provider | 编辑 `providers.yaml` + `magic secrets set` | ✅ |
| 内置 provider 写进库 | `magic provider add` | ✅ |
| 看库里有啥 | `magic catalog` | ✅ |
| 把 provider 拷成文本 | `magic copy <provider>` | ✅ |
| 让各工具生效 | `magic apply` | ✅ |
| 动手前看会写什么 | `magic plan` | ✅（只读） |
| 检查有没有漂移 | `magic verify` | ✅（只读，漂移退出码 1） |
| 退回某次 apply 之前 | `magic rollback` | ✅（`--list` / `--force`） |
| 把 MCP / hooks 投进各工具 | `magic apply`（`packs/mcp` / `packs/harness/<id>/hooks`） | ✅ |
| 看 MCP 投影状态 | `magic mcp list\|status` | ✅ |
| 收一个外部文件进库 | `magic add path <file>` | ✅ |
| 切换 official / 中转 | `magic provider use` | ✅ |
| 验证 key 通不通 | `magic provider test` | ✅ |
| 同步 skill | `magic skills sync` | ✅ |
| 存/列密钥 | `magic secrets set/list` | ✅ |
| 换电脑 | `magic sync push/pull` | ✅ |
| 补模型元数据 | `magic registry sync` + `catalog pick` | ✅ |

---

## 还没定的

已锁定的决定见 [decisions.md](./decisions.md)；未决项见 [open-questions.md](./open-questions.md)。
文档已定、代码未跟的（C 表）不要当成「产品还没定」。
