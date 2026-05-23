import { Archive, CloudUpload, Download, FileAudio, FileText, FileUp, FileVideo, Folder, Image, Plus, RefreshCw } from "lucide-react";
import { ChangeEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { createFolder, downloadFileUrl, DriveContents, DriveFile, DriveFolder, listDriveContents, seedDemoFile, syncFilesToTelegram, uploadFile } from "../api/agent";

export function FileManager() {
  const { t } = useTranslation();
  const [contents, setContents] = useState<DriveContents>({ folders: [], files: [] });
  const [folderStack, setFolderStack] = useState<DriveFolder[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const busy = loading && (contents.files.length > 0 || contents.folders.length > 0);
  const currentFolderId = folderStack.length > 0 ? folderStack[folderStack.length - 1].id : "";

  async function refresh(folderId = currentFolderId) {
    setLoading(true);
    setError(null);
    try {
      setContents(await listDriveContents(folderId));
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

  async function syncTelegram() {
    setLoading(true);
    setError(null);
    setNotice(null);
    try {
      const result = await syncFilesToTelegram();
      setNotice(result.sync.message);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setLoading(false);
    }
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
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;

    setLoading(true);
    setError(null);
    setNotice(null);
    try {
      await uploadFile(file, currentFolderId);
      await refresh();
      setNotice(t("files.uploadDone", { name: file.name }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { refresh(""); }, []);

  const isEmpty = contents.folders.length === 0 && contents.files.length === 0;

  return (
    <section className="file-manager" id="file-manager">
      <div className="file-manager__header">
        <div>
          <h2>{t("files.title")}</h2>
          <p>{t("files.description")}</p>
          <div className="breadcrumb">
            <button onClick={goRoot}>{t("files.root")}</button>
            {folderStack.map((folder, index) => (
              <button key={folder.id} onClick={() => goBreadcrumb(index)}>/{folder.name}</button>
            ))}
          </div>
        </div>
        <div className="file-manager__actions">
          <label className={`button button--primary upload-label ${busy ? "button--disabled" : ""}`}>
            <FileUp size={17} /> {loading && isEmpty ? t("files.uploadAnyway") : t("files.upload")}
            <input className="upload-label__input" type="file" onChange={handleUpload} disabled={busy} />
          </label>
          <button className="button button--secondary" onClick={createNewFolder} disabled={loading}>
            <Folder size={17} /> {t("files.createFolder")}
          </button>
          <button className="button button--secondary" onClick={syncTelegram} disabled={loading}>
            <CloudUpload size={17} /> {t("files.syncTelegram")}
          </button>
          <button className="button button--ghost" onClick={() => refresh()} disabled={loading}>
            <RefreshCw size={17} /> {t("files.refresh")}
          </button>
          <button className="button button--secondary" onClick={createDemo} disabled={loading}>
            <Plus size={17} /> {t("files.createDemo")}
          </button>
        </div>
      </div>

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
          {contents.files.map((file) => (
            <div className="file-row" key={file.id}>
              <div className="file-row__icon">{kindIcon(file.kind)}</div>
              <div className="file-row__name">
                <strong>{file.name}</strong>
                <span>{kindLabel(file.kind)} · {file.mime_type || t("files.unknownType")}</span>
              </div>
              <div className="file-row__meta">{formatBytes(file.size)}</div>
              <div className="file-row__badge">{syncLabel(file.sync_state)}</div>
              <a className="file-row__download" href={downloadFileUrl(file.id)}>
                <Download size={15} /> {t("files.download")}
              </a>
            </div>
          ))}
        </div>
      )}
    </section>
  );
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
  const labels: Record<string, string> = { pending_telegram_upload: "Chờ đồng bộ", telegram_uploading: "Đang đồng bộ", telegram_synced: "Đã lên Telegram", telegram_upload_failed: "Lỗi đồng bộ", metadata_only: "Metadata" };
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
