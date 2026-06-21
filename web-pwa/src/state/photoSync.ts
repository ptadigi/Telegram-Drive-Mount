import { photosMissing, uploadPhoto } from "../api/agent";

// photoSync: one-way phone -> Telegram Drive "Camera" backup.
//
// Android/desktop Chromium: showDirectoryPicker() grants access to a real
// folder (DCIM/Camera). The handle is persisted in IndexedDB so the user only
// picks once. iOS/Safari has no directory picker -> caller falls back to an
// <input type=file multiple> and feeds the File[] into uploadFiles().
//
// Incremental: each file is SHA-256 hashed; we ask the server which hashes are
// missing and upload only those. A local set of already-synced hashes (also in
// IndexedDB) short-circuits before we even hit the network.

const DB_NAME = "td-photo-sync";
const STORE = "kv";
const DIR_KEY = "photoDir";
const SYNCED_KEY = "syncedHashes";

export function photoSyncSupported(): boolean {
  return typeof window !== "undefined" && "showDirectoryPicker" in window;
}

type AnyFsHandle = { name: string };

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, 1);
    req.onupgradeneeded = () => req.result.createObjectStore(STORE);
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

async function idbGet<T>(key: string): Promise<T | undefined> {
  const db = await openDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE, "readonly");
    const r = tx.objectStore(STORE).get(key);
    r.onsuccess = () => resolve(r.result as T | undefined);
    r.onerror = () => reject(r.error);
  });
}

async function idbSet(key: string, val: unknown): Promise<void> {
  const db = await openDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE, "readwrite");
    tx.objectStore(STORE).put(val, key);
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
}

async function loadSyncedHashes(): Promise<Set<string>> {
  const arr = (await idbGet<string[]>(SYNCED_KEY)) || [];
  return new Set(arr);
}

async function persistSyncedHashes(set: Set<string>): Promise<void> {
  // Cap to avoid unbounded growth; keep most recent.
  const arr = [...set].slice(-50000);
  await idbSet(SYNCED_KEY, arr);
}

export async function pickPhotoDirectory(): Promise<string | null> {
  const picker = (window as unknown as { showDirectoryPicker: (opts?: unknown) => Promise<AnyFsHandle> }).showDirectoryPicker;
  const handle = await picker({ id: "td-photos", mode: "read" });
  await idbSet(DIR_KEY, handle);
  return handle.name;
}

export async function savedDirectoryName(): Promise<string | null> {
  const h = await idbGet<AnyFsHandle>(DIR_KEY);
  return h ? h.name : null;
}

export async function clearPhotoDirectory(): Promise<void> {
  await idbSet(DIR_KEY, undefined);
}

async function sha256Hex(buf: ArrayBuffer): Promise<string> {
  const digest = await crypto.subtle.digest("SHA-256", buf);
  const bytes = new Uint8Array(digest);
  let hex = "";
  for (let i = 0; i < bytes.length; i++) hex += bytes[i].toString(16).padStart(2, "0");
  return hex;
}

const IMAGE_RE = /\.(jpe?g|png|gif|webp|heic|heif|bmp|tiff?)$/i;
const VIDEO_RE = /\.(mp4|mov|m4v|3gp|avi|mkv|webm)$/i;

export type SyncProgress = {
  phase: "scanning" | "hashing" | "checking" | "uploading" | "done" | "error";
  scanned: number;
  toUpload: number;
  uploaded: number;
  skipped: number;
  failed: number;
  current?: string;
  error?: string;
};

export type SyncOptions = { includeVideos?: boolean; onProgress?: (p: SyncProgress) => void };

