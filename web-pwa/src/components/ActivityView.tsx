import { useEffect, useState } from "react";
import { AGENT_BASE_URL } from "../api/agent";

type AuditEntry = {
  id: number;
  ts: number;
  actor: string;
  action: string;
  target_kind?: string;
  target_id?: string;
  detail?: string;
};

export function ActivityView() {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`${AGENT_BASE_URL}/v1/audit?limit=200`);
      if (!response.ok) throw new Error(`Lỗi ${response.status}`);
      const body = await response.json();
      setEntries(body.entries || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { refresh(); }, []);

  return (
    <section className="drive-browser">
      <div className="drive-browser__header">
        <div>
          <h2>Hoạt động hệ thống</h2>
          <p>Lịch sử thao tác cấu hình và đồng bộ trong Agent.</p>
        </div>
        <div className="drive-browser__actions">
          <button className="button button--ghost" onClick={refresh} disabled={loading}>Làm mới</button>
        </div>
      </div>
      {error && <div className="error-note">{error}</div>}
      {loading && <div className="muted-box">Đang tải...</div>}
      {!loading && entries.length === 0 && <div className="muted-box">Chưa có hoạt động nào.</div>}
      {!loading && entries.length > 0 && (
        <div className="activity-list">
          {entries.map((entry) => (
            <div className="activity-row" key={entry.id}>
              <div>
                <strong>{entry.action}</strong>
                <span>{new Date(entry.ts * 1000).toLocaleString("vi-VN")} · {entry.actor}</span>
              </div>
              <div className="activity-row__meta">
                {entry.target_kind && <span>{entry.target_kind}</span>}
                {entry.target_id && <span>{entry.target_id}</span>}
                {entry.detail && <code>{entry.detail}</code>}
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
