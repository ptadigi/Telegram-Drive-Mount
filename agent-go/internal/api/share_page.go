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
  .card:has(.preview), .card:has(.docx) { width: min(860px, 100%%); }
  .preview { max-width: 100%%; max-height: 70vh; border-radius: 16px; box-shadow: 0 12px 30px rgba(15,23,42,0.12); background: #000; }
  .preview-pdf { width: 100%%; height: 72vh; border: 0; background: #fff; }
  .docx { width: 100%%; max-height: 70vh; overflow: auto; text-align: left; background: #fff; border: 1px solid rgba(15,23,42,0.08); border-radius: 16px; padding: 24px 28px; line-height: 1.6; }
  .docx img { max-width: 100%%; height: auto; }
  .docx table { border-collapse: collapse; width: 100%%; margin: 0.8em 0; }
  .docx td, .docx th { border: 1px solid #ddd; padding: 6px 10px; }
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
  const name = (file.name || "").toLowerCase();
  const mime = (file.mime_type || "").toLowerCase();
  const kind = file.kind || "";
  const ext = name.indexOf('.') >= 0 ? name.slice(name.lastIndexOf('.')) : "";
  const rawDownload = "/share/" + encodeURIComponent(slug) + "/raw" + (currentPassword ? "?password=" + encodeURIComponent(currentPassword) : "");
  const rawInline = "/share/" + encodeURIComponent(slug) + "/raw?disposition=inline" + (currentPassword ? "&password=" + encodeURIComponent(currentPassword) : "");

  // Render an inline preview for supported types before the title/download.
  if (!isFolder) {
    const isImage = kind === 'image' || mime.indexOf('image/') === 0;
    const isVideo = kind === 'video' || mime.indexOf('video/') === 0;
    const isAudio = kind === 'audio' || mime.indexOf('audio/') === 0;
    const isPdf = ext === '.pdf' || mime === 'application/pdf';
    if (isImage) {
      const img = document.createElement('img'); img.className = 'preview'; img.src = rawInline; img.alt = file.name || ''; status.appendChild(img);
    } else if (isVideo) {
      const v = document.createElement('video'); v.className = 'preview'; v.src = rawInline; v.controls = true; v.playsInline = true; status.appendChild(v);
    } else if (isAudio) {
      const a = document.createElement('audio'); a.src = rawInline; a.controls = true; a.style.width = '100%%'; status.appendChild(a);
    } else if (isPdf) {
      const f = document.createElement('iframe'); f.className = 'preview preview-pdf'; f.src = rawInline; f.title = file.name || ''; status.appendChild(f);
    } else if (ext === '.docx') {
      renderDocx(rawInline);
    } else {
      const icon = document.createElement('div'); icon.className = 'icon'; icon.textContent = '📄'; status.appendChild(icon);
    }
  } else {
    const icon = document.createElement('div'); icon.className = 'icon'; icon.textContent = '📁'; status.appendChild(icon);
  }

  const title = document.createElement('h1'); title.textContent = isFolder ? 'Thư mục chia sẻ' : (file.name || 'File chia sẻ'); status.appendChild(title);
  if (!isFolder && file.size) {
    const meta = document.createElement('p'); meta.textContent = bytes(file.size) + ' · ' + (file.mime_type || file.kind || 'File'); status.appendChild(meta);
  }
  const link = document.createElement('a'); link.className = 'button'; link.textContent = isFolder ? '⬇ Tải ZIP' : '⬇ Tải xuống';
  link.href = rawDownload; link.target = '_blank'; status.appendChild(link);
  if (data.share) {
    const meta = document.createElement('p');
    const parts = [];
    if (data.share.expires_at && data.share.expires_at > 0) parts.push('Hết hạn: ' + new Date(data.share.expires_at * 1000).toLocaleString('vi-VN'));
    if (data.share.max_downloads && data.share.max_downloads > 0) parts.push('Đã tải ' + (data.share.access_count || 0) + '/' + data.share.max_downloads);
    if (parts.length) { meta.textContent = parts.join(' · '); status.appendChild(meta); }
  }
}

async function renderDocx(url) {
  const holder = document.createElement('div'); holder.className = 'docx'; holder.textContent = 'Đang mở tài liệu Word…'; status.appendChild(holder);
  try {
    const res = await fetch(url);
    if (!res.ok) throw new Error('fetch failed');
    const buf = await res.arrayBuffer();
    if (!window.mammoth) {
      await new Promise((resolve, reject) => {
        const s = document.createElement('script');
        s.src = 'https://cdn.jsdelivr.net/npm/mammoth@1.8.0/mammoth.browser.min.js';
        s.onload = resolve; s.onerror = reject; document.head.appendChild(s);
      });
    }
    const result = await window.mammoth.convertToHtml({ arrayBuffer: buf });
    holder.innerHTML = result.value || '<p>(Tài liệu trống)</p>';
  } catch (e) {
    holder.remove();
    const icon = document.createElement('div'); icon.className = 'icon'; icon.textContent = '📄'; status.insertBefore(icon, status.firstChild);
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
