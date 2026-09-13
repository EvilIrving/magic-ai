# Magic 文档

> **想判断产品对不对，先读 [usage.md](./usage.md)**；**已锁定的决定看 [decisions.md](./decisions.md)**；**还没定的看 [open-questions.md](./open-questions.md)**。

| 文档 | 给谁 | 内容 |
|------|------|------|
| [decisions.md](./decisions.md) | **对内** | **产品决定（已锁定，唯一清单）** |
| [open-questions.md](./open-questions.md) | **对内** | **未决清单**：还没定的决策 / 待调研 / 代码待改 |
| [usage.md](./usage.md) | **对外 + 对内** | 产品使用定义：命令 → 输入 → 终端输出 → 生成的文件 |
| [apply-design.md](./apply-design.md) | **对内** | **设计**：apply = 共有原数据投影 + 各 harness 定制 |
| [harness-profiles.md](./harness-profiles.md) | **对内** | **覆盖集 13 家的深度档案**：配置根 / skills / agents / MCP / models / hooks / custom + 声量复核 |
| [supported-harnesses.md](./supported-harnesses.md) | **对内** | 目标 harness 清单（Tier L1/L2/L3）+ 能力支持矩阵（能力不对等，**无优先级**） |
| [mcp-design.md](./mcp-design.md) | **对内** | MCP 多 harness 格式与翻译（已实现：`internal/mcp`） |
| [hooks-design.md](./hooks-design.md) | **对内** | hooks 归属（per-harness 字段级合并，B 方案，已实现） |
| [secrets-and-sync.md](./secrets-and-sync.md) | **对内** | **口径**：本地 `providers.json` 明文；`.enc` 只是云端状态 |
| [getting-started.md](./getting-started.md) | **对外** | 产品是什么、两层内容、安装、五分钟上手 |
| [library-layout.md](./library-layout.md) | **对内** | `~/.magic-ai` 目录已定死（skill / MCP / API / custom） |
| [../README.md](../README.md) | 对外 + 常用命令 | 英文 README：是什么、安装、最短路径、命令、边界 |
| [../README.zh-CN.md](../README.zh-CN.md) | 对外 + 常用命令 | 同上的中文版；命令、providers、进库手段、apply 投影、R2 |
| [harness-matrix.md](./harness-matrix.md) | **对内** | 各 harness 真实路径、投影、SwitchWriter |
| [harness-rollout-plan.md](./harness-rollout-plan.md) | **对内** | **排行榜 29 家全量调研 + 执行方案 + 两轮自审**（第三轮，D21） |
| [../LICENSE](../LICENSE) | 对外 | MIT |
| [comment-contract.md](./comment-contract.md) | **对内** | **注释契约**：`Source:` 来源标签、逻辑解释、开发约束（`Never:` / `Invariant:`）写在哪里 |
| [../AGENTS.md](../AGENTS.md) | **对内** | 产品边界、目的 vs 手段、数据方向、实现约定 |
| [model-registry-from-pi.md](./model-registry-from-pi.md) | 对内 | models.dev → `models.json` |
| [provider-onboarding.md](./provider-onboarding.md) | **对内** | `magic login` → 拉官方模型列表 → 勾选 → 写 providers.yaml + 密钥（已实现） |
| [release-signing.md](./release-signing.md) | **对内** | **规划中**：发预编译发行物时的 macOS 签名/公证 + Windows 签名准备清单与决策点 |
| [agent-guide.md](./agent-guide.md) | **对内** | **规划中**：内置一份给 harness/Agent 读的使用说明（`magic guide` + `--json`） |
| [desktop-app.md](./desktop-app.md) | **对内** | **规划中**：桌面端形态（Tauri / Wails 取舍）、Go 核心 vs 壳层、原生体验验收标准 |
| [deepseek-official-integrations.md](./deepseek-official-integrations.md) | 对内 | Claude / Codex 第三方投影模式 |
| [../NOTES.md](../NOTES.md) | 对内 | 变更日志（不是需求真相源） |

`python-magic/docs/` 原需求说明书若不在工作区，以本仓库 `AGENTS.md` + 上表为准。
