import { Cloud, Database, FolderOpen, HardDrive, Home, Link2, Search, Settings, Share2, Star, Trash2, UploadCloud } from "lucide-react";
import { ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { AgentConfig, AgentInfo, AuthStatus, createFolder, DatabaseStatus, getAuthStatus, getConfig, getDatabaseStatus, getHealth, getInfo } from "./api/agent";
import { DriveBrowser } from "./components/DriveBrowser";
import { SyncRootsPanel } from "./components/SyncRootsPanel";
import { TelegramLoginPanel } from "./components/TelegramLoginPanel";
import { TrashView } from "./components/TrashView";
import { UploadDock } from "./components/UploadDock";
import { useUploadQueue } from "./state/uploads";

type AgentState = "checking" | "online" | "offline";
type ViewKey = "home" | "drive" | "computers" | "shared" | "starred" | "trash";

export function App() {
  const { t } = useTranslation();
  const [agentState, setAgentState] = useState<AgentState>("checking");
  const [info, setInfo] = useState<AgentInfo | null>(null);
  const [config, setConfig] = useState<AgentConfig | null>(null);
  const [database, setDatabase] = useState<DatabaseStatus | null>(null);
  const [auth, setAuth] = useState<AuthStatus | null>(null);
  const [view, setView] = useState<ViewKey>("drive");
  const [newMenu, setNewMenu] = useState(false);
  const newMenuRef = useRef<HTMLDivElement | null>(null);
  const queue = useUploadQueue();

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

  useEffect(() => {
    function onClick(event: MouseEvent) {
      if (!newMenuRef.current) return;
      if (!newMenuRef.current.contains(event.target as Node)) setNewMenu(false);
    }
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  const navItems = useMemo(() => ([
    { key: "home", label: t("drive.home"), icon: <Home size={18} /> },
    { key: "drive", label: t("drive.myDrive"), icon: <FolderOpen size={18} /> },
    { key: "computers", label: t("drive.computers"), icon: <HardDrive size={18} /> },
    { key: "shared", label: t("drive.shared"), icon: <Share2 size={18} /> },
    { key: "starred", label: t("drive.starred"), icon: <Star size={18} /> },
    { key: "trash", label: t("drive.trash"), icon: <Trash2 size={18} /> },
  ]) as { key: ViewKey; label: string; icon: ReactNode }[], [t]);

  function trigger(action: "file" | "folder" | "newfolder" | "share") {
    setNewMenu(false);
    if (action === "file") document.getElementById("hidden-file-input")?.click();
    if (action === "folder") document.getElementById("hidden-folder-input")?.click();
    if (action === "newfolder") {
      const name = window.prompt(t("files.folderNamePrompt"));
      if (name) createFolder(name).catch(() => undefined);
    }
    if (action === "share") window.alert(t("drive.shareSoon"));
  }

  return (
    <main className="drive-shell">
      <aside className="drive-sidebar">
        <div className="brand-mark"><Cloud size={22} /><strong>{t("app.name")}</strong></div>
        <div ref={newMenuRef} className="new-button-wrap">
          <button className="new-button" onClick={() => setNewMenu((m) => !m)}><UploadCloud size={20} /> {t("drive.newAction")}</button>
          {newMenu && <div className="new-menu">
            <button onClick={() => trigger("file")}>{t("files.upload")}</button>
            <button onClick={() => trigger("folder")}>{t("files.uploadFolder")}</button>
            <button onClick={() => trigger("newfolder")}>{t("files.createFolder")}</button>
            <button onClick={() => trigger("share")}>{t("drive.createShareLink")}</button>
          </div>}
        </div>
        <nav className="drive-nav">
          {navItems.map((item) => (
            <button key={item.key} className={`drive-nav__item ${view === item.key ? "drive-nav__item--active" : ""}`} onClick={() => setView(item.key)}>
              {item.icon} {item.label}
            </button>
          ))}
          <a className="drive-nav__item drive-nav__item--ghost"><Link2 size={18} /> {t("drive.links")}</a>
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

        {!auth?.authorized && <TelegramLoginPanel auth={auth} />}

        {view === "home" && <HomeView t={t} info={info} config={config} database={database} auth={auth} agentState={agentState} />}
        {view === "drive" && <DriveBrowser uploadQueue={queue} rootLabel={t("drive.myDrive")} description={t("drive.myDriveDesc")} />}
        {view === "computers" && <ComputersView t={t} />}
        {view === "shared" && <PlaceholderView title={t("drive.shared")} text={t("drive.sharedSoon")} />}
        {view === "starred" && <PlaceholderView title={t("drive.starred")} text={t("drive.starredSoon")} />}
        {view === "trash" && <TrashView />}

        <section className="agent-drawer">
          <Database size={18} />
          <span>{t("agent.dataDir")}: {config?.data_dir || "-"}</span>
        </section>
      </section>

      <UploadDock queue={queue} />
    </main>
  );
}

function HomeView({ t, info, config, database, auth, agentState }: { t: ReturnType<typeof useTranslation>["t"]; info: AgentInfo | null; config: AgentConfig | null; database: DatabaseStatus | null; auth: AuthStatus | null; agentState: AgentState; }) {
  return (
    <section className="drive-hero-card">
      <div>
        <span>{t("drive.heroEyebrow")}</span>
        <h1>{t("drive.heroTitle")}</h1>
        <p>{t("drive.heroText")}</p>
        <p className="muted-text">{t("agent.dataDir")}: {config?.data_dir || "-"}</p>
      </div>
      <div className="drive-stats">
        <MiniStat label={t("status.agentOnline")} value={agentState === "online" ? t("agent.ready") : t("agent.notReady")} />
        <MiniStat label={t("agent.database")} value={database?.exists ? t("agent.ready") : t("agent.notReady")} />
        <MiniStat label={t("agent.telegramSession")} value={auth?.session_exists ? t("agent.ready") : t("agent.notReady")} />
        <MiniStat label={t("agent.uptime")} value={info ? `${info.uptime_sec}s` : "-"} />
      </div>
    </section>
  );
}

function ComputersView({ t }: { t: ReturnType<typeof useTranslation>["t"]; }) {
  return (
    <section className="computers-view">
      <header className="computers-view__header">
        <div>
          <h2>{t("drive.computers")}</h2>
          <p>{t("drive.computersDesc")}</p>
        </div>
      </header>
      <SyncRootsPanel />
    </section>
  );
}

function PlaceholderView({ title, text }: { title: string; text: string }) {
  return (
    <section className="placeholder-view">
      <h2>{title}</h2>
      <p>{text}</p>
    </section>
  );
}

function StatusPill({ state, text }: { state: AgentState | "online" | "offline"; text: string }) {
  return <div className={`status-pill status-pill--${state}`}><span />{text}</div>;
}

function MiniStat({ label, value }: { label: string; value: string }) {
  return <div className="mini-stat"><span>{label}</span><strong>{value}</strong></div>;
}
