import { Archive, Download, FileAudio, FileText, FileVideo, Folder, Image, RefreshCw, Star } from "lucide-react";
import { useEffect, useState } from "react";
import { DriveContents, downloadFileUrl, listStarred, eventsUrl, starFile, starFolder, thumbnailUrl, DriveFile } from "../api/agent";
import { useToast } from "../state/ui";

export function StarredView() {
  const [contents, setContents] = useState<DriveContents>({ folders: [], files: [] });
  const [loading, setLoading] = useState(true);
  const toast = useToast();

  async function refresh() {
    setLoading(true);
    try {
      setContents(await listStarred());
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err), "error");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { refresh(); }, []);

  useEffect(() => {
    const stream = new EventSource(eventsUrl(), { withCredentials: true });
    stream.addEventListener("file.starred", refresh);
    stream.addEventListener("folder.starred", refresh);
    stream.addEventListener("file.updated", refresh);
    stream.addEventListener("folder.updated", refresh);
    return () => stream.close();
  }, []);

  const isEmpty = contents.folders.length === 0 && contents.files.length === 0;

  return (
    <section className="drive-browser">
      <div className="drive-browser__header">
        <div>
          <h2>CÃ³ gáº¯n dáº¥u sao</h2>
          <p>CÃ¡c file vÃ  thÆ° má»¥c báº¡n Ä‘Ã£ Ä‘Ã¡nh dáº¥u sao.</p>
        </div>
        <div className="drive-browser__actions">
          <button className="button button--ghost" onClick={refresh} disabled={loading}><RefreshCw size={15} /></button>
        </div>
      </div>
      {loading && <div className="muted-box">Äang táº£i...</div>}
      {!loading && isEmpty && <div className="muted-box">ChÆ°a cÃ³ má»¥c nÃ o Ä‘Æ°á»£c Ä‘Ã¡nh dáº¥u sao.</div>}
      {!loading && !isEmpty && (
        <div className="file-grid">
          {contents.folders.map((folder) => (
            <div className="drive-card drive-card--folder" key={folder.id}>
              <div className="drive-card__thumb"><Folder size={36} /></div>
              <div className="drive-card__name"><strong>{folder.name}</strong><span>ThÆ° má»¥c</span></div>
              <div className="drive-card__footer">
                <button className="button button--ghost" onClick={() => starFolder(folder.id, false).then(refresh)}><Star size={14} /> Bá» sao</button>
              </div>
            </div>
          ))}
          {contents.files.map((file) => (
            <div className="drive-card" key={file.id}>
              <div className="drive-card__thumb">{file.preview_status === "ready" && file.kind === "image" ? <img src={thumbnailUrl(file.id)} alt="" /> : kindIcon(file.kind)}</div>
              <div className="drive-card__name"><strong>{file.name}</strong><span>{kindLabel(file.kind)} Â· {formatBytes(file.size)}</span></div>
              <div className="drive-card__footer">
                <button className="button button--ghost" onClick={() => starFile(file.id, false).then(refresh)}><Star size={14} /> Bá» sao</button>
                <a className="drive-card__action" href={downloadFileUrl(file.id)}><Download size={14} /></a>
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

function kindIcon(kind: DriveFile["kind"]) {
  if (kind === "image") return <Image size={32} />;
  if (kind === "video") return <FileVideo size={32} />;
  if (kind === "audio") return <FileAudio size={32} />;
  if (kind === "archive") return <Archive size={32} />;
  return <FileText size={32} />;
}

function kindLabel(kind: string) {
  const labels: Record<string, string> = { image: "HÃ¬nh áº£nh", video: "Video", audio: "Ã‚m thanh", document: "TÃ i liá»‡u", archive: "NÃ©n", other: "File" };
  return labels[kind] || "File";
}

function formatBytes(bytes: number) {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) { value /= 1024; unit += 1; }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unit]}`;
}
