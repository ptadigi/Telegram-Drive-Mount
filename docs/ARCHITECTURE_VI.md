# Kien truc san pham: Telegram-backed Virtual Cloud Drive

## 1. Tam nhin san pham

Muc tieu cua du an khong phai chi la mot ung dung upload file len Telegram. San pham can tro thanh mot o cung cloud ao, trong do Telegram duoc dung nhu tang luu tru object storage phia sau.

Nguoi dung se nhin thay mot trai nghiem giong Google Drive, Dropbox, network drive hoac o dia ao tren PC. Telegram duoc an di nhu ha tang luu tru, khong phai giao dien chinh cua san pham.

```text
Nguoi dung thay:
- PWA quan ly file
- Ung dung desktop/dashboard
- Thu muc dong bo tren PC
- O dia ao hoac WebDAV drive
- Link public bang domain rieng
- Media streaming

He thong ben duoi lam:
- Dang nhap Telegram bang MTProto
- Upload/download object len Telegram
- Quan ly metadata rieng
- Stream byte range tu Telegram
- Dong bo local folder
- Mount drive ao
- Proxy public URL
```

## 2. Nguyen tac kien truc

- Telegram chi la storage backend, khong phai data model chinh.
- Drive tree, metadata, share link, sync state phai nam trong database rieng.
- PWA khong chay autosync nen; autosync chi chay trong Go desktop agent.
- Core phai tach khoi UI de sau nay co the dung tren PWA, desktop, gateway va agent.
- Media streaming va public sharing phai ho tro HTTP Range ngay tu dau.
- Thiet ke theo interface storage backend de sau nay co the them S3, R2, local disk hoac backend khac.

## 3. Kien truc tong the

```text
+-----------------------------+
| PWA Mobile / Web / Desktop  |
| upload, browse, share, play |
+--------------+--------------+
               |
               | HTTPS / WebSocket
               v
+-----------------------------+        +-----------------------------+
| Public Gateway Go           |        | Local Go Desktop Agent      |
| upload, share, proxy, API   |        | sync, mount, cache, stream  |
+--------------+--------------+        +--------------+--------------+
               |                                      |
               | MTProto                              | MTProto
               v                                      v
+---------------------------------------------------------------+
| Telegram Storage Layer                                        |
| Saved Messages / private channels / message documents          |
+---------------------------------------------------------------+
```

## 4. Cac thanh phan chinh

### 4.1 PWA

PWA la giao dien chinh cho mobile, web va desktop browser.

Nhiem vu:
- Upload file, anh, video tu dien thoai.
- Duyet folder/file.
- Search file.
- Xem preview.
- Stream media.
- Tao va quan ly public link.
- Xem trang thai sync cua desktop agent neu agent dang online.

Khong lam:
- Khong autosync thu muc nen.
- Khong mount o dia.
- Khong doc toan bo filesystem cua may.
- Khong giu Telegram MTProto session truc tiep trong browser neu co backend/gateway.

Cong nghe de xuat:
- React + Vite hoac Next.js.
- TypeScript.
- PWA manifest.
- Service Worker.
- IndexedDB cache metadata.
- i18next cho tieng Viet.
- WebSocket hoac SSE cho realtime progress.

### 4.2 Go Desktop Agent

Day la thanh phan native quan trong nhat tren PC.

Nhiem vu:
- Dang nhap Telegram bang MTProto.
- Luu session local.
- Upload/download file.
- Scan thu muc local.
- Theo doi thay doi bang filesystem watcher.
- Autosync PC -> Telegram, Telegram -> PC hoac two-way sync.
- Quan ly queue upload/download.
- Resolve conflict.
- Cache file/chunk local.
- Serve local REST API cho PWA desktop.
- Serve WebSocket/SSE event realtime.
- Serve media stream local.
- Expose WebDAV hoac FUSE drive.
- Ket noi tunnel/proxy domain neu chon local-public mode.

Cong nghe de xuat:
- Go.
- Telegram MTProto: `gotd/td`.
- File watcher: `fsnotify`.
- Local DB: SQLite.
- HTTP router: `chi` hoac `net/http`.
- Realtime: WebSocket hoac SSE.
- WebDAV: `golang.org/x/net/webdav`.
- FUSE nang cao: `winfsp/cgofuse`.
- Logging: `zerolog` hoac `zap`.

### 4.3 Public Gateway

Gateway la backend public phuc vu PWA mobile/web va public URL.

Nhiem vu:
- API upload file tu PWA.
- API list/search/browse file.
- Tao public share link.
- Xac thuc password/expiration/revoke.
- Proxy stream/download file qua domain.
- Ho tro HTTP Range cho media.
- Rate limit, audit log, quota.

Co 2 che do trien khai:

#### Cloud Gateway

```text
Client -> https://drive.domain.com -> Gateway -> Telegram
```

Phu hop neu xay SaaS hoac muon trai nghiem de dung.

#### Local Agent + Tunnel

```text
Client -> https://files.domain.com -> Tunnel -> Go Agent tren PC -> Telegram
```

Phu hop neu uu tien privacy-first va muon du lieu di qua may user.

### 4.4 Telegram Storage Adapter

Telegram adapter bien Telegram thanh object storage.

