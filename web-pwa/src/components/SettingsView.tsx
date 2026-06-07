import { CheckCircle2, Cloud, Globe, HardDrive, Wifi, XCircle } from "lucide-react";
import { ReactNode, useEffect, useState } from "react";
import { APIAuthConfig, CacheStats, cleanupCache, controlTunnel, createStorageChannel, eventsUrl, getAPIAuth, getCacheStats, getMountStatus, getShareConfig, getStorageSettings, MountStatus, setCacheConfig, ShareConfig, startMount, stopMount, StorageSettings, updateAPIAuth, updateShareConfig, updateStorageSettings } from "../api/agent";

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
  const [mountInfo, setMountInfo] = useState<MountStatus | null>(null);
  const [mountPoint, setMountPoint] = useState("");
  const [mountLoading, setMountLoading] = useState(false);

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
      try {
        const mountResult = await getMountStatus();
        setMountInfo(mountResult);
        if (mountResult.mount_point) setMountPoint(mountResult.mount_point);
      } catch (err) {
        // mount endpoint optional in older builds
      }
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
      setNotice("ÄÃ£ cáº­p nháº­t cáº¥u hÃ¬nh cache");
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
      setNotice(`ÄÃ£ dá»n ${result.removed} file khá»i cache local`);
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
      setNotice(storageKind === "self" ? "ÄÃ£ chuyá»ƒn vá» Saved Messages" : "ÄÃ£ lÆ°u cáº¥u hÃ¬nh storage channel");
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
      setNotice(authMode === "basic" ? "ÄÃ£ báº­t báº£o vá»‡ Basic Auth" : "ÄÃ£ táº¯t báº£o vá»‡ API");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function runMount() {
    setMountLoading(true);
    setError(null);
    try {
      const status = await startMount(mountPoint || undefined);
      setMountInfo(status);
      if (status.mount_point) setMountPoint(status.mount_point);
      setNotice(status.mounted ? `ÄÃ£ mount á»• áº£o táº¡i ${status.mount_point || ""}` : "Äang khá»Ÿi Ä‘á»™ng mount, kiá»ƒm tra tráº¡ng thÃ¡i sau vÃ i giÃ¢y.");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setMountLoading(false);
    }
  }

  async function runUnmount() {
    setMountLoading(true);
    setError(null);
    try {
      const status = await stopMount();
      setMountInfo(status);
      setNotice("ÄÃ£ unmount á»• áº£o");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setMountLoading(false);
    }
  }

  useEffect(() => { refresh(); }, []);

  useEffect(() => {
    const stream = new EventSource(eventsUrl(), { withCredentials: true });
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
      setNotice(result.config.health_message || "ÄÃ£ lÆ°u cáº¥u hÃ¬nh.");
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
          <h2>Cáº¥u hÃ¬nh chia sáº»</h2>
          <p>Chá»n cÃ¡ch há»‡ thá»‘ng táº¡o link chia sáº». Má»i thá»© ká»¹ thuáº­t do app tá»± lo phÃ­a sau.</p>
        </div>
      </header>

      <div className="settings-modes">
        <ModeCard
          active={mode === "lan"}
          icon={<Wifi size={22} />}
          title="Chá»‰ trong máº¡ng LAN"
          description="KhÃ´ng cáº§n domain. Link chá»‰ má»Ÿ Ä‘Æ°á»£c khi cÃ¹ng Wiâ€‘Fi."
          onClick={() => save("lan")}
        />
        <ModeCard
          active={mode === "domain"}
          icon={<Globe size={22} />}
          title="DÃ¹ng tÃªn miá»n cá»§a tÃ´i"
          description="Trá» subdomain vá» mÃ¡y nÃ y. App tá»± kiá»ƒm tra vÃ  phá»¥c vá»¥ link."
          onClick={() => setMode("domain")}
        />
        <ModeCard
          active={mode === "tunnel"}
          icon={<Cloud size={22} />}
          title="Cloudflare Tunnel"
          description="Báº­t má»™t phÃ¡t cÃ³ ngay link public, khÃ´ng cáº§n domain. Cáº§n cÃ i cloudflared trÃªn mÃ¡y."
          onClick={() => save("tunnel")}
        />
      </div>

      {mode === "domain" && (
        <div className="settings-form">
          <label>
            <span>TÃªn miá»n chia sáº»</span>
            <input value={domain} onChange={(event) => setDomain(event.target.value)} placeholder="share.tencuaban.com" />
          </label>
          <label>
            <span>Base URL (tÃ¹y chá»n)</span>
            <input value={baseUrl} onChange={(event) => setBaseUrl(event.target.value)} placeholder="https://share.tencuaban.com" />
          </label>
          <button className="button button--primary" onClick={() => save("domain")} disabled={loading}>Kiá»ƒm tra vÃ  lÆ°u</button>
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
            <strong>Tráº¡ng thÃ¡i chia sáº»</strong>
            <span>{config.health_message || (config.health_ok ? "Sáºµn sÃ ng" : "ChÆ°a káº¿t ná»‘i Ä‘Æ°á»£c")}</span>
            <span className="muted-text">Local: {config.local_base_url || "-"}</span>
            {config.tunnel_active && config.tunnel_url && <span className="muted-text">Tunnel: {config.tunnel_url}</span>}
          </div>
        </div>
      )}

      {cache && (
        <div className="settings-cache">
          <header>
            <h3>Bá»™ nhá»› cache local</h3>
            <span>{formatBytes(cache.used_bytes)} / {formatBytes(cache.max_bytes)} Â· {cache.files} file</span>
          </header>
          <div className="settings-cache__modes">
            <button className={`mode-card ${cache.mode === "smart" ? "mode-card--active" : ""}`} onClick={() => saveCache("smart", cacheLimitGB)}>
              <strong>Smart cache</strong><span>Tá»± xÃ³a file Ã­t dÃ¹ng Ä‘á»ƒ giá»¯ dÆ°á»›i giá»›i háº¡n (Ä‘á» xuáº¥t)</span>
            </button>
            <button className={`mode-card ${cache.mode === "cloud_only" ? "mode-card--active" : ""}`} onClick={() => saveCache("cloud_only", cacheLimitGB)}>
              <strong>Chá»‰ trÃªn cloud</strong><span>Sync xong xÃ³a cache, má»—i láº§n xem kÃ©o láº¡i tá»« Telegram</span>
            </button>
            <button className={`mode-card ${cache.mode === "mirror" ? "mode-card--active" : ""}`} onClick={() => saveCache("mirror", cacheLimitGB)}>
              <strong>Giá»¯ táº¥t cáº£</strong><span>PhÃ¹ há»£p khi mÃ¡y nhiá»u á»•, má»i file luÃ´n cÃ³ sáºµn local</span>
            </button>
          </div>
          <label className="settings-cache__limit">
            Giá»›i háº¡n cache (GB)
            <input type="number" min={1} value={cacheLimitGB} onChange={(event) => setCacheLimitGB(Math.max(1, Number(event.target.value)))} onBlur={() => saveCache(cache.mode, cacheLimitGB)} />
          </label>
          <button className="button button--secondary" onClick={runCleanup}>Dá»n cache ngay</button>
        </div>
      )}
      {storage && (
        <div className="settings-cache">
          <header>
            <h3>LÆ°u trá»¯ Telegram</h3>
            <span>{storage.peer_kind === "channel" ? "Äang dÃ¹ng channel riÃªng" : "Äang dÃ¹ng Saved Messages"}</span>
          </header>
          <div className="settings-cache__modes">
            <button className={`mode-card ${storageKind === "self" ? "mode-card--active" : ""}`} onClick={() => setStorageKind("self")}>
              <strong>Saved Messages</strong><span>LÆ°u file trong tin nháº¯n Ä‘Ã£ lÆ°u cá»§a tÃ i khoáº£n (máº·c Ä‘á»‹nh)</span>
            </button>
            <button className={`mode-card ${storageKind === "channel" ? "mode-card--active" : ""}`} onClick={() => setStorageKind("channel")}>
              <strong>Channel riÃªng</strong><span>DÃ¹ng private channel Ä‘á»ƒ lÆ°u file, tÃ¡ch khá»i Saved Messages</span>
            </button>
          </div>
          {storageKind === "channel" && (
            <div className="settings-form">
              <label>
                <span>Channel ID</span>
                <input value={channelID} onChange={(event) => setChannelID(event.target.value.replace(/[^0-9-]/g, ""))} placeholder="vÃ­ dá»¥ 1234567890" />
              </label>
              <label>
                <span>Access Hash</span>
                <input value={accessHash} onChange={(event) => setAccessHash(event.target.value.replace(/[^0-9-]/g, ""))} placeholder="vÃ­ dá»¥ 9876543210" />
              </label>
              <label>
                <span>TÃªn channel (tÃ¹y chá»n)</span>
                <input value={channelTitle} onChange={(event) => setChannelTitle(event.target.value)} placeholder="TÃªn hiá»ƒn thá»‹" />
              </label>
            </div>
          )}
          <button className="button button--primary" onClick={saveStorage} disabled={loading}>LÆ°u cáº¥u hÃ¬nh lÆ°u trá»¯</button>
          {storageKind === "channel" && <button className="button button--secondary" onClick={async () => {
            setLoading(true);
            try {
              const result = await createStorageChannel(channelTitle || "á»” ÄÄ©a Cloud áº¢o");
              setStorage(result.storage);
              setChannelID(String(result.storage.channel_id));
              setAccessHash(String(result.storage.access_hash));
              setChannelTitle(result.storage.title || "");
              setNotice(`ÄÃ£ táº¡o channel ${result.storage.title || result.storage.channel_id}`);
            } catch (err) {
              setError(err instanceof Error ? err.message : String(err));
            } finally {
              setLoading(false);
            }
          }} disabled={loading}>Táº¡o channel má»›i tá»± Ä‘á»™ng</button>}
        </div>
      )}

      {auth && (
        <div className="settings-cache">
          <header>
            <h3>Báº£o máº­t API</h3>
            <span>{auth.mode === "basic" ? "Äang báº­t Basic Auth" : "Äang má»Ÿ (chá»‰ dÃ¹ng cho LAN/desktop)"}</span>
          </header>
          <div className="settings-cache__modes">
            <button className={`mode-card ${authMode === "open" ? "mode-card--active" : ""}`} onClick={() => setAuthMode("open")}>
              <strong>Má»Ÿ</strong><span>KhÃ´ng yÃªu cáº§u máº­t kháº©u. PhÃ¹ há»£p khi cháº¡y desktop hoáº·c LAN tin cáº­y.</span>
            </button>
            <button className={`mode-card ${authMode === "basic" ? "mode-card--active" : ""}`} onClick={() => setAuthMode("basic")}>
              <strong>Basic Auth</strong><span>YÃªu cáº§u user/password cho má»i truy cáº­p, ká»ƒ cáº£ WebDAV. Báº¯t buá»™c khi deploy VPS.</span>
            </button>
          </div>
          {authMode === "basic" && (
            <div className="settings-form">
              <label>
                <span>TÃªn Ä‘Äƒng nháº­p</span>
                <input value={authUser} onChange={(event) => setAuthUser(event.target.value)} placeholder="admin" />
              </label>
              <label>
                <span>Máº­t kháº©u má»›i {auth.has_password && <em>(Ä‘á»ƒ trá»‘ng náº¿u giá»¯ nguyÃªn)</em>}</span>
                <input type="password" value={authPass} onChange={(event) => setAuthPass(event.target.value)} placeholder="â€¢â€¢â€¢â€¢â€¢â€¢â€¢â€¢" />
              </label>
            </div>
          )}
          <button className="button button--primary" onClick={saveAuth} disabled={loading}>LÆ°u cÃ i Ä‘áº·t báº£o máº­t</button>
        </div>
      )}

      <div className="settings-cache">
        <header>
          <h3><HardDrive size={16} /> á»” áº£o Telegram Drive</h3>
          <span>{mountInfo?.available ? `Backend: ${mountInfo.backend}` : "Báº£n build hiá»‡n táº¡i khÃ´ng kÃ¨m mount engine"}</span>
        </header>
        {mountInfo?.available ? (
          <div className="settings-form">
            <label>
              <span>Äiá»ƒm mount</span>
              <input value={mountPoint} onChange={(event) => setMountPoint(event.target.value)} placeholder="T:" />
            </label>
            <p className="form-hint">
              Windows nÃªn dÃ¹ng drive letter dáº¡ng <code>T:</code>. macOS dÃ¹ng <code>/Volumes/Telegram Drive</code>. Linux nÃªn trá» vÃ o thÆ° má»¥c trá»‘ng.
            </p>
            {mountInfo.error && <div className="error-note">{mountInfo.error}</div>}
            <div style={{ display: "flex", gap: 8 }}>
              {!mountInfo.mounted && <button className="button button--primary" onClick={runMount} disabled={mountLoading}>Mount á»• áº£o</button>}
              {mountInfo.mounted && <button className="button button--secondary" onClick={runUnmount} disabled={mountLoading}>Unmount</button>}
              {mountInfo.mounted && mountInfo.mount_point && <span className="form-hint">Äang mount táº¡i <strong>{mountInfo.mount_point}</strong></span>}
            </div>
          </div>
        ) : (
          <p className="form-hint">
            Äá»ƒ báº­t á»• áº£o, build agent vá»›i <code>go build -tags fuse ./cmd/agent</code> vÃ  cÃ i WinFsp (Windows) hoáº·c FUSE (macOS/Linux).
          </p>
        )}
      </div>
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
