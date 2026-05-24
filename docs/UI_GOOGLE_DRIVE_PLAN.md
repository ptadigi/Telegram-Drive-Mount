# Kế hoạch giao diện kiểu Google Drive

Tài liệu này mô tả giao diện mong muốn cho **Ổ Đĩa Cloud Ảo**. Mục tiêu là người dùng cảm thấy đang dùng một sản phẩm cloud drive thật, không phải tool Telegram. Telegram chỉ là tầng lưu trữ ẩn phía sau.

## Triết lý

- Nhanh: thao tác file phản hồi tức thì, không chờ Telegram.
- An toàn: không mất file, không ghi đè bậy, có version/retry.
- Hiệu quả: không upload trùng, không scan thừa, không tốn tài nguyên.
- Realtime: trạng thái upload/sync/cache luôn cập nhật trực tiếp.
- Tiếng Việt có dấu trong toàn bộ nội dung.

## Cấu trúc tổng thể

App gồm 3 vùng chính:

1. Sidebar trái
2. Top bar
3. Main content + Detail/Activity panel phải khi cần

### Sidebar trái

- Logo `Ổ Đĩa Cloud Ảo`.
- Nút lớn `+ Mới` mở menu nhanh:
  - Tải file lên
  - Tải thư mục lên
  - Tạo thư mục
  - Tạo link chia sẻ
- Mục điều hướng:
  - `Trang chủ`
  - `Drive của tôi` (nguồn cloud, tương đương ổ cứng cloud Telegram)
  - `Máy tính` (các sync root đang được mount/đồng bộ trên PC)
  - `Được chia sẻ với tôi`
  - `Gần đây`
  - `Có gắn dấu sao`
  - `Nội dung rác`
  - `Thùng rác`
  - `Bộ nhớ`
- Card dung lượng:
  - Đã sử dụng X/Y
  - Nút `Mua thêm bộ nhớ` (placeholder, có thể là gói premium sau).

### Top bar

- Search lớn `Nhận câu trả lời từ Drive` hoặc `Tìm trong Drive`.
- Filter chip:
  - `Loại`
  - `Người`
  - `Lần sửa đổi gần đây nhất`
  - `Nguồn`
- Nút trợ giúp `?`.
- Nút cài đặt.
- Nút trợ lý AI (sau).
- Nút app launcher.
- Avatar người dùng / trạng thái Telegram.

### Main content

- Tiêu đề khu vực hiện tại, ví dụ `Drive của tôi` hoặc `Máy tính`.
- Toggle hiển thị: `List` / `Grid`.
- Nút info bật/tắt Detail panel phải.
- Sort: `Ngày sửa đổi`, `Tên`, `Dung lượng`, `Loại`.
- Khu vực folder/file:
  - Folder hiện dạng card với icon thư mục lớn và tên.
  - File hiện dạng card với thumbnail/icon theo `kind`.
  - Trạng thái sync nhỏ trên từng file:
    - `Chờ đồng bộ`
    - `Đang đồng bộ`
    - `Đã đồng bộ`
    - `Lỗi đồng bộ`
- Hỗ trợ chọn nhiều, kéo thả.

### Detail / Activity panel phải

- Tab `Chi tiết`:
  - tên file/folder
  - loại
  - dung lượng
  - vị trí
  - chủ sở hữu
  - trạng thái Telegram
  - link chia sẻ (nếu có)
- Tab `Hoạt động`:
  - lịch sử upload/sync.
- Có thể đóng panel.

### Drive của tôi vs Máy tính

- `Drive của tôi`:
  - vùng cloud thuần
  - mọi file/folder hiển thị từ metadata cloud
  - được Telegram lưu trữ phía sau
- `Máy tính`:
  - liệt kê các máy có Go Agent kết nối
  - mỗi máy có các sync root đang chạy
  - mỗi sync root tương ứng một thư mục local đang được watch
  - có thể chọn “Mount như ổ ảo trên máy này” khi mount layer sẵn sàng

## Drag & Drop chi tiết

- Vào toàn bộ vùng main:
  - Kéo file đơn → upload vào folder hiện tại
  - Kéo nhiều file → upload nhiều file
  - Kéo folder (Chrome/Edge) → upload folder, tự tạo cây thư mục
  - Hover vào folder card khi kéo → highlight folder, thả vào thì upload vào folder đó
- Khi đang kéo:
  - Hiện overlay drop zone toàn vùng, chữ `Thả file để tải lên thư mục này`
