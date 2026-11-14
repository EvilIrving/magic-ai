# 架构

magic 把配置库 `~/.magic-ai` 编译成每个 AI coding harness 自己的配置文件。它不代理请求、不改运行时、不安装工具、不同步会话。

产品边界、数据方向与已锁定的决定以 [`AGENTS.md`](../AGENTS.md) 和 [`decisions.md`](./decisions.md) 为准。本文只写实现结构：代码怎么分层、数据流经哪里、哪些东西不能破。

## 分层

| 层 | 包 | 职责 | 不依赖 |
|---|---|---|---|
| 命令 | `cmd/magic` | cobra 命令、参数、终端输出 | 不写业务逻辑 |
| 核心 | `internal/core` | `plan`、`apply`、合并规则、原子写、备份、库路径 | 不打印、不读 `os.Args` |
| 契约 | `internal/adapter` | `Adapter` 接口、`Caps`、`Drift`、`Result` | 不依赖任何 harness |
| 方言 | `internal/adapters/<id>` | 一个 harness 的配置根、各资产目录、能力声明 | 不读写文件 |
| 资产 | `internal/{skills,agents,mcp,provider,secrets,catalog,registry,sync}` | skill 目录、sub-agent、MCP 配置翻译、provider 切换、密钥、模型目录、云同步 | 不感知 CLI |
| 备份清单 | `internal/backup` | `backups/<stamp>/manifest.json`：apply 写了什么、原内容在哪、内容指纹（`before`/`after`）是多少、rollback 该恢复还是删除 | 不依赖 `internal/core`（core 与 mcp 都写清单，叶子包避免反向依赖） |

依赖方向只有一条：`cmd/magic` → `internal/core` → `internal/adapter` 与资产包。`internal/core` 的代码里不出现终端输出，加 TUI 或 GUI 时不需要重写核心。`internal/backup` 是叶子包：`internal/core` 与 `internal/mcp` 都写同一份清单。

## 一次 apply 的数据流

1. 从配置库读 `magic.yaml`、`providers.yaml`、`secrets/providers.json`、`packs/`。
2. 每个 adapter 用自己的 `MirrorRoot` 与 `LibraryMirrorRel` 算出目标目录和库内目录。
3. 每个文件用 `projectBytes` 决定写入内容：目标不存在则写库内容；结构化 JSON 深合并；`config.toml` 文本合并；其余原样覆盖。
4. 写入前先在 `backups/<时间戳>/manifest.json` 记一条：目标已存在则先复制到 `backups/<时间戳>/<adapter>/<相对路径>` 并记为 `update`（`before` = 副本内容的指纹），目标不存在则记为 `create`（rollback 时删除）。两条都记 `after` = apply 即将写入内容的指纹。然后原子写（临时文件 + rename）。内容与权限都与库一致的文件跳过写入，不碰 mtime。
5. claude 额外把 `packs/harness/claude/claude.json.partial.json` 的顶层键合并进 `~/.claude.json`。
6. provider 层（`core.ApplyProviders`）：claude 把 provider env 合并进 `~/.claude/settings.json`；pi 把 provider 条目合并进 `~/.pi/agent/models.json` 并另写 `models.magic.json`；opencode 合并进 `~/.config/opencode/opencode.json`，workbuddy 合并进 `~/.codebuddy/models.json`。四家都走 `SwitchWriter.Files` + `provider.ApplyFiles`，与 plan 同一份计算。codex / grok 有 `Files`（`magic provider use codex|grok` 用），`apply` 不写它们的主配置。
7. MCP 层（`mcp.Apply`）：`packs/mcp/mcp.yaml` 按每家方言译成 `mcpServers` / `mcp` / `[mcp_servers]`，合并进目标文件后写盘（JSON / TOML / YAML 三种纯变换，`mcp.ProjectBytes`）。
8. skills 与 sub-agents 走目录投影，和上面按文件写入是两段独立的动作。

## 不变量

- **I1 库是唯一真相源。** 不做 capture，不从 harness 目录反向读取配置。
- **I2 只写配置。** 不安装工具、不改 PATH、不代理流量、不改请求。
- **I3 非破坏合并。** 库没有定义的键和行一律保留（`decisions.md` D5）。`plan` 里的 `keep` 就是这条的可见形式。
- **I4 写前备份、写时原子，且回滚可完整。** 任何覆盖先备份；写入用临时文件加 rename。每次 apply 写 `manifest.json`：`update` 条目可恢复，`create` 条目可删除——没有清单的旧备份只能恢复文件，不能删新建。回滚不丢用户的新改动：目标内容指纹既不是 `after`（apply 写的）也不是 `before`（apply 换掉的）时跳过并报告，`--force` 才按清单执行；目标越出 `UserHome`、备份路径越出备份集的条目一律不执行（`--force` 也不执行）。
- **I5 plan 与 apply 共用同一份写入计算。** 镜像层两者都调用 `projectBytes`，MCP 层两者都调用 `mcp.ProjectBytes` / `mcp.MergeProjection`，provider 层两者都调用该 harness 的 `SwitchWriter.Files`（`provider.ApplyFiles` 落盘）。plan 不可能描述一个 apply 不会做的写入。plan 按 apply 的顺序叠层（镜像 → claude partial → provider → mcp），后置层合并进前一层的字节，`after` 就是 apply 之后的目标内容。
- **I6 plan 不含明文。** 内容只以 `sha256:<前 12 位>` 形式出现，密钥、token 不会进 JSON、日志或提交。
- **I7 库结构变更必须带迁移。** `magic.yaml` 的 `schema_version` 记录库结构版本，当前为 `core.LibrarySchemaVersion = 1`。旧库（字段缺失即 0）由 `magic migrate` 升级：用 `yaml.Node` 只改 `schema_version`，保留其它字段、字段顺序与注释，写盘前把 `magic.yaml` 备份进 `backups/<stamp>/`；已是当前版本时幂等、不写盘。`apply` / `plan` / `verify` / `doctor` 遇到旧库不静默写入，提示先 migrate；库比程序新时一律拒绝写入，避免旧程序改坏新库。

