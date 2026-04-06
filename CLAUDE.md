# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Applad

Self-hosted BaaS (backend-as-a-service) with a built-in workflow engine. Go backend, Flutter Web admin console. Runs as a single `docker compose up`.

## Commands

### Local dev stack
```bash
docker compose up -d       # all services
docker compose down        # stop all
make up / make down        # shortcuts
```

To bring up only the backend (skips the slow Flutter console build):
```bash
docker compose up api mariadb redis proxy -d
```

### Backend (Go 1.22+)
```bash
cd backend
go build ./...          # build all binaries (202 tests, 19 suites)
go test ./...           # all unit tests
go test -tags=integration ./tests/...  # integration tests (requires running services)
gofmt -w .              # format
go vet ./...            # vet
```

### Flutter/Dart workspace (console + sdks/dart)
```bash
make bootstrap          # first time: activates melos, bootstraps workspace, npm install
melos analyze           # dart analyze across all packages
melos test              # flutter test across all packages
melos format            # dart format across all packages
melos build:web         # flutter build web --release (console only)
```

### TypeScript SDKs
```bash
cd sdks/js && npm install && npm run build && npm test    # client SDK
cd sdks/node && npm install && npm run build               # server SDK
```

## Architecture

### Repo layout
```
backend/        Go backend — single Go module (github.com/mittolabs/applad)
  api/          OpenAPI spec (openapi.yaml)
  cmd/api/      API server entry point
  cmd/workers/  10 worker binaries
  internal/     26 packages (see below)
  tests/        Integration tests (build-tag gated: integration)
console/        Flutter Web admin app (Lucide icons, dark Railway-style UI)
sdks/dart/      Flutter/Dart client SDK
sdks/dart-server/  Dart server SDK (http package, no Flutter dep)
sdks/js/        TypeScript client SDK
sdks/node/      Node.js server SDK
sdks/go/        Go server SDK (zero deps)
sdks/python/    Python server SDK (stdlib only)
docker/         Docker Compose + per-service Dockerfiles + nginx config
```

### Backend structure

`cmd/api/` — entry point: connects MariaDB + Redis, runs migrations, starts HTTP server. Validates JWT_SECRET is not default in production.

`cmd/workers/{type}/` — 10 independent worker binaries. All have full Redis queue consumers.

`internal/` packages (26 total):
- `config` — env-var config loader (SMTP, OAuth, Twilio, FCM settings)
- `db` — MariaDB connection + embedded migration runner (`db/migrations/*.sql`)
- `cache` — Redis client
- `queue` — Redis-backed job queue (BRPOP-based, used by workers)
- `model` — shared struct types: `Table` (alias `Collection`), `Row` (alias `Document`), `Column` (alias `Attribute`), `Index`, `User`, `Session`, `Project`, etc.
- `middleware` — CORS, `ProjectContext`, `Authenticate`, `RequireAuth`, `RateLimit`, `SecurityHeaders`, `MaxBodySize`, `ParsePagination` (with cursor support), validators
- `apperr` — standard error response helpers
- `uid` — ID generation (UUID without hyphens)
- `router` — wires all services into a single chi router
- `auth` — accounts, sessions, OAuth2 (15 providers), MFA (TOTP), magic link, email verification, password reset
- `oauth` — OAuth2 provider definitions (Google, GitHub, Apple, Facebook, Discord, Twitter, Microsoft, Slack, Spotify, LinkedIn, GitLab, Bitbucket, Twitch, Notion, Stripe) + per-project config
- `organizations` — multi-org support: CRUD, members, invites, project linking
- `avatars` — generated images: initials PNG, QR SVG, credit card icons, country flags, favicon proxy
- `databases` — databases, tables, columns, indexes, relationships, rows with 12 query operators (equal, notEqual, lessThan, greaterThan, contains, search, isNull, between, etc.)
- `functions` — serverless function management with pre-warming on create/update
- `runtime` — container-based execution engine: Docker Engine API client, warm container pool (5min idle reaper), 8 runtime templates + custom Dockerfile, pre-built base images
- `storage` — buckets, files, chunked uploads, image transformations (resize, format conversion), antivirus (ClamAV), S3/local drivers
- `teams` — teams and memberships
- `projects` — project and API key management, usage stats aggregation
- `deploy` — deployment management with Docker-based executor
- `messaging` — email (SMTP), SMS (Twilio), push (FCM), topics/subscribers
- `realtime` — WebSocket pub/sub hub with auto-publishing from databases + storage services
- `locale` — 196 countries, 50+ currencies, 50+ languages, phone codes
- `console` — system-level admin auth: signup/login/me, name/email/password update, account deletion, signup-enabled config
- `health` — health check endpoints
- `workflows` — native DAG workflow engine: definitions, topological executor, 6 node types, execution history
- `worker` — 10 background workers (all fully implemented)

