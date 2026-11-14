# Magic NOTES

> 历史变更日志（**不是**需求真相源）。**2026-09-10 及更早的条目里，capture / 本地加密 secrets /
> `$KEY_ENV` 占位已被推翻**。当前口径：本地 `secrets/providers.json` 明文；`.enc` **只是云端状态**；
> Pi / WorkBuddy / OpenCode / Claude 投影写真值 key。见 `docs/decisions.md` D2/D3、`docs/secrets-and-sync.md`。
> 下面旧条目里的 `providers.json.enc` / `MAGIC_PASSPHRASE` for copy 是当时实现，不是现行产品。

## 新增 5 家 provider writer + 4 家按证据判为 stub (2026-11-14)

- **新 writer**：`kimi-code`（`config.toml` `[providers.magic-*]` + `[models."magic-*/pick"]` + 根 `default_model`）、
  `vibe`（`config.toml` `[[providers]]`/`[[models]]` managed 注释块 + 根 `active_model`）、
  `hermes`（`config.yaml` `model:` 块，`provider: custom` + `api_mode`）、
  `omp`（`models.yml` `providers.magic-*`，复用 `catalog.BuildPiProviderEntry` 的方言对象）、
  `droid`（`settings.json` `customModels[]`）。全部注册进 `switch.go` 的 `init()` 与 `Writers()` 顺序表。
- **路径复用 adapter**：五家的 `ConfigPath` 只做 `filepath.Join(<adapter>.New().MirrorRoot(home), <文件名>)`，
  `KIMI_CODE_HOME` / `VIBE_HOME` / `HERMES_HOME` / `PI_CONFIG_DIR` / `PI_CODING_AGENT_DIR` 跟着 adapter 走，不另立一套路径表。
- **`none` 要能还原，不只是删**：`hermes` 的 `model:` 块和 `kimi-code`/`vibe` 的默认模型指针都是用户自己的字段，
  删掉就等于改用户配置。新增 `internal/provider/projstate.go`（`magic-provider-state.json` sidecar，RoleSidecar）记录
  Magic 顶掉的字段原值，`provider use none` 按它还原并删除 sidecar；sidecar 内容不依赖时钟，plan 与 apply 算出同一份字节。
- **四种 stub 的理由落进代码**：`internal/provider/stubdialects.go` 用 `Source:` 块逐家写清缺什么——
  `qoder`（厂商文档明说不要手写 settings.json BYOK）、`zcode`（`provider_config.json` 无公开格式）、
  `cline`（凭据与选中态在 `data/settings/providers.json`，其中 `updatedAt` 是 harness 自己的时间戳，
  Magic 无法在不引入时钟的前提下写）、`mmx`（`custom_provider` 的 `api` 取值无文档，且 adapter 记了
  `Never: rewrite config.yaml`）。
- **测试**：`files_test.go` 三张表各加 5 行（路径 / 种子 / 协议），新增 `TestFilesClearLeavesNoMagicResidue`
  （provider 模式写入 → `none` → 配置文件里 `magic` 计数 0、用户自有键仍在、sidecar 已删）；
  `switch_test.go` 的 `SupportsOfficial` 表补上 kimi-code / vibe / hermes / omp（第一方路径是交互式登录，不是可写文件）。
- **e2e**（临时 `MAGIC_HOME` + 临时 `HOME`，`deepseek`）：五家 `provider use` / `none` 均符合预期，
  `none` 后 `grep -ci magic` 全为 0；四家 stub 在 `provider use` 处直接报 `not implemented`。
- **未做**：`ApplyProviders` 白名单未动（`apply` 仍只投影 claude / pi / opencode / workbuddy）；
  `droid` 没有把 Magic 条目设成默认模型——文档没说 `settings.model` 如何引用 `customModels` 条目，不猜。

## plan 的跳过文案 + plan / verify / rollback 使用文档 (2026-11-17)

