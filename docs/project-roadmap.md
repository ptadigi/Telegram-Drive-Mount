# Project Roadmap — Telegram-Drive

Cập nhật: 2026-06-06.

## Trạng thái

- M1 Sync watcher robust + Telegram stream: **xong**.
- M1.5 Storage channel auto: **xong**.
- M2 Tray + autostart: **xong**.
- M3 Native mount ổ ảo (WinFsp/FUSE): **xong - read-mostly**.
- Phase mới hoàn thiện:
  - Sync desktop không nhân đôi cache (`cache_origin`).
  - Cache cleanup an toàn, đẩy file evict vào Recycle Bin OS.
  - Telegram filename gốc + auto private channel.
  - Retry/backoff theo `FLOOD_WAIT_<n>`.
  - QR login Telegram.
  - Mount UI trong PWA Settings.
  - CI build matrix có job `agent-fuse` (`-tags fuse`).

## Đang còn

- Native mount ghi qua ổ ảo (write-back) — hiện đọc + restore từ Telegram.
- Sync hai chiều + conflict resolver.
- Office viewer (docx/xlsx/pptx) khi có domain public.
- Installer Windows/macOS/Linux.

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

`.github/workflows/build.yml` có 2 job song song:

- `agent`: build mặc định cho 3 OS.
- `agent-fuse`: build với `-tags fuse` (Windows cần WinFsp, macOS dùng FUSE-T, Linux libfuse-dev).
