#!/bin/sh
# Applad self-host installer.
#   curl -fsSL https://applad.io/install.sh | sh
#
# Fetches Applad, generates a secure JWT secret, and brings the stack up with
# Docker Compose. Re-running updates an existing install in place.
set -eu

REPO_URL="https://github.com/mittolabs/applad.git"
DIR="${APPLAD_DIR:-applad}"

info() { printf '\033[36m›\033[0m %s\n' "$1"; }
ok()   { printf '\033[32m✓\033[0m %s\n' "$1"; }
die()  { printf '\033[31m✗ %s\033[0m\n' "$1" >&2; exit 1; }

command -v docker >/dev/null 2>&1 || die "Docker is required. Install it from https://docs.docker.com/get-docker/ and re-run."
docker compose version >/dev/null 2>&1 || die "Docker Compose v2 is required (the 'docker compose' command)."
command -v git >/dev/null 2>&1 || die "git is required to fetch Applad."

if [ -d "$DIR/.git" ]; then
  info "Updating existing Applad in ./$DIR"
  git -C "$DIR" pull --ff-only
else
  info "Cloning Applad into ./$DIR"
  git clone --depth 1 "$REPO_URL" "$DIR"
fi
cd "$DIR"

rand_hex() { openssl rand -hex "$1" 2>/dev/null || head -c "$1" /dev/urandom | od -An -tx1 | tr -d ' \n'; }

if [ ! -f .env ]; then
  SECRET="$(rand_hex 32)"
  DBPASS="$(rand_hex 16)"
  if [ -f .env.example ]; then
    # Start from the documented example rather than a bare minimum, so every
    # setting (SMTP, OAuth, storage, AI) is present and commented, and running
    # this is followed by editing one file.
    sed -e "s|^JWT_SECRET=.*|JWT_SECRET=$SECRET|" \
        -e "s|^DB_PASSWORD=.*|DB_PASSWORD=$DBPASS|" \
        -e "s|^APP_ENV=.*|APP_ENV=production|" \
        .env.example > .env
  else
    printf 'JWT_SECRET=%s\nDB_PASSWORD=%s\nAPP_ENV=production\n' "$SECRET" "$DBPASS" > .env
  fi
  ok "Generated .env with a fresh JWT_SECRET and database password"
else
  info "Keeping your existing .env"
fi

info "Building and starting the stack (first run can take a few minutes)"
docker compose up -d --build

printf '\n'
ok "Applad is running."
printf '  Console:  http://localhost\n'
printf '  Configure: %s/.env  (then: docker compose up -d)\n' "$DIR"
printf '  Manage:   cd %s  (docker compose logs -f · docker compose down)\n' "$DIR"
