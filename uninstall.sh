#!/usr/bin/env bash
# =============================================================================
# Applad — uninstaller
#
# One-liner:
#   curl -fsSL https://raw.githubusercontent.com/mittolabs/applad/main/uninstall.sh | bash
#
# Local (from repo clone or install dir):
#   ./uninstall.sh
# =============================================================================

set -euo pipefail
IFS=$'\n\t'

COMPOSE_FILE="docker-compose.yml"

# ── Colours ───────────────────────────────────────────────────────────────────
if [ -t 1 ]; then
  BOLD='\033[1m'; DIM='\033[2m'; RESET='\033[0m'
  RED='\033[31m'; GREEN='\033[32m'; YELLOW='\033[33m'
  BLUE='\033[34m'; CYAN='\033[36m'; WHITE='\033[97m'
else
  BOLD=''; DIM=''; RESET=''
  RED=''; GREEN=''; YELLOW=''
  BLUE=''; CYAN=''; WHITE=''
fi

# ── Helpers ───────────────────────────────────────────────────────────────────
banner() {
  printf '\n'
  printf "${BOLD}${WHITE}"
  printf '  ▄▄▄       ██▓███   ██▓███   ██▓    ▄▄▄      ▓█████▄ \n'
  printf ' ▒████▄    ▓██░  ██▒▓██░  ██▒▓██▒   ▒████▄    ▒██▀ ██▌\n'
  printf ' ▒██  ▀█▄  ▓██░ ██▓▒▓██░ ██▓▒▒██░   ▒██  ▀█▄  ░██   █▌\n'
  printf ' ░██▄▄▄▄██ ▒██▄█▓▒ ▒▒██▄█▓▒ ▒▒██░   ░██▄▄▄▄██ ░▓█▄   ▌\n'
  printf '  ▓█   ▓██▒▒██▒ ░  ░▒██▒ ░  ░░██████▒▓█   ▓██▒░▒████▓ \n'
  printf '  ▒▒   ▓▒█░▒▓▒░ ░  ░▒▓▒░ ░  ░░ ▒░▓  ░▒▒   ▓▒█░ ▒▒▓  ▒ \n'
  printf "${RESET}\n"
  printf "  ${DIM}Uninstaller${RESET}\n"
  printf '\n'
}

log()     { printf "  ${GREEN}✓${RESET}  %s\n" "$*"; }
info()    { printf "  ${CYAN}→${RESET}  %s\n" "$*"; }
warn()    { printf "  ${YELLOW}!${RESET}  %s\n" "$*"; }
err()     { printf "  ${RED}✗${RESET}  %s\n" "$*" >&2; }
die()     { err "$*"; exit 1; }
section() { printf '\n'; printf "  ${BOLD}${BLUE}%s${RESET}\n" "$*"; printf "  ${DIM}──────────────────────────────────────────────────${RESET}\n"; }

ask_yn() {
  local prompt="$1" default="${2:-n}" ans
  local hint; [ "$default" = "y" ] && hint="Y/n" || hint="y/N"
  printf "  ${WHITE}%s${RESET} ${DIM}[%s]${RESET} " "$prompt" "$hint" >&2
  read -r ans </dev/tty
  ans="${ans:-$default}"
  [[ "$ans" =~ ^[Yy] ]]
}

ask() {
  local prompt="$1" default="${2:-}" var
  if [ -n "$default" ]; then
    printf "  ${WHITE}%s${RESET} ${DIM}[%s]${RESET} " "$prompt" "$default" >&2
  else
    printf "  ${WHITE}%s${RESET} " "$prompt" >&2
  fi
  read -r var </dev/tty
  printf '%s' "${var:-$default}"
}

# ── Entry point ───────────────────────────────────────────────────────────────
banner

# ═════════════════════════════════════════════════════════════════════════════
# 1. Find install directory
# ═════════════════════════════════════════════════════════════════════════════
section "Locate installation"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-./uninstall.sh}")" 2>/dev/null && pwd || pwd)"

if [ -f "$SCRIPT_DIR/$COMPOSE_FILE" ]; then
  INSTALL_DIR="$SCRIPT_DIR"
  info "Found installation at: $INSTALL_DIR"
else
  INSTALL_DIR="$(ask "Install directory:" "/opt/applad")"
  [ -f "$INSTALL_DIR/$COMPOSE_FILE" ] || die "No $COMPOSE_FILE found in $INSTALL_DIR. Is Applad installed there?"
fi

