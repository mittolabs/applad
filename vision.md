Here's the fully updated vision document:

---

# Applad — Full Vision Document

## What Is Applad?

I want us to create **Applad**, an open-source BaaS inspired by Appwrite, Supabase, PocketBase, Firebase, Appsmith and Directus — with features from v0.dev, Motia, n8n and Lovable. Written in Dart for the backend core and Flutter for the admin app (enabling true cross-platform support beyond just the web).

The core idea is that it should be **very easy to self-host, extend, scale and configure** — born directly from the frustrations I experienced trying to set up and maintain tools like Appwrite and Supabase. But Applad is more than a BaaS. It is the **Infrastructure as Code tool for your entire backend** — and the visual UI is simply a friendly interface into that same system. And beyond that, Applad is your **lad** — an active, AI-powered collaborator that helps you configure, spin up, debug, and manage everything across your entire stack.

The name says it all: **App** — it helps you build and deploy apps. **Lad** — it's your assistant, your collaborator, your infrastructure companion. In the CLI, this surfaces as `applad instruct` — self-documenting, professional, and consistent with the rest of the CLI's imperative tone.

---

## The Core Reframe

Most teams today run a BaaS for their backend, a separate IaC tool like Terraform for infrastructure, a separate CI/CD tool for deployments, a separate feature flag service, and a separate analytics platform. These tools don't talk to each other, drift apart over time, and require different mental models and skill sets to operate. Applad replaces all of them with a single, coherent system where **everything is config, everything is visual, everything is assisted by AI, and they are always the same thing**.

The config files are the backend. The admin UI is a lens into them. The AI is the lad that helps you work with both. Every click in the UI writes config. Every config change reflects in the UI. Every instruction you give produces real changes, not just suggestions. There is no gap, no drift, no manual translation.

---

## Where Applad Runs — Local, VPS, and Cloud On-Demand

Applad is designed to run anywhere you can point it — and to move between those places without changing your application code, your config structure, or your mental model. There are three deployment contexts, and they form a natural continuum:

**Local:**
Your laptop, a home server, a Raspberry Pi. Single binary, SQLite by default, zero external dependencies. This is where you develop, prototype, and run internal tools. Applad running locally is indistinguishable from Applad running in production from the application's perspective — same config tree, same CLI, same admin UI, same behavior. The only difference is the machine it's on. A developer can run the full stack locally with a single command, work entirely offline, and push to a VPS or cloud when ready.

**VPS:**
A DigitalOcean Droplet, Hetzner server, Linode, bare metal — any machine you own or rent at a flat rate. You have full control, predictable costs, and no vendor dependency at the infrastructure level. Applad connects over SSH, provisions what it needs using Docker, manages everything from your config tree, and leaves. This is the sweet spot for most indie developers, small teams, startups, and anyone who wants full ownership without managing cloud complexity. A single mid-range VPS can comfortably run a production Applad instance serving thousands of users.

**Cloud providers on-demand:**
AWS, GCP, Azure, and others — but used surgically, not as a platform commitment. Applad treats cloud provider resources as adapters you draw from when they make sense for a specific resource, not as an ecosystem you're locked into. You might run your core app on a Hetzner VPS because it's fast and cheap, use S3 for storage because the pricing is right and Rclone already supports it natively, use SES for high-volume email at scale, and spin up a cloud compute instance for a heavy data processing job — paying only for the duration of that job, then tearing it down. Applad connects to cloud provider APIs the same way it connects to any other infrastructure — over your configured credentials, agentlessly, doing the work and leaving.

**The continuum:**

```
Local dev
  → VPS staging
    → VPS production
      → VPS + cloud storage adapter
        → VPS + cloud database adapter
          → Multi-VPS cluster
            → Kubernetes on cloud
```

You move along this line as your needs grow, and Applad moves with you. Nothing changes about how your application is built or how your config is structured. The only thing that changes is what Applad is pointed at. A team can start with a $6/month VPS and a SQLite database, grow to a Postgres instance on the same VPS, then move the database to a managed RDS instance when they need it — all by changing one line in `database/database.yaml` and running `applad config push`.

**Cloud providers as utility, not platform:**

This is fundamentally different from being "an AWS shop" or "a GCP team." Cloud provider resources in Applad are like electricity — you draw from them when they're the right tool for a specific job, without committing your application architecture to their ecosystem. Applad's adapter model means:

