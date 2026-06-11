# Plan: Desktop App Onboarding + Bản Cài Hoàn Thiện (v1.0)

Mục tiêu: app desktop cài 1 file, mở cửa sổ native (WebView2), cho người dùng chọn
nối server có sẵn hoặc chạy server local, nhập URL + mã pairing qua UI (không CLI),
tiếng Việt có dấu toàn bộ, tray hoạt động, không còn cửa sổ console đen. Sẵn sàng
đăng cộng đồng.

## Nguyên tắc
- KISS: dùng WebView2 nhẹ, KHÔNG rebuild full Wails.
- Tái dùng React/PWA hiện có cho trang setup.
- Logic pairing/remote đã có (`--pair`, `--remote`) — chỉ bọc UI, không viết lại.
- Không gõ CLI cho người dùng cuối.
- Mọi text (installer + app + UI) tiếng Việt có dấu.

## Hiện trạng (đã xong trước plan này)
- Agent Go + tray + mount WinFsp chạy được (T: mount OK trên máy anh).
- Installer `TelegramDriveSetup.exe` build qua CI, bundle WinFsp.
- Debug sync log + UI debug đã có.
- Pairing/remote CLI hoạt động.
- Branding Innonet Agency + ký số CI (chờ cert) đã wire.

## Vấn đề cần giải
1. App chạy console → có cửa sổ đen, đóng = tắt agent.
2. Tray icon lỗi `SetIcon` (icon sai định dạng) → không có menu điều khiển.
3. Không có state "lần đầu chạy / chưa cấu hình" → mặc định full local mount ngay.
4. Không có UI nhập URL server + mã pairing.
5. Installer + app text đang không dấu.

---

## Phase 1 — Sửa nền tảng desktop (chặn lỗi hiện tại)
**Mục tiêu:** hết cửa sổ đen, tray hiện, icon chuẩn.

- Build Windows với `-ldflags "-H windowsgui"` cho bản tray/installer.
- Sửa icon tray: nhúng `.ico` hợp lệ (Windows cần ICO, không phải PNG bytes thô).
  - Thêm `internal/tray/icon_windows.ico` + load đúng theo OS.
- Khi chạy GUI mode mà chưa cấu hình: KHÔNG tự mount, mở cửa sổ onboarding.
- Acceptance:
  - chạy app: không cửa sổ console.
  - tray hiện, chuột phải có menu.
  - không crash `SetIcon`.

## Phase 2 — WebView2 window + onboarding state
**Mục tiêu:** cửa sổ native mở trang setup local.

- Thêm `internal/desktop` (build tag `tray`/`windows`):
  - mở WebView2 trỏ `http://127.0.0.1:8750/setup`.
  - dùng lib WebView2 thuần (vd `jchv/go-webview2`) — KHÔNG full Wails.
  - fallback: nếu WebView2 runtime thiếu → mở browser mặc định + thông báo.
- State machine cấu hình lưu ở `data_dir/desktop.json`:
  - `mode`: `unset | local | remote`
  - `server_url`, `device_token` (nếu remote).
- Khởi động:
  - `mode=unset` → mở onboarding.
  - `mode=remote` → reconnect + mount remote.
  - `mode=local` → chạy server local + mount.
- Acceptance:
  - lần đầu mở ra onboarding window.
  - lần sau tự vào đúng mode.

## Phase 3 — Backend API cho onboarding
**Mục tiêu:** UI gọi được, không cần CLI.

- `GET /v1/desktop/state` → mode, server_url, connected.
- `POST /v1/desktop/test-server` { url } → gọi `/health` + `/.td-check` của URL đó, trả ok/version/lỗi rõ.
- `POST /v1/desktop/pair` { url, code, name } → đổi mã pairing lấy token (tái dùng logic `--pair`), lưu state, set mode=remote.
- `POST /v1/desktop/local` → set mode=local, khởi động server local + mount.
- `POST /v1/desktop/mount` / `unmount` (đã có mount API, wire lại).
- `POST /v1/desktop/reset` → xoá cấu hình, về unset.
- Bảo mật: chỉ bind 127.0.0.1, các endpoint desktop chỉ cho loopback.
- Acceptance: từng bước test bằng curl local pass.

