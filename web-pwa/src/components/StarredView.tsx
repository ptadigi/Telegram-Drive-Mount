import { Archive, Download, FileAudio, FileText, FileVideo, Folder, Image, RefreshCw, Star } from "lucide-react";
import { useEffect, useState } from "react";
import { DriveContents, downloadFileUrl, listStarred, eventsUrl, starFile, starFolder, thumbnailUrl, DriveFile } from "../api/agent";
import { useToast } from "../state/ui";
import { useRevalidate } from "../state/revalidate";

export function StarredView() {
  const [contents, setContents] = useState<DriveContents>({ folders: [], files: [] });
  const [loading, setLoading] = useState(true);
  const toast = useToast();

  async function refresh(opts?: { silent?: boolean }) {
    const silent = opts?.silent ?? false;
    if (!silent) setLoading(true);
    try {
      setContents(await listStarred());
    } catch (err) {
      if (!silent) toast(err instanceof Error ? err.message : String(err), "error");
    } finally {
      if (!silent) setLoading(false);
    }
  }

  useEffect(() => { refresh(); }, []);

  useRevalidate(() => refresh({ silent: true }), {
    eventsUrl: eventsUrl(),
    sseEvents: ["file.starred", "folder.starred", "file.updated", "folder.updated"],
    pollMs: 25000,
  });

  const isEmpty = contents.folders.length === 0 && contents.files.length === 0;

  return (
    <section className="drive-browser">
      <div className="drive-browser__header">
        <div>
          <h2>Có gắn dấu sao</h2>
          <p>Các file và thư mục bạn đã đánh dấu sao.</p>
        </div>
        <div className="drive-browser__actions">
          <button className="button button--ghost" onClick={() => refresh()} disabled={loading}><RefreshCw size={15} /></button>
        </div>
      </div>
      {loading && <div className="muted-box">Đang tải...</div>}
      {!loading && isEmpty && <div className="muted-box">Chưa có mục nào được đánh dấu sao.</div>}
      {!loading && !isEmpty && (
        <div className="file-grid">
          {contents.folders.map((folder) => (
            <div className="drive-card drive-card--folder" key={folder.id}>
              <div className="drive-card__thumb"><Folder size={36} /></div>
              <div className="drive-card__name"><strong>{folder.name}</strong><span>Thư mục</span></div>
              <div className="drive-card__footer">
                <button className="button button--ghost" onClick={() => starFolder(folder.id, false).then(() => refresh())}><Star size={14} /> Bỏ sao</button>
              </div>
            </div>
          ))}
          {contents.files.map((file) => (
            <div className="drive-card" key={file.id}>
              <div className="drive-card__thumb">{file.preview_status === "ready" && file.kind === "image" ? <img src={thumbnailUrl(file.id)} alt="" /> : kindIcon(file.kind)}</div>
              <div className="drive-card__name"><strong>{file.name}</strong><span>{kindLabel(file.kind)} · {formatBytes(file.size)}</span></div>
              <div className="drive-card__footer">
                <button className="button button--ghost" onClick={() => starFile(file.id, false).then(() => refresh())}><Star size={14} /> Bỏ sao</button>
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
  const labels: Record<string, string> = { image: "Hình ảnh", video: "Video", audio: "Âm thanh", document: "Tài liệu", archive: "Nén", other: "File" };
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
