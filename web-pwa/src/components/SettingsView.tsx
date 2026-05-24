import { CheckCircle2, Cloud, Globe, Wifi, XCircle } from "lucide-react";
import { ReactNode, useEffect, useState } from "react";
import { APIAuthConfig, CacheStats, cleanupCache, controlTunnel, eventsUrl, getAPIAuth, getCacheStats, getShareConfig, getStorageSettings, setCacheConfig, ShareConfig, StorageSettings, updateAPIAuth, updateShareConfig, updateStorageSettings } from "../api/agent";

type Mode = "lan" | "domain" | "tunnel";

export function SettingsView() {
  const [config, setConfig] = useState<ShareConfig | null>(null);
  const [mode, setMode] = useState<Mode>("lan");
  const [domain, setDomain] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [cache, setCache] = useState<CacheStats | null>(null);
  const [cacheLimitGB, setCacheLimitGB] = useState(5);
  const [storage, setStorage] = useState<StorageSettings | null>(null);
  const [storageKind, setStorageKind] = useState<"self" | "channel">("self");
  const [channelID, setChannelID] = useState("");
  const [accessHash, setAccessHash] = useState("");
  const [channelTitle, setChannelTitle] = useState("");
  const [auth, setAuth] = useState<APIAuthConfig | null>(null);
  const [authMode, setAuthMode] = useState<"open" | "basic">("open");
  const [authUser, setAuthUser] = useState("");
  const [authPass, setAuthPass] = useState("");

  async function refresh() {
    setLoading(true);
    setError(null);
    try {
      const result = await getShareConfig();
      setConfig(result.config);
      setMode((result.config.mode as Mode) || "lan");
      setDomain(result.config.domain || "");
      setBaseUrl(result.config.base_url || "");
      const cacheResult = await getCacheStats();
      setCache(cacheResult.cache);
      setCacheLimitGB(Math.max(1, Math.round(cacheResult.cache.max_bytes / (1024 * 1024 * 1024))));
      const storageResult = await getStorageSettings();
      setStorage(storageResult.storage);
      setStorageKind((storageResult.storage.peer_kind as "self" | "channel") || "self");
      setChannelID(storageResult.storage.channel_id ? String(storageResult.storage.channel_id) : "");
      setAccessHash(storageResult.storage.access_hash ? String(storageResult.storage.access_hash) : "");
      setChannelTitle(storageResult.storage.title || "");
      const authResult = await getAPIAuth();
      setAuth(authResult.auth);
      setAuthMode((authResult.auth.mode as "open" | "basic") || "open");
      setAuthUser(authResult.auth.username || "");
      setAuthPass("");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function saveCache(mode: string, gb: number) {
    setLoading(true);
    try {
      const result = await setCacheConfig(mode, gb * 1024 * 1024 * 1024);
      setCache(result.cache);
      setNotice("Đã cập nhật cấu hình cache");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function runCleanup() {
    setLoading(true);
    try {
      const result = await cleanupCache();
      setCache(result.cache);
      setNotice(`Đã dọn ${result.removed} file khỏi cache local`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function saveStorage() {
    setLoading(true);
    try {
      const payload: { peer_kind: string; channel_id?: number; access_hash?: number; title?: string } = { peer_kind: storageKind };
      if (storageKind === "channel") {
        payload.channel_id = Number(channelID || 0);
        payload.access_hash = Number(accessHash || 0);
        payload.title = channelTitle;
      }
      const result = await updateStorageSettings(payload);
      setStorage(result.storage);
      setNotice(storageKind === "self" ? "Đã chuyển về Saved Messages" : "Đã lưu cấu hình storage channel");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function saveAuth() {
    setLoading(true);
    try {
      const payload: { mode: string; username?: string; password?: string } = { mode: authMode };
      if (authMode === "basic") {
        payload.username = authUser || "admin";
        if (authPass) payload.password = authPass;
      }
      const result = await updateAPIAuth(payload);
      setAuth(result.auth);
      setAuthPass("");
      setNotice(authMode === "basic" ? "Đã bật bảo vệ Basic Auth" : "Đã tắt bảo vệ API");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { refresh(); }, []);

  useEffect(() => {
    const stream = new EventSource(eventsUrl());
    stream.addEventListener("share.config", refresh);
    return () => stream.close();
  }, []);

  async function save(nextMode?: Mode) {
    setLoading(true);
    setError(null);
    setNotice(null);
    try {
      const targetMode: Mode = nextMode || mode;
      const previousMode: string = mode;
      if (targetMode === "tunnel") {
        await controlTunnel("start");
      } else if (previousMode === "tunnel") {
        await controlTunnel("stop");
      }
      const result = await updateShareConfig({
        mode: targetMode,
        domain: targetMode === "domain" ? domain : "",
        base_url: targetMode === "domain" && baseUrl ? baseUrl : "",
      });
      setConfig(result.config);
      setMode((result.config.mode as Mode) || "lan");
      setNotice(result.config.health_message || "Đã lưu cấu hình.");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="settings-view">
      <header className="settings-view__header">
        <div>
          <h2>Cấu hình chia sẻ</h2>
          <p>Chọn cách hệ thống tạo link chia sẻ. Mọi thứ kỹ thuật do app tự lo phía sau.</p>
        </div>
      </header>

      <div className="settings-modes">
        <ModeCard
          active={mode === "lan"}
          icon={<Wifi size={22} />}
          title="Chỉ trong mạng LAN"
          description="Không cần domain. Link chỉ mở được khi cùng Wi‑Fi."
          onClick={() => save("lan")}
        />
        <ModeCard
          active={mode === "domain"}
          icon={<Globe size={22} />}
          title="Dùng tên miền của tôi"
          description="Trỏ subdomain về máy này. App tự kiểm tra và phục vụ link."
          onClick={() => setMode("domain")}
        />
        <ModeCard
          active={mode === "tunnel"}
          icon={<Cloud size={22} />}
          title="Cloudflare Tunnel"
          description="Bật một phát có ngay link public, không cần domain. Cần cài cloudflared trên máy."
          onClick={() => save("tunnel")}
        />
      </div>

      {mode === "domain" && (
        <div className="settings-form">
          <label>
            <span>Tên miền chia sẻ</span>
            <input value={domain} onChange={(event) => setDomain(event.target.value)} placeholder="share.tencuaban.com" />
          </label>
          <label>
            <span>Base URL (tùy chọn)</span>
            <input value={baseUrl} onChange={(event) => setBaseUrl(event.target.value)} placeholder="https://share.tencuaban.com" />
          </label>
          <button className="button button--primary" onClick={() => save("domain")} disabled={loading}>Kiểm tra và lưu</button>
        </div>
      )}

      {notice && <div className="success-note">{notice}</div>}
      {error && <div className="error-note">{error}</div>}

      {config && (
        <div className="settings-status">
          <div className={`settings-status__icon ${config.health_ok ? "settings-status__icon--ok" : "settings-status__icon--warn"}`}>
            {config.health_ok ? <CheckCircle2 size={22} /> : <XCircle size={22} />}
          </div>
          <div>
            <strong>Trạng thái chia sẻ</strong>
            <span>{config.health_message || (config.health_ok ? "Sẵn sàng" : "Chưa kết nối được")}</span>
            <span className="muted-text">Local: {config.local_base_url || "-"}</span>
            {config.tunnel_active && config.tunnel_url && <span className="muted-text">Tunnel: {config.tunnel_url}</span>}
          </div>
        </div>
      )}

      {cache && (
        <div className="settings-cache">
          <header>
            <h3>Bộ nhớ cache local</h3>
            <span>{formatBytes(cache.used_bytes)} / {formatBytes(cache.max_bytes)} · {cache.files} file</span>
          </header>
          <div className="settings-cache__modes">
            <button className={`mode-card ${cache.mode === "smart" ? "mode-card--active" : ""}`} onClick={() => saveCache("smart", cacheLimitGB)}>
              <strong>Smart cache</strong><span>Tự xóa file ít dùng để giữ dưới giới hạn (đề xuất)</span>
            </button>
            <button className={`mode-card ${cache.mode === "cloud_only" ? "mode-card--active" : ""}`} onClick={() => saveCache("cloud_only", cacheLimitGB)}>
              <strong>Chỉ trên cloud</strong><span>Sync xong xóa cache, mỗi lần xem kéo lại từ Telegram</span>
            </button>
            <button className={`mode-card ${cache.mode === "mirror" ? "mode-card--active" : ""}`} onClick={() => saveCache("mirror", cacheLimitGB)}>
              <strong>Giữ tất cả</strong><span>Phù hợp khi máy nhiều ổ, mọi file luôn có sẵn local</span>
            </button>
          </div>
          <label className="settings-cache__limit">
            Giới hạn cache (GB)
            <input type="number" min={1} value={cacheLimitGB} onChange={(event) => setCacheLimitGB(Math.max(1, Number(event.target.value)))} onBlur={() => saveCache(cache.mode, cacheLimitGB)} />
          </label>
          <button className="button button--secondary" onClick={runCleanup}>Dọn cache ngay</button>
        </div>
      )}
      {storage && (
        <div className="settings-cache">
          <header>
            <h3>Lưu trữ Telegram</h3>
            <span>{storage.peer_kind === "channel" ? "Đang dùng channel riêng" : "Đang dùng Saved Messages"}</span>
          </header>
          <div className="settings-cache__modes">
            <button className={`mode-card ${storageKind === "self" ? "mode-card--active" : ""}`} onClick={() => setStorageKind("self")}>
              <strong>Saved Messages</strong><span>Lưu file trong tin nhắn đã lưu của tài khoản (mặc định)</span>
            </button>
            <button className={`mode-card ${storageKind === "channel" ? "mode-card--active" : ""}`} onClick={() => setStorageKind("channel")}>
              <strong>Channel riêng</strong><span>Dùng private channel để lưu file, tách khỏi Saved Messages</span>
            </button>
          </div>
          {storageKind === "channel" && (
            <div className="settings-form">
              <label>
                <span>Channel ID</span>
                <input value={channelID} onChange={(event) => setChannelID(event.target.value.replace(/[^0-9-]/g, ""))} placeholder="ví dụ 1234567890" />
              </label>
              <label>
                <span>Access Hash</span>
                <input value={accessHash} onChange={(event) => setAccessHash(event.target.value.replace(/[^0-9-]/g, ""))} placeholder="ví dụ 9876543210" />
              </label>
              <label>
                <span>Tên channel (tùy chọn)</span>
                <input value={channelTitle} onChange={(event) => setChannelTitle(event.target.value)} placeholder="Tên hiển thị" />
              </label>
            </div>
          )}
          <button className="button button--primary" onClick={saveStorage} disabled={loading}>Lưu cấu hình lưu trữ</button>
        </div>
      )}

      {auth && (
        <div className="settings-cache">
          <header>
            <h3>Bảo mật API</h3>
            <span>{auth.mode === "basic" ? "Đang bật Basic Auth" : "Đang mở (chỉ dùng cho LAN/desktop)"}</span>
          </header>
          <div className="settings-cache__modes">
            <button className={`mode-card ${authMode === "open" ? "mode-card--active" : ""}`} onClick={() => setAuthMode("open")}>
              <strong>Mở</strong><span>Không yêu cầu mật khẩu. Phù hợp khi chạy desktop hoặc LAN tin cậy.</span>
            </button>
            <button className={`mode-card ${authMode === "basic" ? "mode-card--active" : ""}`} onClick={() => setAuthMode("basic")}>
              <strong>Basic Auth</strong><span>Yêu cầu user/password cho mọi truy cập, kể cả WebDAV. Bắt buộc khi deploy VPS.</span>
            </button>
          </div>
          {authMode === "basic" && (
            <div className="settings-form">
              <label>
                <span>Tên đăng nhập</span>
                <input value={authUser} onChange={(event) => setAuthUser(event.target.value)} placeholder="admin" />
              </label>
              <label>
                <span>Mật khẩu mới {auth.has_password && <em>(để trống nếu giữ nguyên)</em>}</span>
                <input type="password" value={authPass} onChange={(event) => setAuthPass(event.target.value)} placeholder="••••••••" />
              </label>
            </div>
          )}
          <button className="button button--primary" onClick={saveAuth} disabled={loading}>Lưu cài đặt bảo mật</button>
        </div>
      )}
    </section>
  );
}

function formatBytes(bytes: number) {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) { value /= 1024; unit += 1; }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unit]}`;
}

function ModeCard({ active, icon, title, description, onClick, disabled }: { active: boolean; icon: ReactNode; title: string; description: string; onClick: () => void; disabled?: boolean; }) {
  return (
    <button
      className={`mode-card ${active ? "mode-card--active" : ""}`}
      onClick={onClick}
      disabled={disabled}
    >
      <div className="mode-card__icon">{icon}</div>
      <div className="mode-card__body">
        <strong>{title}</strong>
        <span>{description}</span>
      </div>
    </button>
  );
}
