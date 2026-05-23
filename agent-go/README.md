# Telegram Drive Agent Go

Go desktop agent cho kien truc Telegram-backed Virtual Cloud Drive.

Agent nay se dan thay the phan core native dai han:

- Local REST API cho PWA/Desktop UI.
- Telegram storage adapter.
- Metadata database.
- Autosync desktop folder.
- Local media streaming.
- WebDAV/FUSE virtual drive.

## Trang thai hien tai

Skeleton MVP:

- `GET /health`
- `GET /v1/info`
- Config mac dinh.
- Storage backend interface.
- Telegram backend placeholder.

## Chay thu

```powershell
cd agent-go
go run ./cmd/agent
```

Mac dinh agent lang nghe tai:

```text
127.0.0.1:8750
```

Kiem tra:

```powershell
Invoke-RestMethod http://127.0.0.1:8750/health
Invoke-RestMethod http://127.0.0.1:8750/v1/info
```

## Cấu hình Telegram API

Người dùng cuối không nhập API ID/API Hash trong UI. Đây là cấu hình nội bộ của Go Agent.

Trong giai đoạn dev, đặt API Telegram bằng `config.example.json` hoặc biến môi trường trước khi chạy agent. Sau này bản phát hành sẽ đóng gói cấu hình này ở tầng ứng dụng/gateway để người dùng chỉ cần đăng nhập bằng số điện thoại hoặc QR.

## Roadmap gần

1. Them config file.
2. Them SQLite metadata.
3. Them Telegram auth bang `gotd/td`.
4. Them upload/download object.
5. Them stream endpoint co HTTP Range.
6. Them sync upload-only.
