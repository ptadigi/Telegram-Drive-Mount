export type AgentHealth = {
  ok: boolean;
  service: string;
  version: string;
  timestamp: string;
};

export type AgentInfo = {
  name: string;
  version: string;
  started_at: string;
  uptime_sec: number;
  features: Record<string, boolean>;
};

export type AgentConfig = {
  host: string;
  port: number;
  data_dir: string;
  database_path: string;
  telegram: {
    api_id_set: boolean;
    api_hash_set: boolean;
    session_path: string;
    session_exists: boolean;
  };
};

export type DatabaseStatus = {
  path: string;
  exists: boolean;
};

export type DriveFile = {
  id: string;
  folder_id?: string;
  name: string;
  size: number;
  mime_type?: string;
  sync_state: string;
  created_at: number;
  updated_at: number;
};

export type AuthStatus = {
  configured: boolean;
  session_exists: boolean;
  login_started: boolean;
  authorized: boolean;
  phone?: string;
  code_type?: string;
};

const currentHost = window.location.hostname || "127.0.0.1";
const AGENT_BASE_URL = `http://${currentHost}:8750`;

async function getJSON<T>(path: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`${AGENT_BASE_URL}${path}`, { signal });
  if (!response.ok) {
    throw new Error(`Agent API lỗi ${response.status}`);
  }
  return response.json() as Promise<T>;
}

export function getHealth(signal?: AbortSignal) {
  return getJSON<AgentHealth>("/health", signal);
}

export function getInfo(signal?: AbortSignal) {
  return getJSON<AgentInfo>("/v1/info", signal);
}

export function getConfig(signal?: AbortSignal) {
  return getJSON<AgentConfig>("/v1/config", signal);
}

export function getDatabaseStatus(signal?: AbortSignal) {
  return getJSON<DatabaseStatus>("/v1/database/status", signal);
}

async function sendJSON<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(`${AGENT_BASE_URL}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Agent API lỗi ${response.status}` }));
    throw new Error(data.error || `Agent API lỗi ${response.status}`);
  }
  return response.json() as Promise<T>;
}

async function putJSON<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(`${AGENT_BASE_URL}${path}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: `Agent API lỗi ${response.status}` }));
    throw new Error(data.error || `Agent API lỗi ${response.status}`);
  }
  return response.json() as Promise<T>;
}

export function getAuthStatus(signal?: AbortSignal) {
  return getJSON<AuthStatus>("/v1/auth/status", signal);
}

export function resetTelegramLogin() {
  return sendJSON<AuthStatus>("/v1/auth/reset", {});
}

export function saveTelegramConfig(apiId: number, apiHash: string) {
  return putJSON<AuthStatus>("/v1/auth/config", { api_id: apiId, api_hash: apiHash });
}

export function startTelegramLogin(phone: string) {
  return sendJSON<{ next_step: string; phone: string; code_type: string; timeout_sec: number }>("/v1/auth/start", { phone });
}

export function submitTelegramCode(code: string) {
  return sendJSON<{ success: boolean; next_step?: string }>("/v1/auth/code", { code });
}

export function submitTelegramPassword(password: string) {
  return sendJSON<{ success: boolean; next_step?: string }>("/v1/auth/password", { password });
}

export function listFiles(signal?: AbortSignal) {
  return getJSON<{ files: DriveFile[] }>("/v1/files", signal);
}

export function seedDemoFile() {
  return sendJSON<{ files: DriveFile[] }>("/v1/files/demo", {});
}
