#!/usr/bin/env bash
# =============================================================================
# Ổ Đĩa Cloud Ảo (Telegram Drive) — One-click installer for Linux servers
#
#   curl -fsSL https://raw.githubusercontent.com/ptadigi/Telegram-Drive-Mount/main/deploy/install.sh | sudo bash
#
# Works on: Ubuntu/Debian, CentOS/RHEL/AlmaLinux/Rocky, and most VPS/cPanel
# boxes that use systemd. Installs the prebuilt Linux agent, a systemd service,
# config, and (best-effort) ffmpeg + poppler for thumbnails.
#
# Re-running this script upgrades an existing install in place.
# =============================================================================
set -euo pipefail

REPO="ptadigi/Telegram-Drive-Mount"
BIN_NAME="td-agent"
INSTALL_BIN="/usr/local/bin/${BIN_NAME}"
CONFIG_DIR="/etc/td-agent"
CONFIG_FILE="${CONFIG_DIR}/config.json"
DATA_DIR="/var/lib/td-agent"
SERVICE_FILE="/etc/systemd/system/td-agent.service"
SERVICE_USER="td-agent"
PORT="${TD_PORT:-8750}"

c_green() { printf '\033[1;32m%s\033[0m\n' "$1"; }
c_blue()  { printf '\033[1;34m%s\033[0m\n' "$1"; }
c_warn()  { printf '\033[1;33m%s\033[0m\n' "$1"; }
c_err()   { printf '\033[1;31m%s\033[0m\n' "$1" >&2; }

require_root() {
  if [ "$(id -u)" -ne 0 ]; then
    c_err "Vui lòng chạy bằng root (dùng sudo)."
    exit 1
  fi
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) c_err "Kiến trúc CPU chưa hỗ trợ: $(uname -m). Hãy build từ mã nguồn."; exit 1 ;;
  esac
}

# Pick the release asset name. CI publishes td-agent-linux-amd64 today.
asset_name() {
  local arch; arch="$(detect_arch)"
  echo "td-agent-linux-${arch}"
}

install_optional_deps() {
  c_blue "==> Cài ffmpeg + poppler (cho thumbnail video/PDF, không bắt buộc)..."
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -y >/dev/null 2>&1 || true
    apt-get install -y ffmpeg poppler-utils curl ca-certificates >/dev/null 2>&1 || c_warn "   Bỏ qua: không cài được qua apt."
  elif command -v dnf >/dev/null 2>&1; then
    dnf install -y ffmpeg poppler-utils curl ca-certificates >/dev/null 2>&1 || c_warn "   Bỏ qua (ffmpeg có thể cần RPM Fusion)."
  elif command -v yum >/dev/null 2>&1; then
    yum install -y poppler-utils curl ca-certificates >/dev/null 2>&1 || true
    yum install -y ffmpeg >/dev/null 2>&1 || c_warn "   ffmpeg cần repo bổ sung (EPEL/RPM Fusion), bỏ qua."
  else
    c_warn "   Không nhận diện được trình quản lý gói; bỏ qua deps tùy chọn."
  fi
}

download_binary() {
  local asset; asset="$(asset_name)"
  local url="https://github.com/${REPO}/releases/latest/download/${asset}"
  c_blue "==> Tải agent: ${url}"
  local tmp; tmp="$(mktemp)"
  if ! curl -fsSL "$url" -o "$tmp"; then
    c_err "Không tải được binary. Kiểm tra mạng hoặc tải thủ công tại:"
    c_err "  https://github.com/${REPO}/releases/latest"
    rm -f "$tmp"; exit 1
  fi
  install -m 0755 "$tmp" "$INSTALL_BIN"
  rm -f "$tmp"
  c_green "   Đã cài ${INSTALL_BIN}"
}

create_user_dirs() {
  if ! id "$SERVICE_USER" >/dev/null 2>&1; then
    useradd --system --home "$DATA_DIR" --shell /usr/sbin/nologin "$SERVICE_USER" 2>/dev/null \
      || useradd --system --home-dir "$DATA_DIR" --shell /sbin/nologin "$SERVICE_USER"
  fi
  install -d -o "$SERVICE_USER" -g "$SERVICE_USER" "$DATA_DIR"
  install -d "$CONFIG_DIR"
}