- Storage can be local filesystem on a VPS today, S3 tomorrow, R2 next month — your application code never changes
- Database can be SQLite locally, Postgres on a VPS in staging, RDS in production — same config structure, different adapter target
- Functions can run in Docker containers on your VPS or burst to cloud compute for heavy workloads — Applad manages the routing
- Email can go through SMTP on a VPS, switch to SES at scale — one line change in `messaging/messaging.yaml`

And because Applad is agentless, spinning up a cloud VM for a burst workload, using it, and tearing it down is just another SSH operation — no persistent infrastructure, no ongoing cost, no agent to clean up.

---

## Agentless — Like Ansible, Powered by Familiar Tools

Applad is **agentless**. There is no daemon sitting on your servers waiting for instructions. No agent to install, maintain, update, patch, or secure on every machine you manage. Applad connects over SSH, does its work — provisions infrastructure, runs migrations, deploys functions, configures services, updates hosting — and leaves. The only requirement on the target machine is Docker (or another supported runtime). When Applad is done, it is gone from that machine until the next operation.

This is the same philosophy that made Ansible win over agent-based tools like Puppet and Chef. The operational overhead of managing agents across a fleet of servers is real, painful, and a source of its own class of bugs and security issues. Applad inherits Ansible's answer to that problem.

**What this means in practice:**

- **Zero persistent footprint on target machines** beyond what your application actually needs to run
- **No agent version mismatch issues** — a common nightmare with agent-based tools
- **Works on any machine you can SSH into** — cloud VMs, VPS, bare metal, Raspberry Pis, on-premise servers. If you can SSH in, Applad can manage it.
- **Smaller security surface** — no persistent daemon means no persistent attack surface on managed machines
- **Familiar mental model** — if your team knows Ansible, they already understand how Applad reaches out to infrastructure, does work, and disconnects
- **Internally uses familiar technologies** — Applad orchestrates Docker for containerized function runtimes, Caddy for reverse proxy and SSL, NATS or Redis for messaging, and other well-understood tools. You are never locked into Applad-proprietary runtime primitives. The containers running your functions are standard Docker containers. The proxy serving your sites is standard Caddy. Operations teams can inspect, debug, and reason about what Applad has provisioned using tools they already know.

**The agentless flow for a typical operation:**

```
Developer runs: applad deploy run android-production

1. Applad reads deployments/android-production.yaml from config tree
2. Applad opens an SSH connection to the target using the developer's SSH key
3. Applad pulls the required Docker images on the remote machine
4. Applad runs the build container, mounts the source, executes the build command
5. Applad signs the artifact using credentials fetched from the encrypted operational database
6. Applad submits to the Play Store via the configured API
7. Applad logs the full operation to the runtime database, attributed to the developer's SSH key identity
8. SSH connection closes — nothing remains on the machine except the build artifacts
```

---

## SSH Keys and Traceability — Every Change Has an Author

When a developer interacts with Applad — whether through the CLI, the admin UI, or via an instruction — their **SSH key is the identity that signs and attributes every action**. This is not just an authentication mechanism. It is the foundation of Applad's full audit trail and traceability model.

**How it works:**

When a developer registers with an Applad instance, they register their SSH public key. From that point forward:

- **CLI operations** are authenticated via SSH key. When `applad db migrate` runs, the migration is attributed to the key that initiated it.
- **Config changes pushed via `applad config push`** are signed with the developer's SSH key before being applied. The signature is stored in the audit log alongside the diff of what changed.
- **Admin UI sessions** are bootstrapped via SSH key authentication — the UI session token is tied to the key identity, meaning every action taken through the UI is ultimately attributed to the same identity as CLI operations. There is no separate "UI user" that bypasses the key-based identity model.
- **`applad instruct` operations** — when an instruction produces a change, the change is attributed to your key identity plus an instruction marker, so the audit trail shows both who authorized it and that it was AI-assisted. The exact instruction prompt is recorded alongside the change.
- **Agentless remote operations** — when Applad SSHs into a remote machine to provision, deploy, or configure, it does so using the initiating developer's key or a scoped deployment key derived from it. The remote machine's auth logs show exactly which key performed which operation and when.
- **Cloud provider operations** — when Applad calls a cloud provider API to spin up a resource, tear down a VM, or access a managed service, that API call is attributed to the developer identity that triggered the operation. Cloud provider access logs and Applad's own audit log both record the same identity.

**What the audit trail captures for every change:**

