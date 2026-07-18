# Marketing site — Vite + React static bundle, served by nginx.
FROM node:22-alpine AS builder
WORKDIR /app

COPY website/package.json website/package-lock.json* ./website/
RUN cd website && npm install

COPY website/ ./website/
RUN cd website && npm run build

FROM nginx:alpine
COPY --from=builder /app/website/dist /usr/share/nginx/html
COPY docker/website/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
