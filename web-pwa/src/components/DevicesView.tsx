import { Laptop2, RefreshCw, ShieldOff, Sparkles } from "../icons";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Device,
  PairingCode,
  listDevices,
  revokeDevice,
  startDevicePairing,
} from "../api/agent";

export function DevicesView() {
  const { t } = useTranslation();
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [code, setCode] = useState<PairingCode | null>(null);
  const [now, setNow] = useState(Math.floor(Date.now() / 1000));

  async function refresh() {
    setLoading(true);
    setError(null);
    try {
      const data = await listDevices();
      setDevices(data.devices ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  useEffect(() => {
    const id = window.setInterval(() => setNow(Math.floor(Date.now() / 1000)), 1000);
    return () => window.clearInterval(id);
  }, []);

  async function generate() {
    try {
      const res = await startDevicePairing();
      setCode(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function revoke(id: string) {
    if (!window.confirm(t("devices.revokeConfirm"))) return;
    try {
      await revokeDevice(id);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  const remaining = code ? Math.max(0, code.expires_at - now) : 0;

  return (
    <section className="devices-view">
      <header className="devices-view__header">
        <div>
          <h2>{t("devices.title")}</h2>
          <p>{t("devices.description")}</p>
        </div>
        <div style={{ display: "flex", gap: 8 }}>
          <button className="button button--secondary" onClick={refresh} disabled={loading}>
            <RefreshCw size={14} /> {t("devices.refresh")}
          </button>
          <button className="button button--primary" onClick={generate}>
            <Sparkles size={14} /> {t("devices.generateCode")}
          </button>
        </div>
      </header>

      {error && <div className="error-note">{error}</div>}

      {code && remaining > 0 && (
        <div className="pair-code-card">
          <strong>{code.code}</strong>
          <p className="form-hint">
            {t("devices.codeHint", { seconds: remaining })}
          </p>
        </div>
      )}

      {code && remaining <= 0 && (
        <div className="muted-box">{t("devices.codeExpired")}</div>
      )}

      <ul className="device-list">
        {devices.length === 0 && (
          <li className="muted-box">{t("devices.empty")}</li>
        )}
        {devices.map((device) => (
          <li key={device.id} className={device.revoked_at ? "device-row device-row--revoked" : "device-row"}>
            <div className="device-row__icon"><Laptop2 size={20} /></div>
            <div className="device-row__body">
              <strong>{device.name}</strong>
              <span>
                {device.platform || t("devices.unknownPlatform")}
                {" · "}
                {t("devices.lastSeen", {
                  time: device.last_seen_at
                    ? new Date(device.last_seen_at * 1000).toLocaleString()
                    : t("devices.never"),
                })}
                {device.last_ip ? " · " + device.last_ip : ""}
              </span>
            </div>
            <button
              className="button button--ghost"
              onClick={() => revoke(device.id)}
              disabled={!!device.revoked_at}
            >
              <ShieldOff size={14} />
              {device.revoked_at ? t("devices.revoked") : t("devices.revoke")}
            </button>
          </li>
        ))}
      </ul>
    </section>
  );
}