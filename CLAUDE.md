# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Applad

Self-hosted BaaS (backend-as-a-service) with a built-in workflow engine. Go backend, React + Vite admin console. Runs as a single `docker compose up` with PostgreSQL and Redis.

> **Scope of this repo.** This is the self-hostable product. `docker compose up`
> brings up exactly what a self-hoster needs — the console (`console.applad.io`),
> its API and workers, and the data stores — and nothing else. The extra web
> properties Mittolabs Cloud runs on top of it (the marketing site `applad.io`,
> docs `docs.applad.io`, the status page `status.applad.io`) and the full
> multi-domain production provisioning live in the **private
> `mittolabs/applad-cloud`** Ansible repo, not here. The docs *source* stays in
> this repo under `apps/docs/` (applad-cloud builds it from here).
>
> Two composes. `docker-compose.yml` is the one stack: each service carries both
> `build:` (from `apps/…`) and `image: ghcr.io/mittolabs/applad-*`, so
> `docker compose up --build` builds locally while `install.sh` does
> `docker compose pull` to run the prebuilt images — same file, no `.release`
> variant. `docker-compose.dev.yml` is the separate hot-reload dev mode (Go
> `go run` over mounted source).

> The admin console was a Flutter Web app; it was rewritten in React + Vite + TypeScript (Tailwind v4 + shadcn/ui) at feature parity and now lives at `apps/console/`. The old Flutter app has been removed. `melos` now manages only `sdks/dart`. Console design/parity notes: `apps/console/CORE_REFERENCE.md`, `apps/console/PARITY_AUDIT.md`.

## Commands

### Local dev stack
```bash
docker compose up -d       # all services
docker compose down        # stop all
make up / make down        # shortcuts
```

To bring up only the backend (skips the console build):
```bash
docker compose up api postgres redis proxy -d
```

### Backend (Go 1.22+)
```bash
cd apps/backend
go build ./...          # build all binaries (202 tests, 19 suites)
go test ./...           # all unit tests
go test -tags=integration ./tests/...  # integration tests (requires running services)
gofmt -w .              # format
go vet ./...            # vet
```

### React console (`apps/console/`)
```bash
cd apps/console
npm install             # first time
npm run dev             # Vite dev server (proxies /v1 → :8080)
npm run build           # tsc -b && vite build
npm test                # vitest run
npm run lint            # eslint
```

### Dart SDK workspace (sdks/dart)
```bash
make bootstrap          # first time: activates melos, bootstraps workspace
melos analyze           # dart analyze
melos test              # flutter test
melos format            # dart format
```

### TypeScript SDKs
```bash
cd sdks/js && npm install && npm run build && npm test    # client SDK
cd sdks/node && npm install && npm run build               # server SDK
```

## Architecture

### Repo layout

App source is grouped under `apps/`. The installer, the SDKs, `docker/`, and the
compose files stay at the root because external consumers reference them there
(the `install.sh` one-liner URL, the `github.com/mittolabs/applad/sdks/go`
import path, `docker compose up`).

```
apps/
  backend/      Go backend — single Go module (github.com/mittolabs/applad)
    api/        OpenAPI spec (openapi.yaml)
    cmd/api/    API server entry point
    cmd/workers/  10 worker binaries
    internal/   26 packages (see below)
    tests/      Integration tests (build-tag gated: integration)
  console/      React + Vite admin app (Tailwind v4 + shadcn/ui, Lucide icons, dark Railway-style UI)
  docs/         Documentation source (Next.js / Fumadocs). Served at docs.applad.io
                by applad-cloud, which builds it from here; a self-host install
                does not run it.
sdks/dart/      Dart SDK — client (Dio) + server (http) in one package
sdks/js/        TypeScript client SDK
sdks/node/      Node.js server SDK
sdks/go/        Go server SDK (zero deps) — imported as github.com/mittolabs/applad/sdks/go
sdks/python/    Python server SDK (stdlib only)
docker/  Per-service Dockerfiles + nginx config (self-host vhosts)
install.sh · docker-compose{,.dev,.release}.yml  self-host install + run (root, external URLs)
```

> Not in this repo: the marketing site, the status page, the full multi-domain
> compose, the multi-vhost production nginx, and the k8s manifests moved to the
> private **`mittolabs/applad-cloud`** Ansible repo, which provisions the full
> hosted stack. This repo builds only the console-facing stack; its CI publishes
> the `ghcr.io/mittolabs/applad-*` images that `install.sh` and applad-cloud both
> pull.

### Backend structure

`cmd/api/` — entry point: connects PostgreSQL + Redis, runs migrations, starts HTTP server. Validates JWT_SECRET is not default in production.

