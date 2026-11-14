# 配置库布局（已定死）

真相源：用户家目录 `~/.magic-ai`（`MAGIC_HOME`）。与该目录 `README.md` 同步。

产品：一份 skill / MCP / 中转 API，按各 harness 格式写进去；外加各工具自己的自定义文件方便换电脑。手改 / CLI 命令是进这个目录的手段。

```
~/.magic-ai/
  README.md
  magic.yaml                哪些 harness 启用
  providers.yaml            中转 + 模型 pick（无密钥）
  secrets/providers.json    明文密钥（0600）；key_env → value
  packs/skills/<name>/
  packs/mcp/                mcp.yaml（你添加）
  packs/agents/<name>.md
  packs/AGENTS.md           共享用户级指令（可选）：一份内容写进各家用户级文件
  packs/harness/<id>/   theme、extensions、AGENTS.md、memories、hooks…（按文件写回该 harness）
  registry/                 models.dev 缓存，不是用户手改的配置
  backups/                  apply 备份，不同步
```

| 层 | 路径 | apply |
|---|---|---|
| 共有原数据 | `providers.yaml` + `secrets/`、`packs/skills/`、`packs/mcp/`、`packs/agents/` | 译成 Pi / Claude / Codex / Grok / OpenCode… 各自格式 |
| 共享用户级指令 | `packs/AGENTS.md` | 同一份内容写进每一家已安装 harness 的用户级指令文件（下表） |
| 某工具自定义 | `packs/harness/<id>/` | 只写回这个 harness，不译成别的工具 |

## 共享用户级指令：一份 AGENTS.md → 各家路径

用户级 AGENTS.md 各家用不同路径，没有统一约定（上游讨论 `agentsmd/agents.md` issue #91）。`packs/AGENTS.md` 是可选的一份共享源，投影规则按优先级：

1. `packs/harness/<id>/AGENTS.md`（Claude Code 是 `CLAUDE.md`）存在 → 用它，写回该家（即镜像层，行为不变）
2. 否则 `packs/AGENTS.md` 存在且**含指令内容** → 同一份内容写进下表每个路径
3. 两者都没有 → 不写

只对**路径已核实**且**已安装**（`Detect` 为真，或 `--only` 指名）的 harness 生效。`magic init` 生成的 `packs/AGENTS.md` 是只含注释的占位：去掉注释与空行后没有内容时 apply 不写任何目标文件，所以 init + apply 是惰性的，不会把注释塞进用户的 CLAUDE.md，也不会用空内容覆盖已有文件。

| harness | 用户级文件 | 来源（2026-11-14 核） |
|---|---|---|
| Codex | `~/.codex/AGENTS.md` | developers.openai.com/codex/agent-configuration/agents-md |
| Claude Code | `~/.claude/CLAUDE.md` | code.claude.com/docs/en/memory（user instructions） |
| Droid | `~/.factory/AGENTS.md` | docs.factory.com/harness/agents-md（personal directories） |
| Grok | `~/.grok/AGENTS.md` | grok 1.0.77 `user-guide/01-getting-started.md`（`GROK_HOME`） |
| OpenCode | `$XDG_CONFIG_HOME/opencode/AGENTS.md`（默认 `~/.config/opencode/AGENTS.md`） | opencode 文档 “Instructions” + 源码 `config/plugin/instruction.ts` |
| Pi | `~/.pi/agent/AGENTS.md` | pi-mono `packages/coding-agent/docs/configuration.md`（agent dir） |

未核实到用户级文件的 harness（不写）：amp（且本仓库没有 amp adapter）、cursor（User Rules 在应用设置里，不是文件）、copilot、gemini、qwen-code、kimi-code、trae、cline、kiro、zcode、iflow、kilo、zed。

这些写入和镜像文件一样进 `manifest.json`，`magic rollback` 能撤回；`magic plan` / `verify` 用相对 home 的路径表示（如 `.codex/AGENTS.md`），与镜像 rel（`AGENTS.md`）区分。

## 密钥：本地明文，`.enc` 是云端状态

- 本地 `secrets/providers.json` 是**明文**（`0600`）。本地没有 `.enc`，不是加密状态，不需要口令。
- `.enc` **只表示云端状态**：`sync push` 把含密钥的文件加密后上传为 `<rel>.enc`，其余明文；`sync pull` 解密后落回本地明文（无 `.enc`）。
- `MAGIC_PASSPHRASE`（或等价口令）只在 push / pull 时需要，**不进库、不同步、不进 R2**。
- 口径见 [secrets-and-sync.md](./secrets-and-sync.md)（D2）。

不进库：access token、OAuth、官方登录态（`auth.json`）、会话、缓存、`npm/`、`bin/`、运行态。
harness 自定义放 `packs/harness/<id>/`（capture 已删，但目录保留作为自定义层，按文件写回该 harness）。
