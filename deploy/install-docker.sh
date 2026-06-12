#!/usr/bin/env bash
# =============================================================================
# Ổ Đĩa Cloud Ảo (Telegram Drive) — One-click Docker installer
#
#   curl -fsSL https://raw.githubusercontent.com/ptadigi/Telegram-Drive-Mount/main/deploy/install-docker.sh | bash
#
# Clones the repo (or uses the current checkout), writes a config, and brings up
# the stack with docker compose. Good for any host with Docker — including
# cPanel/Plesk/Coolify boxes that support Docker.
# =============================================================================
set -euo pipefail

REPO_URL="https://github.com/ptadigi/Telegram-Drive-Mount.git"
WORKDIR="${TD_DIR:-$HOME/telegram-drive}"
PORT="${TD_PORT:-8750}"

c_green() { printf '\033[1;32m%s\033[0m\n' "$1"; }
c_blue()  { printf '\033[1;34m%s\033[0m\n' "$1"; }
c_warn()  { printf '\033[1;33m%s\033[0m\n' "$1"; }
c_err()   { printf '\033[1;31m%s\033[0m\n' "$1" >&2; }

if ! command -v docker >/dev/null 2>&1; then
  c_err "Chưa có Docker. Cài Docker trước: https://docs.docker.com/engine/install/"
  exit 1
fi
if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  c_err "Chưa có docker compose plugin. Cài 'docker compose' rồi chạy lại."
  exit 1
fi

# Get the source: use current dir if it's the repo, else clone.
if [ -f "deploy/docker-compose.yml" ]; then
  WORKDIR="$(pwd)"
  c_blue "==> Dùng mã nguồn hiện tại: ${WORKDIR}"
elif [ -d "${WORKDIR}/.git" ]; then
  c_blue "==> Cập nhật mã nguồn tại ${WORKDIR}"
  git -C "$WORKDIR" pull --ff-only || c_warn "   Không pull được, dùng bản hiện có."
else
  c_blue "==> Clone mã nguồn về ${WORKDIR}"
  git clone --depth 1 "$REPO_URL" "$WORKDIR"
fi
cd "$WORKDIR"

# Config
CONFIG_SRC="deploy/config/config.example.json"
CONFIG_DST="deploy/config/config.json"
if [ ! -f "$CONFIG_DST" ]; then
  cp "$CONFIG_SRC" "$CONFIG_DST"
  if [ -t 0 ]; then
    echo "Lấy api_id/api_hash tại https://my.telegram.org"
    read -rp "  Telegram api_id: " api_id || true
    read -rp "  Telegram api_hash: " api_hash || true
    read -rp "  Mật khẩu Basic Auth (Enter để bỏ qua): " admin_pass || true
    [ -n "${api_id:-}" ]   && sed -i "s/\"api_id\": 0/\"api_id\": ${api_id}/" "$CONFIG_DST"
    [ -n "${api_hash:-}" ] && sed -i "s/\"api_hash\": \"\"/\"api_hash\": \"${api_hash}\"/" "$CONFIG_DST"
    [ -n "${admin_pass:-}" ] && sed -i "s/doi-mat-khau-cua-ban/${admin_pass}/" "$CONFIG_DST"
  else
    c_warn "==> Non-interactive: sửa ${CONFIG_DST} để điền api_id/api_hash."
  fi
  c_green "   Đã tạo ${CONFIG_DST}"
else
  c_green "==> Giữ cấu hình hiện có: ${CONFIG_DST}"
fi

c_blue "==> Khởi động stack với ${COMPOSE} ..."
$COMPOSE -f deploy/docker-compose.yml up -d

ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
c_green ""
c_green "Hoàn tất!"
echo "  • API/Agent: http://${ip:-<ip>}:${PORT}"
echo "  • PWA:       http://${ip:-<ip>}:5173"
echo "  • Log:       ${COMPOSE} -f deploy/docker-compose.yml logs -f"
echo
echo "Mở PWA, tạo tài khoản quản trị rồi đăng nhập Telegram. Đặt domain+HTTPS"
echo "qua reverse proxy để dùng ngoài internet — xem docs/INSTALL.md."
