import { Archive, Download, FileAudio, FileText, FileUp, FileVideo, Folder, Image, Plus, RefreshCw } from "lucide-react";
import { ChangeEvent, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { createFolder, downloadFileUrl, DriveContents, DriveFile, DriveFolder, listDriveContents, listTransfers, seedDemoFile, thumbnailUrl, Transfer, uploadFile, UploadProgress } from "../api/agent";

export function FileManager() {
  const { t } = useTranslation();
  const [contents, setContents] = useState<DriveContents>({ folders: [], files: [] });
  const [folderStack, setFolderStack] = useState<DriveFolder[]>([]);
  const [transfers, setTransfers] = useState<Transfer[]>([]);
  const [localUploads, setLocalUploads] = useState<UploadProgress[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const busy = loading && (contents.files.length > 0 || contents.folders.length > 0);
  const currentFolderId = folderStack.length > 0 ? folderStack[folderStack.length - 1].id : "";
  const transferByFile = useMemo(() => new Map(transfers.map((transfer) => [transfer.file_id, transfer])), [transfers]);

  async function refresh(folderId = currentFolderId) {
    setLoading(true);
    setError(null);
    try {
      setContents(await listDriveContents(folderId));
      const result = await listTransfers();
      setTransfers(result.transfers);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function openFolder(folder: DriveFolder) {
    setFolderStack((stack) => [...stack, folder]);
    await refresh(folder.id);
  }

  async function goRoot() {
    setFolderStack([]);
    await refresh("");
  }

  async function goBreadcrumb(index: number) {
    const nextStack = folderStack.slice(0, index + 1);
    setFolderStack(nextStack);
    await refresh(nextStack.length > 0 ? nextStack[nextStack.length - 1].id : "");
  }

  async function createNewFolder() {
    const name = window.prompt(t("files.folderNamePrompt"));
    if (!name) return;
    setLoading(true);
    setError(null);
    try {
      const result = await createFolder(name, currentFolderId);
      setContents(result.contents);
      setNotice(t("files.folderCreated", { name }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function createDemo() {
    setLoading(true);
    setError(null);
    try {
      await seedDemoFile();
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setLoading(false);
    }
  }

  async function handleUpload(event: ChangeEvent<HTMLInputElement>) {
    const selected = Array.from(event.target.files || []);
    event.target.value = "";
    if (selected.length === 0) return;
    await uploadSelectedFiles(selected, false);
  }

  async function handleFolderUpload(event: ChangeEvent<HTMLInputElement>) {
    const selected = Array.from(event.target.files || []);
    event.target.value = "";
    if (selected.length === 0) return;
    await uploadSelectedFiles(selected, true);
  }

  async function uploadSelectedFiles(selected: File[], preserveRelativePath: boolean) {
    setLoading(true);
    setError(null);
    setNotice(null);
    setLocalUploads(selected.map((file) => ({ phase: "uploading_agent", percent: 0, fileName: file.name })));
    let uploaded = 0;
    try {
      for (const file of selected) {
        const relativePath = preserveRelativePath ? getRelativePath(file) : "";
        await uploadFile(file, currentFolderId, (progress) => {
          setLocalUploads((items) => items.map((item) => item.fileName === file.name ? progress : item));
        }, relativePath);
        uploaded += 1;
      }
      await refresh();
      setNotice(t("files.multiUploadAutoSync", { count: uploaded }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
      setTimeout(() => setLocalUploads([]), 2200);
    }
  }

  useEffect(() => { refresh(""); }, []);

  useEffect(() => {
    const timer = window.setInterval(async () => {
      try {
        const result = await listTransfers();
        setTransfers(result.transfers);
        if (result.transfers.some((transfer) => transfer.phase !== "completed" && transfer.phase !== "failed")) {
          setContents(await listDriveContents(currentFolderId));
        }
      } catch {
        // Polling chỉ hỗ trợ realtime, lỗi tạm thời không chặn file manager.
      }
    }, 1500);
    return () => window.clearInterval(timer);
  }, [currentFolderId]);

  const isEmpty = contents.folders.length === 0 && contents.files.length === 0;
  const activeTransfers = transfers.filter((transfer) => transfer.phase !== "completed" || Date.now() / 1000 - transfer.updated_at < 30);

  return (
    <section className="file-manager" id="file-manager">
      <div className="file-manager__header">
        <div>
          <h2>{t("files.title")}</h2>
          <p>{t("files.description")}</p>
          <div className="breadcrumb">
            <button onClick={goRoot}>{t("files.root")}</button>
            {folderStack.map((folder, index) => <button key={folder.id} onClick={() => goBreadcrumb(index)}>/{folder.name}</button>)}
          </div>
        </div>
        <div className="file-manager__actions">
          <label className={`button button--primary upload-label ${busy ? "button--disabled" : ""}`}>
            <FileUp size={17} /> {loading && isEmpty ? t("files.uploadAnyway") : t("files.upload")}
            <input className="upload-label__input" type="file" multiple onChange={handleUpload} disabled={busy} />
          </label>
          <label className={`button button--secondary upload-label ${busy ? "button--disabled" : ""}`}>
            <Folder size={17} /> {t("files.uploadFolder")}
            <input className="upload-label__input" type="file" multiple onChange={handleFolderUpload} disabled={busy} {...({ webkitdirectory: "", directory: "" } as Record<string, string>)} />
          </label>
          <button className="button button--secondary" onClick={createNewFolder} disabled={loading}><Folder size={17} /> {t("files.createFolder")}</button>
          <button className="button button--ghost" onClick={() => refresh()} disabled={loading}><RefreshCw size={17} /> {t("files.refresh")}</button>
          <button className="button button--secondary" onClick={createDemo} disabled={loading}><Plus size={17} /> {t("files.createDemo")}</button>
        </div>
      </div>

      {localUploads.length > 0 && <div className="transfer-stack">{localUploads.map((upload) => <ProgressCard key={upload.fileName} label={`${t("files.uploadingAgent")} ${upload.fileName}`} percent={upload.percent} error={upload.error} />)}</div>}
      {activeTransfers.length > 0 && <div className="transfer-stack">{activeTransfers.map((transfer) => <ProgressCard key={transfer.id} label={transferLabel(transfer.phase)} percent={transfer.percent} error={transfer.last_error} />)}</div>}
      {notice && <div className="success-note">{notice}</div>}
      {error && <div className="error-note">{error}</div>}
      {loading && <div className="muted-box">{t("files.loading")}</div>}
      {!loading && isEmpty && <div className="muted-box">{t("files.empty")}</div>}

      {!loading && !isEmpty && (
        <div className="file-table">
          {contents.folders.map((folder) => (
            <button className="file-row file-row--folder" key={folder.id} onClick={() => openFolder(folder)}>
              <div className="file-row__icon"><Folder size={20} /></div>
              <div className="file-row__name"><strong>{folder.name}</strong><span>{t("files.folder")}</span></div>
              <div className="file-row__meta">-</div>
              <div className="file-row__badge">{t("files.localFolder")}</div>
            </button>
          ))}
          {contents.files.map((file) => {
            const transfer = transferByFile.get(file.id);
            return <div className="file-row" key={file.id}>
              <div className="file-row__icon">{file.preview_status === "ready" && file.kind === "image" ? <img src={thumbnailUrl(file.id)} alt="" /> : kindIcon(file.kind)}</div>
              <div className="file-row__name">
                <strong>{file.name}</strong>
                <span>{kindLabel(file.kind)} · {file.mime_type || t("files.unknownType")}</span>
                {transfer && transfer.phase !== "completed" && <MiniProgress percent={transfer.percent} />}
              </div>
              <div className="file-row__meta">{formatBytes(file.size)}</div>
              <div className="file-row__badge">{transfer && transfer.phase !== "completed" ? transferLabel(transfer.phase) : syncLabel(file.sync_state)}</div>
              <a className="file-row__download" href={downloadFileUrl(file.id)}><Download size={15} /> {t("files.download")}</a>
            </div>;
          })}
        </div>
      )}
    </section>
  );
}

function getRelativePath(file: File) {
  return (file as File & { webkitRelativePath?: string }).webkitRelativePath || file.name;
}

function ProgressCard({ label, percent, error }: { label: string; percent: number; error?: string }) {
  return <div className={error ? "progress-card progress-card--error" : "progress-card"}><div><strong>{label}</strong><span>{error || `${percent}%`}</span></div><MiniProgress percent={percent} /></div>;
}

function MiniProgress({ percent }: { percent: number }) {
  return <div className="mini-progress"><span style={{ width: `${Math.max(0, Math.min(100, percent))}%` }} /></div>;
}

function transferLabel(phase: string) {
  const labels: Record<string, string> = { queued: "Đang chờ", syncing_telegram: "Đang đồng bộ", completed: "Đã đồng bộ", failed: "Lỗi đồng bộ" };
  return labels[phase] || phase;
}

function kindIcon(kind: DriveFile["kind"]) {
  if (kind === "image") return <Image size={19} />;
  if (kind === "video") return <FileVideo size={19} />;
  if (kind === "audio") return <FileAudio size={19} />;
  if (kind === "archive") return <Archive size={19} />;
  return <FileText size={19} />;
}

function kindLabel(kind: string) {
  const labels: Record<string, string> = { image: "Hình ảnh", video: "Video", audio: "Âm thanh", document: "Tài liệu", archive: "Nén", other: "File" };
  return labels[kind] || "File";
}

function syncLabel(state: string) {
  const labels: Record<string, string> = { pending_telegram_upload: "Chờ đồng bộ", telegram_uploading: "Đang đồng bộ", telegram_synced: "Đã đồng bộ", telegram_upload_failed: "Lỗi đồng bộ", metadata_only: "Metadata" };
  return labels[state] || state;
}

function formatBytes(bytes: number) {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) { value /= 1024; unit += 1; }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unit]}`;
}
