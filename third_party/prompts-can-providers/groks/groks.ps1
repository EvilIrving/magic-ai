#Requires -Version 5.1
<#
================================================================
  groks.ps1  —  公司 Grok Build 网关切换（shu）
================================================================

【本文件即完整说明书】可单独拷走使用，不必回仓库或上网搜文档。
  - 可以用记事本 / VS Code / 右键「编辑」打开本文件查看。
  - 不要双击「运行」本文件：双击多半用记事本打开，或窗口一闪就关，且无法传 install 等参数。
  - 不需要管理员权限。请用「平时登录 Windows、并运行 Grok Build」的同一个用户操作。

意图:
  把 Grok Build 在「官方 SpaceXAI 订阅」与「公司中转 api.shu.cool」之间切换；
  配置与启动模式保存在 config.toml、provider-mode，模型按分组读取环境变量 Key。

----------------------------------------------------------------
一、怎么打开终端（在脚本所在文件夹里，任选一种）
----------------------------------------------------------------
  1) 资源管理器打开本文件所在文件夹
     → 点击上方地址栏 → 输入 powershell → 回车
  2) 或在地址栏输入 cmd → 回车
  3) 或在文件夹空白处 Shift+右键 →「在此处打开 PowerShell / 终端」

  不要：开始菜单乱开一个 PowerShell 却不知道当前目录在哪（除非你会 cd 进来）。

----------------------------------------------------------------
二、首次安装（微信 / 网盘 / 浏览器下载后）——可选
----------------------------------------------------------------
  下载后若文件名变成 groks.ps1.txt，先改回 groks.ps1。

  在「本文件所在文件夹」的 PowerShell 里执行（整行复制）：

    powershell -ExecutionPolicy Bypass -File .\groks.ps1 install

  若提示被阻止，可先解除下载标记再装：

    Unblock-File .\groks.ps1
    powershell -ExecutionPolicy Bypass -File .\groks.ps1 install

  install 会：
    - 复制到  %USERPROFILE%\.grok\groks.ps1
    - 生成     %USERPROFILE%\.grok\groks.cmd 和 grok.cmd
    - 把 .grok 加入当前用户 PATH（并尽量解除下载封锁）
    - 当前窗口一般可直接用 groks；其它已打开的终端请关掉重开

  也可以不 install，始终在仓库/解压目录直接运行本脚本。

----------------------------------------------------------------
三、推荐流程
----------------------------------------------------------------
    groks env openai       输入默认 GPT 模型分组 Key（隐藏输入，推荐）
    groks test openai      测网关和 Key 是否可用
    groks use company      切到公司网关 https://api.shu.cool/v1

  然后：彻底退出 Grok Build，再重新打开。
  只关窗口不够；正在运行的进程不会自动热加载配置。

  切回官方：
    groks use official

----------------------------------------------------------------
四、常用命令
----------------------------------------------------------------
    groks status           看当前 default model / Key / 是否在跑
    groks diagnose         路径、下载封锁、写权限、进程等排查
    groks help             打印本说明（命令行版）
    groks env openai clear 删除本机用户环境变量里的 SHU_API_KEY
    groks env deepseek     配置 DeepSeek 分组 Key
    groks env grok         配置 Grok 分组 Key
    groks backup list|restore|clear

  未 install 时，在脚本目录也可直接：
    powershell -ExecutionPolicy Bypass -File .\groks.ps1 status
    powershell -ExecutionPolicy Bypass -File .\groks.ps1 use company

----------------------------------------------------------------
五、改完「没生效」时先查这几条
----------------------------------------------------------------
  1) Grok 是否还在跑 → 必须全退再开
  2) groks status 里 Provider mode 是否已是 company、API key 是否 set
  3) groks test 是否通过（网络 / Key）
  4) groks diagnose 看 Mark of the Web、config 路径、进程列表
  5) 记事本打开：  %USERPROFILE%\.grok\config.toml
     应有 [models] default = "gpt-5.6-sol" 和完整托管模型目录

----------------------------------------------------------------
六、安全说明（简要）
----------------------------------------------------------------
  - 从微信/网盘下载的文件可能带「Mark of the Web」；install 会尝试自动解除。
  - -ExecutionPolicy Bypass 只绕过 PowerShell 执行策略，不是关闭杀毒/AMSI。
  - SHU_API_KEY、SHU_DEEPSEEK_API_KEY、SHU_GROK_API_KEY 分组保存到当前用户环境变量。
  - 三个分组的 Key 独立使用，不写入 config.toml。
  - 不要用管理员身份乱跑，以免写到错误用户目录。

----------------------------------------------------------------
七、意图（一句话）
----------------------------------------------------------------
  本脚本只改本机 Grok 配置与三个分组环境变量，在「官方模型」和「公司中转」
  之间切换。可单独拷走使用，不依赖本仓库其它文件。

----------------------------------------------------------------
八、常见故障（拷走后也能对照）
----------------------------------------------------------------
  · 双击没反应 / 窗口一闪
      → 不要双击。在脚本目录开 PowerShell，用 -File 运行。
  · 无法加载，因为在此系统上禁止运行脚本
      → powershell -ExecutionPolicy Bypass -File .\groks.ps1 help
      → 或: Unblock-File .\groks.ps1
  · 文件名变成 groks.ps1.txt
      → 改回 .ps1 扩展名
  · SHU_API_KEY is not set
      → .\groks.ps1 env openai；新开终端再试
  · HTTP 401/403
      → Key 错误或过期；不要用官方 XAI_API_KEY 顶替公司 key
  · 连不上网关
      → 公司 VPN / 网络；浏览器或 curl 测 https://api.shu.cool/v1/models
  · 改完 Grok 还是官方
      → 全退 Grok 再开；groks status 看 default 是否 gpt-5.6-sol；打开
        %USERPROFILE%\.grok\config.toml 人工确认
  · 没有 Grok / 命令找不到 grok
      → 先装 Grok Build: irm https://x.ai/cli/install.ps1 | iex
      → 本脚本不负责安装 Grok，只负责切中转配置
  · 想回滚 config
      → groks backup list / restore

.SYNOPSIS
  Switch Grok Build between official SpaceXAI and company shu gateway.

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File .\groks.ps1 install
  groks env openai
  groks test openai
  groks use company
#>

[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet('env', 'use', 'test', 'backup', 'status', 'help', 'install', 'diagnose')]
    [string]$Command = 'status',

    [Parameter(Position = 1)]
    [AllowEmptyString()]
    [string]$Value,

    [Parameter(Position = 2)]
    [AllowEmptyString()]
    [string]$ExtraValue,

    [string]$GrokHome = $(
        if ([string]::IsNullOrWhiteSpace($env:GROK_HOME)) {
            Join-Path $env:USERPROFILE '.grok'
        }
        else {
            $env:GROK_HOME
        }
    ),

    [ValidateRange(1, 100)]
    [int]$BackupLimit = 3,

    [ValidateRange(3, 60)]
    [int]$TimeoutSec = 15,

    [switch]$Force
)

