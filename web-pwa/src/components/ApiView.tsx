import { Copy, KeyRound, Plus, RefreshCw, Trash2 } from "../icons";
import { useEffect, useState } from "react";
import { ApiToken, createApiToken, listApiTokens, revokeApiToken } from "../api/agent";
import { useConfirm, useToast } from "../state/ui";

// The base URL the user should call. In the browser this is the same origin the
// PWA is served from (the user's domain), so every curl example is copy-ready.
const BASE = typeof window !== "undefined" ? window.location.origin : "";

type Endpoint = {
  method: string;
  path: string;
  desc: string;
  curl: (token: string) => string;
};

const TOKEN_PH = "<TOKEN>";

function auth(token: string) {
  const t = token || TOKEN_PH;
  return `-H "Authorization: Device ${t}"`;
}

const GROUPS: { title: string; items: Endpoint[] }[] = [
  {
    title: "Thông tin & thống kê",
    items: [
      { method: "GET", path: "/health", desc: "Kiểm tra agent sống (không cần token).", curl: () => `curl ${BASE}/health` },
      { method: "GET", path: "/v1/stats", desc: "Số tệp, thư mục, tổng dung lượng.", curl: (t) => `curl ${auth(t)} ${BASE}/v1/stats` },
      { method: "GET", path: "/v1/transfers", desc: "Danh sách tác vụ đồng bộ Telegram.", curl: (t) => `curl ${auth(t)} ${BASE}/v1/transfers` },
    ],
  },
  {
    title: "Duyệt file & thư mục",
    items: [
      { method: "GET", path: "/v1/drive/contents", desc: "Nội dung 1 thư mục. ?folder_id= rỗng = gốc.", curl: (t) => `curl ${auth(t)} "${BASE}/v1/drive/contents?folder_id="` },
      { method: "GET", path: "/v1/files", desc: "Liệt kê file (phân trang, lọc, sắp xếp).", curl: (t) => `curl ${auth(t)} "${BASE}/v1/files?page=1&limit=50"` },
      { method: "GET", path: "/v1/search", desc: "Tìm file/thư mục theo tên.", curl: (t) => `curl ${auth(t)} "${BASE}/v1/search?q=hop-dong"` },
      { method: "GET", path: "/v1/starred", desc: "Các mục đã đánh dấu sao.", curl: (t) => `curl ${auth(t)} ${BASE}/v1/starred` },
      { method: "POST", path: "/v1/folders", desc: "Tạo thư mục mới.", curl: (t) => `curl -X POST ${auth(t)} -H "Content-Type: application/json" \\\n  -d '{"name":"Thư mục mới","parent_id":""}' \\\n  ${BASE}/v1/folders` },
    ],
  },
  {
    title: "Tải lên & tải xuống",
    items: [
      { method: "POST", path: "/v1/files/upload", desc: "Upload file (multipart). folder_id rỗng = gốc.", curl: (t) => `curl -X POST ${auth(t)} \\\n  -F "file=@/duong-dan/file.pdf" \\\n  -F "folder_id=" \\\n  ${BASE}/v1/files/upload` },
      { method: "GET", path: "/v1/files/download", desc: "Tải file về theo id.", curl: (t) => `curl ${auth(t)} -OJ "${BASE}/v1/files/download?id=<FILE_ID>"` },
      { method: "GET", path: "/v1/files/stream", desc: "Stream file (hỗ trợ Range, tua video).", curl: (t) => `curl ${auth(t)} "${BASE}/v1/files/stream?id=<FILE_ID>"` },
      { method: "GET", path: "/v1/files/thumbnail", desc: "Ảnh thu nhỏ theo id.", curl: (t) => `curl ${auth(t)} -o thumb.jpg "${BASE}/v1/files/thumbnail?id=<FILE_ID>"` },
    ],
  },
  {
    title: "Thao tác file",
    items: [
      { method: "PUT", path: "/v1/files/rename", desc: "Đổi tên file.", curl: (t) => `curl -X PUT ${auth(t)} -H "Content-Type: application/json" \\\n  -d '{"id":"<FILE_ID>","name":"ten-moi.pdf"}' \\\n  ${BASE}/v1/files/rename` },
      { method: "PUT", path: "/v1/files/move", desc: "Di chuyển file sang thư mục khác.", curl: (t) => `curl -X PUT ${auth(t)} -H "Content-Type: application/json" \\\n  -d '{"id":"<FILE_ID>","new_parent_id":"<FOLDER_ID>"}' \\\n  ${BASE}/v1/files/move` },
      { method: "PUT", path: "/v1/files/star", desc: "Đánh dấu sao / bỏ sao.", curl: (t) => `curl -X PUT ${auth(t)} -H "Content-Type: application/json" \\\n  -d '{"id":"<FILE_ID>","starred":true}' \\\n  ${BASE}/v1/files/star` },
      { method: "POST", path: "/v1/files/trash", desc: "Đưa file vào thùng rác.", curl: (t) => `curl -X POST ${auth(t)} -H "Content-Type: application/json" \\\n  -d '{"id":"<FILE_ID>"}' \\\n  ${BASE}/v1/files/trash` },
    ],
  },
  {
    title: "Chia sẻ link",
    items: [
      { method: "GET", path: "/v1/shares", desc: "Liệt kê link chia sẻ.", curl: (t) => `curl ${auth(t)} "${BASE}/v1/shares?target_kind=file&target_id=<FILE_ID>"` },
      { method: "POST", path: "/v1/shares", desc: "Tạo link (mật khẩu/hết hạn/giới hạn tùy chọn).", curl: (t) => `curl -X POST ${auth(t)} -H "Content-Type: application/json" \\\n  -d '{"target_kind":"file","target_id":"<FILE_ID>","password":"","expires_in":0,"max_downloads":0}' \\\n  ${BASE}/v1/shares` },
      { method: "DELETE", path: "/v1/shares", desc: "Xóa link chia sẻ.", curl: (t) => `curl -X DELETE ${auth(t)} "${BASE}/v1/shares?id=<SHARE_ID>"` },
    ],
  },
];