`cmd/workers/{type}/` — 10 independent worker binaries. All have full Redis queue consumers.

`internal/` packages (26 total):
- `config` — env-var config loader (SMTP, OAuth, Twilio, FCM settings)
- `db` — PostgreSQL connection + embedded migration runner (`db/migrations/*.sql`)
- `cache` — Redis client
- `queue` — Redis-backed job queue (BRPOP-based, used by workers)
- `model` — shared struct types: `Table`, `Row`, `Column`, `Index`, `User`, `Session`, `Project`, etc.
- `middleware` — CORS, `ProjectContext`, `Authenticate`, `RequireAuth`, `RateLimit`, `SecurityHeaders`, `MaxBodySize`, `ParsePagination` (with cursor support), validators
- `apperr` — standard error response helpers
- `uid` — ID generation (UUID without hyphens)
- `router` — wires all services into a single chi router
- `auth` — accounts, sessions, OAuth2 (15 providers), MFA (TOTP), magic link, email verification, password reset
- `oauth` — OAuth2 provider definitions (Google, GitHub, Apple, Facebook, Discord, Twitter, Microsoft, Slack, Spotify, LinkedIn, GitLab, Bitbucket, Twitch, Notion, Stripe) + per-project config
- `organizations` — multi-org support: CRUD, members, invites, project linking
- `avatars` — generated images: initials PNG, QR SVG, credit card icons, country flags, favicon proxy
- `databases` — schema orchestration in Go, row CRUD via direct SQL, RLS policy sync, tables/columns/indexes/relationships/rows
- `functions` — serverless function management with pre-warming on create/update
- `runtime` — container-based execution engine: Docker Engine API client, warm container pool (5min idle reaper), 8 runtime templates + custom Dockerfile, pre-built base images
- `storage` — buckets, files, chunked uploads, image transformations (resize, format conversion), antivirus (ClamAV), S3/local drivers
- `teams` — teams and memberships
- `projects` — project and API key management, usage stats aggregation
- `deploy` — deployment management with Docker-based executor
- `messaging` — email (SMTP), SMS (Twilio), push (FCM), topics/subscribers
- `realtime` — WebSocket pub/sub hub backed by PostgreSQL LISTEN/NOTIFY for table changes plus in-process publishing for non-database events
- `locale` — 196 countries, 50+ currencies, 50+ languages, phone codes
- `console` — system-level admin auth: signup/login/me, name/email/password update, account deletion, signup-enabled config
- `health` — health check endpoints
- `workflows` — native DAG workflow engine: definitions, topological executor, 6 node types, execution history
- `testlab` — runs a project's own test suite in a container and records per-case results, read from the JUnit XML the suite writes
- `cronx` — cron expression parsing and validation (standard 5-field, ranges, lists, names, descriptors, `CRON_TZ=` prefix)
- `worker` — 11 background workers (all fully implemented)

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
| `/databases` | Project + Auth | databases → tables → columns/indexes/relationships → rows, plus `/sql` for schema-scoped SQL execution |
| `/storage` | Project + Auth | Buckets, files, chunked upload, image preview |
| `/functions` (CRUD + executions + runtimes) | Project + Auth | Serverless functions with pre-warming |
| `/messaging` (email + SMS + push + topics) | Project + Auth | Multi-provider messaging |
| `/deploy` | Project + Auth | Deployment management |
| `/workflows` (CRUD + execute + executions) | Project + Auth | Native workflow engine |
| `/workflows/webhooks/{id}` | Project header | Public webhook trigger |
| `/realtime` | Project header | WebSocket connection |

### Database / migrations

Single consolidated migration in `apps/backend/internal/db/migrations/`:
- `001_init.sql` — PostgreSQL schema, triggers, RLS helpers, metadata tables, and all product services

API and internal code use tables/rows/columns terminology only. User data lives in real PostgreSQL schemas named `p_{projectId}_{databaseId}`.

### React console

**Design system**: Dark Railway-style UI (`#0B0B0F` bg, `#16171B` surfaces, `#6C47FF` accent). Lucide icons. 8px border radius globally. Path-based routing (not hash). No slide animations — instant page swap for sidebar nav, subtle fade for full-page transitions. Web-native scroll physics (clamping, no bounce).

**Session**: Console JWT token persisted to `localStorage`. Survives page refresh.

**Route structure**:
- `/login` — split layout: branding panel (left) + form (right), responsive (stacks on <900px)
- `/onboarding` — stepper: create project → API key → SDK snippets
- `/projects` — org-level page (NO sidebar): org heading, project cards grid, members tab, settings tab, org switcher dropdown
- `/account` — profile page (NO sidebar): name/email/password update, MFA toggle, delete account
- `/overview`, `/databases`, `/storage`, `/auth`, `/deploy`, `/functions`, `/messaging`, `/workflows`, `/settings` — project-scoped pages (WITH sidebar)