$ErrorActionPreference = 'Stop'
$script:ConfigPath = Join-Path $GrokHome 'config.toml'
$script:ModePath = Join-Path $GrokHome 'provider-mode'
$script:BackupDir = Join-Path $GrokHome 'provider-switch-backups'
$script:LockPath = Join-Path $GrokHome '.groks.lock'
$script:ModelBlockStart = '# groks-managed-catalog-start'
$script:ModelBlockEnd = '# groks-managed-catalog-end'
$script:OpenAIKeyName = 'SHU_API_KEY'
$script:DeepSeekKeyName = 'SHU_DEEPSEEK_API_KEY'
$script:KeyName = 'SHU_GROK_API_KEY'
$script:CompanyModelKey = 'gpt-5.6-sol'
$script:OfficialModel = 'grok-4.5'
$script:DefaultReasoningEffort = 'high'
$script:CompanyBaseUrl = 'https://api.shu.cool/v1'
$script:ToolVersion = '1.4.0'
$script:Utf8NoBom = New-Object System.Text.UTF8Encoding($false)
$script:CompanyModelIds = @(
    'gpt-5.6-sol',
    'gpt-5.6-terra',
    'gpt-5.6-luna',
    'gpt-5.5',
    'gpt-5.4',
    'deepseek-v4-pro',
    'deepseek-v4-flash-vision-exp',
    'grok-4.6',
    'grok-4.5',
    'grok-4.5-latest',
    'grok-build-latest',
    'grok-composer-2.5-fast',
    'shu'
)

function Write-Ok([string]$Message) { Write-Host $Message -ForegroundColor Green }
function Write-Info([string]$Message) { Write-Host $Message -ForegroundColor Cyan }
function Write-Warn([string]$Message) { Write-Host $Message -ForegroundColor Yellow }

function Get-ScriptSourcePath {
    if (-not [string]::IsNullOrWhiteSpace($PSCommandPath)) { return $PSCommandPath }
    if (-not [string]::IsNullOrWhiteSpace($MyInvocation.PSCommandPath)) { return $MyInvocation.PSCommandPath }
    return $null
}

function Test-MarkOfTheWeb([string]$Path) {
    if ([string]::IsNullOrWhiteSpace($Path) -or -not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        return $false
    }
    return [bool](Get-Item -LiteralPath $Path -Stream Zone.Identifier -ErrorAction SilentlyContinue)
}

function Clear-MarkOfTheWeb([string]$Path) {
    if ([string]::IsNullOrWhiteSpace($Path) -or -not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        return $false
    }
    try {
        if (Test-MarkOfTheWeb $Path) {
            Unblock-File -LiteralPath $Path -ErrorAction Stop
            return $true
        }
    }
    catch {
        Write-Warn "Could not unblock file (Mark of the Web): $Path"
        Write-Warn "  $($_.Exception.Message)"
    }
    return $false
}

