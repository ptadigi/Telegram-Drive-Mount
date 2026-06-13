# REST API — Ổ Đĩa Cloud Ảo (Telegram Drive)

API để tự động hóa (N8N, script, tích hợp khác). Mọi endpoint chạy trên chính domain bạn đã cấu hình, vd `https://drive.cuaban.com`.

> Tạo & quản lý token trong PWA: vào mục **API** ở menu trái. Trang đó còn có lệnh cURL **copy-ready** theo đúng domain của bạn.

## Xác thực

Dùng header (auth máy-máy, không cần cookie):

```
Authorization: Device <token>
```

Token tạo trong PWA → mục **API** → *Tạo token*. Token chỉ hiện **một lần**, hãy lưu lại. Có thể thu hồi bất cứ lúc nào.

> ⚠️ Token có **toàn quyền** với dữ liệu tài khoản — giữ kín. Ghi/đọc tần suất cao có thể bị Telegram giới hạn (FLOOD_WAIT). File lớn nên upload qua tus `/v1/tus/`.

## Dùng với N8N

1. Node **HTTP Request**.
2. Authentication → **Generic Credential Type** → **Header Auth**.
3. Name `Authorization`, Value `Device <token>`.
4. URL = `https://drive.cuaban.com` + đường dẫn endpoint.

## Endpoint thông dụng

Đặt `BASE=https://drive.cuaban.com` và `TOKEN=...` cho gọn.

### Cơ bản
```bash
# Thống kê tệp/thư mục/dung lượng
curl -s "$BASE/v1/stats" -H "Authorization: Device $TOKEN"

# Thông tin agent
curl -s "$BASE/v1/info" -H "Authorization: Device $TOKEN"
```

### File & thư mục
```bash
# Liệt kê nội dung thư mục (folder_id rỗng = gốc)
curl -s "$BASE/v1/drive/contents?folder_id=" -H "Authorization: Device $TOKEN"

# Liệt kê file
curl -s "$BASE/v1/files" -H "Authorization: Device $TOKEN"

# Tìm kiếm
curl -s "$BASE/v1/search?q=hop-dong" -H "Authorization: Device $TOKEN"

# Tạo thư mục
curl -s -X POST "$BASE/v1/folders" -H "Authorization: Device $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Thư mục mới","parent_id":""}'
```

### Upload / Download
```bash
# Upload (multipart, field "file")
curl -s -X POST "$BASE/v1/files/upload" -H "Authorization: Device $TOKEN" \
  -F "file=@/duong-dan/file.pdf" -F "folder_id="

# Download theo id
curl -s -L "$BASE/v1/files/download?id=FILE_ID" -H "Authorization: Device $TOKEN" -o file.bin

# Stream (hỗ trợ Range)
curl -s "$BASE/v1/files/stream?id=FILE_ID" -H "Authorization: Device $TOKEN" -o stream.bin
```

> File lớn: dùng giao thức **tus** tại `/v1/tus/` (chunk + resume) thay cho `/v1/files/upload`.

### Thao tác file
```bash
# Đổi tên
curl -s -X PUT "$BASE/v1/files/rename" -H "Authorization: Device $TOKEN" \
  -H "Content-Type: application/json" -d '{"id":"FILE_ID","name":"ten-moi.pdf"}'

# Di chuyển
curl -s -X PUT "$BASE/v1/files/move" -H "Authorization: Device $TOKEN" \
  -H "Content-Type: application/json" -d '{"id":"FILE_ID","new_parent_id":"FOLDER_ID"}'

# Vào thùng rác
curl -s -X POST "$BASE/v1/files/trash" -H "Authorization: Device $TOKEN" \
  -H "Content-Type: application/json" -d '{"id":"FILE_ID"}'
```

### Chia sẻ
```bash
# Tạo link chia sẻ file
curl -s -X POST "$BASE/v1/shares" -H "Authorization: Device $TOKEN" \
  -H "Content-Type: application/json" -d '{"target_kind":"file","target_id":"FILE_ID"}'

# Liệt kê link
curl -s "$BASE/v1/shares" -H "Authorization: Device $TOKEN"
```

## Tham khảo thêm

Toàn bộ endpoint khác (folders rename/move/star/trash/restore, shares update/delete, storage, audit, transfers, events SSE…) đều theo cùng quy ước và cùng header auth. Mở mục **API** trong PWA để xem danh sách + cURL sinh theo domain của bạn.

## Mã lỗi thường gặp

| HTTP | Ý nghĩa |
|------|---------|
| 401 | Thiếu/sai token (`Authorization: Device <token>`) |
| 403 | Không đủ quyền / vượt giới hạn (vd share hết lượt) |
| 404 | Không tìm thấy tài nguyên |
| 429 | Quá nhiều yêu cầu (rate limit) |
