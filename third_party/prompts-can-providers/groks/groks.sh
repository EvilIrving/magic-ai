#!/usr/bin/env bash
#Requires: bash 3.2+, curl
#
# =============================================================================
#  groks.sh  —  公司 Grok Build 网关切换（shu）
# =============================================================================
#
# 【本文件即完整说明书】用任意文本编辑器打开即可阅读；不必回仓库、不必搜文档。
#  本脚本可单独拷走使用（微信 / 网盘 / U 盘均可），不依赖仓库里其它文件。
#
# ── 意图 ──────────────────────────────────────────────────────────────────
#  把 Grok Build 在「官方 SpaceXAI 订阅」与「公司中转 api.shu.cool」之间切换。
#  配置与启动模式保存在：~/.grok/config.toml、~/.grok/provider-mode
#  API Key 环境变量：SHU_API_KEY、SHU_DEEPSEEK_API_KEY、SHU_GROK_API_KEY
#
#  company 模式配置（摘要）：
#    [models]
#    default = "gpt-5.6-sol"
#
#  每个自定义模型通过 env_key 读取所属分组的环境变量。
#  GPT 与 Grok 模型使用 Responses API，DeepSeek 使用 Chat Completions API。
#  官方模式会移除这些覆盖，恢复官方 grok-4.5 与已有订阅登录。
#
# ── 依赖 ──────────────────────────────────────────────────────────────────
#  - macOS 或 Linux，bash 3.2+，curl
#  - 已安装 Grok Build CLI（可单独安装，不必装本仓库其它东西）
#    官方安装示例：  curl -fsSL https://x.ai/cli/install.sh | bash
#  - 不需要管理员 / sudo（除非你把 Grok 装在了无权限目录）
#
# ── 怎么运行（在脚本所在文件夹）──────────────────────────────────────────
#    chmod +x ./groks.sh          # 仅首次需要
#    ./groks.sh help
#    ./groks.sh status
#
#    若提示 Permission denied：
#      chmod +x ./groks.sh
#    若提示 command not found: ./groks.sh：
#      bash ./groks.sh help
#
# ── macOS 拦截（「无法验证开发者」/ 打不开）──────────────────────────────
#  从微信 / 网盘 / 浏览器 / AirDrop 拿到的脚本常被 Gatekeeper 隔离。任选：
#
#    方案 A — 去掉隔离标记后执行：
#      xattr -d com.apple.quarantine ./groks.sh
#      chmod +x ./groks.sh
#      ./groks.sh help
#
#    方案 B — 不依赖可执行位，直接用 bash：
#      bash ./groks.sh help
#
#    本机自己新建/复制出来的文件一般没有隔离问题。
#
# ── 推荐流程 ──────────────────────────────────────────────────────────────
#    ./groks.sh env openai       # 安全输入默认 GPT 模型分组 Key（隐藏输入）
#    ./groks.sh test openai      # 测网关 GET /v1/models
#    ./groks.sh use company      # 默认模型切到中转
#    然后彻底退出 Grok，新开终端后直接运行 grok
#
#    切回官方：
#      ./groks.sh use official
#      # 并确保已 grok login
#
# ── 可选 install（装到 ~/.grok 并加 alias）────────────────────────────────
#    ./groks.sh install
#    source ~/.zshrc             # 或重开终端
#    groks help
#
#    install 会：
#      - 复制到  ~/.grok/groks.sh
#      - 写入 groks 别名，并确保 ~/.local/bin/grok 直接指向官方 Grok 二进制
#    不 install 也可以始终在当前目录用 ./groks.sh 运行。
#
# ── 常用命令 ──────────────────────────────────────────────────────────────
#    ./groks.sh status
#    ./groks.sh env [openai|deepseek|grok] [clear]
#    ./groks.sh test [openai|deepseek|grok]
#    ./groks.sh use company | use official
#    ./groks.sh backup list | restore | clear
#    ./groks.sh help
#
# ── 改完「没生效」时 ──────────────────────────────────────────────────────
#  1) 是否 install 后没有重开终端 / source ~/.zshrc
#  2) ./groks.sh status  看 Provider mode 是否 company、对应 API key 是否 set
#  3) ./groks.sh test openai  是否 HTTP 200 且包含 gpt-5.6-sol
#  4) 打开 ~/.grok/config.toml 确认 default = "gpt-5.6-sol"
#  5) 新开终端后确认 SHU_API_KEY 已进入环境
#
# ── 常见故障 ──────────────────────────────────────────────────────────────
#  · 缺少 curl
#      macOS 一般自带；Linux: 安装 curl 包
#  · SHU_API_KEY is not set
#      先 ./groks.sh env openai；若刚写过 profile，先 source ~/.zshrc 或重开终端
#  · Company gateway rejected (401/403)
#      Key 错误或过期；向网关管理员重新申请；勿用官方 XAI_API_KEY 顶替
#  · 连不上网关
#      检查 VPN / 公司网络；curl -sS -H "Authorization: Bearer $SHU_API_KEY" \
#        https://api.shu.cool/v1/models
#  · Grok 仍走官方或启动后无输出
#      确认 provider-mode 为 company、config.toml 有 groks-managed-catalog，并彻底重启
#  · config 被锁 / Another groks process
#      等几秒重试；若异常退出残留锁：rm -rf ~/.grok/.groks.lock
#  · 想恢复改之前的 config
#      ./groks.sh backup list
#      ./groks.sh backup restore
#
# ── 安全说明 ──────────────────────────────────────────────────────────────
#  - 不需要管理员权限
#  - 分组 Key 写在用户 shell profile（如 ~/.zshrc），本用户其它程序可读
#  - 不要把 key 贴进聊天记录；优先用 env 隐藏输入
#  - 配置修改有文件锁、校验、备份与原子替换
#
# ── 命令行再打一遍说明 ────────────────────────────────────────────────────
#    ./groks.sh help
#
# =============================================================================
set -euo pipefail

