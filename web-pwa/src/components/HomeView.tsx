import { Clock3, Download, FileText, Folder, HardDrive, KeyRound, Plus, RefreshCw, ShieldCheck } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { AgentInfo, AuthStatus, DatabaseStatus, DriveContents, DriveFile, DriveStats, Device, eventsUrl, getDriveStats, listDevices, listDriveContents, listStarred, startDevicePairing, thumbnailUrl } from "../api/agent";
import { useRevalidate } from "../state/revalidate";

const DESKTOP_RELEASE_URL = "https://github.com/ptadigi/Telegram-Drive-Mount/releases/latest";
type Props = {
  info: AgentInfo | null;
  database: DatabaseStatus | null;
  auth: AuthStatus | null;
  agentState: string;
  onOpenDrive: () => void;
  onOpenStarred: () => void;
  onOpenSettings: () => void;
  onOpenComputers: () => void;
};

export function HomeView({ info, database, auth, agentState, onOpenDrive, onOpenStarred, onOpenSettings, onOpenComputers }: Props) {
  const { t } = useTranslation();
  const [recent, setRecent] = useState<DriveContents>({ folders: [], files: [] });
  const [starred, setStarred] = useState<DriveContents>({ folders: [], files: [] });
  const [devices, setDevices] = useState<Device[]>([]);
  const [stats, setStats] = useState<DriveStats | null>(null);
  const [pairCode, setPairCode] = useState<string | null>(null);
  const [pairLoading, setPairLoading] = useState(false);
  const [pairError, setPairError] = useState<string | null>(null);

  async function refresh() {
    try {
      setRecent(await listDriveContents(""));
      setStarred(await listStarred());
      const deviceResult = await listDevices();
      setDevices(deviceResult.devices ?? []);
      setStats(await getDriveStats());
    } catch {
      // ignore
    }
  }

  useEffect(() => { refresh(); }, []);

  useRevalidate(refresh, {
    eventsUrl: eventsUrl(),
    sseEvents: ["file.created", "file.updated", "folder.updated", "file.starred", "folder.starred", "device.created"],
    pollMs: 25000,
  });

  const recentFiles = recent.files.slice(0, 8);
  const starredItems = [...starred.folders.slice(0, 4), ...starred.files.slice(0, 4)];
  const connectedDevices = useMemo(() => devices.filter((device) => !device.revoked_at).slice(0, 3), [devices]);

  async function generatePairCode() {
    setPairLoading(true);
    setPairError(null);
    try {
      const result = await startDevicePairing();
      setPairCode(result.code);
      await refresh();
    } catch (err) {
      setPairError(err instanceof Error ? err.message : String(err));
    } finally {
      setPairLoading(false);
    }
  }

  const pairCount = devices.filter((device) => !device.revoked_at).length;
  const serverOrigin = window.location.origin;

  return (
    <div className="home-view">
      <section className="dashboard-hero">
        <div className="dashboard-hero__copy">
          <span className="eyebrow">Cloud drive cá nhân</span>
          <h1>Chào bạn, ổ đĩa cloud đã sẵn sàng</h1>
          <p>Tải file lên, đồng bộ thư mục desktop và chia sẻ link an toàn qua domain riêng. Telegram là kho lưu trữ ẩn phía sau.</p>
          <div className="dashboard-hero__actions">
            <button className="button button--primary" onClick={onOpenDrive}><Plus size={18} /> Mở Drive của tôi</button>
            <button className="button button--secondary" onClick={onOpenSettings}><ShieldCheck size={18} /> Cấu hình</button>
          </div>
        </div>
        <div className="dashboard-hero__panel">
          <div className="drive-stats drive-stats--stacked">
            <Stat label="Tệp" value={stats ? stats.file_count.toLocaleString("vi-VN") : "-"} tone="neutral" />
            <Stat label="Thư mục" value={stats ? stats.folder_count.toLocaleString("vi-VN") : "-"} tone="neutral" />
            <Stat label="Tổng dung lượng" value={stats ? formatBytes(stats.total_bytes) : "-"} tone="good" />
            <Stat label="Uptime" value={info ? formatUptime(info.uptime_sec) : "-"} tone="neutral" />
          </div>
        </div>
      </section>

      <section className="home-actions home-actions--large">
        <button className="home-action" onClick={onOpenDrive}><Plus size={18} /><span><strong>Mở Drive của tôi</strong><br /><small>Quản lý file và thư mục</small></span></button>
        <button className="home-action" onClick={onOpenStarred}><span className="home-action__icon home-action__icon--star">★</span><span><strong>Đã đánh dấu sao</strong><br /><small>Truy cập nhanh các mục quan trọng</small></span></button>
        <button className="home-action" onClick={onOpenComputers}><span className="home-action__icon">💻</span><span><strong>Đồng bộ máy tính</strong><br /><small>Thư mục local đang được watch</small></span></button>
        <button className="home-action" onClick={onOpenSettings}><span className="home-action__icon">⚙</span><span><strong>Cấu hình chia sẻ</strong><br /><small>Domain, LAN hoặc Cloudflare Tunnel</small></span></button>
      </section>

      <section className="home-grid">
        <article className="home-card home-card--pairing">
          <header className="home-card__header">
            <div>
              <h2>Kết nối thiết bị mới</h2>
              <p>Cài ứng dụng desktop để mount ổ ảo và đồng bộ. Làm theo 3 bước dưới đây.</p>
            </div>
            <KeyRound size={20} />
          </header>
          <div className="pairing-panel">
            <ol className="pair-steps">
              <li>
                <strong>1. Tải &amp; cài ứng dụng</strong>
                <a className="button button--secondary" href={DESKTOP_RELEASE_URL} target="_blank" rel="noreferrer"><Download size={15} /> Tải app desktop</a>
                <span className="form-hint">Windows: tải <code>TelegramDriveSetup.exe</code> trên trang Releases.</span>
              </li>
              <li>
                <strong>2. Mở app → chọn “Nối máy chủ có sẵn”</strong>
                <span className="form-hint">Nhập địa chỉ máy chủ này: <code>{serverOrigin}</code></span>
              </li>
              <li>
                <strong>3. Tạo mã ghép rồi dán vào app</strong>
                <button className="button button--primary" onClick={generatePairCode} disabled={pairLoading}>{pairLoading ? "Đang tạo mã..." : "Tạo mã ghép"}</button>
                {pairCode && <div className="pair-code-display"><strong>{pairCode}</strong><span>Mã có hiệu lực 5 phút, dùng 1 lần.</span></div>}
                {pairError && <div className="error-note">{pairError}</div>}
              </li>
            </ol>
            <div className="pair-summary">
              <span><strong>{pairCount}</strong> thiết bị đang ghép</span>
              <button className="button button--ghost" onClick={refresh}><RefreshCw size={14} /> Làm mới</button>
            </div>
            {connectedDevices.length > 0 && (
              <ul className="device-quick-list">
                {connectedDevices.map((device) => (
                  <li key={device.id}><strong>{device.name}</strong><span>{device.platform || "Không rõ"} · {device.last_seen_at ? new Date(device.last_seen_at * 1000).toLocaleString() : "Chưa từng"}</span></li>
                ))}
              </ul>
            )}
          </div>
        </article>

        <article className="home-card">
          <header className="home-card__header">
            <div>
              <h2>Thiết bị và trạng thái</h2>
              <p>Kiểm tra nhanh agent, database và Telegram.</p>
            </div>
          </header>
          <div className="device-status-grid">
            <StatusBlock label="Agent" value={agentState === "online" ? "Sống" : "Offline"} tone={agentState === "online" ? "good" : "warn"} />
            <StatusBlock label="Database" value={database?.exists ? "Sẵn" : "Thiếu"} tone={database?.exists ? "good" : "warn"} />
            <StatusBlock label="Telegram" value={auth?.authorized ? "Đã login" : "Chưa login"} tone={auth?.authorized ? "good" : "warn"} />
            <StatusBlock label="Uptime" value={info ? `${Math.floor(info.uptime_sec / 60)}m` : "-"} tone="neutral" />
          </div>
        </article>
      </section>

      <section className="home-recent">
        <header><Clock3 size={18} /> <h2>File cập nhật gần đây</h2></header>
        {recentFiles.length === 0 ? <div className="muted-box">Chưa có file nào.</div> : (
          <div className="file-grid">
            {recentFiles.map((file) => (
              <div className="drive-card" key={file.id}>
                <div className="drive-card__thumb">{file.preview_status === "ready" && file.kind === "image" ? <img src={thumbnailUrl(file.id)} alt="" /> : <FileText size={32} />}</div>
                <div className="drive-card__name"><strong>{file.name}</strong><span>{kindLabel(file.kind)} · {formatBytes(file.size)}</span></div>
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="home-recent">
        <header>★ <h2>Mục có đánh dấu sao</h2></header>
        {starredItems.length === 0 ? <div className="muted-box">Chưa có mục nào được đánh dấu sao.</div> : (
          <div className="file-grid">
            {starred.folders.slice(0, 4).map((folder) => (
              <div className="drive-card drive-card--folder" key={folder.id}>
                <div className="drive-card__thumb"><Folder size={32} /></div>
                <div className="drive-card__name"><strong>{folder.name}</strong><span>Thư mục</span></div>
              </div>
            ))}
            {starred.files.slice(0, 4).map((file) => (
              <div className="drive-card" key={file.id}>
                <div className="drive-card__thumb">{file.preview_status === "ready" && file.kind === "image" ? <img src={thumbnailUrl(file.id)} alt="" /> : <FileText size={32} />}</div>
                <div className="drive-card__name"><strong>{file.name}</strong><span>{kindLabel(file.kind)} · {formatBytes(file.size)}</span></div>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}

function Stat({ label, value, tone }: { label: string; value: string; tone: "good" | "warn" | "neutral"; }) {
  return <div className={`mini-stat mini-stat--${tone}`}><span>{label}</span><strong>{value}</strong></div>;
}

function StatusBlock({ label, value, tone }: { label: string; value: string; tone: "good" | "warn" | "neutral"; }) {
  return <div className={`status-block status-block--${tone}`}><span>{label}</span><strong>{value}</strong></div>;
}

function kindLabel(kind: DriveFile["kind"]) {
  const labels: Record<string, string> = { image: "Hình ảnh", video: "Video", audio: "Âm thanh", document: "Tài liệu", archive: "Nén", other: "File" };
  return labels[kind] || "File";
}

function formatUptime(sec: number) {
  if (!sec || sec < 0) return "-";
  const d = Math.floor(sec / 86400);
  const h = Math.floor((sec % 86400) / 3600);
  const m = Math.floor((sec % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m`;
  return `${sec}s`;
}

function formatBytes(bytes: number) {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) { value /= 1024; unit += 1; }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unit]}`;
}
