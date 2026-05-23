import { Cloud, Database, HardDrive, Radio, Share2, Smartphone, UploadCloud } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { AgentConfig, AgentInfo, AuthStatus, DatabaseStatus, getAuthStatus, getConfig, getDatabaseStatus, getHealth, getInfo } from "./api/agent";

type AgentState = "checking" | "online" | "offline";

export function App() {
  const { t } = useTranslation();
  const [agentState, setAgentState] = useState<AgentState>("checking");
  const [info, setInfo] = useState<AgentInfo | null>(null);
  const [config, setConfig] = useState<AgentConfig | null>(null);
  const [database, setDatabase] = useState<DatabaseStatus | null>(null);
  const [auth, setAuth] = useState<AuthStatus | null>(null);

  useEffect(() => {
    const controller = new AbortController();

    Promise.all([
      getHealth(controller.signal),
      getInfo(controller.signal),
      getConfig(controller.signal),
      getDatabaseStatus(controller.signal),
      getAuthStatus(controller.signal),
    ])
      .then(([, agentInfo, agentConfig, databaseStatus, authStatus]) => {
        setAgentState("online");
        setInfo(agentInfo);
        setConfig(agentConfig);
        setDatabase(databaseStatus);
        setAuth(authStatus);
      })
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

      <section className="panel panel--stack">
        <div className="panel__header">
          <div>
            <h2>{t("sections.agentStatus")}</h2>
            <p>{t("copy.agentStatus")}</p>
          </div>
          <Database className="panel__icon" />
        </div>
        <dl className="status-grid">
          <StatusItem label={t("agent.version")} value={info?.version || "-"} />
          <StatusItem label={t("agent.uptime")} value={info ? `${info.uptime_sec}s` : "-"} />
          <StatusItem label={t("agent.dataDir")} value={config?.data_dir || "-"} />
          <StatusItem label={t("agent.database")} value={database?.exists ? t("agent.ready") : t("agent.notReady")} />
          <StatusItem label={t("agent.telegramSession")} value={auth?.session_exists ? t("agent.ready") : t("agent.notReady")} />
          <StatusItem label={t("agent.telegramApi")} value={auth?.configured ? t("agent.ready") : t("agent.notReady")} />
          <StatusItem label={t("agent.loginState")} value={auth?.login_started ? t("agent.loginStarted") : t("agent.notStarted")} />
        </dl>
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

function StatusItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}
