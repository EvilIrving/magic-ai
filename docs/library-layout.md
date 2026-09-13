# 配置库布局（已定死）

真相源：用户家目录 `~/.magic-ai`（`MAGIC_HOME`）。与该目录 `README.md` 同步。

产品：一份 skill / MCP / 中转 API，按各 harness 格式写进去；外加各工具自己的自定义文件方便换电脑。手改 / CLI 命令是进这个目录的手段。

```
~/.magic-ai/
  README.md
  magic.yaml                哪些 harness 启用
  providers.yaml            中转 + 模型 pick（无密钥）
  secrets/providers.json    明文密钥（0600）；key_env → value
  packs/skills/<name>/
  packs/mcp/                mcp.yaml（你添加）
  packs/agents/<name>.md
  packs/harness/<id>/   theme、extensions、AGENTS.md、memories、hooks…（按文件写回该 harness）
  registry/                 models.dev 缓存，不是用户手改的配置
  backups/                  apply 备份，不同步
```

| 层 | 路径 | apply |
|---|---|---|
| 共有原数据 | `providers.yaml` + `secrets/`、`packs/skills/`、`packs/mcp/`、`packs/agents/` | 译成 Pi / Claude / Codex / Grok / OpenCode… 各自格式 |
| 某工具自定义 | `packs/harness/<id>/` | 只写回这个 harness，不译成别的工具 |

## 密钥：本地明文，`.enc` 是云端状态

- 本地 `secrets/providers.json` 是**明文**（`0600`）。本地没有 `.enc`，不是加密状态，不需要口令。
- `.enc` **只表示云端状态**：`sync push` 把含密钥的文件加密后上传为 `<rel>.enc`，其余明文；`sync pull` 解密后落回本地明文（无 `.enc`）。
- `MAGIC_PASSPHRASE`（或等价口令）只在 push / pull 时需要，**不进库、不同步、不进 R2**。
- 口径见 [secrets-and-sync.md](./secrets-and-sync.md)（D2）。

不进库：access token、OAuth、官方登录态（`auth.json`）、会话、缓存、`npm/`、`bin/`、运行态。
harness 自定义放 `packs/harness/<id>/`（capture 已删，但目录保留作为自定义层，按文件写回该 harness）。
