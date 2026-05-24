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
    const stream = new EventSource(eventsUrl());
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
          <span>Cloud drive cá nhân</span>
          <h1>Chào bạn, ổ đĩa cloud đã sẵn sàng</h1>
          <p>Tải file lên, đồng bộ thư mục desktop và chia sẻ link an toàn qua domain riêng. Telegram là kho lưu trữ ẩn phía sau.</p>
        </div>
        <div className="drive-stats">
          <Stat label="Trạng thái" value={agentState === "online" ? "Đang chạy" : "Chưa kết nối"} />
          <Stat label="Database" value={database?.exists ? "Sẵn sàng" : "Chưa sẵn sàng"} />
          <Stat label="Telegram" value={auth?.session_exists ? "Đã kết nối" : "Chưa đăng nhập"} />
          <Stat label="Uptime" value={info ? `${info.uptime_sec}s` : "-"} />
        </div>
      </section>

      <section className="home-actions">
        <button className="home-action" onClick={onOpenDrive}><Plus size={18} /><span><strong>Mở Drive của tôi</strong><br /><small>Quản lý file và thư mục</small></span></button>
        <button className="home-action" onClick={onOpenStarred}><span className="home-action__icon home-action__icon--star">★</span><span><strong>Đã đánh dấu sao</strong><br /><small>Truy cập nhanh các mục quan trọng</small></span></button>
        <button className="home-action" onClick={onOpenComputers}><span className="home-action__icon">💻</span><span><strong>Đồng bộ máy tính</strong><br /><small>Thư mục local đang được watch</small></span></button>
        <button className="home-action" onClick={onOpenSettings}><span className="home-action__icon">⚙</span><span><strong>Cấu hình chia sẻ</strong><br /><small>Domain, LAN hoặc Cloudflare Tunnel</small></span></button>
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

function Stat({ label, value }: { label: string; value: string }) {
  return <div className="mini-stat"><span>{label}</span><strong>{value}</strong></div>;
}

function kindLabel(kind: DriveFile["kind"]) {
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
