# Applad — Multi-stage Docker build
# Stage 1: Build the Dart CLI and server
# Stage 2: Minimal runtime image

# ─── Stage 1: Build ──────────────────────────────────────────────────────────
FROM dart:stable AS build

WORKDIR /app

# Copy the monorepo
COPY . .

# Remove applad_console from workspace to prevent Flutter SDK errors
RUN sed -i -e '/- packages\/applad_console/d' pubspec.yaml

# Install melos and bootstrap packages
RUN dart pub global activate melos
ENV PATH="$PATH:/root/.pub-cache/bin"
RUN melos bootstrap

# Build the server
WORKDIR /app/packages/applad_server
RUN dart pub global activate dart_frog_cli
RUN dart_frog build
RUN mkdir -p /app/build_bin
RUN dart compile exe build/bin/server.dart -o /app/build_bin/applad-server

# Build the CLI
WORKDIR /app/packages/applad_cli
RUN dart compile exe bin/applad.dart -o /app/build_bin/applad

# ─── Stage 2: Runtime ────────────────────────────────────────────────────────
FROM debian:bookworm-slim AS runtime

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy compiled binaries
COPY --from=build /app/build_bin/applad-server /usr/local/bin/applad-server
COPY --from=build /app/build_bin/applad /usr/local/bin/applad

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
