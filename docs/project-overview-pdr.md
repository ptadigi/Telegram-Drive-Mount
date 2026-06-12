# Project Overview & PDR — Ổ Đĩa Cloud Ảo (Telegram Drive)

> Product Development Requirements + tổng quan sản phẩm. Nguồn chân lý cho định hướng; chi tiết kỹ thuật xem `system-architecture.md` và `codebase-summary.md`.

## 1. Tóm tắt một dòng

Một **ổ đĩa cloud cá nhân mã nguồn mở**, dùng **Telegram làm tầng lưu trữ ẩn** (object storage), với PWA tiếng Việt giống Google Drive và một agent desktop mount ổ ảo `T:` trên máy người dùng.

## 2. Vấn đề & cơ hội

- Dịch vụ cloud (Drive/Dropbox/OneDrive) tính phí theo dung lượng, khóa dữ liệu trong hệ sinh thái của họ.
- Telegram cho tài khoản dung lượng rất lớn, miễn phí, ổn định — nhưng UX không phải là một "ổ đĩa": không cây thư mục, không mount, không share link kiểm soát được.
- **Cơ hội:** bọc Telegram bằng một lớp drive đúng nghĩa (thư mục, đồng bộ, mount, share có mật khẩu/hết hạn), self-host được, để người dùng làm chủ dữ liệu.

## 3. Người dùng mục tiêu

- Cá nhân/kỹ thuật muốn cloud riêng, miễn phí, tự host trên 1 VPS nhỏ.
- Nhóm nhỏ/đội nhóm muốn chia sẻ file qua link an toàn mà không lệ thuộc dịch vụ trả phí.
- Người không rành kỹ thuật: cài 1-click trên Windows, dùng như Google Drive.

## 4. Nguyên tắc cốt lõi (locked)

1. **Telegram là kho lưu trữ, không phải UI.** Người dùng thao tác qua drive; Telegram ẩn phía sau.
2. **Self-host & mã nguồn mở.** Không phụ thuộc một máy chủ trung tâm nào. `tele.pogen.im` chỉ là instance test của maintainer.
3. **Dữ liệu thuộc về người dùng.** Session Telegram mã hóa tại chỗ (AES-256), không gửi đi đâu.
4. **Dễ cài.** Mỗi môi trường có phương án 1-click.
5. **Tôn trọng giới hạn Telegram.** Chống FLOOD_WAIT, upload chunk, cache thông minh.

## 5. Phạm vi sản phẩm (đã có)

- **Lưu trữ qua Telegram:** upload/download, đồng bộ file lên kênh Telegram riêng làm storage.
- **Drive đầy đủ:** cây thư mục, đổi tên/di chuyển/xóa (thùng rác), đánh dấu sao, tìm kiếm (scope theo user).
- **Upload mạnh:** hàng đợi đa luồng, upload chunk có resume (tus) cho file lớn, kéo-thả thư mục hàng chục nghìn file.
- **Xem file trong PWA:** ảnh (zoom/pan), video/audio (stream + seek), PDF, văn bản, **docx** (render bằng mammoth). Thumbnail ảnh/video/PDF.
- **Mount ổ ảo:** WinFsp (Windows) / FUSE (Linux/macOS) qua cgofuse, ổ `T:` "Telegram Drive".
- **Đa thiết bị:** một server chạy agent đầy đủ; máy khác chạy thin-client `--remote` mount qua HTTPS. Ghép thiết bị bằng mã pairing.
- **Chia sẻ link:** public/mật khẩu/hết hạn/giới hạn lượt tải, có trang xem trực tiếp (ảnh/video/pdf/docx).
- **Web Share Target:** chia sẻ file từ điện thoại thẳng vào PWA.
- **Bảo mật:** tài khoản PWA (bcrypt), session token hash, scope dữ liệu theo user, Basic Auth tùy chọn, audit log.
- **Vận hành:** systemd/Docker, reverse proxy, cache LRU, auto-backup metadata, dọn temp upload.

## 6. Ngoài phạm vi (hiện tại)

- Không phải dịch vụ SaaS đa người thuê quy mô lớn (multi-tenant billing).
- Không đồng bộ hai chiều realtime kiểu Dropbox (đồng bộ thư mục desktop là upload-watch).
- Không CDN/edge tích hợp sẵn (dựa vào reverse proxy của người dùng).

## 7. Yêu cầu phi chức năng

- **Bảo mật:** mã hóa session tại chỗ; không lộ secret trong log/repo; mọi `/v1/*` cần phiên đăng nhập hoặc device token.
- **Hiệu năng:** UI mượt với thư mục lớn (virtualize/stream), upload không chặn UI, stream có Range.
- **Khả chuyển:** binary thuần Go (CGO tắt, SQLite pure-Go) — chạy đa nền tảng, dễ cài.
- **Khả dụng offline:** PWA build sẵn icon SVG inline, không fetch runtime.
- **Tự phục hồi:** retry upload, janitor dọn rác, agent restart-on-failure.

## 8. Chỉ số thành công (đề xuất)

- Cài đặt thành công < 5 phút trên VPS phổ thông (1-click).
- Upload/preview file 100MB+ ổn định sau reverse proxy.
- Không mất dữ liệu: Telegram là nguồn chân lý; cache chỉ chứa bản sao.

## 9. Rủi ro & giảm thiểu

| Rủi ro | Giảm thiểu |
|--------|-----------|
| Telegram rate-limit (FLOOD_WAIT) | Upload chunk, hàng đợi có giới hạn, cache |
| Reverse proxy chặn file lớn | Tài liệu hóa `client_max_body_size 0` + `proxy_request_buffering off` |
| SmartScreen cảnh báo bản tự ký | Hướng dẫn + checksum; lộ trình ký SignPath/Certum |
| Mất khóa session | `TD_AGENT_SESSION_KEY` lưu bền vững; tài liệu backup |

## 10. Lộ trình

Xem `project-roadmap.md` (EN) và `ROADMAP_VI.md` (VI) cho trạng thái milestone và mục tiêu kế tiếp.
