# =============================================================================
#  statusline.ps1  —  Claude Code 状态栏（Windows / PowerShell）
# =============================================================================
#
# 【本文件即说明书】供 Claude Code 调用，不要双击运行。
#
# 意图:
#   从 stdin 读 JSON，输出一行：目录 | 模型 | Context% | Out tok/s | 5h | 7d
#
# 依赖:
#   - Windows PowerShell 5.1+ 或 PowerShell 7+
#   - 无需 jq（用 ConvertFrom-Json）
#
# 安装:
#   由同目录 claudes.ps1 install 复制到 %USERPROFILE%\.claude\statusline.ps1
#
# 自测（PowerShell）:
#   '{"model":{"display_name":"t"},"workspace":{"current_dir":"C:\\x"},"context_window":{"context_window_size":100,"current_usage":{"input_tokens":10}}}' | powershell -File .\statusline.ps1
#
# 故障:
#   · 状态栏空白 → settings 路径错误；或执行策略拦截（用 -ExecutionPolicy Bypass）
#   · 不要双击本文件
#
# Claude Code statusline: Dir | Model | Context % | Out tok/s | 5h: % (reset) | 7d: % (reset)
# =============================================================================
$ErrorActionPreference = 'SilentlyContinue'

$raw = [Console]::In.ReadToEnd()
if (-not $raw) { exit 0 }

try {
    $data = $raw | ConvertFrom-Json
} catch {
    exit 0
}

# Context window
$used = [double]($data.context_window.current_usage.input_tokens) +
        [double]($data.context_window.current_usage.cache_creation_input_tokens) +
        [double]($data.context_window.current_usage.cache_read_input_tokens)
$window = [double]($data.context_window.context_window_size)

if ($window -gt 0) {
    $ctxPct = [math]::Round(($used / $window) * 100)
} else {
    $ctxPct = 0
}

# Model name
$modelName = $data.model.display_name

# Directory name
$dirName = if ($data.workspace.current_dir) {
    Split-Path $data.workspace.current_dir -Leaf
} else { '' }

# Output tokens for speed calc
$outputTokens = $data.context_window.current_usage.output_tokens

# Rate limits
$fivePct = [math]::Round($data.rate_limits.five_hour.used_percentage)
$fiveReset = $data.rate_limits.five_hour.resets_at
$sevenPct = [math]::Round($data.rate_limits.seven_day.used_percentage)
$sevenReset = $data.rate_limits.seven_day.resets_at

# Colors
$e = [char]27
$RESET = "${e}[0m"
$DIM = "${e}[2m"
$GREEN = "${e}[32m"
$YELLOW = "${e}[33m"
$RED = "${e}[31m"
$BLUE = "${e}[94m"
$MAGENTA = "${e}[95m"
$CYAN = "${e}[36m"

function Format-Reset($timestamp) {
    if (-not $timestamp) { return '' }
    $epoch = ([DateTimeOffset]$timestamp).ToUnixTimeSeconds()
    $now = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    $diff = $epoch - $now
    if ($diff -le 0) { return '' }
    $mins = [math]::Floor($diff / 60)
    $hours = [math]::Floor($mins / 60)
    $days = [math]::Floor($hours / 24)
    if ($days -ge 1) {
        return "{0}d{1}h" -f $days, ($hours % 24)
    } elseif ($hours -ge 1) {
        return "{0}h{1}m" -f $hours, ($mins % 60)
    } else {
        return "{0}m" -f $mins
    }
}

function Usage-Color($pct) {
    if ($pct -ge 90) { return $RED }
    elseif ($pct -ge 70) { return $MAGENTA }
    else { return $BLUE }
}

$parts = @()

if ($dirName) {
    $parts += "${CYAN}${dirName}${RESET}"
}

$branch = ''
try {
    $wd = $data.workspace.current_dir
    if ($wd) {
        $branch = & git -C $wd rev-parse --abbrev-ref HEAD 2>$null
        if ($branch -eq 'HEAD') {
            $branch = & git -C $wd rev-parse --short HEAD 2>$null
        }
    }
} catch { }
if ($branch) {
    $parts += "${YELLOW}${branch}${RESET}"
}

if ($modelName) {
    $parts += "${DIM}${modelName}${RESET}"
}

if ($ctxPct -ge 85) { $ctxColor = $RED }
elseif ($ctxPct -ge 70) { $ctxColor = $YELLOW }
else { $ctxColor = $GREEN }
$parts += "${DIM}Ctx${RESET} ${ctxColor}${ctxPct}%${RESET}"

$speedCache = Join-Path $env:TEMP '.claude-statusline-speed'
$nowMs = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
if ($outputTokens -gt 0) {
    $prevSpeed = ''
    if (Test-Path $speedCache) {
        $prev = Get-Content $speedCache -Raw
        $parts2 = $prev -split "`t"
        $prevOut = [double]$parts2[0]
        $prevTime = [double]$parts2[1]
        $prevSpeed = $parts2[2]
        $deltaTokens = $outputTokens - $prevOut
        $deltaMs = $nowMs - $prevTime
        if ($deltaTokens -gt 0 -and $deltaMs -gt 0) {
            $prevSpeed = '{0:F1}' -f ($deltaTokens / ($deltaMs / 1000.0))
        }
    }
    if ($prevSpeed) {
        $parts += "${DIM}Out${RESET} ${prevSpeed}${DIM} tok/s${RESET}"
    }
    "$outputTokens`t$nowMs`t$prevSpeed" | Set-Content $speedCache -NoNewline
}

if ($null -ne $fivePct -and $fivePct -gt 0) {
    $color = Usage-Color $fivePct
    $resetStr = Format-Reset $fiveReset
    if ($resetStr) {
        $parts += "${DIM}5h:${RESET} ${color}${fivePct}%${RESET} ${DIM}(${resetStr})${RESET}"
    } else {
        $parts += "${DIM}5h:${RESET} ${color}${fivePct}%${RESET}"
    }
}

if ($null -ne $sevenPct -and $sevenPct -gt 0) {
    $color = Usage-Color $sevenPct
    $resetStr = Format-Reset $sevenReset
    if ($resetStr) {
        $parts += "${DIM}7d:${RESET} ${color}${sevenPct}%${RESET} ${DIM}(${resetStr})${RESET}"
    } else {
        $parts += "${DIM}7d:${RESET} ${color}${sevenPct}%${RESET}"
    }
}

Write-Host ($parts -join ' | ')
