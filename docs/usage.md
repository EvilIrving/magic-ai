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

`magic adapters` ✅ 输出（Caps 以命令为准；Claude / Pi / Codex / Grok 的 MCP 列是 false ——
MCP 投影是另一条能力，见 §5）：

```
ID         NAME                  IMPLEMENTED  SKILLS  SUB-AGENT  MCP
claude     Claude Code           true         true    true    false
codex      Codex                 true         true    true    false
grok       Grok                  true         true    true    false
pi         Pi Agent              true         true    true    false
opencode   OpenCode              true         true    true    true
workbuddy  WorkBuddy             true         false   false   false
gemini     Gemini CLI            true         false   false   true
copilot    GitHub Copilot CLI    true         true    true    true
trae       Trae                  true         true    true    true
mcpuser    MCP User Config       true         false   false   true
```

Skills 同步目标另含 **cursor**（无产品 adapter）。**不做 windsurf / roo。**

`magic status` 产品行为（D12）：探测是全量的，输出**只给本机已装的**。未装的不列。
加 `--all` 才列出全部。🧪（命令已有，目前仍列出全部 adapter，含未装。）

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

### 2a. 你想要的用法 🧪

```
magic login deepseek
  API key: ****
  → 拉 https://api.deepseek.com/models
  → 列出模型让你勾
  → 写入 providers.yaml + secrets/providers.json
```

终端示意（未实现）：

```
$ magic login deepseek
Provider: DeepSeek (内置模板: key_env=DEEPSEEK_API_KEY, 3 endpoints)
Enter DEEPSEEK_API_KEY: ********
GET https://api.deepseek.com/models ... 200 (2 models)
? 选择要用的模型 (space 勾选, enter 确认)
  [x] deepseek-flash
  [x] deepseek-v4-pro
写入 providers.yaml: providers.deepseek.pick = [deepseek-flash, deepseek-v4-pro]
写入密钥库: secrets/providers.json → DEEPSEEK_API_KEY
提示: 已存在的其它 provider 保持不变
```

非交互（脚本 / CI）同样 🧪：

```
magic login deepseek --api-key sk-… --pick deepseek-flash,deepseek-v4-pro
magic login                      # 不带参数：从内置模板列出 provider 让你选
```

`/models` 取不到时必须说明用了哪种回退（内置模板默认 pick 等）。详见
[provider-onboarding.md](./provider-onboarding.md)；未定点见 [open-questions.md](./open-questions.md) A3。

### 2b. 现在就能用的手动方式 ✅

内置模板有 25 家官方 provider 骨架（`key_env` / endpoints / 默认 pick）。中转不内置，写在用户的 `providers.yaml`。
**合并进你的 `providers.yaml` 还没接上**（C4）。现在手写，或 `magic provider seed`
把 seed profiles 里缺的补上：

```yaml
# ~/.magic-ai/providers.yaml
active: deepseek
providers:
  deepseek:
    key_env: DEEPSEEK_API_KEY
    pick: [deepseek-flash, deepseek-v4-pro]
    endpoints:
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
```

全局 skill 源是 **`$MAGIC_HOME/packs/skills`**，不是 `~/.agents/skills`。
`~/.agents/skills` 只出现在项目级 skill 源（`<repo>/.agents/skills`）和少数 harness 的共用目录。

apply 分两层：

| 层 | 从哪 | 怎么写 |
|---|---|---|
| 共有原数据 | `providers.yaml` + secrets、`packs/skills`、`packs/mcp`、`packs/agents` | 按各 harness 格式投影 |
| harness 定制 | `packs/harness/<id>/` | 只写回该工具 |

写回一律**字段级合并**（D5）：结构化文件按键、一文件一项按项，不整文件覆盖，不删用户自己的键。
MCP 从 `packs/mcp/mcp.yaml` 投影 🧪（设计见 [mcp-design.md](./mcp-design.md)；apply 命令还没接上）。

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
- **未探测到的 harness 默认跳过**（除非 `--only` 指定）。
- **能力不对等**（D10）：某家没有的面（skills / MCP / models）跳过，日志 `n/a`，不造空文件。

---

## 6. `magic provider` —— 日常切换 official ↔ 中转

`magic provider use` 是 Grok / Codex 的日常入口；apply 末尾投 Claude / Pi / OpenCode / WorkBuddy。

```bash
magic provider use shu          # 所有已实现的 harness 切到 shu
magic provider use company      # active 别名（= providers.yaml 的 active）
magic provider use official     # 切回官方
magic provider use official --harness grok
magic provider status           # 看每个 harness 当前 official 还是哪个 provider
magic provider test company     # HTTP 打一下 {base}/models，验证 key 通不通
magic provider seed             # 把内置 seed profiles 合并进 providers.yaml（缺的才加）
```

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
magic skills sync                         # 全局源 → 各 agent 全局 skills 目录（受管链接）
magic skills sync --scope project         # 源 = <repo>/.agents/skills
magic skills sync --scope project --include-global   # 项目为主，再继承全局（同名以项目为准）
magic skills status                       # 每个 agent：linked / missing / conflict / stale
magic skills doctor
magic skills import PATH                  # 把一个本地 skill 目录收进源（默认链接）
magic skills adopt --from AGENT           # 把某 agent 目录里没管的 skill 收编进源
```

`--include-global` 只对 **project scope 的 sync** 有意义，不是 `list` 的开关。

行为约定 ✅：**永不删除真实目录**；冲突 = 真目录或非受管链接，报出来不覆盖。
项目级 skill 用 `magic skills sync --scope project`，**不是** `magic apply`。

当前 skills 目标：`claude, codex, cursor, pi, grok, opencode, copilot, trae`。
L2 长尾表（vercel-labs/skills）已锁定为产品配置，代码还没接满。

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

## 命令速查（使用视角）

| 我想…… | 命令 | 状态 |
|---|---|---|
| 初始化配置库 | `magic init` | ✅ |
| 看能投哪些工具 | `magic adapters` | ✅ |
| 看本机已装哪些 | `magic status` | ✅（只列已装 🧪） |
| 加 provider 并选模型 | `magic login <provider>` | 🧪 |
| 手动加 provider | 编辑 `providers.yaml` + `magic secrets set` | ✅ |
| 看库里有啥 | `magic catalog` | ✅ |
| 把 provider 拷成文本 | `magic copy <provider>` | ✅ |
| 让各工具生效 | `magic apply` | ✅ |
| 把 MCP 投进各工具 | `magic apply`（`packs/mcp`） | 🧪 |
| 收一个外部文件进库 | `magic add path <file>` | 🧪 |
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
