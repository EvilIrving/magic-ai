# plan

`plan` 回答一个问题：在执行 `apply` 之前，哪个文件会变成什么样，哪些键会被覆盖，哪些会被保留，哪些文件不用动。它只读文件，不写任何目标路径。

## 覆盖范围

按 `apply` 的写入顺序叠加四层：

1. harness 镜像文件的创建、覆盖与合并（`ApplyAdapter`）；
2. claude 的 `claude.json.partial.json` 合并进 `~/.claude.json`；
3. provider 投影（`ApplyProviders`）：该 harness 的 `SwitchWriter.Files` 算出的文件 —— claude 的 `~/.claude/settings.json` env，pi 的 `~/.pi/agent/models.json` 与 `models.magic.json`，opencode 的 `opencode.json` provider 段，workbuddy 的 `~/.codebuddy/models.json`。plan 与 apply 调的是同一个 `Files`，不另写一份计算；
4. MCP 投影（`mcp.Apply`）：`packs/mcp/mcp.yaml` 译成各家的 `mcpServers` / `mcp` / `[mcp_servers]`。

后置层合并进前一层的输出，所以 `after` 就是 `apply` 之后磁盘上的内容；同一个文件被两层写时只有一个 change 条目，`reason` 里按顺序列出参与的层（如 `镜像 JSON 合并 → provider 投影 env`）。镜像里没有、只由后置层写出的文件（`models.magic.json`、provider / MCP 写到的配置）同样是一个 change。

不覆盖 skills 与 sub-agents 的目录投影。它们做的是链接、复制与冲突检测，用 `magic skills status` 和 `magic agents` 查看。

provider 层只在 apply 会投影的 harness 上参与叠加（claude / pi / opencode / workbuddy）。codex 与 grok 也有 SwitchWriter，但那是 `magic provider use` 的路径，`apply` 不写它们的主配置，plan 因此也不为它们算 provider 层。

## 顶层字段

| 字段 | 类型 | 含义 |
|---|---|---|
| `version` | int | plan 结构版本，当前为 1。只有字段含义变化时才加一 |
| `magic_home` | string | 这次计算使用的配置库路径 |
| `user_home` | string | 目标目录所在的用户主目录 |
| `adapters` | array | 按 adapter ID 排序 |
| `summary` | object | `files`、`create`、`update`、`delete`、`unchanged` 五个计数 |

## adapter 条目

| 字段 | 含义 |
|---|---|
| `adapter` | adapter ID，例如 `pi`、`claude`、`codex` |
| `skipped` | 非空表示 apply 不会为它写任何文件，值说明跳过原因：adapter 未实现、该 adapter 没有镜像根、库内没有对应目录；apply 不投影 provider 的 harness（codex / grok，只有 `magic provider use` 会写）会额外写明这一点。镜像目录缺失但 provider / MCP 投影仍会写时，它照常列出 change，不标跳过 |
| `changes` | 按 `rel` 排序 |

## change 条目

| 字段 | 含义 |
|---|---|
| `adapter` | 属于哪个 adapter |
| `path` | 目标的绝对路径 |
| `rel` | 镜像文件用库内相对路径（`packs/harness/<id>` 之下）；只由后置层写的文件用相对 user home 的路径（如 `.pi/agent/models.magic.json`）。`~/.claude.json` 统一用 `_claude.json` |
| `action` | `create`、`update`、`unchanged` |
| `reason` | 哪些层写了这个文件，按 apply 顺序用 ` → ` 连接（镜像层按 `projectBytes` 实际走的策略命名：新建 / JSON 合并 / TOML 合并 / 整文件覆盖 / JSON 解析失败退回整文件覆盖）。`unchanged` 追加「结果与磁盘一致」 |
| `mode` | 写入权限，最后一层决定：镜像层里含密钥的文件 `0600`、其余 `0644`；provider 与 MCP 投影写出的文件 `0600` |
| `before` | 目标当前内容的指纹，目标不存在时为空 |
| `after` | 写入后内容的指纹 |
| `backup` | apply 是否会先把这个文件复制到 `backups/<时间戳>/<adapter>/<rel>`。只有 `update` 为 true：`create` 没有可备份的内容，`unchanged` 不写盘 |
| `keys` | 结构化配置文件的键级差异 |
| `lines` | `config.toml` 的行级增删计数 |

