# 未决清单

状态：**待你决策**。只列**还没定**的事。
**已锁定的产品决定见 [decisions.md](./decisions.md)**（唯一定义处，本文件不重复）。

更新：2026-09-13。

---

## A. 还没定的决策

> 本表是**全库汇总**：各文档的「未定」段不再重复，只指回这里。

| # | 问题 | 选项 | 出处 |
|---|---|---|---|
| A1 | `sync` 加密口令来源 | 环境变量 `MAGIC_PASSPHRASE` / 系统 keychain / 本机文件 | `secrets-and-sync.md` |
| A2 | Codex `env_key` 是否也写真值 | 保持 `env_key`（Codex 原生机制）/ 改成真值 | `usage.md`、`harness-matrix.md` |
| A3 | `magic login` 细化 | 用哪个 TUI 库；写库粒度（直写 vs dry-run）；无 `/models` 的 provider 回退顺序；多协议是否按 endpoint 再分；是否顺带写 `models_meta` | `provider-onboarding.md` |
| A4 | 多机冲突检测实现路径 | 方向已定（比记录不比整份文件），**怎么实现未定** | `AGENTS.md` |
| A5 | Cursor 会加载 Claude hooks | 两边都写会重复触发 / 只写原生那份 | `hooks-design.md` |
| A6 | Codex hooks 信任 | Magic 写的是否算 managed（用户是否要逐条 trust） | `hooks-design.md` |
| A7 | MCP 管理条目标记 | `magic-` 前缀 / 单独 manifest | `mcp-design.md` |
| A8 | hooks 里本机路径是否折成 `~` | 折（可移植）/ 不折（原样） | `hooks-design.md` |

> 已不属于决策（写进 C 或已定）：代码式 hooks 归类（已定：文档归 hooks、实现走 custom）；
> 重复 provider 合并（= 字段级合并）；`login` 后 auto-push（约定不自动）；
> Codex/Grok TOML 是否复用 `internal/provider`（实现细节，见 C）。

---

## B. 待调研

第三轮（agents + hooks 深挖）已完成想支持的主要几家，新数据在 `supported-harnesses.md`「第二轮调研」表和 `hooks-design.md`。**不做**：Tabnine / Rovo Dev / Junie / Posit Assistant / Neovate（没人用）。

剩下：

| # | 项 | 状态 |
|---|---|---|
| B1 | Droid / Kiro / Goose / Zed / Continue 的 **hook 事件名** | 已知有 hooks，事件名待抄 |
| B2 | iFlow / Amp / OpenHands / Zencoder / Mistral Vibe / Trae / Lingma / Cortex / Pochi / MiniMax 的 **hooks** | ❓ |
| B3 | Trae / Mistral Vibe / Cortex / Pochi / MiniMax 的 **custom**（per-harness 定制文件面） | ❓ |
| B4 | Trae / Gemini / Antigravity CLI / Kiro / OpenHands / Zencoder 的 **models 配置** | ❓ |
| B5 | **grok** 的原生 mcp（`config.toml` 还是只靠兼容读） | ❓ |
| B6 | Qwen Code / Crush / Goose / iFlow / MiniMax 的 **skills 目录** | ❓ |

**第三轮已查：**

- **agents**：Cursor（Agent Plugins）、Cline（Subagents）、**Kilo**（`~/.config/kilo/agents/*.md`、`.kilo/agents/*.md`）、Qwen（Subagents）、iFlow（Sub Agent）、Goose（recipe 定义）、**Droid**（`~/.factory/droids/*.md`）、**Kiro**（`.kiro/agents/`）、**Kimi**（`~/.kimi-code/agents/` + `~/.agents/agents/`，且**跟 Claude/OpenCode agent 文件互通**）、Zed（External Agents / ACP）、Continue（config.yaml）。
- **hooks**：**Cursor**（`hooks.json`，20+ 事件，支持 prompt 型，能加载 Claude hooks）、**Qwen Code**（17 事件，含 http/prompt 类型）、**Kimi Code**（`[[hooks]]` TOML，20 事件）、**Kilo**（❌ 无原生）。

见 `supported-harnesses.md` / `mcp-design.md` / `hooks-design.md`。

---

## C. 代码待改（文档已定，实现未跟）

| # | 差异 | 目标 |
|---|---|---|
| C1 | `internal/secrets` 仍是本地加密/明文双模式 + `MAGIC_SECRETS_MODE`；sync 仍把本地 `.enc` 当上传物 | 本地恒 `secrets/providers.json` 明文、无 `.enc`；push 加密后上传为 `<rel>.enc`；pull 解密落回明文。无 `MAGIC_SECRETS_MODE` |
| C2 | Pi / WorkBuddy apply 仍写 `$KEY_ENV` / `${KEY_ENV}` | 写**真值** key（D3） |
| C3 | apply 的 `<adapterID>.json.enc` reinject | 删除（本地即最终内容，无 reinject） |
| C4 | 内置模板合并进 `providers.yaml` + `magic provider add` | 未实现 |
| C5 | `magic login`（TUI + 拉模型 + 勾选） | 未实现 |
| C6 | `magic add path` | 未实现 |
| C7 | apply 写回仍是整文件覆盖（非字段级合并） | 改为字段级合并 |
| C8 | `magic status` 仍列全部 adapter（含未装） | 只列已安装的 |
| C9 | `magic guide` / `--json` / 桌面端 / 签名发行 | 未实现 |
| C10 | MCP 写 Codex/Grok 的 TOML 是否复用 `internal/provider` 已有处理 | 待定 |

---

## 怎么用

- **A**：勾选项，我落进对应文档。
- **B**：还没查的配置面（B1–B3），我可以并行查。
- **C**：文档已定、代码没改；等你说「开始写代码」再动。
