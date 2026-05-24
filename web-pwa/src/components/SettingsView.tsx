import { CheckCircle2, Cloud, Globe, Wifi, XCircle } from "lucide-react";
import { ReactNode, useEffect, useState } from "react";
import { eventsUrl, getShareConfig, ShareConfig, updateShareConfig } from "../api/agent";

type Mode = "lan" | "domain" | "tunnel";

export function SettingsView() {
  const [config, setConfig] = useState<ShareConfig | null>(null);
  const [mode, setMode] = useState<Mode>("lan");
  const [domain, setDomain] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  async function refresh() {
    setLoading(true);
    setError(null);
    try {
      const result = await getShareConfig();
      setConfig(result.config);
      setMode((result.config.mode as Mode) || "lan");
      setDomain(result.config.domain || "");
      setBaseUrl(result.config.base_url || "");
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
      const targetMode = nextMode || mode;
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
          description="Bật một phát có ngay link public, không cần domain. (Sắp ra)"
          onClick={() => save("tunnel")}
          disabled
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
          </div>
        </div>
      )}
    </section>
  );
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
