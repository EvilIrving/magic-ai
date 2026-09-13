# Provider 接入：登录 → 拉模型 → 挑选（规划中）

状态：**已实现**（`internal/login` Bubble Tea TUI + `internal/provider/modelfetch.go` + `cmd/magic/login.go`）。决定见 [decisions.md](./decisions.md) D16 / D17。

## 要解决的问题

现在往配置库加一个 provider，用户要手写 `providers.yaml`：provider 名、`key_env`、
每个协议一个 `base_url`、`pick` 里的模型 id —— 全部要自己查文档、自己敲。这是
Magic 目前最重的手工步骤。

目标是把这一步变成：**给一把 key，Magic 自己去问该 provider 有哪些模型，用户勾选。**

```
magic login deepseek
  → 提示输入 API key（或复用已存的）
  → GET https://api.deepseek.com/models （带 key）
  → 列出模型，用户多选
  → 写入 providers.yaml（deepseek + pick）与密钥库（DEEPSEEK_API_KEY）
```

这与 Pi 的 `/login` + 模型选择、Cherry Studio 的「获取模型列表」是同一件事。
它属于**进配置库的手段**：结果仍然是标准 Magic 数据（`providers.yaml` + secrets），
不是新的真相源。

## 两层数据从哪里来

| 层 | 来源 | 作用 |
|---|---|---|
| provider 骨架（`key_env`、`endpoints[]`、协议、默认 `pick`） | 内置模板 `internal/provider/builtin/providers.yaml` | 用户不用再查 base_url / 协议 |
| 模型清单 | 目标 provider 的官方 `GET /models`（或回退） | 用户勾选，写进 `pick` |
| 密钥 | `magic secrets set` 同一套逻辑 | 本地明文写入 `secrets/providers.json` |

内置模板已经写明「有 GitHub repo 的官方 provider 长什么样」：25 个 provider 的
`key_env` + 多协议 `endpoints[]`。所以 `magic login deepseek` 只需要补两样：
**一把 key** 和 **勾选的模型**。

## 模型列表怎么取

沿用 `internal/provider/test_http.go` 里已有的探测约定，但**要读响应体**（现在只
判断 HTTP 状态、丢弃 body）：

| 协议 | 请求 | 备注 |
|---|---|---|
| `openai-chat-completions` / `openai-responses` | `GET {base}/models`，`Authorization: Bearer <key>` | base 通常已含 `/v1` |
| `anthropic-messages` | `GET {base}/v1/models`，`x-api-key` + `anthropic-version: 2023-06-01` | base 可能是 `…/anthropic` |

响应解析（OpenAI 形状）：`{"data":[{"id": "...", ...}]}` → 模型 id 列表。
Anthropic 形状有 `display_name`，可作展示名。

### 取不到列表时（必须有回退，不能卡死）

1. **回退到内置模板的 `pick`** —— 该 provider 已经写过一组默认模型。
2. **回退到 `registry/models.json`** —— models.dev 规范化后的目录，按 provider 过滤。
   需要先 `magic registry sync --source models.dev`。
3. **手动录入** —— 用户直接敲模型 id。

三种回退都要在输出里说明用了哪一种，不能静默。

## 勾选后写什么

一次 `login` 成功应当产出：

- `providers.yaml` 里该 provider 的一段（骨架来自内置模板，`pick` 来自勾选）。
- 密钥库里的 `key_env` → key（走当前 secrets 模式）。
- 可选：把勾选结果 + 列表响应里的信息写进 `models_meta`，或触发一次 registry enrichment。

写入前必须让用户确认（`providers.yaml` 是真相源，不是缓存）。已有同名 provider 时：
按内置模板 README 的合并语义处理 —— 用户已有字段优先，`pick` 与 `endpoints` 是
整段替换还是合并，实现前需要定死（见「待定」）。

## 命令面（已实现）

```
magic login <provider>              # 内置模板找到 provider，提示/复用 key，拉模型，勾选，确认写库
magic login                          # 无参数：从内置模板列出 provider 让用户选（需 TTY）
magic login <provider> --api-key …  # key 从参数来
magic login <provider> --api-key … --pick a,b --yes   # 完全脚本化（无需 TTY）
magic login <provider> --no-live    # 跳过 GET /models，直接走回退
```

与现有命令的关系：

- 不是 `magic provider use` —— 那个是「切换已配置的 provider」，不拉模型。
- 不是 `magic provider seed` —— 那个只是把 seed profiles 合并进 `providers.yaml`。
- 与「official 登录」是两回事：`official`（订阅 OAuth）在 Pi 里是独立流程；
  Magic 的 `provider use official` 只是切回官方，不涉及第三方 key。

## 密钥：本地明文（见 secrets-and-sync.md）

`key_env` 是逻辑索引，本地落到明文 `secrets/providers.json`（0600）。本地无 `.enc`。
`.enc` 只是云端状态，加密只在 `sync push`。`magic login` 存 key 时直接写本地明文，不涉及口令。
详见 [secrets-and-sync.md](./secrets-and-sync.md)（D2）。

## 已实现 vs 规划中

**已实现**

- 内置 provider 模板：`internal/provider/builtin/providers.yaml`（25 家官方；中转不内置）、
  `builtin.Load/Providers/Names`。
- `internal/provider/test_http.go` 的 `/models` 可达性探测。
- `internal/provider/modelfetch.go`：`FetchModels` + `ParseModelsResponse`（OpenAI/Anthropic/裸数组）。
- `internal/login`：`LookupProvider` / `Resolve`（三级回退 + `DefaultSelection`）/ `Apply`（用户优先合并）/ 确认；Bubble Tea TUI（key 输入、多选、单选、确认）。
- `magic login [provider]`：`--api-key` / `--pick` / `--yes` / `--no-live`。

**未做**

- 写 `models_meta`（D17 明确不做）。
- 多协议按 endpoint 分 pick（pick 是 provider 级，D17）。

## 已定 / 待定

**已定**（见 [decisions.md](./decisions.md) D4/D16/D17）：`magic login` = **TUI 交互流程**（Bubble Tea）；无迁移入口；密钥本地明文；确认后写库；回退顺序 live → 内置 pick → registry → 手动；不写 `models_meta`。

**待定**：A1（sync 口令来源：环境变量 / keychain / 文件）；MCP / hooks 投影整体设计（见 `mcp-design.md` / `hooks-design.md`）。