## action 取值

| 值 | 条件 |
|---|---|
| `create` | 目标不存在 |
| `update` | 目标存在，写入后的内容与现在不同 |
| `unchanged` | 目标存在，写入后的内容与现在相同。镜像层、claude partial 与 provider 投影在内容与权限都一致时不写盘（不 rename、不改 mtime）；MCP 投影总是写盘，`unchanged` 只表示内容一致 |
| `delete` | apply 会删除这个目标。provider 投影用 `remove` 表达删除（codex 官方模式就是删掉 Magic 管理的 `~/.codex/models.json`），`after` 为空 |

## key op 取值

| 值 | 含义 |
|---|---|
| `add` | 库定义了这个键，目标没有 |
| `overwrite` | 两边都有，库的值不同，写入后库的值生效 |
| `keep` | 库没有定义这个键，合并后目标的值保留 |

provider 层额外报 `env.<KEY>`：目标是 `settings.json` 时可看出 provider 要加/覆盖哪些 env 键。只有键名，键值不进 plan。

键差异的基准是磁盘：`add` / `overwrite` 只描述「磁盘现在的内容 → 最终字节」这一步。值相同的键不列出，所以一个 `unchanged` 的文件不会有任何 key 条目 —— 它没有被重写，也就没有什么被覆盖。嵌套对象逐层展开到值，数组与标量停在路径上。`keep` 只列顶层，来自库内镜像与 claude partial 的合并，用来证明 D5 的非破坏性，不把所有用户键逐条展开。

## 指纹

`before` 与 `after` 是 `sha256:<前 12 位>`。它让 plan 可以安全地写进日志、贴进 issue、提交进仓库：密钥与 token 不会出现在文档里，指纹也无法反推出内容。代价是 plan 不能用来核对一个值是否正确，只能核对它有没有变。

## 一份真实样例

下面这份 `plan.example.json` 由 `TestPlanExampleDocument` 生成，覆盖 create、update、unchanged、跳过、JSON 的键差异、TOML 的行差异，以及镜像 → provider / MCP 的叠层。

