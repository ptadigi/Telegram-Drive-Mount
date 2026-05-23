import { KeyRound, Lock, Phone, ShieldCheck } from "lucide-react";
import { FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { AuthStatus, saveTelegramConfig, startTelegramLogin, submitTelegramCode, submitTelegramPassword } from "../api/agent";

type Step = "credentials" | "phone" | "code" | "password" | "done";

type Props = {
  auth: AuthStatus | null;
  onAuthChange: (auth: AuthStatus) => void;
};

export function TelegramLoginPanel({ auth, onAuthChange }: Props) {
  const { t } = useTranslation();
  const [step, setStep] = useState<Step>(auth?.configured ? "phone" : "credentials");
  const [apiId, setApiId] = useState("");
  const [apiHash, setApiHash] = useState("");
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submitCredentials(event: FormEvent) {
    event.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const parsedApiId = Number(apiId);
      if (!parsedApiId || !apiHash.trim()) throw new Error(t("login.errors.credentials"));
      const nextAuth = await saveTelegramConfig(parsedApiId, apiHash.trim());
      onAuthChange(nextAuth);
      setStep("phone");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function submitPhone(event: FormEvent) {
    event.preventDefault();
    setLoading(true);
    setError(null);
    try {
      await startTelegramLogin(phone.trim());
      setStep("code");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
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
      setError(err instanceof Error ? err.message : String(err));
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
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
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

      {step === "credentials" && (
        <form className="form" onSubmit={submitCredentials}>
          <label>
            {t("login.apiId")}
            <div className="input-wrap"><KeyRound size={17} /><input value={apiId} onChange={(e) => setApiId(e.target.value)} inputMode="numeric" placeholder="12345678" /></div>
          </label>
          <label>
            {t("login.apiHash")}
            <div className="input-wrap"><KeyRound size={17} /><input value={apiHash} onChange={(e) => setApiHash(e.target.value)} placeholder="abcdef123456..." /></div>
          </label>
          <button className="button button--primary" disabled={loading}>{loading ? t("login.loading") : t("login.saveCredentials")}</button>
        </form>
      )}

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
            <div className="input-wrap"><KeyRound size={17} /><input value={code} onChange={(e) => setCode(e.target.value)} placeholder="12345" /></div>
          </label>
          <button className="button button--primary" disabled={loading}>{loading ? t("login.loading") : t("login.verifyCode")}</button>
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
