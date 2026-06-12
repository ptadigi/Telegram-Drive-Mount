import { Folder, RefreshCw, RotateCcw } from "../icons";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { DriveContents, eventsUrl, listTrash, restoreFile, restoreFolder } from "../api/agent";

export function TrashView() {
  const { t } = useTranslation();
  const [contents, setContents] = useState<DriveContents>({ folders: [], files: [] });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    setLoading(true);
    setError(null);
    try {
      setContents(await listTrash());
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { refresh(); }, []);

  useEffect(() => {
    const stream = new EventSource(eventsUrl(), { withCredentials: true });
    stream.addEventListener("file.trashed", refresh);
    stream.addEventListener("folder.trashed", refresh);
    stream.addEventListener("file.restored", refresh);
    stream.addEventListener("folder.restored", refresh);
    return () => stream.close();
  }, []);

  const isEmpty = contents.folders.length === 0 && contents.files.length === 0;

  return (
    <section className="drive-browser">
      <div className="drive-browser__header">
        <div>
          <h2>{t("drive.trash")}</h2>
          <p>{t("drive.trashDesc")}</p>
        </div>
        <div className="drive-browser__actions">
          <button className="button button--ghost" onClick={refresh} disabled={loading}><RefreshCw size={15} /></button>
        </div>
      </div>
      {error && <div className="error-note">{error}</div>}
      {loading && <div className="muted-box">{t("files.loading")}</div>}
      {!loading && isEmpty && <div className="muted-box">{t("drive.trashEmpty")}</div>}
      {!loading && !isEmpty && (
        <div className="file-grid">
          {contents.folders.map((folder) => (
            <div className="drive-card drive-card--folder" key={folder.id}>
              <div className="drive-card__thumb"><Folder size={36} /></div>
              <div className="drive-card__name"><strong>{folder.name}</strong><span>{t("drive.trashedFolder")}</span></div>
              <div className="drive-card__footer">
                <button className="button button--ghost" onClick={() => restoreFolder(folder.id).then(refresh)}>
                  <RotateCcw size={14} /> {t("drive.restore")}
                </button>
              </div>
            </div>
          ))}
          {contents.files.map((file) => (
            <div className="drive-card" key={file.id}>
              <div className="drive-card__thumb">{file.kind?.toUpperCase() || "FILE"}</div>
              <div className="drive-card__name"><strong>{file.name}</strong><span>{t("drive.trashedFile")}</span></div>
              <div className="drive-card__footer">
                <button className="button button--ghost" onClick={() => restoreFile(file.id).then(refresh)}>
                  <RotateCcw size={14} /> {t("drive.restore")}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
