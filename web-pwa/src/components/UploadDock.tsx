import { CheckCircle2, ChevronDown, ChevronUp, Loader2, RotateCw, X, XCircle } from "lucide-react";
import { useState } from "react";
import { UploadQueue } from "../state/uploads";

type Props = { queue: UploadQueue };

export function UploadDock({ queue }: Props) {
  const [collapsed, setCollapsed] = useState(false);
  if (queue.items.length === 0) return null;

  const { stats, activeItems } = queue;
  const inFlight = stats.active + stats.queued;
  const isDone = inFlight === 0;

  const title = isDone
    ? `Hoàn tất ${stats.done}/${stats.total}${stats.failed > 0 ? ` · ${stats.failed} lỗi` : ""}`
    : `Đang tải lên ${stats.done}/${stats.total}`;

  return (
    <aside className="upload-dock">
      <header className="upload-dock__header">
        <div className="upload-dock__title">
          <strong>{title}</strong>
          {!isDone && <span>{stats.percent}% · {formatBytes(stats.bytesPerSec)}/s{stats.etaSec > 0 ? ` · còn ${formatEta(stats.etaSec)}` : ""}</span>}
        </div>
        <div className="upload-dock__controls">
          {stats.failed > 0 && <button className="icon-button" title="Thử lại file lỗi" onClick={queue.retryFailed}><RotateCw size={16} /></button>}
          <button className="icon-button" onClick={() => setCollapsed((c) => !c)}>{collapsed ? <ChevronUp size={16} /> : <ChevronDown size={16} />}</button>
          {isDone && <button className="icon-button" title="Đóng" onClick={queue.clearCompleted}><X size={16} /></button>}
        </div>
      </header>

      <div className="upload-dock__bar"><span style={{ width: `${stats.percent}%` }} className={isDone && stats.failed === 0 ? "is-done" : undefined} /></div>

      {!collapsed && (
        <ul className="upload-dock__list">
          {activeItems.map((item) => (
            <li key={item.id} className={`upload-dock__item upload-dock__item--${item.phase}`}>
              <div className="upload-dock__icon">{phaseIcon(item.phase)}</div>
              <div className="upload-dock__body">
                <strong title={item.fileName}>{item.fileName}</strong>
                <span>{phaseLabel(item.phase)}{item.percent ? ` · ${item.percent}%` : ""}</span>
                <div className="mini-progress"><span style={{ width: `${item.percent}%` }} /></div>
                {item.error && <span className="upload-dock__error">{item.error}</span>}
              </div>
            </li>
          ))}
          {inFlight > activeItems.length && (
            <li className="upload-dock__more">+{inFlight - activeItems.length} mục đang chờ trong hàng đợi…</li>
          )}
        </ul>
      )}
    </aside>
  );
}

function phaseIcon(phase: string) {
  if (phase === "synced") return <CheckCircle2 size={18} />;
  if (phase === "failed") return <XCircle size={18} />;
  return <Loader2 size={18} className="spin" />;
}

function phaseLabel(phase: string) {
  const labels: Record<string, string> = {
    queued: "Đang chờ",
    uploading_agent: "Đang tải lên",
    processing: "Đang đồng bộ Telegram",
    synced: "Đã đồng bộ",
    failed: "Lỗi",
    completed: "Hoàn tất",
  };
  return labels[phase] || phase;
}

function formatBytes(bytes: number) {
  if (!bytes || bytes < 1) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let v = bytes;
  let u = 0;
  while (v >= 1024 && u < units.length - 1) { v /= 1024; u += 1; }
  return `${v.toFixed(v >= 10 || u === 0 ? 0 : 1)} ${units[u]}`;
}

function formatEta(sec: number) {
  if (sec < 60) return `${sec}s`;
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  if (m < 60) return `${m}m ${s}s`;
  const h = Math.floor(m / 60);
  return `${h}h ${m % 60}m`;
}