```
{
  "timestamp": "2026-02-21T10:32:14Z",
  "actor": {
    "key_fingerprint": "SHA256:abc123...",
    "key_label": "alice@macbook-pro",
    "identity": "alice@acme-corp",
    "via": "cli"                          # cli | ui | instruct | api | ci
  },
  "action": "db.migrate",
  "target": {
    "org": "acme-corp",
    "project": "mobile-app",
    "environment": "production",
    "infrastructure": "vps-prod-01.acme-corp.com"
  },
  "change": {
    "migration": "004_add_fulltext_index_to_posts.sql",
    "diff": "...",
    "config_signature": "SHA256:def456...",
    "instruction_prompt": null            # populated when via == "instruct"
  },
  "remote": {
    "host": "prod-db-01.acme-corp.com",
    "ssh_session": "session-uuid",
    "duration_ms": 1240
  }
}
```

When an instruction triggers the change:

```
{
  "actor": {
    "key_fingerprint": "SHA256:abc123...",
    "identity": "alice@acme-corp",
    "via": "instruct"
  },
  "change": {
    "files_modified": ["tables/posts.yaml"],
    "migration": "005_add_fulltext_index.sql",
    "instruction_prompt": "add fulltext search to posts"
  }
}
```

**What this enables:**

- **Full traceability** — every schema change, every deployment, every config push, every migration, every flag toggle, every cloud resource spin-up — has a named human behind it. No anonymous changes, ever.
- **Non-repudiation** — changes are signed with SSH keys. Cryptographic proof of who made each change.
- **Instruction transparency** — every AI-assisted change records the exact instruction that triggered it. Teams can audit not just what changed, but why, and who asked for it.
- **Scoped deployment keys** — for CI/CD pipelines, Applad supports generating scoped deployment keys with limited permissions. Automated actions are clearly distinguishable from human actions in the audit trail.
- **Key rotation without history loss** — new key linked to existing identity. Historical entries retain old fingerprint for forensic integrity.
- **Revocation** — when a developer leaves, their key is revoked. Historical audit entries are preserved.
- **`applad instruct` is never anonymous** — the audit entry records the developer's key identity, the instruction prompt, and every file modified or infrastructure operation triggered. AI-assisted changes are always traceable to a human who authorized them.
- **Cloud operations are attributed** — every cloud resource lifecycle event attributed in both Applad's audit log and the cloud provider's own access logs.

---

## What Lives Where — The Three-Way Separation

One of Applad's core architectural decisions is a clean, three-way separation between what belongs in config files, what belongs in the database as admin-managed operational data, and what belongs in the database as application runtime data.

**Config files — structural decisions requiring developer review, rarely changing:**

- Database schema definitions and migrations
- Table, field, and index definitions
- Permission and security rules
- Auth provider configuration
- Feature flag skeletons
- Function definitions and runtime configuration
- Workflow and automation pipeline structure
- Storage bucket definitions and access rules
- Hosting configuration and deployment pipeline structure
- Environment definitions
- Organization structure and member role definitions
- Plugin and adapter configuration
- Enabled/disabled feature toggles at the instance and project level
- API and webhook endpoint definitions
- Service integration configuration
- Secret references — pointers to environment-injected secrets, never the secrets themselves
- Messaging provider config and template references
- Security policy definitions — rate limits, CORS, CSP, IP allowlists, MFA requirements
- SSH public keys for registered developers and scoped deployment keys
- Infrastructure targets — which VPS, which cloud provider, which region, per environment

**Database (admin-managed operational data) — operational decisions made by admins or team members through the UI:**

- Feature flag targeting rules
- Messaging template content
- External webhook subscriptions
- Per-org and per-project feature enablement
- Notification and communication preferences
- Custom dashboard configurations and layouts
- Scheduled job overrides and pause states
- Store credentials and mobile signing certificates (encrypted at rest)
- AI provider API keys (encrypted at rest)
- Cloud provider credentials and access keys (encrypted at rest)
- Active IP allowlist entries
- MFA enrollment records

**Database (application runtime data) — everything generated by users and the system:**

- User records and auth sessions
- All application data — rows, documents, files
- Feature flag evaluation logs and per-user flag state
- Analytics events and aggregated metrics
- Full audit log — every config change, every SSH operation, every cloud API call, every admin action, every auth event, every instruction and its prompt, with key fingerprint, identity, timestamp, diff, and cryptographic signature
- Deployment history and build logs
- Function execution logs and traces
- OTA update adoption tracking
- Organization member records and invitations
- User-generated API keys and tokens (hashed)
- Messaging send history across all channels
- Real-time subscription state
- Queue and job execution state for workflows
- Storage file metadata and usage records
- In-app notification records
- Security event logs
- Cloud resource lifecycle logs — spin-up, usage, tear-down, cost attribution per operation