- **跳过文案不再误导**：`buildAdapterPlan` 里库内没有镜像目录时，`Skipped` 原先只写「配置库里没有镜像目录 <path>」，容易被读成「放一个 `packs/harness/codex` 就能让 apply 管 codex」。现在由 `nothingToWriteReason` 把决定这次写入的两个事实都写出来：镜像目录/镜像根缺失，加上「apply 不投影该 adapter 的 provider（只有 `magic provider use` 会写）」（判据仍是 `providerProjectionHarnesses`）。apply 会投影 provider 的 harness（claude / pi / opencode / workbuddy）文案不变。
- **镜像目录缺失不再提前返回**：`mirrorRoot == ""` 原先直接 return skipped，把 provider / MCP 层的计算也一起跳过。现在只有镜像层依赖库内目录，拿不到镜像树只记一个 `mirrorProblem`，provider 与 MCP 层照常叠层：库内没有 `packs/harness/codex`、`~/.codex/config.toml` 存在、`packs/mcp/mcp.yaml` 有内容时，plan 报 `codex config.toml update / mcp 投影 / 行 +N -0`，而不是整个 adapter skipped。
- **测试**：`internal/core/plan_layers_test.go` 新增 `TestPlanCodexMCPWithoutMirror`（无镜像目录仍报 MCP 的 `config.toml` change，且与 apply 落盘指纹一致）、`TestPlanNoMirrorRootStillProjectsMCP`（`MirrorRoot` 为空的 adapter 也不再提前 return）、`TestPlanSkipReasonNamesProviderProjectionGap`（codex 的跳过文案含 `magic provider use`，pi 不含）。
- **文档**：`docs/usage.md` 新增 §12 `magic plan`、§13 `magic verify`、§14 `magic rollback`，按「命令 → 输入 → 终端输出 → 生成的文件」体例，输出均为临时库 + 临时 home 的实测结果；`docs/architecture.md` 补「覆盖边界」（plan/verify 覆盖镜像、claude partial、provider 投影 claude/pi/opencode/workbuddy、MCP 投影；不覆盖 skills/sub-agents 目录投影与 codex/grok 的 provider 投影），并修正 provider 层数据流里「opencode / workbuddy 不在这条可预测链上」的旧描述；`docs/plan.md` 的 `skipped` 说明补上 provider 投影这一事实，`delete` 行写明 `~/.codex/models.json`；`README.md` / `README.zh-CN.md` 命令表各加 plan / verify / rollback 三行。
- **未修（已上报）**：同一个文件在一次 apply 里被两层写（镜像 + provider / MCP，或 provider + MCP，如 `~/.codex/config.toml`、`~/.claude/settings.json`）时，`backup.Record` 对同一目标只保留最早那条、`after` 记的是中间态，于是干净 apply 之后 rollback 会把该文件判成「目标在 apply 之后被改动」而跳过。不在本轮范围。

## provider 投影统一成一套纯函数 (2026-11-16)

- **两套实现合并**：`magic apply` 走的 `core.applyClaudeProvider` / `applyPiProvider` / `applySwitchProvider` 与 `magic provider use` 走的 `SwitchWriter.Apply` 是两套逻辑、写同一批文件。接口改成 `Files(ctx, req) ([]FileWrite, error)`（纯函数：算出这次投影要写的文件，不落盘），六个 writer 各自实现 `filesClaude` / `filesCodex` / `filesGrok` / `filesPi` / `filesOpenCode` / `filesWorkBuddy`，写盘与备份统一到 `provider.ApplyFiles`：按 `internal/backup` 记 manifest（create/update + before/after 指纹）→ 备份 → 写盘。
- **不写相同内容**：`needsWrite` / `writeIfChanged` 抽成叶子包 `internal/writefile`（`NeedsWrite` / `Write` / `Atomic`），core 与 provider 共用；内容与权限都一致的目标不写（不 rename、不改 mtime）。provider 层因此也进了 `backups/<stamp>/manifest.json`，rollback 能还原 apply 写过的 provider 文件。
- **plan 用同一份计算**：`internal/core/plan.go` 不再自己算 claude / pi 的 provider 层，改成 `provider.Writer(id)` + `Files`，并把 `Context.ReadFile` 指向 plan 的叠层视图，让 provider 层合并在前一层输出之上。旧的「plan 算不出的投影」字段与输出整体删除：opencode / workbuddy 的切换现在就是普通的 provider 层（`opencode.json` 与 MCP 投影叠在同一文件上，plan 给得出 `after`）。
- **删除也是投影**：`FileWrite` 加 `Remove`（`Data` 必须为 nil），codex 官方模式用它删掉 Magic 管理的 `~/.codex/models.json`；`ApplyFiles` 先按 update 备份再删，plan 的 `action` 新增 `delete`、`verify` 把 delete 当漂移。
- **键差异基准改成磁盘**：`Change.Keys` 比较「磁盘现状 vs 最终字节」，`unchanged` 的文件不再列出 `overwrite env.*`（旧实现拿层输入比层输出，apply 之后再 plan 会假装要覆盖自己刚写的键）。
- **apply 的写集不变**：`ApplyProviders` 仍只投影 claude / pi / opencode / workbuddy。codex / grok 有 `Files`（`magic provider use codex|grok` 用），但 apply 不写它们的主配置——这条在代码里有注释，避免下一批误以为漏了。
- 文档：`docs/plan.md`（删掉「算不出的投影」一节、`action` 与 key op 的基准、已重新生成样例）、`docs/architecture.md`。
- 测试：`internal/provider`（六家 `Files` 与落盘逐字节一致、重复投影不写、官方/清空模式、pi sidecar、codex 删除 + manifest 备份）、`internal/core`（unchanged 无 key、delete action、grep 断言旧字眼彻底消失、opencode provider 层随 apply 叠层）、`cmd/magic`（`apply --only opencode` → `verify --only opencode` 0 漂移）。

