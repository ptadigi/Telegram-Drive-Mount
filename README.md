# Ổ Đĩa Cloud Ảo Dùng Telegram Làm Lưu Trữ

Dự án này đang được chuyển hướng từ prototype Tauri/Rust cũ sang kiến trúc mới nhẹ hơn:

```text
PWA tiếng Việt + Go Desktop Agent + Gateway public domain + Telegram storage backend
```

Mục tiêu là tạo một ổ cứng cloud ảo: người dùng quản lý, đồng bộ, stream và chia sẻ file như một cloud drive thật; Telegram chỉ đóng vai trò tầng lưu trữ phía sau.

## Thành phần chính

- `agent-go/`: Go desktop agent. Phụ trách đăng nhập Telegram, metadata, sync desktop, stream local, WebDAV/FUSE sau này.
- `web-pwa/`: PWA tiếng Việt. Phụ trách upload từ điện thoại/browser, quản lý file, stream media, tạo link chia sẻ.
- `docs/`: Tài liệu kiến trúc, roadmap và design system.
- `app/`: Prototype Tauri cũ, giữ tạm để tham khảo logic Telegram upload/download/stream. Không còn là hướng phát triển chính.

## Trạng thái hiện tại

- Đã có skeleton Go Agent với health API.
- Đã có tài liệu kiến trúc sản phẩm mới.
- Đã bắt đầu tách PWA mới không phụ thuộc Tauri.

## Chạy Go Agent

```powershell
cd agent-go
go run ./cmd/agent
```

Kiểm tra:

```powershell
Invoke-RestMethod http://127.0.0.1:8750/health
Invoke-RestMethod http://127.0.0.1:8750/v1/info
```

## Chạy PWA

```powershell
cd web-pwa
npm install
npm run dev
```

## Nguyên tắc phát triển

- Giao diện và nội dung mặc định dùng tiếng Việt có dấu.
- Không dùng Tauri cho app mới.
- Go Agent là lõi native nhẹ cho desktop.
- PWA là giao diện chính cho mobile/web/desktop.
- Telegram được xem như object storage backend, không phải data model chính.
