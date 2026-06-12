import { Copy, Link2, RefreshCw, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { createShare, deleteShare, getShareConfig, listShares, shareLink, ShareConfig, updateShare, Share } from "../api/agent";

type Props = {
  open: boolean;
  onClose: () => void;
  targetKind: "file" | "folder";
  targetId: string;
  targetName: string;
};

export function ShareDialog({ open, onClose, targetKind, targetId, targetName }: Props) {
  const [shares, setShares] = useState<Share[]>([]);
  const [config, setConfig] = useState<ShareConfig | null>(null);
  const [password, setPassword] = useState("");
  const [expiresIn, setExpiresIn] = useState(0);
  const [maxDownloads, setMaxDownloads] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setError(null);
    setNotice(null);
    refresh();
  }, [open, targetId]);

  async function refresh() {
    setLoading(true);
    setError(null);
    try {
      const cfg = await getShareConfig();
      setConfig(cfg.config);
      const list = await listShares(targetKind, targetId);
      setShares(list.shares);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function handleCreate() {
    setLoading(true);
    setError(null);
    setNotice(null);
    try {
      const result = await createShare(targetKind, targetId, password, expiresIn, maxDownloads);
      setShares((prev) => [result.share, ...prev]);
      setPassword("");
      setNotice(`Đã tạo link ${result.share.slug}.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function handleRevoke(share: Share) {
    setLoading(true);
    try {
      const result = await updateShare(share.id, { revoked: !share.revoked });
      setShares((prev) => prev.map((item) => item.id === share.id ? result.share : item));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(share: Share) {
    if (!window.confirm(`Xóa link ${share.slug}?`)) return;
    setLoading(true);
    try {
      await deleteShare(share.id);
      setShares((prev) => prev.filter((item) => item.id !== share.id));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function handleCopy(share: Share) {
    const link = shareLink(config, share.slug);
    try {
      await navigator.clipboard.writeText(link);
      setNotice(`Đã sao chép link ${link}.`);
    } catch {
      window.prompt("Sao chép link bên dưới:", link);
    }
  }

  if (!open) return null;
  return (
    <div className="modal" onClick={(event) => event.target === event.currentTarget && onClose()}>
      <div className="modal__panel">
        <header className="modal__header">
          <div>
            <h2>Chia sẻ {targetName}</h2>
            <p>Tạo link để mọi người mở qua domain hoặc LAN, không cần Telegram.</p>
          </div>
          <button className="icon-button" onClick={onClose}>×</button>
        </header>
        <section className="share-config-summary">
          {config ? <span>Chế độ: {config.mode.toUpperCase()} · {config.health_ok ? "Sẵn sàng" : "Chưa sẵn sàng"}</span> : <span>Đang tải cấu hình chia sẻ...</span>}
        </section>
        <section className="share-create">
          <label>
            <span>Mật khẩu (tùy chọn)</span>
            <input value={password} onChange={(event) => setPassword(event.target.value)} placeholder="Để trống nếu không cần mật khẩu" />
          </label>
          <label>
            <span>Hết hạn sau</span>
            <select value={expiresIn} onChange={(event) => setExpiresIn(Number(event.target.value))}>
              <option value={0}>Không hết hạn</option>
              <option value={3600}>1 giờ</option>
              <option value={86400}>1 ngày</option>
              <option value={604800}>7 ngày</option>
              <option value={2592000}>30 ngày</option>
            </select>
          </label>
          <label>
            <span>Giới hạn lượt tải</span>
            <input type="number" min={0} value={maxDownloads} onChange={(event) => setMaxDownloads(Math.max(0, Number(event.target.value)))} />
          </label>
          <button className="button button--primary" onClick={handleCreate} disabled={loading}><Link2 size={15} /> Tạo link</button>
        </section>
        {notice && <div className="success-note">{notice}</div>}
        {error && <div className="error-note">{error}</div>}
        <section className="share-list">
          <header>
            <strong>Link đã tạo</strong>
            <button className="button button--ghost" onClick={refresh} disabled={loading}><RefreshCw size={14} /></button>
          </header>
          {shares.length === 0 && <div className="muted-box">Chưa có link nào cho mục này.</div>}
          {shares.map((share) => (
            <div className={`share-row ${share.revoked ? "share-row--revoked" : ""}`} key={share.id}>
              <div>
                <strong>{shareLink(config, share.slug)}</strong>
                <span>{statusLabel(share)}</span>
              </div>
              <div className="share-row__actions">
                <button className="button button--ghost" onClick={() => handleCopy(share)}><Copy size={14} /> Sao chép</button>
                <button className="button button--ghost" onClick={() => handleRevoke(share)}>{share.revoked ? "Bật lại" : "Thu hồi"}</button>
                <button className="button button--ghost" onClick={() => handleDelete(share)}><Trash2 size={14} /> Xóa</button>
              </div>
            </div>
          ))}
        </section>
      </div>
    </div>
  );
}

function statusLabel(share: Share) {
  if (share.revoked) return "Đã thu hồi";
  if (share.expires_at && share.expires_at > 0) {
    const remaining = share.expires_at - Math.floor(Date.now() / 1000);
    if (remaining <= 0) return "Đã hết hạn";
    return `Còn ${formatDuration(remaining)} · ${share.access_count} lượt`;
  }
  return `Không hết hạn · ${share.access_count} lượt`;
}

function formatDuration(seconds: number) {
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes} phút`;
  const hours = Math.round(minutes / 60);
  if (hours < 48) return `${hours} giờ`;
  const days = Math.round(hours / 24);
  return `${days} ngày`;
}