## plan/verify 覆盖 provider 与 MCP 投影 (2026-11-15)

- **plan 按 apply 的真实顺序叠层**：镜像 → claude partial → provider → mcp。每个目标文件维护「磁盘现状」与「当前模拟字节」，后置层在前一层的输出上继续合并（`internal/core/plan.go` 的 `planFiles`），所以 `after` 就是 apply 之后磁盘上的内容；同一文件被多层写时只有一个 change，`reason` 用 ` → ` 连出参与的层（如 `镜像 JSON 合并 → provider 投影 env`）。只由后置层写出的文件（`models.magic.json`、provider / MCP 写到的配置）也是 change，`rel` 用相对 user home 的路径。
- **mcp 抽出纯函数**：`mcp.ProjectBytes(t, userHome, f)`（读目标 → 按 Format 合并）与 `mcp.MergeProjection(t, path, f, existing)`（合并进给定字节，供 plan 叠层）。`Project` 改成调 `ProjectBytes` 再写盘，备份 / manifest / 写盘逻辑未变，`translate.go` 与各 dialect 未动。
- **provider 层进 plan**：claude 的 `settings.json` env 由 plan 自己算（`provider.BuildClaudeEnvFromProvider` + `catalog.UpsertClaudeSettingsJSON`，第四批改成各个 `SwitchWriter.Files`），pi 的 `models.json` / `models.magic.json` 用 `catalog.BuildPiProviderEntry` + `catalog.MergePiModelsJSON`；解析顺序、`--only` 与 Detect 的入口条件、密钥缺失即跳过，都与 `ApplyProviders` 一致。密钥只以指纹与键名出现（`env.<KEY>` 的 add/overwrite）。
- **当时无法预测的投影单独报告**：opencode / workbuddy 的 `provider.Use` 切换当时被列进一个「plan 算不出的投影」清单（`adapter` + `reason`），既不算失败也不算已检查；switch 改写的那份文件（opencode 的 `opencode.json`，同时是 MCP 目标）从 plan 里剔除。**2026-11-16 已删除该机制**：两边合并成 `SwitchWriter.Files` 后它可以预测。
- **顺手修 rollback 输出**：`core.Rollback` 返回 `RollbackResult{Stamp, Restored, Skipped, Refused}`，全部条目被跳过时汇总行写实际使用的 stamp（不再显示「备份 -」）；目标越界 / 备份越界 / 清单缺路径 / 未知 action 归为「拒绝」，单独一行「`--force` 也不执行」。
- 文档：`docs/plan.md`（覆盖范围、reason、已知限制、重新生成的样例）、`docs/architecture.md`（apply 数据流补 provider / MCP 层、I5、plan/verify/rollback 现状）。
- 测试：`internal/mcp`（`ProjectBytes` 与 `Project` 逐字节一致、lenient 不写、`MergeProjection` 用给定字节）；`internal/core`（claude / pi provider、mcp、四层叠加、codex TOML + mcp、plan 与 apply 指纹一致、rollback 拒绝桶与 stamp）；`cmd/magic`（apply → verify 0 漂移、删 provider env 键 / 改脏 sidecar / 改脏 server 块报差异、rollback 汇总 stamp 与拒绝分类）。`docs/plan.example.json` 已重新生成（现在含 provider 与 MCP 叠层）。

## rollback 数据安全加固 (2026-11-14)

- **manifest 记内容指纹**：`backup.Entry` 新增 `before`（apply 写入前目标的 sha256，create 条目为空；由 `backup.Record` 从它复制的副本算出）与 `after`（apply 写入后内容的 sha256），格式与 plan 的 `sha256:<前 12 位>` 一致。哈希实现只在 `internal/backup/fingerprint.go`（`backup.Fingerprint` / `FingerprintOf`），`core.Fingerprint` 改为该类型的别名、`core.FingerprintOf` 转调。`manifest.json` 版本仍是 1，新字段 `omitempty`。
- **rollback 不覆盖用户后续改动**：`update` 只在目标指纹等于 `after`（或等于 `before`，即已被手动还原）或目标不存在时恢复；`create` 只在目标指纹等于 `after` 或目标已不存在时删除；否则跳过并逐条报「目标在 apply 之后被改动，跳过（加 --force 覆盖）」。旧 manifest 没有指纹 → 跳过并提示需要 `--force`。
- **`--force`**：跳过内容校验，按 manifest 执行；路径范围校验仍然生效。
- **路径范围校验**：`Target` 必须是 `UserHome` 之下的绝对路径，`Backup` 必须落在备份集目录内，不满足的条目跳过并报告（`--force` 也不执行）。
- 测试：`internal/core/rollback_test.go` 新增指纹记录、create/update 用户改动保护、`--force`、未被改动正常恢复、`Target` 越界、`Backup` 越界、旧 manifest 七组用例。

