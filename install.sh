#!/usr/bin/env bash
# =============================================================================
# Applad — self-hosted installer
#
# One-liner (fresh server):
#   curl -fsSL https://raw.githubusercontent.com/mittolabs/applad/main/install.sh | bash
#
# Local (from repo clone):
#   ./install.sh
#
# Upgrade existing install:
#   ./install.sh upgrade
#   curl -fsSL .../install.sh | bash -s upgrade
# =============================================================================

set -euo pipefail
IFS=$'\n\t'

APPLAD_VERSION="${APPLAD_VERSION:-latest}"
GITHUB_ORG="mittolabs"
GITHUB_REPO="applad"
RELEASE_BASE="https://raw.githubusercontent.com/${GITHUB_ORG}/${GITHUB_REPO}/main"

# Files we need in the install directory
COMPOSE_FILE="docker-compose.yml"        # the one compose: builds locally, pulls on install
CADDYFILE="docker/caddy/Caddyfile"         # generated here; the compose mounts this path
INIT_SQL="docker/postgres/init.sql"

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
  printf "  ${DIM}Self-hosted backend-as-a-service${RESET}\n"
  printf '\n'
}

log()     { printf "  ${GREEN}✓${RESET}  %s\n" "$*"; }
info()    { printf "  ${CYAN}→${RESET}  %s\n" "$*"; }
warn()    { printf "  ${YELLOW}!${RESET}  %s\n" "$*"; }
err()     { printf "  ${RED}✗${RESET}  %s\n" "$*" >&2; }
die()     { err "$*"; exit 1; }
section() { printf '\n'; printf "  ${BOLD}${BLUE}%s${RESET}\n" "$*"; printf "  ${DIM}──────────────────────────────────────────────────${RESET}\n"; }

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

ask_secret() {
  local prompt="$1" var
  printf "  ${WHITE}%s${RESET} " "$prompt" >&2
  read -rs var </dev/tty
  printf '\n' >&2
  printf '%s' "$var"
}

ask_yn() {
  local prompt="$1" default="${2:-y}" ans
  local hint; [ "$default" = "y" ] && hint="Y/n" || hint="y/N"
  printf "  ${WHITE}%s${RESET} ${DIM}[%s]${RESET} " "$prompt" "$hint" >&2
  read -r ans </dev/tty
  ans="${ans:-$default}"
  [[ "$ans" =~ ^[Yy] ]]
}

ask_choice() {
  local prompt="$1"; shift
  local -a options=("$@")
  printf "  ${WHITE}%s${RESET}\n" "$prompt" >&2
  for i in "${!options[@]}"; do
    printf "    ${DIM}%d.${RESET} %s\n" "$((i+1))" "${options[$i]}" >&2
  done
  printf "  ${WHITE}Choice:${RESET} " >&2
  local choice
  read -r choice </dev/tty
  printf '%d' "$((choice - 1))"
}

gen_secret() {
  openssl rand -hex "${1:-32}"
}

fetch() {
  local url="$1" dest="$2"
  if command -v curl &>/dev/null; then
    curl -fsSL "$url" -o "$dest"
  elif command -v wget &>/dev/null; then
    wget -qO "$dest" "$url"
  else
    die "Neither curl nor wget found. Install one and retry."
  fi
}

# ── Entry point ───────────────────────────────────────────────────────────────
CMD="${1:-install}"
case "$CMD" in
  upgrade) MODE=upgrade ;;
  install) MODE=install ;;
  *) die "Unknown command '$CMD'. Use: install | upgrade" ;;
esac

banner

# ═════════════════════════════════════════════════════════════════════════════
# 1. Prerequisites
# ═════════════════════════════════════════════════════════════════════════════
section "Checking prerequisites"

MISSING=0

if ! command -v docker &>/dev/null; then
  err "Docker not found"
  info "Install: https://docs.docker.com/get-docker/"
  MISSING=1
else
  log "Docker $(docker --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')"
fi

if ! docker compose version &>/dev/null 2>&1; then
  err "Docker Compose v2 not found"
  info "Install: https://docs.docker.com/compose/install/"
  MISSING=1