The guiding rules:

- **If it requires a developer decision and git review → config files**
- **If an admin or non-developer needs to change it through the UI without a deployment → admin-managed database**
- **If a user action or system process generated it → application runtime database**

---

## Security

Security in Applad is not a layer added on top — it is woven through every architectural decision from the ground up.

**Encryption:**

- **At rest** — all databases encrypted at rest. Secrets, signing certificates, cloud credentials, and AI API keys in the operational database are encrypted with an additional application-layer key derived from the instance secret. Database-level access alone is insufficient to read sensitive values.
- **In transit** — TLS everywhere, automatic via Caddy. No unencrypted traffic between any Applad components. Internal service-to-service communication within a cluster is mutually TLS authenticated. SSH connections use key-based auth only — password auth is disabled by design.
- **Config signatures** — every config push signed with the initiating developer's SSH key. Stored in the audit log with the full diff. Cryptographic proof of the config state at every point in time.
- **Cloud credentials in transit** — cloud provider API calls made over TLS with credentials fetched at operation time from the encrypted operational database. Never in environment variables or config files.

**Authentication and Identity:**

- **SSH key-based identity** — all developer and CI/CD interactions authenticated via SSH keys. Password-based access to Applad infrastructure disabled by design.
- **Admin UI MFA** — configurable per org, enforceable via `auth/auth.yaml`. TOTP and WebAuthn/FIDO2 hardware keys supported.
- **End-user MFA** — configurable per project
- **Brute force protection** — automatic lockout and exponential backoff on failed auth attempts
- **Token rotation** — session tokens and API keys rotatable on demand and automatically on security events
- **Argon2id password hashing** — configurable time and memory cost parameters per project
- **Short-lived SSH sessions** — ephemeral, opened for the duration of an operation and immediately closed

**Permissions and Isolation:**

- **Applad-native permission rules** — defined in `tables/*.yaml`, translated to the underlying database's enforcement mechanism
- **Row-level filtering** — permissions support filter expressions
- **Project isolation** — strict by default
- **Organization isolation** — strict. Data never crosses org boundaries.
- **Container isolation** — each function runtime in an isolated Docker container with no host filesystem access and no inter-container networking except through Applad's controlled invocation interface
- **Scoped deployment keys** — explicitly limited permissions in config. A deployment key cannot modify schema or access other projects.
- **Cloud resource isolation** — cloud resources tagged and scoped to the project that provisioned them

**Network Security:**

- **Rate limiting** — configurable per endpoint, per user, per org
- **CORS** — configurable per project and per hosting site. Defaults to restrictive.
- **CSP headers** — configurable per hosting site. Secure defaults shipped out of the box.
- **IP allowlisting** — per org or per project, managed through the admin UI
- **DDoS mitigation** — Caddy-level connection rate limiting at the edge

**Secrets Management:**

- **Secrets never in config files** — only references like `${DATABASE_URL}`
- **Secrets never in logs** — logging layer scrubs known secret patterns
- **Cloud credentials never in config files** — fetched at operation time from encrypted operational database
- **Secret rotation** — `applad secrets rotate <key>` rotates, updates references, logs the event
- **Scoped secrets** — scoped to specific project or environment

**Vulnerability Management:**

- **Container image scanning** — function images scanned before deployment. Critical vulnerabilities block deployment by default.
- **Dependency auditing** — `applad audit` checks function dependencies across all supported runtimes
- **Security event log** — failed auth, rate limit hits, blocked IPs, permission denials in a dedicated log
- **Anomaly detection** — the lad monitors the security event log and surfaces patterns indicating a security issue

**Data Residency and Compliance:**

- **Data residency** — configurable per org via the database connection in `org.yaml`
- **Data retention policies** — configurable per project
- **Right to erasure** — `applad users purge <user-id>` performs coordinated deletion across all tables
- **Export for compliance** — `applad export --user <user-id>` generates full data export for GDPR/CCPA

**Security in the Config Tree:**
Security policies live alongside the resources they protect. A developer cannot add a table without its permission rules being visible to reviewers in the same diff. Security is not a separate concern — it lives alongside the resource it governs, reviewed at the same time.

---

## Config File Structure

