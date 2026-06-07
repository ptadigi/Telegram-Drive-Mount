import { Clock3, FileText, Folder, Plus } from "lucide-react";
import { useEffect, useState } from "react";
import { AgentInfo, AuthStatus, DatabaseStatus, DriveContents, DriveFile, eventsUrl, listDriveContents, listStarred, thumbnailUrl } from "../api/agent";

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
  const [recent, setRecent] = useState<DriveContents>({ folders: [], files: [] });
  const [starred, setStarred] = useState<DriveContents>({ folders: [], files: [] });

  async function refresh() {
    try {
      setRecent(await listDriveContents(""));
      setStarred(await listStarred());
    } catch {
      // ignore
    }
  }

  useEffect(() => { refresh(); }, []);

  useEffect(() => {
    const stream = new EventSource(eventsUrl(), { withCredentials: true });
    stream.addEventListener("file.created", refresh);
    stream.addEventListener("file.updated", refresh);
    stream.addEventListener("folder.updated", refresh);
    stream.addEventListener("file.starred", refresh);
    stream.addEventListener("folder.starred", refresh);
    return () => stream.close();
  }, []);

  const recentFiles = recent.files.slice(0, 8);
  const starredItems = [...starred.folders.slice(0, 4), ...starred.files.slice(0, 4)];

  return (
    <div className="home-view">
      <section className="drive-hero-card">
        <div>
          <span>Cloud drive cÃ¡ nhÃ¢n</span>
          <h1>ChÃ o báº¡n, á»• Ä‘Ä©a cloud Ä‘Ã£ sáºµn sÃ ng</h1>
          <p>Táº£i file lÃªn, Ä‘á»“ng bá»™ thÆ° má»¥c desktop vÃ  chia sáº» link an toÃ n qua domain riÃªng. Telegram lÃ  kho lÆ°u trá»¯ áº©n phÃ­a sau.</p>
        </div>
        <div className="drive-stats">
          <Stat label="Tráº¡ng thÃ¡i" value={agentState === "online" ? "Äang cháº¡y" : "ChÆ°a káº¿t ná»‘i"} />
          <Stat label="Database" value={database?.exists ? "Sáºµn sÃ ng" : "ChÆ°a sáºµn sÃ ng"} />
          <Stat label="Telegram" value={auth?.session_exists ? "ÄÃ£ káº¿t ná»‘i" : "ChÆ°a Ä‘Äƒng nháº­p"} />
          <Stat label="Uptime" value={info ? `${info.uptime_sec}s` : "-"} />
        </div>
      </section>

      <section className="home-actions">
        <button className="home-action" onClick={onOpenDrive}><Plus size={18} /><span><strong>Má»Ÿ Drive cá»§a tÃ´i</strong><br /><small>Quáº£n lÃ½ file vÃ  thÆ° má»¥c</small></span></button>
        <button className="home-action" onClick={onOpenStarred}><span className="home-action__icon home-action__icon--star">â˜…</span><span><strong>ÄÃ£ Ä‘Ã¡nh dáº¥u sao</strong><br /><small>Truy cáº­p nhanh cÃ¡c má»¥c quan trá»ng</small></span></button>
        <button className="home-action" onClick={onOpenComputers}><span className="home-action__icon">ðŸ’»</span><span><strong>Äá»“ng bá»™ mÃ¡y tÃ­nh</strong><br /><small>ThÆ° má»¥c local Ä‘ang Ä‘Æ°á»£c watch</small></span></button>
        <button className="home-action" onClick={onOpenSettings}><span className="home-action__icon">âš™</span><span><strong>Cáº¥u hÃ¬nh chia sáº»</strong><br /><small>Domain, LAN hoáº·c Cloudflare Tunnel</small></span></button>
      </section>

      <section className="home-recent">
        <header><Clock3 size={18} /> <h2>File cáº­p nháº­t gáº§n Ä‘Ã¢y</h2></header>
        {recentFiles.length === 0 ? <div className="muted-box">ChÆ°a cÃ³ file nÃ o.</div> : (
          <div className="file-grid">
            {recentFiles.map((file) => (
              <div className="drive-card" key={file.id}>
                <div className="drive-card__thumb">{file.preview_status === "ready" && file.kind === "image" ? <img src={thumbnailUrl(file.id)} alt="" /> : <FileText size={32} />}</div>
                <div className="drive-card__name"><strong>{file.name}</strong><span>{kindLabel(file.kind)} Â· {formatBytes(file.size)}</span></div>
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="home-recent">
        <header>â˜… <h2>Má»¥c cÃ³ Ä‘Ã¡nh dáº¥u sao</h2></header>
        {starredItems.length === 0 ? <div className="muted-box">ChÆ°a cÃ³ má»¥c nÃ o Ä‘Æ°á»£c Ä‘Ã¡nh dáº¥u sao.</div> : (
          <div className="file-grid">
            {starred.folders.slice(0, 4).map((folder) => (
              <div className="drive-card drive-card--folder" key={folder.id}>
                <div className="drive-card__thumb"><Folder size={32} /></div>
                <div className="drive-card__name"><strong>{folder.name}</strong><span>ThÆ° má»¥c</span></div>
              </div>
            ))}
            {starred.files.slice(0, 4).map((file) => (
              <div className="drive-card" key={file.id}>
                <div className="drive-card__thumb">{file.preview_status === "ready" && file.kind === "image" ? <img src={thumbnailUrl(file.id)} alt="" /> : <FileText size={32} />}</div>
                <div className="drive-card__name"><strong>{file.name}</strong><span>{kindLabel(file.kind)} Â· {formatBytes(file.size)}</span></div>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return <div className="mini-stat"><span>{label}</span><strong>{value}</strong></div>;
}

function kindLabel(kind: DriveFile["kind"]) {
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
