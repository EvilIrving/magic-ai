# Magic

[English](./README.md) · **中文**

**一份配置库，投到你的各个 AI coding 工具。**

Claude Code、Codex、Grok、OpenCode、Pi、Copilot、Trae 各要一份 skill、一份 MCP、一份 API。Magic 让你只维护 `~/.magic-ai` 里的一份，再按各工具自己的格式写进它们已经在用的配置文件。

```
~/.magic-ai/                          你改这里
      │  magic apply · magic skills sync
      ▼
~/.claude/  ~/.codex/  ~/.grok/  ~/.pi/  ~/.config/opencode/  …
```

不装那些工具，也不取代它们。只往它们的配置文件里写。

## 一份进，多份出

| 你维护一份 | 写到各工具里 |
|---|---|
| `packs/skills/` 里的 skill | `~/.claude/skills`、`~/.codex/skills`、`~/.cursor/skills`、`~/.pi/agent/skills`、`~/.grok/skills`、`~/.config/opencode/skills`、`~/.copilot/skills`、`~/.trae/skills` |
| `providers.yaml` + 密钥 | Claude `settings.json` 的 token、Pi / OpenCode 的 `apiKey`、Codex / Grok 的 provider |
| `packs/agents/*.md` 子代理 | 各工具的 `agents/` 目录 |
| 某个工具自己的文件（Pi theme、`AGENTS.md`） | 只还回那一个工具，换电脑后还是原来的样子 |

密钥在本地是明文 `~/.magic-ai/secrets/providers.json`（0600）。推到云端时，只有含密钥的文件才加密成 `.enc`。

**当前状态：** skill、provider / API 投影、子代理、per-harness 自定义文件、MCP 投影、hooks 均已可用。

## 安装

需要 Go 1.24+。

```bash
git clone https://github.com/EvilIrving/magic-ai.git
cd magic-ai
make build          # 或：go build -o magic ./cmd/magic
```

把 `magic` 放进 `PATH`。

## 最短路径

```bash
magic init
magic secrets set DEEPSEEK_API_KEY --value 'sk-…'   # 按你实际用的 key 名
# 编辑 ~/.magic-ai/providers.yaml，设 active 和 pick
magic apply
magic skills sync
```

然后打开 Claude / Pi / OpenCode，配置已经在里面。还原：

```bash
magic provider use official   # 一方订阅（Claude / Codex / Grok / WorkBuddy）
magic provider use none       # 清掉 Magic 写的条目（Pi / OpenCode：多 provider）
```

Grok、Codex 日常切中转用 `magic provider use <名字>`，不是 `apply` 末尾。

项目级 skill 的源是 `<repo>/.agents/skills`：

```bash
magic skills sync --scope project
```

## 常用命令

| 命令 | 做什么 |
|---|---|
| `magic status` | 配置库状态 + 本机装了哪些 harness |
| `magic guide` | 打印上手顺序（`--json`） |
| `magic adapters` | 支持哪些 harness，各自有没有 skill / 子代理 / MCP |
| `magic apply` | 把库写进各 harness（可 `--dry-run` / `--only pi`） |
| `magic plan` | 动手前看 apply 会写什么：create / update / delete / unchanged，加键级与行级差异（`--json` / `--only pi`，只读） |
| `magic verify` | 检查库与本机配置的漂移；漂移、空检查、被跳过的 adapter 都返回退出码 1（只读） |
| `magic rollback` | 按 `backups/<时间戳>/manifest.json` 退回一次 apply（`--list` / `--dry-run` / `--force`） |
| `magic skills sync` | 全局 skill；项目级加 `--scope project` |
| `magic skills status` | 每个 harness 的已链接 / 缺失 / 冲突 / 失效（只读） |
| `magic agents` | 配置库里的子代理，以及哪些 harness 支持 |
| `magic mcp list\|status\|init` | 规范 MCP server · 各 harness 是否已写入 · 示例文件 |
| `magic provider use official\|none\|<名字>` | 在 一方 / 清空 / 第三方 provider 之间切换某个 harness |
| `magic provider add` | 把内置 provider 写进 `providers.yaml`（`--list` / `--all`） |
| `magic login <provider>` | 拉 `/models`、交互勾选，写 provider + 密钥 |
| `magic secrets set\|list` | provider 密钥（`set` 写入 `secrets/providers.json`） |
| `magic add <path>` | 把某个 harness 本地的文件收进配置库 |
| `magic catalog` · `magic copy` | 列 `providers.yaml` · 打印单个 provider 配置（密钥取本地明文库） |
| `magic registry sync\|search\|show` | models.dev 模型目录，apply 时的元数据来源 |
| `magic sync push\|pull\|status` | 把 `~/.magic-ai` 备份到 Cloudflare R2（需 `MAGIC_R2_*`） |

完整「敲什么、屏幕出什么、磁盘变成什么样」见 [docs/usage.md](./docs/usage.md)，先读这一份。

## 支持的 harness

适配器：`claude`、`codex`、`grok`、`pi`、`opencode`、`workbuddy`（CodeBuddy）、`gemini`、
`copilot`、`trae`、`cursor`、`antigravity-cli`、`kimi-code`、`qoder`、`qoder-cn`、`cline`、
`kiro`、`droid`、`zcode`、`qwen-code`、`iflow-cli`、`kilo`、`zed`，
以及只管 MCP 配置的 `mcpuser`。覆盖范围按用量分档（OpenRouter app 榜，v1 核心；长尾推迟），
见 [docs/supported-harnesses.md](./docs/supported-harnesses.md)。
各家路径和能力见 [docs/harness-matrix.md](./docs/harness-matrix.md)。

## 配置库布局

```
~/.magic-ai/
  magic.yaml                  启用哪些 harness
  providers.yaml              中转 + 模型 pick（无密钥）
  secrets/providers.json      明文密钥，0600
  packs/skills/<name>/        跨工具 skill（全局真相源）
  packs/mcp/                  mcp.yaml（你添加）
  packs/agents/<name>.md      子代理
  packs/harness/<id>/         某工具自己的文件，只写回该工具
  registry/                   models.dev 缓存，不是用户手改的配置
  backups/                    apply 备份，不同步
```

详见 [docs/library-layout.md](./docs/library-layout.md)。

## 它不管

- 不安装 Claude / Pi / Grok
- 不同步会话、缓存、登录态
- 不是 skill 市场，也不做 Cherry Studio 那种聊天客户端
- 不内置中转站（SHU 等），那些写在你自己的 `providers.yaml` 里

## 文档

| | |
|---|---|
| [docs/usage.md](./docs/usage.md) | 怎么用（先读这个） |
| [docs/getting-started.md](./docs/getting-started.md) | 安装和五分钟 |
| [docs/harness-matrix.md](./docs/harness-matrix.md) | 各工具路径和能写什么 |
| [docs/library-layout.md](./docs/library-layout.md) | `~/.magic-ai` 目录 |
| [docs/README.md](./docs/README.md) | 文档总索引 |

## 许可

MIT，见 [LICENSE](./LICENSE)。

版本 1.0.0（`magic version`）。
