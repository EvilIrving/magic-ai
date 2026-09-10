#Requires -Version 5.1
<#
================================================================
  gpts.ps1  —  公司 Codex / ChatGPT 网关切换（shu）
================================================================

【本文件即完整说明书】可单独拷走使用，不必回仓库或上网搜文档。
  - 可以用记事本 / VS Code / 右键「编辑」打开本文件查看。
  - 不要双击「运行」本文件：双击多半用记事本打开，或窗口一闪就关，且无法传 install 等参数。
  - 不需要管理员权限。请用「平时登录 Windows、并运行 ChatGPT/Codex」的同一个用户操作。

意图:
  把 Codex / ChatGPT Desktop 在「官方 OpenAI」与「公司中转 api.shu.cool」之间切换；
  改 ~/.codex 等价路径 %USERPROFILE%\.codex\config.toml，Key 为 SHU_API_KEY。

----------------------------------------------------------------
一、怎么打开终端（在脚本所在文件夹里，任选一种）
----------------------------------------------------------------
  1) 资源管理器打开本文件所在文件夹
     → 点击上方地址栏 → 输入 powershell → 回车
  2) 或在地址栏输入 cmd → 回车
  3) 或在文件夹空白处 Shift+右键 →「在此处打开 PowerShell / 终端」

  不要：开始菜单乱开一个 PowerShell 却不知道当前目录在哪（除非你会 cd 进来）。

----------------------------------------------------------------
二、首次安装（微信 / 网盘 / 浏览器下载后）
----------------------------------------------------------------
  下载后若文件名变成 gpts.ps1.txt，先改回 gpts.ps1。

  在「本文件所在文件夹」的 PowerShell 里执行（整行复制）：

    powershell -ExecutionPolicy Bypass -File .\gpts.ps1 install

  若提示被阻止，可先解除下载标记再装：

    Unblock-File .\gpts.ps1
    powershell -ExecutionPolicy Bypass -File .\gpts.ps1 install

  install 会：
    - 复制到  %USERPROFILE%\.codex\gpts.ps1
    - 生成     %USERPROFILE%\.codex\gpts.cmd
    - 把 .codex 加入当前用户 PATH（并尽量解除下载封锁）
    - 当前窗口一般可直接用 gpts；其它已打开的终端请关掉重开

----------------------------------------------------------------
三、推荐流程（装完后）
----------------------------------------------------------------
    gpts env              输入公司 SHU API Key（隐藏输入，推荐）
    gpts test             测网关和 Key 是否可用
    gpts use company      切到公司网关 https://api.shu.cool/v1

  然后：彻底退出 ChatGPT / Codex（任务栏右下角托盘也要退），再重新打开应用。
  只关窗口不够。磁盘上的配置/环境变量，正在运行的桌面端不会自动热加载。

  也可在 PowerShell 中为当前用户永久设置 Key：
    [Environment]::SetEnvironmentVariable('SHU_API_KEY', '你的 Key', 'User')

  切回官方：
    gpts use official

----------------------------------------------------------------
四、常用命令
----------------------------------------------------------------
    gpts status           看当前 provider / Key / 是否在跑应用
    gpts diagnose         路径、下载封锁、写权限、进程等排查
    gpts help             打印本说明（命令行版）
    gpts env clear        删除本机用户环境变量里的 SHU_API_KEY
    gpts backup list|restore|clear

  未 install 时，在脚本目录也可直接：
    powershell -ExecutionPolicy Bypass -File .\gpts.ps1 status
    powershell -ExecutionPolicy Bypass -File .\gpts.ps1 use company

----------------------------------------------------------------
五、改完「没生效」时先查这几条
----------------------------------------------------------------
  1) 应用是否还在跑（托盘）→ 必须全退再开
  2) gpts status 里 Provider 是否已是 shu、API key 是否 set
  3) gpts test 是否通过（网络 / Key）
  4) gpts diagnose 看 Mark of the Web、config 路径、进程列表
  5) 记事本打开：  %USERPROFILE%\.codex\config.toml
     应有 model_provider = "shu" 和 [model_providers.shu]

