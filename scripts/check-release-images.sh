#!/usr/bin/env bash
# Every image docker-compose.yml pulls must be one the release workflow builds.
#
# These drifted apart silently: worker-transfer was added to compose but never
# to the matrix, so the image never existed, and `curl | bash` died pulling it.
# GHCR answers "denied" rather than "not found" for a missing image, so it read
# as a permissions problem for weeks.
set -euo pipefail

cd "$(dirname "$0")/.."

compose=$(grep -oE 'ghcr\.io/mittolabs/applad-[a-z-]+' docker-compose.yml \
  | sed 's|ghcr\.io/mittolabs/||' | sort -u)

# What release.yml actually produces: the two named image jobs, plus one image
# per entry in the worker matrix.
workers=$(sed -n '/^        worker:$/,/^    steps:$/p' .github/workflows/release.yml \
  | grep -oE '^          - [a-z-]+' | sed 's/^          - /applad-worker-/' | sort -u)
built=$(printf '%s\napplad-api\napplad-console\n' "$workers" | grep -v '^$' | sort -u)

missing=$(comm -23 <(printf '%s\n' "$compose") <(printf '%s\n' "$built"))

if [ -n "$missing" ]; then
  echo "These images are referenced by docker-compose.yml but nothing builds them:" >&2
  printf '  %s\n' $missing >&2
  echo >&2
  echo "Add them to the worker matrix in .github/workflows/release.yml, or stop" >&2
  echo "referencing them in docker-compose.yml." >&2
  exit 1
fi

echo "All $(printf '%s\n' "$compose" | wc -l | tr -d ' ') compose images are built by the release workflow."
