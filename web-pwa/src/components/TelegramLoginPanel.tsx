import { Lock, Phone, ShieldCheck } from "lucide-react";
import { FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { AuthStatus, startTelegramLogin, submitTelegramCode, submitTelegramPassword } from "../api/agent";

type Step = "phone" | "code" | "password" | "done";

type Props = {
  auth: AuthStatus | null;
};

export function TelegramLoginPanel({ auth }: Props) {
  const { t } = useTranslation();
  const [step, setStep] = useState<Step>(auth?.authorized ? "done" : "phone");
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [codeInfo, setCodeInfo] = useState<string | null>(null);

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

      {step === "phone" && (
        <form className="form" onSubmit={submitPhone}>
          <label>
            {t("login.phone")}
            <div className="input-wrap"><Phone size={17} /><input value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="+84 901 234 567" /></div>
          </label>
          <button className="button button--primary" disabled={loading}>{loading ? t("login.loading") : t("login.sendCode")}</button>
        </form>
      )}

      {step === "code" && (
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

      {step === "password" && (
        <form className="form" onSubmit={submitPassword}>
          <label>
            {t("login.password")}
            <div className="input-wrap"><Lock size={17} /><input value={password} onChange={(e) => setPassword(e.target.value)} type="password" placeholder={t("login.passwordPlaceholder")} /></div>
          </label>
          <button className="button button--primary" disabled={loading}>{loading ? t("login.loading") : t("login.unlock")}</button>
        </form>
      )}

      {step === "done" && <div className="success-note">{t("login.done")}</div>}
      {error && <div className="error-note">{error}</div>}
    </section>
  );
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
