import { Download, X } from "lucide-react";
import { useEffect, useState } from "react";
import { AGENT_BASE_URL, DriveFile, downloadFileUrl, streamFileUrl } from "../api/agent";

type Props = {
  file: DriveFile | null;
  onClose: () => void;
};

export function FileViewer({ file, onClose }: Props) {
  const [textContent, setTextContent] = useState<string | null>(null);
  const [loadingText, setLoadingText] = useState(false);
  const [textError, setTextError] = useState<string | null>(null);

  const url = file ? `${AGENT_BASE_URL}/v1/files/download?id=${encodeURIComponent(file.id)}` : "";
  const streamUrl = file ? streamFileUrl(file.id) : "";
  const ext = (file?.extension || "").toLowerCase();
  const mime = (file?.mime_type || "").toLowerCase();
  const variant = file ? detectVariant(file) : "binary";

  useEffect(() => {
    if (!file) return;
    const handler = (event: KeyboardEvent) => { if (event.key === "Escape") onClose(); };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [file, onClose]);

  useEffect(() => {
    setTextContent(null);
    setTextError(null);
    if (!file) return;
    if (variant !== "markdown" && variant !== "text") return;
    setLoadingText(true);
    fetch(url)
      .then(async (response) => {
        if (!response.ok) throw new Error("Không tải được nội dung file");
        const data = await response.text();
        setTextContent(data);
      })
      .catch((err) => setTextError(err instanceof Error ? err.message : String(err)))
      .finally(() => setLoadingText(false));
  }, [file, url, variant]);

  if (!file) return null;
  return (
    <div className="viewer" role="dialog" aria-modal="true" onClick={(event) => event.target === event.currentTarget && onClose()}>
      <header className="viewer__header">
        <div>
          <strong>{file.name}</strong>
          <span>{(file.kind || "file").toUpperCase()} · {formatBytes(file.size)}</span>
        </div>
        <div className="viewer__actions">
          <a className="button button--secondary" href={downloadFileUrl(file.id)}><Download size={14} /> Tải xuống</a>
          <button className="icon-button" onClick={onClose} aria-label="Đóng"><X size={16} /></button>
        </div>
      </header>
      <main className="viewer__body">
        {variant === "image" && <img src={url} alt={file.name} />}
        {variant === "video" && <video controls src={streamUrl}></video>}
        {variant === "audio" && <audio controls src={streamUrl}></audio>}
        {variant === "pdf" && <iframe title={file.name} src={url}></iframe>}
        {variant === "markdown" && <ViewerText content={textContent} loading={loadingText} error={textError} mono={false} />}
        {variant === "text" && <ViewerText content={textContent} loading={loadingText} error={textError} mono={true} />}
        {variant === "office" && <OfficeFallback url={url} ext={ext} mime={mime} />}
        {variant === "binary" && <UnsupportedFallback ext={ext} />}
      </main>
    </div>
  );
}

function ViewerText({ content, loading, error, mono }: { content: string | null; loading: boolean; error: string | null; mono: boolean; }) {
  if (loading) return <div className="viewer__notice">Đang tải nội dung...</div>;
  if (error) return <div className="viewer__notice viewer__notice--error">{error}</div>;
  if (content === null) return null;
  return <pre className={mono ? "viewer__text viewer__text--mono" : "viewer__text"}>{content}</pre>;
}

function OfficeFallback({ url, ext, mime }: { url: string; ext: string; mime: string; }) {
  return (
    <div className="viewer__office">
      <p>Loại file <strong>{ext.replace(".", "") || mime}</strong> chưa thể xem trực tiếp khi link Agent chỉ có trong LAN.</p>
      <p>Bạn có thể tải xuống và mở bằng ứng dụng Office. Nếu bật Cloudflare Tunnel hoặc tên miền chia sẻ, hệ thống sẽ tự bật xem online ở phiên bản tiếp theo.</p>
      <a className="button button--primary" href={url}><Download size={14} /> Tải xuống</a>
    </div>
  );
}

function UnsupportedFallback({ ext }: { ext: string }) {
  return (
    <div className="viewer__office">
      <p>Loại file <strong>{ext.replace(".", "") || "này"}</strong> chưa hỗ trợ xem trực tiếp.</p>
    </div>
  );
}

function detectVariant(file: DriveFile): "image" | "video" | "audio" | "pdf" | "markdown" | "text" | "office" | "binary" {
  if (file.kind === "image") return "image";
  if (file.kind === "video") return "video";
  if (file.kind === "audio") return "audio";
  const ext = (file.extension || "").toLowerCase();
  const mime = (file.mime_type || "").toLowerCase();
  if (ext === ".pdf" || mime === "application/pdf") return "pdf";
  if (ext === ".md" || ext === ".markdown" || mime === "text/markdown") return "markdown";
  if (ext === ".txt" || ext === ".log" || ext === ".json" || ext === ".csv" || ext === ".sql" || ext === ".yaml" || ext === ".yml" || ext === ".xml" || ext === ".html" || ext === ".js" || ext === ".ts" || ext === ".css") return "text";
  if (mime.startsWith("text/")) return "text";
  if ([".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx"].includes(ext)) return "office";
  return "binary";
}

function formatBytes(bytes: number) {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) { value /= 1024; unit += 1; }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unit]}`;
}
