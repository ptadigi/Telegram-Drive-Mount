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
export type ShareWithTarget = Share & { target_name: string };
export type ShareAccessEntry = { action: string; ip?: string; user_agent?: string; referer?: string; created_at: number };
export type ShareAccessStats = { share_id: string; views: number; downloads: number; recent: ShareAccessEntry[] };
export function listMyShares(signal?: AbortSignal) { return getJSON<{ shares: ShareWithTarget[] }>("/v1/shares", signal); }
export function getShareAccess(id: string, limit = 50, signal?: AbortSignal) { return getJSON<ShareAccessStats>(`/v1/shares/access?id=${encodeURIComponent(id)}&limit=${limit}`, signal); }
export type CacheStats = { mode: string; max_bytes: number; used_bytes: number; files: number; };
export type StorageSettings = { peer_kind: string; channel_id: number; access_hash: number; title?: string; updated_at?: number; };
export type APIAuthConfig = { mode: string; username: string; has_password: boolean; };
export type AppUser = { id: string; email: string; display_name?: string; role: string; created_at: number; };
export type AppMe = { user?: AppUser; setup?: boolean };
export type AuthStatus = { configured: boolean; session_exists: boolean; login_started: boolean; authorized: boolean; phone?: string; code_type?: string; };
export type SyncRoot = { id: string; local_path: string; remote_folder_id?: string; mode: string; enabled: boolean; status: string; last_scan_at: number; created_at: number; updated_at: number; };
export type UploadProgress = { phase: "uploading_agent" | "processing" | "completed" | "failed"; percent: number; fileName: string; error?: string; };

// AGENT_BASE_URL resolution:
// - Dev (PWA chạy trên :5173 hoặc :5174): gọi thẳng agent ở :8750 cùng host.
// - Production (PWA serve qua reverse proxy cùng domain/HTTPS): gọi same-origin
//   (đường dẫn tương đối), để OpenResty/nginx proxy /v1, /health, /share... về agent.
const currentHost = window.location.hostname || "127.0.0.1";
const devPorts = ["5173", "5174", "3000"];
const isDev = devPorts.includes(window.location.port);
export const AGENT_BASE_URL = isDev ? `http://${currentHost}:8750` : "";

async function getJSON<T>(path: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`${AGENT_BASE_URL}${path}`, { signal, credentials: "include" });
  if (!response.ok) throw new Error(`Agent API lá»—i ${response.status}`);
  return response.json() as Promise<T>;
}

async function sendJSON<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(`${AGENT_BASE_URL}${path}`, { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Agent API lá»—i ${response.status}` }));
    throw new Error(data.error || `Agent API lá»—i ${response.status}`);
  }
  return response.json() as Promise<T>;
}

async function putJSON<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(`${AGENT_BASE_URL}${path}`, { method: "PUT", credentials: "include", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Agent API lá»—i ${response.status}` }));
    throw new Error(data.error || `Agent API lá»—i ${response.status}`);
  }
  return response.json() as Promise<T>;
}

export function getHealth(signal?: AbortSignal) { return getJSON<AgentHealth>("/health", signal); }
export function getInfo(signal?: AbortSignal) { return getJSON<AgentInfo>("/v1/info", signal); }
export function getConfig(signal?: AbortSignal) { return getJSON<AgentConfig>("/v1/config", signal); }
export function getDatabaseStatus(signal?: AbortSignal) { return getJSON<DatabaseStatus>("/v1/database/status", signal); }
export type DriveStats = { file_count: number; folder_count: number; total_bytes: number; };
export function getDriveStats(signal?: AbortSignal) { return getJSON<DriveStats>("/v1/stats", signal); }

