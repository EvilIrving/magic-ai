# 发行签名与公证（macOS / Windows）

> **状态：规划中，未实现。** 本仓库目前没有 CI、没有签名脚本、也没有任何签过名的发行物。
> 这份文档只做一件事：把「将来要出预编译发行物时」需要准备的东西、要做的决策、要问公司的问题一次写清，避免真要做的时候重新调研。
> 真开始做时，从 [§6 决策点](#6-决策点与-todo入口) 往下走。

---

## 0. 先判断：这个项目到底需不需要签名

`magic` 是 Go CLI。签名只在**分发预编译二进制**时才有意义。

| 分发方式 | 需要签名吗 |
|---|---|
| 用户自己 `go install` / 从源码 build | **不需要** |
| Homebrew formula（在用户机器上编译） | **不需要** |
| Homebrew cask / GitHub Releases 直接下载预编译二进制 | **需要** |
| 装进 `.app` / `.dmg` / `.pkg` / `.msi` 安装器 | **需要** |

不签名 macOS 也能跑（用户 `xattr -d com.apple.quarantine` 或右键打开），但 Gatekeeper 会拦、企业环境常直接禁止、Windows SmartScreen 会红屏警告。对外发预编译包时必须签。

---

## 1. 产物形态决定签名范围（这是第一个要定的事）

「CLI / TUI / 别的形态」影响的是打包方式，不是签名原理。三种典型路径：

| 形态 | macOS 要做 | Windows 要做 |
|---|---|---|
| **A. 裸二进制**（`magic-darwin-arm64` 等，打 zip/tar.gz 发） | `codesign` 单个 Mach-O → 装进 zip 公证。**不能 staple**（见 §2.5） | `signtool` 签 `.exe` |
| **B. `.app` 包 / `.dmg`**（双击安装体验） | 签 `.app` → 公证 → `stapler staple` `.app` 和 `.dmg` | 不适用 |
| **C. 安装器**（`.pkg` / `.msi`） | `productbuild` 出 `.pkg`，需 **Developer ID Installer** 证书（当前没有） | `.msi` 需先签内含 `.exe` 再签 `.msi` |

裸二进制（A）是最省事的；只要产品不需要「双击图标」，不必为了签名硬做 `.app`。

---

## 2. macOS

### 2.1 需要什么

| 项 | 状态 | 说明 |
|---|---|---|
| Apple Developer Program 会员 | ✅ 已有 | Team `TIANBAO DONG` / `QZZ878S3NS` |
| **Developer ID Application** 证书 | ✅ 已有 | 登录钥匙串，有效期到 2031-05-04 |
| 公证凭据（API Key `.p8` 或 app-specific password） | ✅ 已有 | 两把 p8，keychain profile `AC_NOTARY` 已配好 |
| **Developer ID Installer** 证书 | ❌ 没有 | 只有做签名 `.pkg` 才需要；DMG 用不到。需要时在 Xcode → Accounts → Manage Certificates 里新建 |
| `codesign` / `notarytool` / `stapler` / `spctl` / `hdiutil` | ✅ 已有 | 随 Xcode 26 / CLT |

证书、`.p8`、profile 的确切位置和口令**不在本文件**。查 `local-environment` skill 的 Apple 段（`references/inventory.md`、`references/secrets.md`），文件在 `~/Library/Developer/Apple`。

### 2.2 签名单个 Go 二进制

```bash
codesign --force --options runtime --timestamp \
  --sign "Developer ID Application: TIANBAO DONG (QZZ878S3NS)" \
  ./magic

# 验证
codesign --verify --strict --verbose=2 ./magic
codesign -dv --verbose=4 ./magic
```

要点：

- **`--options runtime`（Hardened Runtime）是公证的硬性前提**，不能省。
- `--timestamp` 要联网，Apple 时间戳服务偶发抽风。参照 `python-mediabrief/scripts/sign_and_package.sh` 里的 `codesign_with_retry` 做短重试，别让整包重做。
- **entitlements 一般不需要**。纯 Go、无 cgo、无 JIT 的 CLI 不需要任何 entitlement。只有当二进制里嵌了要 JIT 或动态生成代码的东西时才可能要 `allow-jit` / `allow-unsigned-executable-memory`，届时再单独写 `.plist`。
- **universal binary**（同时支持 Intel + Apple Silicon）要先分别构建再 `lipo`，然后签合并后的那一个文件：

  ```bash
  GOOS=darwin GOARCH=arm64 go build -o /tmp/magic-arm64 ./cmd/magic
  GOOS=darwin GOARCH=amd64 go build -o /tmp/magic-amd64 ./cmd/magic
  lipo -create -output ./magic /tmp/magic-arm64 /tmp/magic-amd64
  ```

### 2.3 公证

`notarytool submit` **只接受三种输入：`.zip` / `.pkg` / `.dmg`**（不接受裸二进制）。

```bash
# 裸二进制：装进 zip 提交
ditto -c -k --keepParent ./magic ./magic.zip
xcrun notarytool submit ./magic.zip \
  --keychain-profile AC_NOTARY --wait --output-format json

# 看失败原因（被拒时必看）
xcrun notarytool log <submission-id> --keychain-profile AC_NOTARY
```

不用 keychain profile 时也可以直接给 key：

```bash
xcrun notarytool submit Foo.zip --wait \
  --key ~/Library/Developer/Apple/AuthKey_9LR8743L58.p8 \
  --key-id 9LR8743L58 \
  --issuer 84d76d9f-080c-43b6-ae44-92652cf3ae23
```

公证前提：包里**每一个 Mach-O 都已用 Developer ID 签名 + Hardened Runtime + 安全时间戳**。Go 二进制只有一层，比较省心。

### 2.4 验证（发之前必做）

```bash
codesign --verify --strict --verbose=2 ./magic
spctl --assess --type execute --verbose=2 ./magic     # Gatekeeper 实际判定
```

`.app` / `.dmg` 用 `--deep` 加验证，并 `xcrun stapler validate`。

### 2.5 ⚠️ 裸二进制公证后无法 staple —— 这是选形态的关键

`stapler` 的官方支持范围只有三类（本机 `man stapler` 原文）：

> Supported file formats are: UDIF disk images, code-signed executable bundles, and signed "flat" installer packages.

即 **`.dmg` / `.app`(等已签名 bundle) / 签名 `.pkg`**，**不含裸 Mach-O，也不含 zip**。

含义：

- 选形态 A（裸二进制）：公证能过，但**票据只能在线校验**。用户离线或初次运行时 Gatekeeper 仍需联网查询。日常够用，Homebrew cask 就是这么发的。
- 要**离线可验、更干净**：必须走 `.app` / `.dmg` / `.pkg`（形态 B/C），公证后再 `xcrun stapler staple`。
- 每次重新 `codesign` 都会让已 staple 的票据失效，必须重新 staple。

### 2.6 CI（未来，当前仓库没有 `.github/`）

参照本项目已有的两套成熟做法，别重新发明：

- `python-mediabrief/.github/workflows/release-macos.yml` + `scripts/sign_and_package.sh`（CLI/打包类，最接近本项目）
- `swift-light-stats/.github/workflows/release.yml`

约定做法：

1. **build job 与 sign job 拆开**，build job 不接触任何 Apple 凭据。
2. sign job 绑定受保护的 GitHub **Environment `APPLE`**（开 required reviewers）。
3. runner 上临时 `security create-keychain` 导入 p12，用完销毁。
4. 证书 / `.p8` 全部走 Environment secrets，**永不进仓库**。已用过的命名两套（各仓库自选一套保持一致）：
   - `BUILD_CERTIFICATE_BASE64` / `P12_PASSWORD` / `APPLE_API_KEY_BASE64` / `APPLE_API_KEY_ID` / `APPLE_API_ISSUER_ID`
   - `APPLE_CERTIFICATE` / `APPLE_CERTIFICATE_PASSWORD` / `APPLE_API_KEY` / `APPLE_API_KEY_ID` / `APPLE_API_ISSUER`
5. 失败时**明确失败，不许降级产出未签名发行物**。

---

## 3. Windows

设计前提：**签名由公司执行**，我们提供待签文件并对齐参数。

### 3.1 先明确：2023-06 起证书不能是裸 `.pfx`

CA/Browser Forum 基线要求生效后，代码签名私钥必须存放在 **FIPS 140-2 Level 2 及以上的硬件或云 HSM** 里。所以：

- 不能再把证书导成 `.pfx` 随便拷来拷去；
- 要么公司持有 **USB token / 硬件 HSM**（签名必须在插着 token 的那台机器上做）；
- 要么用**云签名服务**（Azure Artifact Signing 等）通过 API 签。

这直接决定了「谁签、在哪签、能不能进 CI」。

### 3.2 三条路线，让公司选

| 路线 | 怎么签 | 备注 |
|---|---|---|
| **公司自购 OV / EV 证书 + token/HSM** | 公司机器上 `signtool` | 最传统；CI 自动化困难，通常还是人工签 |
| **Azure Artifact Signing**（原 Azure Trusted Signing，已改名） | 云服务，`signtool` 对接，私钥在微软 FIPS 140-3 L3 HSM | 零证书生命周期管理，能进 CI；**Public Trust 对组织所在国有地区限制**（美/加/欧盟/英/澳/新/日/韩/新/瑞士/挪/以；个人开发者仅限美/加），且需 Azure 订阅 + Entra 租户 + 身份验证。Basic / Premium 两档 SKU，见官方定价 |
| **公司已有签名基础设施** | 我们把 `.exe` 交给他们，他们回传签好的 | 最省事，但每版都要走一次流程 |

### 3.3 每次签名需要的东西

- **待签文件**：Go 产物在 Windows 上通常就一个 `magic.exe`；若做 `.msi` 安装器，则「内含 `.exe` 先签 → 再签 `.msi`」，顺序不能反。
- **摘要算法**：SHA-256（`/fd SHA256 /td SHA256`）。
- **RFC 3161 时间戳**：必须要有。没时间戳的话证书一过期签名就全废。
  ```
  /tr http://timestamp.digicert.com
  ```
- 参考命令（token / 云服务的具体参数由公司按他们的设备给）：
  ```bat
  signtool sign /fd SHA256 /td SHA256 /tr http://timestamp.digicert.com app.exe
  ```

### 3.4 要向公司确认的清单

1. 用哪条路线（自购证书 / Azure Artifact Signing / 他们已有设施）？
2. 谁执行签名 —— 他们代签，还是给我们服务访问 / token？
3. 用哪个 RFC 3161 时间戳服务？
4. 签哪些对象：只签 `.exe`，还是安装器（`.msi`）也要？
5. SmartScreen 信誉：EV 与云签名通常立即生效，OV 需要累积 —— 用哪种？
6. 交付方式：他们回传签好的文件，还是我们（CI）调用他们的服务？
7. 是否需要我们提供 SHA-256 / SBOM / 发布清单？
8. 证书主体名（Subject）要显示成什么组织名？会印在签名上，需要和产品归属一致。

### 3.5 边界

- 不在 macOS 上直接签 Windows（除非走云服务 API）。
- 证书、私钥、token 密码不进仓库、不进公开 CI 日志。
- 没有公司侧的确认前，不要假设已有 Windows 签名能力。

---

## 4. 签名之外、两边都要做的事

- **版本单一来源**：仓库根部 `VERSION`（当前 `1.0.0`）是唯一版本源；构建时通过 `-ldflags -X magic/internal/core.Version=...` 注入（见 `Makefile`）。产物文件名、Install 元数据、签名后的清单必须都跟它一致。
- **每个产物出 SHA-256**，并写一份 release manifest（版本 / 架构 / 文件名 / SHA-256 / 签名身份 / 公证 submission id）。参考 mediabrief 的 `write_release_manifest`。
- **产物命名规范**：`magic-<version>-<os>-<arch>.<ext>`。
- **公证 / 签名日志留档**，失败时定位用。

---

## 5. 参考（本机已有先例，别重写）

| 参考 | 位置 |
|---|---|
| CLI 类签名 + 公证 + DMG + 清单 完整脚本 | `python-mediabrief/scripts/sign_and_package.sh` |
| 签名 CI 的凭据准备与验证流程 | `python-mediabrief/CICD.md`、`.github/workflows/release-macos.yml` |
| Swift .app 签名的 tag 发布流水线 | `swift-light-stats/.github/workflows/release.yml` |
| 本机证书 / `.p8` / notary profile 事实 | `local-environment` skill → `references/inventory.md`（Apple 段）、`references/secrets.md` |

---

## 6. 决策点与 TODO（入口）

真开始做时，按顺序处理：

**先定形态 / 产品决策**

- [ ] 定最终分发形态：裸二进制（A）／ `.app`+`.dmg`（B）／ 安装器（C）？
- [ ] 是否要 universal binary（Intel + Apple Silicon）？
- [ ] 正式分发渠道：GitHub Releases / Homebrew tap / 官网下载？

**macOS**

- [ ] 是否需要 **Developer ID Installer** 证书（只有做签名 `.pkg` 才需要）？
- [ ] 是否需要 entitlements？（默认否，除非嵌了 JIT 类组件）
- [ ] 写 `scripts/sign-macos.sh`，参照 `python-mediabrief`，带时间戳重试、被拒时抓 notary log
- [ ] 决定是否 staple：裸二进制不能 staple，要离线可验就得走 `.app`/`.dmg`/`.pkg`

**Windows**

- [ ] 与公司对齐 §3.4 的 8 个问题，确定路线、时间戳服务、交付方式
- [ ] 确认是否需要 `.msi`（GoReleaser / nfpm 可出）

**工程化**

- [ ] 是否引入 GoReleaser 统一多平台构建 + 签名 hook？（vs 手写脚本）
- [ ] 建 `.github/workflows/release.yml`：build / sign 分离，sign 走受保护 Environment
- [ ] secrets 放 GitHub Environment，不进仓库
- [ ] 首发先手动 `workflow_dispatch` 验证一次，再挂 `v*` tag 自动发布