----------------------------------------------------------------
六、常见故障
----------------------------------------------------------------
  · 双击没反应 → 在脚本目录用 powershell -File .\gpts.ps1 help
  · 禁止运行脚本 → -ExecutionPolicy Bypass；或 Unblock-File .\gpts.ps1
  · 文件名 .ps1.txt → 改回 .ps1
  · SHU_API_KEY 未设置 → gpts env
  · HTTP 401/403 → Key 错误/过期
  · 连不上网关 → VPN/网络；gpts test
  · 改完仍官方 → 全退托盘再开；查 config.toml

----------------------------------------------------------------
七、安全说明（简要）
----------------------------------------------------------------
  - 从微信/网盘下载的文件可能带「Mark of the Web」；install 会尝试自动解除。
  - -ExecutionPolicy Bypass 只绕过 PowerShell 执行策略，不是关闭杀毒/AMSI。
  - SHU_API_KEY 写在「当前 Windows 用户」环境变量中，本用户下其它程序也能读到。
  - 不要用管理员身份乱跑，以免写到错误用户目录。

.SYNOPSIS
  Switch Codex between official OpenAI and company shu gateway.

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File .\gpts.ps1 install
  gpts env
  gpts test
  gpts use company
#>

[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet('env', 'use', 'test', 'backup', 'status', 'help', 'install', 'diagnose')]
    [string]$Command = 'status',

    [Parameter(Position = 1)]
    [AllowEmptyString()]
    [string]$Value,

    [string]$CodexHome = $(
        if ([string]::IsNullOrWhiteSpace($env:CODEX_HOME)) {
            Join-Path $env:USERPROFILE '.codex'
        }
        else {
            $env:CODEX_HOME
        }
    ),

    [ValidateRange(1, 100)]
    [int]$BackupLimit = 3,

    [ValidateRange(3, 60)]
    [int]$TimeoutSec = 15,

    [switch]$Force
)

$ErrorActionPreference = 'Stop'
$script:ConfigPath = Join-Path $CodexHome 'config.toml'
$script:BackupDir = Join-Path $CodexHome 'provider-switch-backups'
$script:LockPath = Join-Path $CodexHome '.setup-shu.lock'
$script:KeyName = 'SHU_API_KEY'
$script:CompanyProvider = 'shu'
$script:OfficialProvider = 'openai'
$script:CompanyBaseUrl = 'https://api.shu.cool/v1'
$script:ToolVersion = '1.0.3'
$script:Utf8NoBom = New-Object System.Text.UTF8Encoding($false)

# Fields/sections left by other provider setups can mask or override the selected
# model_provider. gpts.ps1 does not install a custom models.json catalog, so
# DeepSeek's catalog/login fields are removed instead of rewritten.
$script:ConflictKeys = @(
    'model_catalog_json',
    'preferred_auth_method',
    'forced_login_method',
    'profile',
    'openai_base_url',
    'oss_provider'
)
$script:ConflictNotes = New-Object System.Collections.Generic.List[string]

