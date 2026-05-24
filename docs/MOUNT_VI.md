# Hướng dẫn mount Ổ Đĩa Cloud Ảo

Agent mở sẵn endpoint WebDAV ở:

```
http://<địa chỉ Agent>:8750/webdav
```

Mặc định Agent listen `0.0.0.0:8750`. Nếu chạy trên máy khác, dùng IP LAN/VPS.

## Windows

Cách 1 — Map ổ trong Explorer:

1. Mở Explorer, click chuột phải vào `This PC` → `Map network drive...`
2. Folder: `\\127.0.0.1@8750\webdav` (hoặc `\\192.168.1.198@8750\webdav` cho LAN).
3. Tích `Connect using different credentials` nếu cần. Để trống user/pass.
4. Bấm `Finish`.

Cách 2 — Command line:

```powershell
net use Z: http://127.0.0.1:8750/webdav
```

Lưu ý:
- Windows yêu cầu service `WebClient` đang chạy (`services.msc` → WebClient → Auto).
- Mặc định Windows giới hạn dung lượng file 50MB qua WebDAV. Mở regedit:
  - `HKLM\SYSTEM\CurrentControlSet\Services\WebClient\Parameters\FileSizeLimitInBytes`
  - Đổi sang `4294967295` (≈4GB) hoặc giá trị lớn hơn.
  - Restart service WebClient.

## macOS

1. Mở Finder.
2. `Cmd + K` mở Connect to Server.
3. Nhập `http://127.0.0.1:8750/webdav`.
4. Bấm `Connect`. Chọn `Guest`.

Hoặc dùng Terminal:

```bash
mkdir -p /tmp/td-drive
mount_webdav http://127.0.0.1:8750/webdav /tmp/td-drive
```

## Linux

Cài `davfs2`:

```bash
sudo apt install davfs2
sudo mkdir -p /mnt/td-drive
sudo mount -t davfs http://127.0.0.1:8750/webdav /mnt/td-drive
```

Để mount tự động sau đăng nhập, thêm vào `/etc/fstab` hoặc dùng `~/.config/davfs2/secrets` cho user thường.

## Chế độ hiện tại

- Read-only đầy đủ: list folder, mở file, stream qua Telegram khi cache trống.
- Đổi tên, di chuyển, xóa: hoạt động qua WebDAV.
- Tạo file mới qua mount: chưa hỗ trợ ở phiên bản này, vui lòng dùng PWA hoặc thư mục đồng bộ desktop.

## Khi VPS deploy

- Mount qua domain công khai phải bật HTTPS, ví dụ `https://drive.tencuaban.com/webdav`.
- macOS bắt buộc HTTPS. Windows hỗ trợ HTTP nhưng cần bật `BasicAuthLevel = 2` trong registry.
- Nên đặt sau reverse proxy (Nginx/Caddy) có Basic Auth nếu chia sẻ nhiều người.