function Test-PathListContains([string]$PathList, [string]$TargetDir) {
    if ([string]::IsNullOrWhiteSpace($PathList) -or [string]::IsNullOrWhiteSpace($TargetDir)) {
        return $false
    }
    $target = [System.IO.Path]::GetFullPath($TargetDir).TrimEnd('\')
    foreach ($part in @($PathList -split ';')) {
        if ([string]::IsNullOrWhiteSpace($part)) { continue }
        try {
            if ([System.IO.Path]::GetFullPath($part.Trim()).TrimEnd('\') -eq $target) {
                return $true
            }
        }
        catch { }
    }
    return $false
}

function Send-EnvironmentChange {
    try {
        if (-not ('Groks.NativeMethods' -as [type])) {
            Add-Type -Namespace Groks -Name NativeMethods -MemberDefinition @'
[System.Runtime.InteropServices.DllImport("user32.dll", SetLastError = true, CharSet = System.Runtime.InteropServices.CharSet.Auto)]
public static extern IntPtr SendMessageTimeout(
    IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam,
    uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
'@
        }
        $result = [UIntPtr]::Zero
        [void][Groks.NativeMethods]::SendMessageTimeout(
            [IntPtr]0xffff, [uint32]0x1a, [UIntPtr]::Zero, 'Environment',
            [uint32]2, [uint32]5000, [ref]$result
        )
    }
    catch { }
}

function Ensure-GrokHome {
    if ([string]::IsNullOrWhiteSpace($GrokHome)) { throw 'Grok home path is empty.' }
    if (-not (Test-Path -LiteralPath $GrokHome)) {
        New-Item -ItemType Directory -Path $GrokHome | Out-Null
    }
    if (-not (Test-Path -LiteralPath $GrokHome -PathType Container)) {
        throw "Grok home is not a directory: $GrokHome"
    }
}

function Invoke-WithConfigLock([scriptblock]$Action) {
    Ensure-GrokHome
    $lock = $null
    $deadline = [DateTime]::UtcNow.AddSeconds(10)
    while ($null -eq $lock -and [DateTime]::UtcNow -lt $deadline) {
        try {
            $lock = [System.IO.File]::Open(
                $script:LockPath,
                [System.IO.FileMode]::OpenOrCreate,
                [System.IO.FileAccess]::ReadWrite,
                [System.IO.FileShare]::None
            )
        }
        catch [System.IO.IOException] {
            Start-Sleep -Milliseconds 200
        }
    }
    if ($null -eq $lock) { throw 'Another groks process is modifying the configuration. Try again later.' }
    try { & $Action }
    finally { $lock.Dispose() }
}

function Get-ConfigText {
    Ensure-GrokHome
    if (-not (Test-Path -LiteralPath $script:ConfigPath)) { return '' }
    return [System.IO.File]::ReadAllText($script:ConfigPath)
}

function Get-BackupFiles {
    if (-not (Test-Path -LiteralPath $script:BackupDir -PathType Container)) { return @() }
    return @(Get-ChildItem -LiteralPath $script:BackupDir -File -Filter 'config.*.toml' |
        Sort-Object @{ Expression = 'LastWriteTimeUtc'; Descending = $true },
                    @{ Expression = 'Name'; Descending = $true })
}

function Remove-BackupFile([System.IO.FileInfo]$File) {
    $backupRoot = [System.IO.Path]::GetFullPath($script:BackupDir).TrimEnd('\') + '\'
    $filePath = [System.IO.Path]::GetFullPath($File.FullName)
    if (-not $filePath.StartsWith($backupRoot, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to delete a file outside the backup directory: $filePath"
    }
    Remove-Item -LiteralPath $filePath -Force
}

function Remove-ExcessBackups {
    $expired = @(Get-BackupFiles | Select-Object -Skip $BackupLimit)
    foreach ($file in $expired) { Remove-BackupFile $file }
    if ($expired.Count -gt 0) {
        Write-Info "Removed $($expired.Count) old backup(s); keeping the latest $BackupLimit."
    }
}

function New-ConfigBackup {
    if (-not (Test-Path -LiteralPath $script:ConfigPath -PathType Leaf)) { return $null }
    if (-not (Test-Path -LiteralPath $script:BackupDir)) {
        New-Item -ItemType Directory -Path $script:BackupDir | Out-Null
    }
    $stamp = Get-Date -Format 'yyyyMMdd-HHmmss-fff'
    $suffix = [Guid]::NewGuid().ToString('N').Substring(0, 8)
    $backup = Join-Path $script:BackupDir "config.$stamp-$suffix.toml"
    Copy-Item -LiteralPath $script:ConfigPath -Destination $backup
    return $backup
}

function Get-TomlLines([string]$Text) {
    if ([string]::IsNullOrEmpty($Text)) { return @() }
    $lines = @([regex]::Split($Text, '\r?\n'))
    while ($lines.Count -gt 0 -and $lines[$lines.Count - 1] -eq '') {
        if ($lines.Count -eq 1) { return @() }
        $lines = @($lines[0..($lines.Count - 2)])
    }
    return $lines
}

# Read [models].default (Grok's active model id for new sessions)
function Get-CurrentDefault([string]$Text) {
    $lines = @(Get-TomlLines $Text)
    $inModels = $false
    foreach ($line in $lines) {
        if ($line -match '^\s*\[models\]\s*(?:#.*)?$') {
            $inModels = $true
            continue
        }
        if ($inModels -and $line -match '^\s*\[') { break }
        if ($inModels -and $line -match '^\s*default\s*=') {
            if ($line -match '^\s*default\s*=\s*"([^"]+)"\s*(?:#.*)?$') {
                return $Matches[1]
            }
            if ($line -match "^\s*default\s*=\s*'([^']+)'\s*(?:#.*)?$") {
                return $Matches[1]
            }
            throw 'The [models].default value in config.toml is not a valid quoted string.'
        }
    }
    return $script:OfficialModel
}

function Assert-ConfigStructure([string]$Text) {
    if ($Text.IndexOf([char]0) -ge 0) { throw 'config.toml contains a NUL character.' }
    [void](Get-CurrentDefault $Text)
    foreach ($modelId in $script:CompanyModelIds) {
        $escaped = [regex]::Escape($modelId)
        $modelHeaders = @([regex]::Matches(
            $Text,
            "(?m)^\s*\[model\.(?:`"$escaped`"|$escaped)\]\s*(?:#.*)?$"
        ))
        if ($modelHeaders.Count -gt 1) {
            throw "config.toml contains duplicate [model.$modelId] sections."
        }
    }
    $modelsHeaders = @([regex]::Matches($Text, '(?m)^\s*\[models\]\s*(?:#.*)?$'))
    if ($modelsHeaders.Count -gt 1) {
        throw 'config.toml contains duplicate [models] sections.'
    }
    $starts = @([regex]::Matches($Text, ('(?m)^' + [regex]::Escape($script:ModelBlockStart) + '$')))
    $ends = @([regex]::Matches($Text, ('(?m)^' + [regex]::Escape($script:ModelBlockEnd) + '$')))
    if ($starts.Count -ne $ends.Count -or $starts.Count -gt 1) {
        throw 'config.toml contains an incomplete groks-managed model block.'
    }
}

function Set-ModelsStringSetting(
    [string[]]$Lines,
    [string]$SettingName,
    [string]$SettingValue
) {
    $sectionStarts = @()
    for ($i = 0; $i -lt $Lines.Count; $i++) {
        if ($Lines[$i] -match '^\s*\[models\]\s*(?:#.*)?$') {
            $sectionStarts += $i
        }
    }
    if ($sectionStarts.Count -gt 1) {
        throw 'Duplicate [models] sections are not supported.'
    }

    if ($sectionStarts.Count -eq 0) {
        $out = @($Lines)
        if ($out.Count -gt 0 -and $out[$out.Count - 1] -ne '') { $out += '' }
        $out += '[models]'
        $out += "$SettingName = `"$SettingValue`""
        return $out
    }

    $start = $sectionStarts[0]
    $end = $Lines.Count
    for ($i = $start + 1; $i -lt $Lines.Count; $i++) {
        if ($Lines[$i] -match '^\s*\[') { $end = $i; break }
    }

    $settingLine = "$SettingName = `"$SettingValue`""
    $settingPattern = '^\s*' + [Regex]::Escape($SettingName) + '\s*='
    $found = $false
    $sectionBody = @()
    if ($end -gt ($start + 1)) {
        foreach ($line in @($Lines[($start + 1)..($end - 1)])) {
            if ($line -match $settingPattern) {
                if (-not $found) {
                    $sectionBody += $settingLine
                    $found = $true
                }
            }
            else {
                $sectionBody += $line
            }
        }
    }
    if (-not $found) { $sectionBody += $settingLine }

    $prefix = @($Lines[0..$start])
    $suffix = @()
    if ($end -lt $Lines.Count) { $suffix = @($Lines[$end..($Lines.Count - 1)]) }
    return ($prefix + $sectionBody + $suffix)
}

function Set-DefaultModel([string[]]$Lines, [string]$DefaultValue) {
    $result = @(Set-ModelsStringSetting `
        -Lines $Lines `
        -SettingName 'default' `
        -SettingValue $DefaultValue)
    return $result
}

function Set-DefaultReasoningEffort([string[]]$Lines) {
    $result = @(Set-ModelsStringSetting `
        -Lines $Lines `
        -SettingName 'default_reasoning_effort' `
        -SettingValue $script:DefaultReasoningEffort)
    return $result
}

function Remove-ManagedModelBlock([string[]]$Lines) {
    $start = -1
    $end = -1
    for ($i = 0; $i -lt $Lines.Count; $i++) {
        if ($Lines[$i] -eq $script:ModelBlockStart) { $start = $i }
        if ($Lines[$i] -eq $script:ModelBlockEnd) { $end = $i }
    }
    if ($start -lt 0 -and $end -lt 0) { return @($Lines) }
    if ($start -lt 0 -or $end -le $start) { throw 'Invalid groks-managed model block.' }

    $out = @()
    if ($start -gt 0) { $out += @($Lines[0..($start - 1)]) }
    if ($end -lt ($Lines.Count - 1)) { $out += @($Lines[($end + 1)..($Lines.Count - 1)]) }
    while ($out.Count -gt 0 -and $out[$out.Count - 1] -eq '') {
        if ($out.Count -eq 1) { return @() }
        $out = @($out[0..($out.Count - 2)])
    }
    while ($out.Count -gt 0 -and $out[0] -eq '') {
        if ($out.Count -eq 1) { return @() }
        $out = @($out[1..($out.Count - 1)])
    }
    return $out
}

function Test-CompanyModelHeader([string]$Line) {
    $compact = ($Line -replace '\s', '') -replace '"', ''
    foreach ($modelId in $script:CompanyModelIds) {
        if ($compact -eq "[model.$modelId]") { return $true }
        if ($compact -eq ("[[model." + $modelId + ".reasoning_efforts]]")) { return $true }
    }
    return $false
}

function Remove-CompanyModelSections([string[]]$Lines) {
    $out = @()
    $skipping = $false
    foreach ($line in $Lines) {
        if ($line -match '^\s*\[') {
            if (Test-CompanyModelHeader $line) {
                $skipping = $true
                continue
            }
            $skipping = $false
        }
        if (-not $skipping) { $out += $line }
    }
    while ($out.Count -gt 0 -and $out[$out.Count - 1] -eq '') {
        if ($out.Count -eq 1) { return @() }
        $out = @($out[0..($out.Count - 2)])
    }
    return $out
}

function Add-ModelBase([string[]]$Lines, [string]$ModelId, [string]$ModelName, [string]$KeyName, [string]$Backend, [int]$ContextWindow, [int]$MaxCompletionTokens) {
    $out = @($Lines)
    $out += @(
        '[model."' + $ModelId + '"]'
        'model = "' + $ModelId + '"'
        'name = "' + $ModelName + '"'
        "base_url = `"$($script:CompanyBaseUrl)`""
        "env_key = `"$KeyName`""
        'api_backend = "' + $Backend + '"'
        "context_window = $ContextWindow"
        "max_completion_tokens = $MaxCompletionTokens"
        'supports_reasoning_effort = true'
        "reasoning_effort = `"$($script:DefaultReasoningEffort)`""
    )
    return $out
}

function Add-ReasoningEffortTable([string[]]$Lines, [string]$ModelId, [string[]]$Values) {
    $out = @($Lines)
    foreach ($value in $Values) {
        switch ($value) {
            'none' { $label = 'None' }
            'low' { $label = 'Low Effort' }
            'medium' { $label = 'Medium Effort' }
            'high' { $label = 'High Effort' }
            'xhigh' { $label = 'Extra High Effort' }
            'max' { $label = 'Max Effort' }
            default { throw "Unsupported reasoning effort value: $value" }
        }
        $defaultText = if ($value -eq $script:DefaultReasoningEffort) { 'true' } else { 'false' }
        $out += @(
            '[[model."' + $ModelId + '".reasoning_efforts]]'
            'value = "' + $value + '"'
            'label = "' + $label + '"'
            "default = $defaultText"
        )
        $out += ''
    }
    return $out
}

function Add-ManagedCompanyModelBlock([string[]]$Lines) {
    $out = @($Lines)
    if ($out.Count -gt 0) { $out += '' }
    $out += $script:ModelBlockStart

    $out = @(Add-ModelBase $out 'gpt-5.6-sol' 'GPT-5.6 Sol' $script:OpenAIKeyName 'responses' 1050000 128000)
    $out += ''
    $out = @(Add-ReasoningEffortTable $out 'gpt-5.6-sol' @('none', 'low', 'medium', 'high', 'xhigh', 'max'))

    $out = @(Add-ModelBase $out 'gpt-5.6-terra' 'GPT-5.6 Terra' $script:OpenAIKeyName 'responses' 1050000 128000)
    $out += ''
    $out = @(Add-ReasoningEffortTable $out 'gpt-5.6-terra' @('none', 'low', 'medium', 'high', 'xhigh', 'max'))

    $out = @(Add-ModelBase $out 'gpt-5.6-luna' 'GPT-5.6 Luna' $script:OpenAIKeyName 'responses' 1050000 128000)
    $out += ''
    $out = @(Add-ReasoningEffortTable $out 'gpt-5.6-luna' @('none', 'low', 'medium', 'high', 'xhigh', 'max'))

    $out = @(Add-ModelBase $out 'gpt-5.5' 'GPT-5.5' $script:OpenAIKeyName 'responses' 1050000 128000)
    $out += ''
    $out = @(Add-ReasoningEffortTable $out 'gpt-5.5' @('none', 'low', 'medium', 'high', 'xhigh'))

    $out = @(Add-ModelBase $out 'deepseek-v4-pro' 'DeepSeek V4 Pro' $script:DeepSeekKeyName 'chat_completions' 1000000 384000)
    $out += ''
    $out = @(Add-ReasoningEffortTable $out 'deepseek-v4-pro' @('low', 'medium', 'high', 'xhigh', 'max'))

    $out = @(Add-ModelBase $out 'deepseek-v4-flash-vision-exp' 'DeepSeek V4 Flash Vision Exp' $script:DeepSeekKeyName 'responses' 1000000 384000)
    $out += ''
    $out = @(Add-ReasoningEffortTable $out 'deepseek-v4-flash-vision-exp' @('low', 'medium', 'high', 'xhigh', 'max'))

    $out = @(Add-ModelBase $out 'grok-4.6' 'Grok 4.6' $script:KeyName 'responses' 500000 128000)
    $out += ''
    $out = @(Add-ReasoningEffortTable $out 'grok-4.6' @('low', 'medium', 'high', 'xhigh'))

    $out = @(Add-ModelBase $out 'grok-4.5' 'Grok 4.5' $script:KeyName 'responses' 500000 128000)
    $out += ''
    $out = @(Add-ReasoningEffortTable $out 'grok-4.5' @('low', 'medium', 'high', 'xhigh'))

    $out += $script:ModelBlockEnd
    return $out
}

function Set-ManagedConfig([string]$Text, [string]$Mode) {
    Assert-ConfigStructure $Text
    $lines = @(Get-TomlLines $Text)
    $lines = @(Remove-ManagedModelBlock $lines)
    $lines = @(Remove-CompanyModelSections $lines)
    if ($Mode -eq 'company') {
        $lines = @(Add-ManagedCompanyModelBlock $lines)
        $defaultModel = $script:CompanyModelKey
    }
    else {
        $defaultModel = $script:OfficialModel
    }
    $lines = @(Set-DefaultModel -Lines $lines -DefaultValue $defaultModel)
    $lines = @(Set-DefaultReasoningEffort -Lines $lines)

    $result = ($lines -join [Environment]::NewLine).TrimEnd() + [Environment]::NewLine
    Assert-ConfigStructure $result
    $verified = Get-CurrentDefault $result
    if ($verified -ne $defaultModel) {
        throw "Default model verification failed before writing config.toml (got: $verified)."
    }
    return $result
}

function Write-ConfigAtomic([string]$Text) {
    Assert-ConfigStructure $Text
    Ensure-GrokHome
    $temp = Join-Path $GrokHome ('.config.toml.' + [Guid]::NewGuid().ToString('N') + '.tmp')
    $replaceBackup = Join-Path $GrokHome ('.config.toml.' + [Guid]::NewGuid().ToString('N') + '.rollback')
    try {
        [System.IO.File]::WriteAllText($temp, $Text, $script:Utf8NoBom)
        $verify = [System.IO.File]::ReadAllText($temp)
        Assert-ConfigStructure $verify
        if ($verify -ne $Text) { throw 'Temporary configuration verification failed.' }

        $backup = New-ConfigBackup
        if (Test-Path -LiteralPath $script:ConfigPath) {
            [System.IO.File]::Replace($temp, $script:ConfigPath, $replaceBackup)
        }
        else {
            [System.IO.File]::Move($temp, $script:ConfigPath)
        }
        Remove-ExcessBackups
        return $backup
    }
    catch {
        throw "Configuration was not changed: $($_.Exception.Message)"
    }
    finally {
        if (Test-Path -LiteralPath $temp) { Remove-Item -LiteralPath $temp -Force }
        if (Test-Path -LiteralPath $replaceBackup) { Remove-Item -LiteralPath $replaceBackup -Force }
    }
}

function Set-ProviderConfig([string]$Mode) {
    Invoke-WithConfigLock {
        $current = Get-ConfigText
        $updated = Set-ManagedConfig $current $Mode
        if ($updated -eq $current) {
            Write-Info 'Configuration is already up to date; no backup was created.'
            return
        }
        $backup = Write-ConfigAtomic $updated
        if ($backup) { Write-Info "Backup: $backup" }
    }
}

function Get-ProviderMode {
    $text = Get-ConfigText
    $default = Get-CurrentDefault $text
    if (
        $default -eq $script:CompanyModelKey -and
        $text -match ('(?m)^' + [regex]::Escape($script:ModelBlockStart) + '$')
    ) {
        return 'company'
    }
    return 'official'
}

function Write-ProviderMode([string]$Mode) {
    if ($Mode -notin @('company', 'official')) { throw "Invalid provider mode: $Mode" }
    Ensure-GrokHome
    $temp = Join-Path $GrokHome ('.provider-mode.' + [Guid]::NewGuid().ToString('N') + '.tmp')
    try {
        [System.IO.File]::WriteAllText($temp, $Mode + [Environment]::NewLine, $script:Utf8NoBom)
        Move-Item -LiteralPath $temp -Destination $script:ModePath -Force
    }
    finally {
        if (Test-Path -LiteralPath $temp) { Remove-Item -LiteralPath $temp -Force }
    }
}

function Get-KeyGroup([string]$Group) {
    switch (([string]$Group).Trim().ToLowerInvariant()) {
        { $_ -in @('', 'openai') } {
            return [pscustomobject]@{
                Name = 'OpenAI'
                KeyName = $script:OpenAIKeyName
                ModelId = $script:CompanyModelKey
            }
        }
        'deepseek' {
            return [pscustomobject]@{
                Name = 'DeepSeek'
                KeyName = $script:DeepSeekKeyName
                ModelId = 'deepseek-v4-pro'
            }
        }
        'grok' {
            return [pscustomobject]@{
                Name = 'Grok'
                KeyName = $script:KeyName
                ModelId = 'grok-4.5'
            }
        }
        default { throw "Unknown key group: $Group. Use openai, deepseek, or grok." }
    }
}

function Get-CompanyKey([string]$KeyName) {
    $key = [Environment]::GetEnvironmentVariable($KeyName, 'Process')
    if ([string]::IsNullOrWhiteSpace($key)) {
        $key = [Environment]::GetEnvironmentVariable($KeyName, 'User')
    }
    return $key
}

function ConvertFrom-SecureValue([Security.SecureString]$SecureValue) {
    $pointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($SecureValue)
    try { return [Runtime.InteropServices.Marshal]::PtrToStringBSTR($pointer) }
    finally { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($pointer) }
}

function Get-RunningGrokProcesses {
    return @(Get-Process -ErrorAction SilentlyContinue | Where-Object {
        $_.ProcessName -in @('grok', 'Grok', 'Grok Build', 'grok-build')
    })
}

function Show-RestartWarning {
    param([string]$Context = 'config')

    Write-Host ''
    Write-Warn "On-disk $Context was updated. Grok Build does NOT reload it while still running."
    Write-Warn 'Fully quit Grok Build, then start again.'
    Write-Warn 'Closing only the window is often not enough.'

    $running = Get-RunningGrokProcesses
    if ($running.Count -gt 0) {
        $names = ($running | ForEach-Object { "$($_.ProcessName)(PID $($_.Id))" }) -join ', '
        Write-Warn "Still running now: $names"
        Write-Warn 'Until those processes exit, the app keeps the OLD model / API key.'
    }
    else {
        Write-Info 'No Grok process detected. You can start Grok now.'
    }
    Write-Host ''
}

function Set-CompanyKey {
    $groupName = ([string]$Value).ToLowerInvariant()
    $action = [string]$ExtraValue
    if ($groupName -eq 'clear') {
        $groupName = 'openai'
        $action = 'clear'
    }
    $group = Get-KeyGroup $groupName
    if ($action.ToLowerInvariant() -eq 'clear') {
        [Environment]::SetEnvironmentVariable($group.KeyName, $null, 'User')
        [Environment]::SetEnvironmentVariable($group.KeyName, $null, 'Process')
        Send-EnvironmentChange
        Write-Ok "$($group.KeyName) was removed for the current user and process."
        Show-RestartWarning -Context 'API key'
        return
    }

    $key = $action
    if ([string]::IsNullOrWhiteSpace($key)) {
        $secureKey = Read-Host "Enter SHU $($group.Name) API Key (input is hidden)" -AsSecureString
        $key = ConvertFrom-SecureValue $secureKey
    }
    else {
        Write-Warn 'The API key may now be present in PowerShell command history. Prefer: groks env'
    }
    # Clipboard paste often appends CR (Windows/docs); strip before validation.
    $key = ($key -replace "`r", '').Trim()
    if ([string]::IsNullOrWhiteSpace($key)) { throw 'API Key cannot be empty.' }
    if ($key -match '[\x00-\x1F\x7F]') {
        throw 'API Key contains an invalid control character (often from paste). Re-enter a single-line key.'
    }
    [Environment]::SetEnvironmentVariable($group.KeyName, $key, 'User')
    [Environment]::SetEnvironmentVariable($group.KeyName, $key, 'Process')
    Send-EnvironmentChange
    Write-Ok "$($group.KeyName) is set for the current Windows user and this process."
    Write-Info "This key is dedicated to the $($group.Name) model group."
    Write-Warn 'The user environment variable is readable by programs running as this Windows user.'
    Show-RestartWarning -Context 'API key'
}

function Show-Status {
    $text = Get-ConfigText
    $default = Get-CurrentDefault $text
    $running = Get-RunningGrokProcesses
    $modeLabel = Get-ProviderMode

    Write-Host ''
    Write-Host '===== Grok provider status ====='
    Write-Host "Tool version     : $($script:ToolVersion)"
    Write-Host "Config           : $($script:ConfigPath)"
    Write-Host "Provider mode    : $modeLabel"
    Write-Host "Default model    : $default"
    foreach ($groupName in @('openai', 'deepseek', 'grok')) {
        $group = Get-KeyGroup $groupName
        $isSet = -not [string]::IsNullOrWhiteSpace((Get-CompanyKey $group.KeyName))
        Write-Host ("{0,-17}: {1}" -f "$($group.Name) key", $(if ($isSet) { 'set' } else { 'not set' }))
    }
    Write-Host "Gateway          : $($script:CompanyBaseUrl)"
    Write-Host "Backups          : $((Get-BackupFiles).Count) / $BackupLimit retained"
    Write-Host "Running app      : $(if ($running.Count -gt 0) { 'yes' } else { 'no' })"
    Write-Host ''

    if (
        $modeLabel -eq 'company' -and
        [string]::IsNullOrWhiteSpace((Get-CompanyKey $script:OpenAIKeyName))
    ) {
        Write-Warn "Company model is selected, but $($script:OpenAIKeyName) is missing."
    }
    if ($running.Count -gt 0 -and $modeLabel -eq 'company') {
        Write-Warn 'Grok is still running — quit fully and restart or changes may not appear.'
    }
}

function Test-CompanyConnection {
    $group = Get-KeyGroup $Value
    $key = Get-CompanyKey $group.KeyName
    if ([string]::IsNullOrWhiteSpace($key)) {
        throw "$($group.KeyName) is not set. Run: groks env $(([string]$Value).ToLowerInvariant())"
    }
    $uri = $script:CompanyBaseUrl.TrimEnd('/') + '/models'
    Write-Info "Testing company gateway: $uri"
    try {
        $response = Invoke-WebRequest -Uri $uri -Method Get -Headers @{
            Authorization = "Bearer $key"
            Accept = 'application/json'
        } -TimeoutSec $TimeoutSec -UseBasicParsing
        if ($response.Content -notmatch ('"id"\s*:\s*"' + [regex]::Escape($group.ModelId) + '"')) {
            throw "Company gateway accepted the API key, but this key cannot access $($group.ModelId)."
        }
        Write-Ok "Company gateway accepted the $($group.Name) key and exposes $($group.ModelId) (HTTP $($response.StatusCode))."
    }
    catch {
        $statusCode = $null
        if ($_.Exception.Response -and $_.Exception.Response.StatusCode) {
            $statusCode = [int]$_.Exception.Response.StatusCode
        }
        if ($statusCode -eq 401 -or $statusCode -eq 403) {
            throw "Company gateway rejected the API key (HTTP $statusCode)."
        }
        if ($statusCode) { throw "Company gateway test failed (HTTP $statusCode)." }
        throw "Company gateway test failed: $($_.Exception.Message)"
    }
}

function Ensure-GrokHomeOnUserPath {
    Ensure-GrokHome
    $homeFull = [System.IO.Path]::GetFullPath($GrokHome).TrimEnd('\')
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($null -eq $userPath) { $userPath = '' }

    $userParts = @($userPath -split ';' | Where-Object {
        if ([string]::IsNullOrWhiteSpace($_)) { return $false }
        try {
            [System.IO.Path]::GetFullPath($_.Trim()).TrimEnd('\') -ne $homeFull
        }
        catch { $true }
    })
    $newPath = (@($homeFull) + $userParts) -join ';'
    if ($newPath -ne $userPath) {
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        Send-EnvironmentChange
        Write-Ok "Moved to the front of user PATH: $homeFull"
    }
    else {
        Write-Info "User PATH already starts with: $homeFull"
    }

    $processParts = @($env:Path -split ';' | Where-Object {
        if ([string]::IsNullOrWhiteSpace($_)) { return $false }
        try {
            [System.IO.Path]::GetFullPath($_.Trim()).TrimEnd('\') -ne $homeFull
        }
        catch { $true }
    })
    $env:Path = (@($homeFull) + $processParts) -join ';'
}

function New-GroksWrapper([string]$WrapperPath) {
    $content = @"
@echo off
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0groks.ps1" %*
"@
    [System.IO.File]::WriteAllText($WrapperPath, $content, [System.Text.Encoding]::ASCII)
}

function New-GrokWrapper([string]$WrapperPath) {
    $content = @"
@echo off
"%~dp0bin\grok.exe" %*
exit /b %ERRORLEVEL%
"@
    [System.IO.File]::WriteAllText($WrapperPath, $content, [System.Text.Encoding]::ASCII)
}

function Do-Install {
    $sourcePath = Get-ScriptSourcePath
    if ([string]::IsNullOrWhiteSpace($sourcePath)) {
        throw 'Cannot determine my own path. Run the script directly from a file.'
    }

    if (Clear-MarkOfTheWeb $sourcePath) {
        Write-Ok "Removed download block (Mark of the Web) from: $sourcePath"
    }

    $leaf = [System.IO.Path]::GetFileName($sourcePath)
    if ($leaf -match '\.ps1\.(txt|download|tmp)$' -or ($leaf -match '\.txt$' -and $leaf -notmatch '\.ps1$')) {
        Write-Warn "This file looks renamed by a chat/download app: $leaf"
        Write-Warn 'Rename it to groks.ps1, then run install again.'
    }

    $targetPath = Join-Path $GrokHome 'groks.ps1'
    $groksWrapperPath = Join-Path $GrokHome 'groks.cmd'
    $grokWrapperPath = Join-Path $GrokHome 'grok.cmd'

    if (Test-Path -LiteralPath $targetPath) {
        $targetResolved = Resolve-Path -LiteralPath $targetPath -ErrorAction SilentlyContinue
        $sourceResolved = Resolve-Path -LiteralPath $sourcePath -ErrorAction SilentlyContinue
        if ($targetResolved -and $sourceResolved -and $targetResolved.Path -eq $sourceResolved.Path) {
            Write-Info "Already installed at $targetPath."
            New-GroksWrapper $groksWrapperPath
            New-GrokWrapper $grokWrapperPath
            if (Clear-MarkOfTheWeb $targetPath) {
                Write-Ok "Removed download block from installed script: $targetPath"
            }
            Ensure-GrokHomeOnUserPath
            Write-Warn 'Already open terminals may need to be reopened before groks is found.'
            Write-Info 'Next: groks env openai  ->  groks test openai  ->  groks use company  -> quit Grok fully'
            return
        }
    }

    Ensure-GrokHome
    Copy-Item -LiteralPath $sourcePath -Destination $targetPath -Force
    if (Clear-MarkOfTheWeb $targetPath) {
        Write-Ok "Removed download block from installed script: $targetPath"
    }

    New-GroksWrapper $groksWrapperPath
    New-GrokWrapper $grokWrapperPath
    Ensure-GrokHomeOnUserPath

    Write-Ok "Installed: $targetPath"
    Write-Ok "Wrappers:  $groksWrapperPath, $grokWrapperPath"
    Write-Warn 'Already open terminals may need to be reopened before groks is found.'
    Write-Info 'In this shell you can run: groks help'
    Write-Info 'Next: groks env openai  ->  groks test openai  ->  groks use company  -> quit Grok fully'
}

function Do-Diagnose {
    $sourcePath = Get-ScriptSourcePath
    $installedScript = Join-Path $GrokHome 'groks.ps1'
    $groksWrapperPath = Join-Path $GrokHome 'groks.cmd'
    $grokWrapperPath = Join-Path $GrokHome 'grok.cmd'
    $homeFull = try { [System.IO.Path]::GetFullPath($GrokHome).TrimEnd('\') } catch { $GrokHome }

    Write-Host ''
    Write-Host '===== groks.ps1 diagnostic ====='
    Write-Host "Tool version    : $($script:ToolVersion)"
    Write-Host "Script path     : $sourcePath"
    Write-Host "GrokHome        : $GrokHome"
    Write-Host "Config path     : $script:ConfigPath"
    Write-Host "Config exists   : $(Test-Path -LiteralPath $script:ConfigPath)"
    Write-Host "GrokHome exists : $(Test-Path -LiteralPath $GrokHome -PathType Container)"

    Write-Host ''
    Write-Host '--- Download / Mark of the Web ---'
    foreach ($p in @($sourcePath, $installedScript)) {
        if ([string]::IsNullOrWhiteSpace($p)) { continue }
        if (-not (Test-Path -LiteralPath $p -PathType Leaf)) {
            Write-Host "  $p : [missing]"
            continue
        }
        $blocked = Test-MarkOfTheWeb $p
        Write-Host "  $p : $(if ($blocked) { '[BLOCKED]' } else { '[not blocked]' })"
    }

    if (Test-Path -LiteralPath $GrokHome -PathType Container) {
        $testFile = Join-Path $GrokHome '.write-test.tmp'
        try {
            [System.IO.File]::WriteAllText($testFile, 'test', [System.Text.Encoding]::ASCII)
            Remove-Item -LiteralPath $testFile -Force
            Write-Host 'Write test      : OK'
        }
        catch {
            Write-Warn 'Write test      : FAILED'
        }
    }
    else {
        Write-Host 'Write test      : (GrokHome missing; install will create it)'
    }

    $running = Get-RunningGrokProcesses
    Write-Host ''
    Write-Host '--- Running Grok ---'
    if ($running.Count -eq 0) {
        Write-Host '  (none)'
    }
    else {
        foreach ($proc in $running) {
            Write-Host "  $($proc.ProcessName) PID=$($proc.Id) StartTime=$($proc.StartTime)"
        }
        Write-Warn 'Quit these fully after config/key changes.'
    }

    Write-Host ''
    Write-Host '--- Config locations ---'
    foreach ($ac in @(
            @{ Name = 'GrokHome'; Path = $script:ConfigPath },
            @{ Name = 'User .grok'; Path = "$env:USERPROFILE\.grok\config.toml" }
        )) {
        $exists = Test-Path -LiteralPath $ac.Path -PathType Leaf
        Write-Host "  $($ac.Name): $($ac.Path) $(if ($exists) { '[EXISTS]' } else { '[not found]' })"
        if ($exists) {
            $content = [System.IO.File]::ReadAllText($ac.Path)
            $def = if ($content -match '(?m)^\s*default\s*=\s*"([^"]+)"') { $Matches[1] } else { '(not found in first match)' }
            $hasCompanyModel = [regex]::IsMatch(
                $content,
                '(?m)^\s*\[model\.(?:"grok-4\.5"|grok-4\.5)\]\s*(?:#.*)?$'
            )
            Write-Host "    -> default (first match) = $def"
            Write-Host "    -> [model.grok-4.5] = $(if ($hasCompanyModel) { 'present' } else { 'absent' })"
        }
    }

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    Write-Host ''
    Write-Host '--- Environment ---'
    Write-Host "  GROK_HOME               : $(if ($env:GROK_HOME) { $env:GROK_HOME } else { '(not set)' })"
    Write-Host "  SHU_API_KEY (User)      : $(if ([Environment]::GetEnvironmentVariable($script:OpenAIKeyName, 'User')) { 'set' } else { 'not set' })"
    Write-Host "  SHU_DEEPSEEK_API_KEY    : $(if ([Environment]::GetEnvironmentVariable($script:DeepSeekKeyName, 'User')) { 'set' } else { 'not set' })"
    Write-Host "  SHU_GROK_API_KEY (User) : $(if ([Environment]::GetEnvironmentVariable($script:KeyName, 'User')) { 'set' } else { 'not set' })"
    Write-Host "  User PATH has GrokHome  : $(if (Test-PathListContains $userPath $homeFull) { 'yes' } else { 'no' })"
    Write-Host "  groks.cmd               : $(if (Test-Path -LiteralPath $groksWrapperPath) { $groksWrapperPath } else { '(missing - run install)' })"
    Write-Host "  grok.cmd                : $(if (Test-Path -LiteralPath $grokWrapperPath) { $grokWrapperPath } else { '(missing - run install)' })"
    Write-Host "  ExecutionPolicy         : $(Get-ExecutionPolicy)"
    Write-Host ''
    Write-Host 'If disk looks correct but Grok does not: full quit + restart, then groks test.'
    Write-Host ''
}

function Manage-Backups {
    switch (([string]$Value).ToLowerInvariant()) {
        'list' {
            $files = Get-BackupFiles
            if ($files.Count -eq 0) { Write-Host 'No backups found.'; return }
            $files | Select-Object Name, Length, LastWriteTime | Format-Table -AutoSize
        }
        'clear' {
            $ok = $Force
            if (-not $ok) {
                $answer = Read-Host 'Delete all groks configuration backups? Type DELETE to confirm'
                $ok = $answer -ceq 'DELETE'
            }
            if (-not $ok) { Write-Warn 'Backup deletion cancelled.'; return }
            Invoke-WithConfigLock {
                $files = Get-BackupFiles
                foreach ($file in $files) { Remove-BackupFile $file }
                Write-Ok "Removed $($files.Count) backup(s)."
            }
        }
        'restore' {
            Invoke-WithConfigLock {
                $files = Get-BackupFiles
                if ($files.Count -eq 0) { throw 'No configuration backup is available to restore.' }
                $source = $files[0]
                $text = [System.IO.File]::ReadAllText($source.FullName)
                Assert-ConfigStructure $text
                $backup = Write-ConfigAtomic $text
                Write-Ok "Restored: $($source.FullName)"
                if ($backup) { Write-Info "Previous current config backed up as: $backup" }
            }
            Show-RestartWarning
        }
        default { throw 'Usage: groks backup list|restore|clear [-Force]' }
    }
}

function Show-Help {
    Write-Host @"

================================================================
  groks.ps1  v$($script:ToolVersion)  — 公司 Grok Build 网关切换
================================================================

【不要双击运行本脚本】请用记事本打开阅读说明，或在文件夹里开终端执行。

打开终端（在脚本所在文件夹）:
  资源管理器地址栏输入 powershell 回车
  或地址栏输入 cmd 回车
  或 Shift+右键 → 在此处打开 PowerShell / 终端

可选 install（装到 %USERPROFILE%\.grok 并加 PATH）:
  powershell -ExecutionPolicy Bypass -File .\groks.ps1 install

  若被拦截可先:
  Unblock-File .\groks.ps1

不 install 也可以直接用仓库里的脚本:
  powershell -ExecutionPolicy Bypass -File .\groks.ps1 status
  powershell -ExecutionPolicy Bypass -File .\groks.ps1 env openai
  powershell -ExecutionPolicy Bypass -File .\groks.ps1 use company

推荐顺序:
  groks env openai       输入 SHU_API_KEY（默认 GPT 分组）
  groks test openai      测网关与 gpt-5.6-sol 权限
  groks use company      切公司网关
  然后彻底退出 Grok，新终端直接运行 grok

其它命令:
  groks use official     切回官方 grok-4.5（使用 grok login 订阅）
  groks status           状态
  groks diagnose         排查（下载封锁 / 路径 / 进程）
  groks env [openai|deepseek|grok] [clear]
  groks test [openai|deepseek|grok]
  groks backup list|restore|clear

company 模式写入 %USERPROFILE%\.grok\config.toml:
  [models]
  default = "gpt-5.6-sol"

  [model."gpt-5.6-sol"]
  model = "gpt-5.6-sol"
  base_url = "https://api.shu.cool/v1"
  env_key = "SHU_API_KEY"
  api_backend = "responses"

改完没生效:
  1) 彻底退出 Grok 再开
  2) groks status / groks diagnose
  3) company 应为 default = "gpt-5.6-sol"
  4) official 应为 default = "grok-4.5" 且无托管模型目录

说明:
  - Grok Build 支持 Windows：官方可用 irm https://x.ai/cli/install.ps1 | iex
  - 配置目录：%USERPROFILE%\.grok\config.toml
  - 不需要管理员
  - 配置有备份与原子写入；进程不会热加载，必须重启
  - 完整说明也写在本 .ps1 文件最顶部（可用记事本打开）

"@
}

switch ($Command.ToLowerInvariant()) {
    'env' { Set-CompanyKey }
    'test' { Test-CompanyConnection }
    'backup' { Manage-Backups }
    'install' { Do-Install }
    'diagnose' { Do-Diagnose }
    'use' {
        switch (([string]$Value).ToLowerInvariant()) {
            'company' {
                if ([string]::IsNullOrWhiteSpace((Get-CompanyKey $script:OpenAIKeyName))) {
                    throw "$($script:OpenAIKeyName) is not set. Run: groks env openai"
                }
                Set-ProviderConfig 'company'
                Write-ProviderMode 'company'
                Write-Ok "Grok is configured to use the company gateway ($($script:CompanyModelKey))."
            }
            'official' {
                Set-ProviderConfig 'official'
                Write-ProviderMode 'official'
                Write-Ok "Grok is configured to use official SpaceXAI models ($($script:OfficialModel))."
                Write-Info 'Sign in with: grok login   (or set XAI_API_KEY).'
            }
            default { throw 'Usage: groks use official|company' }
        }
        Show-RestartWarning
        Show-Status
    }
    'status' { Show-Status }
    'help' { Show-Help }
}

<#
===============================================================================
 调研与验证记录
===============================================================================

记录目的：
本脚本可脱离仓库单独分发。本节保留配置依据、实测结果和验证边界，避免后续维护者
仅凭模型名称猜测接口类型。记录中不包含 API Key、登录令牌或会话正文。

一、Grok Build 源码确认（本地源码版本 0.2.114）

1. 自定义模型由 [model.<id>] 配置；本脚本实际使用的主要字段包括 model、name、
   base_url、env_key、api_backend、context_window、max_completion_tokens、
   supports_reasoning_effort、reasoning_effort 和 reasoning_efforts。
2. api_backend 决定请求和响应协议，支持 chat_completions、responses、messages。
   客户端不会根据返回 JSON 自动识别协议，也不会在三种协议之间自动转换。
3. chat_completions 请求 /v1/chat/completions，响应使用 created。
   responses 请求 /v1/responses，响应对象要求 created_at。
4. Responses 流式事件由 async-openai 类型直接反序列化。Grok Build 的兼容逻辑
   只会剔除无法识别的 xAI 工具，不会补 created_at 或 output_text.annotations。
5. 因此 HTTP 200 不代表兼容：响应结构与 api_backend 不一致时，仍会出现
   “serialization error: missing field ...”。

二、api.shu.cool 网关非流式实测（2026-07-29）

证据文件：providers/pi/gateway-raw-responses.json
探测脚本：providers/pi/probe_gateway.py
报告不含密钥；当次 19 个探测均收到 HTTP 2xx，但只验证了 stream=false。

1. gpt-5.6-sol、gpt-5.6-terra、gpt-5.6-luna、gpt-5.5、gpt-5.4、
   gpt-5.4-mini、gpt-5.3-codex-spark：
   /v1/responses 返回 object=response、created_at，并在 output_text 中返回
   annotations=[]。
2. grok-4.5：
   /v1/responses 返回 object=response、created_at，非流式 Responses 格式完整。
3. deepseek-v4-flash-vision-exp、deepseek-v4-pro：
   /v1/chat/completions 返回 object=chat.completion、created，不是 Responses 格式。
   所以本脚本将 deepseek-v4-pro 固定为 api_backend="chat_completions"。
4. grok-4.3、grok-build、grok-composer-2.5-fast：
   当次通过 /v1/chat/completions 验证。该结论只覆盖这些精确模型编号，不能自动
   外推到 grok-4.5-latest、grok-build-latest 或其他带 latest 的别名。

三、Grok CLI 流式运行观察

1. DeepSeek 被配置为 responses 时出现 missing field created_at；现有抓包没有保存
   这次失败的原始 /responses 数据，因此不臆测其完整结构。已确认可用的协议是
   chat_completions，改用后 deepseek-v4-pro 的实际会话完成。
2. GPT 的非流式探测包含 annotations=[]；实际流式会话先收到首 token，随后报
   missing field annotations 并把该轮标记为失败。结合 Responses 类型结构判断，
   缺失字段位于后续 SSE 响应对象中，大概率是终止事件；原始失败帧未被日志保存。
3. 若修中转层，应在 Responses SSE 的 response.output[].content[] 中为每个
   type=output_text 的对象保证 annotations=[]；不能靠切换 GPT 后端解决。

四、provider-mode 与凭证结论

1. company 模式写入托管模型目录；各分组通过 env_key 读取
   SHU_API_KEY、SHU_DEEPSEEK_API_KEY、SHU_GROK_API_KEY，配置文件不落明文 Key。
2. official 模式移除托管模型目录和中转覆盖，恢复官方 grok-4.5，并继续使用
   grok login 或 XAI_API_KEY 的官方认证。
3. Grok 进程不会热加载这些配置和环境变量，切换后必须彻底退出并重新启动。

五、本脚本已做的验证

1. macOS Bash 3.2 空数组兼容问题已处理；Bash 语法检查通过。
2. 示例 config.toml.txt 已通过 TOML 解析。
3. 使用临时 GROK_HOME 实际生成 company 配置并断言：
   deepseek-v4-pro=chat_completions，gpt-5.6-sol=responses，
   grok-4.5=responses。
4. Windows 脚本与 Bash 生成逻辑保持一致，但当前没有 Windows PowerShell
   运行环境，尚未在真实 Windows 主机执行端到端验证。

维护原则：
新增或修改模型前，分别验证 /v1/models、目标推理端点、非流式返回和流式 SSE；
不要因为 HTTP 200 或模型名称相似就假设协议兼容。
#>