Applad's config is split across a directory tree of focused `.yaml` files — merged at runtime into one resolved config. The same pattern Terraform uses with `.tf` files, applied to your entire backend.

```
my-project/
├── applad.yaml                        # Root — instance config, AI, observability
├── applad.lock                        # Lock file — resolved versions, checksums
│
├── .applad/
│   ├── cache/
│   └── tmp/
│
├── orgs/
│   └── acme-corp/
│       ├── org.yaml
│       │
│       └── projects/
│           ├── mobile-app/
│           │   ├── project.yaml
│           │   ├── auth/
│           │   │   └── auth.yaml
│           │   ├── database/
│           │   │   ├── database.yaml
│           │   │   └── migrations/
│           │   │       ├── 001_create_users.sql
│           │   │       └── 002_create_posts.sql
│           │   ├── tables/
│           │   │   ├── users.yaml
│           │   │   ├── posts.yaml
│           │   │   └── comments.yaml
│           │   ├── storage/
│           │   │   ├── storage.yaml
│           │   │   ├── avatars.yaml
│           │   │   └── documents.yaml
│           │   ├── functions/
│           │   │   ├── send-welcome-message/
│           │   │   │   ├── function.yaml
│           │   │   │   └── main.dart
│           │   │   └── process-payment/
│           │   │       ├── function.yaml
│           │   │       └── index.js
│           │   ├── workflows/
│           │   │   ├── user-onboarding.yaml
│           │   │   └── payment-failed-recovery.yaml
│           │   ├── messaging/
│           │   │   ├── messaging.yaml
│           │   │   └── templates/
│           │   │       ├── welcome.yaml
│           │   │       ├── password-reset.yaml
│           │   │       └── payment-failed.yaml
│           │   ├── flags/
│           │   │   ├── new-dashboard.yaml
│           │   │   └── checkout-flow.yaml
│           │   ├── hosting/
│           │   │   ├── web.yaml
│           │   │   └── docs.yaml
│           │   ├── deployments/
│           │   │   ├── android-production.yaml
│           │   │   ├── ios-production.yaml
│           │   │   └── ota-update.yaml
│           │   ├── realtime/
│           │   │   └── realtime.yaml
│           │   ├── analytics/
│           │   │   └── analytics.yaml
│           │   └── observability/
│           │       └── observability.yaml
│           │
│           └── internal-dashboard/
│               ├── project.yaml
│               ├── auth/
│               ├── database/
│               ├── tables/
│               └── messaging/
│
└── shared/
    ├── roles/
    │   └── default-roles.yaml
    ├── messaging/
    │   └── slack-integration.yaml
    └── functions/
        └── utils/
```

**The merge rules Applad follows at startup:**

1. Load `applad.yaml` (instance root)
2. Scan `orgs/` directory, load each `org.yaml`
3. For each org, scan `projects/` directory, load each `project.yaml`
4. For each project, scan all subdirectories and load all `.yaml` files
5. Merge into a single resolved config tree in memory
6. Validate the full merged config including security policy completeness
7. Start — or error with the exact file and line that failed validation

Git conflicts are nearly impossible. Security policies always reviewed alongside the resources they protect. `applad instruct` knows exactly which file to edit for any operation.

---

## Core Architecture

Applad's core infrastructure is written in Dart — auth, database engine, storage, realtime, the CLI, and the admin Flutter app. Applad's functions and automation layer is fully polyglot — Dart, Node.js, Python, Go, PHP, Ruby, etc. Each runtime runs in an isolated Docker container. Dart is to Applad what PHP is to Appwrite — the language of the engine, not a constraint on the developer.

---

## Dart First, Pragmatic Where It Matters

Applad is not "Dart only" — it's **Dart first**. The Dart core handles the API server (built on Shelf/Dart Frog), the CLI, the admin Flutter app, configuration management, and the orchestration layer. For infrastructure-heavy pieces, Applad composes rather than reinvents:

- **Docker** — containerized function runtimes and build environments
- **Caddy** — reverse proxy, SSL, hosting layer
- **NATS or Redis** — realtime and pub/sub
- **Rclone** — storage adapters
- **Buildkit** — polyglot function runtime container builds
- **Cloud provider SDKs** — AWS, GCP, Azure adapters for on-demand resource provisioning
- **Go-based tooling** — where performance or ecosystem maturity demands it

Operations teams can inspect and reason about everything Applad provisions using tools they already know.

---

## Bidirectional Infrastructure as Code

