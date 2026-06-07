# Contributing / Đóng góp

> **Tiếng Việt:** Cảm ơn anh/chị đã muốn đóng góp cho Telegram Drive! Đây là project mã nguồn mở miễn phí, phát triển theo tinh thần cộng đồng.
>
> **English:** Thanks for your interest in contributing to Telegram Drive! This is a free, open-source project driven by the community.

---

## English

### Quick start

```bash
# Backend
cd agent-go
go test ./...
go build ./...

# Frontend
cd web-pwa
npm install
npm run build
```

### Project layout

```
agent-go/      Go backend, REST + WebDAV + WinFsp/FUSE mount + Telegram MTProto
web-pwa/       React + Vite PWA, Vietnamese-first, English support
docs/          Architecture, mount, deploy, installer
.github/       CI workflows + funding links
```

### Workflow

1. Fork the repo.
2. Create a branch from `main`: `feature/<topic>` or `fix/<topic>` or `security/<topic>`.
3. Build + test before pushing:
   - `go build ./...` + `go test ./...` (backend)
   - `npm run build` (frontend)
4. Open a Pull Request with:
   - What changed.
   - Why.
   - Smoke test or screenshots if UX.
5. Keep messages concise; prefer Vietnamese OR English consistently inside one PR.

### Reporting bugs

- Open a GitHub issue with steps to reproduce, OS, version of agent (`td-agent --version` if available), and logs.
- Do **not** include `api_id`, `api_hash`, Telegram phone number, or session files.

### Security

- Found a security bug? Email the maintainer or open a private security advisory on GitHub. Do not file a public issue.
- This project takes auth/auth and tenant isolation seriously. See `docs/` for the bug-hunt rounds.

### Coding rules

- Go: gofmt clean, no `panic` outside `main`, no `log.Fatalf` in shutdown paths.
- TypeScript/React: prefer functional components + hooks, avoid `any`, follow existing style in `web-pwa/src`.
- All UI strings must go through i18n (`vi.json` + `en.json`).
- Backend errors should be Vietnamese-friendly when shown to end users; technical messages in logs can be English.

---

## Tiếng Việt

### Bắt đầu nhanh

```bash
# Backend
cd agent-go
go test ./...
go build ./...

# Frontend
cd web-pwa
npm install
npm run build
```

### Cấu trúc dự án

```
agent-go/      Go backend, REST + WebDAV + mount WinFsp/FUSE + Telegram MTProto
web-pwa/       React + Vite PWA, ưu tiên tiếng Việt, hỗ trợ tiếng Anh
docs/          Kiến trúc, mount, deploy, installer
.github/       CI workflows + funding links
```

### Quy trình

1. Fork repo.
2. Tạo branch từ `main`: `feature/<chủ-đề>`, `fix/<chủ-đề>`, hoặc `security/<chủ-đề>`.
3. Build + test trước khi push:
   - `go build ./...` + `go test ./...` (backend)
   - `npm run build` (frontend)
4. Mở Pull Request, ghi rõ:
   - Thay đổi gì.
   - Lý do.
   - Cách smoke test hoặc ảnh chụp nếu là UI.
5. Trong cùng 1 PR, dùng nhất quán tiếng Việt hoặc tiếng Anh.

### Báo lỗi

- Mở issue trên GitHub kèm các bước tái hiện, hệ điều hành, phiên bản agent, log.
- **KHÔNG** đính kèm `api_id`, `api_hash`, số điện thoại Telegram, hoặc file session.

### An toàn / bảo mật

- Tìm thấy lỗi bảo mật? Nhắn riêng cho maintainer hoặc mở Security Advisory trên GitHub. Không mở issue công khai.
- Dự án rất quan tâm auth/auth và cách ly đa người dùng. Xem `docs/` cho các vòng bug-hunt đã làm.

### Quy ước code

- Go: gofmt sạch, không `panic` ngoài `main`, không `log.Fatalf` trong shutdown.
- TypeScript/React: ưu tiên functional + hooks, tránh `any`, theo style đang có trong `web-pwa/src`.
- Mọi chuỗi UI phải đi qua i18n (`vi.json` + `en.json`).
- Lỗi backend hiển thị cho user nên tiếng Việt thân thiện; log kỹ thuật có thể tiếng Anh.

---

## Donate / Ủng hộ

If this project saved you time and money, consider sponsoring development. Donate links live in `.github/FUNDING.yml`.

Nếu project giúp được anh/chị, có thể ủng hộ tác giả qua các kênh trong `.github/FUNDING.yml`. Cảm ơn nhiều!
