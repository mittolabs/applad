<p align="center">
  <img src="assets/logo.jpg" alt="Applad" width="180" />
</p>

<p align="center">
  Open-source backend-as-a-service with a built-in workflow automation engine.<br/>
  Self-hosted. Flutter Web console. Go backend. One <code>docker compose up</code>.
</p>

---

<p align="center">
  <a href="#-self-host-in-30-seconds">Quick Start</a> ·
  <a href="#-whats-included">Features</a> ·
  <a href="#-sdks">SDKs</a> ·
  <a href="#-configuration">Configuration</a> ·
  <a href="#-development">Development</a>
</p>

---

## ⚡ Self-host in 30 seconds

No account. No cloud. No config file required.

```bash
git clone https://github.com/mittolabs/applad
cd applad
docker compose up -d
```

Open **http://localhost** → create your admin account → done.

> **That's it.** All defaults work out of the box for local development. No `.env` editing required.

<details>
<summary><strong>Want only the API? Skip the Flutter build (faster)</strong></summary>

```bash
docker compose up api postgres redis pgbouncer proxy -d
```

The API is available at **http://localhost/v1**. The Flutter console build can take a few minutes — this skips it.
</details>

---

## 🚀 Production deployment

One extra step: set a secret.

```bash
git clone https://github.com/mittolabs/applad
cd applad

# Generate secrets (macOS / Linux)
printf "JWT_SECRET=$(openssl rand -hex 32)\nDB_PASSWORD=$(openssl rand -hex 16)\n" > .env

# Start
docker compose up -d
```

Point your domain at the server and Applad is live. Full configuration options are in [`.env.example`](.env.example).

---

## 📦 What's included

| Service | Capabilities |
|---|---|
| **Auth** | Email/password, OAuth2 (Google, GitHub, Apple, Discord, Twitter + 11 more), magic link, anonymous sessions, MFA (TOTP), email verification, password reset |
| **Databases** | Tables, typed columns, indexes, relationships, row CRUD with 12 query operators, cursor pagination, schema-scoped SQL execution |
| **Storage** | Buckets, single + chunked upload, image resize & format conversion, encryption, ClamAV antivirus |
| **Functions** | Container-based serverless — Node.js, Bun, Python, Go, Dart, Rust, Ruby, PHP, or any Dockerfile. Pre-warmed containers, ~10ms cold start |
| **Realtime** | WebSocket pub/sub — auto-publishes on every database and storage change |
| **Messaging** | Email (SMTP / Mailgun / Resend), SMS (Twilio / Vonage), push (FCM / APNS), topics & subscribers |
| **Workflows** | Native DAG engine — HTTP, email, conditions, delays, code nodes, webhook triggers, execution history |
| **Teams** | Team CRUD, memberships, role-based access |
| **Deploy** | Deployment lifecycle management with Docker-based executor |
| **Avatars** | Generated initials, QR codes, credit card icons, country flags, favicon proxy |
| **Locale** | 196 countries, 50+ currencies, 50+ languages, phone codes |

---

## 🖥 Console

First-run creates your admin account. After that, signup is disabled by default (`CONSOLE_SIGNUP_ENABLED=auto`).

**Sidebar pages**: Overview → Auth → Databases → Functions → Messaging → Storage → Deploy → Workflows → Settings

---

## 🧰 SDKs

Five SDKs ship in this repo. No package registry needed — point directly at the paths.

### Client-side

```bash
# JavaScript / TypeScript
cd sdks/js && npm install && npm run build
# In your project: npm install /path/to/applad/sdks/js

# Dart / Flutter
# pubspec.yaml → dependencies: applad: path: /path/to/applad/sdks/dart
```

```typescript
import { Applad } from '@mittolabs/applad';
const client = new Applad({ endpoint: 'http://localhost', projectId: 'proj_...' });
const user = await client.auth.createAccount('you@example.com', 'hunter2');
```

```dart
final client = Applad(endpoint: 'http://localhost', projectId: 'proj_...');
final user = await client.auth.createAccount(email: 'you@example.com', password: 'hunter2');
```

### Server-side

```bash
# Node.js
cd sdks/node && npm install && npm run build

# Go — zero dependencies
go mod edit -replace github.com/mittolabs/applad-go=./sdks/go

# Python — stdlib only
pip install -e ./sdks/python
```

```typescript
import { ApplAdServer } from '@mittolabs/applad-node';
const server = new ApplAdServer({ endpoint: 'http://localhost', projectId: 'proj_...', apiKey: 'applad_key_...' });
const users = await server.users.listUsers();
```

```go
client := applad.New("http://localhost", "proj_...", "applad_key_...")
users, _ := client.Users().ListUsers()
```

```python
from applad import Client
client = Client("http://localhost", "proj_...", "applad_key_...")
users = client.users.list_users()
```

**Coverage**

| | JS client | Dart client | Node server | Go server | Python server | Dart server |
|---|:---:|:---:|:---:|:---:|:---:|:---:|
| Auth / Users | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Databases | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Storage | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Functions | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Workflows | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Messaging | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Deploy | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Teams | — | — | ✓ | ✓ | ✓ | ✓ |
| Realtime | ✓ | ✓ | — | — | — | — |