else
  log "Docker Compose $(docker compose version --short 2>/dev/null || echo 'v2')"
fi

if ! command -v openssl &>/dev/null; then
  err "openssl not found"
  info "Install: apt install openssl  or  brew install openssl"
  MISSING=1
else
  log "openssl $(openssl version | cut -d' ' -f2)"
fi

[ "$MISSING" -eq 1 ] && die "Install missing dependencies and re-run."

if ! docker info &>/dev/null; then
  die "Docker daemon is not running. Start it and try again."
fi

# ═════════════════════════════════════════════════════════════════════════════
# 2. Install directory
# ═════════════════════════════════════════════════════════════════════════════
section "Install location"

# Detect whether we're running from inside the repo
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-./install.sh}")" 2>/dev/null && pwd || pwd)"
if [ -f "$SCRIPT_DIR/docker-compose.yml" ] && [ -d "$SCRIPT_DIR/apps" ]; then
  # Running from inside the repo — offer to use the current dir or a separate dir
  DEFAULT_DIR="$SCRIPT_DIR"
  info "Repo detected at: $SCRIPT_DIR"
else
  DEFAULT_DIR="/opt/applad"
fi

INSTALL_DIR="$(ask "Install directory:" "$DEFAULT_DIR")"

if [ ! -d "$INSTALL_DIR" ]; then
  mkdir -p "$INSTALL_DIR"
  log "Created $INSTALL_DIR"
fi

cd "$INSTALL_DIR"

# ═════════════════════════════════════════════════════════════════════════════
# 3. Upgrade path
# ═════════════════════════════════════════════════════════════════════════════
if [ "$MODE" = upgrade ]; then
  [ -f "$COMPOSE_FILE" ] || die "No $COMPOSE_FILE found in $INSTALL_DIR. Run install first."
  section "Upgrading Applad"
  NEW_VERSION="$(ask "Target version:" "latest")"

  # Snapshot the database before anything changes, so a bad upgrade is a
  # restore rather than a loss.
  SNAPSHOT="$INSTALL_DIR/backups/pre-${NEW_VERSION}.dump"
  RESTORE_CMD="docker compose -f $COMPOSE_FILE exec -T postgres pg_restore -U applad -d applad --clean --if-exists < $SNAPSHOT"
  mkdir -p "$INSTALL_DIR/backups"
  info "Snapshotting database to ${SNAPSHOT}…"
  if docker compose -f "$COMPOSE_FILE" exec -T postgres pg_dump -Fc -U applad -d applad > "$SNAPSHOT"; then
    log "Snapshot written ($(du -h "$SNAPSHOT" | cut -f1))"
  else
    rm -f "$SNAPSHOT"
    if [ "${SKIP_DB_SNAPSHOT:-0}" = "1" ]; then
      warn "Continuing without a snapshot (SKIP_DB_SNAPSHOT=1)"
    else
      die "Database snapshot failed — is postgres running? Set SKIP_DB_SNAPSHOT=1 to upgrade without one."
    fi
  fi

  # Update the version in .env if it exists
  if [ -f .env ]; then
    if grep -q '^APPLAD_VERSION=' .env; then
      sed -i.bak "s/^APPLAD_VERSION=.*/APPLAD_VERSION=${NEW_VERSION}/" .env
    else
      echo "APPLAD_VERSION=${NEW_VERSION}" >> .env
    fi
  fi

  info "Pulling images (version: ${NEW_VERSION})…"
  if ! APPLAD_VERSION="$NEW_VERSION" docker compose -f "$COMPOSE_FILE" pull; then
    [ -f .env.bak ] && mv .env.bak .env
    err "Image pull failed — nothing was changed, services are still on the previous version."
    info "Database snapshot kept at: ${SNAPSHOT}"
    exit 1
  fi
  info "Restarting services…"
  if ! APPLAD_VERSION="$NEW_VERSION" docker compose -f "$COMPOSE_FILE" up -d --remove-orphans; then
    err "Upgrade failed to start."
    info "Restore the pre-upgrade database with:"
    printf "      ${BOLD}%s${RESET}\n" "$RESTORE_CMD"
    info "Then roll back the images with:"
    printf "      ${BOLD}APPLAD_VERSION=<previous> docker compose -f %s up -d${RESET}\n" "$COMPOSE_FILE"
    exit 1
  fi
  rm -f .env.bak
  log "Upgrade complete — running ${NEW_VERSION}"
  info "Pre-upgrade snapshot: ${SNAPSHOT} (restore: ${RESTORE_CMD})"
  printf '\n'
  info "Tail logs: ${BOLD}docker compose -f %s logs -f${RESET}" "$COMPOSE_FILE"
  exit 0
