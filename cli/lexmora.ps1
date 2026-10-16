# lexmora - Docker lifecycle CLI for lexmora-api and lexmora-webui.
# Usage: lexmora <command> [options]   (run `lexmora help`)
param([Parameter(ValueFromRemainingArguments = $true)] [string[]] $Arguments)

$ErrorActionPreference = 'Stop'
$CliVersion = '1.0.0'
$Project = 'lexmora'
$Network = 't3-net'
$CliDir = $PSScriptRoot
$RepoRoot = Split-Path -Parent $CliDir
$StateDir = Join-Path $env:LOCALAPPDATA 'lexmora'
$ConfigPath = Join-Path $StateDir 'config.json'

$Apps = [ordered]@{
    api = @{ Name = 'lexmora-api'; Dir = (Join-Path $RepoRoot 'lexmora-api'); Image = 'lexmora-api:latest'; PortVar = 'LEXMORA_API_PORT'; DefaultPort = 8083 }
    web = @{ Name = 'lexmora-webui'; Dir = (Join-Path $RepoRoot 'lexmora-webui'); Image = 'lexmora-webui:latest'; PortVar = 'LEXMORA_WEB_PORT'; DefaultPort = 8090 }
}

function Show-Usage {
    @"
lexmora $CliVersion - Docker lifecycle for lexmora-api and lexmora-webui

Usage: lexmora <command> [options]

  status                          Container state and URLs. Exit 0 up, 3 stopped, 4 not installed
  version                         CLI, repo, and Docker versions
  install                         Add the lexmora CLI folder to the user PATH
  run                             Start both apps and wait for their URLs
  stop                            Stop both apps (containers kept)
  port --app=api|web --port=N     Save host port and restart if running (no --port prints ports)
  update                          git pull --ff-only in both repos, then rebuild keeping data
  rebuild [--remove-data=no|yes] [--yes]
                                  Rebuild images. 'yes' also deletes DB/uploads volumes and config
  uninstall [--yes]              Remove containers, volumes, images, config, and PATH entry
  help                            Show this help
"@
}

function Read-Options([string[]] $Items) {
    $opts = @{}
    for ($i = 0; $i -lt $Items.Count; $i++) {
        $item = $Items[$i]
        if ($item -match '^--([^=]+)=(.*)$') {
            $opts[$matches[1]] = $matches[2]
        }
        elseif ($item -match '^--(.+)$') {
            $name = $matches[1]
            if (($i + 1) -lt $Items.Count -and $Items[$i + 1] -notlike '--*') { $i++; $opts[$name] = $Items[$i] }
            else { $opts[$name] = $true }
        }
        else {
            Show-Usage
            Write-Host "Unknown argument: $item" -ForegroundColor Red
            exit 2
        }
    }
    return $opts
}

function Get-Config {
    $cfg = @{ ports = @{} }
    foreach ($key in $Apps.Keys) { $cfg.ports[$key] = $Apps[$key].DefaultPort }
    if (Test-Path $ConfigPath) {
        $saved = Get-Content $ConfigPath -Raw | ConvertFrom-Json
        foreach ($key in $Apps.Keys) {
            if ($saved.ports.$key) { $cfg.ports[$key] = [int]$saved.ports.$key }
        }
    }
    return $cfg
}

function Save-Config($Cfg) {
    New-Item -ItemType Directory -Force -Path $StateDir | Out-Null
    [pscustomobject]@{ version = $CliVersion; ports = $Cfg.ports } | ConvertTo-Json | Set-Content -Path $ConfigPath -Encoding UTF8
}

function Set-PortEnv {
    $cfg = Get-Config
    foreach ($key in $Apps.Keys) {
        [Environment]::SetEnvironmentVariable($Apps[$key].PortVar, [string]$cfg.ports[$key], 'Process')
    }
}

# Combined file set keeps one compose project; build uses per-app Keys so build context stays correct.
function Invoke-Compose {
    param([string[]] $Keys = @('api', 'web'), [Parameter(Mandatory)] [string[]] $ComposeArgs)
    Set-PortEnv
    $files = @()
    foreach ($key in $Keys) {
        $files += '-f', (Join-Path $Apps[$key].Dir 'docker-compose.yml'), '-f', (Join-Path $Apps[$key].Dir 'docker-compose.publish.yml')
    }
    & docker compose -p $Project @files @ComposeArgs
    if ($LASTEXITCODE -ne 0) { throw "docker compose $($ComposeArgs -join ' ') failed (exit $LASTEXITCODE)" }
}

function Get-ProjectContainers {
    return @(docker ps --filter "label=com.docker.compose.project=$Project" --format "{{.Names}}|{{.Status}}|{{.Ports}}")
}

