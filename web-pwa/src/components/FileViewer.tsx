import { Download, ExternalLink, X, ZoomIn, ZoomOut } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { AGENT_BASE_URL, DriveFile, downloadFileUrl, streamFileUrl, thumbnailUrl } from "../api/agent";

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
          <a className="button button--secondary" href={url} target="_blank" rel="noopener noreferrer"><ExternalLink size={14} /> Mở tab mới</a>
          <a className="button button--secondary" href={downloadFileUrl(file.id)}><Download size={14} /> Tải xuống</a>
          <button className="icon-button" onClick={onClose} aria-label="Đóng"><X size={16} /></button>
        </div>
      </header>
      <main className="viewer__body">
        {variant === "image" && <ImageViewer src={url} alt={file.name} poster={file.preview_status === "ready" ? thumbnailUrl(file.id) : undefined} />}
        {variant === "video" && <video controls autoPlay playsInline src={streamUrl} poster={file.preview_status === "ready" ? thumbnailUrl(file.id) : undefined}></video>}
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

function ImageViewer({ src, alt, poster }: { src: string; alt: string; poster?: string }) {
  const [zoom, setZoom] = useState(1);
  const [offset, setOffset] = useState({ x: 0, y: 0 });
  const [loaded, setLoaded] = useState(false);
  const dragRef = useRef<{ x: number; y: number; ox: number; oy: number } | null>(null);

  function clampZoom(z: number) { return Math.min(6, Math.max(1, z)); }
  function zoomBy(delta: number) {
    setZoom((z) => {
      const next = clampZoom(z + delta);
      if (next === 1) setOffset({ x: 0, y: 0 });
      return next;
    });
  }
  function onWheel(e: React.WheelEvent) {
    e.preventDefault();
    zoomBy(e.deltaY < 0 ? 0.3 : -0.3);
  }
  function onDoubleClick() { setZoom((z) => (z > 1 ? (setOffset({ x: 0, y: 0 }), 1) : 2)); }
  function onPointerDown(e: React.PointerEvent) {
    if (zoom <= 1) return;
    dragRef.current = { x: e.clientX, y: e.clientY, ox: offset.x, oy: offset.y };
    (e.target as Element).setPointerCapture(e.pointerId);
  }
  function onPointerMove(e: React.PointerEvent) {
    if (!dragRef.current) return;
    setOffset({ x: dragRef.current.ox + (e.clientX - dragRef.current.x), y: dragRef.current.oy + (e.clientY - dragRef.current.y) });
  }
  function onPointerUp() { dragRef.current = null; }

  return (
    <div className="image-viewer" onWheel={onWheel}>
      {!loaded && poster && <img className="image-viewer__poster" src={poster} alt="" aria-hidden />}
      <img
        className="image-viewer__img"
        src={src}
        alt={alt}
        draggable={false}
        onLoad={() => setLoaded(true)}
        style={{ transform: `translate(${offset.x}px, ${offset.y}px) scale(${zoom})`, cursor: zoom > 1 ? "grab" : "zoom-in" }}
        onDoubleClick={onDoubleClick}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
      />
      <div className="image-viewer__controls">
        <button className="icon-button" onClick={() => zoomBy(-0.3)} aria-label="Thu nhỏ"><ZoomOut size={16} /></button>
        <span>{Math.round(zoom * 100)}%</span>
        <button className="icon-button" onClick={() => zoomBy(0.3)} aria-label="Phóng to"><ZoomIn size={16} /></button>
      </div>
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
  if (isPublicHost()) {
    const viewer = officeViewerUrl(url);
    return (
      <iframe title={`office-${ext}`} src={viewer}></iframe>
    );
  }
  return (
    <div className="viewer__office">
      <p>Loại file <strong>{ext.replace(".", "") || mime}</strong> chưa thể xem trực tiếp khi link Agent chỉ có trong LAN.</p>
      <p>Bạn có thể tải xuống và mở bằng ứng dụng Office. Khi triển khai trên domain công khai (Cloudflare Tunnel hoặc tên miền riêng), Office Online viewer sẽ tự kích hoạt.</p>
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

function isPublicHost(): boolean {
  const host = window.location.hostname;
  if (!host) return false;
  if (host === "localhost" || host === "127.0.0.1") return false;
  if (host.startsWith("192.168.") || host.startsWith("10.")) return false;
  return true;
}

function officeViewerUrl(rawUrl: string) {
  const absolute = new URL(rawUrl, window.location.origin).toString();
  const encoded = encodeURIComponent(absolute);
  return `https://view.officeapps.live.com/op/embed.aspx?src=${encoded}`;
}

function formatBytes(bytes: number) {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) { value /= 1024; unit += 1; }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unit]}`;
}
