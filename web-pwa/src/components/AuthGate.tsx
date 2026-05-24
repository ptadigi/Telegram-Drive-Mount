import { useEffect, useState } from "react";
import { appLogin, appRegister, AppUser, getAppMe } from "../api/agent";
import { useToast } from "../state/ui";

type Props = {
  onAuthorized: (user: AppUser) => void;
};

export function AuthGate({ onAuthorized }: Props) {
  const [needSetup, setNeedSetup] = useState<boolean | null>(null);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [loading, setLoading] = useState(false);
  const toast = useToast();

  useEffect(() => {
    const controller = new AbortController();
    getAppMe(controller.signal)
      .then((result) => {
        if (result.user) {
          onAuthorized(result.user);
        } else {
          setNeedSetup(!!result.setup);
        }
      })
      .catch(() => setNeedSetup(false));
    return () => controller.abort();
  }, [onAuthorized]);

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setLoading(true);
    try {
      const result = needSetup ? await appRegister(email, password, displayName) : await appLogin(email, password);
      onAuthorized(result.user);
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err), "error");
    } finally {
      setLoading(false);
    }
  }

  if (needSetup === null) {
    return <main className="auth-gate"><div className="auth-card"><p>Đang kiểm tra trạng thái đăng nhập...</p></div></main>;
  }

  return (
    <main className="auth-gate">
      <form className="auth-card" onSubmit={submit}>
        <h1>{needSetup ? "Tạo tài khoản đầu tiên" : "Đăng nhập"}</h1>
        <p>{needSetup ? "Đặt mật khẩu cho admin của Ổ Đĩa Cloud Ảo." : "Nhập email và mật khẩu của bạn để mở Drive."}</p>
        <label>
          <span>Email</span>
          <input value={email} onChange={(event) => setEmail(event.target.value)} type="email" required autoComplete="email" />
        </label>
        {needSetup && (
          <label>
            <span>Tên hiển thị (tùy chọn)</span>
            <input value={displayName} onChange={(event) => setDisplayName(event.target.value)} />
          </label>
        )}
        <label>
          <span>Mật khẩu</span>
          <input value={password} onChange={(event) => setPassword(event.target.value)} type="password" required autoComplete={needSetup ? "new-password" : "current-password"} />
        </label>
        <button className="button button--primary" disabled={loading}>{needSetup ? "Tạo tài khoản" : "Đăng nhập"}</button>
      </form>
    </main>
  );
}