fi

# ═════════════════════════════════════════════════════════════════════════════
# 4. Fetch release assets (if not already present in install dir)
# ═════════════════════════════════════════════════════════════════════════════
section "Fetching release files"

# docker-compose.yml — always refresh on install so image references stay current
if [ -f "$SCRIPT_DIR/docker-compose.yml" ] && [ -d "$SCRIPT_DIR/apps" ] && [ "$INSTALL_DIR" != "$SCRIPT_DIR" ]; then
  cp "$SCRIPT_DIR/docker-compose.yml" "$INSTALL_DIR/$COMPOSE_FILE"
  log "docker-compose.yml copied from repo"
else
  info "Downloading docker-compose.yml…"
  fetch "${RELEASE_BASE}/docker-compose.yml" "$COMPOSE_FILE"
  log "docker-compose.yml downloaded"
fi

# postgres init.sql — runs once on first database init
mkdir -p "$(dirname "$INIT_SQL")"
if [ -f "$SCRIPT_DIR/docker/postgres/init.sql" ] && [ "$INSTALL_DIR" != "$SCRIPT_DIR" ]; then
  cp "$SCRIPT_DIR/docker/postgres/init.sql" "$INSTALL_DIR/$INIT_SQL"
  log "postgres-init.sql copied from repo"
elif [ ! -f "$INIT_SQL" ]; then
  info "Downloading postgres-init.sql…"
  fetch "${RELEASE_BASE}/docker/postgres/init.sql" "$INIT_SQL"
  log "postgres-init.sql downloaded"
else
  log "postgres-init.sql already present"
fi


# ═════════════════════════════════════════════════════════════════════════════
# 5. Configuration prompts
# ═════════════════════════════════════════════════════════════════════════════

# If .env already exists, ask whether to reconfigure
if [ -f .env ]; then
  warn ".env already exists"
  if ! ask_yn "Reconfigure? (No = keep existing .env and restart services)"; then
    section "Starting services"
    docker compose -f "$COMPOSE_FILE" up -d --remove-orphans
    log "Services started"
    _DOMAIN="$(grep '^APPLAD_DOMAIN=' .env 2>/dev/null | cut -d= -f2 || echo 'localhost')"
    _SSL="$(grep '^_SSL_MODE=' .env 2>/dev/null | cut -d= -f2 || echo 'none')"
    _PROTO="http"; [ "$_SSL" != "none" ] && _PROTO="https"
    printf '\n'
    info "Applad is at ${BOLD}${_PROTO}://${_DOMAIN}${RESET}"
    exit 0
  fi
fi

section "Configuration"
printf "  ${DIM}Press Enter to accept defaults shown in [brackets].${RESET}\n\n"

