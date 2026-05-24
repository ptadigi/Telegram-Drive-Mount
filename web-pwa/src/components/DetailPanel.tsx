import { CalendarClock, FileText, Folder, HardDrive, Link2, X } from "lucide-react";
import { AGENT_BASE_URL, DriveFile, DriveFolder, thumbnailUrl } from "../api/agent";

type Props = {
  selection: { kind: "file"; data: DriveFile } | { kind: "folder"; data: DriveFolder } | null;
  onClose: () => void;
  onShare: () => void;
};

export function DetailPanel({ selection, onClose, onShare }: Props) {
  if (!selection) return null;
  if (selection.kind === "file") {
    const file = selection.data;
    return (
      <aside className="detail-panel" aria-label="Chi tiết file">
        <header className="detail-panel__header">
          <div>
            <strong>{file.name}</strong>
            <span>{file.kind?.toUpperCase()} · {formatBytes(file.size)}</span>
          </div>
          <button className="icon-button" onClick={onClose} aria-label="Đóng"><X size={16} /></button>
        </header>
        <div className="detail-panel__preview">
          {file.preview_status === "ready" && file.kind === "image"
            ? <img src={`${AGENT_BASE_URL}/v1/files/download?id=${encodeURIComponent(file.id)}`} alt="" />
            : <FileText size={48} />}
        </div>
        {file.kind === "audio" && (
          <div className="detail-panel__media">
            <audio controls src={`${AGENT_BASE_URL}/v1/files/download?id=${encodeURIComponent(file.id)}`}></audio>
          </div>
        )}
        <dl className="detail-panel__list">
          <Row icon={<HardDrive size={14} />} label="Loại" value={file.mime_type || file.kind} />
          <Row icon={<CalendarClock size={14} />} label="Cập nhật" value={formatTimestamp(file.updated_at)} />
          <Row icon={<CalendarClock size={14} />} label="Tạo lúc" value={formatTimestamp(file.created_at)} />
          <Row icon={<FileText size={14} />} label="Trạng thái" value={syncLabel(file.sync_state)} />
        </dl>
        <div className="detail-panel__actions">
          <button className="button button--secondary" onClick={onShare}><Link2 size={14} /> Tạo link chia sẻ</button>
        </div>
      </aside>
    );
  }
  const folder = selection.data;
  return (
    <aside className="detail-panel" aria-label="Chi tiết thư mục">
      <header className="detail-panel__header">
        <div>
          <strong>{folder.name}</strong>
          <span>Thư mục</span>
        </div>
        <button className="icon-button" onClick={onClose} aria-label="Đóng"><X size={16} /></button>
      </header>
      <div className="detail-panel__preview"><Folder size={56} /></div>
      <dl className="detail-panel__list">
        <Row icon={<CalendarClock size={14} />} label="Cập nhật" value={formatTimestamp(folder.updated_at)} />
        <Row icon={<CalendarClock size={14} />} label="Tạo lúc" value={formatTimestamp(folder.created_at)} />
      </dl>
      <div className="detail-panel__actions">
        <button className="button button--secondary" onClick={onShare}><Link2 size={14} /> Tạo link chia sẻ</button>
      </div>
    </aside>
  );
}

function Row({ icon, label, value }: { icon: React.ReactNode; label: string; value: string; }) {
  return (
    <div className="detail-panel__row">
      <span className="detail-panel__label">{icon} {label}</span>
      <span className="detail-panel__value">{value}</span>
    </div>
  );
}

function syncLabel(state: string) {
  const labels: Record<string, string> = { pending_telegram_upload: "Chờ đồng bộ", telegram_uploading: "Đang đồng bộ", telegram_synced: "Đã đồng bộ", telegram_upload_failed: "Lỗi đồng bộ", metadata_only: "Metadata" };
  return labels[state] || state;
}

function formatBytes(bytes: number) {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) { value /= 1024; unit += 1; }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unit]}`;
}

function formatTimestamp(ts: number | undefined) {
  if (!ts) return "-";
  const date = new Date(ts * 1000);
  return date.toLocaleString("vi-VN");
}
