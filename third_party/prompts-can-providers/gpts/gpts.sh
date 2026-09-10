#!/usr/bin/env bash
#Requires: bash 3.2+, curl
#
# =============================================================================
#  gpts.sh  —  公司 Codex / ChatGPT 网关切换（shu）· macOS / Linux
# =============================================================================
#
# 【本文件即完整说明书】用任意文本编辑器打开即可阅读；不必回仓库、不必搜文档。
#  本脚本可单独拷走使用（微信 / 网盘 / U 盘均可），不依赖仓库里其它文件。
#  Windows 请用同目录的 gpts.ps1（或已 install 后的 gpts）。
#
# ── 意图 ──────────────────────────────────────────────────────────────────
#  把 Codex / ChatGPT Desktop 的 model_provider 在「官方 openai」与
#  「公司中转 api.shu.cool」之间切换。
#  改写的是本机配置：  ~/.codex/config.toml
#  API Key 环境变量：  SHU_API_KEY（与 Grok 用的 groks 脚本共用同一 key）
#
#  company 模式会确保：
#    model_provider = "shu"
#    [model_providers.shu]
#    name / base_url / env_key / wire_api / requires_openai_auth
#
# ── 依赖 ──────────────────────────────────────────────────────────────────
#  - macOS 或 Linux，bash 3.2+，curl
#  - 已安装 Codex CLI 或 ChatGPT Desktop（本脚本不负责安装 Codex）
#  - 不需要管理员 / sudo
#
# ── 怎么运行 ──────────────────────────────────────────────────────────────
#    chmod +x ./gpts.sh
#    ./gpts.sh help
#    ./gpts.sh status
#  或:
#    bash ./gpts.sh help
#
# ── macOS 拦截（「无法验证开发者」/ 打不开）──────────────────────────────
#    方案 A:
#      xattr -d com.apple.quarantine ./gpts.sh
#      chmod +x ./gpts.sh
#      ./gpts.sh help
#    方案 B:
#      bash ./gpts.sh help
#
# ── 推荐流程 ──────────────────────────────────────────────────────────────
#    ./gpts.sh env              # 隐藏输入 SHU_API_KEY
#    ./gpts.sh test
#    ./gpts.sh use company
#    然后彻底退出 ChatGPT / Codex（含托盘）再打开
#
#    也可手动永久设置 Key:
#      echo 'export SHU_API_KEY="你的 Key"' >> ~/.zshrc && source ~/.zshrc
#
#    切回官方:
#      ./gpts.sh use official
#
# ── 可选 install ──────────────────────────────────────────────────────────
#    ./gpts.sh install
#    source ~/.zshrc
#    gpts help
#  install 会复制到 ~/.codex/gpts.sh 并设置 alias gpts=...
#  不 install 也可始终用 ./gpts.sh 直接跑。
#
# ── 常用命令 ──────────────────────────────────────────────────────────────
#    ./gpts.sh status | help
#    ./gpts.sh env | env clear | test
#    ./gpts.sh use company | use official
#    ./gpts.sh backup list | restore | clear
#
# ── 改完「没生效」时 ──────────────────────────────────────────────────────
#  1) 应用是否还在跑（托盘）→ 必须全退再开
#  2) ./gpts.sh status  看 Provider 是否 shu、Key 是否 set
#  3) ./gpts.sh test
#  4) 打开 ~/.codex/config.toml 应有 model_provider = "shu" 与 [model_providers.shu]
#  5) 新开终端后 echo $SHU_API_KEY 是否非空
#
# ── 常见故障 ──────────────────────────────────────────────────────────────
#  · Permission denied → chmod +x ./gpts.sh  或  bash ./gpts.sh ...
#  · 无法验证开发者   → xattr -d com.apple.quarantine ./gpts.sh
#  · KEY not set       → ./gpts.sh env ；必要时 source ~/.zshrc
#  · HTTP 401/403      → Key 错误/过期
#  · 连不上网关        → VPN/网络；curl 测 https://api.shu.cool/v1/models
#  · 配置锁冲突        → rm -rf ~/.codex/.setup-shu.lock 后重试
#  · 误改想回滚        → ./gpts.sh backup restore
#
# ── 安全说明 ──────────────────────────────────────────────────────────────
#  - 不需要管理员
#  - SHU_API_KEY 写在用户 shell profile，本用户其它程序可读
#  - 配置有锁、校验、备份、原子写入
#  - 命令行再打说明: ./gpts.sh help
#
# =============================================================================
set -euo pipefail