## 备份清单 + 完整回滚 + plan/verify 校正 (2026-11-13)

- **备份清单**：新增叶子包 `internal/backup`，apply 每次写入前在 `backups/<stamp>/manifest.json` 记一条（`adapter`/`rel`/`target`/`backup`/`action`/`mode`）。三处写入方都写：`core.ApplyAdapter`、`core.applyClaudePartial`、`mcp.Apply`。`action=create`（目标原本不存在）与 `action=update`（有备份）分开记；同一目标每次 apply 只记一次，保留最早的、apply 之前的副本。
- **rollback 按清单还原**：`update` 从备份恢复，`create` 删除目标；`--dry-run` 只打印。没有 `manifest.json` 的旧备份走原逻辑，并明确标注「旧备份，只恢复文件不删除新建」。
- **备份目录命名空间**：`ListBackups` 只把能按 `20060102T150405Z` 解析的目录当作 apply 备份集；`backups/<harness>/`（provider 层）由 `ListProviderBackupDirs` 单独报出，`rollback --list` 标「provider 备份，不参与回滚」。
- **MCP 备份保留子目录**：`mcp.Project` 的备份 rel 改成相对 adapter 镜像根的路径（如 `kiro/settings/mcp.json`）；`~/.claude.json` 统一用 `_claude.json`（`backup.ClaudeJSONRel`，与 apply 一致）；旧备份里的 `claude/.claude.json` 也映射回 `~/.claude.json`，不再落到 `~/.claude/.claude.json`。
- **claude partial 改深合并**：`applyClaudePartial` 与 plan 共用 `claudePartialProjected`，语义为 `deepMergeMap(existing, partial)`（D5）：`mcpServers` 等嵌套对象里用户条目保留，库只在同名键路径上赢。目标 `~/.claude.json` 解析失败时报错而不是覆盖。
- **plan reason 与实际策略一致**：`projectBytes` 返回实际策略（create / json-merge / json-fallback / toml-merge / whole-file），`buildChange` 据此写 `reason` 与 `keys`/`lines`；`backup` 只为 `update` 置 true。
- **verify 不放行空检查**：`checked == 0` 或存在被跳过的 adapter 时退出码为 1，并列出被跳过的 adapter 及原因。
- **unchanged 名副其实**：`writeIfChanged` 在内容与权限都一致时跳过写入（不 rename、不改 mtime）；只差权限时只 `chmod`。
- **删死代码**：删 `mergeJSONObjects`（测试改用 `projectBytes`）；删 `core.BackupPath`（被 `backup.Record` 取代）；删 `plan_golden_test.go` 里未使用的 Plan 反序列化变量。
- 测试：`internal/core`（清单生成、create 删除、update 恢复、旧备份回退、provider 目录不算 stamp、partial 深合并、unchanged 不写盘、reason 一致）、`internal/mcp`（清单子目录 / create 条目 / claude 名字）、`cmd/magic`（verify 空检查、跳过、漂移）。`docs/plan.example.json` 已重新生成。

## 排行榜全量调研 + 第三批 adapter (2026-09-13)

- 用户榜单第 2–30 名逐家调研（官方文档 + 源码），产出 `docs/harness-rollout-plan.md`（含两轮自审、能力矩阵、实施批次）。
- 新增 adapter：`omp`（Oh My Pi，`~/.omp/agent`）、`hermes`（`~/.hermes/config.yaml`）、`mimocode`（MiMo Code，`~/.config/mimocode`）、`dimagent`（`~/.dim/v2`）、`openclaw`（`~/.openclaw`）。
- MCP：新增 `FormatYAML`（`internal/mcp/yaml.go`，`yaml.Node` 级合并保留注释/其它键）+ 4 个 dialect：`omp`、`hermes`、`dimagent`（sse→n/a）、`openclaw`（嵌套 `mcp.servers`）。
- `DefaultAgents` 增 `omp/hermes/mimocode/openclaw`；skills 表增 `omp`/`oh-my-pi`。
- 决定 D21–D25：Roo Code 停运不做；Alma/MiniMax-MCP/Lingma UI-only；DSH MCP 暂缓；Cola 身份待确认。
- 修复工作区里 `internal/provider/switch_test.go` 的编译错误（`&catalogFileForTest(t)` → `catalogFileForTest(t)`，untracked WIP 笔误）。
- 测试：`internal/mcp` 新 dialect + YAML/嵌套键用例；`go test ./...` 全绿；临时 HOME 下 `magic apply` 端到端验证 5 个新 target。

