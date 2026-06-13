# REST API — Ổ Đĩa Cloud Ảo (Telegram Drive)

Toàn bộ chức năng drive đều có REST API, dùng được từ N8N, script, hoặc app khác. Tài liệu này tóm tắt cách xác thực + các endpoint thường dùng kèm ví dụ cURL.

> Base URL = domain bạn deploy, ví dụ `https://drive.tencuaban.com`. Trong PWA, mở mục **API** để lấy lệnh cURL đã điền sẵn đúng domain + token (copy là chạy).

## 1. Xác thực

API dùng **token máy-máy** qua header:

```
Authorization: Device <token>
```

Lấy token: mở PWA → **API** → **Tạo token** (token chỉ hiện 1 lần, hãy lưu lại). Token có thể **thu hồi** bất cứ lúc nào trong cùng trang.

> ⚠️ Token = toàn quyền tài khoản của bạn. Giữ kín. Không commit vào code/repo.

Endpoint công khai không cần token: `GET /health`, trang `/share/{slug}`.

## 2. Dùng với N8N

1. Thêm node **HTTP Request**.
2. **Authentication → Generic Credential → Header Auth**.
3. **Name:** `Authorization` — **Value:** `Device <token>`.
4. **URL:** `https://drive.tencuaban.com` + đường dẫn endpoint.

## 3. Endpoint thường dùng

Đặt biến cho gọn:
```bash
BASE="https://drive.tencuaban.com"
TOKEN="dán-token-của-bạn"
AUTH="-H \"Authorization: Device $TOKEN\""
```

### Thông tin & thống kê
```bash
curl $BASE/health
curl -H "Authorization: Device $TOKEN" $BASE/v1/stats
curl -H "Authorization: Device $TOKEN" $BASE/v1/transfers
```

### Duyệt file & thư mục
```bash
# Nội dung thư mục (folder_id rỗng = gốc)
curl -H "Authorization: Device $TOKEN" "$BASE/v1/drive/contents?folder_id="

# Liệt kê file (phân trang/lọc/sắp xếp)
curl -H "Authorization: Device $TOKEN" "$BASE/v1/files?page=1&limit=50"

# Tìm kiếm
curl -H "Authorization: Device $TOKEN" "$BASE/v1/search?q=hop-dong"

# Tạo thư mục
curl -X POST -H "Authorization: Device $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"Thư mục mới","parent_id":""}' \
  $BASE/v1/folders
```

### Tải lên & tải xuống
```bash
# Upload (multipart). folder_id rỗng = gốc; relative_path để giữ cây thư mục
curl -X POST -H "Authorization: Device $TOKEN" \
  -F "file=@/duong-dan/file.pdf" \
  -F "folder_id=" \
  $BASE/v1/files/upload

# Tải về theo id
curl -H "Authorization: Device $TOKEN" -OJ "$BASE/v1/files/download?id=<FILE_ID>"

# Stream (hỗ trợ Range — tua video)
curl -H "Authorization: Device $TOKEN" "$BASE/v1/files/stream?id=<FILE_ID>"
```

> **File lớn:** nên dùng giao thức **tus** ở `/v1/tus/` (chunk + resume) thay vì multipart, để không vướng giới hạn reverse proxy. N8N có thể gọi tus qua HTTP Request nhiều bước, hoặc upload multipart cho file nhỏ/vừa.

### Thao tác file
```bash
# Đổi tên
curl -X PUT -H "Authorization: Device $TOKEN" -H "Content-Type: application/json" \
  -d '{"id":"<FILE_ID>","name":"ten-moi.pdf"}' $BASE/v1/files/rename

# Di chuyển
curl -X PUT -H "Authorization: Device $TOKEN" -H "Content-Type: application/json" \
  -d '{"id":"<FILE_ID>","new_parent_id":"<FOLDER_ID>"}' $BASE/v1/files/move

# Đánh dấu sao
curl -X PUT -H "Authorization: Device $TOKEN" -H "Content-Type: application/json" \
  -d '{"id":"<FILE_ID>","starred":true}' $BASE/v1/files/star

# Vào thùng rác
curl -X POST -H "Authorization: Device $TOKEN" -H "Content-Type: application/json" \
  -d '{"id":"<FILE_ID>"}' $BASE/v1/files/trash
```

### Chia sẻ link
```bash
# Liệt kê link của 1 file
curl -H "Authorization: Device $TOKEN" "$BASE/v1/shares?target_kind=file&target_id=<FILE_ID>"

# Tạo link (mật khẩu/hết hạn/giới hạn tùy chọn; 0 = không giới hạn)
curl -X POST -H "Authorization: Device $TOKEN" -H "Content-Type: application/json" \
  -d '{"target_kind":"file","target_id":"<FILE_ID>","password":"","expires_in":0,"max_downloads":0}' \
  $BASE/v1/shares

# Xóa link
curl -X DELETE -H "Authorization: Device $TOKEN" "$BASE/v1/shares?id=<SHARE_ID>"
```

## 4. Quản lý token (qua API)

```bash
# Tạo token mới (cần phiên PWA — thường tạo trong giao diện)
curl -X POST -H "Authorization: Device $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"n8n"}' $BASE/v1/api-tokens

# Liệt kê token
curl -H "Authorization: Device $TOKEN" $BASE/v1/api-tokens

# Thu hồi
curl -X DELETE -H "Authorization: Device $TOKEN" "$BASE/v1/api-tokens?id=<TOKEN_ID>"
```

## 5. Lưu ý

- Mọi dữ liệu được phân tách theo người dùng — token chỉ thấy file của chủ token.
- **Rate-limit Telegram:** ghi (upload) nhiều/nhanh có thể bị FLOOD_WAIT. Phù hợp tự động hóa vừa phải, không nên dùng làm storage ghi nặng.
- Response là JSON (trừ download/stream trả nội dung file). Lỗi trả `{"error":"..."}` kèm HTTP status tương ứng.