export function ApiView() {
  const [tokens, setTokens] = useState<ApiToken[]>([]);
  const [newName, setNewName] = useState("n8n");
  const [created, setCreated] = useState<string | null>(null);
  const [active, setActive] = useState(""); // token typed/created, used to fill curl
  const [loading, setLoading] = useState(false);
  const toast = useToast();
  const confirm = useConfirm();

  async function refresh() {
    try { setTokens((await listApiTokens()).tokens); } catch { /* ignore */ }
  }
  useEffect(() => { refresh(); }, []);

  async function create() {
    setLoading(true);
    try {
      const res = await createApiToken(newName || "API token");
      setCreated(res.token);
      setActive(res.token);
      await refresh();
      toast("Đã tạo token. Hãy sao chép ngay, token chỉ hiện 1 lần!", "success");
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err), "error");
    } finally { setLoading(false); }
  }

  async function revoke(id: string) {
    const ok = await confirm({ title: "Thu hồi token", message: "Token này sẽ ngừng hoạt động ngay. Tiếp tục?", tone: "error" });
    if (!ok) return;
    try { await revokeApiToken(id); await refresh(); toast("Đã thu hồi token", "success"); }
    catch (err) { toast(err instanceof Error ? err.message : String(err), "error"); }
  }

  function copy(text: string, label = "Đã sao chép") {
    navigator.clipboard.writeText(text).then(() => toast(label, "success")).catch(() => window.prompt("Sao chép:", text));
  }

  return (
    <section className="api-view">
      <header className="api-view__header">
        <div>
          <h2>API & Tích hợp (N8N)</h2>
          <p>Tạo token để gọi REST API từ N8N, script hay app khác. Mọi lệnh dưới đây dùng đúng domain: <code>{BASE}</code></p>
        </div>
        <button className="button button--ghost" onClick={refresh}><RefreshCw size={15} /> Làm mới</button>
      </header>

      <div className="api-card">
        <h3><KeyRound size={16} /> Token truy cập</h3>
        <div className="api-token-create">
          <input value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="Tên token (vd: n8n)" />
          <button className="button button--primary" onClick={create} disabled={loading}><Plus size={15} /> Tạo token</button>
        </div>
        {created && (
          <div className="api-token-new">
            <span>Token mới (chỉ hiện 1 lần):</span>
            <code>{created}</code>
            <button className="icon-button" onClick={() => copy(created, "Đã sao chép token")} aria-label="Sao chép"><Copy size={15} /></button>
          </div>
        )}
        {tokens.length > 0 && (
          <ul className="api-token-list">
            {tokens.map((t) => (
              <li key={t.id}>
                <div><strong>{t.name}</strong><span>Tạo {new Date(t.created_at * 1000).toLocaleString("vi-VN")}</span></div>
                <button className="button button--ghost" onClick={() => revoke(t.id)}><Trash2 size={14} /> Thu hồi</button>
              </li>
            ))}
          </ul>
        )}
        <p className="api-hint">⚠️ Token = chìa khóa toàn quyền tài khoản của bạn. Giữ kín, chỉ dán vào nơi tin cậy, có thể thu hồi bất cứ lúc nào.</p>
      </div>

      <div className="api-card">
        <h3>Dùng với N8N</h3>
        <ol className="api-n8n">
          <li>Thêm node <strong>HTTP Request</strong>.</li>
          <li>Authentication → <strong>Generic Credential → Header Auth</strong>.</li>
          <li>Name: <code>Authorization</code> — Value: <code>Device &lt;token&gt;</code></li>
          <li>URL: ghép <code>{BASE}</code> + đường dẫn endpoint bên dưới.</li>
        </ol>
        <div className="api-fill">
          <label>Dán token để các lệnh cURL hiện sẵn token:</label>
          <input value={active} onChange={(e) => setActive(e.target.value)} placeholder="Dán token vào đây (không lưu lên server)" />
        </div>
      </div>

      {GROUPS.map((g) => (
        <div className="api-card" key={g.title}>
          <h3>{g.title}</h3>
          {g.items.map((ep) => (
            <div className="api-endpoint" key={ep.method + ep.path}>
              <div className="api-endpoint__head">
                <span className={`api-method api-method--${ep.method.toLowerCase()}`}>{ep.method}</span>
                <code className="api-path">{ep.path}</code>
              </div>
              <p className="api-endpoint__desc">{ep.desc}</p>
              <div className="api-curl">
                <pre>{ep.curl(active)}</pre>
                <button className="icon-button" onClick={() => copy(ep.curl(active), "Đã sao chép lệnh cURL")} aria-label="Sao chép cURL"><Copy size={15} /></button>
              </div>
            </div>
          ))}
        </div>
      ))}
    </section>
  );
}