In Applad, **the UI and config files are always the same thing** — but only for the things that belong in config. Every developer-driven action — creating a table schema, defining a permission rule, configuring a feature flag skeleton, setting an infrastructure target — immediately generates or updates the corresponding `.yaml` file or migration.

`applad instruct` changes live in config files too. When you run `applad instruct "add fulltext search to posts"`, Applad edits `tables/posts.yaml`, generates a migration, and the UI reflects it immediately. The instruction is recorded in the audit trail. The config file is the artifact.

- **UI → Config:** Every structural UI action writes config
- **Config → UI:** Every config change reflects in the UI
- **`applad instruct` → Config:** Every instruction that changes structure writes config and records the prompt in the audit trail
- **Config files stay lean** — never balloon with operational or runtime state

---

## Applad as Your Lad — AI-Powered Infrastructure Assistant

Applad is your **lad** — an active collaborator deeply integrated into every layer of the platform. In the CLI this surfaces as `applad instruct`. In the admin UI the lad has a more characterful presence — the same intelligence, a more conversational interface.

The lad reads your config files, your operational database, your live runtime data, your security event log, your logs, your analytics, and your deployment history — and acts on all three layers appropriately. Every instruction is attributed to your SSH key identity in the audit trail, with the exact prompt recorded alongside every change made.

`applad instruct` usage:

```bash
# Scaffold and configure
applad instruct "create a users table with email, name, avatar, and soft delete"
applad instruct "add fulltext search to posts"
applad instruct "set up a deployment pipeline for my Flutter app to the Play Store"
applad instruct "create a workflow that sends push and email when a post is published"
applad instruct "add a messaging template for order confirmation across email, sms and push"

# Infrastructure
applad instruct "provision a Postgres instance on AWS RDS for production"
applad instruct "spin up a cloud VM for this data processing job and tear it down when done"
applad instruct "set up staging to mirror production"
applad instruct "how much are we spending on cloud resources this month?"

# Debug and diagnose
applad instruct "why is my API error rate high?"
applad instruct "what failed in the last deployment?"
applad instruct "is this permission rule safe?"

# Flags and context
applad instruct --context logs "what failed in the last hour?"
applad instruct --context tables "suggest indexes for better query performance"
applad instruct --context security "are there any anomalous access patterns?"
applad instruct --dry-run "add fulltext search to posts"   # Show what would change without changing it
applad instruct --dry-run "provision RDS for production"  # Show the plan before executing
```

The `--dry-run` flag shows exactly what config changes, migrations, or infrastructure operations the instruction would produce — without committing to them. Essential for review, for onboarding, and for trust-building with AI-assisted infrastructure changes.

**AI Provider Agnosticism:** AI features are powered by your own API keys — stored encrypted in the admin-managed operational database, never in config files. Choose OpenAI, Gemini, Claude, Mistral, or any compatible provider. Swap through the admin UI. AI assistance is fully optional — disable it in `applad.yaml`.

---

## Instance → Organization → Project Hierarchy

Every resource belongs to a **project**. Projects belong to **organizations**. Organizations live on an **instance**. Nothing floats free of this hierarchy.

```
Instance
└── Organization (e.g. acme-corp)
    ├── Project (e.g. mobile-app)
    │   ├── Tables, Auth, Storage, Functions, Workflows
    │   ├── Messaging, Flags, Hosting, Deployments
    │   └── Realtime, Analytics, Observability, Security
    └── Project (e.g. internal-dashboard)
        ├── Tables, Auth, Storage, Functions
        └── Messaging, Security
```

Infrastructure targets — which VPS, which cloud provider, which region — defined per project and per environment in `project.yaml`. The same instance can manage projects running on a Hetzner VPS, an AWS region, and a local machine simultaneously.

One Applad instance is viable for:

- **Agencies and freelancers** — all clients as isolated orgs on one instance
- **Enterprises** — departments as orgs, each with their own infrastructure targets
- **SaaS builders** — customers as organizations
- **Managed cloud** — one fleet, many isolated tenants

---

## Database Agnosticism

No database lock-in. Adapter interface in `database/database.yaml`:

- **Relational:** PostgreSQL, MySQL, MariaDB, SQLite
- **NoSQL:** MongoDB, CouchDB, Firestore-compatible
- **Embedded/Edge:** SQLite, libSQL (Turso)
- **Managed cloud:** AWS RDS, Google Cloud SQL, Azure Database
- **Time-series or specialized:** extensible via custom adapters

