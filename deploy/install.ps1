# =============================================================================
#  Ổ Đĩa Cloud Ảo (Telegram Drive) — One-click installer for Windows
#
#  Run in PowerShell (as Administrator recommended):
#
#    irm https://raw.githubusercontent.com/ptadigi/Telegram-Drive-Mount/main/deploy/install.ps1 | iex
#
#  Downloads the latest TelegramDriveSetup.exe, verifies its SHA256 against the
#  release SHA256SUMS.txt, then runs the installer (bundles WinFsp + PWA + tray).
# =============================================================================
$ErrorActionPreference = "Stop"
$Repo = "ptadigi/Telegram-Drive-Mount"
$Base = "https://github.com/$Repo/releases/latest/download"
$Tmp  = Join-Path $env:TEMP "telegram-drive-setup"
New-Item -ItemType Directory -Force -Path $Tmp | Out-Null

function Info($m) { Write-Host $m -ForegroundColor Cyan }
function Ok($m)   { Write-Host $m -ForegroundColor Green }
function Warn($m) { Write-Host $m -ForegroundColor Yellow }

Info "==> Tải TelegramDriveSetup.exe ..."
$setup = Join-Path $Tmp "TelegramDriveSetup.exe"
Invoke-WebRequest -Uri "$Base/TelegramDriveSetup.exe" -OutFile $setup -UseBasicParsing

# Verify checksum (best-effort; warns instead of failing if sums are missing).
try {
  Info "==> Xác minh SHA256 ..."
  $sumsPath = Join-Path $Tmp "SHA256SUMS.txt"
  Invoke-WebRequest -Uri "$Base/SHA256SUMS.txt" -OutFile $sumsPath -UseBasicParsing
  $expected = (Select-String -Path $sumsPath -Pattern "TelegramDriveSetup.exe").Line.Split(" ")[0].ToLower()
  $actual = (Get-FileHash $setup -Algorithm SHA256).Hash.ToLower()
  if ($expected -and $expected -ne $actual) {
    throw "Checksum KHÔNG khớp! expected=$expected actual=$actual"
  }
  Ok "   Checksum hợp lệ."
} catch {
  Warn "   Bỏ qua xác minh checksum: $($_.Exception.Message)"
}

Info "==> Chạy trình cài đặt ..."
# /SILENT keeps a progress bar; remove it to show the full wizard.
Start-Process -FilePath $setup -ArgumentList "/SILENT","/NORESTART" -Verb RunAs -Wait

Ok ""
Ok "Hoàn tất! Ứng dụng 'Ổ Đĩa Cloud Ảo' đã được cài và sẽ chạy ở khay hệ thống (tray)."
Write-Host "  • Mở giao diện: bấm chuột phải icon ở tray -> 'Mở giao diện'."
Write-Host "  • Ổ ảo sẽ mount tại ổ T: (Telegram Drive)."
Write-Host "  • Lần đầu: chọn 'Chạy máy chủ local' hoặc 'Nối máy chủ có sẵn' (nhập URL VPS)."