// Recursively collect File objects from a saved directory handle.
async function collectFromHandle(handle: unknown, includeVideos: boolean, out: File[]): Promise<void> {
  const dir = handle as { values: () => AsyncIterable<{ kind: string; name: string; getFile?: () => Promise<File> }> };
  for await (const entry of dir.values()) {
    if (entry.kind === "file" && entry.getFile) {
      const matches = IMAGE_RE.test(entry.name) || (includeVideos && VIDEO_RE.test(entry.name));
      if (matches) out.push(await entry.getFile());
    } else if (entry.kind === "directory") {
      await collectFromHandle(entry, includeVideos, out);
    }
  }
}

async function verifyPermission(handle: unknown): Promise<boolean> {
  const h = handle as { queryPermission?: (o: unknown) => Promise<string>; requestPermission?: (o: unknown) => Promise<string> };
  const opts = { mode: "read" };
  if (h.queryPermission && (await h.queryPermission(opts)) === "granted") return true;
  if (h.requestPermission && (await h.requestPermission(opts)) === "granted") return true;
  return false;
}

// Sync from the previously picked directory (Android/desktop).
export async function syncFromSavedDirectory(opts: SyncOptions = {}): Promise<SyncProgress> {
  const handle = await idbGet<unknown>(DIR_KEY);
  if (!handle) return finalError(opts, "Chưa chọn thư mục ảnh");
  if (!(await verifyPermission(handle))) return finalError(opts, "Chưa được cấp quyền truy cập thư mục");
  const files: File[] = [];
  emit(opts, { phase: "scanning", scanned: 0, toUpload: 0, uploaded: 0, skipped: 0, failed: 0 });
  await collectFromHandle(handle, !!opts.includeVideos, files);
  return uploadFiles(files, opts);
}

// Upload an explicit File[] (iOS fallback via <input type=file>).
export async function uploadFiles(files: File[], opts: SyncOptions = {}): Promise<SyncProgress> {
  const onProgress = opts.onProgress;
  const synced = await loadSyncedHashes();
  const prog: SyncProgress = { phase: "hashing", scanned: files.length, toUpload: 0, uploaded: 0, skipped: 0, failed: 0 };
  emit(opts, prog);

  // Hash everything; skip files already known-synced locally.
  const byHash = new Map<string, File>();
  for (const f of files) {
    try {
      const hex = await sha256Hex(await f.arrayBuffer());
      if (synced.has(hex)) { prog.skipped++; continue; }
      if (!byHash.has(hex)) byHash.set(hex, f);
    } catch {
      prog.failed++;
    }
    prog.current = f.name;
    onProgress?.({ ...prog });
  }

  // Ask server which are missing (dedup across devices / prior uploads).
  prog.phase = "checking";
  onProgress?.({ ...prog });
  const hashes = [...byHash.keys()];
  let missing = hashes;
  if (hashes.length > 0) {
    try {
      const res = await photosMissing(hashes);
      missing = res.missing;
    } catch {
      // If the check fails, fall back to uploading all (server still dedups).
      missing = hashes;
    }
  }
  // Hashes present on server but not in local set -> mark synced, skip upload.
  for (const h of hashes) if (!missing.includes(h)) { synced.add(h); prog.skipped++; }

  prog.toUpload = missing.length;
  prog.phase = "uploading";
  onProgress?.({ ...prog });

  for (const h of missing) {
    const file = byHash.get(h);
    if (!file) continue;
    prog.current = file.name;
    onProgress?.({ ...prog });
    try {
      await uploadPhoto(file, file.name);
      synced.add(h);
      prog.uploaded++;
    } catch {
      prog.failed++;
    }
    onProgress?.({ ...prog });
  }

  await persistSyncedHashes(synced);
  prog.phase = "done";
  prog.current = undefined;
  onProgress?.({ ...prog });
  return prog;
}

function emit(opts: SyncOptions, p: SyncProgress) { opts.onProgress?.({ ...p }); }
function finalError(opts: SyncOptions, msg: string): SyncProgress {
  const p: SyncProgress = { phase: "error", scanned: 0, toUpload: 0, uploaded: 0, skipped: 0, failed: 0, error: msg };
  opts.onProgress?.({ ...p });
  return p;
}