Connection pooling from day one. Moving from SQLite locally to Postgres on a VPS to RDS in production is a one-line config change. Application code never changes.

---

## Tables (Not Collections)

`tables` is the universal term throughout config and CLI. Neutral, understood by everyone. Each table in its own file. Permission rules alongside the schema — always reviewed together.

---

## Messaging (Not Just Email)

`messaging` is the umbrella for all communication channels. Provider config in `messaging/messaging.yaml`. Template content in the admin-managed database.

Channels: Email, SMS, Push (FCM/APNS), In-app, Slack, Discord, Teams.

One unified SDK call. One unified CLI namespace. One unified admin panel. All events logged to the runtime database.

---

## Built-in Hosting

First-class hosting powered by Caddy. Each site in its own file under `hosting/`. Custom domains, SSL, git-connected deployments, preview environments, rollbacks, security headers — all in config, reviewed in PRs.

---

## Full Deployment Pipeline — Beyond Just Hosting

Complete deployment platform. Each pipeline in `deployments/`:

- **Mobile** — App Store and Play Store. Pipeline in config. Credentials in operational database.
- **Desktop** — Windows, macOS, Linux
- **OTA updates** — without store review cycles
- **Cloud compute jobs** — spin up, run, tear down
- **Build pipelines** — git events, schedules, or manual triggers
- **Every deployment attributed** — SSH key identity in the audit trail

---

## Feature Flags

Skeletons in `flags/*.yaml`. Targeting rules in admin-managed database — no deployment required to adjust rollouts. Evaluation logs in runtime database.

---

## Analytics

Config in `analytics/analytics.yaml`. All data in runtime database. No data sent anywhere by default. Includes security analytics, cloud resource cost attribution, and messaging delivery analytics.

---

## You Scale It Your Way

Single executable to start. The full continuum:

- **Single executable** — local dev, Raspberry Pi, internal tools
- **Docker / Docker Compose** — official images, sane defaults
- **VPS** — SSH in, Docker-based, full control, predictable costs
- **Kubernetes / Helm charts** — horizontal scaling
- **Cloud on-demand** — resources when you need them, gone when you don't
- **Managed cloud** — hosted Applad for teams that don't want to manage infrastructure

Stateless at the core — config holds structure, database holds state, process holds nothing. `applad up` anywhere with the same config and database connection produces an identical instance.

---

## No Lock-in, At Any Layer

- **Portable by design** — REST and GraphQL first, SDK second
- **Your config is yours** — `.yaml` files you own, version control, take anywhere
- **Your data is yours** — all state in your configured database
- **Your infrastructure is yours** — VPS, cloud, local, or mixed
- **Your secrets are yours** — encrypted in your database
- **No database lock-in** — swap adapters with one config line
- **No runtime lock-in** — any language for functions
- **No AI lock-in** — your keys, your provider
- **No cloud lock-in** — cloud as utility, not platform
- **No feature lock-in** — enable only what you need
- **No scaling lock-in** — local binary to Kubernetes
- **No pricing lock-in** — self-hosting is predictable. Cloud billed only when used.
- **No tool lock-in** — replaces BaaS, IaC, CI/CD, feature flags, analytics, AI assistant

---

## Auth That Doesn't Fight You

Auth config in `auth/auth.yaml`. Runtime preferences in admin-managed database. User records and auth events in runtime database.

- Multi-tenancy first-class
- Complex RBAC in config
- SSO and multiple provider linking
- Custom auth flows
- MFA enforceable per org and project
- Every auth event logged to the security event log

---

## Observability as a Core Feature

Config in `observability/observability.yaml`. Logs, traces, metrics, and security events in the runtime database. Custom dashboards in the admin-managed database.

- Structured logging and distributed tracing
- Dedicated observability panel — logs, traces, query performance, function execution, messaging delivery, security events, cloud resource usage, instruction history
- Clear actionable error messages
- Security event log separate from application log
- Cloud resource lifecycle logs with cost attribution
- Export to OpenTelemetry, Grafana, etc.
- The lad proactively surfaces issues across all layers

---

## Ship Less, Ship It Complete

Every feature that ships is documented, stable, and production-ready before the next one begins. The first 80% — tables, auth, storage, functions — are table stakes implemented well. The last 20% — production hardening, observability, security depth, custom auth flows, scaling, cloud on-demand, and deep customization — gets the same attention and polish. That's where Applad competes.

---

