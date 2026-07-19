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

if [ ! -f .env ]; then
  SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')"
  printf 'JWT_SECRET=%s\nAPP_ENV=production\n' "$SECRET" > .env
  ok "Generated .env with a fresh JWT_SECRET"
fi

info "Building and starting the stack (first run can take a few minutes)"
docker compose up -d --build

printf '\n'
ok "Applad is running."
printf '  Console:  http://localhost\n'
printf '  Manage:   cd %s  (docker compose logs -f · docker compose down)\n' "$DIR"
