export type AgentHealth = { ok: boolean; service: string; version: string; timestamp: string; };
export type AgentInfo = { name: string; version: string; started_at: string; uptime_sec: number; features: Record<string, boolean>; };
export type AgentConfig = { host: string; port: number; data_dir: string; database_path: string; telegram: { api_id_set: boolean; api_hash_set: boolean; session_path: string; session_exists: boolean; }; };
export type DatabaseStatus = { path: string; exists: boolean; };

export type DriveFile = { id: string; folder_id?: string; name: string; extension: string; kind: "image" | "video" | "audio" | "document" | "archive" | "other"; size: number; mime_type?: string; sync_state: string; thumbnail_path?: string; preview_status: string; starred?: boolean; created_at: number; updated_at: number; };
export type DriveFolder = { id: string; parent_id?: string; name: string; starred?: boolean; created_at: number; updated_at: number; };
export type DriveContents = { folder_id?: string; folders: DriveFolder[]; files: DriveFile[]; };
export type SyncResult = { uploaded: number; failed: number; message: string; };
export type Transfer = { id: string; file_id: string; kind: string; phase: string; percent: number; bytes_done: number; bytes_total: number; last_error?: string; created_at: number; updated_at: number; };
export type Share = { id: string; slug: string; target_kind: string; target_id: string; has_password: boolean; expires_at?: number; revoked: boolean; max_downloads: number; access_count: number; last_accessed_at?: number; created_at: number; updated_at: number; };
export type ShareConfig = { mode: string; domain?: string; base_url?: string; local_base_url: string; port: number; health_ok: boolean; health_message?: string; tunnel_url?: string; tunnel_active?: boolean; updated_at?: number; };
export type CacheStats = { mode: string; max_bytes: number; used_bytes: number; files: number; };
export type AuthStatus = { configured: boolean; session_exists: boolean; login_started: boolean; authorized: boolean; phone?: string; code_type?: string; };
export type SyncRoot = { id: string; local_path: string; remote_folder_id?: string; mode: string; enabled: boolean; status: string; last_scan_at: number; created_at: number; updated_at: number; };
export type UploadProgress = { phase: "uploading_agent" | "processing" | "completed" | "failed"; percent: number; fileName: string; error?: string; };

const currentHost = window.location.hostname || "127.0.0.1";
export const AGENT_BASE_URL = `http://${currentHost}:8750`;

async function getJSON<T>(path: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`${AGENT_BASE_URL}${path}`, { signal });
  if (!response.ok) throw new Error(`Agent API lỗi ${response.status}`);
  return response.json() as Promise<T>;
}

async function sendJSON<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(`${AGENT_BASE_URL}${path}`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Agent API lỗi ${response.status}` }));
    throw new Error(data.error || `Agent API lỗi ${response.status}`);
  }
  return response.json() as Promise<T>;
}

async function putJSON<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(`${AGENT_BASE_URL}${path}`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Agent API lỗi ${response.status}` }));
    throw new Error(data.error || `Agent API lỗi ${response.status}`);
  }
  return response.json() as Promise<T>;
}

export function getHealth(signal?: AbortSignal) { return getJSON<AgentHealth>("/health", signal); }
export function getInfo(signal?: AbortSignal) { return getJSON<AgentInfo>("/v1/info", signal); }
export function getConfig(signal?: AbortSignal) { return getJSON<AgentConfig>("/v1/config", signal); }
export function getDatabaseStatus(signal?: AbortSignal) { return getJSON<DatabaseStatus>("/v1/database/status", signal); }
export function getAuthStatus(signal?: AbortSignal) { return getJSON<AuthStatus>("/v1/auth/status", signal); }
export function resetTelegramLogin() { return sendJSON<AuthStatus>("/v1/auth/reset", {}); }
export function saveTelegramConfig(apiId: number, apiHash: string) { return putJSON<AuthStatus>("/v1/auth/config", { api_id: apiId, api_hash: apiHash }); }
export function startTelegramLogin(phone: string) { return sendJSON<{ next_step: string; phone: string; code_type: string; timeout_sec: number }>("/v1/auth/start", { phone }); }
export function submitTelegramCode(code: string) { return sendJSON<{ success: boolean; next_step?: string }>("/v1/auth/code", { code }); }
export function submitTelegramPassword(password: string) { return sendJSON<{ success: boolean; next_step?: string }>("/v1/auth/password", { password }); }