## provider switch 语义修正：official vs none (2026-09-13)

- 读官方文档后修正（Pi https://pi.dev/docs/latest/providers，OpenCode /docs/providers + models.dev）。
- `SwitchWriter` 新增 `SupportsOfficial() bool`：claude/codex/grok/workbuddy = true；**pi/opencode = false**（多 provider 聚合器）。
- 新目标 `none`（别名 off/clear）→ 对所有 harness 都表示「清掉 Magic 写进配置的条目」。
  `provider use official --harness pi|opencode` 报错并提示用 `none`；无 `--harness` 时对这两家记 `n/a` 而非失败。
- 顺便核实 OpenCode 字段：官方 JSON Schema 确认 `provider.<id>.{npm,name,whitelist,blacklist,options.{apiKey,baseURL,timeout},models}` 均合法（之前的 `whitelist` 写法没问题）。
- 决定 D21；AGENTS/usage/README/decisions 已同步。测试：`internal/provider/switch_test.go`。

## Provider 补齐 + MCP 第二轮调研/实现 (2026-09-13)

- 内置模板 `providers.yaml` +5：`google`(Gemini OpenAI 兼容)、`mistral`、`meta`(Llama API)、`upstage`、`hunyuan`(腾讯)，按模型用量榜补。
- MCP 新增 7 个 adapter + target：kiro、droid、zcode(嵌套 `mcp.servers`)、qwen-code、iflow-cli、kilo、zed。
- 新 dialect：kiro(`disabled`)、zcode(`enable`)、zed(stdio only)；target.TopKey 支持点号嵌套键（读写/检查均已）。
- 调研文档：`docs/mcp-design.md` 新增「第二轮调研（2026-09-13）」，列出全部 harness 的 MCP 格式 + 状态。
  未实现（需新 writer）：Hermes/Continue/Goose（YAML）、Mistral Vibe（TOML 数组表）、OpenClaw（JSON5）、Crush（待实测）、MiniMax/Lingma/Amp/Cortex（UI/CLI only）。
- 测试：`internal/registry`（P1+调研批次）、`internal/mcp`（新 dialect/target/嵌套键）；`go test ./...` 全绿。

## 用量驱动的 v1 覆盖：P1 补齐 (2026-09-13)

- 依据 OpenRouter app 榜（2026-09）定 v1 范围（D20，修订 D9 的「全做」）。
- 新增 adapter：`cursor`（升为完整，含 MCP/hooks）、`antigravity-cli`、`kimi-code`、`qoder`、`qoder-cn`、`cline`（+ `internal/adapters/*`，注册到 `registry`、写入 `magic.yaml` 默认 enabled）。
- MCP `translate.go` 新增 dialect：cursor / antigravity(`serverUrl`) / kimi(`transport:sse`) / qoder / cline(`streamableHttp`)；`target.go` 落点从 10 → 16。
- `DefaultAgents` 改为按用量排序的 14 家；skills 路径表增 `kimi-code` 别名。
- P2（droid/kiro/minimax-code/zcode/hermes/openclaw/deepseek-harness/oh-my-pi/dimagent/alma/craftagent/cola）与长尾~50 家**推迟**；skill 路径已在表、`--target all` 可达。
- 测试：`internal/mcp` 新 dialect + target，`internal/registry` v1 core 完整性；`go test ./...` 13 包全绿。

## L2 skills 长尾表接入 (2026-09-13)

- `internal/skills/paths.go`：新增 `globalSkills` 表（vercel-labs/skills 全量，~74 家）、`KnownAgent` / `AllAgents`；`GlobalTargetDir` 改为查表；`AgentDotDir`（project）扩到产品集。
- `--target all` → `AllAgents()`；默认仍 8 家（避免给未安装工具造目录）。`windsurf`/`roo` 剔除（D13）。
- 测试：`internal/skills/paths_test.go`。

## MCP + hooks 投影落地 (2026-09-13)

- 新增 `internal/mcp`：规范形 `packs/mcp/mcp.yaml` → 各 harness 原生格式（JSON + TOML）。
  `translate.go`（claude/codex/grok/opencode/pi/gemini/copilot/workbuddy/trae/generic）、
  `toml.go`（`[mcp_servers.<name>]` 字段级 upsert）、`target.go`（10 个 harness 落点）、
  `apply.go`（merge，保留用户条目）、`inspect.go`。
