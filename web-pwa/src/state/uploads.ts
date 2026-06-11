import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { eventsUrl, listTransfers, Transfer, uploadFile, UploadProgress } from "../api/agent";

export type QueuePhase = UploadProgress["phase"] | "queued" | "synced";

export type QueueItem = {
  id: string;
  fileName: string;
  size: number;
  folderId: string;
  relativePath: string;
  phase: QueuePhase;
  percent: number;
  bytesSent: number;
  error?: string;
  fileId?: string;
};

export type QueueStats = {
  total: number;
  done: number;
  failed: number;
  active: number;
  queued: number;
  totalBytes: number;
  sentBytes: number;
  percent: number;
  bytesPerSec: number;
  etaSec: number;
};

export type UploadQueue = {
  items: QueueItem[];
  activeItems: QueueItem[];
  stats: QueueStats;
  transfers: Transfer[];
  enqueue: (files: File[], options?: { folderId?: string; preserveRelativePath?: boolean }) => Promise<void>;
  clearCompleted: () => void;
  retryFailed: () => void;
  totalActive: number;
};

const CONCURRENCY = 6;
const UI_FLUSH_MS = 250;
let counter = 0;

export function useUploadQueue(): UploadQueue {
  // Source of truth lives in refs so per-chunk progress never triggers a React
  // render. We flush a snapshot to state on a throttled interval instead — this
  // keeps the UI smooth even with thousands of files.
  const itemsRef = useRef<Map<string, QueueItem>>(new Map());
  const orderRef = useRef<string[]>([]);
  const dirtyRef = useRef(false);

  const [items, setItems] = useState<QueueItem[]>([]);
  const [transfers, setTransfers] = useState<Transfer[]>([]);

  // Throughput tracking for speed/ETA.
  const speedRef = useRef({ lastBytes: 0, lastTs: Date.now(), bps: 0 });

  const markDirty = useCallback(() => { dirtyRef.current = true; }, []);

  const snapshot = useCallback((): QueueItem[] => orderRef.current.map((id) => itemsRef.current.get(id)!).filter(Boolean), []);

  // Throttled flush loop: copies ref state into React state at most every
  // UI_FLUSH_MS, and only when something changed.
  useEffect(() => {
    const timer = window.setInterval(() => {
      if (!dirtyRef.current) return;
      dirtyRef.current = false;
      setItems(snapshot());
    }, UI_FLUSH_MS);
    return () => window.clearInterval(timer);
  }, [snapshot]);

  const upsert = useCallback((id: string, patch: Partial<QueueItem>) => {
    const cur = itemsRef.current.get(id);
    if (!cur) return;
    itemsRef.current.set(id, { ...cur, ...patch });
    markDirty();
  }, [markDirty]);

  const refreshTransfers = useCallback(async () => {
    try {
      const result = await listTransfers();
      setTransfers(result.transfers);
    } catch {
      // ignore polling errors
    }
  }, []);

  useEffect(() => {
    refreshTransfers();
    let stream: EventSource | null = null;
    try {
      stream = new EventSource(eventsUrl(), { withCredentials: true });
      stream.addEventListener("transfer.updated", refreshTransfers);
      stream.addEventListener("file.created", refreshTransfers);
    } catch {
      stream = null;
    }
    // Fallback poll in case SSE is dropped by a proxy.
    const poll = window.setInterval(refreshTransfers, 8000);
    return () => {
      if (stream) stream.close();
      window.clearInterval(poll);
    };
  }, [refreshTransfers]);

  // Reconcile server-side telegram-sync transfers into items via O(1) map.
  useEffect(() => {
    if (transfers.length === 0) return;
    const byFile = new Map<string, Transfer>();
    for (const t of transfers) byFile.set(t.file_id, t);
    let changed = false;
    for (const id of orderRef.current) {
      const item = itemsRef.current.get(id);
      if (!item || !item.fileId) continue;
      const t = byFile.get(item.fileId);
      if (!t) continue;
      if (t.phase === "completed" && item.phase !== "synced") {
        itemsRef.current.set(id, { ...item, phase: "synced", percent: 100 });
        changed = true;
      } else if (t.phase === "failed" && item.phase !== "failed") {
        itemsRef.current.set(id, { ...item, phase: "failed", percent: 100, error: t.last_error });
        changed = true;
      } else if (t.phase !== "completed" && t.phase !== "failed" && item.phase === "processing") {
        const pct = Math.max(item.percent, t.percent);
        if (pct !== item.percent) { itemsRef.current.set(id, { ...item, percent: pct }); changed = true; }
      }
    }
    if (changed) { dirtyRef.current = true; }
  }, [transfers]);

  // Worker pool: pull queued items and upload up to CONCURRENCY at a time.
  const runningRef = useRef(0);
  const pendingRef = useRef<string[]>([]);

  const pump = useCallback(() => {
    while (runningRef.current < CONCURRENCY && pendingRef.current.length > 0) {
      const id = pendingRef.current.shift()!;
      const item = itemsRef.current.get(id) as QueueItemInternal | undefined;
      if (!item || !item.fileHandle) continue;
      runningRef.current += 1;
      upsert(id, { phase: "uploading_agent", percent: 0 });
      const file = item.fileHandle;
      uploadFile(file, item.folderId, (progress) => {
        upsert(id, { phase: progress.phase, percent: progress.percent, error: progress.error });
      }, item.relativePath)
        .then((result) => {
          upsert(id, { fileId: result.file.id, phase: "processing", percent: 100 });
        })
        .catch((err) => {
          const message = err instanceof Error ? err.message : String(err);
          upsert(id, { phase: "failed", percent: 100, error: message });
        })
        .finally(() => {
          runningRef.current -= 1;
          // Drop the File handle once done to free memory for huge batches.
          const done = itemsRef.current.get(id);
          if (done) { delete (done as QueueItemInternal).fileHandle; }
          refreshTransfers();
          pump();
        });
    }
  }, [upsert, refreshTransfers]);

  const enqueue = useCallback(async (files: File[], options?: { folderId?: string; preserveRelativePath?: boolean }) => {
    const folderId = options?.folderId || "";
    const preserveRelativePath = options?.preserveRelativePath || false;
    for (const file of files) {
      counter += 1;
      const id = `${Date.now()}-${counter}`;
      const item: QueueItemInternal = {
        id,
        fileName: file.name,
        size: file.size,
        folderId,
        relativePath: preserveRelativePath ? getRelativePath(file) : "",
        phase: "queued",
        percent: 0,
        bytesSent: 0,
        fileHandle: file,
      };
      itemsRef.current.set(id, item);
      orderRef.current.push(id);
      pendingRef.current.push(id);
    }
    dirtyRef.current = true;
    setItems(snapshot());
    pump();
  }, [pump, snapshot]);

  const clearCompleted = useCallback(() => {
    for (const id of [...orderRef.current]) {
      const item = itemsRef.current.get(id);
      if (item && item.phase === "synced") {
        itemsRef.current.delete(id);
      }
    }
    orderRef.current = orderRef.current.filter((id) => itemsRef.current.has(id));
    setItems(snapshot());
  }, [snapshot]);

  const retryFailed = useCallback(() => {
    for (const id of orderRef.current) {
      const item = itemsRef.current.get(id) as QueueItemInternal | undefined;
      if (item && item.phase === "failed" && item.fileHandle) {
        itemsRef.current.set(id, { ...item, phase: "queued", percent: 0, error: undefined });
        pendingRef.current.push(id);
      }
    }
    dirtyRef.current = true;
    pump();
  }, [pump]);

  const stats = useMemo<QueueStats>(() => {
    let done = 0, failed = 0, active = 0, queued = 0, totalBytes = 0, sentBytes = 0;
    for (const item of items) {
      totalBytes += item.size;
      if (item.phase === "synced") { done += 1; sentBytes += item.size; }
      else if (item.phase === "failed") { failed += 1; }
      else if (item.phase === "queued") { queued += 1; }
      else { active += 1; sentBytes += Math.round((item.percent / 100) * item.size); }
    }
    const percent = totalBytes > 0 ? Math.round((sentBytes / totalBytes) * 100) : 0;
    return { total: items.length, done, failed, active, queued, totalBytes, sentBytes, percent, bytesPerSec: speedRef.current.bps, etaSec: 0 };
  }, [items]);

  // Speed + ETA sampling.
  useEffect(() => {
    const now = Date.now();
    const dt = (now - speedRef.current.lastTs) / 1000;
    if (dt >= 0.5) {
      const delta = stats.sentBytes - speedRef.current.lastBytes;
      const bps = delta / dt;
      speedRef.current.bps = bps > 0 ? Math.round(speedRef.current.bps * 0.6 + bps * 0.4) : speedRef.current.bps;
      speedRef.current.lastBytes = stats.sentBytes;
      speedRef.current.lastTs = now;
    }
  }, [stats.sentBytes]);

  const totalActive = stats.active + stats.queued;

  // Warn before leaving while uploads are in flight.
  useEffect(() => {
    function onBeforeUnload(e: BeforeUnloadEvent) {
      if (totalActive > 0) {
        e.preventDefault();
        e.returnValue = "";
      }
    }
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => window.removeEventListener("beforeunload", onBeforeUnload);
  }, [totalActive]);

  const activeItems = useMemo(() => items.filter((i) => i.phase !== "synced").slice(0, 6), [items]);

  const etaSec = stats.bytesPerSec > 0 ? Math.round((stats.totalBytes - stats.sentBytes) / stats.bytesPerSec) : 0;

  return {
    items,
    activeItems,
    stats: { ...stats, etaSec },
    transfers,
    enqueue,
    clearCompleted,
    retryFailed,
    totalActive,
  };
}

// Internal item carries the live File handle (not exposed in the public type).
type QueueItemInternal = QueueItem & { fileHandle?: File };

export function getRelativePath(file: File) {
  return (file as File & { webkitRelativePath?: string }).webkitRelativePath || file.name;
}
