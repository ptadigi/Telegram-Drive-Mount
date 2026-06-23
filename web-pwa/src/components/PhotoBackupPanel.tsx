import { Image as ImageIcon } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useToast } from "../state/ui";
import { AGENT_BASE_URL, createApiToken } from "../api/agent";
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
  const [shortcut, setShortcut] = useState<{ url: string; token: string } | null>(null);
  const [shortcutBusy, setShortcutBusy] = useState(false);
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

  async function makeShortcutToken() {
    setShortcutBusy(true);
    try {
      const res = await createApiToken("iOS Photo Backup");
      const base = AGENT_BASE_URL || window.location.origin;
      setShortcut({ url: `${base}/v1/photos/upload`, token: res.token });
      toast("Đã tạo token cho Shortcut. Lưu lại — token chỉ hiện 1 lần.", "success");
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err), "error");
    } finally {
      setShortcutBusy(false);
    }
  }

  async function copyText(label: string, text: string) {
    try {
      await navigator.clipboard.writeText(text);
      toast(`Đã copy ${label}`, "success");
    } catch {
      toast("Không copy được, hãy chọn thủ công", "info");
    }
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

          <div className="photo-backup__ios-block">
            <strong>Cách 1 — Chọn ảnh thủ công</strong>
            <p className="form-hint">Chọn ảnh ngay trong app, tải lên thư mục Camera.</p>
            <div className="photo-backup__actions">
              <button className="button button--secondary" onClick={() => fileRef.current?.click()} disabled={busy}>{busy ? "Đang sao lưu..." : "Chọn ảnh để sao lưu"}</button>
              <input ref={fileRef} className="visually-hidden" type="file" accept="image/*,video/*" multiple onChange={onFilePick} />
            </div>
          </div>

          <div className="photo-backup__ios-block">
            <strong>Cách 2 — Tự động hoá bằng Apple Shortcuts (khuyên dùng)</strong>
            <p className="form-hint">Tạo 1 Shortcut chạy theo giờ để iPhone tự sao lưu ảnh mới mà không cần mở app. Cấu hình 1 lần.</p>
            {!shortcut ? (
              <button className="button button--primary" onClick={makeShortcutToken} disabled={shortcutBusy}>{shortcutBusy ? "Đang tạo..." : "Tạo token cho Shortcut"}</button>
            ) : (
              <div className="photo-backup__shortcut">
                <label className="form-hint">URL tải lên
                  <div className="photo-backup__copyrow">
                    <code>{shortcut.url}</code>
                    <button className="button button--ghost" onClick={() => copyText("URL", shortcut.url)}>Copy</button>
                  </div>
                </label>
                <label className="form-hint">Token (Authorization: <code>Device &lt;token&gt;</code>) — chỉ hiện 1 lần
                  <div className="photo-backup__copyrow">
                    <code className="photo-backup__token">{shortcut.token}</code>
                    <button className="button button--ghost" onClick={() => copyText("token", shortcut.token)}>Copy</button>
                  </div>
                </label>
                <ol className="photo-backup__steps">
                  <li>Mở app <strong>Shortcuts</strong> → tab <strong>Automation</strong> → tạo <strong>Personal Automation</strong> → <strong>Time of Day</strong> (vd 1 giờ/lần), tắt "Ask Before Running".</li>
                  <li>Thêm action <strong>Find Photos</strong>: lọc "Creation Date is within last 1 day" (hoặc khoảng trùng với lịch chạy).</li>
                  <li>Thêm <strong>Repeat with Each</strong> → bên trong thêm <strong>Get Contents of URL</strong>:
                    <br/>Method <strong>POST</strong>, URL dán ở trên, Headers thêm <code>Authorization = Device &lt;token&gt;</code>, Request Body <strong>Form</strong>, thêm field <code>file</code> kiểu <em>File</em> = "Repeat Item".</li>
                  <li>Lưu. Lần chạy đầu chọn <strong>Allow Always</strong>. Ảnh trùng sẽ tự bị bỏ qua trên server.</li>
                </ol>
                <p className="form-hint">Lưu ý: automation theo giờ chạy đáng tin khi máy đang mở/dùng. Đây là cơ chế của iOS, không phải lỗi app.</p>
              </div>
            )}
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
