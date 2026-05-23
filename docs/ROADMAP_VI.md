# Roadmap phat trien: Telegram-backed Virtual Cloud Drive

## 1. Muc tieu tong quat

Xay dung mot he sinh thai o cung cloud ao dung Telegram lam tang luu tru phia sau. San pham gom PWA cho upload/xem/share/stream, Go desktop agent cho autosync/mount/cache, va public gateway cho domain sharing.

Repo hien tai `Telegram-Drive` duoc dung lam prototype va reference implementation cho Telegram auth, upload/download, streaming, share va dashboard UI.

## 2. Nguyen tac trien khai

- Khong rewrite tat ca ngay lap tuc.
- Dung repo hien tai de hoc va tao demo nhanh.
- Viet tai lieu kien truc truoc khi tach core.
- Viet Go agent rieng de phuc vu sync/mount/stream dai han.
- PWA khong phu trach autosync nen.
- Uu tien MVP chay duoc truoc, sau do moi them two-way sync va FUSE.

## 3. Phase 0: Khao sat va chay prototype hien tai

Muc tieu:
- Dam bao repo hien tai build/chay duoc.
- Test cac tinh nang Telegram co san.
- Ghi lai bug va gioi han.

Cong viec:
- Cai dependency trong `app`.
- Chay frontend build.
- Chay `npm run tauri dev`.
- Test login Telegram.
- Test upload/download/list folder.
- Test video/audio streaming.
- Test share link.
- Test proxy/VPN settings neu can.

Lenh du kien:

```powershell
cd Telegram-Drive/app
npm install
npm run build
npm run tauri dev
```

Ket qua mong muon:
- Co ban demo desktop hien tai.
- Co danh sach bug/han che.
- Co nhan xet phan nao nen migrate sang Go.

## 4. Phase 1: Viet hoa va chuan hoa UI hien tai

Muc tieu:
- Bien prototype hien tai thanh ban demo tieng Viet.
- Tao nen tang UX cho san pham moi.

Cong viec:
- Them i18n cho frontend.
- Tao `vi.json` va `en.json`.
- Gom tat ca text UI vao locale files.
- Dich dashboard, auth, settings, share dialog, queue, toast.
- Chuan hoa format file size, date/time theo tieng Viet.
- Chuan hoa error message than thien.
- Ap dung `docs/DESIGN.md` vao UI dan dan.

Ket qua mong muon:
- App desktop hien tai co giao dien tieng Viet.
- Co ngon ngu san pham ro: "o dia ao", "dong bo", "tai len", "lien ket chia se".

## 5. Phase 2: Thiet ke Drive Core va metadata schema

Muc tieu:
- Tach tu duy "Telegram message" thanh "Drive file/object".
- Xac dinh data model chuan cho PWA, agent va gateway.

Cong viec:
- Chot schema SQLite/PostgreSQL cho folders, files, versions, shares, sync.
- Dinh nghia `StorageBackend` interface.
- Dinh nghia Telegram object mapping.
- Dinh nghia API contract bang OpenAPI.
- Dinh nghia event contract cho WebSocket/SSE.

Ket qua mong muon:
- Co specification de Go agent va PWA cung implement.
- Khong phu thuoc truc tiep vao UI hien tai.

## 6. Phase 3: Go Telegram Core MVP

Muc tieu:
- Tao Go module moi co kha nang login, upload, download, list file qua Telegram.

Thu muc de xuat:

```text
agent-go/
  cmd/agent/
  internal/auth/
  internal/storage/telegram/
  internal/drive/
  internal/db/
  internal/api/
  internal/stream/
```

Cong nghe:
- Go.
- `gotd/td` cho Telegram MTProto.
- SQLite cho local metadata.
- `net/http` hoac `chi` cho local API.

Cong viec:
- Login Telegram bang phone/code/password.
- Luu session local.
- Upload file len Telegram.
- Download file tu Telegram.
- List object/file.
- Ghi metadata vao SQLite.
- Expose local API co ban.

API MVP:

```text
POST /auth/start
POST /auth/code
POST /auth/password
GET  /drive/files
POST /drive/upload
GET  /drive/download/{fileId}
GET  /health
```

Ket qua mong muon:
- Co binary Go chay doc lap.
- Co the upload/download file khong can Tauri/Rust.

## 7. Phase 4: Local streaming bang Go

Muc tieu:
- Thay the/bo sung streaming Rust hien tai bang Go streaming service.

Cong viec:
- Implement `/stream/{fileId}`.
- Ho tro HTTP Range.
- Fetch byte range tu Telegram.
- Cache chunk local.
- MIME detection.
- Signed stream URL.

Tieu chi thanh cong:
- Browser co the play MP4/MP3.
- Co the tua video.
- Stream file lon khong can tai het truoc.

