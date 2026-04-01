# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Applad

Self-hosted BaaS (backend-as-a-service) with a built-in workflow engine. Go backend, Flutter Web admin console. Runs as a single `docker compose up`.

## Commands

### Local dev stack
```bash
make up          # docker compose up -d (all services)
make down        # docker compose down
```

To bring up only the backend (skips the slow Flutter console build):
```bash
docker compose -f docker/docker-compose.yml up api mariadb redis workflows proxy
```

### Backend (Go)
```bash
cd backend
go build ./...          # build all binaries
go test ./...           # all tests
go test ./internal/auth/...   # single package
gofmt -w .              # format
go vet ./...            # vet
```

### Flutter/Dart workspace (console + sdks/dart)
Managed by [Melos](https://melos.codes). Run from repo root:
```bash
make bootstrap          # first time: activates melos, bootstraps workspace, npm install
melos analyze           # dart analyze across all packages
melos test              # flutter test across all packages
melos format            # dart format across all packages
melos build:web         # flutter build web --release (console only)
```

### TypeScript SDK
```bash
cd sdks/js
npm install
npm run build
npm test
```

## Architecture

### Repo layout
```
backend/        Go backend — single Go module (github.com/mittolabs/applad)
console/        Flutter Web admin app
sdks/dart/      Flutter/Dart client SDK (console depends on this as a path dep)
sdks/js/        TypeScript client SDK
docker/         Docker Compose + per-service Dockerfiles + nginx config
```

### Backend structure

`cmd/api/` — entry point: connects MariaDB + Redis, runs migrations, starts HTTP server.

`cmd/workers/{type}/` — 10 independent worker binaries (builds, certificates, databases, deletes, executions, mails, messaging, migrations, usage, webhooks). Each is a separate process, scaled independently via Docker Compose.

`internal/` packages:
- `config` — env-var config loader
- `db` — MariaDB connection + embedded migration runner (`db/migrations/*.sql`)
- `cache` — Redis client
- `model` — all shared struct types with Appwrite-compatible JSON tags (`$id`, `$createdAt`, etc.)
- `middleware` — CORS, `ProjectContext` (reads `X-Applad-Project` header), `Authenticate` (JWT or API key), `RequireAuth`
- `apperr` — standard error response helpers matching Appwrite's error shape
- `uid` — ID generation (UUID without hyphens, matching Appwrite style)
- `router` — wires all services into a single chi router
- `auth`, `databases`, `storage`, `teams`, `health`, `projects` — each split into `service.go` (business logic, takes `*db.DB`) and `handler.go` (HTTP handlers + route registration)

### Request authentication flow

Every API call (except `/v1/health` and `/v1/projects`) requires `X-Applad-Project: <projectId>` header. Auth is then one of:
- `X-Applad-Key: applad_key_<hex>` — API key (server-side)
- `Authorization: Bearer <jwt>` — session JWT (client-side)

Public account endpoints (`POST /v1/account`, `POST /v1/account/sessions/email`, `POST /v1/account/sessions/anonymous`) skip `RequireAuth` — they live inside the project-context group but outside the `RequireAuth` group. See `internal/router/router.go` for the exact grouping.

JWT claims carry `sub` (userID), `sid` (sessionID), `pid` (projectID). Signed with `JWT_SECRET` env var using HS256.

### Database / migrations

Migrations live at `backend/internal/db/migrations/*.sql`, embedded via `//go:embed`. The runner (`db.Migrate()`) bootstraps `schema_migrations` before running any files, so a fresh database works on first start. Add new migrations as `NNN_description.sql` — they run in filename order.

Documents (TablesDB) are stored as JSON in a `documents.data` column. `model.Document` implements `MarshalJSON` to merge data fields into the top-level JSON response, matching Appwrite's document shape.

### Flutter console

Feature-first layout under `console/lib/features/` (auth, databases, storage, deploy, messaging, workflows, settings). Core infra at `console/lib/core/` (router via go_router + Riverpod, Material 3 theme, Dio API client). The console is a plain client of the public REST API — no privileged access.

`sdks/dart/` is a path dependency of `console/` (`path: ../sdks/dart`). Always run `melos bootstrap` after pulling to keep symlinks current.

### Docker services

| Service | Port | Notes |
|---|---|---|
| `proxy` (openresty) | 80 | Routes `/v1/` → api, `/workflows/` → n8n, `/` → console |
| `api` | 8080 (internal) | Go API server |
| `console` | 3000 (also via proxy at `/`) | Flutter Web, served by nginx |
| `workflows` (n8n) | 5678 (internal) | `regular` execution mode for local dev |
| `mariadb` | internal | Primary store |
| `redis` | internal | Cache + pub/sub |
| `clamav` | — | Off by default; enable with `--profile antivirus` |

Workers are built from a single parameterised Dockerfile (`docker/worker/Dockerfile`) using `ARG WORKER_TYPE`. Each worker service in compose passes a different `WORKER_TYPE` build arg.

Go Dockerfiles run `go mod tidy` inside the builder (no `go.sum` is committed).