# Domain
DOMAIN="$(ask "Domain or IP (e.g. applad.example.com):" "localhost")"
if [ "$DOMAIN" = "localhost" ] || [[ "$DOMAIN" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  SSL_CAPABLE=false
else
  SSL_CAPABLE=true
fi

# HTTPS — Caddy obtains and renews certs itself (Let's Encrypt, HTTP-01) when a
# real domain points at this server. No certbot, no cron, no cert paths.
SSL_MODE=none
LE_EMAIL=''
if [ "$SSL_CAPABLE" = true ]; then
  printf '\n'
  if ask_yn "Enable automatic HTTPS (Let's Encrypt via Caddy)?" y; then
    SSL_MODE=auto
    LE_EMAIL="$(ask "Email for Let's Encrypt notices (optional):" "")"
  fi
fi

printf '\n'

# Version
APPLAD_VERSION="$(ask "Applad version:" "latest")"

# Secrets — auto-generated
printf '\n'
info "Generating secrets…"
JWT_SECRET="$(gen_secret 32)"
DB_PASSWORD="$(gen_secret 16)"
CREDENTIALS_ENCRYPTION_KEY="$(gen_secret 32)"
API_KEY_SECRET="$(gen_secret 32)"
STORAGE_ENCRYPTION_KEY="$(gen_secret 32)"
log "JWT_SECRET                 (64 hex chars)"
log "DB_PASSWORD                (32 hex chars)"
log "CREDENTIALS_ENCRYPTION_KEY (64 hex chars)"
log "API_KEY_SECRET             (64 hex chars)"
log "STORAGE_ENCRYPTION_KEY     (64 hex chars)"
printf '\n'

# Storage
STORAGE_IDX="$(ask_choice "Storage driver:" \
  "Local filesystem (default)" \
  "S3-compatible (AWS S3, Backblaze B2, MinIO, Cloudflare R2)")"
STORAGE_DRIVER=local
S3_ENDPOINT=''; S3_BUCKET=''; S3_REGION='us-east-1'
S3_ACCESS_KEY_ID=''; S3_SECRET_ACCESS_KEY=''
if [ "$STORAGE_IDX" = "1" ]; then
  STORAGE_DRIVER=s3
  S3_ENDPOINT="$(ask "Endpoint URL (blank = AWS):" "")"
  S3_BUCKET="$(ask "Bucket name:")"
  S3_REGION="$(ask "Region:" "us-east-1")"
  S3_ACCESS_KEY_ID="$(ask "Access key ID:")"
  S3_SECRET_ACCESS_KEY="$(ask_secret "Secret access key:")"
fi
printf '\n'

# SMTP
SMTP_HOST=''; SMTP_PORT='587'; SMTP_USER=''; SMTP_PASS=''
SMTP_FROM="noreply@${DOMAIN}"
if ask_yn "Configure SMTP for email sending?" "n"; then
  SMTP_HOST="$(ask "SMTP host:" "smtp.example.com")"
  SMTP_PORT="$(ask "SMTP port:" "587")"
  SMTP_USER="$(ask "SMTP user:")"
  SMTP_PASS="$(ask_secret "SMTP password:")"
  SMTP_FROM="$(ask "From address:" "noreply@${DOMAIN}")"
fi
printf '\n'

# Console signup
SIGNUP_IDX="$(ask_choice "Console signup:" \
  "auto (open until first account, then locked)" \
  "always open" \
  "always locked")"
case "$SIGNUP_IDX" in
  1) CONSOLE_SIGNUP_ENABLED=true ;;
  2) CONSOLE_SIGNUP_ENABLED=false ;;
  *) CONSOLE_SIGNUP_ENABLED=auto ;;
esac
printf '\n'

# ClamAV
ENABLE_CLAMAV=false
if ask_yn "Enable ClamAV antivirus for file uploads? (~1 GB image)" "n"; then
  ENABLE_CLAMAV=true
fi

printf '\n'

# AI chat
AI_PROVIDER=''; AI_API_KEY=''; AI_MODEL=''; AI_BASE_URL=''
if ask_yn "Enable Applad AI assistant?" "n"; then
  printf '\n'
  printf "  ${DIM}Providers: anthropic | openai | gemini | ollama${RESET}\n"
  AI_PROVIDER="$(ask "AI provider:" "anthropic")"
  case "$AI_PROVIDER" in
    anthropic) printf "  ${DIM}Models: claude-sonnet-4-6 | claude-opus-4-6 | claude-haiku-4-5${RESET}\n" ;;
    openai)    printf "  ${DIM}Models: gpt-4o | gpt-4o-mini | o3-mini${RESET}\n" ;;
    gemini)    printf "  ${DIM}Models: gemini-2.0-flash | gemini-1.5-pro${RESET}\n" ;;
    ollama)    printf "  ${DIM}Models: llama3.2 | mistral | phi3 (no API key required)${RESET}\n" ;;
  esac
  AI_MODEL="$(ask "Model:")"
  if [ "$AI_PROVIDER" != "ollama" ]; then
    AI_API_KEY="$(ask_secret "API key:")"
  fi
  if [ "$AI_PROVIDER" = "ollama" ]; then
    AI_BASE_URL="$(ask "Ollama base URL:" "http://localhost:11434")"
  fi
