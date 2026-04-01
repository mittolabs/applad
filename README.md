<p align="center">
  <img src="assets/logo.jpg" alt="Applad" width="120" />
</p>

<h1 align="center">Applad</h1>

<p align="center">
  Open-source backend-as-a-service with a built-in workflow automation engine.<br/>
  Self-hosted. Flutter Web console. Go backend.
</p>

---

## What is Applad?

Applad is a self-hosted, open-source backend-as-a-service — Auth, Databases, Storage, Serverless Functions, Realtime, and Messaging — combined with a full workflow automation engine, all managed through a **Flutter Web admin console** as its primary differentiator.

One Docker Compose command. Your data. Your infrastructure.

---

## Services

| Service | Description |
|---|---|
| **Auth** | Email/password, OAuth2 (45+ providers), magic link, phone OTP, anonymous sessions, MFA, Teams |
| **Databases (TablesDB)** | Collections, typed attributes, indexes, document CRUD, full query operators, atomic ops, relationships |
| **Storage** | Buckets, chunked file upload, image transformations, encryption, antivirus |
| **Deploy** | Universal deployment — serverless functions, web apps, mobile builds, containers |
| **Realtime** | WebSocket event subscriptions across all services |
| **Messaging** | Email (Mailgun, Sendgrid, SMTP, Resend), SMS (Twilio, Vonage, MSG91), Push (FCM, APNS) |
| **Workflows** | Full workflow automation engine — triggers, logic nodes, AI/LangChain, Applad-native nodes |
| **Locale** | IP-based locale, countries, currencies, languages |
| **Health** | Service health endpoints for all dependencies |

---

## The Deploy Service

Applad unifies Functions and Sites into a single **Deploy** service. Every deployable artifact — a serverless function, a web app, a Flutter mobile app, a background container — follows the same model:

**Target** (where to deploy + runtime) + **Pipeline** (how to build) + **Release** (one deployment attempt)

### Serverless Targets
- Invoke via HTTP, cron, Realtime event, or webhook
- Runtimes: Node.js, Bun, Python, Go, Dart, Ruby, PHP, Java, Kotlin, Swift, .NET, C++
- Sync and async execution, streaming logs

### Web Targets
- Static hosting with CDN, custom domains, automatic SSL
- Framework auto-detection: Next.js, SvelteKit, Nuxt, Astro, Remix, Angular, Flutter Web
- Branch preview deployments

### Mobile Targets
- Android (APK/AAB → Play Store, Firebase App Distribution)
- iOS (IPA → App Store Connect, TestFlight) — requires macOS build agent
- Build agents: register any macOS/Linux machine with `applad agent start`

### Container Targets
- Dockerfile builds, multi-platform (amd64/arm64)
- Push to Applad registry, Docker Hub, ghcr.io, ECR, GAR
- Deploy to Fly.io, Railway, Cloud Run, ECS, DigitalOcean, or generic SSH

---

## Workflow Engine

Phase 1 ships n8n embedded in the Docker Compose stack, integrated with Applad via SSO. Phase 2 replaces the engine with a native Go implementation maintaining full API and UI compatibility.

### What you get on day one
- **45+ trigger types** — Webhook, Schedule, Email (IMAP), RabbitMQ, MQTT, and Applad-native triggers for Auth, Database, Storage, and Function events
- **Flow control** — IF, Switch, Merge, Loop Over Items, Wait, Stop and Error
- **Data manipulation** — Filter, Sort, Deduplicate, Aggregate, Split Out, Compare Datasets, Code (JS/Python)
- **AI-native** — AI Agent, 9 LLM providers (OpenAI, Anthropic, Gemini, Ollama, Bedrock, etc.), RAG pipeline nodes, 7 vector store integrations
- **Applad-native nodes** — Create/read/update/delete Auth users, Database documents, Storage files, and Function executions directly from any workflow — auto-configured with your project credentials

### Applad-native triggers
| Trigger | Fires on |
|---|---|
| Auth Event | `user.create`, `user.delete`, `session.create`, `session.delete` |
| Database Event | `document.create`, `document.update`, `document.delete` per collection |
| Storage Event | `file.create`, `file.update`, `file.delete` per bucket |
| Function Event | `execution.complete`, `execution.failed` per function |

---

## Flutter Web Console

The admin console is a Flutter Web app — the same codebase runs on web, desktop, and mobile. It is a first-class client of Applad's own public REST/GraphQL API with no privileged backdoors.

- Full Auth admin: user search, impersonation, session management, MFA, labels
- Schema editor: databases, collections, attributes, indexes — live against the API
- Document explorer with query builder
- File browser with drag-and-drop upload and preview
- Deploy console: pipeline wizard, release timeline, live log streaming, one-click rollback
- Execution log viewer for serverless targets
- Workflows section (Phase 1: n8n WebView with SSO; Phase 2: native Flutter canvas)
- Real-time dashboard stats, usage analytics per service

---

## Stack

| Layer | Technology |
|---|---|
| Backend | Go |
| Admin console | Flutter Web |
| Primary database | MariaDB |
| Cache / Realtime | Redis |
| Reverse proxy | OpenResty / Nginx |
| Workflow engine (Phase 1) | n8n (embedded Docker service) |
| Workflow engine (Phase 2) | Native Go (planned) |
| Antivirus | ClamAV (optional) |

---

## Self-hosting

```bash
git clone https://github.com/mittolabs/applad
cd applad
docker compose up -d
```

Applad is environment-variable driven. A single `docker compose up` brings up all services: API, console, workers, MariaDB, Redis, ClamAV, and the workflow engine.

---

## SDKs

| SDK | Priority |
|---|---|
| Flutter / Dart (client) | P1 |
| JavaScript / TypeScript | P1 |
| React Native | P1 |
| Node.js (server) | P2 |
| Go (server) | P2 |
| Python (server) | P2 |
| Dart (server) | P2 |

---

## Build Roadmap

| Phase | What ships |
|---|---|
| 1 | Infrastructure — Docker Compose, API gateway, projects, health |
| 2 | Auth — accounts, sessions, email+password, OAuth2, Teams |
| 3 | Databases (TablesDB) — CRUD, attributes, indexes, queries, permissions |
| 4 | Storage — buckets, upload, download |
| 5 | Deploy — serverless target, Node.js runtime, invocation API |
| 6 | Realtime — WebSocket subscriptions |
| 7 | Admin console (Flutter Web) |
| 8 | Workflows Phase 1 — n8n embedded, SSO, Applad-native nodes |
| 9 | Auth — MFA, full OAuth2 provider list, advanced config |
| 10 | Messaging — providers, topics, messages |
| 11 | Locale, Avatars, Health utility services |
| 12 | Deploy — Web, Mobile, Container targets |
| 13 | Migrations, SDKs |
| 14 | Workflows Phase 2 — native Go engine, native Flutter canvas |

---

## License

BSD 3-Clause. See [LICENSE](LICENSE).

---

*Spec: Applad v1.0*
