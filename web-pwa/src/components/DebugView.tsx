import { Copy, RefreshCw } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { AGENT_BASE_URL, Transfer } from "../api/agent";

type SyncLogEntry = {
  ts: string;
  level: string;
  event: string;
  file_id?: string;
  file_name?: string;
  phase?: string;
  size?: number;
  sync_state?: string;
  error?: string;
  local_path?: string;
};

type DebugPayload = {
  logs: SyncLogEntry[];
  transfers: Transfer[];
};

export function DebugView() {
  const [data, setData] = useState<DebugPayload>({ logs: [], transfers: [] });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`${AGENT_BASE_URL}/v1/debug/sync?limit=120`, { credentials: "include" });
      if (!response.ok) throw new Error(`Lỗi ${response.status}`);
      const payload = await response.json();
      setData({ logs: payload.logs || [], transfers: payload.transfers || [] });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { refresh(); }, []);

  const failedTransfers = useMemo(() => data.transfers.filter((item) => item.phase === "failed" || item.last_error), [data.transfers]);

  async function copyLogs() {
    const text = JSON.stringify({ logs: data.logs, transfers: data.transfers }, null, 2);
    await navigator.clipboard.writeText(text);
  }

  return (
    <section className="debug-view">
      <header className="drive-browser__header">
        <div>
          <h2>Chẩn đoán đồng bộ</h2>
          <p>Xem log sync, transfer failed và lỗi thật để debug.</p>
        </div>
        <div className="drive-browser__actions">
          <button className="button button--secondary" onClick={copyLogs} disabled={!data.logs.length && !data.transfers.length}><Copy size={14} /> Copy log</button>
          <button className="button button--ghost" onClick={refresh} disabled={loading}><RefreshCw size={14} /> Làm mới</button>
        </div>
      </header>

      {error && <div className="error-note">{error}</div>}
      <div className="debug-grid">
        <article className="debug-card">
          <header>
            <h3>Log gần nhất</h3>
            <span>{data.logs.length} dòng</span>
          </header>
          <div className="debug-log-list">
            {data.logs.length === 0 ? <div className="muted-box">Chưa có log sync.</div> : data.logs.map((entry, idx) => (
              <pre key={`${entry.ts}-${idx}`} className={`debug-log debug-log--${entry.level}`}>{formatLog(entry)}</pre>
            ))}
          </div>
        </article>
        <article className="debug-card">
          <header>
            <h3>Transfer lỗi</h3>
            <span>{failedTransfers.length} mục</span>
          </header>
          {failedTransfers.length === 0 ? <div className="muted-box">Không có transfer lỗi.</div> : (
            <div className="debug-transfer-list">
              {failedTransfers.map((item) => (
                <div className="debug-transfer" key={item.id}>
                  <strong>{item.kind}</strong>
                  <span>{item.phase} · {item.percent}%</span>
                  <code>{item.last_error || "-"}</code>
                </div>
              ))}
            </div>
          )}
        </article>
      </div>
    </section>
  );
}

function formatLog(entry: SyncLogEntry) {
  return [
    `[${entry.ts}] ${entry.level.toUpperCase()} ${entry.event}`,
    entry.file_name ? `file=${entry.file_name}` : null,
    entry.file_id ? `id=${entry.file_id}` : null,
    entry.phase ? `phase=${entry.phase}` : null,
    entry.sync_state ? `state=${entry.sync_state}` : null,
    typeof entry.size === "number" ? `size=${entry.size}` : null,
    entry.local_path ? `path=${entry.local_path}` : null,
    entry.error ? `error=${entry.error}` : null,
  ].filter(Boolean).join(" | ");
}
