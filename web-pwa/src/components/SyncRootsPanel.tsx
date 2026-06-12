import { FolderSync, Pause, Play, Plus, RefreshCw, Trash2 } from "../icons";
import { useEffect, useState } from "react";
import { createSyncRoot, deleteSyncRoot, eventsUrl, listSyncRoots, scanSyncRoot, SyncRoot, updateSyncRoot } from "../api/agent";

export function SyncRootsPanel() {
  const [roots, setRoots] = useState<SyncRoot[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    setLoading(true);
    setError(null);
    try {
      const result = await listSyncRoots();
      setRoots(result.roots);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function addRoot() {
    const localPath = window.prompt("Nhập đường dẫn thư mục local cần đồng bộ");
    if (!localPath) return;
    setLoading(true);
    setError(null);
    try {
      const result = await createSyncRoot(localPath);
      setRoots(result.roots);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function runAction(action: () => Promise<{ roots: SyncRoot[] }>) {
    setLoading(true);
    setError(null);
    try {
      const result = await action();
      setRoots(result.roots);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { refresh(); }, []);

  useEffect(() => {
    const stream = new EventSource(eventsUrl(), { withCredentials: true });
    stream.addEventListener("syncroot.created", refresh);
    stream.addEventListener("syncroot.updated", refresh);
    return () => stream.close();
  }, []);

  return (
    <section className="sync-panel">
      <div className="sync-panel__header">
        <div>
          <h2>Đồng bộ máy tính</h2>
          <p>Chọn thư mục local để Go Agent tự quét và đưa file lên ổ đĩa cloud.</p>
        </div>
        <div className="sync-panel__actions">
          <button className="button button--secondary" onClick={refresh} disabled={loading}><RefreshCw size={16} /> Làm mới</button>
          <button className="button button--primary" onClick={addRoot} disabled={loading}><Plus size={16} /> Thêm thư mục</button>
        </div>
      </div>
      {error && <div className="error-note">{error}</div>}
      {roots.length === 0 && <div className="muted-box">Chưa có thư mục desktop nào được đồng bộ.</div>}
      {roots.length > 0 && <div className="sync-root-list">
        {roots.map((root) => (
          <div className="sync-root" key={root.id}>
            <div className="sync-root__icon"><FolderSync size={20} /></div>
            <div className="sync-root__body">
              <strong>{root.local_path}</strong>
              <span>{root.mode} · {root.enabled ? "Đang bật" : "Đã tắt"} · {root.status}</span>
            </div>
            <div className="sync-root__actions">
              <button className="button button--ghost" onClick={() => runAction(() => scanSyncRoot(root.id))} disabled={loading || !root.enabled}>Quét lại</button>
              <button className="button button--ghost" onClick={() => runAction(() => updateSyncRoot(root.id, !root.enabled))} disabled={loading}>
                {root.enabled ? <Pause size={15} /> : <Play size={15} />} {root.enabled ? "Tạm dừng" : "Bật lại"}
              </button>
              <button className="button button--ghost" onClick={() => runAction(() => deleteSyncRoot(root.id))} disabled={loading}><Trash2 size={15} /> Xóa</button>
            </div>
          </div>
        ))}
      </div>}
    </section>
  );
}
