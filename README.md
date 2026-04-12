<p align="center">
  <img src="assets/logo.jpg" alt="Applad" width="160" />
</p>

<h2 align="center">Build products, not infrastructure.</h2>

<p align="center">
  Applad is a self-hosted backend platform with everything your app needs —<br/>
  auth, databases, storage, functions, realtime, messaging, and a workflow engine.<br/>
  Your server, your data, your rules.
</p>

<br/>

<p align="center">
  <img src="assets/console-preview.png" alt="Applad console" width="100%" />
</p>

---

## Self-host

**Requirements**: Docker with the Compose plugin. Nothing else.

```bash
curl -fsSL https://raw.githubusercontent.com/mittolabs/applad/main/install.sh | bash
```

The installer will ask for your domain, TLS preference (none, Let's Encrypt, or custom cert), storage driver, and SMTP settings. Secrets are generated automatically. When it's done, open the URL it prints and create your admin account.

**To upgrade:**

```bash
./install.sh upgrade
```

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
go test ./...     # unit tests
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
