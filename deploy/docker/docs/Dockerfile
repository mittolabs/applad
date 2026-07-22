# Documentation — Next.js (Fumadocs) in standalone output, run by Node.
FROM node:22-alpine AS builder
WORKDIR /app

# Copy the whole app first so the fumadocs-mdx postinstall (which reads
# content/docs + source.config.ts to generate .source) has everything it needs.
COPY apps/docs/ ./

# Cross-site links (site, console) are baked in at build time by Next. Local
# dev passes the .localhost variants; production uses the defaults.
ARG NEXT_PUBLIC_SITE_URL
ARG NEXT_PUBLIC_CONSOLE_URL
ENV NEXT_PUBLIC_SITE_URL=$NEXT_PUBLIC_SITE_URL
ENV NEXT_PUBLIC_CONSOLE_URL=$NEXT_PUBLIC_CONSOLE_URL

RUN npm install && npm run build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=80
ENV HOSTNAME=0.0.0.0

COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public

EXPOSE 80
CMD ["node", "server.js"]