# ── Parameters (mirror PS script param block) ──────────────────────
COMMAND="${1:-status}"
VALUE="${2:-}"

CODEX_HOME="${CODEX_HOME:-$HOME/.codex}"
BACKUP_LIMIT=3
TIMEOUT_SEC=15
FORCE=false
DRY_RUN=false
NO_COLOR=false

# ── Internal constants ─────────────────────────────────────────────
CONFIG_TOML="$CODEX_HOME/config.toml"
BACKUP_DIR="$CODEX_HOME/provider-switch-backups"
LOCK_DIR="$CODEX_HOME/.setup-shu.lock"
KEY_NAME="SHU_API_KEY"
COMPANY_PROVIDER="shu"
OFFICIAL_PROVIDER="openai"
COMPANY_BASE_URL="https://api.shu.cool/v1"
TOOL_VERSION="1.2.0"

# Fields/sections left by other provider setups can mask or override the selected
# model_provider. gpts.sh does not install a custom models.json catalog, so
# DeepSeek's catalog/login fields are removed instead of rewritten.
readonly CONFLICT_KEYS="model_catalog_json preferred_auth_method forced_login_method profile openai_base_url oss_provider"
readonly DEEPSEEK_MODEL_RE='^[[:space:]]*model[[:space:]]*=[[:space:]]*"deepseek-[A-Za-z0-9_.-]+'

# ── Managed provider settings (mirrors $script:CompanySettings) ────
# Each entry: key|line (pipe-delimited, to avoid bash array-of-structs)
readonly COMPANY_SETTINGS=(
    "name|name = \"shu company\""
    "base_url|base_url = \"$COMPANY_BASE_URL\""
    "env_key|env_key = \"$KEY_NAME\""
    "wire_api|wire_api = \"responses\""
    "requires_openai_auth|requires_openai_auth = false"
)

# ── Argument parsing (mirrors PS param block) ──────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        -CodexHome)       CODEX_HOME="$2"; shift 2 ;;
        -BackupLimit)     BACKUP_LIMIT="$2"; shift 2 ;;
        -TimeoutSec)      TIMEOUT_SEC="$2"; shift 2 ;;
        -Force)           FORCE=true; shift ;;
        -DryRun)          DRY_RUN=true; shift ;;
        -NoColor)         NO_COLOR=true; shift ;;
        -h|--help|-?)     COMMAND="help"; shift ;;
        *)
            if [[ -z "${POS1:-}" ]]; then POS1="$1";
            elif [[ -z "${POS2:-}" ]]; then POS2="$1"; fi
            shift
            ;;
    esac
done
COMMAND="${POS1:-$COMMAND}"
VALUE="${POS2:-$VALUE}"

# Re-derive paths in case CodexHome changed
CONFIG_TOML="$CODEX_HOME/config.toml"
BACKUP_DIR="$CODEX_HOME/provider-switch-backups"
LOCK_DIR="$CODEX_HOME/.setup-shu.lock"

# ── Output helpers ──────────────────────────────────────────────────
ok()   { if $NO_COLOR; then echo "[OK]  $*"; else printf '\033[32m%s\033[0m\n' "$*"; fi; }
info() { if $NO_COLOR; then echo "[INFO] $*"; else printf '\033[36m%s\033[0m\n' "$*"; fi; }
warn() { if $NO_COLOR; then echo "[WARN] $*"; else printf '\033[33m%s\033[0m\n' "$*"; fi; }
warn_err() { if $NO_COLOR; then echo "[WARN] $*" >&2; else printf '\033[33m%s\033[0m\n' "$*" >&2; fi; }
err()  { printf '\033[31m[ERR]  %s\033[0m\n' "$*" >&2; }
die()  { err "$@"; exit 1; }

# ── Locking (mkdir mutex — mirrors Invoke-WithConfigLock) ──────────
acquire_lock() {
    local deadline; deadline=$(($(date +%s) + 10))
    while ! mkdir "$LOCK_DIR" 2>/dev/null; do
        if [[ $(date +%s) -gt $deadline ]]; then
            die "Another setup-shu process is modifying the configuration. Try again later."
        fi
        sleep 0.2
    done
}
release_lock() { rmdir "$LOCK_DIR" 2>/dev/null || true; }

with_lock() {
    if $DRY_RUN; then "$@"; return; fi
    acquire_lock
    local rc=0; "$@" || rc=$?; release_lock; return $rc
}

