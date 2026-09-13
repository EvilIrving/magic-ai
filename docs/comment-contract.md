# 注释契约（对内）

更新：2026-09-13。约束**代码注释**必须承担什么，以及外部事实的来源写在哪里。
产品边界与数据方向见 [../AGENTS.md](../AGENTS.md)；本文件只管注释。

## 为什么注释是交付物的一部分

Magic 的实现难度不在逻辑，在**事实**：每个 harness 的磁盘路径、MCP 字段名、
provider base URL、hook 事件名，都是别人的产品行为。这些东西：

- 错了 → 往用户已装的工具里写出无效文件；
- 变了（对方改格式）→ 必须知道当初是照什么写的，才知道该去哪儿复核；
- 看起来"多余"（一份 `mcpServers` 而已）→ 恰恰是下一个人最容易顺手"统一掉"的地方。

所以来源**不能只活在文档里**。`docs/harness-profiles.md` 记的是"我们调研过"，
但代码被改的那一刻，改动者手上只有代码。来源必须落在定义点上。

## 注释的三种职责

一条注释只需要属于其中一种。不需要每条都有，密度由改动者判断；
但**第 1 类在"外部事实定义点"是强制的**（见下节）。

### 1. 来源（provenance）— 强制

格式固定，便于 `rg "Source:"` 一次列出全仓外来事实：

```go
// Source: <kind> | <locator> | checked <YYYY-MM-DD> | <confidence>
```

`kind` 取值：

| kind | 什么时候用 | locator 写什么 |
|---|---|---|
| `upstream` | 照搬别的项目的数据/表/文件 | repo 路径或 URL（`github.com/vercel-labs/skills` → README） |
| `vendor-doc` | 对方官方文档写明的格式 | 真实可访问的文档 URL |
| `vendor-api` | 对方接口实际返回的字段形状 | 接口 URL + 观察到的字段 |
| `local-measure` | 本机装了对方，实测出来的路径/字段 | 工具 + 实测路径（`~/.grok/config.toml`）+ 日期 |
| `internal` | Magic 自己的决定，不是外部事实 | `docs/xxx.md#节` 或 `D<n>` 决定号 |

`confidence` 取值：`verified`（文档/上游原文对照过）· `measured`（本机或真实环境跑出来）·
`inferred`（推断，**必须写清怎么确认**）· `stale?`（当时确认过，如今可能已变，改前先复核）。

规则：

1. **一张表 / 一组常量放一条 `Source:`**；某几行来源与整块不同，才在行上单独标。
2. **不写没核实过的 URL。** 不知道来源就标 `inferred` 或 `stale?`，
   并写出确认方式（"看 `~/.grok/config.toml` 是否仍用 `mcp_servers`"）。
   假 URL 比没有 URL 更坏——它会让人以为已经查过。
3. **偏离对方格式要标 `Magic deviation:`**，一句话说为什么（例：键名不动但值写成 Magic 的目录）。
4. **改一个带 `Source:` 的值**，要么更新 locator + `checked` 日期，要么保留原值并说明理由。
   不许悄悄改掉。
5. `local-measure` 只在真的存在该文件时才算证据；换机器后不成立就改标 `stale?`。

### 2. 逻辑解释（why，不是 what）

只在代码本身说不清的时候写：为什么分两步、为什么不能整文件覆盖、为什么这个顺序。
复述代码的注释（`// set the path`）是噪音，直接删。

### 3. 开发约束（never / invariant）

把"以后别再这么干"写死在会被改到的那个函数上，而不是只写在文档里。
关键词固定，便于 grep：

```go
// Invariant: ...   永不成立的性质，破坏它会写出坏文件
// Never: ...       明确禁止的做法 + 原因
// TODO(FUTURE): ... 没做的能力；当前实现不假装已有
```

例：`Never: delete real directories under a managed skills path`（AGENTS.md 已定死的行为，
落在 `internal/skills/mesh.go` 的删除路径上，而不只在文档里）。

## 哪些是"外部事实定义点"（`Source:` 强制）

| 位置 | 外部事实 |
|---|---|
| `internal/adapters/*` | 各 harness 配置根、skills/agents 目录 |
| `internal/skills/paths.go` | 全量 skills 路径表 |
| `internal/mcp/target.go` | MCP 配置文件路径 + 顶层键 |
| `internal/mcp/translate.go` | 各家 MCP 方言字段（`type`/`url`/`httpUrl`/`serverUrl`/`command` 数组…） |
| `internal/provider/*.go` | SwitchWriter 写的键名、marker、`env_key` 机制 |
| `internal/catalog/*` | provider 方言投影出来的字段 |
| `internal/modelsdev/*` `internal/modelfile/*` `internal/openrouter/*` | 上游接口 URL、响应字段、provider 默认 base URL |
| `internal/sync/*` | 对象存储环境变量名、endpoint 形状 |
| `internal/secrets/*` | 文件权限约定（0600）等 OS 层事实 |
| `internal/provider/builtin/providers.yaml` | 每家官方 base URL / 协议 |

不需要 `Source:` 的：纯 Magic 内部数据结构、CLI 文案、测试脚手架
（但**决定**类注释仍建议带 `D<n>`，例：`// See docs/decisions.md D5`）。

## 文档与代码的分工

- **代码**：事实 + 来源 + 不可破坏的性质。改动者只看代码也能知道这条数据从哪来、能不能动。
- **docs/**：理由、对比、覆盖面、决定过程（为什么覆盖这 13 家、为什么 B 方案）。
- 两边冲突时：代码里的 `Source:` 是**当前实现依据**，docs 是**为什么这样决定**。
  发现 docs 与代码事实不一致，当作 bug 处理，不要任选一边沉默。

## 检查

```bash
./scripts/provenance.sh          # 覆盖报告：定义点有/缺 Source
rg -n "Source: .* (inferred|stale\?)" internal cmd   # 待复核清单
rg -n "Magic deviation:|Never:|Invariant:" internal cmd
```

`provenance.sh` 是报告，不是 CI 门禁——新 harness 允许先标 `stale?` 再补文档，
但**不允许**没有 `Source:` 行。
