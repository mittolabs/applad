<p align="center">
  <img src="assets/logo.jpg" alt="Applad" width="160" />
</p>

<h3 align="center">Open-source backend-as-a-service with a built-in workflow engine.</h3>

<p align="center">
  Auth · Databases · Storage · Functions · Realtime · Messaging · Workflows · Deploy<br/>
  Self-hosted. Go backend. Flutter Web console. One <code>docker compose up</code>.
</p>

<br/>

<p align="center">
  <img src="assets/console-preview.png" alt="Applad console" width="100%" />
</p>

---

## Self-host

**Requirements**: Docker + Docker Compose. Nothing else.

```bash
git clone https://github.com/mittolabs/applad
cd applad
docker compose up -d
```

Open **http://localhost** → create your admin account → you're in.

> Defaults work out of the box for local dev. No `.env` required.

<details>
<summary>API only (skips the Flutter console build)</summary>

```bash
docker compose up api postgres redis proxy -d
```

The API is at **http://localhost/v1**. Useful when iterating on the backend — skips the slow Flutter compile step.
</details>

### Production

Set secrets before going live:

```bash
printf "JWT_SECRET=$(openssl rand -hex 32)\nDB_PASSWORD=$(openssl rand -hex 16)\n" > .env
docker compose up -d
```

Point your domain at the server. That's it.

### Key environment variables

| Variable | Default | Description |
|---|---|---|
| `JWT_SECRET` | `change-me` | **Required in prod.** HS256 signing key |
| `APP_ENV` | `development` | Set to `production` to enforce `JWT_SECRET` |
| `DATABASE_DSN` | `postgres://...` | PostgreSQL connection string |
| `REDIS_ADDR` | `redis:6379` | Redis address |
| `STORAGE_PATH` | `/var/applad/storage` | File storage root |
| `CONSOLE_SIGNUP_ENABLED` | `auto` | `auto` (disabled after first user) · `true` · `false` |
| `SMTP_HOST/PORT/USER/PASS/FROM` | — | Email via SMTP |
| `TWILIO_SID/TOKEN/FROM` | — | SMS via Twilio |
| `FCM_SERVER_KEY` | — | Firebase push notifications |

OAuth2 providers (Google, GitHub, Apple, Discord, Twitter + 11 more) are configured per-project inside the console UI — no env vars needed.

---

## What's included

| | |
|---|---|
| **Auth** | Email/password, 15 OAuth2 providers, magic link, anonymous sessions, MFA (TOTP), email verification, password reset |
| **Databases** | Tables, typed columns, indexes, relationships, row CRUD, 12 query operators, cursor pagination, schema-scoped SQL |
| **Storage** | Buckets, single + chunked upload, image resize & format conversion, optional ClamAV antivirus |
| **Functions** | Container-based serverless — Node.js, Bun, Python, Go, Dart, Rust, Ruby, PHP, or any Dockerfile |
| **Realtime** | WebSocket pub/sub — auto-publishes on every database and storage change |
| **Messaging** | Email (SMTP), SMS (Twilio), push (FCM), topics & subscribers |
| **Workflows** | Native DAG engine — HTTP, email, conditions, delays, code nodes, webhook triggers |
| **Teams** | Team CRUD, memberships, role-based access |
| **Deploy** | Deployment lifecycle management with Docker-based executor |

---

## Contributing

Pull requests are welcome. To get started locally:

**Prerequisites**: Go 1.22+, Flutter 3.22+ / Dart 3.3+, Docker, Node.js 18+

### Backend (Go)

```bash
cd backend
go build ./...    # compile
go test ./...     # unit tests (202 tests across 19 packages)
go vet ./...      # vet
gofmt -w .        # format
```

### Console (Flutter)

```bash
make bootstrap    # first time: activates melos, bootstraps workspace
melos analyze     # lint all Dart packages
melos test        # run all Flutter tests
melos build:web   # production web build
```

### TypeScript SDKs

```bash
cd sdks/js   && npm install && npm run build && npm test
cd sdks/node && npm install && npm run build
```

---

## License

BSD 3-Clause. See [LICENSE](LICENSE).
