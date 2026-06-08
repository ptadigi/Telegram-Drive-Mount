# start-client.ps1 - Khoi dong Telegram Drive thin-client (may con)
# Mount o ao tu VPS qua token da pair. Khong giu Telegram session / DB.
#
# Lan dau phai pair: .\start-client.ps1 -Pair -PairUrl https://drive.example.com -PairCode 4F2A-9K2X
# Cac lan sau:        .\start-client.ps1 -MountPoint T:

param(
    [switch]$Pair,
    [string]$PairUrl,
    [string]$PairCode,
    [string]$PairName,
    [string]$MountPoint = "T:",
    [string]$RemoteUrl,
    [switch]$Insecure
)

$ErrorActionPreference = "Stop"
$exe = Join-Path $PSScriptRoot "td-agent.exe"
if (-not (Test-Path $exe)) {
    Write-Error "Khong tim thay td-agent.exe."
    exit 1
}
if ($Insecure) { $env:TD_AGENT_INSECURE = "1" }

if ($Pair) {
    $args = @("--pair")
    if ($PairUrl)  { $args += @("--pair-url", $PairUrl) }
    if ($PairCode) { $args += @("--pair-code", $PairCode) }
    if ($PairName) { $args += @("--pair-name", $PairName) }
    Write-Host "Ghep thiet bi voi VPS..." -ForegroundColor Green
    & $exe @args
    return
}

$args = @("--remote", "--remote-mount", $MountPoint)
if ($RemoteUrl) { $args += @("--remote-url", $RemoteUrl) }
Write-Host "Mount o ao $MountPoint tu VPS..." -ForegroundColor Green
& $exe @args