- `magic apply` 新增 `## mcp projection`；新增 `magic mcp list|status|init`。
- `adapter.Caps` 新增 `Hooks`；claude/codex/grok/pi/workbuddy 等补上 `MCP:true`；`magic adapters` 加 HOOKS 列。
- hooks 按 B 方案（D19）：`packs/harness/<id>/hooks/…` 字段级合并（`merge.go` 白名单加 `hooks.json`/`mcp.json`/`mcp-config.json`）。
- 决定 D18（MCP：无前缀、按库名字管理、明文密钥） / D19（hooks：B 方案、路径不折 `~`、不写 trust）。A5–A8 已定，C10 已定。
- 测试：`internal/mcp` 翻译/TOML/投影合并；`go test ./...` 13 包全绿。

## `magic login` 落地（Bubble Tea TUI）(2026-09-13)

- 新增 `internal/provider/modelfetch.go`：`ModelsURLFor` / `FetchModels` / `ParseModelsResponse`（OpenAI `data[]`、Anthropic `display_name`、裸数组/字符串；去重排序）。
- 新增 `internal/login`：`LookupProvider` / `Resolve`（三级回退 live → 内置 pick → `registry/models.json` → 手动；`DefaultSelection`）/ `Apply`（用户优先合并，`pick` 替换）/ Bubble Tea TUI（masked key 输入、单选/多选、确认）。
- 新增 `cmd/magic/login.go`：`magic login [provider] [--api-key] [--pick] [--yes] [--no-live]`；非 TTY 强制 `--pick` + `--yes`，TTY 检测用 `go-isatty`。
- 依赖：`charmbracelet/bubbletea` + `bubbles` + `lipgloss`。
- 决定 D17（TUI 库、确认后写、回退顺序、不写 `models_meta`）；A3 已定。
- 测试：`internal/login`（13 个，含 fake `/models` httptest、TUI model 直接驱动）、`internal/provider` 解析。`go test ./...` 全绿。

## C 表落地：D2/D3/D5 实现跟上 (2026-09-13)

- **C1** `internal/secrets` 改为本地恒 `secrets/providers.json` 明文 0600；删 `MAGIC_SECRETS_MODE` / 双模式 / 本地 `.enc`。`sync push` 按「路径在 `secrets/` 或内容含真值」加密后上传 `<rel>.enc`；`sync pull` 解密 `.enc` 落回明文、无 `.enc`。口令仅 push/pull 需要。
- **C2** Pi / WorkBuddy 写**真值** apiKey（`catalog.BuildPiProviderEntry` 收真值；`provider.Use` 对 pi/workbuddy 也 needKey）。无 key 则 skip + 日志，不写空值。
- **C3** 删 adapter 级 `secrets/<id>.json.enc` reinject（`LoadSecrets` / `ReinjectBytes` / `${MAGIC_SECRET:}`）；apply 的 harness 定制文件本地即最终内容。
- **C7** `core/merge.go`：apply 写回结构化文件字段级合并（JSON deep merge / TOML section upsert，`[[array]]` 整段替换）；AGENTS.md 等整份覆盖；一文件一项自然不删其它项。
- **C4** `provider/builtin` 增 `Merge` / `MergeProvider` / `AddSelection`；`provider.LoadCatalog` 读取时叠加内置模板（用户优先）；新增 `magic provider add [ids…|--all|--list]`。
- **C8** `magic status` 只列已安装 harness（D12）。
- **C9** 新增 `magic guide`、`status --json`、`adapters --json`。桌面端 / 签名发行仍不在库内。
- 测试：`go test ./...` 全绿；新增 merge / builtin / sync / secrets 单测。

## 口径统一 (2026-09-13)

- D2：本地明文 `secrets/providers.json`；`.enc` 只表示云端对象（push 加密上传、pull 解密落回明文）。
- D3：Pi / WorkBuddy / OpenCode / Claude 写真值 key。实现差距只列 `docs/open-questions.md` C 表。

## Review fixes (2026-09-10)

- Redact: jsonc comments, `$KEY_ENV` skip, `key`/`access`/`refresh`/`*_token`; Pi deny `auth.json`; Claude partial redacts + reinjects.
- Apply: secret files 0600; skills skip unmanaged links; honor `adapters.enabled`; Claude apply uses `BuildClaudeEnvFromProvider`.
- R2: ListObjectsV2 pagination; reject `vault/../` escapes.
- WorkBuddy: union `availableModels`; Detect only `.codebuddy`.
- OpenCode SwitchWriter prefers existing jsonc and parses comments.

## Docs pass (2026-09-10)

