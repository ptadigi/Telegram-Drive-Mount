# Hướng dẫn cài đặt — Ổ Đĩa Cloud Ảo (Telegram Drive)

Tài liệu này hướng dẫn cài đặt cho **mọi môi trường**: Windows, Linux (Ubuntu/Debian/CentOS), VPS/cPanel, và Docker. Mỗi môi trường đều có **phương án cài nhanh 1 lệnh** (one-click) và phương án thủ công.

> Repo: https://github.com/ptadigi/Telegram-Drive-Mount
> Tải bản mới nhất: https://github.com/ptadigi/Telegram-Drive-Mount/releases/latest

---

## 0. Hai chế độ sử dụng

| Chế độ | Dùng cho | Chạy gì |
|--------|----------|---------|
| **Máy chủ (server)** | VPS/máy luôn bật, lưu session Telegram + cache, phục vụ PWA | `td-agent` đầy đủ |
| **Máy khách desktop** | Máy Windows cá nhân, mount ổ `T:` | App desktop (`TelegramDriveSetup.exe`), nối tới server qua URL + mã ghép |

Bạn có thể chỉ chạy **server** (dùng qua web PWA), hoặc thêm **desktop** để có ổ đĩa ảo trên máy tính.

**Cần trước khi cài:** `api_id` và `api_hash` của Telegram — lấy miễn phí tại https://my.telegram.org → *API development tools*.

---

## 1. Windows — cài nhanh 1-click

### Cách A — Một lệnh PowerShell (khuyến nghị)
Mở **PowerShell (Run as Administrator)** và dán:

```powershell
irm https://raw.githubusercontent.com/ptadigi/Telegram-Drive-Mount/main/deploy/install.ps1 | iex
```

Lệnh này tự: tải `TelegramDriveSetup.exe`, xác minh SHA256, rồi chạy trình cài đặt (đã gói sẵn **WinFsp**, giao diện PWA và icon khay hệ thống).

