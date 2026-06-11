# Plan: Chunked / Resumable Upload (tus protocol)

Mục tiêu: upload file lớn (video, >50MB, GB) không còn phụ thuộc giới hạn body của
reverse proxy (nginx/OpenResty/Cloudflare), có resume khi đứt mạng, song song nhiều
chunk cho nhanh. Đây là cách Google Drive/Dropbox/Vimeo/Cloudflare Stream làm.

## Kết luận nghiên cứu (chốt công nghệ)
- Chuẩn ngành: **tus protocol** (https://tus.io). Open, HTTP-based, resumable.
- Server Go: **github.com/tus/tusd** — nhúng làm `http.Handler`, có custom store.
- Client web: **tus-js-client** — chia chunk, retry, song song, progress.
- KHÔNG tự chế protocol. tus đã giải quyết: offset tracking, resume, retry, parallel.
- Vì sao hợp dự án: mỗi chunk là 1 request nhỏ (vd 8-16MB) → KHÔNG đụng
  `client_max_body_size` của proxy nữa. File 5GB vẫn lọt vì proxy chỉ thấy chunk nhỏ.

## Cách hoạt động (tóm tắt)
1. Client `POST /v1/tus` tạo upload (gửi metadata: tên, size, folder_id, relative_path).
   Server trả URL upload + id.
2. Client `PATCH` từng chunk theo offset. Đứt mạng → `HEAD` hỏi offset → tiếp tục.
3. Chunk ghi tạm vào `<data_dir>/uploads/tus/`.
4. Khi đủ bytes (hoàn tất), server hook `OnUploadFinish`: ghép file hoàn chỉnh →
   đưa vào pipeline hiện có (`SaveLocalFile`/queue sync Telegram) y như upload thường.
5. File xuất hiện trong drive + sync Telegram như bình thường.

## Phạm vi
- KHÔNG đụng pipeline sync Telegram (giữ nguyên queue + worker hiện có).
- KHÔNG bỏ endpoint `/v1/files/upload` cũ (giữ cho file nhỏ + tương thích).
- Thêm đường tus song song; client tự chọn: file > ngưỡng (vd 32MB) → tus, nhỏ → cũ.

---

## Phase 1 — Backend: nhúng tusd
- Thêm dep `github.com/tus/tusd/v2`.
- Tạo `internal/api/tus.go`:
  - filestore trỏ `<data_dir>/uploads/tus/`.
  - `tusd.NewHandler` mount tại `/v1/tus/` (+ `/v1/tus` create).
  - Auth: tái dùng session/device-token guard như `/v1/*` (chặn loopback/CORS y hệt).
  - Giới hạn: cấu hình `MaxSize` (vd 0 = không giới hạn), CORS cho PATCH/HEAD/POST.
- Wire vào `server.go` mux (qua `withAuth`, bỏ qua `withJSON` cho path tus vì tus set
  Content-Type riêng).
- Acceptance: `tus-js-client` test upload 1 file qua `/v1/tus` chạy được.

## Phase 2 — Backend: hook hoàn tất → pipeline cũ
- `OnUploadFinish`/`OnUploadComplete`:
  - đọc metadata (filename, folder_id, relative_path).
  - move/rename file tạm tus → import qua `drive.SaveLocalFile`/`saveFileFromReader`
    (đưa vào DB + tạo transfer + queue sync Telegram).
  - dọn file tus tạm sau khi import.
- Lưu ý: tránh copy 2 lần file lớn — ưu tiên `os.Rename` (cùng volume) thay vì stream copy.
- Acceptance: upload tus xong → file hiện trong drive, sync Telegram, sync_state đúng.

## Phase 3 — Frontend: tus-js-client trong upload queue
- Thêm dep `tus-js-client`.
- Sửa `web-pwa/src/state/uploads.ts`:
  - Ngưỡng: file > 32MB (hoặc cấu hình) → dùng tus; nhỏ hơn → `uploadFile` cũ.
  - tus options: `chunkSize` (vd 16MB), `parallelUploads` hợp lý, `retryDelays`,
    `metadata` (filename, folder_id, relative_path), `endpoint` = `${AGENT_BASE_URL}/v1/tus`.
  - Progress: map `onProgress(bytesSent, bytesTotal)` vào queue item (đã có throttle UI).
  - Tích hợp pool 6 worker hiện có: mỗi file lớn là 1 worker, tus tự chia chunk bên trong.
  - Resume: lưu `fingerprint` (tus localStorage mặc định) để đứt mạng nối lại.
- Acceptance: kéo file 1GB → chạy hết, đứt mạng giữa chừng → nối lại không mất tiến trình.

## Phase 4 — Resume UX + retry
- Khi reload trang giữa upload lớn: tus localStorage còn → cho "tiếp tục" thay vì làm lại.
- Nút retry đã có (retryFailed) → với tus là resume từ offset.
- beforeunload guard đã có.
- Acceptance: tắt tab giữa upload 1GB, mở lại → tiếp tục được.

## Phase 5 — Proxy/deploy
- tus chunk nhỏ nên KHÔNG cần `client_max_body_size` lớn nữa (chỉ cần > chunkSize).
- Cập nhật `docs/DEPLOY.md`: tus là cách chuẩn cho file lớn; proxy chỉ cần body >
  chunkSize (vd 64MB) + không buffer.
- Giữ hướng dẫn `client_max_body_size` cũ cho người chưa bật tus.
- Lưu ý Cloudflare 100MB: với tus chunk 16-64MB → lọt, hết bị chặn file lớn.

## Phase 6 — Test + release
- Backend: `go test ./...`; test import-after-tus (file tạm → drive → queue).
- Frontend: `npm run build`.
- Manual: upload 100MB, 1GB; đứt mạng giữa chừng; song song nhiều file.
- Deploy test instance, verify file lớn qua proxy 50MB cũ vẫn chạy (vì chunk nhỏ).
- Bump version, tag release.

---

## Rủi ro / lưu ý
- tusd v2 API: cần kiểm tra đúng signature hook + store hiện tại.
- Ghép file lớn tốn đĩa tạm: file tus + file import. Dùng rename cùng volume để tránh
  nhân đôi; nếu khác volume phải stream (chậm/tốn đĩa).
- Dọn rác: chunk tus dở dang (upload bỏ giữa chừng) cần job dọn theo TTL.
- Auth cho tus: phải chắc chắn guard giống `/v1/*`, không hở endpoint upload public.
- CORS: tus cần expose header `Location`, `Upload-Offset`, `Tus-Resumable`.
- Mobile Safari/PWA: kiểm tra tus-js-client chạy ổn trên iOS (chunk + localStorage).
- Không phá `/v1/files/upload` cũ (nhiều nơi đang dùng: folder collect, drag-drop).

## Quyết định đã chốt
- chunkSize: 16MB.
- Ngưỡng dùng tus: file > 32MB (nhỏ hơn giữ đường `/v1/files/upload` cũ).
- Song song chunk trong 1 file lớn: BẬT (parallelUploads).
- Resume xuyên phiên (localStorage fingerprint); lỗi thì hiện để retry.

## Câu hỏi mở
- chunkSize mặc định: 16MB hay 64MB? (nhỏ = an toàn proxy hơn, nhiều request hơn).
- Ngưỡng chuyển sang tus: 32MB hợp lý? File nhỏ giữ đường cũ cho đơn giản.
- parallelUploads cho 1 file lớn: bật (nhanh hơn) hay tắt (ổn định, đỡ flood)?
- Có cần resume xuyên phiên (localStorage) hay chỉ resume trong phiên là đủ?