# ── Parameters ─────────────────────────────────────────────────────
COMMAND="status"
VALUE=""
EXTRA_VALUE=""

GROK_HOME="${GROK_HOME:-$HOME/.grok}"
BACKUP_LIMIT=3
TIMEOUT_SEC=15
FORCE=false
DRY_RUN=false
NO_COLOR=false

# ── Internal constants ─────────────────────────────────────────────
CONFIG_TOML="$GROK_HOME/config.toml"
MODE_PATH="$GROK_HOME/provider-mode"
BACKUP_DIR="$GROK_HOME/provider-switch-backups"
LOCK_DIR="$GROK_HOME/.groks.lock"
MODEL_BLOCK_START="# groks-managed-catalog-start"
MODEL_BLOCK_END="# groks-managed-catalog-end"
OPENAI_KEY_NAME="SHU_API_KEY"
DEEPSEEK_KEY_NAME="SHU_DEEPSEEK_API_KEY"
GROK_KEY_NAME="SHU_GROK_API_KEY"
COMPANY_MODEL_KEY="gpt-5.6-sol"
OFFICIAL_MODEL="grok-4.5"
DEFAULT_REASONING_EFFORT="high"
COMPANY_BASE_URL="https://api.shu.cool/v1"
TOOL_VERSION="1.4.0"
readonly COMPANY_MODEL_IDS=(
    "gpt-5.6-sol"
    "gpt-5.6-terra"
    "gpt-5.6-luna"
    "gpt-5.5"
    "gpt-5.4"
    "deepseek-v4-pro"
    "deepseek-v4-flash-vision-exp"
    "grok-4.6"
    "grok-4.5"
    "grok-4.5-latest"
    "grok-build-latest"
    "grok-composer-2.5-fast"
    "shu"
)

# ── Argument parsing ───────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        -GrokHome)        GROK_HOME="$2"; shift 2 ;;
        -BackupLimit)     BACKUP_LIMIT="$2"; shift 2 ;;
        -TimeoutSec)      TIMEOUT_SEC="$2"; shift 2 ;;
        -Force)           FORCE=true; shift ;;
        -DryRun)          DRY_RUN=true; shift ;;
        -NoColor)         NO_COLOR=true; shift ;;
        -h|--help|-?)     COMMAND="help"; shift ;;
        *)
            if [[ -z "${POS1:-}" ]]; then POS1="$1";
            elif [[ -z "${POS2:-}" ]]; then POS2="$1";
            elif [[ -z "${POS3:-}" ]]; then POS3="$1"; fi
            shift
            ;;
    esac
done
COMMAND="${POS1:-$COMMAND}"
VALUE="${POS2:-$VALUE}"
EXTRA_VALUE="${POS3:-$EXTRA_VALUE}"

CONFIG_TOML="$GROK_HOME/config.toml"
MODE_PATH="$GROK_HOME/provider-mode"
BACKUP_DIR="$GROK_HOME/provider-switch-backups"
LOCK_DIR="$GROK_HOME/.groks.lock"

# ── Output helpers ──────────────────────────────────────────────────
ok()   { if $NO_COLOR; then echo "[OK]  $*"; else printf '\033[32m%s\033[0m\n' "$*"; fi; }
info() { if $NO_COLOR; then echo "[INFO] $*"; else printf '\033[36m%s\033[0m\n' "$*"; fi; }
warn() { if $NO_COLOR; then echo "[WARN] $*"; else printf '\033[33m%s\033[0m\n' "$*"; fi; }
err()  { printf '\033[31m[ERR]  %s\033[0m\n' "$*" >&2; }
die()  { err "$@"; exit 1; }

