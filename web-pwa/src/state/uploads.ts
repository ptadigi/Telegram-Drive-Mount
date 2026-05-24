import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { eventsUrl, listTransfers, Transfer, uploadFile, UploadProgress } from "../api/agent";

export type QueueItem = {
  id: string;
  fileName: string;
  size: number;
  folderId: string;
  relativePath: string;
  phase: UploadProgress["phase"] | "synced";
  percent: number;
  error?: string;
  fileId?: string;
};

export type UploadQueue = {
  items: QueueItem[];
  transfers: Transfer[];
  enqueue: (files: File[], options?: { folderId?: string; preserveRelativePath?: boolean }) => Promise<void>;
  clearCompleted: () => void;
  totalActive: number;
};

let counter = 0;

export function useUploadQueue(): UploadQueue {
  const [items, setItems] = useState<QueueItem[]>([]);
  const [transfers, setTransfers] = useState<Transfer[]>([]);
  const itemsRef = useRef<QueueItem[]>([]);
  itemsRef.current = items;

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
    const stream = new EventSource(eventsUrl());
    stream.addEventListener("transfer.updated", refreshTransfers);
    stream.addEventListener("file.created", refreshTransfers);
    return () => stream.close();
  }, [refreshTransfers]);

  useEffect(() => {
    setItems((current) => current.map((item) => {
      if (!item.fileId) return item;
      const transfer = transfers.find((t) => t.file_id === item.fileId);
      if (!transfer) return item;
      if (transfer.phase === "completed") return { ...item, phase: "synced", percent: 100 };
      if (transfer.phase === "failed") return { ...item, phase: "failed", percent: 100, error: transfer.last_error };
      return { ...item, phase: "processing", percent: Math.max(item.percent, transfer.percent) };
    }));
  }, [transfers]);

  const enqueue = useCallback(async (files: File[], options?: { folderId?: string; preserveRelativePath?: boolean }) => {
    const folderId = options?.folderId || "";
    const preserveRelativePath = options?.preserveRelativePath || false;
    const newItems: QueueItem[] = files.map((file) => {
      counter += 1;
      return {
        id: `${Date.now()}-${counter}`,
        fileName: file.name,
        size: file.size,
        folderId,
        relativePath: preserveRelativePath ? getRelativePath(file) : "",
        phase: "uploading_agent",
        percent: 0,
      };
    });
    setItems((current) => [...newItems, ...current]);
    for (let i = 0; i < files.length; i += 1) {
      const file = files[i];
      const queueId = newItems[i].id;
      try {
        const result = await uploadFile(file, folderId, (progress) => {
          setItems((current) => current.map((item) => item.id === queueId ? { ...item, phase: progress.phase, percent: progress.percent, error: progress.error } : item));
        }, newItems[i].relativePath);
        setItems((current) => current.map((item) => item.id === queueId ? { ...item, fileId: result.file.id, phase: "processing", percent: 100 } : item));
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        setItems((current) => current.map((item) => item.id === queueId ? { ...item, phase: "failed", percent: 100, error: message } : item));
      }
    }
    refreshTransfers();
  }, [refreshTransfers]);

  const clearCompleted = useCallback(() => {
    setItems((current) => current.filter((item) => item.phase !== "synced"));
  }, []);

  const totalActive = useMemo(() => items.filter((item) => item.phase !== "synced" && item.phase !== "failed").length, [items]);

  return { items, transfers, enqueue, clearCompleted, totalActive };
}

export function getRelativePath(file: File) {
  return (file as File & { webkitRelativePath?: string }).webkitRelativePath || file.name;
}
