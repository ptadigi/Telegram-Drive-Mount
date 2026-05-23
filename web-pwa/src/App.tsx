import { Cloud, HardDrive, Radio, Share2, Smartphone, UploadCloud } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

type AgentState = "checking" | "online" | "offline";

export function App() {
  const { t } = useTranslation();
  const [agentState, setAgentState] = useState<AgentState>("checking");

  useEffect(() => {
    const controller = new AbortController();

    fetch("http://127.0.0.1:8750/health", {
      signal: controller.signal,
    })
      .then((res) => setAgentState(res.ok ? "online" : "offline"))
      .catch(() => setAgentState("offline"));

    return () => controller.abort();
  }, []);

  const statusText =
    agentState === "checking"
      ? t("status.agentChecking")
      : agentState === "online"
        ? t("status.agentOnline")
        : t("status.agentOffline");

  return (
    <main className="shell">
      <section className="hero">
        <div className="hero__badge">
          <Cloud size={18} />
          <span>{t("status.telegramStorage")}</span>
        </div>
        <h1>{t("app.name")}</h1>
        <p>{t("app.tagline")}</p>
        <div className={`agent agent--${agentState}`}>
          <span />
          {statusText}
        </div>
        <div className="actions">
          <button className="button button--primary">
            <UploadCloud size={18} />
            {t("actions.upload")}
          </button>
          <button className="button button--secondary">
            <Radio size={18} />
            {t("actions.connectAgent")}
          </button>
        </div>
      </section>

      <section className="grid">
        <FeatureCard icon={<Smartphone />} title={t("sections.mobileUpload")} text={t("copy.mobileUpload")} />
        <FeatureCard icon={<HardDrive />} title={t("sections.virtualDrive")} text={t("copy.virtualDrive")} />
        <FeatureCard icon={<Share2 />} title={t("status.publicSharing")} text={t("copy.roadmap")} />
      </section>

      <section className="panel">
        <div>
          <h2>{t("sections.recentFiles")}</h2>
          <p>{t("copy.emptyRecent")}</p>
        </div>
        <button className="button button--ghost">{t("actions.openDrive")}</button>
      </section>
    </main>
  );
}

function FeatureCard({ icon, title, text }: { icon: React.ReactNode; title: string; text: string }) {
  return (
    <article className="card">
      <div className="card__icon">{icon}</div>
      <h2>{title}</h2>
      <p>{text}</p>
    </article>
  );
}
