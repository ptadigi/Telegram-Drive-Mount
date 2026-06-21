import { Image as ImageIcon } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useToast } from "../state/ui";
import {
  clearPhotoDirectory,
  photoSyncSupported,
  pickPhotoDirectory,
  savedDirectoryName,
  syncFromSavedDirectory,
  uploadFiles,
  SyncProgress,
} from "../state/photoSync";

// PhotoBackupPanel: one-way phone -> Telegram Drive "Camera" backup.
//
// Android/desktop Chromium: pick a folder once (handle persisted), then "Sao
// lưu ngay" scans + uploads new photos (dedup by SHA-256).
// iOS/Safari: no directory picker -> a file picker fallback + Apple Shortcut
// guidance for scheduled automation. Honest about the platform limits: this is
// NOT silent background sync on iOS.
export function PhotoBackupPanel() {
  const supported = photoSyncSupported();
  const [dirName, setDirName] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [includeVideos, setIncludeVideos] = useState(false);
  const [progress, setProgress] = useState<SyncProgress | null>(null);
  const fileRef = useRef<HTMLInputElement | null>(null);
  const toast = useToast();

  useEffect(() => {
    if (supported) savedDirectoryName().then(setDirName).catch(() => undefined);
  }, [supported]);

  async function pick() {
    try {
      const name = await pickPhotoDirectory();
      setDirName(name);
      toast(`Đã chọn thư mục: ${name}`, "success");
    } catch {
      // user cancelled picker — no toast
    }
  }

  async function syncNow() {
    setBusy(true);
    setProgress(null);
    try {
      const result = await syncFromSavedDirectory({ includeVideos, onProgress: setProgress });
      if (result.phase === "error") toast(result.error || "Sao lưu thất bại", "error");
      else toast(`Đã sao lưu: ${result.uploaded} mới, ${result.skipped} bỏ qua${result.failed ? `, ${result.failed} lỗi` : ""}`, result.failed ? "info" : "success");
    } finally {
      setBusy(false);
    }
  }

  async function onFilePick(e: React.ChangeEvent<HTMLInputElement>) {
    const files = Array.from(e.target.files || []);
    e.target.value = "";
    if (files.length === 0) return;
    setBusy(true);
    setProgress(null);
    try {
      const result = await uploadFiles(files, { onProgress: setProgress });
      toast(`Đã sao lưu: ${result.uploaded} mới, ${result.skipped} bỏ qua${result.failed ? `, ${result.failed} lỗi` : ""}`, result.failed ? "info" : "success");
    } finally {
      setBusy(false);
    }
  }

  async function forget() {
    await clearPhotoDirectory();
    setDirName(null);
    toast("Đã quên thư mục ảnh", "info");
  }

  const pct = progress && progress.toUpload > 0 ? Math.round((progress.uploaded / progress.toUpload) * 100) : 0;

  return (
    <section className="settings-view photo-backup">
      <header className="settings-view__header">
        <div>
          <h2><ImageIcon size={20} /> Sao lưu ảnh điện thoại</h2>
          <p>Đồng bộ ảnh từ điện thoại lên thư mục "Camera" trên Telegram Drive. Chỉ tải ảnh mới (bỏ qua ảnh đã có).</p>
        </div>
      </header>

      {supported ? (
        <div className="photo-backup__body">
          {dirName ? (
            <p className="form-hint">Thư mục đang chọn: <strong>{dirName}</strong></p>
          ) : (
            <p className="form-hint">Chọn thư mục ảnh (ví dụ DCIM/Camera). Chỉ cần chọn 1 lần.</p>
          )}
          <label className="photo-backup__opt">
            <input type="checkbox" checked={includeVideos} onChange={(e) => setIncludeVideos(e.target.checked)} /> Bao gồm cả video
          </label>
          <div className="photo-backup__actions">
            <button className="button button--secondary" onClick={pick} disabled={busy}>{dirName ? "Đổi thư mục" : "Chọn thư mục ảnh"}</button>
            <button className="button button--primary" onClick={syncNow} disabled={busy || !dirName}>{busy ? "Đang sao lưu..." : "Sao lưu ngay"}</button>
            {dirName && <button className="button button--ghost" onClick={forget} disabled={busy}>Quên thư mục</button>}
          </div>
        </div>
      ) : (
        <div className="photo-backup__body">
          <p className="form-hint">Trình duyệt này (iPhone/Safari) không cho chọn cả thư mục. Bạn có 2 cách:</p>
          <ol className="photo-backup__steps">
            <li>Bấm "Chọn ảnh để sao lưu" và chọn ảnh thủ công.</li>
            <li>Hoặc dùng <strong>Apple Shortcuts</strong> (Tự động hoá theo giờ) để tự sao lưu định kỳ — xem hướng dẫn trong tài liệu.</li>
          </ol>
          <div className="photo-backup__actions">
            <button className="button button--primary" onClick={() => fileRef.current?.click()} disabled={busy}>{busy ? "Đang sao lưu..." : "Chọn ảnh để sao lưu"}</button>
            <input ref={fileRef} className="visually-hidden" type="file" accept="image/*,video/*" multiple onChange={onFilePick} />
          </div>
        </div>
      )}

      {progress && progress.phase !== "done" && progress.phase !== "error" && (
        <div className="photo-backup__progress">
          <div className="photo-backup__bar"><span style={{ width: `${pct}%` }} /></div>
          <span className="form-hint">
            {progress.phase === "scanning" && "Đang quét thư mục..."}
            {progress.phase === "hashing" && `Đang kiểm tra ${progress.scanned} ảnh...`}
            {progress.phase === "checking" && "Đang đối chiếu với server..."}
            {progress.phase === "uploading" && `Đang tải ${progress.uploaded}/${progress.toUpload}${progress.current ? ` · ${progress.current}` : ""}`}
          </span>
        </div>
      )}
    </section>
  );
}
