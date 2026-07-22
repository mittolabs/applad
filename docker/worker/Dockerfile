FROM golang:1.25-alpine AS builder
ARG WORKER_TYPE
WORKDIR /app
# Cache module download — only re-runs when go.mod/go.sum change
COPY go.mod go.sum ./
RUN go mod download
# Copy source and build
COPY . .
RUN GOMAXPROCS=1 CGO_ENABLED=0 GOOS=linux go build -p=1 -o /worker ./cmd/workers/${WORKER_TYPE}

# Railpack turns a source tree into an image without a Dockerfile. It is
# fetched here rather than installed at runtime so a build never depends on
# GitHub being reachable, and the version is pinned: railpack is pre-1.0 and
# ships often, and "latest" is somebody else's release becoming our outage.
FROM debian:bookworm-slim AS railpack
ARG RAILPACK_VERSION=v0.31.2
ARG TARGETARCH
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates && rm -rf /var/lib/apt/lists/* && \
    case "${TARGETARCH}" in \
      arm64) RP_ARCH=arm64 ;; \
      amd64) RP_ARCH=x86_64 ;; \
      *) echo "unsupported arch: ${TARGETARCH}" >&2; exit 1 ;; \
    esac && \
    curl -fsSL "https://github.com/railwayapp/railpack/releases/download/${RAILPACK_VERSION}/railpack-${RAILPACK_VERSION}-${RP_ARCH}-unknown-linux-musl.tar.gz" \
      -o /tmp/railpack.tar.gz && \
    tar xzf /tmp/railpack.tar.gz -C /tmp && \
    mv /tmp/railpack /usr/local/bin/railpack && \
    chmod +x /usr/local/bin/railpack

# The Docker CLI, client only — the daemon is the host's, reached through the
# mounted socket. Railpack streams a finished image to `docker load`, so
# without this a build ran to completion and then hung forever on "sending
# tarball" with nothing on the other end.
FROM debian:bookworm-slim AS dockercli
ARG DOCKER_VERSION=27.5.1
ARG TARGETARCH
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates && rm -rf /var/lib/apt/lists/* && \
    case "${TARGETARCH}" in \
      arm64) DK_ARCH=aarch64 ;; \
      amd64) DK_ARCH=x86_64 ;; \
      *) echo "unsupported arch: ${TARGETARCH}" >&2; exit 1 ;; \
    esac && \
    curl -fsSL "https://download.docker.com/linux/static/stable/${DK_ARCH}/docker-${DOCKER_VERSION}.tgz" -o /tmp/docker.tgz && \
    tar xzf /tmp/docker.tgz -C /tmp && \
    mv /tmp/docker/docker /usr/local/bin/docker && \
    chmod +x /usr/local/bin/docker

# glibc rather than musl: railpack fetches mise to manage language versions,
# and mise ships glibc binaries. On Alpine they fail to exec at all, with an
# error that names the file rather than the reason.
FROM debian:bookworm-slim
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates tzdata git && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /worker .
COPY --from=railpack /usr/local/bin/railpack /usr/local/bin/railpack
COPY --from=dockercli /usr/local/bin/docker /usr/local/bin/docker
ENTRYPOINT ["/app/worker"]
