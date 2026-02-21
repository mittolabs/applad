# Applad — Multi-stage Docker build
# Stage 1: Build the Dart CLI and server
# Stage 2: Minimal runtime image

# ─── Stage 1: Build ──────────────────────────────────────────────────────────
FROM dart:3.5 AS build

WORKDIR /app

# Copy the monorepo
COPY . .

# Install melos and bootstrap packages
RUN dart pub global activate melos
RUN melos bootstrap

# Build the server
WORKDIR /app/packages/applad_server
RUN dart compile exe bin/server.dart -o /app/build/applad_server

# Build the CLI
WORKDIR /app/packages/applad_cli
RUN dart compile exe bin/applad.dart -o /app/build/applad

# ─── Stage 2: Runtime ────────────────────────────────────────────────────────
FROM debian:bookworm-slim AS runtime

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy compiled binaries
COPY --from=build /app/build/applad_server /usr/local/bin/applad-server
COPY --from=build /app/build/applad /usr/local/bin/applad

# Create data directory for SQLite
RUN mkdir -p /data && chmod 755 /data

WORKDIR /app

# Default config path (mount your config tree here)
ENV APPLAD_CONFIG_PATH=/app/config
ENV PORT=8080

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/v1/health || exit 1

CMD ["applad-server"]
