# Roadmap Ổ Đĩa Cloud Ảo

Bản cập nhật theo mục tiêu đã chốt với chủ dự án.

## Nguyên tắc cốt lõi (đã chốt)

1. **Telegram là source of truth duy nhất**. Mọi file gốc nằm ở Telegram.
2. **Agent (desktop hoặc VPS)** chỉ là **cache nóng + gateway** giữa PWA/mobile và Telegram.
3. **PWA/mobile** là client thuần. Không lưu cố định gì trên máy người dùng cuối, chỉ xem qua HTTP.
4. **Cache, thumbnail, metadata** được phép lưu ở Agent với 3 chính sách: `mirror`, `smart`, `cloud_only`.
5. **Tiếng Việt có dấu** trong toàn bộ giao diện và thông báo.
6. **Tự lo phía sau, đơn giản phía trước**: người dùng không cần biết đến API ID, channel Telegram, token, v.v.

## Hai kịch bản triển khai

| | Chạy desktop | Deploy VPS |
|---|---|---|
| Agent ở đâu | Máy người dùng | Máy chủ |
| Cache file ở đâu | Máy người dùng | VPS |
| Điện thoại có lưu file? | Không | Không |
| Mỗi lần xem | HTTP localhost / LAN | HTTP qua domain/tunnel |
| Khi cache miss | Agent kéo từ Telegram | VPS kéo từ Telegram |

Cả 2 kịch bản dùng chung 1 codebase, chỉ khác:
- Desktop có tray app, autostart, mount ổ ảo.
- VPS chạy headless, có CLI flag và docs deploy.

## Đã hoàn thành

- Đăng nhập Telegram số điện thoại + 2FA.
- Upload nhiều file, kéo thả file/thư mục, tự tạo cây thư mục.
- Sync nền lên Telegram (queue + worker + retry).
- Hash SHA-256 chống upload trùng.
- Fallback: tải lại file từ Telegram khi cache mất.
- Thumbnail ảnh.
- File manager: rename, di chuyển, sao, thùng rác, xóa hẳn, ZIP folder.
- Tìm kiếm theo tên, sort, grid/list view.
- Multi-select bulk: sao/bỏ sao, di chuyển, xóa hàng loạt.
- Drag & drop nâng cao: thả file ngoài vào folder card, kéo file giữa folder.
- Tạo link chia sẻ: mật khẩu (bcrypt), hết hạn, giới hạn lượt tải, thu hồi, xóa.
- Trang public `/share/<slug>` HTML đẹp tiếng Việt, hỗ trợ folder ZIP.
- Cấu hình domain chia sẻ: LAN, tên miền riêng, Cloudflare Tunnel auto.
- Sync thư mục desktop: thêm/quét lại/tạm dừng/bật lại/xóa, watcher fsnotify.
- Realtime SSE cho file/transfer/sync root/share/cache.
- 3 chính sách cache: mirror / smart / cloud_only, có nút dọn cache thủ công.
- File viewer trực tiếp: ảnh, video, audio, PDF, markdown, text.
- PWA installable, bottom nav mobile, Home view, toast/confirm.
- Backend: WAL SQLite, busy_timeout, rate-limit `/share/*`.
- Deploy: CLI flag `--config / --data-dir / --addr`.

## Mốc đang làm

### M1 — Sync watcher robust + Telegram stream (đang ưu tiên)

Mục tiêu: cache miss trên VPS không còn phải tải full file rồi mới serve. Sync watcher không bỏ sót/lặp.

- ✅ Stream Telegram chunk-by-chunk theo HTTP Range:
  - `GET /v1/files/stream?id=...` proxy Telegram, hỗ trợ `Range`.
  - Khi xem video chưa cache: vừa tải vừa serve, không chờ full.
- ✅ Sync watcher:
  - Debounce file đang copy (đợi mtime+size ổn định).
  - Xử lý sự kiện `rename`, `delete` (soft delete metadata).
  - Bỏ qua file trùng mtime đã import.

### M1.5 — Storage channel chuyên dụng

