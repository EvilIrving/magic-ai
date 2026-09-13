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
