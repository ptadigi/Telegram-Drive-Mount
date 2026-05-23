export type AgentHealth = { ok: boolean; service: string; version: string; timestamp: string; };
export type AgentInfo = { name: string; version: string; started_at: string; uptime_sec: number; features: Record<string, boolean>; };
export type AgentConfig = { host: string; port: number; data_dir: string; database_path: string; telegram: { api_id_set: boolean; api_hash_set: boolean; session_path: string; session_exists: boolean; }; };
export type DatabaseStatus = { path: string; exists: boolean; };

export type DriveFile = { id: string; folder_id?: string; name: string; extension: string; kind: "image" | "video" | "audio" | "document" | "archive" | "other"; size: number; mime_type?: string; sync_state: string; thumbnail_path?: string; preview_status: string; created_at: number; updated_at: number; };
export type DriveFolder = { id: string; parent_id?: string; name: string; created_at: number; updated_at: number; };
export type DriveContents = { folder_id?: string; folders: DriveFolder[]; files: DriveFile[]; };
export type SyncResult = { uploaded: number; failed: number; message: string; };
export type Transfer = { id: string; file_id: string; kind: string; phase: string; percent: number; bytes_done: number; bytes_total: number; last_error?: string; created_at: number; updated_at: number; };
export type AuthStatus = { configured: boolean; session_exists: boolean; login_started: boolean; authorized: boolean; phone?: string; code_type?: string; };
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
export function createFolder(name: string, parentId = "") { return sendJSON<{ folder: DriveFolder; contents: DriveContents }>("/v1/folders", { name, parent_id: parentId }); }
export function downloadFileUrl(id: string) { return `${AGENT_BASE_URL}/v1/files/download?id=${encodeURIComponent(id)}`; }
export function thumbnailUrl(id: string) { return `${AGENT_BASE_URL}/v1/files/thumbnail?id=${encodeURIComponent(id)}`; }
export function seedDemoFile() { return sendJSON<{ contents?: DriveContents; files: DriveFile[] }>("/v1/files/demo", {}); }

export function uploadFile(file: File, folderId = "", onProgress?: (progress: UploadProgress) => void) {
  return new Promise<{ file: DriveFile }>((resolve, reject) => {
    const formData = new FormData();
    formData.append("file", file);
    formData.append("folder_id", folderId);
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
