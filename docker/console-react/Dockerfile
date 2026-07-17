# Build the React console with Vite, serve the static bundle with nginx.
# Far smaller/faster than the Flutter image — no 800MB toolchain pull.
FROM node:22-alpine AS builder
WORKDIR /app

# Install deps first for layer caching.
COPY console-react/package.json console-react/package-lock.json* ./console-react/
RUN cd console-react && npm ci

# Build.
COPY console-react/ ./console-react/
RUN cd console-react && npm run build

FROM nginx:alpine
COPY --from=builder /app/console-react/dist /usr/share/nginx/html
COPY docker/console-react/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