fi

# ═════════════════════════════════════════════════════════════════════════════
# 6. Write .env
# ═════════════════════════════════════════════════════════════════════════════
section "Writing .env"

cat > .env << EOF
# Applad — generated by install.sh on $(date -u '+%Y-%m-%dT%H:%M:%SZ')
# Re-run ./install.sh to reconfigure, or ./install.sh upgrade to update.

APPLAD_VERSION=${APPLAD_VERSION}
APPLAD_DOMAIN=${DOMAIN}
_SSL_MODE=${SSL_MODE}

APP_ENV=production
PORT=8080

JWT_SECRET=${JWT_SECRET}

# At-rest secrets. Each falls back to JWT_SECRET if unset, but dedicated keys
# are generated here so rotating one never invalidates the others.
CREDENTIALS_ENCRYPTION_KEY=${CREDENTIALS_ENCRYPTION_KEY}
API_KEY_SECRET=${API_KEY_SECRET}
# Encrypts files in buckets that have encryption enabled. Only used once such a
# bucket exists — but if you LOSE this key, that data is UNRECOVERABLE. Back it up.
STORAGE_ENCRYPTION_KEY=${STORAGE_ENCRYPTION_KEY}

DB_PASSWORD=${DB_PASSWORD}
DATABASE_DSN=postgres://applad:${DB_PASSWORD}@pgbouncer:5432/applad?sslmode=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10

REDIS_ADDR=redis:6379

STORAGE_PATH=/var/applad/storage
STORAGE_DRIVER=${STORAGE_DRIVER}
S3_ENDPOINT=${S3_ENDPOINT}
S3_BUCKET=${S3_BUCKET}
S3_REGION=${S3_REGION}
S3_ACCESS_KEY_ID=${S3_ACCESS_KEY_ID}
S3_SECRET_ACCESS_KEY=${S3_SECRET_ACCESS_KEY}

CONSOLE_SIGNUP_ENABLED=${CONSOLE_SIGNUP_ENABLED}

SMTP_HOST=${SMTP_HOST}
SMTP_PORT=${SMTP_PORT}
SMTP_USER=${SMTP_USER}
SMTP_PASS=${SMTP_PASS}
SMTP_FROM=${SMTP_FROM}

# AI chat assistant (optional)
# anthropic  →  claude-sonnet-4-6 | claude-opus-4-6 | claude-haiku-4-5
# openai     →  gpt-4o | gpt-4o-mini | o3-mini
# gemini     →  gemini-2.0-flash | gemini-1.5-pro
# ollama     →  llama3.2 | mistral | phi3  (AI_API_KEY not required)
AI_PROVIDER=${AI_PROVIDER}
AI_API_KEY=${AI_API_KEY}
AI_MODEL=${AI_MODEL}
AI_BASE_URL=${AI_BASE_URL}
EOF

log ".env written"
warn "Back up .env: STORAGE_ENCRYPTION_KEY is unrecoverable — losing it loses any encrypted-bucket data."

# ═════════════════════════════════════════════════════════════════════════════
# 7. Write the Caddyfile
# ═════════════════════════════════════════════════════════════════════════════
section "Configuring reverse proxy"
mkdir -p "$(dirname "$CADDYFILE")"

# One Caddyfile. Caddy obtains and renews TLS itself — HTTP-01 for the domain,
# or plain HTTP when reached by IP/localhost. No certbot, no renewal cron.
if [ "$SSL_MODE" = auto ]; then
  SITE="$DOMAIN"
  log "Caddyfile written (automatic HTTPS for $DOMAIN)"
