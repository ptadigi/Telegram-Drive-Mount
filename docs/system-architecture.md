# System Architecture — Ổ Đĩa Cloud Ảo (Telegram Drive)

Kiến trúc hệ thống ở mức tổng quan tới chi tiết thành phần. Tham chiếu: `codebase-summary.md` (bố cục mã), `INSTALL.md`/`DEPLOY.md` (vận hành).

## 1. Tổng quan

```
   ┌─────────────┐     ┌──────────────┐     ┌──────────────────┐     ┌─────────────┐
   │  PWA (web)  │     │ Desktop app  │     │  Reverse proxy   │     │  Telegram   │
   │  React/Vite │ ──▶ │  tray + T:   │ ──▶ │ Nginx/Caddy/CF   │ ──▶ │  (storage)  │
   └─────────────┘     └──────────────┘     └────────┬─────────┘     └──────▲──────┘
                                                      │                      │
                                              ┌───────▼────────┐             │
                                              │   td-agent     │  gotd MTProto│
                                              │  (Go backend)  │ ─────────────┘
                                              │  API + VFS +   │
                                              │  SQLite + cache│
                                              └───────┬────────┘
                                                      │
                                       <data_dir>: metadata.db, uploads/,
                                       chunks/, thumbs/, telegram.session
```

Ba khối: **PWA** (giao diện), **td-agent** (lõi backend), **Telegram** (kho lưu trữ). Desktop app cũng chính là `td-agent` chạy ở chế độ tray + mount.

## 2. Backend `td-agent` (Go)

### Tầng HTTP (`internal/api`)
- Router `net/http` mux. Middleware theo thứ tự: `withCORS` → `withJSON` (chỉ set JSON cho `/v1/*`, `/health`, `/.td-check`; loại `/v1/tus`) → `withAuth`.
- **withAuth:** `/v1/*` và `/webdav` bắt buộc phiên đăng nhập (cookie session) hoặc **device token** (`Authorization: Device <token>`). Public path: `/health`, `/share/*`, `/share-target`, `/v1/users/{login,register,me,logout}`, `/v1/auth/*`, `/v1/devices/pair/exchange`, `/v1/desktop/*` (nhưng handler desktop còn chặn thêm tray-mode + loopback).
- **Endpoint chính:** files (upload/download/stream/thumbnail/hls), tus (`/v1/tus/`), folders/search, shares (`/share/{slug}`, `/raw`, `/unlock`), devices/pairing, desktop onboarding, mount, stats (`/v1/stats`), SSE (`/v1/events`), webdav (`/webdav`).
- **Rate limit:** `viewRate` 600/phút cho xem share + stream; `shareRate` 20/phút cho `/unlock` (chống dò mật khẩu). Tôn trọng `X-Forwarded-For` khi peer là loopback (sau reverse proxy).

### Tầng nghiệp vụ (`internal/drive`)
- `Service` bọc SQLite + Telegram uploader + cache. Mọi truy vấn scope `user_id`.
- Upload: lưu vào `<data_dir>/uploads/` (tên `id-basename`, chống path traversal), ghi metadata, đẩy hàng đợi sync Telegram.
- Thumbnail: ảnh (bilinear, `golang.org/x/image`), video (ffmpeg), PDF (pdftoppm/mutool) — best-effort, lazy regen.
- Stream: ưu tiên cache local; thiếu thì kéo từ Telegram; hỗ trợ HTTP Range (seek video) và HLS qua ffmpeg.
- Share: slug ngẫu nhiên, mật khẩu (hash), hết hạn, giới hạn lượt tải; trang xem server-render.

### Lưu trữ Telegram (`internal/telegramstorage` + `internal/auth`)
- Dùng `gotd/td` (MTProto). Đăng nhập bằng QR/điện thoại; session lưu file mã hóa AES-256 (`internal/secret`, khóa từ `TD_AGENT_SESSION_KEY`).
- File được tải lên một **kênh Telegram riêng** làm object storage; metadata (channel/message/file id) lưu trong SQLite.

