import { Archive, Download, FileAudio, FileText, FileVideo, Folder, Image, Search } from "../icons";
import { useEffect, useState } from "react";
import { DriveFile, DriveFolder, downloadFileUrl, search, thumbnailUrl } from "../api/agent";
import { useToast } from "../state/ui";

type Props = {
  query: string;
};

export function SearchView({ query }: Props) {
  const [folders, setFolders] = useState<DriveFolder[]>([]);
  const [files, setFiles] = useState<DriveFile[]>([]);
  const [loading, setLoading] = useState(false);
  const toast = useToast();

  useEffect(() => {
    if (!query) {
      setFolders([]);
      setFiles([]);
      return;
    }
    const controller = new AbortController();
    const handle = window.setTimeout(async () => {
      setLoading(true);
      try {
        const result = await search(query, controller.signal);
        setFolders(result.folders);
        setFiles(result.files);
      } catch (err) {
        if ((err as Error).name !== "AbortError") {
          toast(err instanceof Error ? err.message : String(err), "error");
        }
      } finally {
        setLoading(false);
      }
    }, 250);
    return () => {
      controller.abort();
      window.clearTimeout(handle);
    };
  }, [query, toast]);

  if (!query) {
    return (
      <section className="drive-browser">
        <div className="drive-browser__header">
          <div>
            <h2>Tìm trong Drive</h2>
            <p>Nhập từ khóa vào thanh tìm kiếm để tìm file và thư mục theo tên.</p>
          </div>
        </div>
        <div className="muted-box drop-hint"><Search size={20} /> Bắt đầu bằng cách gõ từ khóa.</div>
      </section>
    );
  }

  const isEmpty = folders.length === 0 && files.length === 0;

  return (
    <section className="drive-browser">
      <div className="drive-browser__header">
        <div>
          <h2>Kết quả cho "{query}"</h2>
          <p>{folders.length} thư mục · {files.length} file</p>
        </div>
      </div>
      {loading && <div className="muted-box">Đang tìm...</div>}
      {!loading && isEmpty && <div className="muted-box">Không có mục nào khớp với từ khóa.</div>}
      {!loading && !isEmpty && (
        <div className="file-grid">
          {folders.map((folder) => (
            <div className="drive-card drive-card--folder" key={folder.id}>
              <div className="drive-card__thumb"><Folder size={36} /></div>
              <div className="drive-card__name"><strong>{folder.name}</strong><span>Thư mục</span></div>
            </div>
          ))}
          {files.map((file) => (
            <div className="drive-card" key={file.id}>
              <div className="drive-card__thumb">{file.preview_status === "ready" && file.kind === "image" ? <img src={thumbnailUrl(file.id)} alt="" /> : kindIcon(file.kind)}</div>
              <div className="drive-card__name"><strong>{file.name}</strong><span>{kindLabel(file.kind)} · {formatBytes(file.size)}</span></div>
              <div className="drive-card__footer">
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
