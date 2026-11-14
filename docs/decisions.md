# 产品决定（已锁定）

状态：**锁定**。这是**唯一**的产品决定清单——其它文档只讲「怎么做」，不再重复列决定。
要改决定就改这里，再同步受影响的文档。仍**未定**的看 [open-questions.md](./open-questions.md)。

更新：2026-09-13。

## 一、配置库与密钥

| # | 决定 |
|---|---|
| D1 | 配置库 `~/.magic-ai` 是**唯一真相源**；各 harness 家目录只是**投影目标**（不是备份源） |
| D2 | 密钥本地写明文 `secrets/providers.json`（0600）。本地**没有** `.enc`、**不是**加密状态。`.enc` **只表示云端状态**：`sync push` 把含密钥的文件加密后以 `.enc` 上传，其余明文；`sync pull` 解密后落回本地明文（无 `.enc`）。口令只用于 push/pull |
| D3 | 投影到 harness 的 key 一律写**真值**（Pi / WorkBuddy / OpenCode / Claude），不用 `$KEY_ENV` 占位。Codex 用其原生 `env_key` |
| D4 | **无迁移入口**：开发阶段、零用户，不写兼容旧格式的代码。capture 已删，不做回流 |

## 二、apply / 写回

| # | 决定 |
|---|---|
| D5 | 写回一律**字段级合并**（结构化文件按键、一文件一项按项），**不整文件覆盖**；绝不删用户自己的键/文件 |
| D6 | per-harness 定制放 `packs/harness/<id>/`，只写回该 harness，不译成别的工具 |
| D7 | 收外部文件进库用 **`magic add path <file> [--as <harness>/<rel>]`**（复制，不动原文件） |
| D8 | MCP 密钥就写**明文真值**进 harness 配置，不做 `${ENV}` 间接；**project 级 MCP 暂不做** |

## 三、harness 覆盖

| # | 决定 |
|---|---|
| D9 | **不分优先级**：所有 harness、所有能力都做 |
| D10 | **能力不对等**：某家不支持就跳过、日志 `n/a`，不造空文件 |
| D11 | L2 skills 路径表 = **直接采纳 `vercel-labs/skills` 的表**（固定配置，不做运行时探测） |
| D12 | `magic status`：**全量探测**本机，但**只显示已安装的** harness |
| D13 | **不做 windsurf / roo**（均停运） |
| D14 | `gemini` 保留作过渡；新目标 **`antigravity-cli`**（`agy`，取代 Gemini CLI） |
| D15 | `workbuddy` = **CodeBuddy**（同一产品，共用 `~/.codebuddy/`） |

## 四、交互

| # | 决定 |
|---|---|
| D16 | `magic login` = **TUI 交互流程**（选 provider → 输 key → 拉 `/models` → 勾选 → 写库） |

## 五、登录流程（`magic login`）

| # | 决定 |
|---|---|
| D17 | TUI 库 = **Bubble Tea**（`charmbracelet/bubbletea` + `bubbles` + `lipgloss`）。步骤：选/定 provider → key（参数 / 已存 / 输入）→ GET `/models` → 多选 → **确认后**写 `providers.yaml`（`pick` 整段替换，`key_env`/`endpoints`/`models` 用户优先）+ 本地明文 secret。回退顺序（必须说明用了哪种）：live `/models` → 内置模板 `pick` → `registry/models.json` → 手动输入。非交互：`--api-key` + `--pick` + `--yes`（可选 `--no-live`）。不写 `models_meta`。 |

## 六、MCP 投影

| # | 决定 |
|---|---|
| D18 | 库 `packs/mcp/mcp.yaml`（规范形：`transport/command/args/env/cwd/url/headers/enabled`）→ 各 harness 原生格式。密钥写**明文真值**（D8）。**不做名字前缀**：Magic 管理的条目 = 名字出现在 `packs/mcp/mcp.yaml` 的条目；用户其它条目**永不删除**（A7 定）。不支持/无法表达的 server → `n/a` 日志、跳过（D10）。仅 user 级（project 级暂不做，D8）。落点：Claude `~/.claude.json` `mcpServers`；Codex/Grok `config.toml` `[mcp_servers.*]`；OpenCode `opencode.json` `mcp`（`type:local/remote`）；Pi `~/.pi/agent/mcp.json` `mcpServers`（remote 需第三方 adapter，跳过）；Gemini `settings.json` `mcpServers`（http→`httpUrl`，sse→`url`）；Copilot `mcp-config.json`（`type:local/http/sse`）；WorkBuddy `~/.codebuddy/settings.json`；Trae `~/.trae/mcp.json`；mcpuser `~/.config/mcp/mcp.json`。 |

## 七、hooks 投影