export function listFiles(signal?: AbortSignal) { return getJSON<{ files: DriveFile[] }>("/v1/files", signal); }
export function listDriveContents(folderId = "", signal?: AbortSignal) { return getJSON<DriveContents>(`/v1/drive/contents?folder_id=${encodeURIComponent(folderId)}`, signal); }
export function listTransfers(signal?: AbortSignal) { return getJSON<{ transfers: Transfer[] }>("/v1/transfers", signal); }
export function eventsUrl() { return `${AGENT_BASE_URL}/v1/events`; }
export function listSyncRoots(signal?: AbortSignal) { return getJSON<{ roots: SyncRoot[] }>("/v1/sync/roots", signal); }
export function createSyncRoot(localPath: string, remoteFolderId = "") { return sendJSON<{ root: SyncRoot; roots: SyncRoot[] }>("/v1/sync/roots", { local_path: localPath, remote_folder_id: remoteFolderId, mode: "upload_only" }); }
export function scanSyncRoot(id: string) { return sendJSON<{ roots: SyncRoot[] }>("/v1/sync/roots/scan", { id }); }
export function updateSyncRoot(id: string, enabled: boolean) { return putJSON<{ roots: SyncRoot[] }>("/v1/sync/roots", { id, enabled }); }
export async function deleteSyncRoot(id: string) {
  const response = await fetch(`${AGENT_BASE_URL}/v1/sync/roots?id=${encodeURIComponent(id)}`, { method: "DELETE" });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Agent API lỗi ${response.status}` }));
    throw new Error(data.error || `Agent API lỗi ${response.status}`);
  }
  return response.json() as Promise<{ roots: SyncRoot[] }>;
}
export function createFolder(name: string, parentId = "") { return sendJSON<{ folder: DriveFolder; contents: DriveContents }>("/v1/folders", { name, parent_id: parentId }); }
export function renameFolder(id: string, name: string) { return putJSON<{ folder: DriveFolder }>("/v1/folders/rename", { id, name }); }
export function trashFolder(id: string) { return sendJSON<{ ok: boolean }>("/v1/folders/trash", { id }); }
export function restoreFolder(id: string) { return sendJSON<{ ok: boolean }>("/v1/folders/restore", { id }); }
export function renameFile(id: string, name: string) { return putJSON<{ file: DriveFile }>("/v1/files/rename", { id, name }); }
export function trashFile(id: string) { return sendJSON<{ ok: boolean }>("/v1/files/trash", { id }); }
export function restoreFile(id: string) { return sendJSON<{ ok: boolean }>("/v1/files/restore", { id }); }
export function listTrash(signal?: AbortSignal) { return getJSON<DriveContents>("/v1/trash", signal); }
export function search(query: string, signal?: AbortSignal) { return getJSON<{ folders: DriveFolder[]; files: DriveFile[] }>(`/v1/search?q=${encodeURIComponent(query)}`, signal); }
export function listStarred(signal?: AbortSignal) { return getJSON<DriveContents>("/v1/starred", signal); }
export function moveFile(id: string, newParentId: string) { return putJSON<{ file: DriveFile }>("/v1/files/move", { id, new_parent_id: newParentId }); }
export function moveFolder(id: string, newParentId: string) { return putJSON<{ folder: DriveFolder }>("/v1/folders/move", { id, new_parent_id: newParentId }); }
export function starFile(id: string, starred: boolean) { return putJSON<{ ok: boolean }>("/v1/files/star", { id, starred }); }
export function starFolder(id: string, starred: boolean) { return putJSON<{ ok: boolean }>("/v1/folders/star", { id, starred }); }
export async function permanentDeleteFile(id: string) {
  const response = await fetch(`${AGENT_BASE_URL}/v1/files?id=${encodeURIComponent(id)}`, { method: "DELETE" });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Agent API lỗi ${response.status}` }));
    throw new Error(data.error || `Agent API lỗi ${response.status}`);
  }
  return response.json() as Promise<{ ok: boolean }>;
}
export async function permanentDeleteFolder(id: string) {
  const response = await fetch(`${AGENT_BASE_URL}/v1/folders?id=${encodeURIComponent(id)}`, { method: "DELETE" });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Agent API lỗi ${response.status}` }));
    throw new Error(data.error || `Agent API lỗi ${response.status}`);
  }
  return response.json() as Promise<{ ok: boolean }>;
}
export function zipFolderUrl(id: string) { return `${AGENT_BASE_URL}/v1/folders/zip?id=${encodeURIComponent(id)}`; }
export function listShares(targetKind?: string, targetId?: string, signal?: AbortSignal) {
  const params = new URLSearchParams();
  if (targetKind) params.set("target_kind", targetKind);
  if (targetId) params.set("target_id", targetId);
  const query = params.toString();
  return getJSON<{ shares: Share[] }>(query ? `/v1/shares?${query}` : "/v1/shares", signal);
}
export function createShare(targetKind: string, targetId: string, password = "", expiresIn = 0, maxDownloads = 0) {
  return sendJSON<{ share: Share }>("/v1/shares", { target_kind: targetKind, target_id: targetId, password, expires_in: expiresIn, max_downloads: maxDownloads });
}
export function updateShare(id: string, payload: { password?: string | null; expires_in?: number | null; revoked?: boolean | null; max_downloads?: number | null; }) {
  const body: Record<string, unknown> = { id };
  if (payload.password !== undefined) body.password = payload.password;
  if (payload.expires_in !== undefined) body.expires_in = payload.expires_in;
  if (payload.revoked !== undefined) body.revoked = payload.revoked;
  if (payload.max_downloads !== undefined) body.max_downloads = payload.max_downloads;
  return putJSON<{ share: Share }>("/v1/shares", body);
}
export async function deleteShare(id: string) {
  const response = await fetch(`${AGENT_BASE_URL}/v1/shares?id=${encodeURIComponent(id)}`, { method: "DELETE" });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Agent API lỗi ${response.status}` }));
    throw new Error(data.error || `Agent API lỗi ${response.status}`);
  }
  return response.json() as Promise<{ ok: boolean }>;
}
export function getShareConfig(signal?: AbortSignal) { return getJSON<{ config: ShareConfig }>("/v1/share/config", signal); }
export function updateShareConfig(payload: { mode: string; domain?: string; base_url?: string; }) { return putJSON<{ config: ShareConfig }>("/v1/share/config", payload); }
export function controlTunnel(action: "start" | "stop") { return sendJSON<{ tunnel: { active: boolean; url?: string; last_error?: string } }>("/v1/share/tunnel", { action }); }
export function getCacheStats(signal?: AbortSignal) { return getJSON<{ cache: CacheStats }>("/v1/cache", signal); }
export function setCacheConfig(mode: string, maxBytes: number) { return putJSON<{ cache: CacheStats }>("/v1/cache", { mode, max_bytes: maxBytes }); }
export function cleanupCache() { return sendJSON<{ removed: number; cache: CacheStats }>("/v1/cache/cleanup", {}); }
export function shareLink(config: ShareConfig | null, slug: string) {
  const base = (config?.base_url && config.base_url.trim()) || (config?.local_base_url && config.local_base_url.trim()) || `${AGENT_BASE_URL}`;
  return `${base.replace(/\/$/, "")}/share/${slug}`;
}
export function downloadFileUrl(id: string) { return `${AGENT_BASE_URL}/v1/files/download?id=${encodeURIComponent(id)}`; }
export function thumbnailUrl(id: string) { return `${AGENT_BASE_URL}/v1/files/thumbnail?id=${encodeURIComponent(id)}`; }
export function seedDemoFile() { return sendJSON<{ contents?: DriveContents; files: DriveFile[] }>("/v1/files/demo", {}); }