**Sidebar**: 68px icon rail + 220px labelled panel. Groups: Overview → Build
(Auth, Databases, Functions, Storage, Messaging, Realtime, Feature Flags) →
Deploy (Sites, Containers, Mobile, Desktop) → Automate (Workflows) →
Observe (8 routes) → Settings pinned to bottom. The order mirrors the
lifecycle the marketing site sells: Build, Test, Deploy, Automate, Monitor.
Every child of Deploy is a deploy target — /deploy/targets filtered by type.
Workflows sits at rail level rather than inside Build because it composes the
other primitives rather than being one.

**Shared widgets**: `PageTabs` (horizontal tab bar), `SearchListHeader` (search + total + trailing button), `SearchListFooter` (per-page dropdown + pagination).

Feature pages: `console/src/features/`:

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

### SDKs (5 total)

| SDK | Path | Auth | Services |
|---|---|---|---|
| Dart | `sdks/dart/` | Session/JWT (client) or API key (server) | Client: Auth, Users, Databases, Storage, Deploy, Functions, Messaging, Realtime, Workflows, Flags. Server: Users, Databases, Storage, Functions, Teams, Workflows, Messaging, Deploy, Flags |
| JS/TS client | `sdks/js/` | Session/JWT | Auth, Avatars, Databases, Functions, Locale, Messaging, Realtime, Storage, Deploy, Workflows, Flags, Analytics, Search, Vectors, Edge, Billing, Regions |
| Node.js server | `sdks/node/` | API key | Users, Databases, Storage, Functions, Teams, Workflows, Messaging, Deploy, Flags, Analytics, Search, Vectors, Edge, Billing, Regions |
| Go server | `sdks/go/` | API key | Users, Databases, Storage, Functions, Teams, Workflows, Messaging, Deploy, Flags |
| Python server | `sdks/python/` | API key | Users, Databases, Storage, Functions, Teams, Workflows, Messaging, Deploy, Flags |

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
| `proxy` (openresty) | 80 | Name-based vhosts (see below); unknown hosts fall back to the console |
| `api` | 8080 (internal) | Go API server |
| `console` | 3000 (internal) | React + Vite static bundle, served by nginx with SPA fallback |
| `postgres` | internal | Primary store |
| `redis` | internal | Cache + pub/sub + job queues |
| `10 workers` | internal | builds (with Docker socket), certificates, databases, deletes, executions, mails, messaging, migrations, usage, webhooks |
| `clamav` | — | Off by default; enable with `--profile antivirus` |

Root-level `docker-compose.yml` — run from repo root with `docker compose up -d`.

### Hosts

This is the *full* Mittolabs Cloud layout. A **self-hosted** install serves only
the rows marked ✓ below — the console, the API, deployed apps, and the fallback;
its `docker/nginx/nginx.conf` (and the smaller one `install.sh` writes) contain
exactly those vhosts. The ⛅ rows — marketing, docs, status — are extra web
properties provisioned by the private **`applad-cloud`** repo's multi-vhost edge
(`roles/proxy`), not by anything in this repo.

Two domains, deliberately separated: everything of ours is on `applad.io`,
while deployed customer apps get `applad.dev` to themselves. Deployed apps run
arbitrary customer code, so sharing a registrable domain with the console
would let one of them set cookies the console receives and make every app
same-site with it.