export type ApiToken = { id: string; name: string; created_at: number; last_seen_at?: number; };
export type ApiTokenCreated = { id: string; name: string; token: string; created_at: number; };
export function listApiTokens(signal?: AbortSignal) { return getJSON<{ tokens: ApiToken[] }>("/v1/api-tokens", signal); }
export function createApiToken(name: string) { return sendJSON<ApiTokenCreated>("/v1/api-tokens", { name }); }
export async function revokeApiToken(id: string) {
  const response = await fetch(`${AGENT_BASE_URL}/v1/api-tokens?id=${encodeURIComponent(id)}`, { method: "DELETE", credentials: "include" });
  if (!response.ok) throw new Error(safeError(await response.text(), "Không thu hồi được token"));
  return response.json();
}
export function getAuthStatus(signal?: AbortSignal) { return getJSON<AuthStatus>("/v1/auth/status", signal); }
export function resetTelegramLogin() { return sendJSON<AuthStatus>("/v1/auth/reset", {}); }
export function saveTelegramConfig(apiId: number, apiHash: string) { return putJSON<AuthStatus>("/v1/auth/config", { api_id: apiId, api_hash: apiHash }); }
export function startTelegramLogin(phone: string) { return sendJSON<{ next_step: string; phone: string; code_type: string; timeout_sec: number }>("/v1/auth/start", { phone }); }
export function submitTelegramCode(code: string) { return sendJSON<{ success: boolean; next_step?: string }>("/v1/auth/code", { code }); }
export function submitTelegramPassword(password: string) { return sendJSON<{ success: boolean; next_step?: string }>("/v1/auth/password", { password }); }
export type TelegramQRStatus = { state: "idle" | "pending" | "awaiting_password" | "authorized" | "expired" | "error"; token_url?: string; expires_at?: number; error?: string; };
export function startTelegramQR() { return sendJSON<TelegramQRStatus>("/v1/auth/qr/start", {}); }
export function getTelegramQRStatus(signal?: AbortSignal) { return getJSON<TelegramQRStatus>("/v1/auth/qr/status", signal); }
export function submitTelegramQRPassword(password: string) { return sendJSON<TelegramQRStatus>("/v1/auth/qr/password", { password }); }
export function cancelTelegramQR() { return sendJSON<TelegramQRStatus>("/v1/auth/qr/cancel", {}); }

export type Device = { id: string; user_id: string; name: string; platform?: string; created_at: number; last_seen_at: number; last_ip?: string; revoked_at?: number; };
export type PairingCode = { code: string; expires_at: number; };
export type PairingResult = { device: Device; token: string; };
export function startDevicePairing() { return sendJSON<PairingCode>("/v1/devices/pair/start", {}); }
export function exchangeDevicePairing(code: string, name: string, platform?: string) { return sendJSON<PairingResult>("/v1/devices/pair/exchange", { code, name, platform: platform || "" }); }
export function listDevices(signal?: AbortSignal) { return getJSON<{ devices: Device[] }>("/v1/devices", signal); }
export async function revokeDevice(id: string) {
  const response = await fetch(`${AGENT_BASE_URL}/v1/devices?id=${encodeURIComponent(id)}`, { method: "DELETE", credentials: "include" });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Lá»—i ${response.status}` }));
    throw new Error(data.error || `Lá»—i ${response.status}`);
  }
  return response.json() as Promise<{ ok: boolean }>;
}
export type MountStatus = { available: boolean; mounted: boolean; mount_point?: string; drive_letter?: string; backend: string; error?: string; started_at?: number; };
export function getMountStatus(signal?: AbortSignal) { return getJSON<MountStatus>("/v1/mount", signal); }
export function startMount(mountPoint?: string) { return sendJSON<MountStatus>("/v1/mount", { mount_point: mountPoint || "" }); }
export async function stopMount() {
  const response = await fetch(`${AGENT_BASE_URL}/v1/mount`, { method: "DELETE", credentials: "include" });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Lá»—i ${response.status}` }));
    throw new Error(data.error || `Lá»—i ${response.status}`);
  }
  return response.json() as Promise<MountStatus>;
}

export function listFiles(signal?: AbortSignal) { return getJSON<{ files: DriveFile[] }>("/v1/files", signal); }

