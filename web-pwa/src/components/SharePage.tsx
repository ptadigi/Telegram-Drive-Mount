import { Cloud, Download, FileText, Lock } from "lucide-react";
import { useEffect, useState } from "react";
import { AGENT_BASE_URL } from "../api/agent";

type ShareInfo = {
  share: { slug: string; target_kind: string; has_password: boolean; expires_at?: number; max_downloads?: number; access_count?: number };
  file?: { name: string; size: number; mime_type?: string; kind?: string; updated_at?: number };
  requires_password?: boolean;
};

export function SharePage({ slug }: { slug: string }) {
  const [data, setData] = useState<ShareInfo | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [password, setPassword] = useState("");
  const [unlocking, setUnlocking] = useState(false);

  async function load(pwd?: string) {
    setLoading(true);
    setError(null);
    try {
      const url = `${AGENT_BASE_URL}/share/${encodeURIComponent(slug)}${pwd ? `?password=${encodeURIComponent(pwd)}` : ""}`;
      const response = await fetch(url);
      const body = await response.json().catch(() => null);
      if (!response.ok) throw new Error((body && body.error) || `Lỗi ${response.status}`);
      setData(body as ShareInfo);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, [slug]);

  async function unlock() {
    if (!password) return;
    setUnlocking(true);
    setError(null);
    try {
      const response = await fetch(`${AGENT_BASE_URL}/share/${encodeURIComponent(slug)}/unlock`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ password }),
      });
      const body = await response.json().catch(() => null);
      if (!response.ok) throw new Error((body && body.error) || `Mật khẩu chưa đúng`);
      await load(password);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setUnlocking(false);
    }
  }

  return (
    <main className="share-page">
      <header className="share-page__brand"><Cloud size={20} /> <strong>Ổ Đĩa Cloud Ảo</strong></header>
      <section className="share-page__card">
        {loading && <div className="muted-box">Đang tải link chia sẻ...</div>}
        {!loading && error && <div className="error-note">{error}</div>}
        {!loading && data?.requires_password && (
          <div className="share-page__locked">
            <Lock size={28} />
            <h1>Link cần mật khẩu</h1>
            <p>Vui lòng nhập mật khẩu để mở link.</p>
            <input value={password} onChange={(event) => setPassword(event.target.value)} placeholder="Mật khẩu" type="password" />
            <button className="button button--primary" onClick={unlock} disabled={unlocking}>{unlocking ? "Đang mở..." : "Mở khóa"}</button>
          </div>
        )}
        {!loading && data && data.file && (
          <div className="share-page__file">
            <SharePreview slug={slug} file={data.file} password={password} />
            <h1>{data.file.name}</h1>
            <p>{formatBytes(data.file.size)} · {data.file.mime_type || data.file.kind || "File"}</p>
            <a className="button button--primary" href={`${AGENT_BASE_URL}/share/${encodeURIComponent(slug)}/raw${password ? `?password=${encodeURIComponent(password)}` : ""}`}>
              <Download size={16} /> Tải xuống
            </a>
            {data.share.expires_at && data.share.expires_at > 0 && <p className="muted-text">Hết hạn lúc {new Date(data.share.expires_at * 1000).toLocaleString("vi-VN")}</p>}
            {data.share.max_downloads && data.share.max_downloads > 0 && <p className="muted-text">Đã tải {data.share.access_count || 0}/{data.share.max_downloads} lượt</p>}
          </div>
        )}
        {!loading && data && data.share.target_kind === "folder" && !data.requires_password && (
          <div className="share-page__file">
            <div className="share-page__icon"><FileText size={36} /></div>
            <h1>Thư mục chia sẻ</h1>
            <p>Tải toàn bộ thư mục dưới dạng tệp ZIP.</p>
            <a className="button button--primary" href={`${AGENT_BASE_URL}/share/${encodeURIComponent(slug)}/raw${password ? `?password=${encodeURIComponent(password)}` : ""}`}>
              <Download size={16} /> Tải ZIP
            </a>
          </div>
        )}
      </section>
      <footer className="share-page__footer">Powered by Ổ Đĩa Cloud Ảo · Telegram làm kho lưu trữ</footer>
    </main>
  );
}

function SharePreview({ slug, file, password }: { slug: string; file: { name: string; mime_type?: string; kind?: string }; password: string }) {
  const rawUrl = `${AGENT_BASE_URL}/share/${encodeURIComponent(slug)}/raw?disposition=inline${password ? `&password=${encodeURIComponent(password)}` : ""}`;
  const name = (file.name || "").toLowerCase();
  const mime = (file.mime_type || "").toLowerCase();
  const kind = file.kind || "";
  const ext = name.includes(".") ? name.slice(name.lastIndexOf(".")) : "";

  const isImage = kind === "image" || mime.startsWith("image/");
  const isVideo = kind === "video" || mime.startsWith("video/");
  const isAudio = kind === "audio" || mime.startsWith("audio/");
  const isPdf = ext === ".pdf" || mime === "application/pdf";

  if (isImage) return <div className="share-preview"><img src={rawUrl} alt={file.name} /></div>;
  if (isVideo) return <div className="share-preview"><video src={rawUrl} controls playsInline /></div>;
  if (isAudio) return <div className="share-preview share-preview--audio"><audio src={rawUrl} controls /></div>;
  if (isPdf) return <div className="share-preview share-preview--pdf"><iframe title={file.name} src={rawUrl} /></div>;
  return <div className="share-page__icon"><FileText size={36} /></div>;
}

function formatBytes(bytes: number) {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) { value /= 1024; unit += 1; }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unit]}`;
}