prompt_config() {
  if [ -f "$CONFIG_FILE" ]; then
    c_green "==> Đã có cấu hình tại ${CONFIG_FILE}, giữ nguyên (chỉ nâng cấp binary)."
    return
  fi
  c_blue "==> Thiết lập cấu hình ban đầu"
  echo "    Lấy api_id và api_hash tại https://my.telegram.org (mục API development tools)."

  local api_id api_hash admin_pass
  if [ -n "${TD_API_ID:-}" ] && [ -n "${TD_API_HASH:-}" ]; then
    api_id="$TD_API_ID"; api_hash="$TD_API_HASH"
  else
    if [ -t 0 ]; then
      read -rp "    Telegram api_id: " api_id
      read -rp "    Telegram api_hash: " api_hash
    else
      api_id=0; api_hash=""
      c_warn "   Chạy non-interactive: để trống Telegram, điền sau trong ${CONFIG_FILE}."
    fi
  fi
  admin_pass="${TD_ADMIN_PASSWORD:-}"
  if [ -z "$admin_pass" ] && [ -t 0 ]; then
    read -rp "    Mật khẩu Basic Auth (Enter để bỏ qua, dùng tài khoản PWA): " admin_pass
  fi

  local auth_block
  if [ -n "$admin_pass" ]; then
    auth_block="\"auth\": { \"mode\": \"basic\", \"username\": \"admin\", \"password\": \"${admin_pass}\" }"
  else
    auth_block="\"auth\": { \"mode\": \"open\" }"
  fi

  cat > "$CONFIG_FILE" <<EOF
{
  "host": "0.0.0.0",
  "port": ${PORT},
  "data_dir": "${DATA_DIR}",
  "telegram": {
    "api_id": ${api_id:-0},
    "api_hash": "${api_hash:-}"
  },
  ${auth_block},
  "cache": {
    "mode": "smart",
    "max_bytes": 10737418240
  }
}
EOF
  chmod 0640 "$CONFIG_FILE"
  c_green "   Đã ghi ${CONFIG_FILE}"
}

ensure_session_key() {
  # AES-256 key so the Telegram session is encrypted at rest. Stored in the
  # service environment file so it survives restarts.
  local env_file="${CONFIG_DIR}/td-agent.env"
  if [ ! -f "$env_file" ]; then
    local key; key="$(head -c 32 /dev/urandom | xxd -p -c 64 2>/dev/null || openssl rand -hex 32)"
    echo "TD_AGENT_SESSION_KEY=${key}" > "$env_file"
    chmod 0600 "$env_file"
    c_green "   Đã sinh khóa mã hóa session: ${env_file}"
  fi
}

write_service() {
  cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=Telegram Virtual Drive Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=-${CONFIG_DIR}/td-agent.env
ExecStart=${INSTALL_BIN} --config ${CONFIG_FILE} --data-dir ${DATA_DIR} --addr 0.0.0.0:${PORT}
Restart=on-failure
RestartSec=5
User=${SERVICE_USER}
Group=${SERVICE_USER}
StateDirectory=td-agent
ConfigurationDirectory=td-agent
WorkingDirectory=${DATA_DIR}
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ReadWritePaths=${DATA_DIR}
ProtectHome=yes
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectControlGroups=yes
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6
LockPersonality=yes

[Install]
WantedBy=multi-user.target
EOF
  c_green "   Đã ghi ${SERVICE_FILE}"
}

start_service() {
  systemctl daemon-reload
  systemctl enable --now td-agent
  sleep 2
  if systemctl is-active --quiet td-agent; then
    c_green "==> Dịch vụ td-agent đang chạy."
  else
    c_warn "==> Dịch vụ chưa chạy. Xem log: journalctl -u td-agent -e"
  fi
}

main() {
  require_root
  c_blue "================================================================"
  c_blue "  Cài đặt Ổ Đĩa Cloud Ảo (Telegram Drive) — Linux one-click"
  c_blue "================================================================"
  install_optional_deps
  download_binary
  create_user_dirs
  prompt_config
  ensure_session_key
  write_service
  start_service

  local ip; ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  echo
  c_green "Hoàn tất!"
  echo "  • API/PWA:   http://${ip:-<ip-vps>}:${PORT}"
  echo "  • Cấu hình:  ${CONFIG_FILE}"
  echo "  • Dữ liệu:   ${DATA_DIR}"
  echo "  • Log:       journalctl -u td-agent -f"
  echo
  echo "Bước tiếp theo:"
  echo "  1) Mở http://${ip:-<ip-vps>}:${PORT} trên trình duyệt."
  echo "  2) Tạo tài khoản quản trị, rồi đăng nhập Telegram bằng QR/điện thoại."
  echo "  3) (Khuyến nghị) Đặt domain + HTTPS qua reverse proxy — xem docs/INSTALL.md."
}

main "$@"
