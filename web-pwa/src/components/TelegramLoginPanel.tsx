import { Lock, Phone, QrCode, ShieldCheck } from "lucide-react";
import { FormEvent, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  AuthStatus,
  cancelTelegramQR,
  getTelegramQRStatus,
  startTelegramLogin,
  startTelegramQR,
  submitTelegramCode,
  submitTelegramPassword,
  submitTelegramQRPassword,
  TelegramQRStatus,
} from "../api/agent";

type Mode = "phone" | "qr";
type Step = "phone" | "code" | "password" | "done";

type Props = {
  auth: AuthStatus | null;
};

export function TelegramLoginPanel({ auth }: Props) {
  const { t } = useTranslation();
  const [mode, setMode] = useState<Mode>("qr");
  const [step, setStep] = useState<Step>(auth?.authorized ? "done" : "phone");
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [codeInfo, setCodeInfo] = useState<string | null>(null);
  const [qr, setQr] = useState<TelegramQRStatus | null>(null);
  const [qrPassword, setQrPassword] = useState("");
  const pollRef = useRef<number | null>(null);

  useEffect(() => {
    return () => stopPolling();
  }, []);

  function stopPolling() {
    if (pollRef.current !== null) {
      window.clearInterval(pollRef.current);
      pollRef.current = null;
    }
  }

  function pollQRStatus() {
    stopPolling();
    pollRef.current = window.setInterval(async () => {
      try {
        const status = await getTelegramQRStatus();
        setQr(status);
        if (status.state === "authorized") {
          stopPolling();
          setStep("done");
        }
        if (status.state === "expired" || status.state === "error" || status.state === "idle") {
          stopPolling();
        }
      } catch (err) {
        setError(readableLoginError(err));
        stopPolling();
      }
    }, 2000);
  }

  async function startQR() {
    setLoading(true);
    setError(null);
    try {
      const status = await startTelegramQR();
      setQr(status);
      pollQRStatus();
    } catch (err) {
      setError(readableLoginError(err));
    } finally {
      setLoading(false);
    }
  }

  async function submitQRPassword(event: FormEvent) {
    event.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const status = await submitTelegramQRPassword(qrPassword);
      setQr(status);
      setQrPassword("");
    } catch (err) {
      setError(readableLoginError(err));
    } finally {
      setLoading(false);
    }
  }

  async function cancelQR() {
    stopPolling();
    try {
      await cancelTelegramQR();
    } catch (err) {
      // ignore
    }
    setQr(null);
  }

  async function submitPhone(event: FormEvent) {
    event.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const normalizedPhone = normalizePhone(phone);
      setPhone(normalizedPhone);
      const result = await startTelegramLogin(normalizedPhone);
      setCodeInfo(describeCodeType(result.code_type, result.timeout_sec));
      setStep("code");
    } catch (err) {
      setError(readableLoginError(err));
    } finally {
      setLoading(false);
    }
  }

  async function submitCode(event: FormEvent) {
    event.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const result = await submitTelegramCode(code.trim());
      if (result.next_step === "password") {
        setStep("password");
      } else {
        setStep("done");
      }
    } catch (err) {
      setError(readableLoginError(err));
    } finally {
      setLoading(false);
    }
  }

  async function submitPassword(event: FormEvent) {
    event.preventDefault();
    setLoading(true);
    setError(null);
    try {
      await submitTelegramPassword(password);
      setStep("done");
    } catch (err) {
      setError(readableLoginError(err));
    } finally {
      setLoading(false);
    }
  }

  if (auth?.authorized) {
    return (
      <section className="login-card login-card--connected">
        <div className="login-card__header">
          <div className="card__icon"><ShieldCheck /></div>
          <div>
            <h2>{t("login.connectedTitle")}</h2>
            <p>{t("login.connectedDescription")}</p>
          </div>
        </div>
        <div className="success-note">{t("login.done")}</div>
      </section>
    );
  }

  return (
    <section className="login-card">
      <div className="login-card__header">
        <div className="card__icon"><ShieldCheck /></div>
        <div>
          <h2>{t("login.title")}</h2>
          <p>{t("login.description")}</p>
        </div>
      </div>

      <div className="login-tabs">
        <button type="button" className={mode === "qr" ? "active" : ""} onClick={() => { setMode("qr"); setError(null); }}>
          <QrCode size={16} /> Quét QR Telegram
        </button>
        <button type="button" className={mode === "phone" ? "active" : ""} onClick={() => { setMode("phone"); setError(null); cancelQR(); }}>
          <Phone size={16} /> Số điện thoại
        </button>
      </div>

      {mode === "qr" && (
        <div className="qr-login">
          {!qr || qr.state === "idle" ? (
            <button className="button button--primary" onClick={startQR} disabled={loading}>
              {loading ? "Đang tạo mã..." : "Tạo mã QR Telegram"}
            </button>
          ) : null}

          {qr && (qr.state === "pending") && qr.token_url && (
            <div className="qr-block">
              <img alt="Telegram QR" src={qrImageUrl(qr.token_url)} />
              <p className="form-hint">
                Mở Telegram trên điện thoại → Cài đặt → Thiết bị → Liên kết thiết bị → Quét mã.
              </p>
              {qr.expires_at ? <p className="form-hint">Mã hết hạn lúc {new Date(qr.expires_at * 1000).toLocaleTimeString()}.</p> : null}
              <button type="button" className="link-button" onClick={cancelQR}>Huỷ</button>
            </div>
          )}

          {qr && qr.state === "awaiting_password" && (
            <form className="form" onSubmit={submitQRPassword}>
              <label>
                Mật khẩu xác minh hai bước
                <div className="input-wrap"><Lock size={17} /><input value={qrPassword} onChange={(e) => setQrPassword(e.target.value)} type="password" placeholder={t("login.passwordPlaceholder")} /></div>
              </label>
              <button className="button button--primary" disabled={loading}>{loading ? t("login.loading") : t("login.unlock")}</button>
            </form>
          )}

          {qr && qr.state === "expired" && (
            <div className="error-note">Mã QR đã hết hạn. Bấm tạo mã mới.</div>
          )}
          {qr && qr.state === "error" && qr.error && (
            <div className="error-note">{qr.error}</div>
          )}
          {qr && qr.state === "authorized" && (
            <div className="success-note">{t("login.done")}</div>
          )}
        </div>
      )}

      {mode === "phone" && step === "phone" && (
        <form className="form" onSubmit={submitPhone}>
          <label>
            {t("login.phone")}
            <div className="input-wrap"><Phone size={17} /><input value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="+84 901 234 567" /></div>
          </label>
          <button className="button button--primary" disabled={loading}>{loading ? t("login.loading") : t("login.sendCode")}</button>
        </form>
      )}

      {mode === "phone" && step === "code" && (
        <form className="form" onSubmit={submitCode}>
          <label>
            {t("login.code")}
            <div className="input-wrap"><Phone size={17} /><input value={code} onChange={(e) => setCode(e.target.value)} placeholder="12345" /></div>
          </label>
          <button className="button button--primary" disabled={loading}>{loading ? t("login.loading") : t("login.verifyCode")}</button>
          <p className="form-hint">{codeInfo || t("login.codeHint")}</p>
          <button type="button" className="link-button" onClick={() => setStep("phone")}>{t("login.changePhone")}</button>
        </form>
      )}

      {mode === "phone" && step === "password" && (
        <form className="form" onSubmit={submitPassword}>
          <label>
            {t("login.password")}
            <div className="input-wrap"><Lock size={17} /><input value={password} onChange={(e) => setPassword(e.target.value)} type="password" placeholder={t("login.passwordPlaceholder")} /></div>
          </label>
          <button className="button button--primary" disabled={loading}>{loading ? t("login.loading") : t("login.unlock")}</button>
        </form>
      )}

      {mode === "phone" && step === "done" && <div className="success-note">{t("login.done")}</div>}
      {error && <div className="error-note">{error}</div>}
    </section>
  );
}

