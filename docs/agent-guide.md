# 面向 Agent 的内置使用说明（规划中）

> **状态：未实现。** 这是待办清单 + 设计决策，不是已有功能。当前二进制里**没有**任何 agent 专用说明，命令也**没有** `--json`。
> 真开始做时，从 [§6 待办](#6-待办入口) 往下走。

---

## 1. 问题

Magic 不只是给人用的。它是 CLI，**其他 harness（Claude Code / Pi / Codex / OpenCode / Grok…）会通过 shell 调用它**。用户的 agent 可能会被要求「用 magic 把 skill 同步一下」「用 magic 切到公司 provider」。

但 agent 调用时缺一份**它能读懂的、跟版本对齐的**使用说明：

- 它可能**拿不到本仓库**。用户是全局装的 `magic`，agent 只看得到二进制。
- 它需要知道每条命令**改什么文件、有什么副作用、能不能安全自动跑**，而不是命令列表。
- 它需要**结构化输出**，人类表格它解析不可靠。
- 它需要**安全边界**：哪些命令有破坏性、哪些需要用户确认、失败怎么回退。

现在这些信息散在人读的 `README.md` / `docs/usage.md` / `--help` 里，二进制里没有，输出也不是给机器消费的。

---

## 2. 现有事实（代码，不是设想）

| 事实 | 证据 |
|---|---|
| 全仓库**没有** `--json`，所有输出都是 `fmt.Println` 人类文本 | `grep -rn '"json"' cmd/magic/*.go` 无结果 |
| 全仓库只有**一处** `go:embed` | `internal/provider/builtin/builtin.go:18`（`providers.yaml`） |
| 没有 `magic guide` / `--agent` 之类入口 | `cmd/magic/main.go` 的 `AddCommand` 列表 |
| `magic` 根命令的 `Long` 有少量 agent 相关信息 | 说明真相源、环境变量、`agents` 命令歧义 |
| **`agents` 这个名字已被占用** | `magic agents` = 列出配置库 `packs/agents` 里的 **sub-agent 定义**，与本文说的「Magic 自己的 agent 说明」是两件事 |

`docs/usage.md` 已经是「命令 → 输入 → 终端输出 → 生成的文件」的人工真相源，是派生 agent 说明的最佳底本。

---

## 3. 命名：别撞 `agents`

新功能**不能**叫 `agents` / `magic agent`，会和现有的 sub-agent 语义混淆（用户和 agent 都会混）。

候选（待定，选一个）：

- `magic guide` —— 打印内置说明。简洁，动词少。
- `magic agent-guide` —— 明确受众。
- `magic docs` / `magic usage` —— 容易和 docs 目录/usage.md 混。

倾向 `magic guide`，`--json` / `--format` 作为参数区分人读 / 机读。

---

## 4. 要内置哪些内容（三层，别混在一起）

| 层 | 内容 | 谁最需要 |
|---|---|---|
| **L1 全局综述** | Magic 是什么、真相源在哪（`~/.magic-ai`）、数据方向（用户配置进库、apply 出库）、哪些命令改库 / 改 harness | 第一次接触的 agent |
| **L2 命令参考** | 每条子命令 + 参数 + **副作用**（写哪些路径）+ 退出码 + 幂等性 | 要执行具体操作的 agent |
| **L3 操作规约** | 安全边界、典型工作流（「用户想加一个 skill」→ 具体步骤）、出错怎么办 | 要自主决策的 agent |

L2 是重点。agent 不需要「Magic 很强大」，它需要「`magic skills sync` 会写 `~/.claude/skills` 等 8 个目录，幂等，可安全重跑」。

L2 的每条应至少标注：

- 改的是**配置库**还是**harness 文件**（这决定副作用可逆性）
- 是否**幂等** / 能否安全重复执行
- 是否**有破坏性**（删链接、覆盖文件、prune）
- 是否需要口令等前置条件（现状：仅 `sync push/pull` 需要；`apply` / `copy` 不需）
- 是否**需要用户确认**才能跑

---

## 5. 输出通道

| 通道 | 作用 | 现状 |
|---|---|---|
| `magic guide`（内嵌 `go:embed` markdown） | agent 不用仓库就能读到说明 | ❌ |
| `--json` 用于 `list` / `status` / `adapters` / `skills status` 等只读命令 | agent 解析结构化结果，不解析表格 | ❌ |
| 命令级安全标记（幂等 / 破坏性 / 需确认） | agent 判断能否自动执行 | ❌ |
| 把 Magic 自己当作一个 **skill** 投影到各 harness（`magic apply` 的元用法） | agent 在 harness 里直接看到 magic 技能 | ❌ |
| 仓库内 `AGENTS.md` / 说明文件 | 只对**有仓库**的 agent 有效 | 已有（但非二进制内置） |

倾向：**L2 的单一真相源放仓库一处**（派生自 `docs/usage.md`），构建时 `go:embed` 进二进制，`magic guide` 输出。避免二进制说明和仓库文档各写一份、各自过期。

版本对齐：`magic guide` 输出里带版本号（现有 `magic/internal/core.Version`，构建时 `-ldflags` 注入，见 `Makefile`），并在版本不符时提示。

---

## 6. 待办（入口）

- [ ] 定命令名（`magic guide`？）与参数（`--json` / `--format`）
- [ ] 定 L2 命令参考的单一真相源：从 `docs/usage.md` 派生，还是新建结构化清单（YAML/JSON）再渲染成人和 agent 两种格式
- [ ] 为只读命令加 `--json`（`adapters` / `status` / `skills list|status` / `agents` / `catalog`）
- [ ] 给每条命令标注副作用 / 幂等性 / 破坏性 / 前置条件（这张表本身要能进 help 和 guide）
- [ ] `go:embed` + `magic guide` 命令
- [ ] 评估「把 magic 自己作为 skill 投影」的可行性（与 `packs/skills`、`magic apply` 的关系）
- [ ] 写 `packs/agents/` 里一个示例 sub-agent？—— 注意：这是**另一件事**，别和本功能混

---

## 7. 边界

- 这不是「给 Magic 加一个 AI」。是让**别的** AI 能正确使用 Magic。
- 不替代 `docs/usage.md`（人读）和 `AGENTS.md`（贡献者契约），而是给它们加一个**内置、机读、随版本**的出口。
- 不与 `magic agents`（sub-agent 定义投影）合并。