```json
{
  "version": 1,
  "magic_home": "<magic-home>",
  "user_home": "<user-home>",
  "adapters": [
    {
      "adapter": "claude",
      "changes": [
        {
          "adapter": "claude",
          "path": "<user-home>/.claude.json",
          "rel": "_claude.json",
          "action": "update",
          "reason": "claude partial 深合并 → mcp 投影",
          "mode": "0600",
          "before": "sha256:a4a3ad3d34d5",
          "after": "sha256:d1ba469713c8",
          "backup": true,
          "keys": [
            {
              "key": "mcpServers.docs.args",
              "op": "add"
            },
            {
              "key": "mcpServers.docs.command",
              "op": "add"
            },
            {
              "key": "projects",
              "op": "keep"
            }
          ]
        },
        {
          "adapter": "claude",
          "path": "<user-home>/.claude/settings.json",
          "rel": "settings.json",
          "action": "update",
          "reason": "镜像 JSON 合并 → provider 投影 env",
          "mode": "0600",
          "before": "sha256:aa023373b963",
          "after": "sha256:0f994c9d63c2",
          "backup": true,
          "keys": [
            {
              "key": "env.ANTHROPIC_API_KEY",
              "op": "add"
            },
            {
              "key": "env.ANTHROPIC_AUTH_TOKEN",
              "op": "overwrite"
            },
            {
              "key": "env.ANTHROPIC_BASE_URL",
              "op": "add"
            },
            {
              "key": "env.ANTHROPIC_DEFAULT_HAIKU_MODEL",
              "op": "add"
            },
            {
              "key": "env.ANTHROPIC_DEFAULT_OPUS_MODEL",
              "op": "add"
            },
            {
              "key": "env.ANTHROPIC_DEFAULT_SONNET_MODEL",
              "op": "add"
            },
            {
              "key": "env.ANTHROPIC_MODEL",
              "op": "add"
            },
            {
              "key": "env.CLAUDE_CODE_SUBAGENT_MODEL",
              "op": "add"
            },
            {
              "key": "statusLine",
              "op": "keep"
            }
          ]
        }
      ]
    },
    {
      "adapter": "codex",
      "changes": [
        {
          "adapter": "codex",
          "path": "<user-home>/.codex/config.toml",
          "rel": "config.toml",
          "action": "update",
          "reason": "镜像 TOML 合并 → mcp 投影",
          "mode": "0600",
          "before": "sha256:969c7d8df1c3",
          "after": "sha256:7e6248f70fbe",
          "backup": true,
          "lines": {
            "added": 6,
            "removed": 1
          }
        }
      ]
    },
    {
      "adapter": "nope",
      "skipped": "adapter 未实现"
    },
    {
      "adapter": "pi",
      "changes": [
        {
          "adapter": "pi",
          "path": "<user-home>/.pi/agent/AGENTS.md",
          "rel": "AGENTS.md",
          "action": "create",
          "reason": "镜像新建",
          "mode": "0644",
          "after": "sha256:f895f32a48d8"
        },
        {
          "adapter": "pi",
          "path": "<user-home>/.pi/agent/mcp.json",
          "rel": "mcp.json",
          "action": "create",
          "reason": "mcp 投影",
          "mode": "0600",
          "after": "sha256:a4b1fd76f96b",
          "keys": [
            {
              "key": "mcpServers.docs.args",
              "op": "add"
            },
            {
              "key": "mcpServers.docs.command",
              "op": "add"
            },
            {
              "key": "mcpServers.docs.disabled",
              "op": "add"
            }
          ]
        },
        {
          "adapter": "pi",
          "path": "<user-home>/.pi/agent/models.json",
          "rel": "models.json",
          "action": "update",
          "reason": "镜像整文件覆盖 → provider 投影 models.json",
          "mode": "0600",
          "before": "sha256:c30d3a63c009",
          "after": "sha256:7f644d12b954",
          "backup": true,
          "keys": [
            {
              "key": "providers.kimi.api",
              "op": "overwrite"
            },
            {
              "key": "providers.magic-deepseek.api",
              "op": "add"
            },
            {
              "key": "providers.magic-deepseek.apiKey",
              "op": "add"
            },
            {
              "key": "providers.magic-deepseek.baseUrl",
              "op": "add"
            },
            {
              "key": "providers.magic-deepseek.models",
              "op": "add"
            }
          ]
        },
        {
          "adapter": "pi",
          "path": "<user-home>/.pi/agent/models.magic.json",
          "rel": "models.magic.json",
          "action": "create",
          "reason": "provider 投影 models.magic.json",
          "mode": "0600",
          "after": "sha256:4b784165a420",
          "keys": [
            {
              "key": "providers.magic-deepseek.api",
              "op": "add"
            },
            {
              "key": "providers.magic-deepseek.apiKey",
              "op": "add"
            },
            {
              "key": "providers.magic-deepseek.baseUrl",
              "op": "add"
            },
            {
              "key": "providers.magic-deepseek.models",
              "op": "add"
            }
          ]
        },
        {
          "adapter": "pi",
          "path": "<user-home>/.pi/agent/themes/dark.json",
          "rel": "themes/dark.json",
          "action": "unchanged",
          "reason": "镜像整文件覆盖（结果与磁盘一致）",
          "mode": "0644",
          "before": "sha256:a143b7d0d534",
          "after": "sha256:a143b7d0d534"
        }
      ]
    }
  ],
  "summary": {
    "files": 8,
    "create": 3,
    "update": 4,
    "delete": 0,
    "unchanged": 1
  }
}
```

## 保持样例不过期

样例是 golden 文件，改了 plan 结构就要重新生成：

```bash
go test ./internal/core -run TestPlanExampleDocument -update-golden
```

日常测试会比对这份文件，忘记更新时会失败。

## 已知限制

- `config.toml` 只给行数增删，不给键级差异。文本合并的定位精度低于 JSON。
- 原样覆盖的文件（例如 `models.json`）只给 `before` 与 `after` 两个指纹，不给逐键差异。
- skills 与 sub-agents 的投影不在 plan 内。
- codex / grok 的 provider 投影不在 apply 内（`magic provider use codex|grok` 才写），所以 plan 不为它们算 provider 层。
- provider 密钥缺失时 apply 跳过该层，plan 也跳过：两边都不写，所以不会互相报漂移。
