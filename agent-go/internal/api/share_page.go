package api

import (
	"encoding/json"
	"fmt"
)

func renderSharePageHTML(slug string) string {
	// JSON-encode the slug for safe embedding in a JS string literal.
	// We deliberately do NOT html.EscapeString here because that would
	// encode characters like & or ' as HTML entities and break the JS
	// runtime usage of the slug (e.g. when fetching /share/<slug>).
	encoded, err := json.Marshal(slug)
	if err != nil {
		encoded = []byte(`""`)
	}
	jsSlug := string(encoded)
	return fmt.Sprintf(`<!doctype html>
<html lang="vi">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Ổ Đĩa Cloud Ảo · Link chia sẻ</title>
<style>
  :root { color-scheme: light; }
  body { margin: 0; font-family: "Be Vietnam Pro", "Segoe UI", sans-serif; background: linear-gradient(135deg, #f4f9ff 0%%, #eaf6f7 100%%); color: #172033; min-height: 100vh; display: grid; grid-template-rows: auto 1fr auto; }
  header { display: flex; align-items: center; gap: 8px; padding: 22px 24px; color: #075985; font-weight: 600; }
  main { display: grid; place-items: center; padding: 24px; }
  .card { width: min(440px, 100%%); padding: 32px; background: #fff; border-radius: 24px; box-shadow: 0 24px 60px rgba(15, 23, 42, 0.1); display: grid; gap: 14px; text-align: center; }
  .card h1 { margin: 0; font-size: 22px; }
  .card p { margin: 0; color: #6b7a90; }
  .icon { display: grid; place-items: center; width: 70px; height: 70px; border-radius: 22px; background: #eef4ff; color: #0b66ef; margin: 0 auto; font-size: 28px; }
  input { width: 100%%; max-width: 280px; border: 1px solid rgba(15,23,42,0.12); border-radius: 14px; padding: 10px 12px; box-sizing: border-box; }
  button, a.button { border: 0; border-radius: 999px; padding: 11px 18px; background: #0b66ef; color: #fff; font-weight: 600; cursor: pointer; text-decoration: none; display: inline-flex; align-items: center; gap: 8px; }
  button.secondary { background: transparent; color: #1f2937; border: 1px solid rgba(15,23,42,0.12); }
  .error { color: #991b1b; background: #fee2e2; border: 1px solid #fecaca; padding: 10px 12px; border-radius: 14px; }
  footer { color: #6b7a90; padding: 16px; text-align: center; font-size: 13px; }
</style>
</head>
<body>
<header>☁️ Ổ Đĩa Cloud Ảo</header>
<main>
  <div class="card" id="status"><div class="icon">⏳</div><h1>Đang tải link…</h1></div>
</main>
<footer>Powered by Ổ Đĩa Cloud Ảo · Telegram làm kho lưu trữ ẩn</footer>
<script>
const slug = %s;
const status = document.getElementById('status');
let currentPassword = "";

async function fetchInfo(password) {
  const url = "/share/" + encodeURIComponent(slug) + (password ? "?password=" + encodeURIComponent(password) : "");
  const response = await fetch(url, { headers: { "Accept": "application/json" } });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(body.error || "Lỗi " + response.status);
  return body;
}

function bytes(value) {
  if (!value) return "0 B";
  const units = ["B","KB","MB","GB","TB"];
  let v = value, i = 0;
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i += 1; }
  return v.toFixed(v >= 10 ? 0 : 1) + " " + units[i];
}

function renderLocked() {
  status.innerHTML = "";
  const icon = document.createElement('div'); icon.className = 'icon'; icon.textContent = '🔒'; status.appendChild(icon);
  const title = document.createElement('h1'); title.textContent = 'Link cần mật khẩu'; status.appendChild(title);
  const p = document.createElement('p'); p.textContent = 'Vui lòng nhập mật khẩu để mở link.'; status.appendChild(p);
  const input = document.createElement('input'); input.type = 'password'; input.placeholder = 'Mật khẩu'; status.appendChild(input);
  const btn = document.createElement('button'); btn.textContent = 'Mở khóa'; btn.onclick = () => attemptUnlock(input.value); status.appendChild(btn);
  input.addEventListener('keydown', (event) => { if (event.key === 'Enter') attemptUnlock(input.value); });
}

function renderError(message) {
  status.innerHTML = "";
  const icon = document.createElement('div'); icon.className = 'icon'; icon.textContent = '❌'; status.appendChild(icon);
  const title = document.createElement('h1'); title.textContent = 'Không mở được link'; status.appendChild(title);
  const p = document.createElement('p'); p.className = 'error'; p.textContent = message; status.appendChild(p);
}

function renderFile(data) {
  status.innerHTML = "";
  const isFolder = data.share && data.share.target_kind === 'folder';
  const file = data.file || {};
  const icon = document.createElement('div'); icon.className = 'icon'; icon.textContent = isFolder ? '📁' : '📄'; status.appendChild(icon);
  const title = document.createElement('h1'); title.textContent = isFolder ? 'Thư mục chia sẻ' : (file.name || 'File chia sẻ'); status.appendChild(title);
  if (!isFolder && file.size) {
    const meta = document.createElement('p'); meta.textContent = bytes(file.size) + ' · ' + (file.mime_type || file.kind || 'File'); status.appendChild(meta);
  }
  const link = document.createElement('a'); link.className = 'button'; link.textContent = isFolder ? '⬇ Tải ZIP' : '⬇ Tải xuống';
  const rawUrl = "/share/" + encodeURIComponent(slug) + "/raw" + (currentPassword ? "?password=" + encodeURIComponent(currentPassword) : "");
  link.href = rawUrl; link.target = '_blank'; status.appendChild(link);
  if (data.share) {
    const meta = document.createElement('p');
    const parts = [];
    if (data.share.expires_at && data.share.expires_at > 0) parts.push('Hết hạn: ' + new Date(data.share.expires_at * 1000).toLocaleString('vi-VN'));
    if (data.share.max_downloads && data.share.max_downloads > 0) parts.push('Đã tải ' + (data.share.access_count || 0) + '/' + data.share.max_downloads);
    if (parts.length) { meta.textContent = parts.join(' · '); status.appendChild(meta); }
  }
}

async function attemptUnlock(password) {
  try {
    currentPassword = password;
    const data = await fetchInfo(password);
    if (data.requires_password) { renderLocked(); return; }
    renderFile(data);
  } catch (err) {
    renderError(err.message || String(err));
  }
}

(async () => {
  try {
    const data = await fetchInfo();
    if (data.requires_password) { renderLocked(); return; }
    renderFile(data);
  } catch (err) {
    renderError(err.message || String(err));
  }
})();
</script>
</body>
</html>`, jsSlug)
}
