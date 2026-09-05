# Build the React console with Vite, serve the static bundle with Caddy.
# Far smaller/faster than the old Flutter image — no 800MB toolchain pull.
FROM node:22-alpine AS builder
WORKDIR /app

# Install deps first for layer caching.
COPY apps/console/package.json apps/console/package-lock.json* ./console/
RUN cd console && npm ci

# Build.
#
# The version is baked in here because Vite resolves import.meta.env at build
# time: nothing set on the running container can reach the bundle. Without it
# the console fell back to a hard-coded 0.2.0 and every release since has
# reported that, whatever it actually was.
ARG APP_VERSION=dev
ENV VITE_APP_VERSION=$APP_VERSION
COPY apps/console/ ./console/
RUN cd console && npm run build

FROM caddy:2-alpine
COPY --from=builder /app/console/dist /srv
COPY docker/console/Caddyfile /etc/caddy/Caddyfile
EXPOSE 80