---

## ⚙️ Configuration

All config is via environment variables. For local dev the defaults just work. For production, create a `.env` (see [`.env.example`](.env.example)).

### Required for production

| Variable | How to generate |
|---|---|
| `JWT_SECRET` | `openssl rand -hex 32` |
| `DB_PASSWORD` | `openssl rand -hex 16` |

### Core

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `development` | `development` or `production` |
| `PORT` | `8080` | API server port |
| `STORAGE_PATH` | `/var/applad/storage` | File storage root |
| `CONSOLE_SIGNUP_ENABLED` | `auto` | `auto` · `true` · `false` |

### Email

| Variable | Default |
|---|---|
| `SMTP_HOST` | — |
| `SMTP_PORT` | `587` |
| `SMTP_USER` | — |
| `SMTP_PASS` | — |
| `SMTP_FROM` | `noreply@applad.local` |

Alternative providers: `MAILGUN_API_KEY` / `MAILGUN_DOMAIN`, or `RESEND_API_KEY`.

### Messaging (SMS + push)

```
TWILIO_SID / TWILIO_TOKEN / TWILIO_FROM   # SMS via Twilio
VONAGE_API_KEY / VONAGE_API_SECRET        # SMS via Vonage
FCM_SERVER_KEY                            # Firebase push (Android)
APNS_KEY_ID / APNS_TEAM_ID / APNS_KEY_PATH / APNS_BUNDLE_ID  # Apple push
```

### OAuth2

OAuth credentials are **configured per-project through the console UI** — no env vars needed. Go to your project → Settings → Auth → OAuth providers.

### S3-compatible storage

```
STORAGE_DRIVER=s3
S3_ENDPOINT=          # leave blank for AWS; set for MinIO, R2, B2, etc.
S3_BUCKET=
S3_REGION=us-east-1
S3_ACCESS_KEY_ID=
S3_SECRET_ACCESS_KEY=
```

### Antivirus (optional)

```bash
docker compose --profile antivirus up -d
```

---

## 🛠 Development

**Prerequisites**: Go 1.22+, Flutter 3.22+ / Dart 3.3+, Docker, Node.js 18+

### Backend

```bash
cd backend
go build ./...   # compile
go test ./...    # 202 unit tests
go vet ./...     # lint
```

### Console + Dart SDK

```bash
make bootstrap   # first time: activates melos, bootstraps workspace
melos analyze    # lint all Dart packages
melos test       # test all Dart packages
melos build:web  # production Flutter build
```

### TypeScript SDKs

```bash
cd sdks/js   && npm install && npm run build && npm test
cd sdks/node && npm install && npm run build
```

---

## 🏗 Architecture

```
         ┌─────────────────┐
         │   Proxy  :80    │  (OpenResty / nginx)
         └────────┬────────┘
          ┌───────┴────────┐
    ┌─────▼─────┐    ┌─────▼──────┐
    │  API :8080 │    │ Console    │
    └─────┬─────┘    └────────────┘
          │
   ┌──────┼────────────┐
   │      │            │
┌──▼───┐ ┌▼────┐ ┌─────▼──────┐
│  PG  │ │Redis│ │ 11 Workers │
└──────┘ └─────┘ └────────────┘
```

- **Go API** — single binary, chi router, 26 internal packages
- **PostgreSQL** (via PgBouncer) — primary store, per-database schemas, RLS
- **Redis** — cache, job queues, realtime pub/sub
- **11 workers** — builds, certificates, cron, databases, deletes, executions, mails, messaging, migrations, usage, webhooks
- **Flutter console** — Riverpod + GoRouter, 11 feature pages

---

## API Reference

All endpoints under `/v1`. Full OpenAPI spec: [`backend/api/openapi.yaml`](backend/api/openapi.yaml).

| Route | Auth | Description |
|---|---|---|
| `/health` | None | Server, DB, cache checks |
| `/console` | Console JWT | Admin auth + profile |
| `/projects` | None | Project + API key management |
| `/avatars` | None | Generated images |
| `/locale` | None | Countries, currencies, languages |
| `/account` | Project header | Client auth (signup, login, OAuth2, MFA, magic link) |
| `/users` | Project + Auth | Server-side user management |
| `/teams` | Project + Auth | Teams + memberships |
| `/databases` | Project + Auth | Tables, columns, indexes, rows, SQL |
| `/storage` | Project + Auth | Buckets, files, chunked upload, image preview |
| `/functions` | Project + Auth | Serverless functions |
| `/messaging` | Project + Auth | Email, SMS, push, topics |
| `/deploy` | Project + Auth | Deployment management |
| `/workflows` | Project + Auth | DAG workflows + execution history |
| `/workflows/webhooks/{id}` | Project header | Public webhook trigger |
| `/realtime` | Project header | WebSocket connection |

---

## License

BSD 3-Clause. See [LICENSE](LICENSE).