### API routes (all under /v1)

| Route | Auth | Description |
|---|---|---|
| `/health` | None | Health checks (server, DB, cache) |
| `/console` (signup, login, me, me/name, me/email, me/password) | None / Console JWT | Admin console auth + profile management |
| `/organizations` (CRUD + members + invites) | Console JWT | Multi-org management |
| `/projects` (CRUD + keys + usage) | None | Project management |
| `/locale` | None | 196 countries, currencies, languages |
| `/avatars` | None | Generated images |
| `/account` (CRUD + sessions + OAuth + MFA + magic link + verification + recovery) | Project header | Client-side auth |
| `/users` | Project + Auth | Server-side user management |
| `/teams` | Project + Auth | Teams and memberships |
| `/databases` | Project + Auth | databases → tables → columns/indexes/relationships → rows with query operators |
| `/storage` | Project + Auth | Buckets, files, chunked upload, image preview |
| `/functions` (CRUD + executions + runtimes) | Project + Auth | Serverless functions with pre-warming |
| `/messaging` (email + SMS + push + topics) | Project + Auth | Multi-provider messaging |
| `/deploy` | Project + Auth | Deployment management |
| `/workflows` (CRUD + execute + executions) | Project + Auth | Native workflow engine |
| `/workflows/webhooks/{id}` | Project header | Public webhook trigger |
| `/realtime` | Project header | WebSocket connection |

### Database / migrations

10 migrations in `backend/internal/db/migrations/`:
- `001_init.sql` — core tables (projects, api_keys, users, sessions, teams, memberships, _databases, collections, attributes, _indexes, documents, buckets, files)
- `002_deployments.sql` — deployments
- `003_workflows.sql` — workflows, workflow_executions
- `004_console_users.sql` — console admin users
- `005_oauth.sql` — OAuth provider/ID columns on users
- `006_auth_extras.sql` — MFA (TOTP), auth tokens (magic link, verification, reset)
- `007_functions.sql` — functions, function_executions
- `008_project_oauth.sql` — per-project OAuth provider config
- `009_relationships.sql` — collection relationships
- `010_organizations.sql` — organizations, organization_members, project org_id

API uses tables/rows/columns terminology; MySQL tables are named `collections`/`documents`/`attributes` internally. Type aliases maintain backward compatibility.

### Flutter console

**Design system**: Dark Railway-style UI (`#0B0B0F` bg, `#16171B` surfaces, `#6C47FF` accent). Lucide icons. 8px border radius globally. Path-based routing (not hash). No slide animations — instant page swap for sidebar nav, subtle fade for full-page transitions. Web-native scroll physics (clamping, no bounce).

**Session**: Console JWT token persisted to `localStorage`. Survives page refresh.

**Route structure**:
- `/login` — split layout: branding panel (left) + form (right), responsive (stacks on <900px)
- `/onboarding` — stepper: create project → API key → SDK snippets
- `/projects` — org-level page (NO sidebar): org heading, project cards grid, members tab, settings tab, org switcher dropdown
- `/account` — profile page (NO sidebar): name/email/password update, MFA toggle, delete account
- `/overview`, `/databases`, `/storage`, `/auth`, `/deploy`, `/functions`, `/messaging`, `/workflows`, `/settings` — project-scoped pages (WITH sidebar)

**Sidebar** (220px, labeled): Org dropdown at top → Get started / Overview → BUILD section (Auth, Databases, Functions, Messaging, Storage) → DEPLOY section (Deploy) → WORKFLOWS section (Workflows) → Settings pinned to bottom.

**Shared widgets**: `PageTabs` (horizontal tab bar), `SearchListHeader` (search + total + trailing button), `SearchListFooter` (per-page dropdown + pagination).

Feature pages: `console/lib/features/`:

| Feature | Description |
|---|---|
| `login` | Split-layout sign in/up with branding, responsive |
| `onboarding` | 3-step stepper (project → API key → SDK snippets) |
| `projects` | Org-level: project cards grid, members, settings, org switcher |
| `account` | Console user profile management |
| `overview` | Railway-style interactive canvas showing project resources |
| `auth` | Users/Teams/Settings tabs with search + pagination |
| `databases` | 3-panel: databases → tables → rows with search |
| `storage` | Buckets/Usage tabs with search + pagination |
| `functions` | Function list, runtime picker, source editor, execution history |
| `deploy` | Deployment list with status chips |
| `messaging` | Email/SMS/push send forms |
| `workflows` | DAG builder: node editor, manual execute, step-by-step logs |
| `settings` | Project management + API keys |

### SDKs (6 total)

| SDK | Path | Auth | Services |
|---|---|---|---|
| Dart client | `sdks/dart/` | Session/JWT | Auth, Users, Databases, Storage, Deploy, Functions, Messaging, Realtime, Workflows |
| JS/TS client | `sdks/js/` | Session/JWT | Auth, Avatars, Databases, Functions, Locale, Messaging, Realtime, Storage, Deploy, Workflows |
| Node.js server | `sdks/node/` | API key | Users, Databases, Storage, Functions, Teams, Workflows, Messaging, Deploy |
| Dart server | `sdks/dart-server/` | API key | Users, Databases, Storage, Functions, Teams, Workflows, Messaging, Deploy |
| Go server | `sdks/go/` | API key | Users, Databases, Storage, Functions, Teams, Workflows, Messaging, Deploy |
| Python server | `sdks/python/` | API key | Users, Databases, Storage, Functions, Teams, Workflows, Messaging, Deploy |

### Testing

**202 unit tests** across 19 packages using `go-sqlmock` for database mocking:
- `auth` — signup, login, sessions, JWT, bcrypt, MFA/TOTP, auth tokens, password reset (16 tests)
- `databases` — table/row CRUD, 12 query operators, relationships, cursor pagination (17 tests)
- `storage` — bucket/file CRUD, disk I/O, image resize/format conversion, chunked upload assembly (12 tests)
- `workflows` — DAG execution, HTTP mock server, condition branching, context cancellation, all operators (22 tests)
- `console` — signup/login with DB mock, JWT roundtrip, signup-enabled auto/true/false (15 tests)
- `oauth` — auth URL generation, user parsers (Google/GitHub/Discord/generic), provider definitions (12 tests)
- `realtime` — hub pub/sub, unsubscribe, multi-channel, event formatting (6 tests)
- `messaging` — config validation, SMTP/Twilio/FCM not-configured errors, topic CRUD (10 tests)
- `functions`, `deploy`, `avatars`, `locale`, `middleware`, `model`, `apperr`, `config`, `projects`, `teams`, `uid` — validation, data completeness, JSON tags

### Docker services

| Service | Port | Notes |
|---|---|---|
| `proxy` (openresty) | 80 | Routes `/v1/` → api, `/` → console |
| `api` | 8080 (internal) | Go API server |
| `console` | 3000 (internal) | Flutter Web, served by nginx with SPA fallback |
| `mariadb` | internal | Primary store |
| `redis` | internal | Cache + pub/sub + job queues |
| `10 workers` | internal | builds (with Docker socket), certificates, databases, deletes, executions, mails, messaging, migrations, usage, webhooks |
| `clamav` | — | Off by default; enable with `--profile antivirus` |

Root-level `docker-compose.yml` — run from repo root with `docker compose up -d`.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `JWT_SECRET` | `change-me-in-production` | **Required.** HS256 signing key. Fatal error in production if unchanged. |
| `DATABASE_DSN` | `applad:applad@tcp(mariadb:3306)/applad?parseTime=true` | MariaDB DSN |
| `REDIS_ADDR` | `redis:6379` | Redis address |
| `STORAGE_PATH` | `/var/applad/storage` | Local file storage path |
| `APP_ENV` | `development` | `development` or `production` |
| `PORT` | `8080` | API server port |
| `CONSOLE_SIGNUP_ENABLED` | `auto` | `auto` (disabled after first user), `true`, or `false` |
| `SMTP_HOST/PORT/USER/PASS/FROM` | (empty) | SMTP config for email |
| `OAUTH_GOOGLE_CLIENT_ID/SECRET` | (empty) | Google OAuth2 |
| `OAUTH_GITHUB_CLIENT_ID/SECRET` | (empty) | GitHub OAuth2 |
| `OAUTH_APPLE_CLIENT_ID/SECRET` | (empty) | Apple Sign-In |
| `TWILIO_SID/TOKEN/FROM` | (empty) | Twilio SMS |
| `FCM_SERVER_KEY` | (empty) | Firebase Cloud Messaging |