- Khi thả:
  - Tạo upload queue ngay.
  - Mỗi file có dòng riêng, progress riêng.
  - Auto sync Telegram nền, không cần bấm nút.

## Upload queue panel

- Hiện ở góc dưới phải.
- Có thể thu nhỏ.
- Mỗi item:
  - Tên file
  - Phase: `Đang tải lên Agent`, `Đang xử lý`, `Đang đồng bộ Telegram`, `Đã đồng bộ`, `Lỗi`
  - Progress %
  - Tốc độ (sau)
  - Nút thử lại nếu lỗi
- Tổng:
  - Số file xong / tổng
  - Trạng thái chung
- Khi không còn item nào active, panel tự ẩn.

## Tạo link chia sẻ

- Trên file/folder:
  - Menu hoặc nút `Tạo link chia sẻ`.
- Modal chia sẻ:
  - Trạng thái: `Chỉ tôi` / `Mọi người có link`.
  - Có thể đặt mật khẩu.
  - Có thể đặt thời hạn.
  - Có thể thu hồi.
  - Hiển thị link copy được, dạng:
    - `https://share.tencuaban.com/abcdef`
- Backend lưu metadata share:
  - slug
  - password_hash
  - expires_at
  - revoked
  - access_count
- Gateway public phục vụ link:
  - `gateway-go` sau này
  - hoặc tunnel/proxy domain riêng.

## Mount “Máy tính” thành ổ ảo

- Khi user vào `Máy tính` → chọn máy local có Agent → chọn `Mount ổ ảo`:
  - Windows: WinFsp
  - macOS: macFUSE
  - Linux: FUSE
- Ổ ảo:
  - thấy folder/file giống cloud
  - mở file → fetch từ local cache hoặc Telegram
  - ghi file → cache local + queue upload
- Tray app:
  - mở ổ ảo
  - tạm dừng/tiếp tục sync
  - mở giao diện web

## Trạng thái và phản hồi UI

Mỗi đối tượng phải có trạng thái rõ:

- File:
  - `local_indexing`
  - `local_ready`
  - `thumbnailing`
  - `pending_upload`
  - `uploading_telegram`
  - `telegram_synced`
  - `upload_failed`
  - `cloud_only`
  - `cache_missing`
  - `deleted`
- Transfer:
  - `queued`
  - `preparing`
  - `uploading_agent`
  - `thumbnailing`
  - `syncing_telegram`
  - `verifying`
  - `completed`
  - `failed`
  - `paused`
- Sync root:
  - `idle`
  - `scanning`
  - `watching`
  - `paused`
  - `error`
  - `removed`
- System:
  - `agent_online`
  - `agent_offline`
  - `telegram_connected`
  - `telegram_disconnected`
  - `database_ready`
  - `sync_paused`

UI luôn cập nhật qua SSE `GET /v1/events` và polling fallback `GET /v1/transfers`, `GET /v1/sync/roots`.

## Realtime

- Backend phát event SSE:
  - `file.created`
  - `file.updated`
  - `transfer.updated`
  - `syncroot.created`
  - `syncroot.updated`
  - `telegram.connected`
  - `telegram.error`
- Frontend dùng `EventSource` cho:
  - file manager refresh
  - transfer progress live
  - sync roots panel live
- Khi mất kết nối SSE, tự reconnect và polling tạm.

## Lộ trình triển khai UI

1. Hoàn thiện DB reliability và watcher debounce trước (đang làm).
2. Refactor PWA layout giống mock này:
   - sidebar điều hướng
   - top bar search
   - filter chip
   - main grid/list toggle
   - detail panel phải
3. Drag & drop thật:
   - file đơn
   - nhiều file
   - folder
4. Upload queue panel góc dưới phải.
5. Mục `Máy tính` hiển thị sync roots theo thiết bị.
6. Tạo link chia sẻ (UI + backend share).
7. Tray app desktop sau khi UI ổn.
8. Mount ổ ảo cuối cùng.

## Nguyên tắc bảo mật và sản phẩm

- Không hiển thị API ID/API Hash trong giao diện.
- Người dùng chỉ thấy “Kết nối Telegram”.
- Telegram là backend, không phải UI chính.
- Mọi thao tác file đều có metadata local làm nguồn hiển thị.
- Bảo vệ link chia sẻ bằng password/expiry/revoke.
- Không tự xóa file local cho đến khi sync xong.