## Adapter 契约

```go
type Adapter interface {
	ID() ID
	Name() string
	Platforms() []string
	Implemented() bool
	Caps() Caps
	Detect(home string) bool
	SkillsDir(home string) string
	AgentsDir(home string) string
	MirrorRoot(home string) string
	LibraryMirrorRel() string
}
```

- `Caps` 声明这个 harness 支持哪些投影（skills、sub-agents、MCP、hooks、models、themes）。声明为 `false` 的投影不会被写入。
- `Implemented() == false` 的 adapter 只能出现在 `status` 和 `plan` 的「跳过」里，永远不写文件。
- `MirrorRoot` 是这个 harness 的配置根；`LibraryMirrorRel` 固定为 `packs/harness/<id>`。
- adapter 只做路径与能力声明，不读写文件。写入决策留在 `internal/core`。

## 四个动作的现状

| 动作 | 做什么 | 状态 |
|---|---|---|
| `plan` | 只读，算出每个目标文件会变成什么，输出 create / update / delete / unchanged 与 key、行级差异 | 已实现（`internal/core/plan.go`，`magic plan`）。四层按 apply 顺序叠加，`reason` 列出参与的层；provider 层调 `SwitchWriter.Files`，与 apply 同一份计算 |
| `apply` | 按 plan 描述的写入执行：备份、原子写、claude partial 合并 | 已实现 |
| `verify` | 报告库与目标之间的漂移 | 已实现（`magic verify`，`internal/core/plan.go` 之上）。空检查（`checked == 0`）与被跳过的 adapter 都会让退出码为 1；create / update / delete 都算漂移 |
| `rollback` | 按 `manifest.json` 完整回滚一次 apply | 已实现（`magic rollback`，`internal/core/rollback.go`）：`update` 恢复、`create` 删除；默认只处理仍持有 `after` 指纹的目标，用户改过的逐条报告跳过（`--force` 才覆盖/删除）；目标越出 `UserHome` 或备份越出备份集的条目一律不执行（`--force` 也不执行，单独归为「拒绝」，与可加 `--force` 的「跳过」分开报）；无清单的旧备份回退到「只恢复文件」并明确标注。汇总行始终写明实际使用的备份 stamp。`backups/<harness>/` 属于 provider 层，不参与回滚 |

### 覆盖边界

`plan` / `verify` 覆盖的是 apply 按文件写入的那四层：harness 镜像、claude partial（`~/.claude.json`）、provider 投影（claude / pi / opencode / workbuddy）、MCP 投影。镜像目录缺失不会让 provider / MCP 层跟着被跳过：库内没有 `packs/harness/codex` 时，plan 仍报出 MCP 写 `~/.codex/config.toml` 的那条 change；两件事都成立（镜像缺失、该 adapter 又没有 provider 投影）时，「跳过」文案会同时写明两个事实，免得读成「放一个 `packs/harness/codex` 就能让 apply 管 codex」。

不覆盖的部分：

- skills 与 sub-agents 的目录投影（链接、复制、冲突检测）。它们由 `magic skills status` / `magic agents` 报告。
- provider 的 codex / grok 投影。那两家的主配置只有 `magic provider use codex|grok` 会写，`apply` 不写，plan / verify 因此也不为它们算这一层（代码里的唯一边界是 `providerProjectionHarnesses`）。

## 不做什么

产品边界见 [`AGENTS.md`](../AGENTS.md) 的「不做」一节。落到代码结构上，本文额外约束两条：

- 不引入请求代理层。没有网关、没有协议翻译、没有账号与额度调度。这些属于运行时的流量层，magic 只产出配置文件。
- 不让核心依赖终端。`internal/core` 里不出现打印与参数解析，将来加 TUI 或 GUI 时复用同一份核心。

## 文档索引

| 文档 | 内容 |
|---|---|
| [`architecture.md`](./architecture.md) | 代码分层、数据流、不变量、adapter 契约 |
| [`plan.md`](./plan.md) | plan 的 JSON 结构与样例 |
| [`usage.md`](./usage.md) | 每个命令的输入、输出与产出的文件 |
| [`decisions.md`](./decisions.md) | 已经定下的设计决定 |
| [`library-layout.md`](./library-layout.md) | `~/.magic-ai` 的目录 |
| [`harness-matrix.md`](./harness-matrix.md) | 每个 harness 的路径与能力 |
| [`harness-profiles.md`](./harness-profiles.md) | 每个 harness 的配置结构调研与核实日期 |
