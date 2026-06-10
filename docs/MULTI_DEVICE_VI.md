# Đa thiết bị / Multi-device

> Tiếng Việt trước, English below.

Mô hình: 1 máy chủ (VPS hoặc 1 PC) chạy `td-agent` đầy đủ — giữ Telegram session, metadata SQLite, cache. Các máy còn lại chạy `td-agent --remote` ở chế độ thin-client: không có Telegram session, không có DB, chỉ mount ổ ảo và gọi máy chủ qua HTTPS.

`
        Telegram (kho lưu trữ thật)
              | MTProto (session mã hoá AES-256-GCM)
        Máy chủ td-agent  (VPS hoặc PC chính)
        - PWA + REST API + chunk cache + coalesce
              | HTTPS + Device token
   PC2 --remote--+--remote-- PC3 ... + PWA trên mọi trình duyệt
`

---

## Tiếng Việt

### 1. Chuẩn bị máy chủ

`powershell
# Sinh khoá mã hoá session (giữ cố định, mất khoá = phải login Telegram lại)
# Dung: openssl rand -hex 32   (hoac bat ky chuoi 64 ky tu hex)
$env:TD_AGENT_SESSION_KEY = "<64-ky-tu-hex>"

# Chạy máy chủ (build kèm mount: go build -tags 'fuse tray')
.\td-agent.exe --config config.local.json
`

Mở PWA, đăng nhập, kết nối Telegram (QR hoặc số điện thoại).

### 2. Tạo mã ghép thiết bị

- Trong PWA: **Thiết bị đã ghép** → **Tạo mã ghép thiết bị**.
- Mã dạng `4F2A-9K2X`, hiệu lực 5 phút, dùng 1 lần.

### 3. Ghép máy con

> Người dùng Windows có thể tải `td-agent.exe` sẵn từ GitHub Releases để test ngay. Nếu muốn mount ổ ảo + tray, dùng bản build `go build -tags 'fuse tray'`.


`powershell
# Trên máy con (đã cài WinFsp/FUSE + td-agent)
.\td-agent.exe --pair --pair-url https://drive.tencuaban.com --pair-code 4F2A-9K2X
# Token lưu vào %APPDATA%\TelegramVirtualDrive\agent-client\token.json (chmod 0600 trên *nix)
`

### 4. Mount ổ ảo trên máy con

`powershell
.\td-agent.exe --remote --remote-mount T:
`

Máy con thấy đúng cây thư mục như máy chủ. Mở file = tải qua máy chủ (stream từ Telegram, có chunk cache). Tạo/sửa/xoá file trong `T:` đồng bộ ngược lên máy chủ.

### 5. Thu hồi thiết bị

PWA → **Thiết bị đã ghép** → **Thu hồi**. Token client bị vô hiệu ngay.

### Lưu ý bảo mật

- Production bắt buộc HTTPS. Chỉ dev local mới đặt `TD_AGENT_INSECURE=1`.
- `TD_AGENT_SESSION_KEY` phải cố định giữa các lần khởi động máy chủ.
- Token thiết bị lưu dạng SHA-256 hash trong DB, không lưu plaintext.
- Mỗi user/VPS dùng tài khoản Telegram riêng của họ.

---

## English

### 1. Server

`ash
export TD_AGENT_SESSION_KEY="$(openssl rand -hex 32)"   # keep this stable
./td-agent --config config.local.json                   # build with -tags 'fuse tray'
`

Open the PWA, sign in, connect Telegram (QR or phone).

### 2. Generate a pairing code

PWA → **Paired devices** → **Generate pairing code**. Code like `4F2A-9K2X`, valid 5 minutes, single use.

### 3. Pair a client machine

`ash
./td-agent --pair --pair-url https://drive.example.com --pair-code 4F2A-9K2X
# token saved under the OS config dir, file mode 0600
`

### 4. Mount the virtual drive on the client

`ash
./td-agent --remote --remote-mount T:          # Windows
./td-agent --remote --remote-mount ~/TGDrive   # Linux/macOS
`

The client sees the same tree as the server. Reads stream through the server (chunk-cached); writes sync back up.

### 5. Revoke a device

PWA → **Paired devices** → **Revoke**. The client token is invalidated immediately.

### Security notes

- HTTPS is mandatory in production; `TD_AGENT_INSECURE=1` is dev-only.
- Keep `TD_AGENT_SESSION_KEY` stable across server restarts.
- Device tokens are stored as SHA-256 hashes, never plaintext.
- Each user/VPS uses their own Telegram account.

---

## Cache & performance

- Server keeps a 1 MB-aligned chunk cache on disk (default budget 5 GB, LRU).
- Concurrent reads of the same range coalesce into a single Telegram fetch (avoids FLOOD_WAIT).
- Large cold files stream through without caching the whole file.