### Cách B — Tải installer thủ công
1. Vào trang [Releases](https://github.com/ptadigi/Telegram-Drive-Mount/releases/latest).
2. Tải `TelegramDriveSetup.exe`.
3. (Tùy chọn) Xác minh checksum:
   ```powershell
   certutil -hashfile TelegramDriveSetup.exe SHA256
   ```
   So với dòng tương ứng trong `SHA256SUMS.txt`.
4. Chạy file, làm theo trình cài.

### Sau khi cài
- App chạy ở **khay hệ thống (tray)**. Chuột phải icon → **Mở giao diện**.
- Ổ ảo mount tại **`T:` (Telegram Drive)**.
- Lần đầu mở: chọn
  - **Chạy máy chủ local** (máy này tự lưu trữ), hoặc
  - **Nối máy chủ có sẵn** → nhập URL server (vd `https://drive.tencuaban.com`) + dán **mã ghép** tạo từ PWA.

> SmartScreen có thể hiện "Unknown publisher" (bản tự ký). Bấm *More info → Run anyway*. Xem [CODE_SIGNING.md](CODE_SIGNING.md).

---

## 2. Linux (VPS / Ubuntu / Debian / CentOS) — cài nhanh 1-click

### Cách A — Một lệnh (binary + systemd)
```bash
curl -fsSL https://raw.githubusercontent.com/ptadigi/Telegram-Drive-Mount/main/deploy/install.sh | sudo bash
```

Script sẽ:
- Tải agent Linux mới nhất về `/usr/local/bin/td-agent`.
- Cài (best-effort) `ffmpeg` + `poppler-utils` cho thumbnail video/PDF.
- Hỏi `api_id`, `api_hash`, mật khẩu Basic Auth (có thể bỏ qua, điền sau).
- Sinh khóa mã hóa session (`TD_AGENT_SESSION_KEY`), tạo service `td-agent`, bật chạy nền.

Cài không tương tác (CI/script):
```bash
TD_API_ID=12345 TD_API_HASH=abcd... TD_ADMIN_PASSWORD=secret \
  bash -c "curl -fsSL https://raw.githubusercontent.com/ptadigi/Telegram-Drive-Mount/main/deploy/install.sh | sudo -E bash"
```

Sau khi xong: mở `http://<ip-vps>:8750`, tạo tài khoản quản trị, đăng nhập Telegram.

Lệnh quản trị:
```bash
systemctl status td-agent
journalctl -u td-agent -f
systemctl restart td-agent
```

### Cách B — Build thủ công từ mã nguồn
Yêu cầu Go 1.23+.
```bash
git clone https://github.com/ptadigi/Telegram-Drive-Mount.git
cd Telegram-Drive-Mount/agent-go
CGO_ENABLED=0 go build -o td-agent ./cmd/agent
sudo install -m 0755 td-agent /usr/local/bin/td-agent
sudo useradd --system --home /var/lib/td-agent --shell /usr/sbin/nologin td-agent
sudo install -d -o td-agent -g td-agent /var/lib/td-agent
sudo install -d /etc/td-agent
sudo cp ../deploy/config/config.example.json /etc/td-agent/config.json
sudo nano /etc/td-agent/config.json     # điền api_id, api_hash, password
sudo cp ../deploy/td-agent.service /etc/systemd/system/td-agent.service
sudo systemctl daemon-reload
sudo systemctl enable --now td-agent
```

---

## 3. Docker (mọi hệ điều hành có Docker) — cài nhanh 1-click

```bash
curl -fsSL https://raw.githubusercontent.com/ptadigi/Telegram-Drive-Mount/main/deploy/install-docker.sh | bash
```

Script clone mã nguồn, hỏi cấu hình, rồi `docker compose up -d`. Mặc định:
- Agent: `http://<ip>:8750`
- PWA: `http://<ip>:5173`

Thủ công:
```bash
git clone https://github.com/ptadigi/Telegram-Drive-Mount.git
cd Telegram-Drive-Mount
cp deploy/config/config.example.json deploy/config/config.json
nano deploy/config/config.json          # điền api_id, api_hash, password
docker compose -f deploy/docker-compose.yml up -d
docker compose -f deploy/docker-compose.yml logs -f
```

---

## 4. cPanel / aaPanel / Plesk / 1Panel

Các panel này thường là **VPS có toàn quyền root** với một reverse proxy (OpenResty/Nginx/Apache) phía trước. Cách làm:

1. **Cài agent** bằng [Cách A mục 2](#2-linux-vps--ubuntu--debian--centos--cài-nhanh-1-click) (SSH vào VPS, chạy 1 lệnh). Agent nghe ở `127.0.0.1:8750`.
   - Nếu panel có Docker, dùng [mục 3](#3-docker-mọi-hệ-điều-hành-có-docker--cài-nhanh-1-click).
2. **Trỏ tên miền/subdomain** về VPS (bản ghi A).
3. **Tạo reverse proxy** trỏ `https://drive.tencuaban.com` → `http://127.0.0.1:8750`.

> ⚠️ **Bắt buộc cho upload file lớn.** Reverse proxy mặc định của 1Panel/aaPanel giới hạn `client_max_body_size 50m` → upload >50MB sẽ treo. Thêm vào server block của site:
> ```nginx
> client_max_body_size 0;          # 0 = không giới hạn (hoặc đặt 5g)
> proxy_request_buffering off;     # stream thẳng, không buffer cả file
> proxy_read_timeout 3600s;
> proxy_send_timeout 3600s;
> ```
> rồi `nginx -s reload`. Nếu dùng Cloudflare proxy (cam): bản Free chặn cứng 100MB/request — tắt proxy (DNS-only) cho subdomain này nếu cần upload >100MB.

> Nếu chỉ có **shared hosting cPanel** (không root, không systemd/Docker), **không** cài được agent ở đó. Hãy dùng một VPS nhỏ cho agent, còn cPanel chỉ trỏ domain.

---

## 5. Đặt domain + HTTPS (khuyến nghị cho production)

**Nginx:**
```nginx
server {
  listen 443 ssl;
  server_name drive.tencuaban.com;
  ssl_certificate     /etc/letsencrypt/live/drive.tencuaban.com/fullchain.pem;
  ssl_certificate_key /etc/letsencrypt/live/drive.tencuaban.com/privkey.pem;

  client_max_body_size 0;
  proxy_request_buffering off;
  proxy_read_timeout 3600s;

  location / {
    proxy_pass http://127.0.0.1:8750;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $remote_addr;
    proxy_set_header X-Forwarded-Proto https;
  }
  location /v1/events {           # realtime SSE — không buffer
    proxy_pass http://127.0.0.1:8750;
    proxy_buffering off;
    proxy_cache off;
  }
}
```

**Caddy (tự lo HTTPS):**
```caddy
drive.tencuaban.com {
  reverse_proxy 127.0.0.1:8750
}
```

---

## 6. Cấu hình (`config.json`)

```json
{
  "host": "0.0.0.0",
  "port": 8750,
  "data_dir": "/var/lib/td-agent",
  "telegram": { "api_id": 0, "api_hash": "" },
  "auth": { "mode": "basic", "username": "admin", "password": "doi-mat-khau" },
  "cache": { "mode": "smart", "max_bytes": 10737418240 }
}
```

| Trường | Ý nghĩa |
|--------|---------|
| `telegram.api_id` / `api_hash` | Lấy tại my.telegram.org. Bắt buộc để đăng nhập Telegram. |
| `auth.mode` | `open` (chỉ dùng tài khoản PWA) hoặc `basic` (thêm 1 lớp HTTP Basic Auth). |
| `cache.mode` | `smart` (LRU, khuyến nghị) / `cloud_only` (xóa sau khi sync) / `mirror` (giữ tất cả). |
| `cache.max_bytes` | Dung lượng cache local tối đa (byte). 10 GiB ở ví dụ trên. |

**Biến môi trường hữu ích:** `TD_AGENT_CONFIG` (đường dẫn config), `TD_AGENT_DATA_DIR`, `TD_AGENT_PORT`, `TD_AGENT_SESSION_KEY` (khóa AES-256 mã hóa session ở dạng nghỉ — installer/script tự sinh).

---

## 7. Thumbnail video & PDF (tùy chọn)

Ảnh luôn có thumbnail. Để có thumbnail **video/PDF**, máy chạy agent cần:
```bash
# Debian/Ubuntu
sudo apt-get install -y ffmpeg poppler-utils
```
Thiếu cũng không sao — file vẫn upload bình thường, chỉ hiển thị icon thay ảnh xem trước.

---

## 8. Gỡ cài đặt

**Windows:** Settings → Apps → *Ổ Đĩa Cloud Ảo* → Uninstall (hoặc chạy `unins000.exe` trong thư mục cài).

**Linux:**
```bash
sudo systemctl disable --now td-agent
sudo rm /etc/systemd/system/td-agent.service /usr/local/bin/td-agent
sudo rm -rf /etc/td-agent /var/lib/td-agent      # XÓA cả dữ liệu/cache
sudo userdel td-agent
```

**Docker:**
```bash
docker compose -f deploy/docker-compose.yml down -v
```

---

Gặp lỗi? Xem [TROUBLESHOOTING.md](TROUBLESHOOTING.md) (đã tổng hợp các sự cố thực tế: upload file lớn, SSE behind proxy, mount, thumbnail, share…).
