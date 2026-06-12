import { Clock3, Download, FileText, Folder, HardDrive, KeyRound, Plus, RefreshCw, ShieldCheck } from "../icons";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { AgentInfo, AuthStatus, DatabaseStatus, DriveContents, DriveFile, Device, eventsUrl, listDevices, listDriveContents, listStarred, startDevicePairing, thumbnailUrl } from "../api/agent";
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
  const [pairCode, setPairCode] = useState<string | null>(null);
  const [pairLoading, setPairLoading] = useState(false);
  const [pairError, setPairError] = useState<string | null>(null);

  async function refresh() {
    try {
      setRecent(await listDriveContents(""));
      setStarred(await listStarred());
      const deviceResult = await listDevices();
      setDevices(deviceResult.devices ?? []);
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
          <h1>Chào b?n, ? dia cloud dã s?n sàng</h1>
          <p>T?i file lên, d?ng b? thu m?c desktop và chia s? link an toàn qua domain riêng. Telegram là kho luu tr? ?n phía sau.</p>
          <div className="dashboard-hero__actions">
            <button className="button button--primary" onClick={onOpenDrive}><Plus size={18} /> M? Drive c?a tôi</button>
            <button className="button button--secondary" onClick={onOpenSettings}><ShieldCheck size={18} /> C?u hình</button>
          </div>
        </div>
        <div className="dashboard-hero__panel">
          <div className="drive-stats drive-stats--stacked">
            <Stat label="Tr?ng thái" value={agentState === "online" ? "Ðang ch?y" : "Chua k?t n?i"} tone={agentState === "online" ? "good" : "warn"} />
            <Stat label="Database" value={database?.exists ? "S?n sàng" : "Chua s?n sàng"} tone={database?.exists ? "good" : "warn"} />
            <Stat label="Telegram" value={auth?.session_exists ? "Ðã k?t n?i" : "Chua dang nh?p"} tone={auth?.session_exists ? "good" : "warn"} />
            <Stat label="Uptime" value={info ? `${info.uptime_sec}s` : "-"} tone="neutral" />
          </div>
          <div className="dashboard-hero__meta">
            <span><HardDrive size={16} /> {database?.path || "-"}</span>
            <span><Clock3 size={16} /> {info?.started_at ? new Date(info.started_at).toLocaleString() : "-"}</span>
          </div>
        </div>
      </section>

      <section className="home-actions home-actions--large">
        <button className="home-action" onClick={onOpenDrive}><Plus size={18} /><span><strong>M? Drive c?a tôi</strong><br /><small>Qu?n lý file và thu m?c</small></span></button>
        <button className="home-action" onClick={onOpenStarred}><span className="home-action__icon home-action__icon--star">?</span><span><strong>Ðã dánh d?u sao</strong><br /><small>Truy c?p nhanh các m?c quan tr?ng</small></span></button>
        <button className="home-action" onClick={onOpenComputers}><span className="home-action__icon">??</span><span><strong>Ð?ng b? máy tính</strong><br /><small>Thu m?c local dang du?c watch</small></span></button>
        <button className="home-action" onClick={onOpenSettings}><span className="home-action__icon">?</span><span><strong>C?u hình chia s?</strong><br /><small>Domain, LAN ho?c Cloudflare Tunnel</small></span></button>
      </section>

      <section className="home-grid">
        <article className="home-card home-card--pairing">
          <header className="home-card__header">
            <div>
              <h2>K?t n?i thi?t b? m?i</h2>
              <p>Cài ?ng d?ng desktop d? mount ? ?o và d?ng b?. Làm theo 3 bu?c du?i dây.</p>
            </div>
            <KeyRound size={20} />
          </header>
          <div className="pairing-panel">
            <ol className="pair-steps">
              <li>
                <strong>1. T?i &amp; cài ?ng d?ng</strong>
                <a className="button button--secondary" href={DESKTOP_RELEASE_URL} target="_blank" rel="noreferrer"><Download size={15} /> T?i app desktop</a>
                <span className="form-hint">Windows: t?i <code>TelegramDriveSetup.exe</code> trên trang Releases.</span>
              </li>
              <li>
                <strong>2. M? app ? ch?n “N?i máy ch? có s?n”</strong>
                <span className="form-hint">Nh?p d?a ch? máy ch? này: <code>{serverOrigin}</code></span>
              </li>
              <li>
                <strong>3. T?o mã ghép r?i dán vào app</strong>
                <button className="button button--primary" onClick={generatePairCode} disabled={pairLoading}>{pairLoading ? "Ðang t?o mã..." : "T?o mã ghép"}</button>
                {pairCode && <div className="pair-code-display"><strong>{pairCode}</strong><span>Mã có hi?u l?c 5 phút, dùng 1 l?n.</span></div>}
                {pairError && <div className="error-note">{pairError}</div>}
              </li>
            </ol>
            <div className="pair-summary">
              <span><strong>{pairCount}</strong> thi?t b? dang ghép</span>
              <button className="button button--ghost" onClick={refresh}><RefreshCw size={14} /> Làm m?i</button>
            </div>
            {connectedDevices.length > 0 && (
              <ul className="device-quick-list">
                {connectedDevices.map((device) => (
                  <li key={device.id}><strong>{device.name}</strong><span>{device.platform || "Không rõ"} · {device.last_seen_at ? new Date(device.last_seen_at * 1000).toLocaleString() : "Chua t?ng"}</span></li>
                ))}
              </ul>
            )}
          </div>
        </article>

        <article className="home-card">
          <header className="home-card__header">
            <div>
              <h2>Thi?t b? và tr?ng thái</h2>
              <p>Ki?m tra nhanh agent, database và Telegram.</p>
            </div>
          </header>
          <div className="device-status-grid">
            <StatusBlock label="Agent" value={agentState === "online" ? "S?ng" : "Offline"} tone={agentState === "online" ? "good" : "warn"} />
            <StatusBlock label="Database" value={database?.exists ? "S?n" : "Thi?u"} tone={database?.exists ? "good" : "warn"} />
            <StatusBlock label="Telegram" value={auth?.authorized ? "Ðã login" : "Chua login"} tone={auth?.authorized ? "good" : "warn"} />
            <StatusBlock label="Uptime" value={info ? `${Math.floor(info.uptime_sec / 60)}m` : "-"} tone="neutral" />
          </div>
        </article>
      </section>

      <section className="home-recent">
        <header><Clock3 size={18} /> <h2>File c?p nh?t g?n dây</h2></header>
        {recentFiles.length === 0 ? <div className="muted-box">Chua có file nào.</div> : (
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
        <header>? <h2>M?c có dánh d?u sao</h2></header>
        {starredItems.length === 0 ? <div className="muted-box">Chua có m?c nào du?c dánh d?u sao.</div> : (
          <div className="file-grid">
            {starred.folders.slice(0, 4).map((folder) => (
              <div className="drive-card drive-card--folder" key={folder.id}>
                <div className="drive-card__thumb"><Folder size={32} /></div>
                <div className="drive-card__name"><strong>{folder.name}</strong><span>Thu m?c</span></div>
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
  const labels: Record<string, string> = { image: "Hình ?nh", video: "Video", audio: "Âm thanh", document: "Tài li?u", archive: "Nén", other: "File" };
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