# ── Locking ─────────────────────────────────────────────────────────
acquire_lock() {
    local deadline; deadline=$(($(date +%s) + 10))
    while ! mkdir "$LOCK_DIR" 2>/dev/null; do
        if [[ $(date +%s) -gt $deadline ]]; then
            die "Another groks process is modifying the configuration. Try again later."
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

# ── Ensure Grok home exists ─────────────────────────────────────────
ensure_home() {
    [[ -n "$GROK_HOME" ]] || die "Grok home path is empty."
    if [[ ! -d "$GROK_HOME" ]]; then
        if $DRY_RUN; then info "[DryRun] Would create directory: $GROK_HOME"; return; fi
        mkdir -p "$GROK_HOME"
    fi
    [[ -d "$GROK_HOME" ]] || die "Grok home is not a directory: $GROK_HOME"
}

# ── TOML helpers ────────────────────────────────────────────────────
get_config_text() {
    ensure_home
    [[ -f "$CONFIG_TOML" ]] && cat "$CONFIG_TOML" || echo ""
}

# Read [models].default (Grok's active model id for new sessions)
get_current_default() {
    local text="$1"
    local in_models=false line
    while IFS= read -r line || [[ -n "$line" ]]; do
        if [[ "$line" =~ ^[[:space:]]*\[models\][[:space:]]*(#.*)?$ ]]; then
            in_models=true
            continue
        fi
        if $in_models && [[ "$line" =~ ^[[:space:]]*\[ ]]; then
            break
        fi
        if $in_models && [[ "$line" =~ ^[[:space:]]*default[[:space:]]*= ]]; then
            if [[ "$line" =~ default[[:space:]]*=[[:space:]]*\"([^\"]+)\" ]]; then
                echo "${BASH_REMATCH[1]}"; return
            fi
            if [[ "$line" =~ default[[:space:]]*=[[:space:]]*\'([^\']+)\' ]]; then
                echo "${BASH_REMATCH[1]}"; return
            fi
            die "The [models].default value in config.toml is not a valid quoted string."
        fi
    done <<<"$text"
    echo "$OFFICIAL_MODEL"
}

assert_config_structure() {
    local text="$1"
    if [[ -f "$CONFIG_TOML" ]]; then
        local null_check orig_size
        null_check=$(tr -d '\000' <"$CONFIG_TOML" | wc -c | xargs)
        orig_size=$(wc -c <"$CONFIG_TOML" | xargs)
        [[ "$null_check" == "$orig_size" ]] || die "config.toml contains a NUL character."
    fi
    get_current_default "$text" >/dev/null
    local count
    count=$(grep -Ec "^[[:space:]]*\[models\][[:space:]]*(#.*)?$" <<<"$text" || true)
    if [[ $count -gt 1 ]]; then
        die "config.toml contains duplicate [models] sections."
    fi
    local starts ends
    starts=$(grep -Fxc "$MODEL_BLOCK_START" <<<"$text" || true)
    ends=$(grep -Fxc "$MODEL_BLOCK_END" <<<"$text" || true)
    [[ "$starts" == "$ends" && "$starts" -le 1 ]] || die "config.toml contains an incomplete groks-managed catalog block."
}

# Working buffers for config edits (global: bash 3.2 has no nameref)
__cfg_lines=()
__managed_result=""

# Set or update a quoted string setting in [models].
upsert_models_string_setting() {
    local setting_name="$1" setting_value="$2"
    local section_starts=() i

    if [[ ${#__cfg_lines[@]} -gt 0 ]]; then
        for i in "${!__cfg_lines[@]}"; do
            if [[ "${__cfg_lines[$i]}" =~ ^[[:space:]]*\[models\][[:space:]]*(#.*)?$ ]]; then
                section_starts+=("$i")
            fi
        done
    fi
    [[ ${#section_starts[@]} -gt 1 ]] && die "Duplicate [models] sections are not supported."

    if [[ ${#section_starts[@]} -eq 0 ]]; then
        if [[ ${#__cfg_lines[@]} -gt 0 ]]; then
            local last; last=$((${#__cfg_lines[@]} - 1))
            if [[ -n "${__cfg_lines[$last]:-}" ]]; then __cfg_lines+=(""); fi
        fi
        __cfg_lines+=("[models]" "$setting_name = \"$setting_value\"")
        return
    fi

    local start="${section_starts[0]}" end=${#__cfg_lines[@]}
    for ((i=start+1; i<${#__cfg_lines[@]}; i++)); do
        if [[ "${__cfg_lines[$i]}" =~ ^[[:space:]]*\[ ]]; then end=$i; break; fi
    done

    local setting_line="$setting_name = \"$setting_value\""
    local found=false section_body=()
    for ((i=start+1; i<end; i++)); do
        local cur="${__cfg_lines[$i]}"
        if [[ "$cur" =~ ^[[:space:]]*${setting_name}[[:space:]]*= ]]; then
            if ! $found; then
                section_body+=("$setting_line")
                found=true
            fi
        else
            section_body+=("$cur")
        fi
    done
    if ! $found; then
        section_body+=("$setting_line")
    fi

    local new_lines=("${__cfg_lines[@]:0:$((start+1))}")
    if [[ ${#section_body[@]} -gt 0 ]]; then
        new_lines+=("${section_body[@]}")
    fi
    if [[ $end -lt ${#__cfg_lines[@]} ]]; then
        new_lines+=("${__cfg_lines[@]:$end}")
    fi
    if [[ ${#new_lines[@]} -gt 0 ]]; then
        __cfg_lines=("${new_lines[@]}")
    else
        __cfg_lines=()
    fi
}

upsert_default_model() {
    upsert_models_string_setting "default" "$1"
}

upsert_default_reasoning_effort() {
    upsert_models_string_setting "default_reasoning_effort" "$DEFAULT_REASONING_EFFORT"
}

remove_managed_model_block() {
    local start="" end="" i
    if [[ ${#__cfg_lines[@]} -gt 0 ]]; then
        for i in "${!__cfg_lines[@]}"; do
            case "${__cfg_lines[$i]}" in
                "$MODEL_BLOCK_START"|"# groks-managed-model-start") start="$i" ;;
                "$MODEL_BLOCK_END"|"# groks-managed-model-end") end="$i" ;;
            esac
        done
    fi
    if [[ -z "$start" && -z "$end" ]]; then return; fi
    [[ -n "$start" && -n "$end" && "$start" -lt "$end" ]] || die "Invalid groks-managed model block."

    local new_lines=()
    if [[ "$start" -gt 0 ]]; then
        new_lines=("${__cfg_lines[@]:0:$start}")
    fi
    if [[ "$end" -lt $((${#__cfg_lines[@]} - 1)) ]]; then
        new_lines+=("${__cfg_lines[@]:$((end+1))}")
    fi
    while [[ ${#new_lines[@]} -gt 0 && -z "${new_lines[${#new_lines[@]}-1]}" ]]; do
        local last_index
        last_index=$((${#new_lines[@]} - 1))
        unset "new_lines[$last_index]"
    done
    while [[ ${#new_lines[@]} -gt 0 && -z "${new_lines[0]}" ]]; do
        if [[ ${#new_lines[@]} -eq 1 ]]; then
            new_lines=()
        else
            new_lines=("${new_lines[@]:1}")
        fi
    done
    if [[ ${#new_lines[@]} -gt 0 ]]; then
        __cfg_lines=("${new_lines[@]}")
    else
        __cfg_lines=()
    fi
}

is_company_model_header() {
    local compact="$1" model_id
    compact="${compact//[[:space:]]/}"
    compact="${compact//\"/}"
    for model_id in "${COMPANY_MODEL_IDS[@]}"; do
        [[ "$compact" == "[model.$model_id]" ]] && return 0
        [[ "$compact" == "[[model.${model_id}.reasoning_efforts]]" ]] && return 0
    done
    return 1
}

remove_company_model_sections() {
    local new_lines=() line skipping=false
    if [[ ${#__cfg_lines[@]} -gt 0 ]]; then
        for line in "${__cfg_lines[@]}"; do
            if [[ "$line" =~ ^[[:space:]]*\[ ]]; then
                if is_company_model_header "$line"; then
                    skipping=true
                    continue
                fi
                skipping=false
            fi
            $skipping || new_lines+=("$line")
        done
    fi
    while [[ ${#new_lines[@]} -gt 0 && -z "${new_lines[${#new_lines[@]}-1]}" ]]; do
        local last_index
        last_index=$((${#new_lines[@]} - 1))
        unset "new_lines[$last_index]"
    done
    if [[ ${#new_lines[@]} -gt 0 ]]; then
        __cfg_lines=("${new_lines[@]}")
    else
        __cfg_lines=()
    fi
}

add_model_base() {
    local model_id="$1" model_name="$2" key_name="$3" backend="$4" context="$5" max_tokens="$6"
    __cfg_lines+=(
        "[model.\"$model_id\"]"
        "model = \"$model_id\""
        "name = \"$model_name\""
        "base_url = \"$COMPANY_BASE_URL\""
        "env_key = \"$key_name\""
        "api_backend = \"$backend\""
        "context_window = $context"
        "max_completion_tokens = $max_tokens"
        "supports_reasoning_effort = true"
        "reasoning_effort = \"$DEFAULT_REASONING_EFFORT\""
    )
}

add_reasoning_effort_table() {
    local model_id="$1" values="$2"
    local value label default
    for value in $values; do
        case "$value" in
            none) label="None" ;;
            low) label="Low Effort" ;;
            medium) label="Medium Effort" ;;
            high) label="High Effort" ;;
            xhigh) label="Extra High Effort" ;;
            max) label="Max Effort" ;;
            *) die "Unsupported reasoning effort value: $value" ;;
        esac
        default=false
        [[ "$value" == "$DEFAULT_REASONING_EFFORT" ]] && default=true
        __cfg_lines+=(
            "[[model.\"$model_id\".reasoning_efforts]]"
            "value = \"$value\""
            "label = \"$label\""
            "default = $default"
            ""
        )
    done
}

add_managed_company_model_block() {
    if [[ ${#__cfg_lines[@]} -gt 0 ]]; then __cfg_lines+=(""); fi
    __cfg_lines+=("$MODEL_BLOCK_START")

    add_model_base "gpt-5.6-sol" "GPT-5.6 Sol" "$OPENAI_KEY_NAME" "responses" 1050000 128000
    __cfg_lines+=("")
    add_reasoning_effort_table "gpt-5.6-sol" "none low medium high xhigh max"

    add_model_base "gpt-5.6-terra" "GPT-5.6 Terra" "$OPENAI_KEY_NAME" "responses" 1050000 128000
    __cfg_lines+=("")
    add_reasoning_effort_table "gpt-5.6-terra" "none low medium high xhigh max"

    add_model_base "gpt-5.6-luna" "GPT-5.6 Luna" "$OPENAI_KEY_NAME" "responses" 1050000 128000
    __cfg_lines+=("")
    add_reasoning_effort_table "gpt-5.6-luna" "none low medium high xhigh max"

    add_model_base "gpt-5.5" "GPT-5.5" "$OPENAI_KEY_NAME" "responses" 1050000 128000
    __cfg_lines+=("")
    add_reasoning_effort_table "gpt-5.5" "none low medium high xhigh"

    add_model_base "deepseek-v4-pro" "DeepSeek V4 Pro" "$DEEPSEEK_KEY_NAME" "chat_completions" 1000000 384000
    __cfg_lines+=("")
    add_reasoning_effort_table "deepseek-v4-pro" "low medium high xhigh max"

    add_model_base "deepseek-v4-flash-vision-exp" "DeepSeek V4 Flash Vision Exp" "$DEEPSEEK_KEY_NAME" "responses" 1000000 384000
    __cfg_lines+=("")
    add_reasoning_effort_table "deepseek-v4-flash-vision-exp" "low medium high xhigh max"

    add_model_base "grok-4.6" "Grok 4.6" "$GROK_KEY_NAME" "responses" 500000 128000
    __cfg_lines+=("")
    add_reasoning_effort_table "grok-4.6" "low medium high xhigh"

    add_model_base "grok-4.5" "Grok 4.5" "$GROK_KEY_NAME" "responses" 500000 128000
    __cfg_lines+=("")
    add_reasoning_effort_table "grok-4.5" "low medium high xhigh"

    __cfg_lines+=("$MODEL_BLOCK_END")
}

# ── Config editing ──────────────────────────────────────────────────
set_managed_config() {
    local text="$1" mode="$2"
    assert_config_structure "$text"

    __cfg_lines=()
    local line
    if grep -q '[^[:space:]]' <<<"$text"; then
        while IFS= read -r line || [[ -n "$line" ]]; do
            __cfg_lines+=("$line")
        done <<<"$text"
    fi

    remove_managed_model_block
    remove_company_model_sections
    if [[ "$mode" == "company" ]]; then
        add_managed_company_model_block
        upsert_default_model "$COMPANY_MODEL_KEY"
    else
        upsert_default_model "$OFFICIAL_MODEL"
    fi
    upsert_default_reasoning_effort

    local result=""
    if [[ ${#__cfg_lines[@]} -gt 0 ]]; then
        result=$(printf '%s\n' "${__cfg_lines[@]}")
        result="${result%"${result##*[![:space:]]}"}"
    fi
    result+=$'\n'

    assert_config_structure "$result"
    local verified; verified=$(get_current_default "$result")
    if [[ "$mode" == "company" ]]; then
        [[ "$verified" == "$COMPANY_MODEL_KEY" ]] || die "Default model verification failed before writing config.toml (got: $verified)."
    else
        [[ "$verified" == "$OFFICIAL_MODEL" ]] || die "Default model verification failed before writing config.toml (got: $verified)."
    fi
    __managed_result="${result%$'\n'}"
}

# ── Atomic write ────────────────────────────────────────────────────
write_config_atomic() {
    local text="$1"
    assert_config_structure "$text"

    if $DRY_RUN; then
        info "[DryRun] Would write config.toml:"
        echo "$text"
        return
    fi

    ensure_home
    local temp; temp="$GROK_HOME/.config.toml.$(uuidgen | tr -d '-').tmp"
    local rollback; rollback=""

    printf '%s' "$text" >"$temp"
    local verify; verify=$(cat "$temp")
    assert_config_structure "$verify"
    [[ "$verify" == "$text" ]] || die "Temporary configuration verification failed."

    local backup; backup=$(new_config_backup)

    if [[ -f "$CONFIG_TOML" ]]; then
        rollback="$GROK_HOME/.config.toml.$(uuidgen | tr -d '-').rollback"
        ln "$CONFIG_TOML" "$rollback" 2>/dev/null || cp "$CONFIG_TOML" "$rollback"
    fi
    mv "$temp" "$CONFIG_TOML"
    if [[ -f "$rollback" ]]; then rm -f "$rollback"; fi

    remove_excess_backups
    if [[ -n "${backup:-}" ]]; then info "Backup: $backup"; fi
}

set_provider() {
    local mode="$1"
    with_lock set_provider_impl "$mode"
}

set_provider_impl() {
    local mode="$1"
    local current updated
    current=$(get_config_text) || return
    set_managed_config "$current" "$mode" || return
    updated="$__managed_result"
    if [[ "$updated" == "$current" ]]; then
        info "Configuration is already up to date; no backup was created."
    else
        write_config_atomic "$updated"
    fi
    write_provider_mode "$mode"
}

# ── Provider mode ──────────────────────────────────────────────────
get_provider_mode() {
    local text default
    text=$(get_config_text)
    default=$(get_current_default "$text")
    if [[ "$default" == "$COMPANY_MODEL_KEY" ]] &&
       grep -Fqx "$MODEL_BLOCK_START" <<<"$text"; then
        echo "company"
    else
        echo "official"
    fi
}

write_provider_mode() {
    local mode="$1"
    [[ "$mode" == "company" || "$mode" == "official" ]] ||
        die "Invalid provider mode: $mode"
    if $DRY_RUN; then
        info "[DryRun] Would write provider mode: $mode"
        return
    fi
    ensure_home
    local temp="$GROK_HOME/.provider-mode.$(uuidgen | tr -d '-').tmp"
    printf '%s\n' "$mode" >"$temp"
    mv "$temp" "$MODE_PATH"
}

# ── Backups ─────────────────────────────────────────────────────────
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

# ── Provider API keys ──────────────────────────────────────────────
get_persistent_key() {
    local key_name="$1"
    local key="${!key_name:-}"
    if [[ -n "$key" ]]; then
        printf '%s' "$key"
        return
    fi

    local profile line value
    for profile in "$HOME/.zshrc" "$HOME/.bash_profile" "$HOME/.bashrc"; do
        [[ -f "$profile" ]] || continue
        line=$(grep -E "^[[:space:]]*export[[:space:]]+$key_name=" "$profile" 2>/dev/null | tail -1 || true)
        [[ -n "$line" ]] || continue
        value="${line#export $key_name=}"
        [[ "$value" != "$line" ]] || continue
        value="${value%%  # *}"
        if [[ "$value" == \'*\' && "$value" == *\' ]]; then
            value="${value:1:${#value}-2}"
        fi
        [[ -n "$value" ]] || continue
        printf '%s' "$value"
        return
    done
    return 1
}

select_key_group() {
    local group="${1:-openai}"
    case "$group" in
        openai|"")
            SELECTED_KEY_NAME="$OPENAI_KEY_NAME"
            SELECTED_MODEL_ID="$COMPANY_MODEL_KEY"
            SELECTED_GROUP_LABEL="OpenAI"
            ;;
        deepseek)
            SELECTED_KEY_NAME="$DEEPSEEK_KEY_NAME"
            SELECTED_MODEL_ID="deepseek-v4-pro"
            SELECTED_GROUP_LABEL="DeepSeek"
            ;;
        grok)
            SELECTED_KEY_NAME="$GROK_KEY_NAME"
            SELECTED_MODEL_ID="grok-4.5"
            SELECTED_GROUP_LABEL="Grok"
            ;;
        *)
            die "Unknown key group: $group. Use openai, deepseek, or grok."
            ;;
    esac
}

set_company_key() {
    local group="${VALUE_LOWER:-openai}"
    local action="${EXTRA_VALUE:-}"
    if [[ "$group" == "clear" ]]; then
        group="openai"
        action="clear"
    fi
    select_key_group "$group"
    local key_name="$SELECTED_KEY_NAME"
    local key_marker="groks: $key_name"
    local key=""

    if [[ "$(echo "$action" | tr '[:upper:]' '[:lower:]')" == "clear" ]]; then
        if $DRY_RUN; then
            info "[DryRun] Would remove $key_name."
            return
        fi
        for profile in "$HOME/.zshrc" "$HOME/.bash_profile" "$HOME/.bashrc"; do
            [[ -f "$profile" ]] && sed -i '' "/# $key_marker/d" "$profile" 2>/dev/null || true
        done
        unset "$key_name"
        ok "$key_name was removed for the current user."
        warn "Restart running Grok processes to remove their inherited copy."
        return
    fi

    if [[ -z "$action" ]]; then
        printf "Enter SHU %s API Key (input is hidden): " "$SELECTED_GROUP_LABEL"
        stty -echo; read -r key; stty echo; echo
    else
        key="$action"
        warn "The API key may now be present in shell command history. Prefer: ./groks.sh env"
    fi

    # Clipboard paste often appends CR (Windows/docs); strip before validation.
    key="${key//$'\r'/}"
    key="${key#"${key%%[![:space:]]*}"}"
    key="${key%"${key##*[![:space:]]}"}"

    [[ -n "$key" ]] || die "API Key cannot be empty."
    # Reject remaining control chars (e.g. embedded newline from multi-line paste).
    if [[ "$key" =~ [[:cntrl:]] ]]; then
        die "API Key contains an invalid control character (often from paste). Re-enter a single-line key."
    fi

    if $DRY_RUN; then
        info "[DryRun] Would set $key_name for the current user."
        return
    fi

    local profile="$HOME/.zshrc"
    [[ "$SHELL" == *bash* ]] && profile="$HOME/.bash_profile"
    local candidate
    for candidate in "$HOME/.zshrc" "$HOME/.bash_profile" "$HOME/.bashrc"; do
        [[ -f "$candidate" ]] && sed -i '' "/# $key_marker/d" "$candidate" 2>/dev/null || true
    done
    local quoted_key
    printf -v quoted_key '%q' "$key"
    printf 'export %s=%s  # %s\n' "$key_name" "$quoted_key" "$key_marker" >>"$profile"

    export "$key_name=$key"

    ok "$key_name is set for the current macOS user."
    info "This key is dedicated to the $SELECTED_GROUP_LABEL model group."
    warn "The environment variable is readable by programs running as this user."
    warn "Fully exit and restart the Grok TUI before use."
}

# ── Process detection ───────────────────────────────────────────────
get_running_grok() {
    {
        pgrep -x "grok" 2>/dev/null || true
        pgrep -f "[/]grok " 2>/dev/null || true
    } | sort -u
    return 0
}

show_restart_warning() {
    local running
    running=$(get_running_grok || true)
    if [[ -n "$running" ]]; then
        warn "Grok is currently running. Fully exit the app, then restart it."
    else
        warn "Start or restart Grok for the change to take effect."
    fi
}

# ── Status ──────────────────────────────────────────────────────────
show_status() {
    local text default running provider_mode
    text=$(get_config_text)
    default=$(get_current_default "$text")
    running=$(get_running_grok || true)
    local backups; backups=$(get_backup_files | wc -l | xargs)
    local running_flag="no"
    if [[ -n "$running" ]]; then running_flag="yes"; fi
    provider_mode=$(get_provider_mode)

    echo
    echo "===== Grok provider status ====="
    echo "Tool version     : $TOOL_VERSION"
    echo "Config           : $CONFIG_TOML"
    echo "Provider mode    : $provider_mode"
    echo "Default model    : $default"
    echo "OpenAI key       : $(get_persistent_key "$OPENAI_KEY_NAME" >/dev/null 2>&1 && echo 'set' || echo 'not set')"
    echo "DeepSeek key     : $(get_persistent_key "$DEEPSEEK_KEY_NAME" >/dev/null 2>&1 && echo 'set' || echo 'not set')"
    echo "Grok key         : $(get_persistent_key "$GROK_KEY_NAME" >/dev/null 2>&1 && echo 'set' || echo 'not set')"
    echo "Gateway          : $COMPANY_BASE_URL"
    echo "Backups          : $backups / $BACKUP_LIMIT retained"
    echo "Running app      : $running_flag"
    echo

    if [[ "$provider_mode" == "company" ]] &&
       ! get_persistent_key "$OPENAI_KEY_NAME" >/dev/null 2>&1; then
        warn "Company model is selected, but $OPENAI_KEY_NAME is missing."
    fi
}

# ── Test connection ─────────────────────────────────────────────────
test_company_connection() {
    select_key_group "${VALUE_LOWER:-openai}"
    local key
    key=$(get_persistent_key "$SELECTED_KEY_NAME") ||
        die "$SELECTED_KEY_NAME is not set. Run: groks env ${VALUE_LOWER:-openai}"

    local uri="${COMPANY_BASE_URL%/}/models"
    info "Testing company gateway: $uri"

    local resp code body
    resp=$(curl -s -w '\n%{http_code}' --connect-timeout "$TIMEOUT_SEC" --max-time "$TIMEOUT_SEC" \
        -H "Authorization: Bearer $key" -H "Accept: application/json" "$uri" 2>&1) || {
        die "Company gateway test failed: could not reach $uri. Check your network/VPN."
    }

    code=$(echo "$resp" | tail -1)
    body="${resp%$'\n'*}"

    case "$code" in
        200)
            if ! grep -Fq "\"id\":\"$SELECTED_MODEL_ID\"" <<<"${body//[[:space:]]/}"; then
                die "Company gateway accepted the API key, but this key cannot access $SELECTED_MODEL_ID."
            fi
            ok "Company gateway accepted the $SELECTED_GROUP_LABEL key and exposes $SELECTED_MODEL_ID (HTTP $code)."
            ;;
        401|403) die "Company gateway rejected the API key (HTTP $code)." ;;
        *) die "Company gateway test failed (HTTP $code)." ;;
    esac
}

# ── Backup management ───────────────────────────────────────────────
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
                printf "Delete all groks configuration backups? Type DELETE to confirm: "
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
        *) die "Usage: ./groks.sh backup list|restore|clear [-Force]" ;;
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
    local files=() source_path text
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

# ── Install ─────────────────────────────────────────────────────────
do_install() {
    local source_path target_path profile
    source_path="$(cd "$(dirname "$0")" 2>/dev/null && pwd)/$(basename "$0")" || die "Cannot determine my own path."
    target_path="$GROK_HOME/groks.sh"
    profile="$HOME/.zshrc"

    if [[ -f "$target_path" ]]; then
        local target_resolved
        target_resolved="$(cd "$(dirname "$target_path")" 2>/dev/null && pwd)/$(basename "$target_path")"
        if [[ "$source_path" == "$target_resolved" ]]; then
            info "Already installed at $target_path."
            setup_shell_aliases "$profile"
            restore_grok_command
            return
        fi
    fi

    mkdir -p "$GROK_HOME"
    cp "$source_path" "$target_path"
    chmod 755 "$target_path"
    setup_shell_aliases "$profile"
    restore_grok_command
    ok "Installed: $target_path"
    warn 'Reload the shell (source ~/.zshrc), then run: grok'
}

setup_shell_aliases() {
    local profile="$1"
    [[ -f "$profile" ]] || : > "$profile"
    if grep -Eq '^[[:space:]]*alias[[:space:]]+groks=' "$profile"; then
        sed -i '' 's|^[[:space:]]*alias[[:space:]]\+groks=.*$|alias groks="$HOME/.grok/groks.sh"|' "$profile"
    else
        printf '\nalias groks="$HOME/.grok/groks.sh"\n' >> "$profile"
    fi
    # Remove every legacy launcher alias that routes `grok` back through groks.sh.
    sed -i '' '/^[[:space:]]*alias[[:space:]][[:space:]]*grok=.*\.grok\/groks\.sh/d' "$profile"
}

restore_grok_command() {
    local local_bin="$HOME/.local/bin"
    local command_path="$local_bin/grok"
    local real_grok="$GROK_HOME/bin/grok"
    mkdir -p "$local_bin"
    if [[ -e "$command_path" && ! -L "$command_path" ]]; then
        warn "Preserving non-symlink command: $command_path"
        return
    fi
    if [[ -x "$real_grok" ]]; then
        ln -sfn "$real_grok" "$command_path"
        ok "Grok command: $command_path -> $real_grok"
    fi
}

# ── Help ────────────────────────────────────────────────────────────
show_help() {
    # bash 3.2 + UTF-8: never put $VAR immediately before multibyte chars; use ${VAR}.
    cat <<EOF
================================================================
  groks.sh  v${TOOL_VERSION}  — 公司 Grok Build 网关切换（shu）
================================================================
【本文件顶部有更完整的中文说明】用编辑器打开本脚本即可阅读。
本脚本可单独拷走使用，不依赖仓库其它文件。

意图:
  在官方 SpaceXAI 订阅与公司中转 ${COMPANY_BASE_URL} 之间切换。
  配置文件: ~/.grok/config.toml（可用 -GrokHome 覆盖）
  API Key : ${OPENAI_KEY_NAME} / ${DEEPSEEK_KEY_NAME} / ${GROK_KEY_NAME}

依赖:
  bash 3.2+、curl、已安装的 Grok Build
  Grok 安装示例: curl -fsSL https://x.ai/cli/install.sh | bash

推荐流程:
  ./groks.sh env openai       输入默认 GPT 模型分组 Key（隐藏输入）
  ./groks.sh test openai      测网关与 gpt-5.6-sol 权限
  ./groks.sh use company      切公司网关
  然后彻底退出并重启 Grok

常用命令:
  ./groks.sh status
  ./groks.sh use company | use official
  ./groks.sh env [openai|deepseek|grok] [clear]
  ./groks.sh test [openai|deepseek|grok]
  ./groks.sh backup list|restore|clear
  ./groks.sh install          装到 ~/.grok，并修正 grok 命令路径

不 install 也可以:
  bash ./groks.sh status
  bash ./groks.sh use company

company 写入的核心配置:
  [models]
  default = "${COMPANY_MODEL_KEY}"

  [model."${COMPANY_MODEL_KEY}"]
  model = "${COMPANY_MODEL_KEY}"
  base_url = "${COMPANY_BASE_URL}"
  env_key = "${OPENAI_KEY_NAME}"
  api_backend = "responses"

改完没生效:
  1) 彻底退出 Grok 后重开
  2) ./groks.sh status / test
  3) company 应为 default = "${COMPANY_MODEL_KEY}"
  4) official 应为 default = "${OFFICIAL_MODEL}" 且无托管模型目录

故障速查:
  Permission denied     → chmod +x ./groks.sh  或  bash ./groks.sh ...
  无法验证开发者       → xattr -d com.apple.quarantine ./groks.sh
  KEY not set           → ./groks.sh env openai
  HTTP 401/403          → Key 错误/过期，勿用 XAI 官方 key 顶替
  连不上网关            → VPN/网络；curl 测 ${COMPANY_BASE_URL}/models
  配置锁冲突            → rm -rf ~/.grok/.groks.lock 后重试
  误改想回滚            → ./groks.sh backup restore

选项:
  -BackupLimit 3   -TimeoutSec 15   -GrokHome <path>
  -Force   -DryRun   -NoColor

说明:
  - 不需要管理员
  - 配置有锁、校验、备份、原子写入
  - 会话内仍可用 /model ${COMPANY_MODEL_KEY} 临时切模型
EOF
}

# ── Main dispatcher ─────────────────────────────────────────────────
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
                    get_persistent_key "$OPENAI_KEY_NAME" >/dev/null ||
                        die "$OPENAI_KEY_NAME is not set. Run: groks env openai"
                    set_provider "company"
                    ok "Grok is configured with the company model catalog ($COMPANY_MODEL_KEY default)."
                    ;;
                official)
                    set_provider "official"
                    ok "Grok is configured to use official SpaceXAI models ($OFFICIAL_MODEL)."
                    info "Sign in with: grok login   (or set XAI_API_KEY)."
                    ;;
                *) die "Usage: groks use official|company" ;;
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
            die "Unknown command: $COMMAND. Run: groks help"
            ;;
    esac
}

main

# =============================================================================
#  调研与验证记录
# =============================================================================
#
#  记录目的：
#  本脚本可脱离仓库单独分发。本节保留配置依据、实测结果和验证边界，避免后续维护者
#  仅凭模型名称猜测接口类型。记录中不包含 API Key、登录令牌或会话正文。
#
#  一、Grok Build 源码确认（本地源码版本 0.2.114）
#
#  1. 自定义模型由 [model.<id>] 配置；本脚本实际使用的主要字段包括 model、name、
#     base_url、env_key、api_backend、context_window、max_completion_tokens、
#     supports_reasoning_effort、reasoning_effort 和 reasoning_efforts。
#  2. api_backend 决定请求和响应协议，支持 chat_completions、responses、messages。
#     客户端不会根据返回 JSON 自动识别协议，也不会在三种协议之间自动转换。
#  3. chat_completions 请求 /v1/chat/completions，响应使用 created。
#     responses 请求 /v1/responses，响应对象要求 created_at。
#  4. Responses 流式事件由 async-openai 类型直接反序列化。Grok Build 的兼容逻辑
#     只会剔除无法识别的 xAI 工具，不会补 created_at 或 output_text.annotations。
#  5. 因此 HTTP 200 不代表兼容：响应结构与 api_backend 不一致时，仍会出现
#     “serialization error: missing field ...”。
#
#  二、api.shu.cool 网关非流式实测（2026-07-29）
#
#  证据文件：providers/pi/gateway-raw-responses.json
#  探测脚本：providers/pi/probe_gateway.py
#  报告不含密钥；当次 19 个探测均收到 HTTP 2xx，但只验证了 stream=false。
#
#  1. gpt-5.6-sol、gpt-5.6-terra、gpt-5.6-luna、gpt-5.5、gpt-5.4、
#     gpt-5.4-mini、gpt-5.3-codex-spark：
#     /v1/responses 返回 object=response、created_at，并在 output_text 中返回
#     annotations=[]。
#  2. grok-4.5：
#     /v1/responses 返回 object=response、created_at，非流式 Responses 格式完整。
#  3. deepseek-v4-flash-vision-exp、deepseek-v4-pro：
#     /v1/chat/completions 返回 object=chat.completion、created，不是 Responses 格式。
#     所以本脚本将 deepseek-v4-pro 固定为 api_backend="chat_completions"。
#  4. grok-4.3、grok-build、grok-composer-2.5-fast：
#     当次通过 /v1/chat/completions 验证。该结论只覆盖这些精确模型编号，不能自动
#     外推到 grok-4.5-latest、grok-build-latest 或其他带 latest 的别名。
#
#  三、Grok CLI 流式运行观察
#
#  1. DeepSeek 被配置为 responses 时出现 missing field created_at；现有抓包没有保存
#     这次失败的原始 /responses 数据，因此不臆测其完整结构。已确认可用的协议是
#     chat_completions，改用后 deepseek-v4-pro 的实际会话完成。
#  2. GPT 的非流式探测包含 annotations=[]；实际流式会话先收到首 token，随后报
#     missing field annotations 并把该轮标记为失败。结合 Responses 类型结构判断，
#     缺失字段位于后续 SSE 响应对象中，大概率是终止事件；原始失败帧未被日志保存。
#  3. 若修中转层，应在 Responses SSE 的 response.output[].content[] 中为每个
#     type=output_text 的对象保证 annotations=[]；不能靠切换 GPT 后端解决。
#
#  四、provider-mode 与凭证结论
#
#  1. company 模式写入托管模型目录；各分组通过 env_key 读取
#     SHU_API_KEY、SHU_DEEPSEEK_API_KEY、SHU_GROK_API_KEY，配置文件不落明文 Key。
#  2. official 模式移除托管模型目录和中转覆盖，恢复官方 grok-4.5，并继续使用
#     grok login 或 XAI_API_KEY 的官方认证。
#  3. Grok 进程不会热加载这些配置和环境变量，切换后必须彻底退出并重新启动。
#
#  五、本脚本已做的验证
#
#  1. macOS Bash 3.2 空数组兼容问题已处理；Bash 语法检查通过。
#  2. 示例 config.toml.txt 已通过 TOML 解析。
#  3. 使用临时 GROK_HOME 实际生成 company 配置并断言：
#     deepseek-v4-pro=chat_completions，gpt-5.6-sol=responses，
#     grok-4.5=responses。
#  4. Windows 脚本与 Bash 生成逻辑保持一致，但当前没有 Windows PowerShell
#     运行环境，尚未在真实 Windows 主机执行端到端验证。
#
#  维护原则：
#  新增或修改模型前，分别验证 /v1/models、目标推理端点、非流式返回和流式 SSE；
#  不要因为 HTTP 200 或模型名称相似就假设协议兼容。
