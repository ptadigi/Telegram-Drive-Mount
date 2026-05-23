import { CloudUpload, Download, FileUp, FileText, Plus, RefreshCw } from "lucide-react";
import { ChangeEvent, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { DriveFile, downloadFileUrl, listFiles, seedDemoFile, syncFilesToTelegram, uploadFile } from "../api/agent";

export function FileManager() {
  const { t } = useTranslation();
  const [files, setFiles] = useState<DriveFile[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  async function refresh() {
    setLoading(true);
    setError(null);
    try {
      const result = await listFiles();
      setFiles(result.files);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function syncTelegram() {
    setLoading(true);
    setError(null);
    setNotice(null);
    try {
      const result = await syncFilesToTelegram();
      setFiles(result.files);
      setNotice(result.sync.message);
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
      const result = await seedDemoFile();
      setFiles(result.files);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function handleUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;

    setLoading(true);
    setError(null);
    try {
      await uploadFile(file);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setLoading(false);
    } finally {
      event.target.value = "";
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  return (
    <section className="file-manager">
      <div className="file-manager__header">
        <div>
          <h2>{t("files.title")}</h2>
          <p>{t("files.description")}</p>
        </div>
        <div className="file-manager__actions">
          <input
            ref={fileInputRef}
            className="visually-hidden"
            type="file"
            onChange={handleUpload}
          />
          <button className="button button--primary" onClick={() => fileInputRef.current?.click()} disabled={loading}>
            <FileUp size={17} /> {t("files.upload")}
          </button>
          <button className="button button--secondary" onClick={syncTelegram} disabled={loading}>
            <CloudUpload size={17} /> {t("files.syncTelegram")}
          </button>
          <button className="button button--ghost" onClick={refresh} disabled={loading}>
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
      {!loading && files.length === 0 && <div className="muted-box">{t("files.empty")}</div>}

      {!loading && files.length > 0 && (
        <div className="file-table">
          {files.map((file) => (
            <div className="file-row" key={file.id}>
              <div className="file-row__icon"><FileText size={18} /></div>
              <div className="file-row__name">
                <strong>{file.name}</strong>
                <span>{file.mime_type || t("files.unknownType")}</span>
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

function syncLabel(state: string) {
  const labels: Record<string, string> = {
    pending_telegram_upload: "Chờ đồng bộ",
    telegram_uploading: "Đang đồng bộ",
    telegram_synced: "Đã lên Telegram",
    telegram_upload_failed: "Lỗi đồng bộ",
    metadata_only: "Metadata",
  };
  return labels[state] || state;
}

function formatBytes(bytes: number) {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unit]}`;
}