## Radical Configurability — Including Disabling Itself

Every major feature toggleable in `applad.yaml`. Applad is viable as:

- A full-featured BaaS with visual admin, AI assistance, and full security suite
- A headless API-only backend
- A lightweight single-binary embedded backend
- A complete mobile and web deployment platform
- A cloud resource orchestrator
- A massive horizontally scaled multi-org cluster

Nothing assumed required. Everything opt-in or opt-out.

---

## Problems We're Solving

**Vendor lock-in** → Portable config, database agnosticism, cloud as utility, no proprietary SDK patterns

**Scaling surprises** → Stateless core, connection pooling from day one, clear paths from local to cloud

**Pricing cliffs** → Single binary self-hosting, cloud billed only when used, transparent managed cloud pricing

**Limited customization** → Everything opt-in, plugins and adapters first-class, last 20% gets same polish

**Real-time complexity** → NATS/Redis handles the broker

**Auth edge cases** → Multi-tenancy, RBAC, SSO, MFA, custom flows — first-class from day one

**Debugging and observability** → Core feature. Clear errors, structured logs, security event log, cloud logs, instruction history.

**RLS complexity** → Applad-native permission rules alongside table definitions

**Security as an afterthought** → Security policies in config alongside the resources they govern, reviewed in the same PR

**No audit trail** → Every change attributed to an SSH key identity with cryptographic proof. Every instruction recorded with its prompt.

**Agent overhead** → Agentless like Ansible. SSH in, do the work, leave.

**Cloud complexity** → Cloud as utility. Spin up when needed, tear down when done.

**Maturity gaps** → Ship less, ship it complete.

**The 80/20 problem** → The last 20% is where Applad competes.

---

## Key Principles

- **Applad is your lad** — an active AI-powered collaborator for your entire stack
- **`applad instruct`** — the CLI surface of the lad. Self-documenting, professional, consistent with the CLI's imperative tone. `--dry-run` shows the plan before executing.
- **Applad is the IaC tool for your entire backend** — config files and the UI are always the same thing
- **Runs anywhere** — local, VPS, cloud on-demand, or all three at once
- **Cloud as utility, not platform** — draw from cloud providers when they make sense
- **Agentless like Ansible** — SSH in, do the work, leave
- **Internally familiar** — Docker, Caddy, NATS, Redis. No black boxes.
- **SSH keys are identity** — every change cryptographically attributed to a named developer. Every instruction records its prompt. No anonymous changes, ever.
- **Three layers, clean separation** — structural intent in config, operational state in admin database, runtime data in runtime database
- **The config tree IS the backend** — one focused file per resource
- **Security lives alongside what it protects** — reviewed in the same PR
- **Self-hosting shouldn't require a DevOps PhD** — one binary, sane defaults, point at your config directory
- **Configuration visual and code-friendly simultaneously** — always bidirectionally in sync
- **Admins and marketers are first-class operators** — no deployments required for operational changes
- **Instance → Organization → Project → Everything else** — nothing floats free of this hierarchy
- **Tables, not collections** — neutral terminology for everyone
- **Messaging, not email** — one unified channel abstraction
- **Extending it should feel native** — plugins, adapters, functions, workflows all first-class
- **Scaling is the developer's choice** — local binary to Kubernetes cluster
- **The admin UI is a Flutter app** — desktop, mobile, and web, but optional
- **Workflow automation is built-in** — no duct-taping a separate service
- **AI-assisted scaffolding** — generative, layer-aware, file-aware, always with `--dry-run` available
- **AI provider agnosticism** — your keys, your provider, encrypted in your database
- **Multi-organization support** — one instance, many isolated orgs and projects
- **Auth that grows with you** — multi-tenancy, RBAC, SSO, MFA, custom flows
- **Observability and security by default** — you and your lad know exactly why anything breaks or anything suspicious happens
- **Feature flags without a third-party tool** — skeletons in config, targeting in admin database
- **Analytics without leaving the platform** — with escape hatches when you need them
- **Deployment beyond hosting** — app stores, OTA updates, cloud compute jobs, full build pipelines
- **Portable by design** — config, data, secrets, AI keys, infrastructure. Never locked in.
- **The last 20% is where we compete**
- **Compose, don't reinvent** — Dart orchestrates, proven open source tools do the heavy lifting
- **Ship less, ship it complete**
- **One source of truth, three layers, one lad** — UI, config, or `applad instruct`. Always coherent, always correct, always attributed.

---