function Test-Url([string] $Url) {
    try { Invoke-WebRequest -UseBasicParsing -Uri $Url -TimeoutSec 3 | Out-Null; return $true }
    catch { return [bool]$_.Exception.Response }
}

function Wait-Url([string] $Url, [int] $Seconds = 30) {
    $deadline = (Get-Date).AddSeconds($Seconds)
    while ((Get-Date) -lt $deadline) {
        if (Test-Url $Url) { return $true }
        Start-Sleep -Seconds 1
    }
    return $false
}

function Test-PortFree([int] $Port) {
    $listener = New-Object System.Net.Sockets.TcpListener ([System.Net.IPAddress]::Loopback, $Port)
    try { $listener.Start(); return $true }
    catch { return $false }
    finally { $listener.Stop() }
}

function Assert-Installed {
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { throw 'Docker not found on PATH.' }
    foreach ($key in $Apps.Keys) {
        foreach ($file in 'docker-compose.yml', 'docker-compose.publish.yml') {
            if (-not (Test-Path (Join-Path $Apps[$key].Dir $file))) { throw "Missing $file in $($Apps[$key].Dir)" }
        }
    }
}

function Assert-Images {
    foreach ($key in $Apps.Keys) {
        if (-not (docker image ls -q $Apps[$key].Image)) { throw "Image $($Apps[$key].Image) missing. Run: lexmora rebuild" }
    }
}

function Ensure-Network {
    if (-not (docker network ls --filter "name=^$Network$" --format "{{.Name}}")) { docker network create $Network | Out-Null }
}

function Wait-AppUrls {
    $cfg = Get-Config
    foreach ($key in $Apps.Keys) {
        $url = "http://localhost:$($cfg.ports[$key])/"
        if (Wait-Url $url 30) { Write-Host "$($Apps[$key].Name) is up: $url" }
        else { throw "$($Apps[$key].Name) did not respond at $url within 30s" }
    }
}

function Invoke-Version {
    Write-Host "lexmora CLI: $CliVersion ($CliDir)"
    foreach ($key in $Apps.Keys) {
        $desc = git -C $Apps[$key].Dir describe --tags --always --dirty 2>$null
        if (-not $desc) { $desc = 'unknown' }
        Write-Host ("{0,-14} {1}" -f $Apps[$key].Name, $desc)
    }
    Write-Host ("docker: " + (docker version --format '{{.Server.Version}}'))
}

function Invoke-Status {
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { Write-Host 'Docker not found.'; exit 4 }
    foreach ($key in $Apps.Keys) {
        if (-not (Test-Path (Join-Path $Apps[$key].Dir 'docker-compose.yml'))) { Write-Host "Not installed: compose file missing for $key."; exit 4 }
    }
    $cfg = Get-Config
    $containers = Get-ProjectContainers
    Write-Host "Mode: docker   Project: $Project   Config: $ConfigPath"
    if ($containers.Count -eq 0) { Write-Host 'Containers: none' }
    else { Write-Host 'Containers:'; $containers | ForEach-Object { Write-Host "  $_" } }
    $allUp = $true
    foreach ($key in $Apps.Keys) {
        $url = "http://localhost:$($cfg.ports[$key])/"
        $up = Test-Url $url
        if (-not $up) { $allUp = $false }
        Write-Host ("{0,-6} {1,-14} {2}  {3}" -f $key, $Apps[$key].Name, $url, $(if ($up) { 'OK' } else { 'DOWN' }))
    }
    if ($allUp) { exit 0 }
    exit 3
}

function Invoke-Install {
    $entries = @([Environment]::GetEnvironmentVariable('Path', 'User') -split ';' | Where-Object { $_ })
    if ($entries -contains $CliDir) { Write-Host "Already on user PATH: $CliDir"; return }
    [Environment]::SetEnvironmentVariable('Path', (($entries + $CliDir) -join ';'), 'User')
    Write-Host "Added to user PATH: $CliDir (open a new shell)"
}

function Invoke-Run {
    Assert-Installed
    Assert-Images
    Ensure-Network
    Invoke-Compose -ComposeArgs @('up', '-d', '--no-build')
    Wait-AppUrls
}

function Invoke-Port($Opts) {
    $cfg = Get-Config
    if (-not $Opts.ContainsKey('port')) {
        foreach ($key in $Apps.Keys) { Write-Host ("{0,-6} port {1}" -f $key, $cfg.ports[$key]) }
        return
    }
    $key = [string]$Opts['app']
    if ($key -notin @('api', 'web')) { throw '--app must be api or web' }
    $port = 0
    if (-not [int]::TryParse([string]$Opts['port'], [ref]$port) -or $port -lt 1 -or $port -gt 65535) { throw '--port must be 1-65535' }
    if ($port -ne $cfg.ports[$key] -and -not (Test-PortFree $port)) { throw "port $port is already in use" }
    $cfg.ports[$key] = $port
    Save-Config $cfg
    Write-Host "$($Apps[$key].Name) port saved: $port"
    if (@(Get-ProjectContainers).Count -gt 0) {
        Invoke-Compose -ComposeArgs @('up', '-d', '--no-build')
        Wait-AppUrls
    }
}

