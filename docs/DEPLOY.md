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
