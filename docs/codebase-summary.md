# Codebase Summary — Ổ Đĩa Cloud Ảo (Telegram Drive)

Tổng quan mã nguồn để người mới đóng góp định hướng nhanh. Chi tiết kiến trúc xem `system-architecture.md`; quy ước code xem `code-standards.md`.

## Bố cục thư mục gốc

```
Telegram-Drive/
├── agent-go/          # Backend Go: API, Telegram sync, VFS/mount, tray, onboarding
├── web-pwa/           # PWA React + Vite + TypeScript (UI tiếng Việt)
├── deploy/            # docker-compose, systemd unit, config mẫu, script cài 1-click
├── installer/         # Inno Setup (.iss) đóng gói installer Windows
├── docs/              # Tài liệu (file này nằm ở đây)
├── plans/             # Ghi chú thiết kế / kế hoạch tính năng
├── .github/workflows/ # CI: build đa nền tảng + release
├── README.md  CHANGELOG.md  CONTRIBUTING.md  LICENSE
```

## agent-go (backend Go)

- Module: `telegram-drive-agent`. Entry: `cmd/agent/main.go` (`const version`). CGO tắt, SQLite thuần Go (`modernc.org/sqlite`).
- `internal/api/` — HTTP server, router, middleware auth, các handler (`server.go`), SSE events, tus upload (`tus.go`), trang share server-render (`share_page.go`), thống kê, rate limit (`ratelimit.go`), webdav (`webdav.go`).
- `internal/drive/` — lõi nghiệp vụ drive: `service.go` (upload/lưu/quét), `operations.go` & `operations_v2.go` (đổi tên/di chuyển/xóa/tìm kiếm), `share.go` (link chia sẻ), `thumbnail.go` (ảnh bilinear + ffmpeg/poppler), `hls.go`, `cache.go`, `context.go` (user context), `debug.go`.
- `internal/auth/` — đăng nhập Telegram (gotd): `service.go`, `qrlogin.go`.
- `internal/users/` — tài khoản PWA (bcrypt), phiên đăng nhập (token hash).
- `internal/devices/` — ghép thiết bị (pairing code, device token).
- `internal/desktop/` — trạng thái onboarding desktop (`state.go`, `client.go`): unset/local/remote.
- `internal/remote/` — thin-client `--remote`: backend HTTPS tới server, lưu token.
- `internal/vfs/` — mount manager + backend (local/remote), `platform_fuse.go` (cgofuse, build tag `fuse`), `manager.go` (`SwitchBackend` đổi backend lúc chạy).
- `internal/telegramstorage/` — đọc/ghi file qua kênh Telegram.
- `internal/secret/` — mã hóa session tại chỗ (AES-256, `TD_AGENT_SESSION_KEY`).
- `internal/config/` — đọc config JSON + env override.
- `internal/db/` — mở SQLite, migration, pragma.
- `internal/tray/` — systray + icon (build tag `tray`); `cmd/agent/setup_window_windows.go` mở WebView2 onboarding.
- `internal/tunnel/`, `internal/trash/`, `internal/webdavfs/` — phụ trợ.

## web-pwa (frontend)

- Vite 7 + React 19 + TypeScript. Build: `npm run build` → `dist/`.
- `src/App.tsx` — khung app, điều hướng view, xử lý `?view=`/`?shared=` (Web Share Target).
- `src/api/agent.ts` — client gọi API agent (fetch + cookie), upload thường + tus, `AGENT_BASE_URL` (dev dùng IP:8750, prod dùng same-origin).
- `src/components/` — `DriveBrowser` (lưới file, kéo-thả streaming, context menu chuột phải/long-press), `FileViewer` (xem ảnh/video/docx…), `HomeView` (dashboard), `ShareDialog`/`SharePage`, `SetupWizard` (onboarding `/setup`), `UploadDock`, `DevicesView`, `SettingsView`…
- `src/state/` — `uploads.ts` (hàng đợi upload pool 6 + tus), `revalidate.ts` (stale-while-revalidate), `ui.tsx` (toast/confirm).
- `src/icons.tsx` — bộ icon Phosphor (build-time SVG qua unplugin-icons); `lucide-react` được alias về đây trong `vite.config.ts`. Có `FileIcon` màu theo định dạng.
- `src/styles.css` — design system token-driven.
- Phục vụ: agent set `pwa_dir` (hoặc tự dò `<exeDir>/pwa`) và serve SPA; fallback `index.html` cho route client (trừ `/v1/`, `/webdav`).

## deploy / installer / CI

- `deploy/docker-compose.yml` — agent (golang image) + pwa (node image). `deploy/td-agent.service` — systemd unit (user `td-agent`, `/usr/local/bin/td-agent`, `/etc/td-agent/config.json`, `/var/lib/td-agent`, cổng 8750). `deploy/config/config.example.json` — config mẫu.
- `deploy/install.sh` / `install.ps1` / `install-docker.sh` — script cài 1-click.
- `installer/td-agent.iss` — Inno Setup: gói `td-agent.exe` (tags `fuse tray`), `pwa/`, WinFsp MSI; tạo autostart + sinh `TD_AGENT_SESSION_KEY`.
- `.github/workflows/build.yml` — jobs: `agent` (linux/windows/macos), `windows-tray`, `pwa`, `windows-installer`, `release` (tag `v*` → GitHub Release + `SHA256SUMS.txt`).

## Luồng dữ liệu chính (rút gọn)

1. PWA/desktop → `POST /v1/files/upload` hoặc tus `/v1/tus/` → agent lưu vào `<data_dir>/uploads/` → ghi metadata SQLite (scope theo user).
2. Worker đồng bộ đẩy file lên **kênh Telegram** → đánh dấu `telegram_synced`.
3. Đọc file: stream từ cache local, thiếu thì kéo lại từ Telegram (có Range cho video).
4. Mount `T:` đọc/ghi qua VFS backend (local hoặc remote-HTTPS).

## Điểm vào để đọc đầu tiên

- API & routing: `agent-go/internal/api/server.go`
- Nghiệp vụ file: `agent-go/internal/drive/service.go`
- UI chính: `web-pwa/src/components/DriveBrowser.tsx`
- Cài đặt/vận hành: `docs/INSTALL.md`, `docs/DEPLOY.md`