Nhiem vu:
- Login/logout/session.
- Upload object.
- Download object.
- Download byte range.
- Delete object.
- Lay metadata Telegram message.
- Tao channel/private folder neu can.
- Map Telegram message/channel ve file object.

Interface de xuat:

```go
type StorageBackend interface {
    PutObject(ctx context.Context, input PutObjectInput) (ObjectRef, error)
    GetObject(ctx context.Context, ref ObjectRef) (io.ReadCloser, error)
    GetObjectRange(ctx context.Context, ref ObjectRef, start, end int64) (io.ReadCloser, error)
    DeleteObject(ctx context.Context, ref ObjectRef) error
    StatObject(ctx context.Context, ref ObjectRef) (ObjectInfo, error)
}
```

## 5. Data model de xuat

### 5.1 folders

```text
id
parent_id
name
created_at
updated_at
deleted_at
storage_bucket_ref
```

### 5.2 files

```text
id
folder_id
name
size
mime_type
hash
created_at
updated_at
deleted_at
current_version_id
sync_state
```

### 5.3 file_versions

```text
id
file_id
version_number
size
hash
telegram_channel_id
telegram_message_id
telegram_file_id
created_at
```

### 5.4 sync_roots

```text
id
local_path
remote_folder_id
mode              -- upload_only, download_only, two_way
enabled
created_at
updated_at
```

### 5.5 sync_entries

```text
id
sync_root_id
local_path
remote_file_id
local_hash
remote_hash
local_mtime
remote_updated_at
state             -- synced, pending_upload, pending_download, conflict, error
last_error
```

### 5.6 shares

```text
id
file_id
slug
password_hash
expires_at
revoked
created_at
last_accessed_at
access_count
```

## 6. Luong upload tu PWA mobile

```text
User mo PWA tren dien thoai
-> Chon file/anh/video
-> PWA upload qua HTTPS API
-> Gateway nhan upload
-> Gateway upload object len Telegram
-> Gateway ghi metadata vao DB
-> PWA nhan progress va ket qua
```

Luu y:
- Browser khong nen giu MTProto session truc tiep trong MVP.
- Upload can co progress, retry, resume sau nay.
- File lon can chunk upload o gateway.

## 7. Luong autosync desktop

```text
User cai Go Agent
-> Login Telegram
-> Chon folder local
-> Agent scan folder
-> Agent tao sync index SQLite
-> File moi/sua duoc dua vao queue
-> Agent upload len Telegram
-> Metadata duoc cap nhat
-> PWA/Desktop UI nhan event realtime
```

Sync mode:
- `upload_only`: PC day file len cloud.
- `download_only`: cloud keo file ve PC.
- `two_way`: dong bo hai chieu, can conflict resolver.

MVP nen bat dau voi `upload_only`.

## 8. Luong mount o ao

### MVP: WebDAV

```text
Go Agent -> WebDAV server local -> OS mount network drive
```

Uu diem:
- De hien thuc hon FUSE.
- Dung duoc voi Windows/macOS/Linux.
- Phu hop MVP.

### Nang cao: FUSE/WinFsp

```text
Go Agent -> cgofuse -> Virtual Drive native
```

Uu diem:
- Trai nghiem giong o dia that hon.
- Kiem soat cache va read/write tot hon.

Nhanh nhat nen lam WebDAV truoc, sau do moi lam FUSE.

## 9. Luong media streaming

```text
Client GET /stream/{fileId}
Header: Range: bytes=start-end
-> Gateway/Agent kiem tra token
-> Tim Telegram object ref
-> Fetch chunk tu Telegram
-> Cache chunk local/server
-> Tra HTTP 206 Partial Content
```

Bat buoc co:
- `Accept-Ranges: bytes`.
- `Content-Range`.
- `Content-Length`.
- MIME type dung.
- Token hoac signed URL.
- Cache chunk.

MVP khong transcode, chi stream file goc. HLS/FFmpeg de giai doan sau.

## 10. Public URL qua proxy domain

Dinh dang URL de xuat:

```text
https://files.domain.com/s/{slug}/{filename}
https://files.domain.com/d/{slug}/{filename}
https://files.domain.com/stream/{slug}/{filename}
```

Security:
- Slug random du dai.
- Optional password.
- Expiration.
- Revoke.
- Rate limit.
- Signed temporary stream URL.
- Audit access.

## 11. Vi tri cua repo hien tai

Repo `Telegram-Drive` hien tai nen duoc xem la:
- Prototype da chung minh Telegram co the lam storage.
- Reference cho auth/upload/download/stream/share.
- Nguon de hoc UI dashboard va flow Telegram.
- Nen tang demo ngan han.

Khong nen xem la kien truc cuoi cung vi:
- Backend dang la Rust/Tauri, trong khi core moi nen la Go agent/gateway.
- Chua co PWA-first architecture.
- Chua co autosync desktop dung nghia.
- Chua co mount drive ao.

## 12. Ket luan

Kien truc dung cho muc tieu cua san pham la PWA + Go Desktop Agent + Public Gateway + Telegram Storage Adapter. Telegram duoc dung nhu object storage, con toan bo trai nghiem cloud drive, sync, mount, share va stream nam trong he thong cua minh.