$script:CompanySettings = @(
    [pscustomobject]@{ Key = 'name'; Line = 'name = "shu company"' },
    [pscustomobject]@{ Key = 'base_url'; Line = 'base_url = "https://api.shu.cool/v1"' },
    [pscustomobject]@{ Key = 'env_key'; Line = 'env_key = "SHU_API_KEY"' },
    [pscustomobject]@{ Key = 'wire_api'; Line = 'wire_api = "responses"' },
    [pscustomobject]@{ Key = 'requires_openai_auth'; Line = 'requires_openai_auth = false' }
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
    # Best-effort: new processes may see updated User env. Running ChatGPT/Codex will not.
    try {
        if (-not ('SetupShu.NativeMethods' -as [type])) {
            Add-Type -Namespace SetupShu -Name NativeMethods -MemberDefinition @'
[System.Runtime.InteropServices.DllImport("user32.dll", SetLastError = true, CharSet = System.Runtime.InteropServices.CharSet.Auto)]
public static extern IntPtr SendMessageTimeout(
    IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam,
    uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
'@
        }
        $result = [UIntPtr]::Zero
        [void][SetupShu.NativeMethods]::SendMessageTimeout(
            [IntPtr]0xffff, [uint32]0x1a, [UIntPtr]::Zero, 'Environment',
            [uint32]2, [uint32]5000, [ref]$result
        )
    }
    catch { }
}

function Ensure-CodexHome {
    if ([string]::IsNullOrWhiteSpace($CodexHome)) { throw 'Codex home path is empty.' }
    if (-not (Test-Path -LiteralPath $CodexHome)) {
        New-Item -ItemType Directory -Path $CodexHome | Out-Null
    }
    if (-not (Test-Path -LiteralPath $CodexHome -PathType Container)) {
        throw "Codex home is not a directory: $CodexHome"
    }
}

function Invoke-WithConfigLock([scriptblock]$Action) {
    Ensure-CodexHome
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
    if ($null -eq $lock) { throw 'Another setup-shu process is modifying the configuration. Try again later.' }
    try { & $Action }
    finally { $lock.Dispose() }
}

function Get-ConfigText {
    Ensure-CodexHome
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

function Get-CurrentProvider([string]$Text) {
    $lines = @(Get-TomlLines $Text)
    $providerLines = @()
    foreach ($line in $lines) {
        if ($line -match '^\s*\[') { break }
        if ($line -match '^\s*model_provider\s*=') { $providerLines += $line }
    }
    if ($providerLines.Count -gt 1) { throw 'config.toml contains duplicate top-level model_provider entries.' }
    if ($providerLines.Count -eq 1) {
        if ($providerLines[0] -match '^\s*model_provider\s*=\s*"([^"]+)"\s*(?:#.*)?$') {
            return $Matches[1]
        }
        if ($providerLines[0] -match "^\s*model_provider\s*=\s*'([^']+)'\s*(?:#.*)?$") {
            return $Matches[1]
        }
        throw 'The top-level model_provider value in config.toml is not a valid quoted string.'
    }
    return $script:OfficialProvider
}

function Test-DeepSeekModelLine([string]$Line) {
    return $Line -match '^\s*model\s*=\s*"deepseek-[A-Za-z0-9_.-]+'
}

function Test-ConflictSection([string]$Header) {
    return (
        $Header -eq 'profiles' -or
        $Header -like 'profiles.*' -or
        $Header -eq 'model_providers.deepseek' -or
        $Header -like 'model_providers.deepseek.*'
    )
}

function Assert-ConfigStructure([string]$Text) {
    if ($Text.IndexOf([char]0) -ge 0) { throw 'config.toml contains a NUL character.' }
    [void](Get-CurrentProvider $Text)
    $providerHeaders = @([regex]::Matches($Text, '(?m)^\s*\[model_providers\.shu\]\s*(?:#.*)?$'))
    if ($providerHeaders.Count -gt 1) {
        throw 'config.toml contains duplicate [model_providers.shu] sections.'
    }
}

function Set-ManagedConfig([string]$Text, [string]$Provider) {
    Assert-ConfigStructure $Text
    $lines = @(Get-TomlLines $Text)

    $firstHeader = $lines.Count
    for ($i = 0; $i -lt $lines.Count; $i++) {
        if ($lines[$i] -match '^\s*\[') { $firstHeader = $i; break }
    }

    $kept = New-Object System.Collections.Generic.List[string]
    for ($i = 0; $i -lt $lines.Count; $i++) {
        if ($i -lt $firstHeader) {
            $lineKey = ''
            if ($lines[$i] -match '^\s*([^=]+?)\s*=') { $lineKey = $Matches[1].Trim() }
            if ($lineKey -and $script:ConflictKeys -contains $lineKey) {
                $script:ConflictNotes.Add("top-level $lineKey")
                continue
            }
            if (Test-DeepSeekModelLine $lines[$i]) {
                $script:ConflictNotes.Add('top-level model (DeepSeek model)')
                continue
            }
        }
        [void]$kept.Add($lines[$i])
    }
    $lines = @($kept.ToArray())

    $firstHeader = $lines.Count
    for ($i = 0; $i -lt $lines.Count; $i++) {
        if ($lines[$i] -match '^\s*\[') { $firstHeader = $i; break }
    }

    $providerLine = -1
    for ($i = 0; $i -lt $firstHeader; $i++) {
        if ($lines[$i] -match '^\s*model_provider\s*=') { $providerLine = $i; break }
    }
    if ($providerLine -ge 0) {
        $lines[$providerLine] = "model_provider = `"$Provider`""
    }
    else {
        $lines = @("model_provider = `"$Provider`"") + $lines
    }

    $sectionStarts = @()
    for ($i = 0; $i -lt $lines.Count; $i++) {
        if ($lines[$i] -match '^\s*\[model_providers\.shu\]\s*(?:#.*)?$') { $sectionStarts += $i }
    }
    if ($sectionStarts.Count -gt 1) { throw 'Duplicate [model_providers.shu] sections are not supported.' }

    if ($sectionStarts.Count -eq 0) {
        if ($lines.Count -gt 0 -and $lines[$lines.Count - 1] -ne '') { $lines += '' }
        $lines += '[model_providers.shu]'
        foreach ($setting in $script:CompanySettings) { $lines += $setting.Line }
    }
    else {
        $start = $sectionStarts[0]
        $end = $lines.Count
        for ($i = $start + 1; $i -lt $lines.Count; $i++) {
            if ($lines[$i] -match '^\s*\[') { $end = $i; break }
        }

        $seen = @{}
        $section = @()
        if ($end -gt ($start + 1)) {
            foreach ($line in @($lines[($start + 1)..($end - 1)])) {
                $managed = $null
                foreach ($setting in $script:CompanySettings) {
                    if ($line -match ("^\s*" + [regex]::Escape($setting.Key) + "\s*=")) {
                        $managed = $setting
                        break
                    }
                }
                if ($null -eq $managed) {
                    $section += $line
                }
                elseif (-not $seen.ContainsKey($managed.Key)) {
                    $section += $managed.Line
                    $seen[$managed.Key] = $true
                }
            }
        }
        foreach ($setting in $script:CompanySettings) {
            if (-not $seen.ContainsKey($setting.Key)) { $section += $setting.Line }
        }

        $prefix = @($lines[0..$start])
        $suffix = @()
        if ($end -lt $lines.Count) { $suffix = @($lines[$end..($lines.Count - 1)]) }
        $lines = $prefix + $section + $suffix
    }

    # Remove profile and old DeepSeek provider sections left by other setups.
    $kept = New-Object System.Collections.Generic.List[string]
    for ($i = 0; $i -lt $lines.Count; $i++) {
        if ($lines[$i] -match '^\s*\[') {
            $header = ''
            if ($lines[$i] -match '^\[([^\]]+)\]') { $header = $Matches[1].Trim() }
            if (Test-ConflictSection $header) {
                $script:ConflictNotes.Add("section [$header]")
                $i++
                while ($i -lt $lines.Count -and $lines[$i] -notmatch '^\s*\[') { $i++ }
                if ($i -lt $lines.Count) { $i-- }
                continue
            }
        }
        [void]$kept.Add($lines[$i])
    }
    $lines = @($kept.ToArray())

    $result = ($lines -join [Environment]::NewLine).TrimEnd() + [Environment]::NewLine
    Assert-ConfigStructure $result
    if ((Get-CurrentProvider $result) -ne $Provider) { throw 'Provider verification failed before writing config.toml.' }
    return $result
}

function Write-ConfigAtomic([string]$Text) {
    Assert-ConfigStructure $Text
    Ensure-CodexHome
    $temp = Join-Path $CodexHome ('.config.toml.' + [Guid]::NewGuid().ToString('N') + '.tmp')
    $replaceBackup = Join-Path $CodexHome ('.config.toml.' + [Guid]::NewGuid().ToString('N') + '.rollback')
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

function Set-Provider([string]$Provider) {
    Invoke-WithConfigLock {
        $script:ConflictNotes.Clear()
        $current = Get-ConfigText
        $updated = Set-ManagedConfig $current $Provider
        if ($updated -eq $current) {
            Write-Info 'Configuration is already up to date; no backup was created.'
            return
        }
        $backup = Write-ConfigAtomic $updated
        if ($backup) { Write-Info "Backup: $backup" }
        foreach ($note in $script:ConflictNotes) {
            Write-Warn "Removed conflicting config entry: $note"
        }
    }
}

function ConvertFrom-SecureValue([Security.SecureString]$SecureValue) {
    $pointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($SecureValue)
    try { return [Runtime.InteropServices.Marshal]::PtrToStringBSTR($pointer) }
    finally { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($pointer) }
}

function Get-RunningCodexProcesses {
    return @(Get-Process -ErrorAction SilentlyContinue | Where-Object {
        $_.ProcessName -in @('ChatGPT', 'Codex')
    })
}

function Show-RestartWarning {
    param([string]$Context = 'config')

    Write-Host ''
    Write-Warn "On-disk $Context was updated. Desktop apps do NOT reload it while still running."
    Write-Warn 'Fully quit ChatGPT / Codex (including the tray icon near the clock), then start again.'
    Write-Warn 'Closing only the window is often not enough.'

    $running = Get-RunningCodexProcesses
    if ($running.Count -gt 0) {
        $names = ($running | ForEach-Object { "$($_.ProcessName)(PID $($_.Id))" }) -join ', '
        Write-Warn "Still running now: $names"
        Write-Warn 'Until those processes exit, the app keeps the OLD provider / API key.'
    }
    else {
        Write-Info 'No ChatGPT/Codex process detected. You can start the app now.'
    }
    Write-Host ''
}

function Set-CompanyKey {
    if (([string]$Value).ToLowerInvariant() -eq 'clear') {
        [Environment]::SetEnvironmentVariable($script:KeyName, $null, 'User')
        [Environment]::SetEnvironmentVariable($script:KeyName, $null, 'Process')
        Send-EnvironmentChange
        Write-Ok "$($script:KeyName) was removed for the current user and process."
        Show-RestartWarning -Context 'API key'
        return
    }

    $key = $Value
    if ([string]::IsNullOrWhiteSpace($key)) {
        $secureKey = Read-Host 'Enter SHU API Key (input is hidden)' -AsSecureString
        $key = ConvertFrom-SecureValue $secureKey
    }
    else {
        Write-Warn 'The API key may now be present in PowerShell command history. Prefer: gpts env'
    }
    # Clipboard paste often appends CR (Windows/docs); strip before validation.
    $key = ($key -replace "`r", '').Trim()
    if ([string]::IsNullOrWhiteSpace($key)) { throw 'API Key cannot be empty.' }
    if ($key -match '[\x00-\x1F\x7F]') {
        throw 'API Key contains an invalid control character (often from paste). Re-enter a single-line key.'
    }
    [Environment]::SetEnvironmentVariable($script:KeyName, $key, 'User')
    [Environment]::SetEnvironmentVariable($script:KeyName, $key, 'Process')
    Send-EnvironmentChange
    Write-Ok "$($script:KeyName) is set for the current Windows user and this process."
    Write-Warn 'The user environment variable is readable by programs running as this Windows user.'
    Show-RestartWarning -Context 'API key'
}

function Show-Status {
    $text = Get-ConfigText
    $provider = Get-CurrentProvider $text
    $userKeySet = -not [string]::IsNullOrWhiteSpace(
        [Environment]::GetEnvironmentVariable($script:KeyName, 'User')
    )
    $processKeySet = -not [string]::IsNullOrWhiteSpace(
        [Environment]::GetEnvironmentVariable($script:KeyName, 'Process')
    )
    $companyBlock = [regex]::IsMatch($text, '(?m)^\s*\[model_providers\.shu\]\s*(?:#.*)?$')
    $running = Get-RunningCodexProcesses

    Write-Host ''
    Write-Host '===== Codex provider status ====='
    Write-Host "Tool version     : $($script:ToolVersion)"
    Write-Host "Config           : $($script:ConfigPath)"
    Write-Host "Provider         : $provider"
    Write-Host "Company config   : $(if ($companyBlock) { 'installed' } else { 'not installed' })"
    Write-Host "API key (User)   : $(if ($userKeySet) { 'set' } else { 'not set' })"
    Write-Host "API key (Process): $(if ($processKeySet) { 'set' } else { 'not set' })"
    Write-Host "Backups          : $((Get-BackupFiles).Count) / $BackupLimit retained"
    Write-Host "Running app      : $(if ($running.Count -gt 0) { 'yes' } else { 'no' })"
    Write-Host ''

    if ($provider -eq $script:CompanyProvider -and -not $userKeySet) {
        Write-Warn 'Company provider is selected, but SHU_API_KEY is missing.'
    }
    if ($running.Count -gt 0 -and $provider -eq $script:CompanyProvider) {
        Write-Warn 'ChatGPT/Codex is still running — quit tray + restart or changes may not appear.'
    }
}

function Test-CompanyConnection {
    $key = [Environment]::GetEnvironmentVariable($script:KeyName, 'User')
    if ([string]::IsNullOrWhiteSpace($key)) {
        throw 'SHU_API_KEY is not set. Run: gpts env   (or: .\gpts.ps1 env)'
    }
    $uri = $script:CompanyBaseUrl.TrimEnd('/') + '/models'
    Write-Info "Testing company gateway: $uri"
    try {
        $response = Invoke-WebRequest -Uri $uri -Method Get -Headers @{
            Authorization = "Bearer $key"
            Accept = 'application/json'
        } -TimeoutSec $TimeoutSec -UseBasicParsing
        Write-Ok "Company gateway is reachable and accepted the API key (HTTP $($response.StatusCode))."
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

function Ensure-CodexHomeOnUserPath {
    Ensure-CodexHome
    $homeFull = [System.IO.Path]::GetFullPath($CodexHome).TrimEnd('\')
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($null -eq $userPath) { $userPath = '' }

    if (-not (Test-PathListContains $userPath $homeFull)) {
        $newPath = if ($userPath.Trim().TrimEnd(';')) {
            "$($userPath.Trim().TrimEnd(';'));$homeFull"
        }
        else {
            $homeFull
        }
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        Send-EnvironmentChange
        Write-Ok "Added to user PATH: $homeFull"
    }
    else {
        Write-Info "User PATH already contains: $homeFull"
    }

    if (-not (Test-PathListContains $env:Path $homeFull)) {
        $env:Path = if ($env:Path -and $env:Path.TrimEnd(';')) {
            "$($env:Path.TrimEnd(';'));$homeFull"
        }
        else {
            $homeFull
        }
    }
}

function New-GptsWrapper([string]$WrapperPath) {
    $content = @"
@echo off
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0gpts.ps1" %*
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
        Write-Warn 'Rename it to gpts.ps1, then run install again.'
    }

    $targetPath = Join-Path $CodexHome 'gpts.ps1'
    $wrapperPath = Join-Path $CodexHome 'gpts.cmd'

    if (Test-Path -LiteralPath $targetPath) {
        $targetResolved = Resolve-Path -LiteralPath $targetPath -ErrorAction SilentlyContinue
        $sourceResolved = Resolve-Path -LiteralPath $sourcePath -ErrorAction SilentlyContinue
        if ($targetResolved -and $sourceResolved -and $targetResolved.Path -eq $sourceResolved.Path) {
            Write-Info "Already installed at $targetPath."
            New-GptsWrapper $wrapperPath
            if (Clear-MarkOfTheWeb $targetPath) {
                Write-Ok "Removed download block from installed script: $targetPath"
            }
            Ensure-CodexHomeOnUserPath
            Write-Warn 'Already open terminals may need to be reopened before gpts is found.'
            Write-Info 'Next: gpts env  ->  gpts test  ->  gpts use company  -> quit ChatGPT/Codex fully'
            return
        }
    }

    Ensure-CodexHome
    Copy-Item -LiteralPath $sourcePath -Destination $targetPath -Force
    if (Clear-MarkOfTheWeb $targetPath) {
        Write-Ok "Removed download block from installed script: $targetPath"
    }

    New-GptsWrapper $wrapperPath
    Ensure-CodexHomeOnUserPath

    Write-Ok "Installed: $targetPath"
    Write-Ok "Wrapper:   $wrapperPath"
    Write-Warn 'Already open terminals may need to be reopened before gpts is found.'
    Write-Info 'In this shell you can run: gpts help'
    Write-Info 'Next: gpts env  ->  gpts test  ->  gpts use company  -> quit ChatGPT/Codex fully'
}

function Do-Diagnose {
    $sourcePath = Get-ScriptSourcePath
    $installedScript = Join-Path $CodexHome 'gpts.ps1'
    $wrapperPath = Join-Path $CodexHome 'gpts.cmd'
    $homeFull = try { [System.IO.Path]::GetFullPath($CodexHome).TrimEnd('\') } catch { $CodexHome }

    Write-Host ''
    Write-Host '===== gpts.ps1 diagnostic ====='
    Write-Host "Tool version    : $($script:ToolVersion)"
    Write-Host "Script path     : $sourcePath"
    Write-Host "CodexHome       : $CodexHome"
    Write-Host "Config path     : $script:ConfigPath"
    Write-Host "Config exists   : $(Test-Path -LiteralPath $script:ConfigPath)"
    Write-Host "CodexHome exists: $(Test-Path -LiteralPath $CodexHome -PathType Container)"

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

    if (Test-Path -LiteralPath $CodexHome -PathType Container) {
        $testFile = Join-Path $CodexHome '.write-test.tmp'
        try {
            [System.IO.File]::WriteAllText($testFile, 'test', [System.Text.Encoding]::ASCII)
            Remove-Item -LiteralPath $testFile -Force
            Write-Host "Write test      : OK"
        }
        catch {
            Write-Warn 'Write test      : FAILED'
        }
    }
    else {
        Write-Host 'Write test      : (CodexHome missing; install will create it)'
    }

    Write-Host ''
    Write-Host '--- App install paths ---'
    foreach ($cp in @(
            @{ Name = 'Codex'; Path = "$env:LOCALAPPDATA\Programs\Codex" },
            @{ Name = 'ChatGPT'; Path = "$env:LOCALAPPDATA\Programs\ChatGPT" }
        )) {
        $exists = Test-Path -LiteralPath $cp.Path -PathType Container
        Write-Host "  $($cp.Name): $($cp.Path) $(if ($exists) { '[EXISTS]' } else { '[not found]' })"
    }

    $running = Get-RunningCodexProcesses
    Write-Host ''
    Write-Host '--- Running ChatGPT / Codex ---'
    if ($running.Count -eq 0) {
        Write-Host '  (none)'
    }
    else {
        foreach ($proc in $running) {
            Write-Host "  $($proc.ProcessName) PID=$($proc.Id) StartTime=$($proc.StartTime)"
        }
        Write-Warn 'Quit these fully (tray too) after config/key changes.'
    }

    Write-Host ''
    Write-Host '--- Config locations ---'
    foreach ($ac in @(
            @{ Name = 'CodexHome'; Path = $script:ConfigPath },
            @{ Name = 'User .codex'; Path = "$env:USERPROFILE\.codex\config.toml" }
        )) {
        $exists = Test-Path -LiteralPath $ac.Path -PathType Leaf
        Write-Host "  $($ac.Name): $($ac.Path) $(if ($exists) { '[EXISTS]' } else { '[not found]' })"
        if ($exists) {
            $content = [System.IO.File]::ReadAllText($ac.Path)
            $modelProvider = if ($content -match 'model_provider\s*=\s*"([^"]+)"') { $Matches[1] } else { '(not found)' }
            Write-Host "    -> model_provider = $modelProvider"
        }
    }

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    Write-Host ''
    Write-Host '--- Environment ---'
    Write-Host "  CODEX_HOME              : $(if ($env:CODEX_HOME) { $env:CODEX_HOME } else { '(not set)' })"
    Write-Host "  SHU_API_KEY (User)      : $(if ([Environment]::GetEnvironmentVariable('SHU_API_KEY', 'User')) { 'set' } else { 'not set' })"
    Write-Host "  User PATH has CodexHome : $(if (Test-PathListContains $userPath $homeFull) { 'yes' } else { 'no' })"
    Write-Host "  gpts.cmd                : $(if (Test-Path -LiteralPath $wrapperPath) { $wrapperPath } else { '(missing — run install)' })"
    Write-Host "  ExecutionPolicy         : $(Get-ExecutionPolicy)"
    Write-Host ''
    Write-Host 'If disk looks correct but the app does not: full quit (tray) + restart, then gpts test.'
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
                $answer = Read-Host 'Delete all setup-shu configuration backups? Type DELETE to confirm'
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
        default { throw 'Usage: gpts backup list|restore|clear [-Force]' }
    }
}

function Show-Help {
    Write-Host @"

================================================================
  gpts.ps1  v$($script:ToolVersion)  — 公司 Codex 网关切换
================================================================

【不要双击运行本脚本】请用记事本打开阅读说明，或在文件夹里开终端执行。

打开终端（在脚本所在文件夹）:
  资源管理器地址栏输入 powershell 回车
  或地址栏输入 cmd 回车
  或 Shift+右键 → 在此处打开 PowerShell / 终端

首次安装（微信/网盘下载后）:
  powershell -ExecutionPolicy Bypass -File .\gpts.ps1 install

  若被拦截可先:
  Unblock-File .\gpts.ps1

装完后推荐顺序:
  gpts env              输入 SHU API Key（隐藏输入）
  gpts test             测网关
  gpts use company      切公司网关
  然后彻底退出 ChatGPT/Codex（含托盘）再打开

手动永久设置当前用户 Key:
  [Environment]::SetEnvironmentVariable('SHU_API_KEY', '你的 Key', 'User')

其它命令:
  gpts use official     切回官方
  gpts status           状态
  gpts diagnose         排查（下载封锁 / 路径 / 进程）
  gpts env clear        删除 Key
  gpts backup list|restore|clear

未 install 时也可:
  powershell -ExecutionPolicy Bypass -File .\gpts.ps1 <命令>

改完没生效:
  1) 托盘彻底退出应用再开
  2) gpts status / gpts diagnose
  3) 看 %USERPROFILE%\.codex\config.toml 是否 model_provider = "shu"

说明:
  - 不需要管理员
  - install 会写入用户 PATH，并尝试解除下载封锁
  - 切换时会清理旧 DeepSeek 安装、profile/global override 等冲突字段
  - 不写入 models.json，也不安装自定义 ChatGPT Desktop 模型目录
  - 配置有备份与原子写入；桌面端不会热加载，必须重启应用
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
                if (-not [Environment]::GetEnvironmentVariable($script:KeyName, 'User')) {
                    throw 'SHU_API_KEY is not set. Run: gpts env'
                }
                Set-Provider $script:CompanyProvider
                Write-Ok 'Codex is configured to use the company gateway.'
            }
            'official' {
                Set-Provider $script:OfficialProvider
                Write-Ok 'Codex is configured to use official OpenAI / ChatGPT authentication.'
                Write-Info 'Sign in with ChatGPT Desktop or configure an official OpenAI API key.'
            }
            default { throw 'Usage: gpts use official|company' }
        }
        Show-RestartWarning
        Show-Status
    }
    'status' { Show-Status }
    'help' { Show-Help }
}