function qrImageUrl(tokenUrl: string) {
  const encoded = encodeURIComponent(tokenUrl);
  return `https://api.qrserver.com/v1/create-qr-code/?size=240x240&margin=8&data=${encoded}`;
}

function normalizePhone(value: string) {
  const trimmed = value.trim().replace(/\s+/g, "");
  if (trimmed.startsWith("+")) return trimmed;
  if (trimmed.startsWith("00")) return `+${trimmed.slice(2)}`;
  if (trimmed.startsWith("84")) return `+${trimmed}`;
  if (trimmed.startsWith("0")) return `+84${trimmed.slice(1)}`;
  return `+${trimmed}`;
}

function describeCodeType(codeType: string, timeout: number) {
  const waitText = timeout > 0 ? ` Nếu chưa thấy, hãy chờ khoảng ${timeout} giây rồi thử lại.` : "";
  if (codeType.includes("App")) return `Telegram báo mã đã được gửi vào app Telegram của số này.${waitText}`;
  if (codeType.includes("SMS")) return `Telegram báo mã đã được gửi qua SMS.${waitText}`;
  if (codeType.includes("Call")) return `Telegram báo mã sẽ được gửi qua cuộc gọi.${waitText}`;
  return `Telegram đã chấp nhận yêu cầu gửi mã (${codeType}).${waitText}`;
}

function readableLoginError(err: unknown) {
  const message = err instanceof Error ? err.message : String(err);
  if (message.includes("chưa cấu hình API Telegram")) {
    return "Go Agent chưa có cấu hình Telegram nội bộ. Vui lòng khởi động lại Agent bằng cấu hình local của dự án.";
  }
  return message;
}
