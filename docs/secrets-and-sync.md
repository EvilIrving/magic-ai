# 密钥与多机：本地明文，`.enc` 是云端状态

状态：**产品口径（已锁定，见 [decisions.md](./decisions.md) D2 / D3）**。
怎么做写在这里；实现差距只列在 [open-questions.md](./open-questions.md) 的 C 项，不把「代码还没改」写进产品句子。

## 口径（只这一段）

| 位置 | 文件 | 形态 |
|---|---|---|
| **本地** | `secrets/providers.json` | **明文**，0600。无 `.enc`，不需要口令 |
| **云端** | `vault/secrets/providers.json.enc` | **密文**。`.enc` **只表示云端状态**，不是本地文件 |
| **本地其它含密钥文件** | 原路径明文（如 `packs/harness/pi/models.json` 的 `apiKey`） | 明文 |
| **云端对应** | 同相对路径加 `.enc` | 密文 |
| **不含密钥的文件** | 同路径 | 云端也明文，不加 `.enc` |

- 本地永远不是加密状态。日常 `apply` / `copy` / `secrets` / `provider` 只读写明文，从不解密。
- 加密只在 `sync push`；解密只在 `sync pull`。pull 落盘永远是明文、无 `.enc`。
- 口令（`MAGIC_PASSPHRASE` 或等价）只用于 push/pull，**不进库、不同步、不进 R2**。

投影到 harness 的 key（D3）：Pi / WorkBuddy / OpenCode / Claude 写**真值**，不用 `$KEY_ENV`。Codex 用其原生 `env_key`。

## 为什么本地不加密

1. 钥匙和数据在同一台机器上。能读本地文件的人通常也能拿到解密钥匙，本地再加密一层没有收益。
2. 用户要随时 `cat` / 拷贝 / `diff` / 手改，不能每次先解密。
3. 加密只对「离开这台机器」的那份有意义。云端那份才可能被第三方碰到。

一句话：**加密是推送时的变换；`.enc` 是云端对象名，不是本地存储形态。**

## 加密范围：只加密含密钥的文件

逐文件做，不整库打包（push/pull 能按文件增量）。

| 本地文件 | 云端对象 |
|---|---|
| `secrets/providers.json` | `vault/secrets/providers.json.enc`（密文） |
| `packs/harness/**` 里含明文 key 的文件 | 同相对路径 + `.enc`（密文） |
| 其它（skills / mcp / agents / `providers.yaml` / 不含 key 的定制） | 同路径明文 |

「含密钥」判定：文件内容出现 `secrets/providers.json` 里某个 value 的真值。命中则加密后以 `.enc` 上传，否则原样上传。

## 本地布局

```
~/.magic-ai/
  providers.yaml                 # provider / endpoints / pick（没有密钥值）
  secrets/
    providers.json               # {"DEEPSEEK_API_KEY": "sk-…", …}   0600 明文
  packs/harness/<id>/            # 可能含明文 key（如 Pi models.json）
  ...（skills / mcp / agents / registry / backups）
```

- `providers.json` 是 `key_env → value` 的明文 map。
- 本地不含 `providers.json.enc`，不含 `MAGIC_SECRETS_MODE`，不含 adapter `*.json.enc` reinject。
- per-harness 定制文件里的密钥也是明文（Pi `models.json` 的 `apiKey` 真值）。

## 推送 / 拉取

```
magic sync push:
  含密钥的文件  ──加密(passphrase)──►  云端 <rel>.enc
  其余文件      ──原样─────────────►  云端 <rel>

magic sync pull:
  <rel>.enc  ──解密(passphrase)──►  本地 <rel>（明文，无 .enc）
  其它对象   ──原样─────────────►  本地同路径明文
```

口令来源（沿用现状，可再定）：环境变量 `MAGIC_PASSPHRASE`，或本机约定文件
（如 `~/.magic-passphrase`）。它**不进库、不进 R2**。未定项见 [open-questions.md](./open-questions.md) A1。

## 日常命令与口令

| 命令 | 需要口令？ | 说明 |
|---|---|---|
| `magic apply` | 否 | 读本地明文；Pi / WorkBuddy / OpenCode / Claude 写真值 key |
| `magic copy` | 否 | 读本地明文 |
| `magic secrets set/list` | 否 | 直接读写本地 `providers.json` |
| `magic provider ...` | 否 | 读本地明文 |
| `magic sync push` | **是** | 含密钥文件加密后以 `.enc` 上传 |
| `magic sync pull` | **是** | 下载 `.enc` 解密，落回本地明文 |

## 安全边界

- 本地明文意味着：任何能读用户目录的进程都能读到密钥。这是刻意取舍。
- 云端 `.enc` 能防的是：拿到 R2 凭证/对象、但没拿到本机口令的人。
- 它**不能**防已经控制本机的人（钥匙与数据同机）。

## 实现状态

已实现（C1 / C2 / C3 已落代码）：

- 本地 `secrets/providers.json` 明文，无 `.enc`、无 `MAGIC_SECRETS_MODE`。
- `sync push` 按「路径在 `secrets/` 或内容含真值」加密上传 `<rel>.enc`；`sync pull` 解密落回明文。
- Pi / WorkBuddy / OpenCode / Claude 投影写真值 key；无 key 则 skip + 日志，不写空值。
- 已删 adapter `*.json.enc` reinject。

仍开放：A1（口令来源：环境变量 / keychain / 文件）。见 [open-questions.md](./open-questions.md)。
