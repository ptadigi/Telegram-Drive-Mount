# Code Standards — Ổ Đĩa Cloud Ảo (Telegram Drive)

Quy ước viết mã & cấu trúc để đóng góp nhất quán. Áp dụng cho cả backend Go và PWA.

## Nguyên tắc chung

- **An toàn dữ liệu trước tiên.** Telegram là nguồn chân lý; cache chỉ là bản sao. Không thao tác phá hủy mà không có thùng rác/backup.
- **Scope theo user.** Mọi truy vấn file/folder/share phải lọc `COALESCE(user_id,'') = COALESCE(?, '')`. Tìm kiếm cũng vậy (tránh lộ chéo dữ liệu).
- **Best-effort cho tính năng phụ.** Thiếu ffmpeg/poppler → bỏ thumbnail, không làm fail upload.
- **Self-host first.** Không hardcode domain/instance của maintainer. Dùng `window.location.origin` hoặc URL người dùng nhập.
- **UTF-8 tiếng Việt.** Mọi file nguồn là UTF-8. **Không** sửa file hàng loạt bằng PowerShell `Set-Content` (đã từng phá dấu tiếng Việt) — dùng công cụ ghi UTF-8 (Edit/WriteFile, hoặc Python `io.open(...encoding="utf-8")`).

## Backend (Go)

- **Module:** `telegram-drive-agent`. **CGO tắt** ở mọi build (SQLite thuần Go). Mount native dùng build tag `fuse`; tray dùng tag `tray`.
- **Bố cục:** logic đặt trong `internal/<domain>/`; HTTP handler trong `internal/api/`. Không để nghiệp vụ trong handler — gọi xuống `drive.Service`.
- **Context:** truyền `r.Context()` (mang `user_id` qua `drive.WithUser`) xuống tầng DB. Tránh `context.Background()` cho thao tác cần user (lỗi tus trước đây import bằng background context → file mất chủ).
- **SQL:** dùng tham số hóa (`?`), không nối chuỗi. `COALESCE` cột nullable trong cả SELECT lẫn UPDATE WHERE (đã có bug 403 share do `expires_at`/`max_downloads` NULL).
- **Lỗi:** trả lỗi có thông điệp tiếng Việt rõ ràng; không nuốt lỗi gây mất dữ liệu thầm lặng. `defer rows.Close()` cho mọi query nhiều dòng.
- **Đồng thời:** bảo vệ map/state dùng chung bằng mutex. Tạo folder serialize để tránh trùng khi upload song song.
- **Tài nguyên:** đóng file/reader; tránh `defer` trong vòng lặp dài; janitor dọn temp theo TTL (không đụng upload đang chạy).
- **Bảo mật:** token lưu dạng hash (SHA-256/bcrypt), không plaintext; pairing code dùng `crypto/rand`, có expiry + consume; endpoint desktop chỉ bật ở chế độ tray + loopback.
- **Phiên bản:** cập nhật `const version` trong `cmd/agent/main.go` **và** `#define AppVersion` trong `installer/td-agent.iss` cùng nhau.
- **Kiểm tra trước khi commit:** `go build ./...`, `go vet ./internal/api`, `go test ./...` (ít nhất các package đụng tới). Cả build mặc định lẫn build `-tags "fuse tray"` (Windows).

### Định dạng & lint
- `gofmt`/`go vet` sạch. Tên export có doc comment. Hằng số/biến môi trường gom vào package phù hợp (`internal/secret`, `internal/config`).

## Frontend (PWA — React/TS)

- **Icon:** import từ `lucide-react` (đã alias → `src/icons.tsx` bộ Phosphor). Thêm icon mới → khai báo trong `icons.tsx`, không import trực tiếp thư viện icon trong component. Một bộ icon duy nhất.
- **Fetch:** dùng helper trong `src/api/agent.ts`, luôn kèm `credentials: "include"`. Không hardcode host — `AGENT_BASE_URL` xử lý dev/prod.
- **State:** upload qua `state/uploads.ts` (pool 6, flush throttle ~250ms, tus cho file >32MB). Làm tươi danh sách qua `useRevalidate` ở chế độ **silent** cho nền (tránh nháy loading).
- **Hooks:** dọn dẹp effect (đóng `EventSource`, clear interval). Tránh stale closure; deps đúng.
- **Hiệu năng:** kéo-thả thư mục lớn dùng streaming theo lô (không gom hết vào RAM). Ảnh `loading="lazy"`. Không chặn main-thread > ~16ms.
- **i18n:** chuỗi UI tiếng Việt; tránh mojibake (kiểm tra hiển thị dấu sau khi sửa).
- **Build:** `npm run build` (tsc + vite) phải sạch trước khi commit.

### UI/UX (theo design system)
- Touch target ≥ 44px; có phản hồi khi nhấn; menu mở được bằng chuột phải + long-press mobile.
- Tương phản chữ ≥ 4.5:1; không dùng màu làm dấu hiệu duy nhất.
- Animation 150–300ms; tôn trọng `prefers-reduced-motion`.

## Quy ước Git & PR

- **Commit message** dạng `type(scope): mô tả ngắn` (vd `fix(share): ...`, `feat(upload): ...`), thân commit giải thích nguyên nhân + cách sửa.
- **Không commit secret** (`config.local.json`, `*.session`, `*.pfx`, token). Đã loại khỏi lịch sử Git.
- **Branch:** không push thẳng lên `main` cho thay đổi lớn nếu làm việc nhóm; với repo cá nhân hiện tại commit thẳng `main` + tag `v*` để release.
- **Release:** tạo tag `vX.Y.Z` → CI tự build binary + installer + `SHA256SUMS.txt` và đăng GitHub Release. Cập nhật `CHANGELOG.md`.

## Tài liệu

- `docs/` là nguồn chân lý. Khi đổi hành vi cài đặt/vận hành, cập nhật `docs/INSTALL.md` / `DEPLOY.md` / `TROUBLESHOOTING.md` tương ứng.
- README giữ ngắn gọn (< 300 dòng), trỏ vào `docs/` cho chi tiết.
