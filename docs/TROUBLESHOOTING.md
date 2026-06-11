# Khắc phục sự cố (Troubleshooting)

Tổng hợp các vấn đề thực tế đã gặp khi deploy/test và cách xử lý. Dành cho người
self-host và maintainer.

## 1. PWA hiển thị HTML source / không render

Triệu chứng: mở trang ra HTML thô, hoặc trắng màn.
Nguyên nhân: agent serve mọi response với `Content-Type: application/json` (kể cả
HTML/JS/CSS) → browser không render. Hoặc reverse proxy trả sai MIME.
Fix: agent chỉ set JSON cho `/v1/*`, `/health`, `/.td-check`; static PWA do
`http.ServeFile` tự set MIME. Đã xử lý trong code.

## 2. Trang cấu hình `/setup` báo 404 trên bản desktop

Nguyên nhân: installer copy PWA vào `{app}\pwa` nhưng agent không được set
`pwa_dir` → không route trang nào → 404.
Fix: agent tự dò `pwa/` cạnh file thực thi khi `pwa_dir` trống. Không cần config tay.

## 3. Tray "Mở giao diện" luôn mở localhost

Nguyên nhân: hardcode `127.0.0.1:8750`.
Fix: đọc cấu hình thật mỗi lần bấm — chế độ remote mở đúng URL máy chủ đã pair;
local/chưa cấu hình mới mở localhost.

## 4. Ổ T: mount được nhưng "not accessible"

Nguyên nhân: chế độ remote nhưng mountManager dùng backend local (DB rỗng) thay vì
backend remote trỏ về server đã pair.
Fix: khi desktop ở chế độ remote, mountManager dùng `remote.Backend` (server URL +
device token). Lưu ý: chọn backend lúc agent khởi động — pair lần đầu xong nên
**thoát app rồi mở lại** trước khi mount.

## 5. App desktop có cửa sổ console đen / tray crash icon

Nguyên nhân: build không có `-H windowsgui`; tray nạp icon sai định dạng (PNG thay vì .ico).
Fix: build `-ldflags "-H windowsgui"`; nhúng `.ico` chuẩn cho Windows tray.

## 6. Tiếng Việt bị lỗi font (mojibake) / installer không dấu

Nguyên nhân: file lưu sai encoding; installer Inno Setup thiếu BOM.
Fix: chuẩn hóa UTF-8 (ftfy cho PWA), `.iss` lưu UTF-8 BOM. Mọi text có dấu.

## 7. File lên Telegram nhưng PWA không thấy (realtime)

Nguyên nhân: Cloudflare/reverse proxy buffer Server-Sent Events; tab mobile bị
suspend → SSE đứt → list không tự cập nhật. File thực tế đã vào DB.
Fix:
- Agent: `X-Accel-Buffering: no`, `Cache-Control: no-transform`, `retry: 3000`,
  heartbeat 15s cho `/v1/events`.
- PWA: pattern stale-while-revalidate — refresh khi focus/visible/online + SSE +
  poll dự phòng 20-25s. Kể cả SSE bị chặn, list vẫn cập nhật.
Proxy nên không buffer `/v1/events` (xem docs/DEPLOY.md).

## 8. Kéo file ra ngoài vùng drop → browser tải/mở file

Nguyên nhân: chỉ vùng `.drive-browser` chặn default; thả trúng vùng trống thì browser
mở file, thay trang.
Fix: guard cấp window nuốt drag/drop file ngoài dropzone.

## 9. Upload file lớn treo ở vài % (KHÔNG qua chunked)

Nguyên nhân: reverse proxy chặn body vượt `client_max_body_size`. File không tới agent
(không có dòng `upload_received` trong sync.log). Ví dụ thật: OpenResty/1Panel mặc
định `client_max_body_size 50m` → file 53MB bị chặn.
Fix nhanh (proxy): tăng `client_max_body_size`, `proxy_request_buffering off`,
`proxy_read_timeout/send_timeout` lớn. Xem docs/DEPLOY.md.
Fix gốc: chunked upload (mục dưới).

## 10. Chunked upload (tus) — các lỗi đã gặp

Đường upload file lớn dùng tus protocol (`/v1/tus`), chunk 16MB, ngưỡng >32MB.

- **URL upload trả http thay vì https → PATCH bị chặn (mixed content)**:
  tusd sau proxy thấy scheme http. Fix: bật `RespectForwardedHeaders` để dùng
  `X-Forwarded-Proto: https`. Proxy phải set header này.
- **File rất lớn (vài trăm MB) lỗi `ERR_UPLOAD_NOT_FOUND` (404) lúc gần xong + UI đơ**:
  do `parallelUploads` dùng server-side concatenation — mong manh sau proxy và spike
  RAM khi slice file lớn. Fix: `parallelUploads: 1` (single-stream, vẫn chunked +
  resumable). Song song giữa NHIỀU file vẫn còn nhờ pool 6 worker.

## 11. Upload nghìn file / folder lớn bị giật

Nguyên nhân (bản cũ): upload tuần tự 1 file/lần; mỗi tick progress re-render O(n);
reconcile O(n²).
Fix: pool 6 worker song song; state trong ref + flush UI throttle 250ms; reconcile
transfer bằng Map O(1); dock hiển thị tổng hợp + vài item đang chạy; cảnh báo
`beforeunload`; nút retry file lỗi.

## 12. Cache cục bộ và dung lượng đĩa

Hạ cache <1GB an toàn, không mất file (Telegram là source of truth; chỉ file đã
`telegram_synced` mới bị dọn). Đánh đổi: xem lại file cũ hay phải kéo từ Telegram.
File đang chờ sync vẫn chiếm đĩa tạm trong `uploads/`. Xem docs/DEPLOY.md.

## Công cụ debug

- `GET /v1/debug/sync` (cần đăng nhập): log sync gần nhất + transfer lỗi.
- Tab "Debug sync" trong PWA: xem log + transfer failed, nút Copy log.
- File log: `<data_dir>/logs/sync.log` (JSON lines).