cd "$INSTALL_DIR"

# ═════════════════════════════════════════════════════════════════════════════
# 2. Confirm intent
# ═════════════════════════════════════════════════════════════════════════════
section "Confirm uninstall"

printf "  ${YELLOW}This will remove Applad from this server.${RESET}\n"
printf "  ${DIM}Install directory: %s${RESET}\n\n" "$INSTALL_DIR"

ask_yn "Continue?" "n" || { printf '\n'; info "Aborted."; printf '\n'; exit 0; }

# ═════════════════════════════════════════════════════════════════════════════
# 3. Stop and remove containers
# ═════════════════════════════════════════════════════════════════════════════
section "Stopping services"

if docker compose -f "$COMPOSE_FILE" ps -q 2>/dev/null | grep -q .; then
  info "Stopping and removing containers…"
  docker compose -f "$COMPOSE_FILE" down --remove-orphans
  log "Containers stopped and removed"
else
  info "No running containers found"
fi

# ═════════════════════════════════════════════════════════════════════════════
# 4. Remove images (optional)
# ═════════════════════════════════════════════════════════════════════════════
section "Docker images"

printf "  ${DIM}Removing images frees ~2–3 GB of disk space.${RESET}\n\n"

if ask_yn "Remove Applad Docker images?" "n"; then
  info "Removing images…"
  docker compose -f "$COMPOSE_FILE" down --rmi all 2>/dev/null || true
  # Also remove any dangling applad images not caught by compose
  docker images --format '{{.Repository}}:{{.Tag}}' | grep 'mittolabs/applad' | xargs -r docker rmi --force 2>/dev/null || true
  log "Images removed"
else
  info "Skipping image removal"
fi

# ═════════════════════════════════════════════════════════════════════════════
# 5. Remove volumes / data (destructive — explicit confirmation required)
# ═════════════════════════════════════════════════════════════════════════════
section "Data volumes"

printf "  ${RED}${BOLD}WARNING: Deleting volumes permanently destroys all data.${RESET}\n"
printf "  ${DIM}This includes your PostgreSQL database, uploaded files, and Redis data.${RESET}\n"
printf "  ${DIM}This cannot be undone.${RESET}\n\n"

if ask_yn "Delete all data volumes?" "n"; then
  printf "  ${RED}Are you absolutely sure? Type${RESET} ${BOLD}yes${RESET} ${RED}to confirm:${RESET} " >&2
  read -r confirm </dev/tty
  if [ "$confirm" = "yes" ]; then
    info "Removing volumes…"
    docker compose -f "$COMPOSE_FILE" down -v 2>/dev/null || true
    # Remove any named volumes that may have been left behind
    docker volume ls --format '{{.Name}}' | grep -E 'applad|postgres_data|redis_data|storage_data|ssl_certs|clamav_data' | xargs -r docker volume rm 2>/dev/null || true
    log "Volumes deleted"
  else
    info "Volume deletion cancelled"
  fi
else
  info "Keeping data volumes (data is safe)"
fi

# ═════════════════════════════════════════════════════════════════════════════
# 6. Remove install directory (optional)
# ═════════════════════════════════════════════════════════════════════════════
section "Install directory"

printf "  ${DIM}Directory: %s${RESET}\n\n" "$INSTALL_DIR"

if ask_yn "Delete install directory and all its files?" "n"; then
  # Don't delete if we're inside the repo itself
  if [ "$INSTALL_DIR" = "$(pwd)" ] && [ -f "$INSTALL_DIR/.git/HEAD" ]; then
    warn "Install directory appears to be a git repo — skipping deletion"
  else
    rm -rf "$INSTALL_DIR"
    log "Deleted $INSTALL_DIR"
  fi
else
  info "Keeping install directory"
fi

# ═════════════════════════════════════════════════════════════════════════════
# 7. Done
# ═════════════════════════════════════════════════════════════════════════════
printf '\n'
printf "  ${BOLD}${GREEN}╔══════════════════════════════════════════════════╗${RESET}\n"
printf "  ${BOLD}${GREEN}║        Applad has been uninstalled.              ║${RESET}\n"
printf "  ${BOLD}${GREEN}╚══════════════════════════════════════════════════╝${RESET}\n"
printf '\n'
printf "  ${DIM}To reinstall at any time:${RESET}\n"
printf "  ${CYAN}curl -fsSL https://raw.githubusercontent.com/mittolabs/applad/main/install.sh | bash${RESET}\n"
printf '\n'
