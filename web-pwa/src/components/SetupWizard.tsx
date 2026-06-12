import { CheckCircle2, Cloud, KeyRound, Loader2, Server, XCircle } from "../icons";
import { useEffect, useState } from "react";
import { DesktopServerInfo, DesktopState, getDesktopState, pairDesktop, resetDesktop, setDesktopLocal, testDesktopServer } from "../api/agent";

type Step = "loading" | "choose" | "remote" | "local" | "done";

export function SetupWizard() {
  const [step, setStep] = useState<Step>("loading");
  const [state, setState] = useState<DesktopState | null>(null);
  const [url, setUrl] = useState("");
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [serverInfo, setServerInfo] = useState<DesktopServerInfo | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getDesktopState()
      .then((res) => {
        setState(res.state);
        setStep(res.state.mode === "unset" ? "choose" : "done");
        if (res.state.server_url) setUrl(res.state.server_url);
      })
      .catch(() => setStep("choose"));
  }, []);

  async function checkServer() {
    setBusy(true);
    setError(null);
    setServerInfo(null);
    try {
      const info = await testDesktopServer(url);
      setServerInfo(info);
      if (!info.ok) setError(info.error || "Không kết nối được máy chủ");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function doPair() {
    setBusy(true);
    setError(null);
    try {
      const res = await pairDesktop(url, code, name);
      setState(res.state);
      setStep("done");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function doLocal() {
    setBusy(true);
    setError(null);
    try {
      const res = await setDesktopLocal();
      setState(res.state);
      setStep("done");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function doReset() {
    setBusy(true);
    try {
      const res = await resetDesktop();
      setState(res.state);
      setServerInfo(null);
      setCode("");
      setStep("choose");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="setup-shell">
      <div className="setup-card">
        <header className="setup-card__header">
          <div className="setup-card__logo"><Cloud size={26} /></div>
          <div>
            <h1>Ổ Đĩa Cloud Ảo</h1>
            <p>Thiết lập kết nối tới máy chủ Telegram Drive của bạn.</p>
          </div>
        </header>

        {step === "loading" && <div className="setup-loading"><Loader2 className="spin" size={20} /> Đang tải cấu hình...</div>}

        {step === "choose" && (
          <div className="setup-choices">
            <button className="setup-choice" onClick={() => setStep("remote")}>
              <Server size={22} />
              <div>
                <strong>Nối tới máy chủ có sẵn</strong>
                <span>Máy chủ chạy trên VPS hoặc máy khác trong mạng. Nhập URL và mã ghép thiết bị.</span>
              </div>
            </button>
            <button className="setup-choice" onClick={() => setStep("local")}>
              <Cloud size={22} />
              <div>
                <strong>Chạy máy chủ trên máy này</strong>
                <span>Biến máy này thành máy chủ đầy đủ: lưu metadata, đăng nhập Telegram, mount ổ ảo.</span>
              </div>
            </button>
          </div>
        )}

        {step === "remote" && (
          <div className="setup-form">
            <label>
              <span>1. Địa chỉ máy chủ</span>
              <input value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://drive.tencuaban.com" />
            </label>
            <button className="button button--secondary" onClick={checkServer} disabled={busy || !url}>
              {busy ? "Đang kiểm tra..." : "Kiểm tra kết nối"}
            </button>
            {serverInfo?.ok && (
              <div className="setup-status setup-status--ok"><CheckCircle2 size={18} /> Đã kết nối: {serverInfo.service} {serverInfo.version}</div>
            )}
            {serverInfo && !serverInfo.ok && (
              <div className="setup-status setup-status--err"><XCircle size={18} /> {serverInfo.error}</div>
            )}
            <div className="setup-hint">
              Mở giao diện máy chủ trên trình duyệt, vào mục <strong>Ghép thiết bị</strong>, bấm <strong>Tạo mã ghép</strong>, rồi dán mã vào đây.
            </div>
            <label>
              <span>2. Mã ghép thiết bị</span>
              <input value={code} onChange={(e) => setCode(e.target.value.toUpperCase())} placeholder="4F2A-9K2X" />
            </label>
            <label>
              <span>Tên thiết bị (tùy chọn)</span>
              <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Laptop của tôi" />
            </label>
            {error && <div className="setup-status setup-status--err"><XCircle size={18} /> {error}</div>}
            <div className="setup-actions">
              <button className="button button--ghost" onClick={() => setStep("choose")} disabled={busy}>Quay lại</button>
              <button className="button button--primary" onClick={doPair} disabled={busy || !serverInfo?.ok || !code}><KeyRound size={16} /> Kết nối</button>
            </div>
          </div>
        )}

        {step === "local" && (
          <div className="setup-form">
            <p>Máy này sẽ chạy máy chủ đầy đủ. Sau khi bật, hãy đăng nhập Telegram bằng QR trong giao diện. Ổ ảo sẽ mount tại <strong>T:</strong> (Telegram Drive).</p>
            <div className="setup-hint">Bảo mật: Basic Auth sẽ được bật mặc định khi chạy máy chủ. Đổi mật khẩu trong phần Cài đặt.</div>
            {error && <div className="setup-status setup-status--err"><XCircle size={18} /> {error}</div>}
            <div className="setup-actions">
              <button className="button button--ghost" onClick={() => setStep("choose")} disabled={busy}>Quay lại</button>
              <button className="button button--primary" onClick={doLocal} disabled={busy}>Bật máy chủ local</button>
            </div>
          </div>
        )}

        {step === "done" && state && (
          <div className="setup-done">
            <div className="setup-status setup-status--ok"><CheckCircle2 size={20} /> Đã cấu hình xong</div>
            <ul className="setup-summary">
              <li><span>Chế độ</span><strong>{state.mode === "remote" ? "Nối máy chủ" : "Máy chủ local"}</strong></li>
              {state.server_url && <li><span>Máy chủ</span><strong>{state.server_url}</strong></li>}
              <li><span>Ổ ảo</span><strong>{state.mount_point || "T:"} · Telegram Drive</strong></li>
            </ul>
            <div className="setup-actions">
              <button className="button button--ghost" onClick={doReset} disabled={busy}>Cấu hình lại</button>
              <a className="button button--primary" href={state.server_url || "/"} target="_blank" rel="noreferrer">Mở Drive</a>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
