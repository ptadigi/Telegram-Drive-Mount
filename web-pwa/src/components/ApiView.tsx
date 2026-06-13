import { Copy, KeyRound, Plus, RefreshCw, Trash2 } from "../icons";
import { useEffect, useMemo, useState } from "react";
import { ApiToken, createApiToken, listApiTokens, revokeApiToken } from "../api/agent";
import { useToast } from "../state/ui";

type Endpoint = {
  group: string;
  method: string;
  path: string;
  desc: string;
  // builds the curl command body (after the auth header)
  curl: (base: string, token: string) => string;
};

const TOKEN_PLACEHOLDER = "<TOKEN>";

const ENDPOINTS: Endpoint[] = [
  { group: "Cơ bản", method: "GET", path: "/v1/stats", desc: "Số liệu tệp/thư mục/dung lượng", curl: (b, t) => `curl -s "${b}/v1/stats" -H "Authorization: Device ${t}"` },
  { group: "Cơ bản", method: "GET", path: "/v1/info", desc: "Thông tin agent (version, uptime)", curl: (b, t) => `curl -s "${b}/v1/info" -H "Authorization: Device ${t}"` },
  { group: "File & thư mục", method: "GET", path: "/v1/drive/contents", desc: "Liệt kê nội dung thư mục (folder_id rỗng = gốc)", curl: (b, t) => `curl -s "${b}/v1/drive/contents?folder_id=" -H "Authorization: Device ${t}"` },
  { group: "File & thư mục", method: "GET", path: "/v1/files", desc: "Liệt kê file", curl: (b, t) => `curl -s "${b}/v1/files" -H "Authorization: Device ${t}"` },
  { group: "File & thư mục", method: "GET", path: "/v1/search", desc: "Tìm file/thư mục theo tên", curl: (b, t) => `curl -s "${b}/v1/search?q=hop-dong" -H "Authorization: Device ${t}"` },
  { group: "File & thư mục", method: "POST", path: "/v1/folders", desc: "Tạo thư mục", curl: (b, t) => `curl -s -X POST "${b}/v1/folders" -H "Authorization: Device ${t}" -H "Content-Type: application/json" -d '{"name":"Thư mục mới","parent_id":""}'` },
  { group: "Upload", method: "POST", path: "/v1/files/upload", desc: "Tải file lên (multipart, field: file)", curl: (b, t) => `curl -s -X POST "${b}/v1/files/upload" -H "Authorization: Device ${t}" -F "file=@/duong-dan/file.pdf" -F "folder_id="` },
  { group: "Download", method: "GET", path: "/v1/files/download", desc: "Tải file về theo id", curl: (b, t) => `curl -s -L "${b}/v1/files/download?id=FILE_ID" -H "Authorization: Device ${t}" -o file.bin` },
  { group: "Download", method: "GET", path: "/v1/files/stream", desc: "Stream file (hỗ trợ Range)", curl: (b, t) => `curl -s "${b}/v1/files/stream?id=FILE_ID" -H "Authorization: Device ${t}" -o stream.bin` },
  { group: "Thao tác", method: "PUT", path: "/v1/files/rename", desc: "Đổi tên file", curl: (b, t) => `curl -s -X PUT "${b}/v1/files/rename" -H "Authorization: Device ${t}" -H "Content-Type: application/json" -d '{"id":"FILE_ID","name":"ten-moi.pdf"}'` },
  { group: "Thao tác", method: "PUT", path: "/v1/files/move", desc: "Di chuyển file sang thư mục khác", curl: (b, t) => `curl -s -X PUT "${b}/v1/files/move" -H "Authorization: Device ${t}" -H "Content-Type: application/json" -d '{"id":"FILE_ID","new_parent_id":"FOLDER_ID"}'` },
  { group: "Thao tác", method: "POST", path: "/v1/files/trash", desc: "Đưa file vào thùng rác", curl: (b, t) => `curl -s -X POST "${b}/v1/files/trash" -H "Authorization: Device ${t}" -H "Content-Type: application/json" -d '{"id":"FILE_ID"}'` },
  { group: "Chia sẻ", method: "POST", path: "/v1/shares", desc: "Tạo link chia sẻ", curl: (b, t) => `curl -s -X POST "${b}/v1/shares" -H "Authorization: Device ${t}" -H "Content-Type: application/json" -d '{"target_kind":"file","target_id":"FILE_ID"}'` },
  { group: "Chia sẻ", method: "GET", path: "/v1/shares", desc: "Liệt kê link chia sẻ", curl: (b, t) => `curl -s "${b}/v1/shares" -H "Authorization: Device ${t}"` },
];

