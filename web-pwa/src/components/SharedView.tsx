import { BarChart2, Copy, ExternalLink, Eye, Link2, RefreshCw, Trash2 } from "../icons";
import { useEffect, useState } from "react";
import {
  deleteShare, eventsUrl, getShareAccess, getShareConfig, listMyShares, shareLink,
  ShareAccessStats, ShareConfig, ShareWithTarget, updateShare,
} from "../api/agent";
import { useRevalidate } from "../state/revalidate";
import { useConfirm, useToast } from "../state/ui";

export function SharedView() {
  const [shares, setShares] = useState<ShareWithTarget[]>([]);
  const [config, setConfig] = useState<ShareConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [openId, setOpenId] = useState<string | null>(null);
  const [stats, setStats] = useState<ShareAccessStats | null>(null);
  const [statsLoading, setStatsLoading] = useState(false);
  const toast = useToast();
  const confirm = useConfirm();

  async function refresh(opts?: { silent?: boolean }) {
    const silent = opts?.silent ?? false;
    if (!silent) setLoading(true);
    try {
      const [cfg, list] = await Promise.all([getShareConfig(), listMyShares()]);
      setConfig(cfg.config);
      setShares(list.shares);
    } catch (err) {
      if (!silent) toast(err instanceof Error ? err.message : String(err), "error");
    } finally {
      if (!silent) setLoading(false);
    }
  }
  useEffect(() => { refresh(); }, []);
  useRevalidate(() => refresh({ silent: true }), {
    eventsUrl: eventsUrl(),
    sseEvents: ["share.created", "share.deleted", "share.updated"],
    pollMs: 25000,
  });

  async function toggleStats(id: string) {
    if (openId === id) { setOpenId(null); setStats(null); return; }
    setOpenId(id);
    setStats(null);
    setStatsLoading(true);
    try { setStats(await getShareAccess(id, 50)); }
    catch (err) { toast(err instanceof Error ? err.message : String(err), "error"); }
    finally { setStatsLoading(false); }
  }

  function copy(slug: string) {
    const link = shareLink(config, slug);
    navigator.clipboard.writeText(link).then(() => toast("Đã sao chép link", "success")).catch(() => window.prompt("Sao chép:", link));
  }

  async function revoke(s: ShareWithTarget) {
    try { await updateShare(s.id, { revoked: !s.revoked }); await refresh({ silent: true }); toast(s.revoked ? "Đã bật lại link" : "Đã thu hồi link", "success"); }
    catch (err) { toast(err instanceof Error ? err.message : String(err), "error"); }
  }

  async function remove(s: ShareWithTarget) {
    const ok = await confirm({ title: "Xóa link", message: `Xóa link chia sẻ "${s.target_name || s.slug}"?`, tone: "error" });
    if (!ok) return;
    try { await deleteShare(s.id); await refresh({ silent: true }); toast("Đã xóa link", "success"); }
    catch (err) { toast(err instanceof Error ? err.message : String(err), "error"); }
  }

  const isEmpty = shares.length === 0;

  return (
    <section className="shared-view">
      <header className="shared-view__header">
        <div>
          <h2>Đã chia sẻ</h2>
          <p>Tất cả link bạn đã tạo, kèm thống kê lượt xem &amp; tải.</p>
        </div>
        <button className="button button--ghost" onClick={() => refresh()} disabled={loading}><RefreshCw size={15} /> Làm mới</button>
      </header>

      {loading && <div className="muted-box">Đang tải...</div>}
      {!loading && isEmpty && (
        <div className="muted-box shared-empty">
          <Link2 size={28} />
          <p>Chưa có link chia sẻ nào. Mở 1 file/thư mục → chuột phải → <strong>Chia sẻ</strong> để tạo link.</p>
        </div>
      )}

      {!loading && !isEmpty && (
        <ul className="shared-list">
          {shares.map((s) => (
            <li key={s.id} className={`shared-item ${s.revoked ? "shared-item--revoked" : ""}`}>
              <div className="shared-item__main">
                <div className="shared-item__info">
                  <strong>{s.target_name || "(không tên)"}</strong>
                  <span className="shared-item__meta">
                    {s.target_kind === "folder" ? "Thư mục" : "File"}
                    {s.has_password && " · 🔒 mật khẩu"}
                    {s.expires_at && s.expires_at > 0 ? ` · hết hạn ${new Date(s.expires_at * 1000).toLocaleDateString("vi-VN")}` : ""}
                    {s.max_downloads > 0 ? ` · giới hạn ${s.max_downloads}` : ""}
                    {s.revoked && " · ĐÃ THU HỒI"}
                  </span>
                </div>
                <div className="shared-item__stats">
                  <span title="Lượt xem"><Eye size={14} /> {s.access_count ?? 0}</span>
                </div>
                <div className="shared-item__actions">
                  <button className="icon-button" title="Sao chép link" onClick={() => copy(s.slug)}><Copy size={15} /></button>
                  <a className="icon-button" title="Mở link" href={shareLink(config, s.slug)} target="_blank" rel="noreferrer"><ExternalLink size={15} /></a>
                  <button className="icon-button" title="Thống kê truy cập" onClick={() => toggleStats(s.id)}><BarChart2 size={15} /></button>
                  <button className="icon-button" title={s.revoked ? "Bật lại" : "Thu hồi"} onClick={() => revoke(s)}>{s.revoked ? "↺" : "⛔"}</button>
                  <button className="icon-button" title="Xóa" onClick={() => remove(s)}><Trash2 size={15} /></button>
                </div>
              </div>
              {openId === s.id && (
                <div className="shared-track">
                  {statsLoading && <div className="muted-box">Đang tải thống kê...</div>}
                  {!statsLoading && stats && (
                    <>
                      <div className="shared-track__summary">
                        <span><Eye size={14} /> {stats.views} lượt xem</span>
                        <span>⬇ {stats.downloads} lượt tải</span>
                      </div>
                      {stats.recent.length === 0 ? (
                        <div className="muted-box">Chưa có lượt truy cập nào.</div>
                      ) : (
                        <table className="shared-track__table">
                          <thead><tr><th>Thời gian</th><th>Hành động</th><th>IP</th><th>Thiết bị</th></tr></thead>
                          <tbody>
                            {stats.recent.map((e, i) => (
                              <tr key={i}>
                                <td>{new Date(e.created_at * 1000).toLocaleString("vi-VN")}</td>
                                <td>{e.action === "download" ? "Tải" : "Xem"}</td>
                                <td>{maskIP(e.ip)}</td>
                                <td>{shortUA(e.user_agent)}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      )}
                      <p className="shared-track__note">Dữ liệu truy cập (IP/thiết bị) lưu trên máy chủ của bạn, chỉ bạn xem được.</p>
                    </>
                  )}
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function maskIP(ip?: string) {
  if (!ip) return "—";
  // IPv4: ẩn octet cuối; IPv6: rút gọn
  if (ip.includes(".")) { const p = ip.split("."); if (p.length === 4) return `${p[0]}.${p[1]}.${p[2]}.x`; }
  if (ip.includes(":")) return ip.split(":").slice(0, 3).join(":") + "::";
  return ip;
}

function shortUA(ua?: string) {
  if (!ua) return "—";
  if (/Android/i.test(ua)) return "Android";
  if (/iPhone|iPad|iOS/i.test(ua)) return "iOS";
  if (/Windows/i.test(ua)) return "Windows";
  if (/Mac OS/i.test(ua)) return "macOS";
  if (/Linux/i.test(ua)) return "Linux";
  return ua.slice(0, 24);
}