## Phase 4 — Onboarding UI (React, có dấu)
**Mục tiêu:** UX rõ ràng kiểu wizard.

- Route `/setup` trong PWA (tách layout, không cần login drive).
- Bước 1 — Chọn chế độ:
  - "Nối tới máy chủ có sẵn" (client)
  - "Chạy máy chủ trên máy này" (server local)
- Bước 2a (client):
  - nhập URL (local `http://127.0.0.1:8750` hoặc VPS `https://...`)
  - nút "Kiểm tra kết nối" → gọi test-server, hiện trạng thái.
  - hướng dẫn: mở PWA server → tạo mã ghép → dán mã vào đây.
  - nhập mã pairing → "Kết nối" → mount.
- Bước 2b (server local):
  - bật server local + mount T:
  - hướng dẫn đăng nhập Telegram (QR) ngay.
- Trạng thái cuối: connected, drive letter, nút mở Drive / Cấu hình lại.
- Tiếng Việt có dấu toàn bộ. Mobile-friendly không bắt buộc (desktop).
- Acceptance: end-to-end pairing với prod `tele.pogen.im` không cần CLI.

## Phase 5 — Tray menu hoàn chỉnh (có dấu)
- Mở giao diện
- Cấu hình máy chủ
- Mount / Unmount ổ ảo
- Trạng thái kết nối
- Tự khởi động cùng Windows (toggle)
- Thoát
- Acceptance: mọi mục hoạt động, tiếng Việt có dấu.

## Phase 6 — Tiếng Việt có dấu toàn bộ
- `installer/td-agent.iss`: lưu UTF-8 BOM, đổi hết text sang có dấu
  (Inno Setup 6 Unicode hỗ trợ).
- Kiểm tra lại các chuỗi Go `log`/notification.
- Quét lại PWA mojibake còn sót.
- Acceptance: cài đặt + app + UI không còn chữ không dấu/mojibake.

## Phase 7 — Build, test, đóng gói
- CI:
  - bản tray + installer build `-H windowsgui`.
  - bundle WebView2 bootstrapper trong installer (Evergreen) nếu cần.
- Test:
  - `go test ./...`, `npm run build`.
  - manual: cài sạch trên Windows → onboarding → pair prod → mount → mở Drive.
- Installer shortcut KHÔNG còn `--mount-on-start` cứng; thay bằng mở app (onboarding/auto theo state).

## Phase 8 — Release cộng đồng
- Bump version 1.0.0 (đã làm) + tag `v1.0.0`.
- README: hướng dẫn 2 luồng (server trước → cài app → nhập URL/mã).
- GitHub Release: installer + binaries + checksum.
- Ghi rõ: chưa ký số nếu chưa có cert (doc CODE_SIGNING.md đã có).

---

## Rủi ro
- WebView2 runtime thiếu trên máy user cũ → cần bundle Evergreen bootstrap.
- `go-webview2` cần đúng arch/DLL; test kỹ trên Windows sạch.
- Đổi shortcut/flow có thể phá hành vi `--mount-on-start` hiện tại → cần migrate state.
- CI Windows GUI build + WebView2 phải giữ `CGO_ENABLED=0` nếu lib hỗ trợ; xác minh sớm.

## Thứ tự ưu tiên
1. Phase 6 (tiếng Việt) — nhanh, lỗi đang lộ rõ.
2. Phase 1 (windowsgui + tray icon) — chặn lỗi cửa sổ đen.
3. Phase 2–4 (onboarding) — phần lõi anh cần.
4. Phase 5,7,8 — hoàn thiện + phát hành.

## Quyết định đã chốt
- WebView2: ĐƯỢC bundle Evergreen bootstrapper vào installer.
- Server local: BẬT Basic Auth mặc định.
- Drive: mặc định `T:`, volume label `Telegram Drive`.
