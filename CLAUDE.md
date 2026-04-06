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
docker compose up api mariadb redis proxy
```

### Backend (Go)
```bash
cd backend
go build ./...          # build all binaries
go test ./...           # all tests (unit)
go test -tags=integration ./tests/...  # integration tests (requires running services)
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
  api/          OpenAPI spec (openapi.yaml)
  cmd/api/      API server entry point
  cmd/workers/  10 worker binaries (builds, certificates, databases, deletes, executions, mails, messaging, migrations, usage, webhooks)
  internal/     All packages (see below)
  tests/        Integration tests (build-tag gated: integration)
console/        Flutter Web admin app
sdks/dart/      Flutter/Dart client SDK (console depends on this as a path dep)
sdks/js/        TypeScript client SDK
docker/         Docker Compose + per-service Dockerfiles + nginx config
```

### Backend structure

`cmd/api/` — entry point: connects MariaDB + Redis, runs migrations, starts HTTP server.

`cmd/workers/{type}/` — 10 independent worker binaries. Each is a separate process, scaled independently via Docker Compose. All workers have full Redis queue consumers.

`internal/` packages:
- `config` — env-var config loader (includes SMTP settings)
- `db` — MariaDB connection + embedded migration runner (`db/migrations/*.sql`)
- `cache` — Redis client
- `queue` — Redis-backed job queue (BRPOP-based, used by workers)
- `model` — all shared struct types with Appwrite-compatible JSON tags (`$id`, `$createdAt`, etc.)
- `middleware` — CORS, `ProjectContext`, `Authenticate` (JWT or API key), `RequireAuth`, `RateLimit`, `SecurityHeaders`, `MaxBodySize`, `ParsePagination`, input validators (`ValidateEmail`, `ValidatePassword`, `SanitizeString`)
- `apperr` — standard error response helpers matching Appwrite's error shape
- `uid` — ID generation (UUID without hyphens, matching Appwrite style)
- `router` — wires all services into a single chi router
- `auth` — accounts, sessions, OAuth2 (Google/GitHub/Apple), MFA (TOTP), magic link, email verification, password reset (service.go + handler.go)
- `oauth` — OAuth2 provider integration: authorization URL, code exchange, user info fetching (provider.go)
- `avatars` — generated avatars: initials, QR codes, credit card icons, country flags, favicons (handler.go)
- `databases` — databases, collections, attributes, indexes, documents with query operators (service.go + handler.go)
- `functions` — serverless function management: CRUD, execution, build queue integration (service.go + handler.go)
- `runtime` — container-based function execution engine: Docker Engine API client, warm container pool, runtime templates for Node/Python/Go/Dart/Bun/Rust/Ruby/PHP, custom Dockerfile support (docker.go + pool.go + executor.go + templates.go)
- `storage` — buckets, files, chunked uploads, image transformations, antivirus scanning, S3/local driver (service.go + handler.go + driver.go + antivirus.go)
- `teams` — teams and memberships (service.go + handler.go)
- `projects` — project and API key management (service.go + handler.go)
- `deploy` — deployment management with status lifecycle (service.go + handler.go)
- `messaging` — email (SMTP), SMS (Twilio), push notifications (FCM), topics/subscribers (service.go + handler.go)
- `realtime` — WebSocket pub/sub hub for live events (hub.go + client.go + handler.go)
- `locale` — 195 countries, 50+ currencies, 50+ languages, continents, phone codes, IP locale detection (service.go)
- `console` — system-level admin auth: signup, login, JWT validation, signup-enabled config (service.go + handler.go)
- `health` — health check endpoints (handler.go)
- `workflows` — native workflow engine: definitions, DAG executor, execution history (service.go + handler.go + engine.go)
- `worker` — 10 background workers: builds, certificates, databases, deletes, executions, mails, messaging, migrations, usage, webhooks

### API routes (all under /v1)

| Route | Auth | Description |
|---|---|---|
| `/health`, `/health/db`, `/health/cache` | None | Health checks |
| `/console` (signup, login, me, signup-status) | None (console JWT for /me) | Admin console auth |
| `/projects` (CRUD + keys) | None | Project management |
| `/locale` (countries, currencies, etc.) | None | Locale data (195 countries, 50+ currencies, 50+ languages) |
| `/avatars` (initials, qr, flags, etc.) | None | Generated avatars and icons |
| `/account` (CRUD + sessions + OAuth + MFA) | Project header, some public | Client-side auth with OAuth2, MFA, magic link, verification |
| `/realtime` (WebSocket) | Project header, optional JWT | Live events |
| `/users` (CRUD + sessions) | Project + Auth | Server-side user management |
| `/teams` (CRUD + memberships) | Project + Auth | Team management |
| `/databases` (full nested CRUD + queries) | Project + Auth | Databases → collections → attributes/indexes → documents with query operators |
| `/storage` (buckets + files + chunked + preview) | Project + Auth | File storage with chunked upload, image transformations |
| `/messaging` (email + SMS + push + topics) | Project + Auth | Email (SMTP), SMS (Twilio), push (FCM), topics/subscribers |
| `/functions` (CRUD + executions) | Project + Auth | Serverless functions with execution tracking |
| `/deploy` (CRUD + status) | Project + Auth | Deployment management |
| `/workflows` (CRUD + execute + executions) | Project + Auth | Native workflow engine |
| `/workflows/webhooks/{workflowId}` | Project header only | Public webhook trigger |

### Request authentication flow

Every API call (except `/v1/health`, `/v1/projects`, `/v1/locale`) requires `X-Applad-Project: <projectId>` header. Auth is then one of:
- `X-Applad-Key: applad_key_<hex>` — API key (server-side)
- `Authorization: Bearer <jwt>` — session JWT (client-side)

Public account endpoints (`POST /v1/account`, `POST /v1/account/sessions/email`, `POST /v1/account/sessions/anonymous`) skip `RequireAuth` — they live inside the project-context group but outside the `RequireAuth` group. See `internal/router/router.go` for the exact grouping.

JWT claims carry `sub` (userID), `sid` (sessionID), `pid` (projectID). Signed with `JWT_SECRET` env var using HS256.

### Security middleware stack

Applied globally in this order: RequestID → RealIP → Logger → Recoverer → CORS → SecurityHeaders → RateLimit(100/min per IP) → MaxBodySize(10MB).

### Database / migrations

Migrations live at `backend/internal/db/migrations/*.sql`, embedded via `//go:embed`. The runner (`db.Migrate()`) bootstraps `schema_migrations` before running any files, so a fresh database works on first start. Add new migrations as `NNN_description.sql` — they run in filename order.

Current migrations:
- `001_init.sql` — projects, api_keys, users, sessions, teams, memberships, _databases, collections, attributes, _indexes, documents, buckets, files
- `002_deployments.sql` — deployments table
- `003_workflows.sql` — workflows and workflow_executions tables
- `004_console_users.sql` — console admin users table
- `005_oauth.sql` — OAuth provider and ID columns on users
- `006_auth_extras.sql` — MFA (TOTP secret, recovery codes), auth tokens (magic link, verification, password reset)
- `007_functions.sql` — functions and function_executions tables

Documents (TablesDB) are stored as JSON in a `documents.data` column. `model.Document` implements `MarshalJSON` to merge data fields into the top-level JSON response, matching Appwrite's document shape.

### Storage drivers

`internal/storage/driver.go` defines a `Driver` interface with `Write`, `Read`, `Delete` methods. Two implementations:
- `LocalDriver` — writes to local filesystem (default, configured via `STORAGE_PATH`)
- `S3Driver` — S3-compatible object storage (MinIO, AWS S3)

ClamAV antivirus scanning is available via `antivirus.go` — connects to clamd via TCP and uses INSTREAM protocol.

### Worker queue system

`internal/queue/queue.go` provides a Redis list-based job queue (LPUSH to enqueue, BRPOP to dequeue with 5s timeout). Workers connect to Redis and poll for jobs.

Workers:
- `builds` — processes deployment builds: transitions deployments through building → deploying → active status
- `certificates` — generates self-signed SSL/TLS certificates for custom domains
- `databases` — database maintenance: attribute status updates, index builds, collection stats
- `deletes` — cascade-deletes resources (users, projects, databases, collections, buckets, workflows) with all related data
- `executions` — runs workflow DAGs: loads workflow definition, executes nodes in topological order, updates execution status/logs
- `mails` — transactional email delivery via SMTP (password resets, welcome emails)
- `messaging` — user-initiated messaging: batch emails and notifications from the /messaging API
- `migrations` — async schema migrations, table optimization, expired session cleanup
- `usage` — aggregates per-project usage statistics (users, docs, files, storage) into Redis
- `webhooks` — delivers webhook payloads with HMAC-SHA256 signing, 3 retries with exponential backoff

### Flutter console

Shell layout with NavigationRail sidebar (`console/lib/core/shell/shell.dart`). Feature-first layout under `console/lib/features/`:

| Feature | Status | Description |
|---|---|---|
| `login` | Complete | Console admin signup/login with auto-disable signup after first user |
| `onboarding` | Complete | Post-login stepper: create project → generate API key → SDK snippets |
| `auth` | Complete | User management table with create/delete |
| `databases` | Complete | 3-panel layout: databases → collections → documents as DataTable with dynamic columns |
| `storage` | Complete | Buckets panel + files list with download/delete |
| `settings` | Complete | Project management + API key creation with copy-to-clipboard |
| `deploy` | Complete | Deployment list with status chips, create dialog, status management |
| `messaging` | Complete | Email send form (to, subject, HTML body) with SMTP config info |
| `workflows` | Complete | Native workflow engine: CRUD, node editor, manual execute, execution history with step logs |

Core infra at `console/lib/core/`:
- `router/` — GoRouter with ShellRoute, auth guard redirect, login/onboarding routes
- `theme/` — Material 3 theme (seed color #6C47FF)
- `api/` — Dio-based API client as Riverpod provider (base URL `/v1` for proxy)
- `shell/` — NavigationRail layout with 7 destinations
- `providers/` — Project, API key, and console auth Riverpod providers

`sdks/dart/` is a path dependency of `console/` (`path: ../sdks/dart`). Always run `melos bootstrap` after pulling to keep symlinks current.

### SDKs

**Dart SDK** (`sdks/dart/`): Full client with service classes — `Auth`, `Users`, `Databases`, `Storage`, `Deploy`, `Functions`, `Messaging`, `Realtime`, `Workflows`. Main entry: `Applad(endpoint:, projectId:)` exposes all services.

**TypeScript SDK** (`sdks/js/`): Full client with service classes — `Auth`, `Avatars`, `Databases`, `Functions`, `Locale`, `Messaging`, `Realtime`, `Storage`, `Deploy`, `Workflows`. Uses native `fetch()`. Main entry: `new Applad({endpoint, projectId})` exposes all services.

### Testing

**Backend unit tests** — 12 test files across: uid, apperr, model, config, middleware (including rate limit and validation), auth handler, projects handler, databases handler, storage handler, teams handler.

**Backend integration tests** — `backend/tests/integration_test.go` (build tag: `integration`). Tests health, project CRUD, auth flow, database+document flow against a live API.

**Dart SDK tests** — `sdks/dart/test/applad_test.dart` — client instantiation, service exposure, header setting.

**TypeScript SDK tests** — `sdks/js/src/__tests__/client.test.ts` — client creation, URL building, error handling, auth headers (Jest with fetch mocks).

### Docker services

| Service | Port | Notes |
|---|---|---|
| `proxy` (openresty) | 80 | Routes `/v1/` → api, `/` → console |
| `api` | 8080 (internal) | Go API server (includes native workflow engine) |
| `console` | 3000 (also via proxy at `/`) | Flutter Web, served by nginx |
| `mariadb` | internal | Primary store |
| `redis` | internal | Cache + pub/sub + job queues |
| `clamav` | — | Off by default; enable with `--profile antivirus` |

Workers are built from a single parameterised Dockerfile (`docker/worker/Dockerfile`) using `ARG WORKER_TYPE`. Each worker service in compose passes a different `WORKER_TYPE` build arg.

Go Dockerfiles run `go mod tidy` inside the builder (no `go.sum` is committed).

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | API server port |
| `DATABASE_DSN` | `applad:applad@tcp(mariadb:3306)/applad?parseTime=true` | MariaDB DSN |
| `REDIS_ADDR` | `redis:6379` | Redis address |
| `JWT_SECRET` | `change-me-in-production` | HS256 signing key |
| `STORAGE_PATH` | `/var/applad/storage` | Local file storage path |
| `APP_ENV` | `development` | Environment name |
| `SMTP_HOST` | (empty) | SMTP server host |
| `SMTP_PORT` | `587` | SMTP port |
| `SMTP_USER` | (empty) | SMTP username |
| `SMTP_PASS` | (empty) | SMTP password |
| `SMTP_FROM` | `noreply@applad.local` | Sender email address |
| `CONSOLE_SIGNUP_ENABLED` | `auto` | Console signup: `auto` (disabled after first user), `true`, or `false` |
| `OAUTH_GOOGLE_CLIENT_ID` | (empty) | Google OAuth2 client ID |
| `OAUTH_GOOGLE_CLIENT_SECRET` | (empty) | Google OAuth2 client secret |
| `OAUTH_GITHUB_CLIENT_ID` | (empty) | GitHub OAuth2 client ID |
| `OAUTH_GITHUB_CLIENT_SECRET` | (empty) | GitHub OAuth2 client secret |
| `OAUTH_APPLE_CLIENT_ID` | (empty) | Apple OAuth2 client ID |
| `OAUTH_APPLE_CLIENT_SECRET` | (empty) | Apple OAuth2 client secret |
| `TWILIO_SID` | (empty) | Twilio account SID for SMS |
| `TWILIO_TOKEN` | (empty) | Twilio auth token |
| `TWILIO_FROM` | (empty) | Twilio sender phone number |
| `FCM_SERVER_KEY` | (empty) | Firebase Cloud Messaging server key |