else
  SITE=":80"
  log "Caddyfile written (HTTP)"
fi

{
  [ -n "$LE_EMAIL" ] && printf '{
	email %s
}

' "$LE_EMAIL"
  cat <<CADDY
${SITE} {
	handle /v1/* {
		reverse_proxy api:8080
	}
	handle /realtime* {
		reverse_proxy api:8080
	}
	handle {
		reverse_proxy console:80
	}
}
CADDY
} > "$CADDYFILE"


# ═════════════════════════════════════════════════════════════════════════════
# 9. Pull images and start
# ═════════════════════════════════════════════════════════════════════════════
section "Pulling images"

PROFILES=""
[ "$ENABLE_CLAMAV" = true ] && PROFILES="--profile antivirus"

info "Pulling from ghcr.io/mittolabs (version: ${APPLAD_VERSION})…"
docker compose -f "$COMPOSE_FILE" $PROFILES pull

section "Starting services"
docker compose -f "$COMPOSE_FILE" $PROFILES up -d --remove-orphans
log "All services started"


# ═════════════════════════════════════════════════════════════════════════════
# 11. Health check
# ═════════════════════════════════════════════════════════════════════════════
section "Waiting for API to be healthy"

MAX_WAIT=90; WAITED=0
printf '  '
until curl -sf "http://localhost/v1/health" &>/dev/null; do
  if [ "$WAITED" -ge "$MAX_WAIT" ]; then
    printf '\n'
    warn "API did not respond in ${MAX_WAIT}s"
    info "Check logs: docker compose -f $COMPOSE_FILE logs api"
    break
  fi
  printf '.'
  sleep 3
  WAITED=$((WAITED + 3))
done
curl -sf "http://localhost/v1/health" &>/dev/null && { printf '\n'; log "API healthy"; }

# ═════════════════════════════════════════════════════════════════════════════
# 12. Done
# ═════════════════════════════════════════════════════════════════════════════
PROTO="http"; [ "$SSL_MODE" != "none" ] && PROTO="https"
BASE_URL="${PROTO}://${DOMAIN}"

printf '\n'
printf "  ${BOLD}${GREEN}╔══════════════════════════════════════════════════╗${RESET}\n"
printf "  ${BOLD}${GREEN}║        Applad is up and running!                 ║${RESET}\n"
printf "  ${BOLD}${GREEN}╚══════════════════════════════════════════════════╝${RESET}\n"
printf '\n'
printf "  ${BOLD}Console:${RESET}      ${CYAN}%s${RESET}\n"           "$BASE_URL"
printf "  ${BOLD}API:${RESET}          ${CYAN}%s/v1${RESET}\n"        "$BASE_URL"
printf "  ${BOLD}Install dir:${RESET}  ${DIM}%s${RESET}\n"            "$INSTALL_DIR"
printf "  ${BOLD}Version:${RESET}      ${DIM}%s${RESET}\n"            "$APPLAD_VERSION"
printf '\n'
printf "  ${BOLD}Next steps${RESET}\n"
printf "  ${DIM}──────────────────────────────────────────────────${RESET}\n"
printf "  1. Open %s and create your admin account\n"                  "$BASE_URL"
printf "  2. Create a project and copy the API key\n"
printf "  3. Install an SDK:\n"
printf "       ${DIM}npm install @applad/js${RESET}\n"
printf "       ${DIM}dart pub add applad${RESET}\n"
printf '\n'
printf "  ${BOLD}Commands${RESET}\n"
printf "  ${DIM}──────────────────────────────────────────────────${RESET}\n"
printf "  Logs:     docker compose -f %s logs -f\n"                   "$COMPOSE_FILE"
printf "  Stop:     docker compose -f %s down\n"                      "$COMPOSE_FILE"
printf "  Upgrade:  ./install.sh upgrade\n"
printf "  Status:   docker compose -f %s ps\n"                        "$COMPOSE_FILE"
printf '\n'
