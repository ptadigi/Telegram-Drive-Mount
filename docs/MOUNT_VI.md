# Hướng dẫn mount Ổ Đĩa Cloud Ảo

Có 2 chế độ mount hỗ trợ chính thức:

1. Native mount (production) - dùng WinFsp/FUSE qua cgofuse.
2. WebDAV (legacy/dev) - dùng cho khi không có WinFsp/FUSE.

## 1. Native mount qua FUSE/WinFsp

### Build agent có mount

```powershell
# Windows
choco install winfsp -y
cd agent-go
go build -tags fuse -o td-agent.exe ./cmd/agent
```

```bash
# macOS (FUSE-T khuyên dùng)
brew tap macos-fuse-t/cask
brew install --cask fuse-t
cd agent-go
go build -tags fuse -o td-agent ./cmd/agent
```

```bash
# Linux
sudo apt-get install -y libfuse-dev pkg-config
cd agent-go
go build -tags fuse -o td-agent ./cmd/agent
```

### Mount/unmount qua PWA

1. Mở `Cài đặt > Ổ ảo Telegram Drive`.
2. Chọn drive letter / mount point (mặc định `T:` trên Windows, `/Volumes/Telegram Drive` trên macOS, `/tmp/telegram-drive` trên Linux).
3. Bấm `Mount ổ ảo`.
4. Khi xong việc bấm `Unmount`.

### Mount/unmount qua API

```bash
curl http://127.0.0.1:8750/v1/mount        # status
curl -X POST http://127.0.0.1:8750/v1/mount -d '{"mount_point":"T:"}' -H 'Content-Type: application/json'
curl -X DELETE http://127.0.0.1:8750/v1/mount
```

### Tray menu

Khi chạy `td-agent --tray`, menu có sẵn `Mount ổ ảo` / `Unmount ổ ảo`.

## 2. WebDAV (legacy)

Agent vẫn mở endpoint:

```
http://<địa chỉ Agent>:8750/webdav
```

Cách map ổ qua WebDAV xem phần legacy bên dưới. Khuyên dùng native mount khi có sẵn.

### Windows WebDAV

```powershell
net use Z: http://127.0.0.1:8750/webdav
```

Yêu cầu service `WebClient` đang chạy. Mặc định Windows giới hạn file 50MB qua WebDAV; chỉnh registry `HKLM\SYSTEM\CurrentControlSet\Services\WebClient\Parameters\FileSizeLimitInBytes` lên `4294967295` rồi restart `WebClient`.

### macOS WebDAV

```bash
mkdir -p /tmp/td-drive
mount_webdav http://127.0.0.1:8750/webdav /tmp/td-drive
```

### Linux WebDAV

```bash
sudo apt install davfs2
sudo mkdir -p /mnt/td-drive
sudo mount -t davfs http://127.0.0.1:8750/webdav /mnt/td-drive
```

## Recycle Bin OS

- Khi cache local bị evict, Agent ưu tiên đẩy vào Recycle Bin/Trash hệ thống. Nếu không khả dụng (CI/headless) sẽ fallback xóa cứng.
- File trong sync folder không bao giờ bị xóa tự động.

## Khi VPS deploy

- Mount qua domain công khai phải bật HTTPS.
- macOS bắt buộc HTTPS. Windows hỗ trợ HTTP nhưng cần `BasicAuthLevel = 2` registry.
- Khi expose ra internet, bật Basic Auth ở `Settings > Bảo mật API`.