export function ApiView() {
  const toast = useToast();
  const [tokens, setTokens] = useState<ApiToken[]>([]);
  const [loading, setLoading] = useState(false);
  const [newName, setNewName] = useState("");
  const [creating, setCreating] = useState(false);
  const [freshToken, setFreshToken] = useState<string | null>(null);
  // The token typed/pasted to render runnable curl. Defaults to the freshly
  // created token; stays client-side only.
  const [activeToken, setActiveToken] = useState("");

  const base = window.location.origin;
  const effectiveToken = activeToken.trim() || TOKEN_PLACEHOLDER;

  async function refresh() {
    setLoading(true);
    try {
      setTokens((await listApiTokens()).tokens);
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err), "error");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { refresh(); }, []);

  async function handleCreate() {
    setCreating(true);
    try {
      const res = await createApiToken(newName.trim() || "n8n");
      setFreshToken(res.token);
      setActiveToken(res.token);
      setNewName("");
      toast("Đã tạo token. Hãy sao chép & lưu lại — chỉ hiện 1 lần!", "success");
      refresh();
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err), "error");
    } finally {
      setCreating(false);
    }
  }

  async function handleRevoke(id: string, name: string) {
    if (!window.confirm(`Thu hồi token "${name}"? Mọi nơi đang dùng token này sẽ mất quyền.`)) return;
    try {
      await revokeApiToken(id);
      toast("Đã thu hồi token", "success");
      refresh();
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err), "error");
    }
  }

  async function copy(text: string, label = "Đã sao chép") {
    try {
      await navigator.clipboard.writeText(text);
      toast(label, "success");
    } catch {
      window.prompt("Sao chép:", text);
    }
  }

  const grouped = useMemo(() => {
    const map = new Map<string, Endpoint[]>();
    for (const e of ENDPOINTS) {
      if (!map.has(e.group)) map.set(e.group, []);
      map.get(e.group)!.push(e);
    }
    return [...map.entries()];
  }, []);

  return (
    <section className="api-view">
      <header className="api-view__header">
        <div>
          <h2><KeyRound size={20} /> API & Automation</h2>
          <p>REST API cho N8N, script, hay tích hợp khác. Xác thực bằng header <code>Authorization: Device &lt;token&gt;</code>.</p>
        </div>
      </header>

      <div className="api-base">
        <span>Base URL</span>
        <code>{base}</code>
        <button className="icon-button" title="Sao chép" onClick={() => copy(base)}><Copy size={15} /></button>
      </div>

      {/* Token management */}
      <div className="api-card">
        <div className="api-card__head">
          <strong>Token truy cập</strong>
          <button className="icon-button" title="Làm mới" onClick={refresh} disabled={loading}><RefreshCw size={15} /></button>
        </div>
        <div className="api-create">
          <input value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="Tên token (vd: n8n)" />
          <button className="button button--primary" onClick={handleCreate} disabled={creating}><Plus size={15} /> {creating ? "Đang tạo..." : "Tạo token"}</button>
        </div>
        {freshToken && (
          <div className="api-fresh">
            <span>⚠️ Token chỉ hiện 1 lần. Hãy sao chép & lưu lại an toàn:</span>
            <div className="api-fresh__token">
              <code>{freshToken}</code>
              <button className="icon-button" title="Sao chép" onClick={() => copy(freshToken, "Đã sao chép token")}><Copy size={15} /></button>
            </div>
          </div>
        )}
        {tokens.length === 0 ? (
          <div className="muted-box">Chưa có token nào. Tạo một cái để dùng cho N8N/script.</div>
        ) : (
          <ul className="api-token-list">
            {tokens.map((t) => (
              <li key={t.id}>
                <div><strong>{t.name}</strong><span>Tạo {new Date(t.created_at * 1000).toLocaleString("vi-VN")}</span></div>
                <button className="button button--ghost" onClick={() => handleRevoke(t.id, t.name)}><Trash2 size={14} /> Thu hồi</button>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* N8N guide */}
      <div className="api-card">
        <strong>Dùng với N8N</strong>
        <ol className="api-n8n">
          <li>Thêm node <b>HTTP Request</b>.</li>
          <li>Authentication → <b>Generic Credential Type</b> → <b>Header Auth</b>.</li>
          <li>Name: <code>Authorization</code> — Value: <code>Device &lt;token&gt;</code> (dán token ở trên).</li>
          <li>URL: ghép từ Base URL + endpoint bên dưới. Xong!</li>
        </ol>
      </div>

      {/* Endpoint catalog with copy-ready curl */}
      <div className="api-card">
        <div className="api-card__head">
          <strong>Danh sách endpoint (cURL ăn-ngay)</strong>
        </div>
        <div className="api-token-input">
          <input value={activeToken} onChange={(e) => setActiveToken(e.target.value)} placeholder="Dán token vào đây để cURL hiển thị token thật (chỉ ở máy bạn)" />
        </div>
        {grouped.map(([group, items]) => (
          <div className="api-group" key={group}>
            <h3>{group}</h3>
            {items.map((e) => {
              const cmd = e.curl(base, effectiveToken);
              return (
                <div className="api-endpoint" key={e.method + e.path}>
                  <div className="api-endpoint__head">
                    <span className={`api-method api-method--${e.method.toLowerCase()}`}>{e.method}</span>
                    <code className="api-endpoint__path">{e.path}</code>
                    <span className="api-endpoint__desc">{e.desc}</span>
                  </div>
                  <div className="api-curl">
                    <pre>{cmd}</pre>
                    <button className="icon-button" title="Sao chép cURL" onClick={() => copy(cmd, "Đã sao chép lệnh cURL")}><Copy size={15} /></button>
                  </div>
                </div>
              );
            })}
          </div>
        ))}
      </div>

      <p className="api-note">⚠️ Token có toàn quyền với dữ liệu tài khoản của bạn — giữ kín, có thể thu hồi bất cứ lúc nào. Ghi/đọc tần suất cao có thể bị Telegram giới hạn (FLOOD_WAIT); file lớn nên dùng tus <code>/v1/tus/</code>.</p>
    </section>
  );
}
