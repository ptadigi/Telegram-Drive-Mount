import { Cloud, Database, FolderOpen, HardDrive, Home, Link2, Search, Settings, Share2, Star, Trash2, UploadCloud } from "lucide-react";
import { ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { AgentConfig, AgentInfo, AppUser, AuthStatus, createFolder, DatabaseStatus, getAuthStatus, getConfig, getDatabaseStatus, getHealth, getInfo, appLogout } from "./api/agent";
import { ActivityView } from "./components/ActivityView";
import { AuthGate } from "./components/AuthGate";
import { DriveBrowser } from "./components/DriveBrowser";
import { HomeView } from "./components/HomeView";
import { SearchView } from "./components/SearchView";
import { SettingsView } from "./components/SettingsView";
import { SharePage } from "./components/SharePage";
import { StarredView } from "./components/StarredView";
import { SyncRootsPanel } from "./components/SyncRootsPanel";
import { TelegramLoginPanel } from "./components/TelegramLoginPanel";
import { TrashView } from "./components/TrashView";
import { UploadDock } from "./components/UploadDock";
import { useUploadQueue } from "./state/uploads";

type AgentState = "checking" | "online" | "offline";
type ViewKey = "home" | "drive" | "computers" | "shared" | "starred" | "trash" | "settings" | "search" | "activity";

export function App() {
  const sharedPath = window.location.pathname.match(/^\/share\/(.+)$/);
  if (sharedPath) {
    return <SharePage slug={decodeURIComponent(sharedPath[1])} />;
  }
  return <AuthGuard />;
}

function AuthGuard() {
  const [user, setUser] = useState<AppUser | null>(null);
  if (!user) {
    return <AuthGate onAuthorized={setUser} />;
  }
  return <DriveApp currentUser={user} onLogout={async () => { await appLogout().catch(() => undefined); setUser(null); }} />;
}

function DriveApp({ currentUser, onLogout }: { currentUser: AppUser; onLogout: () => void }) {
  const { t } = useTranslation();
  const [agentState, setAgentState] = useState<AgentState>("checking");
  const [info, setInfo] = useState<AgentInfo | null>(null);
  const [config, setConfig] = useState<AgentConfig | null>(null);
  const [database, setDatabase] = useState<DatabaseStatus | null>(null);
  const [auth, setAuth] = useState<AuthStatus | null>(null);
  const [view, setView] = useState<ViewKey>("drive");
  const [newMenu, setNewMenu] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
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
          <a className="drive-nav__item drive-nav__item--ghost" onClick={() => setView("activity")}><Link2 size={18} /> Hoạt động</a>
        </nav>
        <div className="storage-card">
          <HardDrive size={18} />
          <div><strong>{t("drive.virtualDisk")}</strong><span>{t("drive.telegramBackend")}</span></div>
        </div>
      </aside>

      <section className="drive-main">
        <header className="drive-topbar">
          <div className="search-box"><Search size={18} /><input value={searchQuery} onChange={(event) => { setSearchQuery(event.target.value); if (event.target.value) setView("search"); }} placeholder={t("drive.search")} /></div>
          <div className="status-pills">
            <StatusPill state={agentState} text={agentState === "online" ? t("status.agentOnline") : agentState === "offline" ? t("status.agentOffline") : t("status.agentChecking")} />
            <StatusPill state={auth?.authorized ? "online" : "offline"} text={auth?.authorized ? t("login.connectedTitle") : t("login.title")} />
            <button className="icon-button" onClick={() => setView("settings")}><Settings size={18} /></button>
            <button className="icon-button" onClick={onLogout} title={`Đăng xuất ${currentUser.email}`}>↩</button>
          </div>
        </header>

        {!auth?.authorized && <TelegramLoginPanel auth={auth} />}

        {view === "home" && <HomeView info={info} database={database} auth={auth} agentState={agentState} onOpenDrive={() => setView("drive")} onOpenStarred={() => setView("starred")} onOpenSettings={() => setView("settings")} onOpenComputers={() => setView("computers")} />}
        {view === "drive" && <DriveBrowser uploadQueue={queue} rootLabel={t("drive.myDrive")} description={t("drive.myDriveDesc")} />}
        {view === "computers" && <ComputersView t={t} />}
        {view === "shared" && <PlaceholderView title={t("drive.shared")} text={t("drive.sharedSoon")} />}
        {view === "starred" && <StarredView />}
        {view === "search" && <SearchView query={searchQuery} />}
        {view === "trash" && <TrashView />}
        {view === "settings" && <SettingsView />}
        {view === "activity" && <ActivityView />}

        <section className="agent-drawer">
          <Database size={18} />
          <span>{t("agent.dataDir")}: {config?.data_dir || "-"}</span>
        </section>
      </section>

      <UploadDock queue={queue} />
      <nav className="bottom-nav" aria-label="Điều hướng nhanh">
        <button className={view === "drive" ? "active" : ""} onClick={() => setView("drive")}><FolderOpen size={18} /> Drive</button>
        <button className={view === "search" ? "active" : ""} onClick={() => setView("search")}><Search size={18} /> Tìm</button>
        <button className={view === "starred" ? "active" : ""} onClick={() => setView("starred")}><Star size={18} /> Sao</button>
        <button className={view === "computers" ? "active" : ""} onClick={() => setView("computers")}><HardDrive size={18} /> Máy tính</button>
        <button className={view === "settings" ? "active" : ""} onClick={() => setView("settings")}><Settings size={18} /> Cài đặt</button>
      </nav>
    </main>
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