| # | 决定 |
|---|---|
| D19 | 用 **B 方案**（per-harness 文件）：`packs/harness/<id>/hooks/…` 按 custom 层**字段级合并**写回，**不做语义层翻译**（A5 定）。声明式：Claude `settings.json` 的 `hooks`、Codex `hooks.json`、Grok `hooks/*.json`、Copilot `hooks/*.json`；代码式：Pi `extensions/*.ts`、OpenCode `plugins/*.js|ts` 原样收放。不支持 → `n/a`，不造空文件。hook 里的路径值**原样写**（不折 `~`，A8 定）。Codex hooks 不写 trust 元数据（A6 定）。Cursor 无产品适配器，Magic **不写** Cursor hooks（A5）。 |

## 八、v1 覆盖范围（修订 D9）

| # | 决定 |
|---|---|
| D20 | **v1 按真实用量分档，不追求全市场**（修订 D9「所有 harness 都做」）。依据 OpenRouter apps 榜（2026-09）。P0 = 已有；**P1 = 本次补齐 Cursor / Antigravity CLI / Kimi Code / Qoder(+CN) / Cline**（已有 adapter + MCP + skills）；P2（droid/kiro/minimax-code/zcode/hermes/openclaw/deepseek-harness/oh-my-pi/dimagent/alma/craftagent/cola）与长尾~50 家**推迟**，skill 路径已入表、`--target all` 可达。**Cursor 由「不做产品适配器」改为完整 adapter**（skills/MCP/hooks，无 model SwitchWriter）。（D19 里「不写 Cursor hooks」随之失效：现在写。）windsurf/roo 仍不做（D13）。 |

## 九、provider switch 语义（`official` vs `none`）

| # | 决定 |
|---|---|
| D21 | **`official` 只在一方有原生订阅/登录的 harness 上成立**：Claude / Codex / Grok / WorkBuddy。**Pi / OpenCode 是多 provider 聚合器，没有单一 official**（Pi：`/login` 订阅或 API key + `auth.json`；OpenCode：预加载 75+ provider + `auth.json`，Zen 可选）→ 这两家用 **`none`**（清掉 Magic 写进配置的条目）。`provider use official` 对 Pi/OpenCode 报错并提示改用 `none`；无 `--harness` 时对这两家记 `n/a` 而不是失败。`none`/`off`/`clear` 对所有 harness 有效。依据各自官方文档（2026-09-13）。 |

## 九、排行榜全量支持（2026-09-13 第三轮）

| # | 决定 |
|---|---|
| D21 | **用户排行榜第 2–30 名逐家调研，能投的都投。** 新增完整 adapter：`omp`（Oh My Pi）、`hermes`、`mimocode`（MiMo Code）、`dimagent`、`openclaw`；MCP 新增 4 个 dialect + `FormatYAML`（Hermes）。方案与证据见 [harness-rollout-plan.md](./harness-rollout-plan.md)。 |
| D22 | **Roo Code（榜 #30）确认 2026-05-15 归档停运 → 不做**（维持 D13）。如需照顾存量用户，另议是否支持社区 fork ZooCode。 |
| D23 | **UI-only 不投影**：Alma（供应商编排桌面客户端）、MiniMax Code 的 MCP、Lingma/Qoder CN 的 IDE MCP——无公开可写配置文件，Magic 不写，`n/a`。 |
| D24 | **DeepSeek Harness MCP 暂缓**：其 MCP 走 cordis 插件树（`~/.dsh/cordis.patch.yml`），不是扁平配置；产品处 developer preview，P3 单独设计，先只做 `settings.yaml` 自定义层。 |
| D25 | **`Cola` 身份未确认** → 标未确认，需用户提供官网/仓库后再定；不按猜测建 adapter。 |

> **2026-09-13 确认**：D22–D25 均按建议执行，本轮不再推进（Roo 不做、Alma/Cola 不投、DSH MCP 暂缓）；
> 除非用户后续主动提出。`dsh`/`craft`/`trae-agent` 等未完成项同理，见 [harness-rollout-plan.md](./harness-rollout-plan.md) §5/§9。

## 十、apply 的写入范围

| # | 决定 |
|---|---|
| D26 | **`magic apply` 默认只写已安装的 harness**：adapter 的 `Detect(userHome)` 命中才写，镜像树、`packs/skills`、`packs/agents`、共享 `packs/AGENTS.md`、provider、MCP 六处写入共用这一道门。`--only <id>` 显式指名时绕过，用于在新机器上先写配置（预装），再打开工具。`magic plan` 用同一个门：未安装且未被 `--only` 指名的 adapter 报 skipped（原因「未安装」），不算漂移。理由：不污染用户主目录（否则一台只装了 3 个工具的计算机会被造出另外 17 个工具目录）、与 `magic status`（D12）以及 provider / MCP 的探测语义保持一致。 |