Mục tiêu: chuyển từ Telegram Saved Messages sang storage channel/chat riêng.

- Tự tạo private channel khi user đăng nhập lần đầu nếu cần.
- Lưu đủ `telegram_channel_id`, `access_hash`, `file_reference` vào `file_versions`.
- Đổi download/stream sang `ChannelsGetMessages` cho file thuộc channel.
- UI Settings cho user chọn/đổi storage channel.

### M2 — Tray app + autostart + native folder picker

Mục tiêu: người dùng cài 1 lần, app luôn sẵn sàng, không phải `go run`.

- Tray bằng `getlantern/systray`.
- Menu tiếng Việt: Mở giao diện, Mở thư mục dữ liệu, Thêm thư mục đồng bộ, Tạm dừng/Tiếp tục, Trạng thái Telegram, Thoát.
- Embed Agent trong cùng process, không cần shell ngoài.
- Native folder picker khi `Thêm thư mục đồng bộ`.
- Auto start theo OS:
  - Windows: registry `Run` key.
  - macOS: LaunchAgent plist.
  - Linux: `~/.config/autostart`.
- Notification khi sync xong/lỗi.

### M3 — Mount ổ ảo

Mục tiêu: người dùng thấy ổ đĩa thật trên máy, mở file = mở từ ổ ảo.

- Windows: WinFsp.
- macOS: macFUSE / FUSE-T.
- Linux: FUSE.
- Đọc:
  - Folder list từ metadata.
  - Mở file → nếu cache có thì đọc local, không có thì stream Telegram.
- Ghi:
  - Tạo file mới trong ổ ảo → đẩy queue upload.
  - Sửa file → version mới.
- Quyền: read-only trước, sau đó mới read-write.

## Mốc tiếp theo (sau M1-M3)

### M4 — Multi-user và bảo mật cho VPS

Khi deploy VPS, có nhiều người dùng chung:

- App-level login (email/password hoặc magic link).
- Mỗi user có space metadata riêng, channel Telegram riêng.
- API token cho client gọi.
- Audit log thao tác file/share.
- Backup metadata DB tự động.

### M5 — Stream tối ưu và preview office

- Adaptive streaming cho video lớn (HLS hoặc range chunk).
- Office Online viewer khi có domain public (docx/xlsx/pptx).
- PDF.js fallback cho mobile.
- Markdown render rich text (marked + sanitize).

### M6 — Distribution

- Installer Windows MSIX/Inno Setup.
- macOS .pkg ký notarized.
- Linux AppImage/deb.
- `docker-compose.yml` cho self-host VPS.
- Systemd unit mẫu.
- `docs/DEPLOY.md` quickstart self-host.
- GitHub Actions build cho 3 OS.

### M7 — UX nâng cao

- Filter chip Loại / Người / Lần sửa đổi.
- Bin auto-retention sau N ngày.
- Multi-select bulk download (ZIP nhiều mục).
- Shortcut tới folder.
- Recent activity log riêng.

## Việc đang KHÔNG làm (loại khỏi scope ngắn hạn)

- Bot Telegram phụ trợ.
- Trình quản lý kế toán/cộng tác.
- AI search nội dung.
- Mobile native app riêng (iOS/Android).
- Bridge sang Drive/Dropbox khác.

Sẽ xét sau M3.

## Tiêu chí xong cho mỗi mốc

Mỗi mốc xem như xong khi:
1. Backend `go test ./...` pass.
2. Frontend `npm run build` pass.
3. Smoke test thật một flow end-to-end.
4. Trong 2 kịch bản triển khai (desktop + VPS) đều hoạt động đúng.
5. Có dòng cập nhật trong README.

## Quyết định chốt

- Em không làm tray, mount, deploy docs trước khi xong M1.
- M1 là gốc của trải nghiệm: nếu stream Telegram chậm thì VPS deploy không có ý nghĩa.
- Sau M1 mới qua M2 (tray) rồi M3 (mount).
- Mỗi mốc đóng kín end-to-end, không nửa nhơ.
