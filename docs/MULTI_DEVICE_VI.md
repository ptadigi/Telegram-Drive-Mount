# Äa thiáº¿t bá»‹ / Multi-device

> Tiáº¿ng Viá»‡t trÆ°á»›c, English below.

MÃ´ hÃ¬nh: 1 mÃ¡y chá»§ (VPS hoáº·c 1 PC) cháº¡y `td-agent` Ä‘áº§y Ä‘á»§ â€” giá»¯ Telegram session, metadata SQLite, cache. CÃ¡c mÃ¡y cÃ²n láº¡i cháº¡y `td-agent --remote` á»Ÿ cháº¿ Ä‘á»™ thin-client: khÃ´ng cÃ³ Telegram session, khÃ´ng cÃ³ DB, chá»‰ mount á»• áº£o vÃ  gá»i mÃ¡y chá»§ qua HTTPS.

`
        Telegram (kho lÆ°u trá»¯ tháº­t)
              | MTProto (session mÃ£ hoÃ¡ AES-256-GCM)
        MÃ¡y chá»§ td-agent  (VPS hoáº·c PC chÃ­nh)
        - PWA + REST API + chunk cache + coalesce
              | HTTPS + Device token
   PC2 --remote--+--remote-- PC3 ... + PWA trÃªn má»i trÃ¬nh duyá»‡t
`

---

## Tiáº¿ng Viá»‡t

### 1. Chuáº©n bá»‹ mÃ¡y chá»§

`powershell
# Sinh khoÃ¡ mÃ£ hoÃ¡ session (giá»¯ cá»‘ Ä‘á»‹nh, máº¥t khoÃ¡ = pháº£i login Telegram láº¡i)
# Dung: openssl rand -hex 32   (hoac bat ky chuoi 64 ky tu hex)
$env:TD_AGENT_SESSION_KEY = "<64-ky-tu-hex>"

# Cháº¡y mÃ¡y chá»§ (build kÃ¨m mount: go build -tags 'fuse tray')
.\td-agent.exe --config config.local.json
`

Má»Ÿ PWA, Ä‘Äƒng nháº­p, káº¿t ná»‘i Telegram (QR hoáº·c sá»‘ Ä‘iá»‡n thoáº¡i).

### 2. Táº¡o mÃ£ ghÃ©p thiáº¿t bá»‹

- Trong PWA: **Thiáº¿t bá»‹ Ä‘Ã£ ghÃ©p** â†’ **Táº¡o mÃ£ ghÃ©p thiáº¿t bá»‹**.
- MÃ£ dáº¡ng `4F2A-9K2X`, hiá»‡u lá»±c 5 phÃºt, dÃ¹ng 1 láº§n.

### 3. GhÃ©p mÃ¡y con

`powershell
# TrÃªn mÃ¡y con (Ä‘Ã£ cÃ i WinFsp/FUSE + td-agent)
.\td-agent.exe --pair --pair-url https://drive.tencuaban.com --pair-code 4F2A-9K2X
# Token lÆ°u vÃ o %APPDATA%\TelegramVirtualDrive\agent-client\token.json (chmod 0600 trÃªn *nix)
`

### 4. Mount á»• áº£o trÃªn mÃ¡y con

`powershell
.\td-agent.exe --remote --remote-mount T:
`

MÃ¡y con tháº¥y Ä‘Ãºng cÃ¢y thÆ° má»¥c nhÆ° mÃ¡y chá»§. Má»Ÿ file = táº£i qua mÃ¡y chá»§ (stream tá»« Telegram, cÃ³ chunk cache). Táº¡o/sá»­a/xoÃ¡ file trong `T:` Ä‘á»“ng bá»™ ngÆ°á»£c lÃªn mÃ¡y chá»§.

### 5. Thu há»“i thiáº¿t bá»‹

PWA â†’ **Thiáº¿t bá»‹ Ä‘Ã£ ghÃ©p** â†’ **Thu há»“i**. Token client bá»‹ vÃ´ hiá»‡u ngay.

### LÆ°u Ã½ báº£o máº­t

- Production báº¯t buá»™c HTTPS. Chá»‰ dev local má»›i Ä‘áº·t `TD_AGENT_INSECURE=1`.
- `TD_AGENT_SESSION_KEY` pháº£i cá»‘ Ä‘á»‹nh giá»¯a cÃ¡c láº§n khá»Ÿi Ä‘á»™ng mÃ¡y chá»§.
- Token thiáº¿t bá»‹ lÆ°u dáº¡ng SHA-256 hash trong DB, khÃ´ng lÆ°u plaintext.
- Má»—i user/VPS dÃ¹ng tÃ i khoáº£n Telegram riÃªng cá»§a há».

---

## English

### 1. Server

`ash
export TD_AGENT_SESSION_KEY="$(openssl rand -hex 32)"   # keep this stable
./td-agent --config config.local.json                   # build with -tags 'fuse tray'
`

Open the PWA, sign in, connect Telegram (QR or phone).

### 2. Generate a pairing code

PWA â†’ **Paired devices** â†’ **Generate pairing code**. Code like `4F2A-9K2X`, valid 5 minutes, single use.

### 3. Pair a client machine

`ash
./td-agent --pair --pair-url https://drive.example.com --pair-code 4F2A-9K2X
# token saved under the OS config dir, file mode 0600
`

### 4. Mount the virtual drive on the client

`ash
./td-agent --remote --remote-mount T:          # Windows
./td-agent --remote --remote-mount ~/TGDrive   # Linux/macOS
`

The client sees the same tree as the server. Reads stream through the server (chunk-cached); writes sync back up.

### 5. Revoke a device

PWA â†’ **Paired devices** â†’ **Revoke**. The client token is invalidated immediately.

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
