# Build the React console with Vite, serve the static bundle with nginx.
# Far smaller/faster than the old Flutter image — no 800MB toolchain pull.
FROM node:22-alpine AS builder
WORKDIR /app

# Install deps first for layer caching.
COPY apps/console/package.json apps/console/package-lock.json* ./console/
RUN cd console && npm ci

# Build.
COPY apps/console/ ./console/
RUN cd console && npm run build

FROM nginx:alpine
COPY --from=builder /app/console/dist /usr/share/nginx/html
COPY deploy/docker/console/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
