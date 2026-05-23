import { FileText, Plus, RefreshCw } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { DriveFile, listFiles, seedDemoFile } from "../api/agent";

export function FileManager() {
  const { t } = useTranslation();
  const [files, setFiles] = useState<DriveFile[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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
          <button className="button button--ghost" onClick={refresh} disabled={loading}>
            <RefreshCw size={17} /> {t("files.refresh")}
          </button>
          <button className="button button--primary" onClick={createDemo} disabled={loading}>
            <Plus size={17} /> {t("files.createDemo")}
          </button>
        </div>
      </div>

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
              <div className="file-row__badge">{file.sync_state}</div>
            </div>
          ))}
        </div>
      )}
    </section>
  );
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
