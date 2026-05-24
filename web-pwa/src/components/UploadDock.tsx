import { CheckCircle2, ChevronDown, ChevronUp, Loader2, X, XCircle } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { UploadQueue } from "../state/uploads";

type Props = { queue: UploadQueue };

export function UploadDock({ queue }: Props) {
  const { t } = useTranslation();
  const [collapsed, setCollapsed] = useState(false);
  if (queue.items.length === 0) return null;

  const totalSynced = queue.items.filter((item) => item.phase === "synced").length;
  const totalFailed = queue.items.filter((item) => item.phase === "failed").length;
  const totalActive = queue.totalActive;
  const summary = totalActive > 0
    ? t("uploadDock.activeSummary", { active: totalActive, synced: totalSynced })
    : t("uploadDock.doneSummary", { synced: totalSynced, failed: totalFailed });

  return (
    <aside className="upload-dock">
      <header className="upload-dock__header">
        <div>
          <strong>{summary}</strong>
        </div>
        <div className="upload-dock__controls">
          <button className="icon-button" onClick={() => setCollapsed((c) => !c)}>{collapsed ? <ChevronUp size={16} /> : <ChevronDown size={16} />}</button>
          {totalActive === 0 && <button className="icon-button" onClick={queue.clearCompleted}><X size={16} /></button>}
        </div>
      </header>
      {!collapsed && (
        <ul className="upload-dock__list">
          {queue.items.slice(0, 12).map((item) => (
            <li key={item.id} className={`upload-dock__item upload-dock__item--${item.phase}`}>
              <div className="upload-dock__icon">{phaseIcon(item.phase)}</div>
              <div className="upload-dock__body">
                <strong>{item.fileName}</strong>
                <span>{phaseLabel(item.phase)} {item.percent ? `${item.percent}%` : ""}</span>
                <div className="mini-progress"><span style={{ width: `${item.percent}%` }} /></div>
                {item.error && <span className="upload-dock__error">{item.error}</span>}
              </div>
            </li>
          ))}
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
    uploading_agent: "Đang tải lên Agent",
    processing: "Đang xử lý",
    synced: "Đã đồng bộ",
    failed: "Lỗi",
    completed: "Hoàn tất",
  };
  return labels[phase] || phase;
}
