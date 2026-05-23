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

export type AuthStatus = {
  configured: boolean;
  session_exists: boolean;
  login_started: boolean;
  phone?: string;
};

const AGENT_BASE_URL = "http://127.0.0.1:8750";

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

export function getAuthStatus(signal?: AbortSignal) {
  return getJSON<AuthStatus>("/v1/auth/status", signal);
}
