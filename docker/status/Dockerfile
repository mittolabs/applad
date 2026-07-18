# Status page — Vite + React static bundle, served by nginx.
FROM node:22-alpine AS builder
WORKDIR /app

COPY status/package.json status/package-lock.json* ./status/
RUN cd status && npm install

COPY status/ ./status/
RUN cd status && npm run build

FROM nginx:alpine
COPY --from=builder /app/status/dist /usr/share/nginx/html
COPY docker/status/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