- `docs/README.md` 文档目录；`getting-started.md` 改为中文对外入门（含当前 harness / R2 `magic-ai`）。
- `harness-matrix.md` 从调研草稿改为实现对照（不再写「建议/第二步」）。
- README / AGENTS 对齐：apply 投影 OpenCode+WorkBuddy、R2 桶约定、去掉失效的 `python-magic` / `/workspace/magic` 路径。

## Harness coverage (2026-09-10)

- Adapters implemented: gemini, copilot, workbuddy, trae (plus existing pi/grok/claude/codex/mcpuser/opencode).
- SwitchWriter: OpenCode (`opencode.json` provider.magic-*) + WorkBuddy (`~/.codebuddy/models.json` `$KEY_ENV`).
- **Dropped windsurf** (no product adapter / no skills target).
- Skills sync targets: + copilot, trae (global `~/.copilot/skills`, `~/.trae/skills`).
- `magic apply` also projects active provider into OpenCode / WorkBuddy when detected.

## Stub research + OpenCode adapter + docs (2026-09-10)

- Added `docs/harness-matrix.md` (real on-disk configs for stub harnesses + apply/SwitchWriter plan).
- Added `docs/getting-started.md` (external onboarding).
- OpenCode **Adapter implemented** (`~/.config/opencode`); skills sync target `opencode` added; auth.json under `~/.local/share/opencode` not captured by default.
- Removed `mcpuser` from provider SwitchWriter stub list (MCP ≠ model provider).
- R2：已单独建桶 `magic-ai`（前缀 `vault/`）。本机 `MAGIC_R2_*` 写在 local-environment `secrets.md`，不进本仓库。

## 1.0.0 formal (2026-09-09)

- Removed misleading `agents` adapter ("Central Agents"). Product language: **agents = sub-agents** under `packs/agents/`.
- Skills source of truth: `$MAGIC_HOME/packs/skills` (default `~/.magic-ai`).
- Default work dir remains `~/.magic-ai` (`MAGIC_HOME` / `MAGIC_VAULT`).
- Sub-agent projection: `packs/agents/<name>.md` → Caps.Agents harness dirs.
- Secrets: capture redact + apply reinject (passphrase); roundtrip tests.
- Skills apply: symlink → (Windows) junction → copy; never delete real dirs.
- Claude `~/.claude.json`: field-level extract → `claude.json.partial.json`.
- `magic catalog` / `magic copy <provider>` from `providers.yaml`.
- Implemented adapters: pi, grok, claude, codex, mcpuser, opencode, gemini, copilot, workbuddy, trae. No windsurf. No cursor product adapter (skills target only).

- Future UX (deferred): native Magic menubar/tray + global hotkey palette — not Raycast/Alfred.

## Skills UX
- `magic skills`: sync/status/list/doctor/clean/init/import/adopt/enable/disable.
- Global source: `$MAGIC_HOME/packs/skills` only (never ~/.agents/skills as Magic global central).
- Project source: `<project>/.agents/skills`; sync via `magic skills sync --scope project`.

## OpenRouter registry + providers copy (2026-09-09)

- `magic registry sync|search|show` — OpenRouter models cache under `$MAGIC_HOME/registry/`.
- `magic copy <provider>` — providers.yaml only; key from `secrets/providers.json.enc` via `key_env` + `MAGIC_PASSPHRASE` (redact / `--reveal-key`). No process env for copy keys.
- `magic secrets set|list` — manage provider key map in `secrets/providers.json.enc`.
- Removed harness product paths: `copy pi/...`, `copy grok/...`, `--harness` / `--file` / `--auth-file`; deleted `internal/copyfmt` + `testdata/copy-fixtures/`.
- `magic catalog pick <provider> <model-id>` appends pick + optional `models_meta`.
- Adapter `secrets/<id>.json.enc` reinject remains apply-only; provider map is separate.

## Multi-endpoint providers → Claude / Pi (2026-09-09)

- `providers.yaml`: `endpoints[]` (`id`/`base_url`/`protocol`) + optional root `active`; legacy top-level `base_url`+`protocol` still synthesizes one endpoint.
- Protocols: `openai-chat-completions`, `openai-responses`, `anthropic-messages` (+ aliases).
- `magic copy --protocol` / `--endpoint` selects which base_url/protocol to print; keys still secrets-only.
- `magic apply` projects active provider: Claude `~/.claude/settings.json` env (plaintext token, 0600); Pi `~/.pi/agent/models.json` with `apiKey: $KEY_ENV` (+ `models.magic.json` sidecar). Grok/Codex also covered by `magic provider use` SwitchWriters.

## Provider switch: official ↔ third-party (2026-09-09)