### Mount / VFS (`internal/vfs`, tag `fuse`)
- `Manager` điều phối `Mounter` (cgofuse → WinFsp/FUSE). Backend trừu tượng: `localBackend` (đọc DB local) hoặc `remote.Backend` (gọi server qua HTTPS).
- `SwitchBackend` đổi backend lúc chạy → pair xong là remount sang remote, không cần khởi động lại app.

### Lưu trữ cục bộ (`<data_dir>`)
- `metadata.db` (SQLite, pure-Go), `uploads/` (file đang chờ/sync + tus temp), `chunks/`/`thumbs/` (cache), `telegram.session`, `backups/` (auto-backup metadata mỗi 6h), `desktop.json`.

## 3. PWA (React/Vite)

- SPA phục vụ bởi agent (`pwa_dir` hoặc tự dò `<exeDir>/pwa`); fallback `index.html` cho route client.
- Giao tiếp qua REST (`/v1/*`, cookie session) + SSE (`/v1/events`) cho realtime, kèm stale-while-revalidate (focus/visible/online/poll) để bền với proxy buffer.
- Upload: hàng đợi pool 6, file >32MB dùng tus (chunk 16MB, resume). Kéo-thả thư mục lớn streaming theo lô.
- Xem file in-app: ảnh (zoom/pan), video/audio (stream), pdf (iframe), docx (mammoth). Icon Phosphor build-time (offline).

## 4. Mô hình đa thiết bị

```
   [VPS server]  td-agent đầy đủ (Telegram session + SQLite + cache + PWA)
        ▲  HTTPS + device token (pairing)
        │
   [Desktop A] td-agent --remote  → mount T:  (thin client, không giữ session)
   [Desktop B] td-agent --remote  → mount T:
   [Điện thoại] PWA qua trình duyệt
```

- Một server giữ trạng thái; các máy khác là thin-client mount qua HTTPS sau khi ghép thiết bị (mã pairing dùng 1 lần, hết hạn 5 phút).
- Trên cùng tài khoản Telegram → tất cả thấy cùng dữ liệu.

## 5. Bảo mật

- Tài khoản PWA: mật khẩu bcrypt; session token lưu hash SHA-256; cookie HttpOnly, Secure sau HTTPS.
- Device token: `crypto/rand`, lưu hash, có thu hồi.
- Session Telegram: mã hóa tại chỗ (AES-256-GCM) bằng `TD_AGENT_SESSION_KEY`.
- Phân tách dữ liệu theo `user_id` ở mọi tầng (list, search, ops, stats).
- Endpoint onboarding desktop chỉ hoạt động khi agent ở tray-mode + request loopback (server công khai trả 403).
- Audit log trong SQLite (`audit_log`, `GET /v1/audit`).

## 6. Triển khai & CI/CD

- **Vận hành:** systemd (`td-agent.service`) hoặc Docker compose; reverse proxy cho domain+HTTPS; cấu hình `client_max_body_size 0` + `proxy_request_buffering off` cho file lớn; `proxy_buffering off` cho `/v1/events`.
- **CI (`.github/workflows/build.yml`):** build agent (linux/windows/macos, CGO tắt), windows-tray, windows-installer (Inno Setup + WinFsp), pwa; tag `v*` → release kèm `SHA256SUMS.txt`.
- **Build tags:** `fuse` (mount native), `tray` (systray). `-H windowsgui` ẩn console trên Windows.

## 7. Quyết định kiến trúc đáng chú ý

- **SQLite thuần Go + CGO tắt** → binary tĩnh, đa nền tảng, cài 1-click dễ.
- **Telegram làm object storage, metadata ở SQLite** → tách UX khỏi giới hạn Telegram, kiểm soát thư mục/chia sẻ.
- **VFS backend hoán đổi lúc chạy** → một binary phục vụ cả server-local lẫn thin-client remote.
- **Icon build-time (unplugin-icons)** → PWA chạy offline, không phụ thuộc CDN icon.