| | Production | Local | Serves |
|---|---|---|---|
| ⛅ | `applad.io` | `applad.io.localhost` | Marketing site (applad-cloud) |
| ✓ | `console.applad.io` | `console.applad.io.localhost` | Admin console (+ same-origin `/v1`) |
| ✓ | `api.applad.io` | `api.applad.io.localhost` | Public API for SDKs |
| ⛅ | `docs.applad.io` | `docs.applad.io.localhost` | Documentation (built from this repo's `docs/`) |
| ⛅ | `status.applad.io` | `status.applad.io.localhost` | Status page (applad-cloud) |
| ✓ | `<app>.applad.dev` | `<app>.applad.dev.localhost` | Deployed apps |
| ⛅ | `applad.dev` | `applad.dev.localhost` | Redirects to the marketing site |

Any other host (an IP, `localhost`) falls back to the console, which is how a
self-hosted install is reached. Local names need no `/etc/hosts` entry:
browsers resolve every `*.localhost` name to `127.0.0.1`.

On login the API sets two cookies: the HttpOnly session, and
`applad_session=1`, a token-free marker the marketing site reads to swap
"Get started" for "Go to console". Their scope is derived from the console's
own hostname, so `console.applad.io` shares them with `applad.io` and a
self-hosted console on an IP keeps them host-only.

### Test

A suite says how a project runs its tests — base image, setup command, test
command, and where it leaves a JUnit XML report. That report is the interchange
format, so Applad supports a new framework by configuration rather than code.

Runs execute on the builds worker: an image is built from the source (git or
upload), run to completion, and the report copied out of the stopped container.
Four objects, because conflating them is what made the first cut unusable:

| | |
|---|---|
| **runner** | how tests execute — image, setup, command, report path |
| **test** | one behaviour, recorded or discovered by running. Tagged, quarantinable, carries history |
| **suite** | a selection of tests by tag, plus triggers: on deploy, on a schedule |
| **run** | a suite executed against a target, at a moment |

The target belongs to the run, not the test, so one suite checks main and a
branch rather than being duplicated per environment. Recordings share a single
generated runner; giving each its own is what listed every recording twice.

Retries are on, and a test that fails then passes within a run is recorded as
flaky rather than failing it. A quarantined test still runs and still reports
but no longer decides the verdict, which is how a known-flaky test stops
blocking deploys without being deleted and forgotten.

A suite may also name an `artifactsPath`; that directory is copied out of the
container and stored under `STORAGE_PATH/test-artifacts/<runId>`. Where a
runner names its output after the test, the artifact is attached to that case,
so a failing browser test shows its own recording. Test containers join the
deploy network, so a browser suite reaches the app it exercises by container
name — pass it as `BASE_URL`.

### The recording studio

`Test → Flows → Record` opens a real browser in a container against a target,
streams it into the console over a WebSocket, and forwards clicks and
keystrokes back. A recorder injected into the page turns each interaction into
a step with a durable selector — role and name, a test id, a label — rather
than a coordinate, and records which match was clicked so strict matching does
not fail on replay. Assert mode turns a click into an assertion instead of an
action.

Flows are stored as steps, not code, and compile to Playwright for the web and
Maestro for devices. Saving writes a complete generated Playwright project to
the runner's source path, so a recording is immediately a suite that runs on
every change. Device platforms need an emulator (virtualisation) or a Mac,
which is why only web is wired up.

The browser is started by the builds worker, not the API: only that worker
holds the Docker socket. The API reaches the browser over the shared network.
Chromium binds DevTools to loopback and rejects non-localhost Host headers, so
the browser image runs a forwarder and every call presents `Host: localhost`.

### GitHub

Deploying from a repository goes through a GitHub App — **Applad Cloud**
(`github.com/apps/applad-cloud`, owned by `mittolabs`) — rather than a token
somebody pasted. An app holds a private key, signs a short-lived JWT to prove
it is itself, and exchanges that for a token scoped to one *installation*: one
account that installed it, on the repositories they picked. Those tokens last
an hour and are minted per use, so the key is the only long-lived secret and it
never reaches the database.

A connection is therefore a record of permission, not a credential:
`git_connections` stores which installation a project may act through, and
nothing that can be replayed if the row leaks.

| | |
|---|---|
| **Authorisation** | Whose repo may a project clone? The app can reach every repository anyone installed it on, so `CloneTokenForRepo` resolves the installation for the repo and refuses unless a `git_connections` row ties it to the asking project |
| **Clone** | `x-access-token:<token>@github.com/...`, passed separately from the URL that is quoted in errors — git echoes what it was given, and a failed private clone would otherwise print the token into a build log |
| **Webhook** | One URL for every installation (`POST /v1/deploy/git/webhook`), verified against the app's single secret, then routed by `installation.id`. The per-connection route (`/webhook/{connectionId}`) stays for GitLab and hand-wired hooks |
| **Install** | Console → *Connect GitHub* → `github.com/apps/applad-cloud/installations/new?state=…`, where the state is held in Redis against the project that started it, so a link somebody is tricked into following cannot attach their installation elsewhere |

Permissions requested: `contents:read`, `metadata:read`, `statuses:write`,
`pull_requests:write`. The last is wider than today's use — only PR events are
read — because adding a permission later makes every existing installation
re-approve, and commenting a preview URL on a pull request is the next step.

Self-hosted instances have no app. `githubapp.ErrNotConfigured` is a condition,
not a failure: public repositories still deploy by URL, and the console says so
instead of offering a button that cannot work. An operator who wants private
repos registers their own app and sets `GITHUB_APP_*`.

### Rate limits

A request is not a unit of cost. Reading a list is a cached query; starting a
deploy is minutes of CPU; sending an SMS is money and a sender reputation. One
counter over all of them protected the cheapest thing in the system and
nothing else — refreshing a page hit the limit while nothing capped builds,
messages or password guesses.

Three layers, each keyed by whoever should bear it:

| | Keyed by | Guards against |
|---|---|---|
| Generic | address, split anonymous vs signed-in | a flood from an unknown source |
| Credentials | address **and** the account attempted | password guessing; an attacker rotates the first and cannot rotate the second |
| Project work | project | deploys, messages, executions and test runs — the operations that cost something |

The signed-in generic limit is sized for a console, which issues twenty or
more requests to render a page. Expensive operations are named individually in
`middleware.ProjectWorkRules`, so a limit is a statement about one operation
rather than a number covering everything.

Redis being unavailable allows the request: this is fairness, not
authorisation, and failing closed would turn a cache outage into a total one.

### Scheduling

`worker-cron` ticks once a minute and fires anything due: workflows with
`trigger_type='cron'`, functions with a `cron` field, and deploy targets with
a `cron` field. Expressions are parsed by `internal/cronx` and validated when
written, so an unusable schedule is rejected at the API rather than accepted
and never run.

State lives in `cron_state` (one row per scheduled thing, keyed by kind +
id), holding the expression, last and next run, and any parse error. The
worker asks "what is due?" rather than "does this minute match?", so runs
owed while it was down are not lost. A backlog fires once and resyncs rather
than replaying every missed occurrence, and a Redis lock keeps replicas from
double-firing.

### Hosted vs self-hosted

The same build serves both, and `CONSOLE_SIGNUP_ENABLED` is what separates
them:

| | Hosted (Mittolabs Cloud) | Self-hosted |
|---|---|---|
| Setting | `true` | `auto` (default) |
| Who can register | anyone | the first account, then invitees only |
| First run | normal signup | "Create the owner account" |

Invites are not signup. A closed instance stays closed to `/console/signup`;
colleagues arrive through `POST /console/invites/{token}/redeem`, where the
token is the credential and the address is read from the invite rather than
supplied by the caller. Redemption creates the account and activates the
membership in one transaction, and consumes the token. The console serves
this at `/invite/:token`, and shows the link once after an invite is created
since a self-hosted instance usually has no SMTP configured.

`GET /console/signup-status` returns `signupEnabled`, `firstRun` and
`inviteOnly` so the login page can say which applies rather than silently
dropping people onto the sign-in form.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `JWT_SECRET` | `change-me-in-production` | **Required.** HS256 signing key. Fatal error in production if unchanged. |
| `DATABASE_DSN` | `postgres://applad:applad@postgres:5432/applad?sslmode=disable` | PostgreSQL DSN |
| `REDIS_ADDR` | `redis:6379` | Redis address |
| `STORAGE_PATH` | `/var/applad/storage` | Local file storage path |
| `APP_ENV` | `development` | `development` or `production` |
| `PORT` | `8080` | API server port |
| `CONSOLE_SIGNUP_ENABLED` | `auto` | `auto` (self-hosted: closes after the first account), `true` (hosted), or `false` |
| `SESSION_COOKIE_DOMAIN` | (empty) | Overrides console cookie scope. Normally derived from the host: `console.<parent>` shares cookies with `<parent>` |
| `GITHUB_APP_ID` | (empty) | Applad Cloud's GitHub App. Absent on self-hosted, where git deploys fall back to public repositories |
| `GITHUB_APP_SLUG` | `applad` | App name in URLs — `github.com/apps/<slug>` |
| `GITHUB_APP_PRIVATE_KEY` / `_PATH` | (empty) | The PEM inline (newlines escaped) or a path to it. Prefer the path: an env var is visible to anything that can inspect the container |
| `GITHUB_APP_WEBHOOK_SECRET` | (empty) | Verifies every inbound delivery for the app |
| `GITHUB_APP_CLIENT_ID/SECRET` | (empty) | OAuth identity of the app |
| `SMTP_HOST/PORT/USER/PASS/FROM` | (empty) | SMTP config for email |
| `OAUTH_GOOGLE_CLIENT_ID/SECRET` | (empty) | Google OAuth2 |
| `OAUTH_GITHUB_CLIENT_ID/SECRET` | (empty) | GitHub OAuth2 |
| `OAUTH_APPLE_CLIENT_ID/SECRET` | (empty) | Apple Sign-In |
| `TWILIO_SID/TOKEN/FROM` | (empty) | Twilio SMS |
| `FCM_SERVER_KEY` | (empty) | Firebase Cloud Messaging |