- Product loop for every Implemented harness: `official | <providerName>`; `company` is **only** an alias for `providers.active` / `MAGIC_ACTIVE_PROVIDER` (never a core type / never hardcodes shu).
- Layers: library (`providers.yaml` + `secrets/providers.json.enc`) → SwitchWriter dialect → thin CLI `magic provider *`.
- Implemented writers: Claude (`settings.json`), Codex (`config.toml` model_providers.<name>), Grok (`# magic-managed-catalog-*`), Pi (`models.json` magic-<name>).
- Stubs (opencode/gemini/…): same SwitchWriter interface, return not-implemented.
- Seed profiles (shu / shu-deepseek / shu-grok / deepseek / kimi / minimax) via `magic provider seed` — vendor URLs live in library seed data.
- Keys: `magic secrets set` / `magic provider env set` → `secrets/providers.json.enc` (MAGIC_PASSPHRASE).
- Backups: `$MAGIC_HOME/backups/<harness>/` limit 3.
- `magic provider test` is harness-agnostic HTTP against provider endpoint (httptest in unit tests).
- Aliases: `magic use`, `magic test`.

## Capture allow + config refs (2026-09-09)

- Codex Allow: +`hooks.json`, +`themes/**` (deny sessions/sqlite/cache unchanged).
- Grok Allow: +`hooks/**`, +`commands/**`, +`disabled-hooks`.
- `settings_refs.go`: generic harness-root settings fold + ref follow; Codex `config.toml` → `config.ref`.

## Model registry from Pi (design 2026-09-09)

- Design: `docs/model-registry-from-pi.md` — models.dev primary, OpenRouter optional enrichment.
- Stub (earlier): raw `models.dev.json` download via `internal/modelsdev`.

## Model registry normalize (2026-09-09)

- `internal/modelfile` writes Pi-shaped `$MAGIC_HOME/registry/models.json` after sync.
- `thinkingLevelMap` from models.dev `reasoning_options` effort values (Pi `getEffortThinkingLevelMap`).
- ProviderDefaults table for api/baseUrl (openai→responses, anthropic→messages, deepseek→completions, …); unknown → unsupported + provider.api URL.
- CLI: `sync --source models.dev|all` always normalize; `openrouter` re-normalizes if models.dev.json exists.
- `registry search|show` + `catalog pick` prefer models.json (fuzzy: exact, strip vendor prefix, collapse .-_).
- Apply / `provider use` enrich Implemented harness writers from `registry/models.json` (Pi/Grok/Claude/Codex); sync `--source models.dev` first for best results. Missing meta → defaults, never fail apply.
- Include-all models; `toolCall` field kept. Costs are $/MTok as in models.dev (OR enrichment ×1e6).
- Still out of scope: aliases.json, patches.json, meta etags, Pi compat quirks.
- Tests: `testdata/registry/models.dev.fixture.json` (no live network).

## DeepSeek-aligned Claude / Codex projection (2026-09-09)

- Reference: `docs/deepseek-official-integrations.md`.
- **Claude** third-party: `ANTHROPIC_AUTH_TOKEN` primary (+ `ANTHROPIC_API_KEY` same value); role envs incl. `CLAUDE_CODE_SUBAGENT_MODEL`; optional effort/auto-compact from registry meta; official clears `ANTHROPIC_*` + `CLAUDE_CODE_*`.
- **Codex** third-party: writes Codex-native `~/.codex/models.json` (minimal schema, no instruction blobs) + sets `model_catalog_json` **only** for third-party; `env_key` never plaintext bearer. Official removes catalog_json + deletes Magic `models.json`.
- Seed `deepseek`: anthropic `https://api.deepseek.com/anthropic`, responses `https://api.deepseek.com/`, Models pro/flash (+ subagent).

## Docs + sync/capture policy snapshot (2026-09-09)

- README / AGENTS / NOTES / `docs/*` / `python-magic/docs/需求.md` 对齐当前 CLI（`go run ./cmd/magic --help` 核对）。
- **copy**：secrets-only（`providers.json.enc` + `MAGIC_PASSPHRASE`）；**无** OS env override；无 `copy pi/...` / `grok/...`。
- **R2 sync**：可同步加密 `secrets/*.enc`；passphrase 本机；push 跳过 `backups/`。自动 push 约定挂库变更（capture），**不**挂 apply；当前以手动 `magic sync push` 为主。
- **Capture**：settings.json ref-follow；Codex/Grok hooks in allow；Pi themes yes，`models-store.json` deny，memory 未 allow → 默认不收。
- **Codex**：第三方写 `~/.codex/models.json` + `model_catalog_json`；官方移除 catalog、**不用** `model_catalog_json`；无 plaintext bearer。
- **非目标重申**：整机还原、Raycast/Alfred、Cherry Studio（取消）、拷项目到 Documents。
- **FUTURE**：`aliases.json`、`patches.json`、registry `meta.json` etags、跟手壳、R2 冲突 UI。
