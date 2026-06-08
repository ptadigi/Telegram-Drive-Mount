# start-server.ps1 - Khoi dong Telegram Drive server (PC chinh / VPS Windows)
# Tu mount o ao T: + giu session Telegram ma hoa.
#
# Lan dau: tu sinh session key va luu vao file .session-key (gitignored).
# Cac lan sau: doc lai key do, dam bao session Telegram khong bi mat.

param(
    [string]$MountPoint = "T:",
    [string]$ConfigPath = "$PSScriptRoot\agent-go\config.local.json",
    [switch]$NoMount,
    [switch]$NoTray
)

$ErrorActionPreference = "Stop"
$exe = Join-Path $PSScriptRoot "td-agent.exe"
if (-not (Test-Path $exe)) {
    Write-Error "Khong tim thay td-agent.exe. Build truoc: cd agent-go; go build -tags 'fuse tray' -o ..\td-agent.exe .\cmd\agent"
    exit 1
}

# Session key: doc tu env, hoac file, hoac sinh moi
$keyFile = Join-Path $PSScriptRoot ".session-key"
if ($env:TD_AGENT_SESSION_KEY) {
    $key = $env:TD_AGENT_SESSION_KEY
} elseif (Test-Path $keyFile) {
    $key = (Get-Content $keyFile -Raw).Trim()
} else {
    $bytes = New-Object byte[] 32
    [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
    $key = -join ($bytes | ForEach-Object { $_.ToString('x2') })
    Set-Content -Path $keyFile -Value $key -NoNewline
    Write-Host "Da sinh session key moi, luu tai $keyFile (giu file nay an toan)" -ForegroundColor Yellow
}
$env:TD_AGENT_SESSION_KEY = $key

$args = @("--config", $ConfigPath)
if (-not $NoTray)  { $args += "--tray" }
if (-not $NoMount) { $args += @("--mount-on-start", "--mount-point", $MountPoint) }

Write-Host "Khoi dong Telegram Drive server..." -ForegroundColor Green
Write-Host "  Mount: $(if ($NoMount) { 'tat' } else { $MountPoint })"
Write-Host "  Tray:  $(if ($NoTray) { 'tat' } else { 'bat' })"
Write-Host "  PWA:   mo http://localhost:5173 (chay rieng: cd web-pwa; npm run dev)"
& $exe @args