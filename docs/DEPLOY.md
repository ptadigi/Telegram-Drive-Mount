# Hướng dẫn deploy Ổ Đĩa Cloud Ảo trên VPS

Tài liệu này dành cho self-host trên server (Linux/VPS). Trên desktop, dùng tray app hoặc `go run ./cmd/agent --tray` là đủ.

## Yêu cầu

- Linux x86_64 hoặc ARM64.
- Đã cài Docker + docker-compose, hoặc Go 1.23+ và Node 20+ nếu chạy trực tiếp.
- Đã có Telegram API ID/API Hash (https://my.telegram.org/apps).

## Phương án 1: Docker compose (đơn giản nhất)

```bash
git clone <repo-of-this-project>
cd Telegram-Drive
cp deploy/config/config.example.json deploy/config/config.json
nano deploy/config/config.json   # điền api_id, api_hash, password basic auth
docker compose -f deploy/docker-compose.yml up -d
```

Truy cập:
- API Agent: `http://<ip-vps>:8750`
- PWA: `http://<ip-vps>:5173`
- WebDAV: `http://<ip-vps>:8750/webdav`

Lần đầu mở PWA, đăng nhập Telegram bằng số điện thoại. Session lưu cố định trong volume `td-agent-data`.

## Phương án 2: Build binary chạy trực tiếp

```bash
cd agent-go
CGO_ENABLED=0 go build -o td-agent ./cmd/agent
sudo install -m 0755 td-agent /usr/local/bin/td-agent

sudo useradd --system --home /var/lib/td-agent --shell /usr/sbin/nologin td-agent
sudo install -d -o td-agent -g td-agent /var/lib/td-agent
sudo install -d /etc/td-agent
sudo cp deploy/config/config.example.json /etc/td-agent/config.json
sudo nano /etc/td-agent/config.json   # điền cấu hình thật

sudo cp deploy/td-agent.service /etc/systemd/system/td-agent.service
sudo systemctl daemon-reload
sudo systemctl enable --now td-agent
sudo journalctl -u td-agent -f
```

## Phương án 3: Cloudflare Tunnel cho domain công khai

Khi VPS không có IP public hoặc không muốn lo TLS:

```bash
sudo apt install cloudflared
cloudflared login
cloudflared tunnel create td-agent
cloudflared tunnel route dns td-agent drive.tencuaban.com
cloudflared tunnel run td-agent
```

Hoặc bật trực tiếp trong PWA Settings → Cấu hình chia sẻ → Cloudflare Tunnel auto.

## Bảo mật khi deploy public

- Bật `auth.mode = basic` trong `config.json`. Đặt mật khẩu mạnh.
- Đặt agent sau Nginx/Caddy có HTTPS:

```nginx
server {
  listen 443 ssl;
  server_name drive.tencuaban.com;
  ssl_certificate /etc/letsencrypt/live/drive.tencuaban.com/fullchain.pem;
  ssl_certificate_key /etc/letsencrypt/live/drive.tencuaban.com/privkey.pem;
  client_max_body_size 5G;
  proxy_request_buffering off;

  location / {
    proxy_pass http://127.0.0.1:8750;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $remote_addr;
  }
}
```

- Caddy:

```caddy
drive.tencuaban.com {
  reverse_proxy 127.0.0.1:8750
}
```

## Upload file lớn bị treo ở vài % (rất hay gặp)

Triệu chứng: upload file nhỏ bình thường, nhưng file lớn (video, >50MB) treo ở vài %
rồi đứng. Trong `sync.log` KHÔNG có dòng `upload_received` cho file đó.

Nguyên nhân: reverse proxy chặn body request vượt giới hạn `client_max_body_size`,
file không bao giờ tới được agent. Đây KHÔNG phải lỗi agent.

Bắt buộc cấu hình ở reverse proxy:

- **Nginx/OpenResty**: trong `server { }` của site:

```nginx
client_max_body_size 0;          # 0 = không giới hạn, hoặc đặt số lớn ví dụ 5g
proxy_request_buffering off;     # stream thẳng lên agent, không buffer cả file vào proxy
proxy_read_timeout 3600s;
proxy_send_timeout 3600s;
```

- **1Panel / aaPanel (OpenResty trong Docker)**: mặc định `client_max_body_size 50m`
  trong `nginx.conf` global. Thêm 4 dòng trên vào server block của site (file
  `conf.d/<domain>.conf`), rồi reload: `docker exec <openresty-container> nginx -s reload`.
- **Cloudflare (Free)**: giới hạn cứng 100MB/request, không nâng được trừ Enterprise.
  File >100MB phải bypass Cloudflare (DNS-only) hoặc chờ tính năng chunked upload.

## SSE realtime sau proxy

Nếu danh sách file không tự cập nhật khi có thay đổi, proxy đang buffer Server-Sent
Events. Đảm bảo không buffer `/v1/events`:

```nginx
location /v1/events {
  proxy_pass http://127.0.0.1:8750;
  proxy_buffering off;
  proxy_cache off;
}
```

Agent đã gửi `X-Accel-Buffering: no` + heartbeat, và PWA có cơ chế revalidate khi
focus/visible/online + poll dự phòng, nên kể cả proxy chặn SSE thì UI vẫn cập nhật
trong vài chục giây.

## Cache cục bộ và dung lượng đĩa

Telegram là nơi lưu trữ thật; cache cục bộ chỉ là bản nóng để xem nhanh.

- Hạ cache xuống <1GB **an toàn, không mất file**: chỉ file đã `telegram_synced` mới
  bị dọn; file đang chờ/đang sync/lỗi không bao giờ bị xóa.
- Đánh đổi: cache càng nhỏ, xem lại file cũ càng hay phải kéo lại từ Telegram (chậm
  hơn, tốn băng thông, dễ chạm FLOOD_WAIT nếu nhiều người dùng).
- Lưu ý: file đang upload tạm trú trong `<data_dir>/uploads/` trước khi sync — đĩa VPS
  vẫn cần đủ chỗ cho các file đang chờ, cache nhỏ không chặn phần này.
- Chế độ cache: `smart` (giữ file nóng dưới ngưỡng, evict LRU — khuyến nghị),
  `cloud_only` (sync xong xóa ngay), `mirror` (giữ tất cả, không evict).

## Backup & restore

- Agent tự backup `metadata.db` mỗi 6 giờ vào `<data_dir>/backups/`. Giữ 14 file gần nhất.
- Để restore, dừng Agent, copy file backup về `<data_dir>/metadata.db`, khởi động lại.

```bash
sudo systemctl stop td-agent
sudo cp /var/lib/td-agent/backups/metadata-20260524-120000.db /var/lib/td-agent/metadata.db
sudo systemctl start td-agent
```

## Cập nhật

```bash
git pull
cd agent-go && go build -o td-agent ./cmd/agent
sudo install -m 0755 td-agent /usr/local/bin/td-agent
sudo systemctl restart td-agent
```

## Audit log

Tất cả thay đổi cấu hình quan trọng (storage, share config, cache) đều được ghi vào bảng `audit_log` trong SQLite. Truy vấn:

```bash
sqlite3 /var/lib/td-agent/metadata.db "SELECT ts, actor, action, target_kind, target_id FROM audit_log ORDER BY id DESC LIMIT 50"
```

Hoặc dùng API: `GET /v1/audit?limit=50`.

## Mount ổ ảo

Tham khảo `docs/MOUNT_VI.md`. WebDAV chạy ngay tại `/webdav` trên cùng port Agent.
