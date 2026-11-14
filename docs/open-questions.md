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
| ~~A3~~ | `magic login` 细化 — **已定**（D17） | TUI=Bubble Tea；确认后写库；回退 live→内置 pick→registry→手动；不写 `models_meta` | `provider-onboarding.md` |
| A4 | 多机冲突检测实现路径 | 方向已定（比记录不比整份文件），**怎么实现未定** | `AGENTS.md` |
| ~~A5~~ | Cursor 加载 Claude hooks — **已定**（D19） | 本仓库不做 Cursor 产品适配器 → **不写 Cursor hooks**；将来加时只写 Cursor 原生那份 | `hooks-design.md` |
| ~~A6~~ | Codex hooks 信任 — **已定**（D19） | 按普通用户配置写，**不加** trust 元数据（Codex 自身机制处理） | `hooks-design.md` |
| ~~A7~~ | MCP 管理条目标记 — **已定**（D18） | **无前缀**；按名字出现在 `packs/mcp/mcp.yaml` 判定 Magic 管理，用户其它条目永不删 | `mcp-design.md` |
| ~~A8~~ | hooks 路径折 `~` — **已定**（D19） | **不折**，原样写 | `hooks-design.md` |

> 已不属于决策（写进 C 或已定）：代码式 hooks 归类（已定：文档归 hooks、实现走 custom）；
> 重复 provider 合并（= 字段级合并）；`login` 后 auto-push（约定不自动）；
> Codex/Grok TOML 是否复用 `internal/provider`（实现细节，见 C）。

---

## B. 待调研

第三轮（agents + hooks 深挖）已完成想支持的主要几家，新数据在 `supported-harnesses.md`「第二轮调研」表和 `hooks-design.md`。**不做**：Tabnine / Rovo Dev / Junie / Posit Assistant / Neovate（没人用）。

剩下：

| # | 项 | 状态 |
|---|---|---|
| B1 | Droid / Kiro / Goose / Zed / Continue 的 **hook 事件名** | 已知有 hooks，事件名待抄；**这些属 v1 的 P2/长尾**（D20） |
| B2 | iFlow / Amp / OpenHands / Zencoder / Mistral Vibe / Trae / Lingma / Cortex / Pochi / MiniMax 的 **hooks** | ❓（P2/长尾） |
| B3 | Trae / Mistral Vibe / Cortex / Pochi / MiniMax 的 **custom**（per-harness 定制文件面） | ❓ |
| B4 | Trae / Gemini / Antigravity CLI / Kiro / OpenHands / Zencoder 的 **models 配置** | ❓ |
| B5 | **grok** 的原生 mcp（`config.toml` 还是只靠兼容读） | ❓ |
| B6 | Qwen Code / Crush / Goose / iFlow / MiniMax 的 **skills 目录** | ❓ |

> 哪些 harness 值得做、哪些推迟，见 [supported-harnesses.md](./supported-harnesses.md) 的
> 「用量驱动的 v1 覆盖范围」（D20）。B 表不再单独决定优先级。

**第三轮已查：**

- **agents**：Cursor（Agent Plugins）、Cline（Subagents）、**Kilo**（`~/.config/kilo/agents/*.md`、`.kilo/agents/*.md`）、Qwen（Subagents）、iFlow（Sub Agent）、Goose（recipe 定义）、**Droid**（`~/.factory/droids/*.md`）、**Kiro**（`.kiro/agents/`）、**Kimi**（`~/.kimi-code/agents/` + `~/.agents/agents/`，且**跟 Claude/OpenCode agent 文件互通**）、Zed（External Agents / ACP）、Continue（config.yaml）。
- **hooks**：**Cursor**（`hooks.json`，20+ 事件，支持 prompt 型，能加载 Claude hooks）、**Qwen Code**（17 事件，含 http/prompt 类型）、**Kimi Code**（`[[hooks]]` TOML，20 事件）、**Kilo**（❌ 无原生）。

见 `supported-harnesses.md` / `mcp-design.md` / `hooks-design.md`。

---

## C. 代码待改（文档已定，实现未跟）

更新：2026-09-13 第二轮。✅ = 已落代码并有单测。

| # | 项 | 状态 | 落点 |
|---|---|---|---|
| C1 | 本地恒 `secrets/providers.json` 明文、无 `.enc`；push 加密上传 `<rel>.enc`；pull 解密回明文；无 `MAGIC_SECRETS_MODE` | ✅ | `internal/secrets`、`internal/sync/r2.go` |
| C2 | Pi / WorkBuddy 写**真值** key（D3） | ✅ | `catalog.BuildPiProviderEntry`、`provider/pi.go`、`provider/workbuddy.go`、`core/providers_apply.go` |
| C3 | 删除 apply 的 `<adapterID>.json.enc` reinject | ✅ | `core/apply.go`、`core/claude_json.go`、`internal/secrets/store.go` |
| C4 | 内置模板合并进 cfg 读取 + `magic provider add` | ✅ | `provider/builtin`（`Merge` / `AddSelection`）、`provider.LoadCatalog`、`cmd/magic/provider_cmd.go` |
| C5 | `magic login`（TUI + 拉模型 + 勾选） | ✅ | 已落代码（`internal/login` Bubble Tea；`internal/provider/modelfetch.go`；`cmd/magic/login.go`）。决定见 D17 |
| C6 | `magic add path` | ✅ | 早已实现（`cmd/magic/add.go`、`core/add_path.go`） |
| C7 | apply 写回字段级合并（非整文件覆盖） | ✅ | `core/merge.go`（JSON deep merge + TOML section upsert；AGENTS.md 等整份覆盖） |
| C8 | `magic status` 只列已安装 harness（D12） | ✅ | `cmd/magic/main.go` |
| C9 | `magic guide` / `--json`（桌面端 / 签名发行不在本仓库范围） | ✅ / ➖ | `cmd/magic/main.go`（guide、`status --json`、`adapters --json`）；桌面/签名为独立产品 |
| C10 | MCP 写 Codex/Grok 的 TOML 是否复用 `internal/provider` 已有处理 | ✅ | 自写 `internal/mcp/toml.go`（`[mcp_servers.<name>]` 字段级 upsert，保留用户同表其它键）；D18 |
| C11 | `hermes` / `mimocode` 在 `DefaultAgents` 里、不在 vercel 表里：默认 sync/status 现在按「表 → adapter 的 `SkillsDir`」回退解析（`internal/skills/paths.go: globalAgentDir`），但 `ParseAgents` 仍用 `KnownAgent`（只认表），所以 `--target hermes` 会被拒、`--target all` 不含它们 | ❓ 未定：要不要让 `KnownAgent` 也认 adapter 声明的 skills 目录（统一），还是保持「显式 `--target` 只认上游表」 | `internal/skills/paths.go`、`internal/skills/mesh.go` |

---

## 怎么用

- **A**：勾选项，我落进对应文档。
- **B**：还没查的配置面（B1–B3），我可以并行查。
- **C**：已执行到 2026-09-13 第三轮；**C1–C10 全部已落代码并测试**。