export type DesktopState = { mode: "unset" | "local" | "remote"; server_url?: string; device_name?: string; mount_point?: string; updated_at?: number; };
export type DesktopServerInfo = { ok: boolean; url: string; service?: string; version?: string; error?: string; };
export function getDesktopState(signal?: AbortSignal) { return getJSON<{ state: DesktopState }>("/v1/desktop/state", signal); }
export function testDesktopServer(url: string) { return sendJSON<DesktopServerInfo>("/v1/desktop/test-server", { url }); }
export function pairDesktop(url: string, code: string, name: string) { return sendJSON<{ state: DesktopState }>("/v1/desktop/pair", { url, code, name }); }
export function setDesktopLocal() { return sendJSON<{ state: DesktopState }>("/v1/desktop/local", {}); }
export function resetDesktop() { return sendJSON<{ state: DesktopState }>("/v1/desktop/reset", {}); }
export function listDriveContents(folderId = "", signal?: AbortSignal) { return getJSON<DriveContents>(`/v1/drive/contents?folder_id=${encodeURIComponent(folderId)}`, signal); }
export function listTransfers(signal?: AbortSignal) { return getJSON<{ transfers: Transfer[] }>("/v1/transfers", signal); }
export function eventsUrl() { return `${AGENT_BASE_URL}/v1/events`; }
export function listSyncRoots(signal?: AbortSignal) { return getJSON<{ roots: SyncRoot[] }>("/v1/sync/roots", signal); }
export function createSyncRoot(localPath: string, remoteFolderId = "") { return sendJSON<{ root: SyncRoot; roots: SyncRoot[] }>("/v1/sync/roots", { local_path: localPath, remote_folder_id: remoteFolderId, mode: "upload_only" }); }
export function scanSyncRoot(id: string) { return sendJSON<{ roots: SyncRoot[] }>("/v1/sync/roots/scan", { id }); }
export function updateSyncRoot(id: string, enabled: boolean) { return putJSON<{ roots: SyncRoot[] }>("/v1/sync/roots", { id, enabled }); }
export async function deleteSyncRoot(id: string) {
  const response = await fetch(`${AGENT_BASE_URL}/v1/sync/roots?id=${encodeURIComponent(id)}`, { method: "DELETE", credentials: "include" });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Agent API lá»—i ${response.status}` }));
    throw new Error(data.error || `Agent API lá»—i ${response.status}`);
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
  const response = await fetch(`${AGENT_BASE_URL}/v1/files?id=${encodeURIComponent(id)}`, { method: "DELETE", credentials: "include" });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Agent API lá»—i ${response.status}` }));
    throw new Error(data.error || `Agent API lá»—i ${response.status}`);
  }
  return response.json() as Promise<{ ok: boolean }>;
}
export async function permanentDeleteFolder(id: string) {
  const response = await fetch(`${AGENT_BASE_URL}/v1/folders?id=${encodeURIComponent(id)}`, { method: "DELETE", credentials: "include" });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Agent API lá»—i ${response.status}` }));
    throw new Error(data.error || `Agent API lá»—i ${response.status}`);
  }
  return response.json() as Promise<{ ok: boolean }>;
}
export function zipFolderUrl(id: string) { return `${AGENT_BASE_URL}/v1/folders/zip?id=${encodeURIComponent(id)}`; }
export async function downloadBundle(fileIds: string[], folderIds: string[]) {
  const response = await fetch(`${AGENT_BASE_URL}/v1/bundle/zip`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ file_ids: fileIds, folder_ids: folderIds }),
  });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Lá»—i ${response.status}` }));
    throw new Error(data.error || `Lá»—i ${response.status}`);
  }
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = "bundle.zip";
  link.click();
  URL.revokeObjectURL(url);
}
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
  const response = await fetch(`${AGENT_BASE_URL}/v1/shares?id=${encodeURIComponent(id)}`, { method: "DELETE", credentials: "include" });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Agent API lá»—i ${response.status}` }));
    throw new Error(data.error || `Agent API lá»—i ${response.status}`);
  }
  return response.json() as Promise<{ ok: boolean }>;
}
export function getShareConfig(signal?: AbortSignal) { return getJSON<{ config: ShareConfig }>("/v1/share/config", signal); }
export function updateShareConfig(payload: { mode: string; domain?: string; base_url?: string; }) { return putJSON<{ config: ShareConfig }>("/v1/share/config", payload); }
export function controlTunnel(action: "start" | "stop") { return sendJSON<{ tunnel: { active: boolean; url?: string; last_error?: string } }>("/v1/share/tunnel", { action }); }
export function getCacheStats(signal?: AbortSignal) { return getJSON<{ cache: CacheStats }>("/v1/cache", signal); }
export function setCacheConfig(mode: string, maxBytes: number) { return putJSON<{ cache: CacheStats }>("/v1/cache", { mode, max_bytes: maxBytes }); }
export function cleanupCache() { return sendJSON<{ removed: number; cache: CacheStats }>("/v1/cache/cleanup", {}); }
export function getStorageSettings(signal?: AbortSignal) { return getJSON<{ storage: StorageSettings }>("/v1/storage", signal); }
export function updateStorageSettings(payload: { peer_kind: string; channel_id?: number; access_hash?: number; title?: string }) { return putJSON<{ storage: StorageSettings }>("/v1/storage", payload); }
export function createStorageChannel(title: string) { return sendJSON<{ storage: StorageSettings }>("/v1/storage/channel", { title }); }
export function getAPIAuth(signal?: AbortSignal) { return getJSON<{ auth: APIAuthConfig }>("/v1/auth/api-config", signal); }
export function updateAPIAuth(payload: { mode: string; username?: string; password?: string }) { return putJSON<{ auth: APIAuthConfig }>("/v1/auth/api-config", payload); }
export function getAppMe(signal?: AbortSignal) { return getJSON<AppMe>("/v1/users/me", signal); }
export function appRegister(email: string, password: string, displayName = "") { return sendJSON<{ user: AppUser }>("/v1/users/register", { email, password, display_name: displayName }); }
export function appLogin(email: string, password: string) { return sendJSON<{ user: AppUser }>("/v1/users/login", { email, password }); }
export function appLogout() { return sendJSON<{ ok: boolean }>("/v1/users/logout", {}); }
export function shareLink(config: ShareConfig | null, slug: string) {
  const base = (config?.base_url && config.base_url.trim()) || (config?.local_base_url && config.local_base_url.trim()) || `${AGENT_BASE_URL}`;
  return `${base.replace(/\/$/, "")}/share/${slug}`;
}
export function downloadFileUrl(id: string) { return `${AGENT_BASE_URL}/v1/files/download?id=${encodeURIComponent(id)}`; }
export function streamFileUrl(id: string) { return `${AGENT_BASE_URL}/v1/files/stream?id=${encodeURIComponent(id)}`; }
export function thumbnailUrl(id: string) { return `${AGENT_BASE_URL}/v1/files/thumbnail?id=${encodeURIComponent(id)}`; }
export function seedDemoFile() { return sendJSON<{ contents?: DriveContents; files: DriveFile[] }>("/v1/files/demo", {}); }

export function uploadFile(file: File, folderId = "", onProgress?: (progress: UploadProgress) => void, relativePath = "") {  return new Promise<{ file: DriveFile }>((resolve, reject) => {
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
        const error = safeError(request.responseText, `Agent API lá»—i ${request.status}`);
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

// TUS_CHUNK_SIZE / TUS_THRESHOLD: files larger than the threshold are uploaded
// via the resumable tus protocol in 16MB chunks so they sidestep reverse-proxy
// body-size limits and can resume after interruption.
export const TUS_CHUNK_SIZE = 16 * 1024 * 1024;
export const TUS_THRESHOLD = 32 * 1024 * 1024;

// uploadFileResumable uploads a large file via tus (chunked + resumable).
// Returns a promise resolving with the created DriveFile once the agent has
// assembled + imported the upload. onProgress reports 0-100.
export function uploadFileResumable(
  file: File,
  folderId = "",
  onProgress?: (progress: UploadProgress) => void,
  relativePath = "",
): { promise: Promise<{ file: DriveFile }>; abort: () => void } {
  let aborter: (() => void) | null = null;
  const promise = new Promise<{ file: DriveFile }>((resolve, reject) => {
    void import("tus-js-client").then(({ Upload }) => {
      const metadata: Record<string, string> = { filename: file.name, filetype: file.type || "" };
      if (folderId) metadata.folder_id = folderId;
      if (relativePath) metadata.relative_path = relativePath;
      const upload = new Upload(file, {
        endpoint: `${AGENT_BASE_URL}/v1/tus/`,
        chunkSize: TUS_CHUNK_SIZE,
        // Single-stream: parallel chunk uploads use server-side concatenation
        // which is fragile behind a reverse proxy (ERR_UPLOAD_NOT_FOUND) and
        // spikes memory on huge files. Cross-file parallelism still comes from
        // the queue's worker pool. Single-stream stays chunked + resumable.
        parallelUploads: 1,
        retryDelays: [0, 1000, 3000, 5000, 10000],
        removeFingerprintOnSuccess: true,
        metadata,
        onError: (err) => {
          const message = err instanceof Error ? err.message : String(err);
          onProgress?.({ phase: "failed", percent: 100, fileName: file.name, error: message });
          reject(new Error(message));
        },
        onProgress: (sent, total) => {
          const pct = total > 0 ? Math.round((sent / total) * 100) : 0;
          onProgress?.({ phase: "uploading_agent", percent: pct, fileName: file.name });
        },
        onSuccess: () => {
          // Agent imports the assembled file asynchronously; report processing.
          onProgress?.({ phase: "processing", percent: 100, fileName: file.name });
          // The created DriveFile id is not returned by tus; the queue resolves
          // it from the transfers/file list. Resolve with a placeholder.
          resolve({ file: { id: "", name: file.name } as DriveFile });
        },
      });
      aborter = () => upload.abort();
      // Start clean instead of auto-resuming a stored upload URL. In practice
      // stale tus fingerprints can point at server-side temp files already
      // imported/cleaned up, causing HEAD 404 before the user can retry.
      upload.start();
    }).catch(reject);
  });
  return { promise, abort: () => aborter?.() };
}

function safeError(raw: string, fallback: string) {
  try { return JSON.parse(raw).error || fallback; } catch { return fallback; }
}

export function syncFilesToTelegram() { return sendJSON<{ sync: SyncResult; contents?: DriveContents; files: DriveFile[] }>("/v1/files/sync", {}); }
