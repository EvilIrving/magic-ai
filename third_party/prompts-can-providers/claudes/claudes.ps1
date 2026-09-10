#Requires -Version 5.1
<#
================================================================
  claudes.ps1 - Windows Claude Code provider switcher
================================================================

This is the Windows counterpart of claudes.sh. It is self-contained and does
not need repository-side common.json or providers/*.json files.

First install:
  powershell -ExecutionPolicy Bypass -File .\claudes.ps1 install

After opening a new terminal:
  claudes list
  claudes env deepseek
  claudes use deepseek
  claudes status

Supported routes:
  - Anthropic official login
  - DeepSeek, Kimi, and MiniMax official APIs
  - api.shu.cool OpenAI, DeepSeek, and Grok models

OpenAI and Grok subscription logins remain managed by gpts and groks. Product
subscriptions are not API credentials that Claude Code can reuse directly.

Legacy migration:
  claudes migrate

This imports non-empty keys from %USERPROFILE%\.claude\models.json into Windows
user environment variables. The legacy file is never deleted automatically.
#>

[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet('status', 'list', 'use', 'env', 'key', 'migrate', 'backup', 'install', 'help')]
    [string]$Command = 'status',

    [Parameter(Position = 1)]
    [AllowEmptyString()]
    [string]$Value,

    [Parameter(Position = 2)]
    [AllowEmptyString()]
    [string]$ExtraValue,

    [string]$ClaudeHome = $(
        if ([string]::IsNullOrWhiteSpace($env:CLAUDE_HOME)) {
            Join-Path $env:USERPROFILE '.claude'
        }
        else {
            $env:CLAUDE_HOME
        }
    ),

    [ValidateRange(1, 100)]
    [int]$BackupLimit = 3,

    [switch]$DryRun
)

$ErrorActionPreference = 'Stop'
$script:ToolVersion = '2.0.0'
$script:SettingsPath = Join-Path $ClaudeHome 'settings.json'
$script:LegacyModelsPath = if ([string]::IsNullOrWhiteSpace($env:CLAUDE_MODELS_FILE)) {
    Join-Path $ClaudeHome 'models.json'
}
else {
    $env:CLAUDE_MODELS_FILE
}
$script:BackupDir = Join-Path $ClaudeHome 'provider-switch-backups'
$script:LockPath = Join-Path $ClaudeHome '.claudes.lock'
$script:Utf8NoBom = New-Object System.Text.UTF8Encoding($false)

$script:ProviderCatalogJson = @'
{
  "official": {
    "name": "Anthropic Official",
    "baseUrl": "",
    "keyEnv": "",
    "models": {}
  },
  "deepseek": {
    "name": "DeepSeek Official API",
    "baseUrl": "https://api.deepseek.com/anthropic",
    "keyEnv": "DEEPSEEK_API_KEY",
    "models": {
      "haiku": "deepseek-v4-flash-vision-exp",
      "sonnet": "deepseek-v4-pro",
      "opus": "deepseek-v4-pro",
      "default": "deepseek-v4-pro"
    }
  },
  "kimi": {
    "name": "Kimi Official API",
    "baseUrl": "https://api.moonshot.cn/anthropic",
    "keyEnv": "KIMI_API_KEY",
    "models": {
      "haiku": "kimi-k2.5",
      "sonnet": "kimi-k2.5",
      "opus": "kimi-k2.5",
      "default": "kimi-k2.5",
      "reasoning": "kimi-k2.5"
    }
  },
  "minimax": {
    "name": "MiniMax Official API",
    "baseUrl": "https://api.minimaxi.com/anthropic",
    "keyEnv": "MINIMAX_API_KEY",
    "models": {
      "haiku": "MiniMax-M2.7",
      "sonnet": "MiniMax-M2.7",
      "opus": "MiniMax-M2.7",
      "default": "MiniMax-M2.7"
    }
  },
  "s-openai": {
    "name": "Shu OpenAI",
    "baseUrl": "https://api.shu.cool",
    "keyEnv": "SHU_API_KEY",
    "models": {
      "haiku": "gpt-5.6-luna",
      "sonnet": "gpt-5.6-terra",
      "opus": "gpt-5.6-sol",
      "default": "gpt-5.6-sol"
    }
  },
  "s-deepseek": {
    "name": "Shu DeepSeek",
    "baseUrl": "https://api.shu.cool",
    "keyEnv": "SHU_DEEPSEEK_API_KEY",
    "models": {
      "haiku": "deepseek-v4-flash-vision-exp",
      "sonnet": "deepseek-v4-pro",
      "opus": "deepseek-v4-pro",
      "default": "deepseek-v4-pro"
    }
  },
  "s-grok": {
    "name": "Shu Grok",
    "baseUrl": "https://api.shu.cool",
    "keyEnv": "SHU_GROK_API_KEY",
    "models": {
      "haiku": "grok-composer-2.5-fast",
      "sonnet": "grok-4.5",
      "opus": "grok-4.5",
      "default": "grok-4.5"
    }
  }
}
'@

$script:ProviderAliases = [ordered]@{
    'anthropic' = 'official'
    'shu-openai' = 's-openai'
    'shu-deepseek' = 's-deepseek'
    'shu-grok' = 's-grok'
    'shu' = 's-openai'
}
$script:ModelEnvKeys = [ordered]@{
    'haiku' = 'ANTHROPIC_DEFAULT_HAIKU_MODEL'
    'sonnet' = 'ANTHROPIC_DEFAULT_SONNET_MODEL'
    'opus' = 'ANTHROPIC_DEFAULT_OPUS_MODEL'
    'default' = 'ANTHROPIC_MODEL'
    'reasoning' = 'ANTHROPIC_REASONING_MODEL'
}
$script:ManagedEnvKeys = @(
    'ANTHROPIC_API_KEY',
    'ANTHROPIC_AUTH_TOKEN',
    'ANTHROPIC_BASE_URL',
    'ANTHROPIC_DEFAULT_FABLE_MODEL',
    'ANTHROPIC_DEFAULT_HAIKU_MODEL',
    'ANTHROPIC_DEFAULT_SONNET_MODEL',
    'ANTHROPIC_DEFAULT_OPUS_MODEL',
    'ANTHROPIC_MODEL',
    'ANTHROPIC_REASONING_MODEL'
)
$script:LegacyKeyProviders = [ordered]@{
    'DEEPSEEK_API_KEY' = @('deepseek')
    'KIMI_API_KEY' = @('kimi')
    'MINIMAX_API_KEY' = @('minimax')
    'SHU_API_KEY' = @('s-openai', 'shu-openai')
    'SHU_DEEPSEEK_API_KEY' = @('s-deepseek', 'shu-deepseek')
    'SHU_GROK_API_KEY' = @('s-grok', 'shu-grok')
}

function Write-Ok([string]$Message) { Write-Host $Message -ForegroundColor Green }
function Write-Info([string]$Message) { Write-Host $Message -ForegroundColor Cyan }
function Write-Warn([string]$Message) { Write-Host $Message -ForegroundColor Yellow }

function ConvertTo-PlainObject {
    param([Parameter(ValueFromPipeline = $true)]$InputObject)

    process {
        if ($null -eq $InputObject) { return $null }
        if ($InputObject -is [System.Collections.IDictionary]) {
            $result = [ordered]@{}
            foreach ($key in $InputObject.Keys) {
                $result[$key] = ConvertTo-PlainObject $InputObject[$key]
            }
            return $result
        }
        if ($InputObject -is [System.Collections.IEnumerable] -and $InputObject -isnot [string]) {
            $items = @()
            foreach ($item in $InputObject) {
                $items += ,(ConvertTo-PlainObject $item)
            }
            Write-Output -NoEnumerate $items
            return
        }
        if ($InputObject -is [pscustomobject]) {
            $result = [ordered]@{}
            foreach ($property in $InputObject.PSObject.Properties) {
                $result[$property.Name] = ConvertTo-PlainObject $property.Value
            }
            return $result
        }
        return $InputObject
    }
}

$script:Providers = ConvertTo-PlainObject ($script:ProviderCatalogJson | ConvertFrom-Json)

function Ensure-ClaudeHome {
    if ([string]::IsNullOrWhiteSpace($ClaudeHome)) { throw 'Claude home path is empty.' }
    if (-not (Test-Path -LiteralPath $ClaudeHome)) {
        New-Item -ItemType Directory -Path $ClaudeHome | Out-Null
    }
    if (-not (Test-Path -LiteralPath $ClaudeHome -PathType Container)) {
        throw "Claude home is not a directory: $ClaudeHome"
    }
}

function Invoke-WithConfigLock([scriptblock]$Action) {
    Ensure-ClaudeHome
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
    if ($null -eq $lock) { throw 'Another claudes process is modifying the configuration.' }
    try { & $Action }
    finally { $lock.Dispose() }
}

function Read-JsonObject([string]$Path, [switch]$MissingOk) {
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        if ($MissingOk) { return [ordered]@{} }
        throw "Missing file: $Path"
    }
    try {
        $text = [System.IO.File]::ReadAllText($Path)
        $value = ConvertTo-PlainObject ($text | ConvertFrom-Json)
    }
    catch {
        throw "Cannot read valid JSON from ${Path}: $($_.Exception.Message)"
    }
    if ($value -isnot [System.Collections.IDictionary]) {
        throw "Expected a JSON object in $Path"
    }
    return $value
}

function Copy-PlainObject([System.Collections.IDictionary]$Value) {
    $json = $Value | ConvertTo-Json -Depth 30 -Compress
    return ConvertTo-PlainObject ($json | ConvertFrom-Json)
}

function ConvertTo-StableJson([System.Collections.IDictionary]$Value, [switch]$Compress) {
    if ($Compress) { return ($Value | ConvertTo-Json -Depth 30 -Compress) }
    return ($Value | ConvertTo-Json -Depth 30)
}

function Write-JsonAtomic([string]$Path, [System.Collections.IDictionary]$Value) {
    $directory = Split-Path -Parent $Path
    if (-not (Test-Path -LiteralPath $directory)) {
        New-Item -ItemType Directory -Path $directory | Out-Null
    }
    $temp = Join-Path $directory ('.' + [System.IO.Path]::GetFileName($Path) + '.' + [Guid]::NewGuid().ToString('N') + '.tmp')
    try {
        $json = (ConvertTo-StableJson $Value) + [Environment]::NewLine
        [System.IO.File]::WriteAllText($temp, $json, $script:Utf8NoBom)
        if (Test-Path -LiteralPath $Path -PathType Leaf) {
            [System.IO.File]::Replace($temp, $Path, $null)
        }
        else {
            [System.IO.File]::Move($temp, $Path)
        }
    }
    finally {
        if (Test-Path -LiteralPath $temp) { Remove-Item -LiteralPath $temp -Force }
    }
}

function Get-BackupFiles {
    if (-not (Test-Path -LiteralPath $script:BackupDir -PathType Container)) { return @() }
    return @(Get-ChildItem -LiteralPath $script:BackupDir -File -Filter 'settings.*.json' |
        Sort-Object @{ Expression = 'LastWriteTimeUtc'; Descending = $true },
                    @{ Expression = 'Name'; Descending = $true })
}

function Remove-BackupFile([System.IO.FileInfo]$File) {
    $root = [System.IO.Path]::GetFullPath($script:BackupDir).TrimEnd('\') + '\'
    $path = [System.IO.Path]::GetFullPath($File.FullName)
    if (-not $path.StartsWith($root, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to delete a file outside the backup directory: $path"
    }
    Remove-Item -LiteralPath $path -Force
}

function Remove-ExcessBackups {
    $expired = @(Get-BackupFiles | Select-Object -Skip $BackupLimit)
    foreach ($file in $expired) { Remove-BackupFile $file }
}

function New-SettingsBackup {
    if (-not (Test-Path -LiteralPath $script:SettingsPath -PathType Leaf)) { return $null }
    if (-not (Test-Path -LiteralPath $script:BackupDir)) {
        New-Item -ItemType Directory -Path $script:BackupDir | Out-Null
    }
    $stamp = Get-Date -Format 'yyyyMMdd-HHmmss-fff'
    $suffix = [Guid]::NewGuid().ToString('N').Substring(0, 8)
    $target = Join-Path $script:BackupDir "settings.$stamp-$suffix.json"
    Copy-Item -LiteralPath $script:SettingsPath -Destination $target
    Remove-ExcessBackups
    return $target
}

function Resolve-ProviderId([string]$ProviderValue) {
    if ([string]::IsNullOrWhiteSpace($ProviderValue)) { throw 'Provider name is required.' }
    $normalized = $ProviderValue.ToLowerInvariant()
    if ($script:ProviderAliases.Contains($normalized)) {
        $normalized = $script:ProviderAliases[$normalized]
    }
    if (-not $script:Providers.Contains($normalized)) {
        throw "Unknown provider '$ProviderValue'. Available: $($script:Providers.Keys -join ', ')"
    }
    return $normalized
}

function Get-ProviderForKey([string]$ProviderValue) {
    $normalized = ([string]$ProviderValue).ToLowerInvariant()
    if ($normalized -in @('shu', 'company')) { $normalized = 's-openai' }
    $providerId = Resolve-ProviderId $normalized
    $provider = $script:Providers[$providerId]
    if ([string]::IsNullOrWhiteSpace([string]$provider['keyEnv'])) {
        throw 'The official provider does not use a stored third-party API key.'
    }
    return $provider
}

function Get-PersistentKey([string]$KeyName) {
    if ([string]::IsNullOrWhiteSpace($KeyName)) { return $null }
    $value = [Environment]::GetEnvironmentVariable($KeyName, 'Process')
    if ([string]::IsNullOrWhiteSpace($value)) {
        $value = [Environment]::GetEnvironmentVariable($KeyName, 'User')
    }
    if ([string]::IsNullOrWhiteSpace($value)) { return $null }
    return $value.Trim()
}

function Assert-ValidKey([string]$Key) {
    $value = ([string]$Key).Trim()
    if ([string]::IsNullOrWhiteSpace($value)) {
        throw "API key cannot be empty; use 'clear' to remove it."
    }
    if ($value -match '[\x00-\x1f]') {
        throw 'API key contains an invalid control character.'
    }
    return $value
}

function ConvertFrom-SecureValue([Security.SecureString]$SecureValue) {
    $pointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($SecureValue)
    try { return [Runtime.InteropServices.Marshal]::PtrToStringBSTR($pointer) }
    finally {
        if ($pointer -ne [IntPtr]::Zero) {
            [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($pointer)
        }
    }
}

function Send-EnvironmentChange {
    try {
        if (-not ('Claudes.NativeMethods' -as [type])) {
            Add-Type -Namespace Claudes -Name NativeMethods -MemberDefinition @'
[System.Runtime.InteropServices.DllImport("user32.dll", SetLastError = true, CharSet = System.Runtime.InteropServices.CharSet.Auto)]
public static extern IntPtr SendMessageTimeout(
    IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam,
    uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
'@
        }
        $result = [UIntPtr]::Zero
        [void][Claudes.NativeMethods]::SendMessageTimeout(
            [IntPtr]0xffff, [uint32]0x1a, [UIntPtr]::Zero, 'Environment',
            [uint32]2, [uint32]5000, [ref]$result
        )
    }
    catch { }
}

function Build-Settings(
    [System.Collections.IDictionary]$Current,
    [string]$ProviderId,
    [AllowNull()][string]$ApiKey
) {
    $provider = $script:Providers[$ProviderId]
    $result = Copy-PlainObject $Current
    if (-not $result.Contains('env')) {
        $result['env'] = [ordered]@{}
    }
    if ($result['env'] -isnot [System.Collections.IDictionary]) {
        throw "settings.json field 'env' must be an object."
    }
    foreach ($key in $script:ManagedEnvKeys) {
        if ($result['env'].Contains($key)) { $result['env'].Remove($key) }
    }
    if ($ProviderId -ne 'official') {
        if ([string]::IsNullOrWhiteSpace($ApiKey)) {
            throw "$($provider['keyEnv']) is not set. Run: claudes env $ProviderId"
        }
        $result['env']['ANTHROPIC_BASE_URL'] = $provider['baseUrl']
        $result['env']['ANTHROPIC_AUTH_TOKEN'] = $ApiKey
        foreach ($role in $provider['models'].Keys) {
            if ($script:ModelEnvKeys.Contains($role)) {
                $result['env'][$script:ModelEnvKeys[$role]] = $provider['models'][$role]
            }
        }
    }
    return $result
}

function Get-DetectedProvider([System.Collections.IDictionary]$Settings) {
    if (-not $Settings.Contains('env') -or $Settings['env'] -isnot [System.Collections.IDictionary]) {
        return 'official'
    }
    $baseUrl = ([string]$Settings['env']['ANTHROPIC_BASE_URL']).TrimEnd('/')
    if ([string]::IsNullOrWhiteSpace($baseUrl)) { return 'official' }
    $matches = @()
    foreach ($providerId in $script:Providers.Keys) {
        $candidate = ([string]$script:Providers[$providerId]['baseUrl']).TrimEnd('/')
        if (-not [string]::IsNullOrWhiteSpace($candidate) -and $candidate -eq $baseUrl) {
            $matches += $providerId
        }
    }
    if ($matches.Count -eq 1) { return $matches[0] }
    $model = [string]$Settings['env']['ANTHROPIC_MODEL']
    $modelMatches = @(
        $matches | Where-Object { [string]$script:Providers[$_]['models']['default'] -eq $model }
    )
    if ($modelMatches.Count -eq 1) { return $modelMatches[0] }
    return 'unknown'
}

function Use-Provider([string]$ProviderValue) {
    $providerId = Resolve-ProviderId $ProviderValue
    $provider = $script:Providers[$providerId]
    $apiKey = $null
    if (-not [string]::IsNullOrWhiteSpace([string]$provider['keyEnv'])) {
        $apiKey = Get-PersistentKey $provider['keyEnv']
    }
    if ($DryRun) {
        $current = Read-JsonObject $script:SettingsPath -MissingOk
        $updated = Build-Settings $current $providerId $apiKey
        if ((ConvertTo-StableJson $current -Compress) -eq (ConvertTo-StableJson $updated -Compress)) {
            Write-Info "Already active: $providerId"
            return
        }
        Write-Info "[DryRun] Would switch to: $providerId"
        Write-Info "[DryRun] Would update: $($script:SettingsPath)"
        return
    }
    $changed = Invoke-WithConfigLock {
        $current = Read-JsonObject $script:SettingsPath -MissingOk
        $updated = Build-Settings $current $providerId $apiKey
        if ((ConvertTo-StableJson $current -Compress) -eq (ConvertTo-StableJson $updated -Compress)) {
            Write-Info "Already active: $providerId"
            return $false
        }
        $backup = New-SettingsBackup
        Write-JsonAtomic $script:SettingsPath $updated
        if ($backup) { Write-Info "Backup: $backup" }
        return $true
    }
    if (-not $changed) { return }
    Write-Ok "Active provider: $providerId"
    Write-Warn 'Fully exit and restart Claude Code for the change to take effect.'
}

function Clear-ActiveAuth([string]$KeyName) {
    if (-not (Test-Path -LiteralPath $script:SettingsPath -PathType Leaf)) { return }
    $current = Read-JsonObject $script:SettingsPath
    $active = Get-DetectedProvider $current
    if (-not $script:Providers.Contains($active)) { return }
    $provider = $script:Providers[$active]
    if ([string]$provider['keyEnv'] -ne $KeyName) { return }
    if (-not $current.Contains('env') -or $current['env'] -isnot [System.Collections.IDictionary]) { return }
    if (
        -not $current['env'].Contains('ANTHROPIC_AUTH_TOKEN') -and
        -not $current['env'].Contains('ANTHROPIC_API_KEY')
    ) { return }
    if ($DryRun) {
        Write-Info "[DryRun] Would remove the active $KeyName credential from $($script:SettingsPath)."
        return
    }
    $changed = Invoke-WithConfigLock {
        $currentLocked = Read-JsonObject $script:SettingsPath
        $activeLocked = Get-DetectedProvider $currentLocked
        if (-not $script:Providers.Contains($activeLocked)) { return $false }
        if ([string]$script:Providers[$activeLocked]['keyEnv'] -ne $KeyName) { return $false }
        $updated = Copy-PlainObject $currentLocked
        $updated['env'].Remove('ANTHROPIC_AUTH_TOKEN')
        $updated['env'].Remove('ANTHROPIC_API_KEY')
        $backup = New-SettingsBackup
        Write-JsonAtomic $script:SettingsPath $updated
        if ($backup) { Write-Info "Backup: $backup" }
        return $true
    }
    if (-not $changed) { return }
    Write-Info "Removed active credential from: $($script:SettingsPath)"
}

function Set-ApiKey([string]$ProviderValue, [AllowNull()][string]$KeyValue) {
    $provider = Get-ProviderForKey $ProviderValue
    $keyName = [string]$provider['keyEnv']
    if ([string]$KeyValue -eq 'clear') {
        if ($DryRun) {
            Write-Info "[DryRun] Would clear: $keyName"
        }
        else {
            [Environment]::SetEnvironmentVariable($keyName, $null, 'User')
            [Environment]::SetEnvironmentVariable($keyName, $null, 'Process')
            Send-EnvironmentChange
            Write-Ok "Cleared: $keyName"
        }
        Clear-ActiveAuth $keyName
        return
    }
    if ([string]::IsNullOrWhiteSpace($KeyValue)) {
        $secure = Read-Host "$keyName (input hidden)" -AsSecureString
        $KeyValue = ConvertFrom-SecureValue $secure
    }
    else {
        Write-Warn 'The API key may be present in PowerShell command history.'
    }
    $key = Assert-ValidKey $KeyValue
    if ($DryRun) {
        Write-Info "[DryRun] Would set: $keyName"
        return
    }
    [Environment]::SetEnvironmentVariable($keyName, $key, 'User')
    [Environment]::SetEnvironmentVariable($keyName, $key, 'Process')
    Send-EnvironmentChange
    Write-Ok "Set: $keyName"
    Write-Warn 'Open a new terminal before starting Claude Code.'
}

function Show-ProviderList {
    $settings = Read-JsonObject $script:SettingsPath -MissingOk
    $active = if (Test-Path -LiteralPath $script:SettingsPath -PathType Leaf) {
        Get-DetectedProvider $settings
    }
    else {
        'missing settings'
    }
    foreach ($providerId in $script:Providers.Keys) {
        $provider = $script:Providers[$providerId]
        $marker = if ($providerId -eq $active) { '*' } else { ' ' }
        $model = if ($provider['models'].Contains('default')) {
            [string]$provider['models']['default']
        }
        else {
            'Claude default'
        }
        $keyState = if ([string]::IsNullOrWhiteSpace([string]$provider['keyEnv'])) {
            'not required'
        }
        elseif (Get-PersistentKey $provider['keyEnv']) {
            'set'
        }
        else {
            'missing'
        }
        Write-Host ("{0} {1,-14} {2,-26} {3} [{4}]" -f $marker, $providerId, $model, $provider['name'], $keyState)
    }
}

function Show-Status {
    $exists = Test-Path -LiteralPath $script:SettingsPath -PathType Leaf
    $settings = Read-JsonObject $script:SettingsPath -MissingOk
    $active = if ($exists) { Get-DetectedProvider $settings } else { 'missing settings' }
    $envSettings = if ($settings.Contains('env') -and $settings['env'] -is [System.Collections.IDictionary]) {
        $settings['env']
    }
    else {
        [ordered]@{}
    }
    $provider = if ($script:Providers.Contains($active)) { $script:Providers[$active] } else { $null }
    $keyState = if ($null -eq $provider -or [string]::IsNullOrWhiteSpace([string]$provider['keyEnv'])) {
        'not required'
    }
    elseif (Get-PersistentKey $provider['keyEnv']) {
        'set'
    }
    else {
        'missing'
    }
    $defaultModel = if ($envSettings.Contains('ANTHROPIC_MODEL')) {
        $envSettings['ANTHROPIC_MODEL']
    }
    else {
        'Claude default'
    }
    $baseUrl = if ($envSettings.Contains('ANTHROPIC_BASE_URL')) {
        $envSettings['ANTHROPIC_BASE_URL']
    }
    else {
        'Anthropic official'
    }
    Write-Host '===== Claude Code provider status ====='
    Write-Host "Settings file : $($script:SettingsPath)"
    Write-Host "Provider      : $active"
    Write-Host "Default model : $defaultModel"
    Write-Host "Base URL      : $baseUrl"
    Write-Host "Persistent key: $keyState"
    Write-Host "Auth in config: $(if ($envSettings.Contains('ANTHROPIC_AUTH_TOKEN')) { 'yes' } else { 'no' })"
    Write-Host "Backups       : $((Get-BackupFiles).Count) / $BackupLimit retained"
}

function Migrate-LegacyKeys {
    $legacy = Read-JsonObject $script:LegacyModelsPath
    if (-not $legacy.Contains('providers') -or $legacy['providers'] -isnot [System.Collections.IDictionary]) {
        throw "Legacy models.json field 'providers' must be an object."
    }
    $pending = [ordered]@{}
    $skipped = @()
    foreach ($keyName in $script:LegacyKeyProviders.Keys) {
        if (Get-PersistentKey $keyName) {
            $skipped += $keyName
            continue
        }
        $values = @()
        foreach ($providerId in $script:LegacyKeyProviders[$keyName]) {
            if (
                $legacy['providers'].Contains($providerId) -and
                $legacy['providers'][$providerId] -is [System.Collections.IDictionary] -and
                -not [string]::IsNullOrWhiteSpace([string]$legacy['providers'][$providerId]['apiKey'])
            ) {
                $candidate = ([string]$legacy['providers'][$providerId]['apiKey']).Trim()
                if ($candidate -notin $values) { $values += $candidate }
            }
        }
        if ($values.Count -gt 1) {
            throw "Legacy providers contain conflicting values for $keyName"
        }
        if ($values.Count -eq 1) {
            $pending[$keyName] = Assert-ValidKey $values[0]
        }
    }
    if (-not $DryRun) {
        foreach ($keyName in $pending.Keys) {
            [Environment]::SetEnvironmentVariable($keyName, $pending[$keyName], 'User')
            [Environment]::SetEnvironmentVariable($keyName, $pending[$keyName], 'Process')
        }
        if ($pending.Count -gt 0) { Send-EnvironmentChange }
    }
    $prefix = if ($DryRun) { '[DryRun] Would migrate' } else { 'Migrated' }
    $names = if ($pending.Count -gt 0) { $pending.Keys -join ', ' } else { 'no keys' }
    Write-Info "${prefix}: $names"
    if ($skipped.Count -gt 0) { Write-Info "Already configured: $($skipped -join ', ')" }
    Write-Info "Legacy file preserved: $($script:LegacyModelsPath)"
}

function Manage-Backups([string]$Action) {
    $normalized = ([string]$Action).ToLowerInvariant()
    $files = @(Get-BackupFiles)
    switch ($normalized) {
        'list' {
            if ($files.Count -eq 0) { Write-Host 'No backups.' }
            foreach ($file in $files) { Write-Host $file.FullName }
        }
        'clear' {
            if ($DryRun) {
                Write-Info "[DryRun] Would remove $($files.Count) backup(s)."
                return
            }
            Invoke-WithConfigLock {
                foreach ($file in @(Get-BackupFiles)) { Remove-BackupFile $file }
                if (
                    (Test-Path -LiteralPath $script:BackupDir -PathType Container) -and
                    @(Get-ChildItem -LiteralPath $script:BackupDir -Force).Count -eq 0
                ) {
                    Remove-Item -LiteralPath $script:BackupDir
                }
            }
            Write-Ok "Removed $($files.Count) backup(s)."
        }
        'restore' {
            if ($files.Count -eq 0) { throw 'No configuration backup is available to restore.' }
            $source = $files[0]
            $restored = Read-JsonObject $source.FullName
            if ($DryRun) {
                Write-Info "[DryRun] Would restore: $($source.FullName)"
                return
            }
            Invoke-WithConfigLock {
                $backup = New-SettingsBackup
                Write-JsonAtomic $script:SettingsPath $restored
                if ($backup) { Write-Info "Backup: $backup" }
            }
            Write-Ok "Restored: $($source.FullName)"
        }
        default { throw 'Usage: claudes backup list|restore|clear' }
    }
}

function Get-ScriptSourcePath {
    if (-not [string]::IsNullOrWhiteSpace($PSCommandPath)) { return $PSCommandPath }
    if (-not [string]::IsNullOrWhiteSpace($MyInvocation.PSCommandPath)) {
        return $MyInvocation.PSCommandPath
    }
    return $null
}

function Test-PathListContains([string]$PathList, [string]$TargetDir) {
    if ([string]::IsNullOrWhiteSpace($PathList)) { return $false }
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

function Ensure-ClaudeHomeOnUserPath {
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (-not (Test-PathListContains $userPath $ClaudeHome)) {
        $newPath = if ([string]::IsNullOrWhiteSpace($userPath)) {
            $ClaudeHome
        }
        else {
            "$userPath;$ClaudeHome"
        }
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        Write-Info "Added to user PATH: $ClaudeHome"
    }
    if (-not (Test-PathListContains $env:Path $ClaudeHome)) {
        $env:Path = "$ClaudeHome;$env:Path"
    }
    Send-EnvironmentChange
}

function Install-StatusLine([string]$SourceScript) {
    $sourceDir = Split-Path -Parent $SourceScript
    $sourceStatusLine = Join-Path $sourceDir 'statusline.ps1'
    if (-not (Test-Path -LiteralPath $sourceStatusLine -PathType Leaf)) {
        throw "Missing Windows status line script: $sourceStatusLine"
    }
    $targetStatusLine = Join-Path $ClaudeHome 'statusline.ps1'
    if ([System.IO.Path]::GetFullPath($sourceStatusLine) -ne [System.IO.Path]::GetFullPath($targetStatusLine)) {
        Copy-Item -LiteralPath $sourceStatusLine -Destination $targetStatusLine -Force
    }
    try { Unblock-File -LiteralPath $targetStatusLine -ErrorAction SilentlyContinue }
    catch { }

    $commandPath = $targetStatusLine.Replace('\', '/')
    $statusCommand = 'powershell.exe -NoProfile -ExecutionPolicy Bypass -File "' + $commandPath + '"'
    Invoke-WithConfigLock {
        $current = Read-JsonObject $script:SettingsPath -MissingOk
        $updated = Copy-PlainObject $current
        $updated['statusLine'] = [ordered]@{
            'type' = 'command'
            'command' = $statusCommand
        }
        if ((ConvertTo-StableJson $current -Compress) -eq (ConvertTo-StableJson $updated -Compress)) {
            return
        }
        $backup = New-SettingsBackup
        Write-JsonAtomic $script:SettingsPath $updated
        if ($backup) { Write-Info "Backup: $backup" }
    }
    Write-Ok "Status line: $targetStatusLine"
}

function Install-Claudes {
    Ensure-ClaudeHome
    $source = Get-ScriptSourcePath
    if ([string]::IsNullOrWhiteSpace($source) -or -not (Test-Path -LiteralPath $source -PathType Leaf)) {
        throw 'Cannot determine the current script path.'
    }
    $target = Join-Path $ClaudeHome 'claudes.ps1'
    $wrapper = Join-Path $ClaudeHome 'claudes.cmd'
    if ([System.IO.Path]::GetFullPath($source) -ne [System.IO.Path]::GetFullPath($target)) {
        Copy-Item -LiteralPath $source -Destination $target -Force
    }
    try { Unblock-File -LiteralPath $target -ErrorAction SilentlyContinue }
    catch { }
    $wrapperText = @(
        '@echo off',
        'powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0claudes.ps1" %*',
        ''
    ) -join "`r`n"
    [System.IO.File]::WriteAllText($wrapper, $wrapperText, $script:Utf8NoBom)
    Install-StatusLine $source
    Ensure-ClaudeHomeOnUserPath
    Write-Ok "Installed: $target"
    Write-Ok "Command wrapper: $wrapper"
    if (Test-Path -LiteralPath $script:LegacyModelsPath -PathType Leaf) {
        Write-Warn 'Legacy keys found. Open a new terminal, then run: claudes migrate'
    }
    Write-Warn 'Open a new terminal, then run: claudes help'
}

function Show-Help {
    @"
claudes v$($script:ToolVersion) - Windows Claude Code provider switcher

Commands:
  claudes list
  claudes status
  claudes use <provider>
  claudes env <provider|shu> [clear]
  claudes migrate
  claudes backup list|restore|clear
  claudes install
  claudes help

Providers:
  official, deepseek, kimi, minimax, s-openai, s-deepseek, s-grok

API key environment variables:
  DEEPSEEK_API_KEY, KIMI_API_KEY, MINIMAX_API_KEY
  SHU_API_KEY, SHU_DEEPSEEK_API_KEY, SHU_GROK_API_KEY

Options:
  -ClaudeHome <path>  -BackupLimit <count>  -DryRun

Direct use before installation:
  powershell -ExecutionPolicy Bypass -File .\claudes.ps1 list

Only provider-owned env fields in settings.json are changed. Existing settings
are preserved and every real content change creates a backup.
"@ | Write-Host
}

try {
    switch ($Command.ToLowerInvariant()) {
        'status' { Show-Status }
        'list' { Show-ProviderList }
        'use' { Use-Provider $Value }
        'env' { Set-ApiKey $Value $ExtraValue }
        'key' { Set-ApiKey $Value $ExtraValue }
        'migrate' { Migrate-LegacyKeys }
        'backup' { Manage-Backups $Value }
        'install' { Install-Claudes }
        'help' { Show-Help }
    }
}
catch {
    Write-Host "error: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
