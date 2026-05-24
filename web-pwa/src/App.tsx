import { Clock3, Cloud, Database, FolderOpen, HardDrive, Link2, Search, Settings, Share2, Trash2, UploadCloud } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { AgentConfig, AgentInfo, AuthStatus, DatabaseStatus, getAuthStatus, getConfig, getDatabaseStatus, getHealth, getInfo } from "./api/agent";
import { FileManager } from "./components/FileManager";
import { SyncRootsPanel } from "./components/SyncRootsPanel";
import { TelegramLoginPanel } from "./components/TelegramLoginPanel";

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
    Promise.all([getHealth(controller.signal), getInfo(controller.signal), getConfig(controller.signal), getDatabaseStatus(controller.signal), getAuthStatus(controller.signal)])
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

  return (
    <main className="drive-shell">
      <aside className="drive-sidebar">
        <div className="brand-mark"><Cloud size={22} /><strong>{t("app.name")}</strong></div>
        <a className="new-button" href="#file-manager"><UploadCloud size={20} /> {t("actions.upload")}</a>
        <nav className="drive-nav">
          <a className="drive-nav__item drive-nav__item--active"><FolderOpen size={18} /> {t("drive.myDrive")}</a>
          <a className="drive-nav__item"><Clock3 size={18} /> {t("drive.recent")}</a>
          <a className="drive-nav__item"><Share2 size={18} /> {t("drive.shared")}</a>
          <a className="drive-nav__item"><Link2 size={18} /> {t("drive.links")}</a>
          <a className="drive-nav__item"><Trash2 size={18} /> {t("drive.trash")}</a>
        </nav>
        <div className="storage-card">
          <HardDrive size={18} />
          <div><strong>{t("drive.virtualDisk")}</strong><span>{t("drive.telegramBackend")}</span></div>
        </div>
      </aside>

      <section className="drive-main">
        <header className="drive-topbar">
          <div className="search-box"><Search size={18} /><input placeholder={t("drive.search")} /></div>
          <div className="status-pills">
            <StatusPill state={agentState} text={agentState === "online" ? t("status.agentOnline") : agentState === "offline" ? t("status.agentOffline") : t("status.agentChecking")} />
            <StatusPill state={auth?.authorized ? "online" : "offline"} text={auth?.authorized ? t("login.connectedTitle") : t("login.title")} />
            <button className="icon-button"><Settings size={18} /></button>
          </div>
        </header>

        <section className="drive-hero-card">
          <div>
            <span>{t("drive.heroEyebrow")}</span>
            <h1>{t("drive.heroTitle")}</h1>
            <p>{t("drive.heroText")}</p>
          </div>
          <div className="drive-stats">
            <MiniStat label={t("agent.database")} value={database?.exists ? t("agent.ready") : t("agent.notReady")} />
            <MiniStat label={t("agent.telegramSession")} value={auth?.session_exists ? t("agent.ready") : t("agent.notReady")} />
            <MiniStat label={t("agent.uptime")} value={info ? `${info.uptime_sec}s` : "-"} />
          </div>
        </section>

        {!auth?.authorized && <TelegramLoginPanel auth={auth} />}
        <FileManager />
        <SyncRootsPanel />

        <section className="agent-drawer">
          <Database size={18} />
          <span>{t("agent.dataDir")}: {config?.data_dir || "-"}</span>
        </section>
      </section>
    </main>
  );
}

function StatusPill({ state, text }: { state: AgentState | "online" | "offline"; text: string }) {
  return <div className={`status-pill status-pill--${state}`}><span />{text}</div>;
}

function MiniStat({ label, value }: { label: string; value: string }) {
  return <div className="mini-stat"><span>{label}</span><strong>{value}</strong></div>;
}