# ── Ensure Codex home exists ────────────────────────────────────────
ensure_home() {
    [[ -n "$CODEX_HOME" ]] || die "Codex home path is empty."
    if [[ ! -d "$CODEX_HOME" ]]; then
        if $DRY_RUN; then info "[DryRun] Would create directory: $CODEX_HOME"; return; fi
        mkdir -p "$CODEX_HOME"
    fi
    [[ -d "$CODEX_HOME" ]] || die "Codex home is not a directory: $CODEX_HOME"
}

# ── TOML helpers (mirrors Get-TomlLines, Get-CurrentProvider, etc.) ─
get_config_text() {
    ensure_home
    [[ -f "$CONFIG_TOML" ]] && cat "$CONFIG_TOML" || echo ""
}

get_toml_lines() {
    # Split text into lines, strip trailing blank lines
    local text="${1:-}"
    [[ -z "$text" ]] && return
    local lines=()
    while IFS= read -r line; do lines+=("$line"); done <<<"$text"
    # Strip trailing empty lines
    while [[ ${#lines[@]} -gt 0 && -z "${lines[${#lines[@]}-1]}" ]]; do
        unset 'lines[${#lines[@]}-1]'
    done
    printf '%s\n' "${lines[@]}"
}

trim_str() {
    local s="$1"
    s="${s#"${s%%[![:space:]]*}"}"
    s="${s%"${s##*[![:space:]]}"}"
    printf '%s' "$s"
}

line_key() {
    local line="$1" key
    line=$(trim_str "$line")
    [[ "$line" != *=* ]] && return 1
    key=$(trim_str "${line%%=*}")
    [[ -n "$key" ]] || return 1
    printf '%s' "$key"
}

get_current_provider() {
    local text="$1" line provider_lines=()
    while IFS= read -r line; do
        [[ "$line" =~ ^[[:space:]]*\[ ]] && break
        [[ "$line" =~ ^[[:space:]]*model_provider[[:space:]]*= ]] && provider_lines+=("$line")
    done < <(get_toml_lines "$text")

    if [[ ${#provider_lines[@]} -gt 1 ]]; then
        die "config.toml contains duplicate top-level model_provider entries."
    fi
    if [[ ${#provider_lines[@]} -eq 1 ]]; then
        if [[ "${provider_lines[0]}" =~ model_provider[[:space:]]*=[[:space:]]*\"([^\"]+)\" ]]; then
            echo "${BASH_REMATCH[1]}"; return
        fi
        if [[ "${provider_lines[0]}" =~ model_provider[[:space:]]*=[[:space:]]*\'([^\']+)\' ]]; then
            echo "${BASH_REMATCH[1]}"; return
        fi
        die "The top-level model_provider value in config.toml is not a valid quoted string."
    fi
    echo "$OFFICIAL_PROVIDER"
}

assert_config_structure() {
    local text="$1"
    # bash cannot hold NUL in variables; if file has one, cat piped to var would truncate
    if [[ -f "$CONFIG_TOML" ]]; then
        local null_check; null_check=$(tr -d '\000' <"$CONFIG_TOML" | wc -c | xargs)
        local orig_size; orig_size=$(wc -c <"$CONFIG_TOML" | xargs)
        [[ "$null_check" == "$orig_size" ]] || die "config.toml contains a NUL character."
    fi
    get_current_provider "$text" >/dev/null
    local count
    count=$(grep -Ec "^[[:space:]]*\[model_providers\.$COMPANY_PROVIDER\][[:space:]]*(#.*)?$" <<<"$text" || true)
    if [[ $count -gt 1 ]]; then
        die "config.toml contains duplicate [model_providers.$COMPANY_PROVIDER] sections."
    fi
}

# ── Config editing (mirrors Set-ManagedConfig) ──────────────────────
set_managed_config() {
    local text="$1" provider="$2"
    assert_config_structure "$text"

    local lines=() line first_header provider_line_idx section_start cleanup_action="Removed"
    $DRY_RUN && cleanup_action="Would remove"
    while IFS= read -r line; do lines+=("$line"); done < <(get_toml_lines "$text")

    # Locate first TOML section header
    first_header=${#lines[@]}
    for i in "${!lines[@]}"; do
        if [[ "${lines[$i]}" =~ ^[[:space:]]*\[ ]]; then first_header=$i; break; fi
    done

    # Remove top-level keys left by other provider setups. DeepSeek model lines
    # are only removed when they clearly reference a DeepSeek slug; unknown
    # model choices such as official ChatGPT models are preserved.
    local kept_lines=() key
    for ((i=0; i<${#lines[@]}; i++)); do
        if (( i < first_header )); then
            key=$(line_key "${lines[$i]}") || true
            if [[ -n "$key" && " $CONFLICT_KEYS " == *" $key "* ]]; then
                warn_err "$cleanup_action conflicting config entry: top-level $key"
                continue
            fi
            if [[ "${lines[$i]}" =~ $DEEPSEEK_MODEL_RE ]]; then
                warn_err "$cleanup_action conflicting config entry: top-level model (DeepSeek model)"
                continue
            fi
        fi
        kept_lines+=("${lines[$i]}")
    done
    lines=("${kept_lines[@]}")
    first_header=${#lines[@]}
    for i in "${!lines[@]}"; do
        if [[ "${lines[$i]}" =~ ^[[:space:]]*\[ ]]; then first_header=$i; break; fi
    done

    # Set model_provider in preamble
    provider_line_idx=-1
    for ((i=0; i<first_header; i++)); do
        if [[ "${lines[$i]}" =~ ^[[:space:]]*model_provider[[:space:]]*= ]]; then
            provider_line_idx=$i; break
        fi
    done
    if [[ $provider_line_idx -ge 0 ]]; then
        lines[$provider_line_idx]="model_provider = \"$provider\""
    else
        lines=("model_provider = \"$provider\"" "${lines[@]}")
    fi

    # Remove profile and old DeepSeek provider sections left by other setups.
    local kept2=() header trimmed
    for ((i=0; i<${#lines[@]}; i++)); do
        trimmed=$(trim_str "${lines[$i]}")
        if [[ "$trimmed" == '['* ]]; then
            header="${trimmed#[}"
            header="${header%%]*}"
            header=$(trim_str "$header")
            header="${header#\"}"; header="${header%\"}"
            header="${header#\'}"; header="${header%\'}"
            case "$header" in
                profiles|profiles.*|model_providers.deepseek|model_providers.deepseek.*)
                    warn_err "$cleanup_action conflicting config entry: section [$header]"
                    i=$((i+1))
                    while (( i < ${#lines[@]} )); do
                        trimmed=$(trim_str "${lines[$i]}")
                        [[ "$trimmed" == '['* ]] && { i=$((i-1)); break; }
                        i=$((i+1))
                    done
                    continue
                    ;;
            esac
        fi
        kept2+=("${lines[$i]}")
    done
    lines=("${kept2[@]}")

    # Manage [model_providers.shu] section
    local section_starts=()
    for i in "${!lines[@]}"; do
        if [[ "${lines[$i]}" =~ ^[[:space:]]*\[model_providers\.$COMPANY_PROVIDER\][[:space:]]*(#.*)?$ ]]; then
            section_starts+=("$i")
        fi
    done
    [[ ${#section_starts[@]} -gt 1 ]] && die "Duplicate [model_providers.$COMPANY_PROVIDER] sections are not supported."

    if [[ ${#section_starts[@]} -eq 0 ]]; then
        # Append section at end
        local last; last=$((${#lines[@]} - 1))
        if [[ ${#lines[@]} -gt 0 && -n "${lines[$last]}" ]]; then lines+=(""); fi
        lines+=("[model_providers.$COMPANY_PROVIDER]")
        for setting in "${COMPANY_SETTINGS[@]}"; do
            lines+=("${setting#*|}")
        done
    else
        # Merge managed keys, preserve unrecognised lines
        local start="${section_starts[0]}" end=${#lines[@]}
        for ((i=start+1; i<${#lines[@]}; i++)); do
            if [[ "${lines[$i]}" =~ ^[[:space:]]*\[ ]]; then end=$i; break; fi
        done

        local seen_keys="|" section=() managed_key managed_line
        for ((i=start+1; i<end; i++)); do
            local cur="${lines[$i]}" found_key=""
            for setting in "${COMPANY_SETTINGS[@]}"; do
                managed_key="${setting%%|*}"
                managed_line="${setting#*|}"
                if [[ "$cur" =~ ^[[:space:]]*${managed_key}[[:space:]]*= ]]; then
                    found_key="$managed_key"
                    break
                fi
            done
            if [[ -z "$found_key" ]]; then
                section+=("$cur")  # preserve unknown lines
            elif [[ "$seen_keys" != *"|$found_key|"* ]]; then
                section+=("$managed_line")
                seen_keys+="$found_key|"
            fi
        done
        # Append missing managed keys
        for setting in "${COMPANY_SETTINGS[@]}"; do
            managed_key="${setting%%|*}"
            managed_line="${setting#*|}"
            [[ "$seen_keys" == *"|$managed_key|"* ]] || section+=("$managed_line")
        done

        # Reassemble
        local new_lines=("${lines[@]:0:$((start+1))}")
        new_lines+=("${section[@]}")
        if [[ $end -lt ${#lines[@]} ]]; then
            new_lines+=("${lines[@]:$end}")
        fi
        lines=("${new_lines[@]}")
    fi

    # Join and verify
    local result; result=$(printf '%s\n' "${lines[@]}")
    result="${result%"${result##*[![:space:]]}"}"  # trim trailing whitespace
    result+=$'\n'

    assert_config_structure "$result"
    local verified; verified=$(get_current_provider "$result")
    [[ "$verified" == "$provider" ]] || die "Provider verification failed before writing config.toml."
    echo "$result"
}

# ── Atomic write (mirrors Write-ConfigAtomic) ───────────────────────
write_config_atomic() {
    local text="$1"
    assert_config_structure "$text"

    if $DRY_RUN; then
        info "[DryRun] Would write config.toml:"
        echo "$text"
        return
    fi

    ensure_home
    local temp; temp="$CODEX_HOME/.config.toml.$(uuidgen | tr -d '-').tmp"
    local rollback; rollback=""

    # Write temp file
    printf '%s\n' "$text" >"$temp"
    local verify; verify=$(cat "$temp")
    assert_config_structure "$verify"
    [[ "$verify" == "$text" ]] || die "Temporary configuration verification failed."

    # Backup current
    local backup; backup=$(new_config_backup)

    # Atomic replace
    if [[ -f "$CONFIG_TOML" ]]; then
        rollback="$CODEX_HOME/.config.toml.$(uuidgen | tr -d '-').rollback"
        ln "$CONFIG_TOML" "$rollback" 2>/dev/null || cp "$CONFIG_TOML" "$rollback"
    fi
    mv "$temp" "$CONFIG_TOML"
    [[ -f "$rollback" ]] && rm -f "$rollback"

    remove_excess_backups
    [[ -n "$backup" ]] && info "Backup: $backup"
}

# ── Set provider (mirrors Set-Provider) ─────────────────────────────
set_provider() {
    local provider="$1"
    with_lock set_provider_impl "$provider"
}

set_provider_impl() {
    local provider="$1"
    local current updated
    current=$(get_config_text) || return
    updated=$(set_managed_config "$current" "$provider") || return
    if [[ "$updated" == "$current" ]]; then
        info "Configuration is already up to date; no backup was created."
        return
    fi
    write_config_atomic "$updated"
}

# ── Backups (mirrors New-ConfigBackup, Remove-ExcessBackups, Get-BackupFiles) ─
get_backup_files() {
    if [[ ! -d "$BACKUP_DIR" ]]; then return; fi
    ls -1t "$BACKUP_DIR"/config.*.toml 2>/dev/null || true
}

new_config_backup() {
    if $DRY_RUN; then info "[DryRun] Would create a config backup."; return; fi
    if [[ ! -f "$CONFIG_TOML" ]]; then return; fi
    mkdir -p "$BACKUP_DIR"
    local stamp suffix
    stamp=$(date '+%Y%m%d-%H%M%S')
    suffix=$(uuidgen | tr -d '-' | cut -c1-8)
    local backup="$BACKUP_DIR/config.$stamp-$suffix.toml"
    cp "$CONFIG_TOML" "$backup"
    echo "$backup"
}

remove_excess_backups() {
    local files=() expired=()
    while IFS= read -r f; do [[ -n "$f" ]] && files+=("$f"); done < <(get_backup_files)
    if [[ ${#files[@]} -gt $BACKUP_LIMIT ]]; then
        expired=("${files[@]:$BACKUP_LIMIT}")
        for f in "${expired[@]}"; do rm -f "$f"; done
        info "Removed ${#expired[@]} old backup(s); keeping the latest $BACKUP_LIMIT."
    fi
}

# ── API key (mirrors Set-CompanyKey) ────────────────────────────────
get_company_key() {
    local key="${!KEY_NAME:-}"
    [[ -n "$key" ]] || return 1
    printf '%s' "$key"
}

set_company_key() {
    local key="$VALUE"

    if [[ "$(echo "$key" | tr '[:upper:]' '[:lower:]')" == "clear" ]]; then
        if $DRY_RUN; then
            info "[DryRun] Would remove $KEY_NAME."
            return
        fi
        # Remove from shell profiles
        for profile in "$HOME/.zshrc" "$HOME/.bash_profile" "$HOME/.bashrc"; do
            [[ -f "$profile" ]] && sed -i '' "/# setup-shu: $KEY_NAME/d" "$profile" 2>/dev/null || true
        done
        unset "$KEY_NAME"
        ok "$KEY_NAME was removed for the current user."
        warn "Restart running ChatGPT / Codex processes to remove their inherited copy."
        return
    fi

    if [[ -z "$key" ]]; then
        # Secure hidden input
        printf "Enter SHU API Key (input is hidden): "
        stty -echo; read -r key; stty echo; echo
    else
        warn "The API key may now be present in shell command history. Prefer: ./gpts.sh env"
    fi

    # Clipboard paste often appends CR (Windows/docs); strip before validation.
    key="${key//$'\r'/}"
    key="${key#"${key%%[![:space:]]*}"}"  # trim leading whitespace
    key="${key%"${key##*[![:space:]]}"}"  # trim trailing whitespace

    [[ -n "$key" ]] || die "API Key cannot be empty."
    # Reject remaining control chars (e.g. embedded newline from multi-line paste).
    if [[ "$key" =~ [[:cntrl:]] ]]; then
        die "API Key contains an invalid control character (often from paste). Re-enter a single-line key."
    fi

    if $DRY_RUN; then
        info "[DryRun] Would set $KEY_NAME for the current user."
        return
    fi

    # Write to shell profile for persistence
    local profile="$HOME/.zshrc"
    [[ "$SHELL" == *bash* ]] && profile="$HOME/.bash_profile"
    if grep -q "# setup-shu: $KEY_NAME" "$profile" 2>/dev/null; then
        sed -i '' "/# setup-shu: $KEY_NAME/d" "$profile"
    fi
    local quoted_key
    printf -v quoted_key '%q' "$key"
    printf 'export %s=%s  # setup-shu: %s\n' "$KEY_NAME" "$quoted_key" "$KEY_NAME" >>"$profile"

    export "$KEY_NAME=$key"

    ok "$KEY_NAME is set for the current macOS user."
    warn "The environment variable is readable by programs running as this user."
    warn "Fully exit and restart ChatGPT / Codex Desktop before use."
}

# ── Process detection (mirrors Get-RunningCodexProcesses) ────────────
get_running_codex() {
    pgrep -f "/Applications/ChatGPT.app" 2>/dev/null || true
    pgrep -i "codex" 2>/dev/null || true
}

show_restart_warning() {
    local running; running=$(get_running_codex)
    if [[ -n "$running" ]]; then
        warn "ChatGPT/Codex is currently running. Fully exit the app, then restart it."
    else
        warn "Start or restart ChatGPT / Codex Desktop for the change to take effect."
    fi
}

# ── Status (mirrors Show-Status) ─────────────────────────────────────
show_status() {
    local text provider user_key process_key company_block running
    text=$(get_config_text)
    provider=$(get_current_provider "$text")
    user_key=false; get_company_key >/dev/null && user_key=true
    process_key=false; [[ -n "${!KEY_NAME:-}" ]] && process_key=true
    company_block=false; grep -Eq "^[[:space:]]*\[model_providers\.$COMPANY_PROVIDER\][[:space:]]*(#.*)?$" <<<"$text" 2>/dev/null && company_block=true
    running=$(get_running_codex || true)
    local backups; backups=$(get_backup_files | wc -l | xargs)
    local running_flag="no"; [[ -n "$running" ]] && running_flag="yes"

    echo
    echo "===== Codex provider status ====="
    echo "Tool version     : $TOOL_VERSION"
    echo "Config           : $CONFIG_TOML"
    echo "Provider         : $provider"
    echo "Company config   : $($company_block && echo 'installed' || echo 'not installed')"
    echo "API key (stored) : $($user_key && echo 'set' || echo 'not set')"
    echo "API key (env)    : $($process_key && echo 'set' || echo 'not set')"
    echo "Backups          : $backups / $BACKUP_LIMIT retained"
    echo "Running app      : $running_flag"
    echo

    if [[ "$provider" == "$COMPANY_PROVIDER" ]] && ! $user_key; then
        warn "Company provider is selected, but $KEY_NAME is missing."
    fi
}

# ── Test connection (mirrors Test-CompanyConnection) ──────────────────
test_company_connection() {
    local key; key=$(get_company_key) || die "$KEY_NAME is not set. Run: gpts env"

    local uri="${COMPANY_BASE_URL%/}/models"
    info "Testing company gateway: $uri"

    local resp code body
    resp=$(curl -s -w '\n%{http_code}' --connect-timeout "$TIMEOUT_SEC" --max-time "$TIMEOUT_SEC" \
        -H "Authorization: Bearer $key" -H "Accept: application/json" "$uri" 2>&1) || {
        die "Company gateway test failed: could not reach $uri. Check your network/VPN."
    }

    code=$(echo "$resp" | tail -1)
    body=$(echo "$resp" | sed '$d')

    case "$code" in
        200) ok "Company gateway is reachable and accepted the API key (HTTP $code)." ;;
        401|403) die "Company gateway rejected the API key (HTTP $code)." ;;
        *) die "Company gateway test failed (HTTP $code)." ;;
    esac
}

# ── Backup management (mirrors Manage-Backups) ───────────────────────
manage_backups() {
    case "$VALUE_LOWER" in
        list)
            local files=()
            while IFS= read -r f; do [[ -n "$f" ]] && files+=("$f"); done < <(get_backup_files)
            if [[ ${#files[@]} -eq 0 ]]; then echo "No backups found."; return; fi
            for f in "${files[@]}"; do
                local sz ts
                sz=$(stat -f%z "$f" 2>/dev/null || stat -c%s "$f" 2>/dev/null || echo "?")
                ts=$(stat -f%Sm -t '%Y-%m-%d %H:%M:%S' "$f" 2>/dev/null || echo "?")
                printf "%-65s %8s bytes  %s\n" "$(basename "$f")" "$sz" "$ts"
            done
            ;;
        clear)
            if ! $FORCE; then
                printf "Delete all setup-shu configuration backups? Type DELETE to confirm: "
                read -r answer
                [[ "$answer" == "DELETE" ]] || { warn "Backup deletion cancelled."; return; }
            fi
            $DRY_RUN && { info "[DryRun] Would delete all backups."; return; }
            with_lock clear_backups_impl
            ;;
        restore)
            with_lock restore_backup_impl
            show_restart_warning
            ;;
        *) die "Usage: ./gpts.sh backup list|restore|clear [-Force]" ;;
    esac
}

clear_backups_impl() {
    local files=()
    while IFS= read -r f; do [[ -n "$f" ]] && files+=("$f"); done < <(get_backup_files)
    for f in "${files[@]}"; do
        local backup_root; backup_root="$(cd "$BACKUP_DIR" 2>/dev/null && pwd)"
        local fp; fp="$(cd "$(dirname "$f")" 2>/dev/null && pwd)/$(basename "$f")"
        [[ "$fp" == "$backup_root"/* ]] || die "Refusing to delete a file outside the backup directory: $fp"
        rm -f "$fp"
    done
    ok "Removed ${#files[@]} backup(s)."
}

restore_backup_impl() {
    local files=() source_path text current
    while IFS= read -r f; do [[ -n "$f" ]] && files+=("$f"); done < <(get_backup_files)
    [[ ${#files[@]} -gt 0 ]] || die "No configuration backup is available to restore."
    source_path="${files[0]}"

    if $DRY_RUN; then
        info "[DryRun] Would restore: $source_path"
        return
    fi

    text=$(cat "$source_path")
    assert_config_structure "$text"
    write_config_atomic "$text"
    ok "Restored: $source_path"
}

# ── Install (merged from the old install.sh) ────────────────────────
do_install() {
    local source_path target_path profile
    source_path="$(cd "$(dirname "$0")" 2>/dev/null && pwd)/$(basename "$0")" || die "Cannot determine my own path."
    target_path="$CODEX_HOME/gpts.sh"
    profile="$HOME/.zshrc"

    # Already installed at the target location?
    if [[ -f "$target_path" ]]; then
        local target_resolved
        target_resolved="$(cd "$(dirname "$target_path")" 2>/dev/null && pwd)/$(basename "$target_path")"
        if [[ "$source_path" == "$target_resolved" ]]; then
            info "Already installed at $target_path."
            setup_gpts_alias "$profile" "$target_path"
            return
        fi
    fi

    mkdir -p "$CODEX_HOME"
    cp "$source_path" "$target_path"
    chmod 755 "$target_path"
    setup_gpts_alias "$profile" "$target_path"
    ok "Installed: $target_path"
    warn 'Reload the shell (source ~/.zshrc), then run: gpts help'
}

setup_gpts_alias() {
    local profile="$1" target="$2"
    [[ -f "$profile" ]] || : > "$profile"
    if grep -Eq '^[[:space:]]*alias[[:space:]]+gpts=' "$profile"; then
        sed -i '' 's|^[[:space:]]*alias[[:space:]]\+gpts=.*$|alias gpts="$HOME/.codex/gpts.sh"|' "$profile"
    else
        printf '\nalias gpts="$HOME/.codex/gpts.sh"\n' >> "$profile"
    fi
}

# ── Help (mirrors Show-Help) ─────────────────────────────────────────
show_help() {
    # bash 3.2 + UTF-8: use ${VAR} when next to multibyte characters.
    cat <<EOF
================================================================
  gpts.sh  v${TOOL_VERSION}  — 公司 Codex / ChatGPT 网关切换
================================================================
【本文件顶部有更完整的中文说明】用编辑器打开本脚本即可阅读。
本脚本可单独拷走使用，不依赖仓库其它文件。Windows 请用 gpts.ps1。

意图:
  在官方 OpenAI 与公司中转 ${COMPANY_BASE_URL} 之间切换 Codex provider。
  配置: ~/.codex/config.toml    Key: ${KEY_NAME}（与 groks 共用）
  切换时会清理旧 DeepSeek 安装、profile/global override 等冲突字段；
  不写入 models.json，也不安装自定义 ChatGPT Desktop 模型目录。

推荐流程:
  ./gpts.sh env
  ./gpts.sh test
  ./gpts.sh use company
  然后彻底退出 ChatGPT/Codex（含托盘）再开

手动永久设置 Key:
  echo 'export SHU_API_KEY="你的 Key"' >> ~/.zshrc && source ~/.zshrc

常用命令:
  ./gpts.sh status | help
  ./gpts.sh use company | use official
  ./gpts.sh env | env clear | test
  ./gpts.sh backup list|restore|clear
  ./gpts.sh install          可选：装到 ~/.codex 并 alias gpts

故障速查:
  Permission denied     → chmod +x ./gpts.sh  或  bash ./gpts.sh ...
  无法验证开发者       → xattr -d com.apple.quarantine ./gpts.sh
  KEY not set           → ./gpts.sh env
  HTTP 401/403          → Key 错误/过期
  连不上网关            → VPN；curl 测 ${COMPANY_BASE_URL}/models
  配置锁                → rm -rf ~/.codex/.setup-shu.lock
  回滚                  → ./gpts.sh backup restore
  改完应用仍旧 provider → 全退托盘再开；status 看 Provider；查 config.toml

选项:
  -BackupLimit 3  -TimeoutSec 15  -CodexHome <path>
  -Force  -DryRun  -NoColor
EOF
}

# ── Main dispatcher (mirrors PS script switch) ──────────────────────
main() {
    ensure_home

    CMD_LOWER=$(echo "$COMMAND" | tr '[:upper:]' '[:lower:]')
    VALUE_LOWER=$(echo "$VALUE" | tr '[:upper:]' '[:lower:]')

    case "$CMD_LOWER" in
        env)
            set_company_key
            ;;
        test)
            test_company_connection
            ;;
        backup)
            manage_backups
            ;;
        use)
            case "$VALUE_LOWER" in
                company)
                    get_company_key >/dev/null || die "$KEY_NAME is not set. Run: gpts env"
                    set_provider "$COMPANY_PROVIDER"
                    ok "Codex is configured to use the company gateway."
                    ;;
                official)
                    set_provider "$OFFICIAL_PROVIDER"
                    ok "Codex is configured to use official OpenAI / ChatGPT authentication."
                    info "Sign in with ChatGPT Desktop or configure an official OpenAI API key."
                    ;;
                *) die "Usage: gpts use official|company" ;;
            esac
            show_restart_warning
            show_status
            ;;
        status)
            show_status
            ;;
        install)
            do_install
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            die "Unknown command: $COMMAND. Run: gpts help"
            ;;
    esac
}

main