function Build-Images {
    Write-Host 'Building lexmora-webui image...'
    Invoke-Compose -Keys @('web') -ComposeArgs @('build', '--no-cache', '--pull')
    Write-Host 'Building lexmora-api image...'
    $image = $Apps.api.Image
    Push-Location $Apps.api.Dir
    try {
        & docker build --no-cache --pull -f dockerfile -t $image .
        if ($LASTEXITCODE -ne 0) { throw 'docker build failed for lexmora-api' }
    }
    finally { Pop-Location }
}

function Invoke-Rebuild([bool] $RemoveData, [bool] $AssumeYes) {
    Assert-Installed
    Ensure-Network
    $wasRunning = @(Get-ProjectContainers).Count -gt 0
    if ($RemoveData) {
        Write-Host "remove-data=yes deletes: containers, volumes lexmora-api-vol and lexmora-webui-vol, local images, and $StateDir"
        if (-not $AssumeYes -and (Read-Host 'Type yes to continue') -ne 'yes') { throw 'Aborted.' }
        Invoke-Compose -ComposeArgs @('down', '-v', '--rmi', 'local', '--remove-orphans')
        Remove-Item $StateDir -Recurse -Force -ErrorAction SilentlyContinue
    }
    # Build before stopping, so a failed build leaves the running stack untouched.
    Build-Images
    if (-not $RemoveData) { Invoke-Compose -ComposeArgs @('down', '--remove-orphans') }
    if ($wasRunning) {
        Invoke-Compose -ComposeArgs @('up', '-d', '--no-build')
        Wait-AppUrls
    }
    else { Write-Host 'Rebuilt. Stack was stopped, so it stays stopped. Use: lexmora run' }
}

function Invoke-Update {
    Assert-Installed
    foreach ($key in $Apps.Keys) {
        Write-Host "git pull --ff-only in $($Apps[$key].Dir)"
        & git -C $Apps[$key].Dir pull --ff-only
        if ($LASTEXITCODE -ne 0) { throw "git pull failed in $($Apps[$key].Dir). Aborted, nothing rebuilt." }
    }
    Invoke-Rebuild -RemoveData $false -AssumeYes $true
}

function Invoke-Uninstall([bool] $AssumeYes) {
    Write-Host "uninstall deletes: containers, volumes lexmora-api-vol and lexmora-webui-vol, local images, $StateDir, and the PATH entry. Source repos are not touched."
    if (-not $AssumeYes -and (Read-Host 'Type yes to continue') -ne 'yes') { throw 'Aborted.' }
    if (Get-Command docker -ErrorAction SilentlyContinue) {
        Invoke-Compose -ComposeArgs @('down', '-v', '--rmi', 'local', '--remove-orphans')
    }
    Remove-Item $StateDir -Recurse -Force -ErrorAction SilentlyContinue
    $entries = @([Environment]::GetEnvironmentVariable('Path', 'User') -split ';' | Where-Object { $_ -and $_ -ne $CliDir })
    [Environment]::SetEnvironmentVariable('Path', ($entries -join ';'), 'User')
    Write-Host 'Uninstalled.'
}

function Invoke-Main {
    if (-not $Arguments) { $Arguments = @('help') }
    $command = $Arguments[0]
    $opts = Read-Options @($Arguments | Select-Object -Skip 1)
    switch ($command) {
        'help'      { Show-Usage }
        'version'   { Invoke-Version }
        'install'   { Invoke-Install }
        'status'    { Invoke-Status }
        'run'       { Invoke-Run }
        'stop'      { Invoke-Compose -ComposeArgs @('stop'); Write-Host 'Stopped.' }
        'port'      { Invoke-Port $opts }
        'update'    { Invoke-Update }
        'rebuild' {
            $mode = 'no'
            if ($opts.ContainsKey('remove-data')) { $mode = [string]$opts['remove-data'] }
            if ($mode -notin @('no', 'yes')) { throw '--remove-data must be no or yes' }
            Invoke-Rebuild -RemoveData ($mode -eq 'yes') -AssumeYes $opts.ContainsKey('yes')
        }
        'uninstall' { Invoke-Uninstall $opts.ContainsKey('yes') }
        default     { Show-Usage; exit 2 }
    }
}

try {
    Invoke-Main
    exit 0
}
catch {
    Write-Host "lexmora: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