## 8. Phase 5: PWA MVP

Muc tieu:
- Tao app web/PWA de upload tu dien thoai va quan ly file co ban.

Thu muc de xuat:

```text
web-pwa/
  src/
  public/manifest.webmanifest
  src/locales/vi.json
  src/locales/en.json
```

Cong viec:
- Login/connect account theo flow gateway/agent.
- Upload file tu mobile.
- Browse folder/file.
- Search.
- Preview/stream media.
- Tao share link.
- PWA manifest + service worker.
- IndexedDB cache metadata gan day.

Khong lam trong MVP:
- Autosync nen.
- Mount drive.
- Offline full file sync.

## 9. Phase 6: Desktop autosync trong Go Agent

Muc tieu:
- Go agent dong bo thu muc desktop len Telegram.

Cong viec:
- Chon sync root.
- Scan local folder.
- Theo doi thay doi bang `fsnotify`.
- Upload-only sync.
- Queue + retry.
- Pause/resume.
- Bandwidth limit.
- Sync status event qua WebSocket/SSE.

Trang thai file:
- `synced`.
- `pending_upload`.
- `uploading`.
- `error`.
- `conflict`.

Tieu chi thanh cong:
- User chon mot folder tren PC.
- File moi tu dong upload len Telegram.
- UI thay progress realtime.

## 10. Phase 7: Public Gateway va proxy domain

Muc tieu:
- Tao URL public cho file qua domain rieng.

Cong viec:
- Tao gateway Go.
- Tao share slug.
- Password optional.
- Expiration.
- Revoke.
- Download endpoint.
- Stream endpoint co Range.
- Rate limit.
- CORS va security headers.
- Domain + TLS qua Caddy/Cloudflare.

URL de xuat:

```text
https://files.domain.com/s/{slug}/{filename}
https://files.domain.com/stream/{slug}/{filename}
```

Tieu chi thanh cong:
- Gui link cho nguoi khac download/stream duoc.
- Link het han/bi thu hoi dung chinh xac.

## 11. Phase 8: WebDAV virtual drive MVP

Muc tieu:
- Bien drive thanh network drive mount duoc tren OS.

Cong viec:
- Go agent expose WebDAV local.
- Map folder/file metadata sang WebDAV tree.
- Doc file bang lazy download.
- Cache file local.
- Ghi file moi qua WebDAV neu kha thi.

Tieu chi thanh cong:
- Windows/macOS/Linux mount duoc drive.
- Mo file tu Explorer/Finder thanh cong.
- File lon co cache va progress hop ly.

## 12. Phase 9: FUSE/WinFsp virtual drive nang cao

Muc tieu:
- Tao trai nghiem o dia ao native tot hon WebDAV.

Cong nghe:
- `winfsp/cgofuse`.
- Windows: WinFsp.
- macOS: macFUSE.
- Linux: FUSE2/FUSE3.

Cong viec:
- Read-only mount truoc.
- Lazy read + cache.
- Metadata operations.
- Write support sau.
- Conflict handling.

## 13. Phase 10: Two-way sync va conflict resolver

Muc tieu:
- Dong bo hai chieu giong Google Drive.

Cong viec:
- Remote change detection.
- Local change detection.
- Rename/move detection.
- Delete policy.
- Conflict copies.
- Version history.
- UI resolve conflict.

Can than:
- Day la phase kho, khong nen lam truoc khi upload-only on dinh.

## 14. Uu tien ngan han de lam ngay

1. Cai dependency va build repo hien tai.
2. Tao issue list/bug list sau khi chay app.
3. Them i18n va Viet hoa frontend hien tai.
4. Tao skeleton `agent-go`.
5. Implement Go auth/upload/download MVP bang `gotd/td`.
6. Viet OpenAPI cho local agent.
7. Tao PWA skeleton dua tren `docs/DESIGN.md`.

## 15. Ranh gioi MVP dau tien

MVP dau tien nen gom:
- Desktop prototype hien tai chay duoc.
- UI tieng Viet co ban.
- Go agent login Telegram va upload/download duoc.
- PWA upload file tu dien thoai qua API.
- Stream media co Range.
- Share link public co expiration/revoke.

Chua can gom:
- FUSE native.
- Two-way sync.
- HLS transcoding.
- Multi-tenant SaaS day du.
- End-to-end encryption rieng.

## 16. Ket luan

Duong di tot nhat la dung repo hien tai de tang toc hieu biet va demo, dong thoi xay core moi bang Go theo tung phase. San pham cuoi cung se khong bi gioi han boi Tauri/Rust hien tai, nhung van tan dung duoc rat nhieu bai hoc va logic da co.