export function uploadFile(file: File, folderId = "", onProgress?: (progress: UploadProgress) => void, relativePath = "") {
  return new Promise<{ file: DriveFile }>((resolve, reject) => {
    const formData = new FormData();
    formData.append("file", file);
    formData.append("folder_id", folderId);
    formData.append("relative_path", relativePath);
    const request = new XMLHttpRequest();
    request.open("POST", `${AGENT_BASE_URL}/v1/files/upload`);
    request.upload.onprogress = (event) => {
      if (!event.lengthComputable) return;
      onProgress?.({ phase: "uploading_agent", percent: Math.round((event.loaded / event.total) * 100), fileName: file.name });
    };
    request.onload = () => {
      if (request.status >= 200 && request.status < 300) {
        onProgress?.({ phase: "processing", percent: 100, fileName: file.name });
        resolve(JSON.parse(request.responseText) as { file: DriveFile });
      } else {
        const error = safeError(request.responseText, `Agent API lỗi ${request.status}`);
        onProgress?.({ phase: "failed", percent: 100, fileName: file.name, error });
        reject(new Error(error));
      }
    };
    request.onerror = () => {
      const error = "Không kết nối được Go Agent";
      onProgress?.({ phase: "failed", percent: 100, fileName: file.name, error });
      reject(new Error(error));
    };
    request.send(formData);
  });
}

function safeError(raw: string, fallback: string) {
  try { return JSON.parse(raw).error || fallback; } catch { return fallback; }
}

export function syncFilesToTelegram() { return sendJSON<{ sync: SyncResult; contents?: DriveContents; files: DriveFile[] }>("/v1/files/sync", {}); }
