# Ổ Đĩa Cloud Ảo dùng Telegram làm lưu trữ

Cloud drive cá nhân, mã nguồn mở, lấy Telegram làm tầng lưu trữ ẩn phía sau. Người dùng thấy giao diện như Google Drive, không cần biết Telegram là gì.

```text
PWA tiếng Việt + Go Desktop Agent + Gateway domain/Tunnel + Telegram storage
```

## Thành phần

- `agent-go/`: Go agent chạy nền, mở API local (mặc định `127.0.0.1:8750`). Phụ trách đăng nhập Telegram, metadata SQLite, queue đồng bộ Telegram, watcher thư mục desktop, gateway link chia sẻ, Cloudflare Tunnel.
- `web-pwa/`: PWA giao diện tiếng Việt có dấu. Phụ trách upload file/thư mục, quản lý drive, đánh dấu sao, tìm kiếm, đổi tên, di chuyển, thùng rác, link chia sẻ và trang public mở link.
- `docs/`: Kiến trúc, plan UI Google Drive, roadmap.

## Tính năng đã có

- Đăng nhập Telegram bằng số điện thoại + mã (hỗ trợ 2FA).
- Upload nhiều file, kéo thả file/thư mục từ máy, tự tạo cây thư mục.
- Auto sync nền lên Telegram, không cần bấm nút.
- Hash SHA-256 chống upload trùng.
- Fallback: tải lại file từ Telegram khi cache local mất.
- Thumbnail ảnh.
- File manager: đổi tên, di chuyển, đánh dấu sao, thùng rác (xóa/khôi phục), xóa hẳn, tải file, tải thư mục dạng ZIP.
- Search file/folder theo tên.
- Drag & drop nâng cao: thả file ngoài vào folder card, kéo file giữa các thư mục để di chuyển, kéo về root.
- Link chia sẻ:
  - mật khẩu, hết hạn, giới hạn lượt tải, thu hồi
  - chia sẻ file hoặc thư mục (tải về dạng ZIP)
  - trang public `/share/<slug>` đẹp tiếng Việt
- Cấu hình domain chia sẻ:
  - LAN
  - tên miền của bạn
  - Cloudflare Tunnel auto (cần cài `cloudflared`)
- Đồng bộ thư mục desktop: thêm/quét lại/tạm dừng/bật lại/xóa, watcher `fsnotify`.
- Realtime SSE cho file/transfer/sync root/share config.
- PWA installable, dark mode, bottom nav cho mobile.

## Chạy thử nhanh

### Tải bản Windows EXE

Người dùng phổ thông nên tải **`TelegramDriveSetup.exe`** từ **GitHub Releases** — cài 1 file, tự cài WinFsp, có cửa sổ thiết lập kết nối.

- Bản ổn định: `https://github.com/ptadigi/Telegram-Drive-Mount/releases`
- Cài nhanh: `TelegramDriveSetup.exe`
- Portable tray: `td-agent-windows-amd64-tray.exe`
- Build theo source: dùng lệnh ở phần dưới để tạo `td-agent.exe` có tray + mount ổ ảo.

### Xác minh file tải về

Mỗi release kèm `SHA256SUMS.txt`. Kiểm tra file không bị sửa đổi:

```powershell
Get-FileHash .\TelegramDriveSetup.exe -Algorithm SHA256
# So sánh với dòng tương ứng trong SHA256SUMS.txt
```

Lưu ý: bản phát hành hiện chưa ký số bằng chứng chỉ CA, nên Windows SmartScreen có thể báo "Unknown publisher". Bấm **More info → Run anyway** để chạy. Xem `docs/CODE_SIGNING.md`.

### Ứng dụng desktop / sync hai chiều

- Build với `-tags fuse tray` để có tray + mount ổ ảo.
- Cài xong, mở app sẽ hiện cửa sổ thiết lập: chọn nối máy chủ có sẵn (local/VPS) hoặc chạy máy chủ trên máy này, rồi nhập mã ghép thiết bị.
- Với máy chủ/VPS, agent tự mount lại theo cấu hình đã lưu khi khởi động.


Yêu cầu: Go 1.22+, Node.js 20+, file `agent-go/config.local.json` chứa Telegram `api_id` và `api_hash` (lấy tại https://my.telegram.org/apps), không commit lên git.

### 1. Chạy Go Agent

```powershell
cd agent-go
$env:TD_AGENT_CONFIG = 'config.local.json'
go run ./cmd/agent
```

Kiểm tra:

```powershell
Invoke-RestMethod http://127.0.0.1:8750/health
```

### 2. Chạy PWA

```powershell
cd web-pwa
npm install
npm run dev -- --host 0.0.0.0
```

Mở `http://192.168.x.x:5173` trên trình duyệt cùng mạng để dùng từ điện thoại.

## Đăng nhập Telegram

1. Mở PWA, panel `Kết nối Telegram` xuất hiện nếu chưa đăng nhập.
2. Nhập số điện thoại đã đăng ký Telegram.
3. Nhập mã Telegram gửi tới ứng dụng.
4. Nếu tài khoản bật xác minh hai bước, nhập mật khẩu 2FA.

Sau khi kết nối, app hoạt động như cloud drive: upload xong tự đồng bộ lên Telegram nền.

## Cấu hình link chia sẻ

Trong PWA mở `Cài đặt`:

- LAN: link chỉ mở trong mạng nội bộ, không cần cấu hình thêm.
- Tên miền của tôi: nhập domain trỏ về máy chạy agent. App tự kiểm tra `/.td-check`.
- Cloudflare Tunnel: bật một phát có ngay link public `*.trycloudflare.com`. Yêu cầu cài `cloudflared` trong PATH.

## Đồng bộ thư mục desktop

1. Mở mục `Máy tính` trong PWA.
2. Bấm `Thêm thư mục`, nhập đường dẫn local.
3. Agent quét và đẩy file lên cloud, sau đó watch nền để đồng bộ file mới.

## Cấu trúc dữ liệu

- Metadata lưu trong SQLite tại thư mục dữ liệu của Agent.
- File local cache trong `dataDir/uploads/`.
- Thumbnail trong `dataDir/thumbs/`.
- Session Telegram là file local, không gửi đi đâu.

## Định hướng

- Không dùng Tauri.
- Telegram là object storage, không phải data model chính.
- Mọi giao diện và thông báo dùng tiếng Việt có dấu.
- Mã nguồn mở để mỗi cá nhân/tổ chức self-host được.

## Mốc tiếp theo

- ✅ Tray app desktop + auto start (đã có).
- ✅ Storage channel/chat Telegram chuyên dụng (auto-tạo).
- ✅ QR login Telegram.
- ✅ Mount ổ ảo native (WinFsp/FUSE) qua build `-tags fuse`.
- ✅ Recycle Bin OS khi evict cache.
- Sync hai chiều và conflict resolver.
- Stream video range tối ưu.
- Office viewer cho docx/xlsx/pptx khi có domain public.
- Installer Windows MSIX/Inno Setup.

Đóng góp và issue luôn được hoan nghênh.
