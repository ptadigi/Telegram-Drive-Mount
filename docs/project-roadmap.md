# Project Roadmap — Telegram-Drive

Cập nhật: 2026-06-11.

## Trạng thái

- M1 Sync watcher robust + Telegram stream: **xong**.
- M1.5 Storage channel auto: **xong**.
- M2 Tray + autostart: **xong**.
- M3 Native mount ổ ảo (WinFsp/FUSE): **xong - read-mostly**.
- Desktop client + phát hành (đợt 1.7.x):
  - Onboarding native WebView2: chọn local/remote, nhập URL + mã pairing, không cần CLI.
  - Installer `TelegramDriveSetup.exe`: bundle WinFsp, tự dò pwa_dir, windowsgui, tray .ico, tiếng Việt có dấu, branding Innonet Agency.
  - Remote mount: mount backend máy chủ đã pair (fix "not accessible").
  - Chunked upload resumable (tus) cho file lớn >32MB, vượt giới hạn proxy.
  - Upload pool 6 worker, UI throttle, progress tổng + ETA, retry, chống thoát.
  - Realtime stale-while-revalidate (fix SSE bị proxy buffer).
  - Debug sync API + UI + sync.log.
  - Docs: TROUBLESHOOTING, DEPLOY (proxy/cache), CODE_SIGNING.

## Đang còn

- Native mount ghi qua ổ ảo (write-back) — hiện đọc + restore từ Telegram.
- Sync hai chiều + conflict resolver.
- Chunked upload: dọn rác chunk tus dở dang (job TTL); test thật file vài trăm MB.
- Remote mount: khởi tạo lại backend khi đổi cấu hình mà không cần thoát app.
- Code-signing thật (SignPath OSS / Certum) để gỡ cảnh báo SmartScreen.
- Office viewer (docx/xlsx/pptx) khi có domain public.
- Installer macOS/Linux.
- Nâng GitHub Actions khỏi Node 20 (sắp deprecate).

## Lệnh build production

```bash
# Default (không cần CGO/WinFsp)
cd agent-go
go build -o td-agent ./cmd/agent

# With native mount
cd agent-go
go build -tags fuse -o td-agent ./cmd/agent
```

## CI workflow

`.github/workflows/build.yml`:

- `agent`: build mặc định 3 OS (Linux/Windows/macOS) + `go test`.
- `windows-tray`: build `-tags tray -H windowsgui`.
- `windows-installer`: build `-tags "fuse tray"` + bundle WinFsp + Inno Setup → `TelegramDriveSetup.exe` (ký số nếu có secret).
- `pwa`: build web-pwa.
- `release` (tag `v*`): gom artifacts + `SHA256SUMS.txt` → GitHub Release.

