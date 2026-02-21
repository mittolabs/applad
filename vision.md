Here's the fully updated vision document:

---

# Applad — Full Vision Document

## What Is Applad?

I want us to create **Applad**, an open-source BaaS inspired by Appwrite, Supabase, PocketBase, Firebase, Appsmith and Directus — with features from v0.dev, Motia, n8n and Lovable. Written in Dart for the backend core and Flutter for the admin app (enabling true cross-platform support beyond just the web).

The core idea is that it should be **very easy to self-host, extend, scale and configure** — born directly from the frustrations I experienced trying to set up and maintain tools like Appwrite and Supabase. But Applad is more than a BaaS. It is the **Infrastructure as Code tool for your entire backend** — and the visual UI is simply a friendly interface into that same system. And beyond that, Applad is your **lad** — an active, AI-powered collaborator that helps you configure, spin up, debug, and manage everything across your entire stack.

The name says it all: **App** — it helps you build and deploy apps. **Lad** — it's your assistant, your collaborator, your infrastructure companion. In the CLI this surfaces as `applad instruct` — self-documenting, professional, and consistent with the rest of the CLI's imperative tone.

---

## The Core Reframe

Most teams today run a BaaS for their backend, a separate IaC tool like Terraform for infrastructure, a separate CI/CD tool for deployments, a separate feature flag service, and a separate analytics platform. These tools don't talk to each other, drift apart over time, and require different mental models and skill sets to operate. Applad replaces all of them with a single, coherent system where **everything is config, everything is visual, everything is assisted by AI, and they are always the same thing**.

The config files are the backend. The admin UI is a lens into them. The AI is the lad that helps you work with both. Every click in the UI writes config. Every config change reflects in the UI. Every instruction you give produces real changes, not just suggestions. There is no gap, no drift, no manual translation.

---

## Deploy vs Release — A Critical Distinction

Applad treats deployment and release as fundamentally separate concerns, because they are:

**Deploy** is the technical act of putting an artifact somewhere — installing code on a server, submitting a build to the Play Store, pushing a web app to a domain, updating an OTA channel. It is low-risk, often invisible to users, and fully reversible. It is what `applad deploy` does.

**Release** is the business decision to make functionality live to users. It carries user-facing risk, requires judgment about timing and audience, and is controlled through feature flags. A deployment can happen without any users seeing the new functionality. The flag releases it to them — gradually, to a subset, at a scheduled time, or all at once. These are separate decisions made by separate people at separate times.

This distinction is encoded throughout Applad:

- `deployments/` config files define pipelines for putting artifacts somewhere
- `flags/` config files define the skeletons for releasing functionality to users
- `applad deploy run` triggers a deployment — technical, attributed to a developer's SSH key
- Feature flag targeting rules — managed by product managers through the admin UI — control the release
- The two never touch each other, because they shouldn't

This is not academic. It is what makes safe, continuous delivery possible. Code goes to production continuously via deployments. Users see new features when the team is ready to release them via flags. Testing in production, dark launches, canary releases, instant rollbacks — all of this falls out naturally when deploy and release are kept separate.

---

## Where Applad Runs — Local, VPS, and Cloud On-Demand

Applad is designed to run anywhere you can point it — and to move between those places without changing your application code, your config structure, or your mental model. There are three deployment contexts, and they form a natural continuum:

**Local:**
Your laptop, a home server, a Raspberry Pi. Single binary, SQLite by default, zero external dependencies. This is where you develop, prototype, and run internal tools. Applad running locally is indistinguishable from Applad running in production from the application's perspective — same config tree, same CLI, same admin UI, same behavior. The only difference is the machine it's on. A developer can run the full stack locally with a single command, work entirely offline, and push to a VPS or cloud when ready.

**VPS:**
A DigitalOcean Droplet, Hetzner server, Linode, bare metal — any machine you own or rent at a flat rate. You have full control, predictable costs, and no vendor dependency at the infrastructure level. Applad connects over SSH, provisions what it needs using Docker, manages everything from your config tree, and leaves. This is the sweet spot for most indie developers, small teams, startups, and anyone who wants full ownership without managing cloud complexity. A single mid-range VPS can comfortably run a production Applad instance serving thousands of users.

**Cloud providers on-demand:**
AWS, GCP, Azure, and others — but used surgically, not as a platform commitment. Applad treats cloud provider resources as adapters you draw from when they make sense for a specific resource. You might run your core app on a Hetzner VPS because it's fast and cheap, use S3 for storage because the pricing is right, use SES for high-volume email at scale, and spin up a cloud compute instance for a heavy data processing job or an iOS build — paying only for the duration, then tearing it down. Applad connects to cloud provider APIs the same way it connects to any other infrastructure — over your configured credentials, agentlessly, doing the work and leaving.

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

- Storage can be local filesystem today, S3 tomorrow, R2 next month — application code never changes
- Database can be SQLite locally, Postgres on a VPS in staging, RDS in production — same config structure, different adapter target
- Functions can run in Docker containers on your VPS or burst to cloud compute for heavy workloads
- Email can go through SMTP on a VPS, switch to SES at scale — one line change in `messaging/messaging.yaml`
- iOS builds that require macOS can spin up an AWS Mac instance, build, and tear it down — paying only for the build duration

---

## Agentless — Like Ansible, Powered by Familiar Tools

Applad is **agentless**. There is no daemon sitting on your servers waiting for instructions. No agent to install, maintain, update, patch, or secure on every machine you manage. Applad connects over SSH, does its work — provisions infrastructure, runs migrations, deploys functions, configures services, runs deployments — and leaves. The only requirement on the target machine is Docker. When Applad is done, it is gone from that machine until the next operation.

This is the same philosophy that made Ansible win over agent-based tools like Puppet and Chef. The operational overhead of managing agents across a fleet of servers is real, painful, and a source of its own class of bugs and security issues. Applad inherits Ansible's answer to that problem.

**What this means in practice:**

- **Zero persistent footprint on target machines** beyond what your application actually needs to run
- **No agent version mismatch issues** — a common nightmare with agent-based tools
- **Works on any machine you can SSH into** — cloud VMs, VPS, bare metal, Raspberry Pis, on-premise servers
- **Smaller security surface** — no persistent daemon means no persistent attack surface on managed machines
- **Internally uses familiar technologies** — Docker for containerized function runtimes and build environments, Caddy for reverse proxy and SSL, NATS or Redis for messaging. Operations teams can inspect and reason about everything Applad provisions using tools they already know. Nothing is a black box.

**The agentless flow for a typical deployment:**

```
Developer runs: applad deploy run android-production

1. Applad reads deployments/android-production.yaml from config tree
2. Applad opens an SSH connection to the build VPS using the developer's SSH key
3. Applad pulls the required Docker images on the remote machine
4. Applad runs the build container, mounts the source, executes the build command
5. Applad fetches signing credentials from the encrypted operational database
6. Applad signs the artifact and submits to the Play Store via API
7. Applad logs the full operation to the runtime database, attributed to the developer's SSH key
8. SSH connection closes — nothing remains on the machine except the build artifact
9. When the team is ready to release the new version to users, a product manager
   adjusts the staged rollout targeting rule in the admin UI — no developer required
```

---

## SSH Keys and Traceability — Every Change Has an Author

When a developer interacts with Applad — whether through the CLI, the admin UI, or via `applad instruct` — their **SSH key is the identity that signs and attributes every action**. This is not just an authentication mechanism. It is the foundation of Applad's full audit trail and traceability model.

**How it works:**

When a developer registers with an Applad instance, they register their SSH public key. From that point forward:

- **CLI operations** are authenticated via SSH key. When `applad db migrate` runs, the migration is attributed to the key that initiated it.
- **Config changes via `applad config push`** are signed with the developer's SSH key. The signature is stored in the audit log alongside the full diff of what changed.
- **Admin UI sessions** are bootstrapped via SSH key authentication. Every UI action is attributed to the same identity as CLI operations. There is no separate "UI user" that bypasses the key-based identity model.
- **`applad instruct` operations** — the change is attributed to your key identity plus an instruction marker. The exact prompt is recorded in the audit trail alongside every file modified, migration generated, or infrastructure operation triggered.
- **Agentless remote operations** — when Applad SSHs into a remote machine, it does so using the initiating developer's key or a scoped deployment key. The remote machine's auth logs show exactly which key performed which operation and when.
- **Cloud provider operations** — cloud API calls are attributed to the developer identity in both Applad's audit log and the cloud provider's own access logs.
- **Deployment operations** — every `applad deploy run` is attributed to the initiating SSH key identity. The audit trail records which pipeline ran, which artifact was produced, which infrastructure was used, and who triggered it.

**What the audit trail captures:**

```
{
  "timestamp": "2026-02-21T10:32:14Z",
  "actor": {
    "key_fingerprint": "SHA256:abc123...",
    "key_label": "alice@macbook-pro",
    "identity": "alice@acme-corp",
    "via": "cli"                     # cli | ui | instruct | api | ci
  },
  "action": "deployments.run",
  "target": {
    "org": "acme-corp",
    "project": "mobile-app",
    "deployment": "android-production",
    "type": "play-store",
    "environment": "production"
  },
  "change": {
    "artifact": "app-release-v2.1.0.aab",
    "config_signature": "SHA256:def456...",
    "instruction_prompt": null        # Populated when via == "instruct"
  },
  "remote": {
    "host": "build.acme-corp.com",
    "ssh_session": "session-uuid",
    "duration_ms": 84200
  }
}
```

**What this enables:**

- **Full traceability** — every schema change, every deployment of every type, every config push, every migration, every flag toggle, every cloud resource spin-up — has a named human behind it. No anonymous changes, ever.
- **Non-repudiation** — cryptographic proof of who made each change via SSH key signatures
- **Instruction transparency** — every `applad instruct` action records the exact prompt that triggered it alongside every change made
- **Deploy/release traceability** — deployments are attributed to developers via SSH keys. Flag targeting rule changes are attributed to whoever changed them in the admin UI. The two audit trails are separate and clear.
- **Scoped deployment keys** — CI/CD pipelines use scoped keys with limited permissions. Automated actions are clearly distinguishable from human actions in the audit trail.
- **Key rotation without history loss** — new key linked to existing identity. Historical entries retain the old fingerprint.
- **Revocation** — when a developer leaves, their key is revoked. Pending operations rejected. Historical entries preserved.

---

## .env.example — Auto-Generated, Always In Sync

Applad eliminates one of the most persistent sources of developer friction — figuring out which environment variables are needed and where. Every `${VAR_NAME}` reference across the entire config tree is automatically extracted and placed into a scoped `.env.example` file that mirrors the config tree structure.

**How it works:**

Applad scans every `.yaml` file in the tree and extracts every `${VAR}` reference. It places each variable into the `.env.example` at the scope where it is first meaningfully referenced — instance-level vars at the root, org-level vars in `orgs/<org>/.env.example`, project-level vars in `orgs/<org>/projects/<project>/.env.example`. Variables used across multiple modules within a project roll up to the project-level file. Variables used across multiple projects roll up to the org level.

Each `.env.example` is annotated — not just a list of empty keys but a documented file that tells you what each variable is for, which config file uses it, what format it expects, and whether it should go through `applad secrets set` rather than a `.env` file in production.

**The scoped structure:**

```
my-project/
├── .env.example                     # Instance-level vars only — APPLAD_SECRET, OTEL_ENDPOINT
├── .env                             # Never committed — auto-gitignored by applad init
│
├── orgs/
│   └── acme-corp/
│       ├── .env.example             # Org-level vars if any
│       ├── .env                     # Never committed
│       │
│       └── projects/
│           └── mobile-app/
│               ├── .env.example     # All project vars — database, messaging, auth, storage
│               └── .env             # Never committed
```

**Key behaviors:**

- **Auto-generated, never manually edited** — `applad env generate` regenerates from the config tree. Adding a new `${VAR}` reference to any yaml file means the next `applad env generate` picks it up, annotates it with which file uses it, and places it at the right scope.
- **Always gitignored for `.env`** — `applad init` writes `.gitignore` entries for all `.env` files across the entire tree automatically. `.env.example` files are always committed.
- **Validated on startup** — `applad up` checks that all referenced `${VAR}` values are present before starting. Missing variables produce a clear error: `Missing required variable STRIPE_SECRET — used by functions/process-payment/function.yaml`
- **Environment-aware** — `applad env generate --env production` generates a `.env.example` containing only the variables needed for the production environment, skipping development-only overrides
- **Secret classification** — variables that reference credentials are annotated with a note pointing to `applad secrets set`, distinguishing between variables safe for `.env` files and those that should go into the encrypted operational database in production
- **Across organizations** — each org and project has its own `.env.example` scoped to exactly what it needs. A developer onboarding to `mobile-app` under `acme-corp` only sees that project's variables — not variables for `internal-dashboard` or any other org.

**Onboarding a new developer becomes:**

```bash
git clone github.com/myorg/myapp-infra
cp orgs/acme-corp/projects/mobile-app/.env.example \
   orgs/acme-corp/projects/mobile-app/.env
# Fill in values — every variable is annotated with what it's for
applad up
```

Everything they need is documented in the `.env.example`. Nothing is missing. Nothing is a mystery.

---

## What Lives Where — The Three-Way Separation

One of Applad's core architectural decisions is a clean, three-way separation between what belongs in config files, what belongs in the database as admin-managed operational data, and what belongs in the database as application runtime data.

**Config files — structural decisions requiring developer review, rarely changing:**

- Database schema definitions and migrations
- Table, field, and index definitions
- Permission and security rules
- Auth provider configuration
- Feature flag skeletons — that a flag exists, its variants, its default state, its environments
- Function definitions and runtime configuration
- Workflow and automation pipeline structure
- Storage bucket definitions and access rules
- Deployment pipeline definitions — web, mobile, desktop, OTA
- Environment definitions and infrastructure targets
- Organization structure and member role definitions
- Plugin and adapter configuration
- Enabled/disabled feature toggles at instance and project level
- API and webhook endpoint definitions
- Service integration configuration
- Secret references — pointers to environment-injected secrets, never the secrets themselves
- Messaging provider config and template references
- Security policy definitions — rate limits, CORS, CSP, IP allowlists, MFA requirements
- SSH public keys for registered developers and scoped deployment keys
- Infrastructure targets — which VPS, which cloud provider, which region, per environment
- `.env.example` files — auto-generated from `${VAR}` references, always committed

**Database (admin-managed operational data):**

- Feature flag targeting rules — who sees which variant, rollout percentages, scheduling
- Messaging template content — actual copy, HTML, and variables for all channels
- External webhook subscriptions
- Per-org and per-project feature enablement
- Custom dashboard configurations and layouts
- Scheduled job overrides and pause states
- Store credentials and mobile signing certificates (encrypted at rest)
- AI provider API keys (encrypted at rest)
- Cloud provider credentials and access keys (encrypted at rest)
- Active IP allowlist entries
- MFA enrollment records

**Database (application runtime data):**

- User records and auth sessions
- All application data — rows, documents, files
- Feature flag evaluation logs and per-user flag state
- Analytics events and aggregated metrics
- Full audit log — every config change, every SSH operation, every cloud API call, every deployment of every type, every admin action, every auth event, every `applad instruct` prompt and its changes, with key fingerprint, identity, timestamp, diff, and cryptographic signature
- Deployment history and build logs — web, mobile, desktop, OTA all in one place
- Function execution logs and traces
- OTA update adoption tracking and gradual rollout state
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

- **Developer decision requiring git review → config files**
- **Admin or non-developer changes through UI without deployment → admin-managed database**
- **Generated by user actions or system processes → application runtime database**

---

## Security

Security in Applad is woven through every architectural decision from the ground up. The agentless model, the SSH key identity system, the three-way data separation, and the config-as-code approach all have direct security benefits.

**Encryption:**

- All databases encrypted at rest. Secrets, signing certificates, cloud credentials, and AI API keys in the operational database are encrypted with an additional application-layer key derived from the instance secret — database-level access alone is insufficient to read sensitive values.
- TLS everywhere, automatic via Caddy. SSH connections use key-based auth only — password auth is disabled by design.
- Every config push signed with the initiating developer's SSH key — cryptographic proof of the config state at every point in time.
- Cloud provider API calls made over TLS with credentials fetched at operation time from the encrypted operational database. Never in environment variables or config files.

**Authentication and Identity:**

- SSH key-based identity for all developer and CI/CD interactions. Password-based access disabled.
- Admin UI MFA — TOTP and WebAuthn/FIDO2. Configurable per org, enforceable via `auth/auth.yaml`.
- Brute force protection — automatic lockout and exponential backoff.
- Argon2id password hashing with configurable cost parameters.
- Short-lived SSH sessions — ephemeral, opened for the duration of an operation and immediately closed.

**Permissions and Isolation:**

- Applad-native permission rules in `tables/*.yaml`, translated to the underlying database's enforcement mechanism.
- Row-level filtering — permissions support filter expressions.
- Strict project and organization isolation by default.
- Container isolation — each function runtime in an isolated Docker container with no host filesystem access.
- Scoped deployment keys — CI/CD keys cannot modify schema or access other projects.
- Cloud resource isolation — resources tagged and scoped to the project that provisioned them.

**Network Security:**

- Rate limiting configurable per endpoint, per user, per org.
- CORS configurable per project and per deployment. Defaults to restrictive.
- CSP headers configurable per web deployment. Secure defaults shipped.
- IP allowlisting per org or project, managed through admin UI.
- DDoS mitigation via Caddy-level connection rate limiting.

**Secrets Management:**

- Secrets never in config files — only `${VAR}` references.
- Secrets never in logs — logging layer scrubs known secret patterns.
- `.env` files auto-gitignored. `.env.example` annotates which vars are secrets and points to `applad secrets set`.
- Cloud credentials never in config files — fetched at operation time from encrypted operational database.
- Secret rotation via `applad secrets rotate <key>`.

**Vulnerability Management:**

- Container image scanning before deployment. Critical vulnerabilities block by default.
- Dependency auditing via `applad audit`.
- Dedicated security event log separate from the general application log.
- Anomaly detection monitored by the lad — unusual `applad instruct` volume, impossible travel, repeated permission denials.

**Data Residency and Compliance:**

- Configurable per org via the database connection in `org.yaml`.
- Right to erasure via `applad auth users purge <user-id>`.
- GDPR/CCPA data export via `applad auth export --user <user-id>`.

**Security in the config tree:**
Security policies live alongside the resources they protect. Permission rules in `tables/*.yaml`. Auth security in `auth/auth.yaml`. Rate limits in `observability/observability.yaml`. A developer cannot add a table without its permission rules being visible to reviewers in the same diff. Security is reviewed at the same time as the resource it governs.

---

## Config File Structure

Applad's config is split across a directory tree of focused `.yaml` files — merged at runtime into one resolved config. The same pattern Terraform uses with `.tf` files, applied to your entire backend. `.env.example` files are generated automatically and live alongside the config files they document.

```
my-project/
├── applad.yaml                        # Root — instance config, AI, observability
├── applad.lock                        # Lock file — resolved versions, checksums
├── .env.example                       # Instance-level vars — auto-generated
├── .env                               # Never committed — auto-gitignored
├── .gitignore                         # Auto-written by applad init
│
├── .applad/
│   ├── cache/
│   └── tmp/
│
├── orgs/
│   └── acme-corp/
│       ├── org.yaml
│       ├── .env.example               # Org-level vars — auto-generated
│       ├── .env                       # Never committed
│       │
│       └── projects/
│           ├── mobile-app/
│           │   ├── project.yaml
│           │   ├── .env.example       # All project vars — auto-generated, annotated
│           │   ├── .env               # Never committed
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
│           │   │   ├── process-payment/
│           │   │   │   ├── function.yaml
│           │   │   │   └── index.js
│           │   │   └── daily-report/
│           │   │       ├── function.yaml
│           │   │       └── daily.py
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
│           │   ├── deployments/       # Unified — web, mobile, desktop, OTA
│           │   │   ├── web.yaml       # type: web
│           │   │   ├── docs.yaml      # type: web
│           │   │   ├── android-production.yaml  # type: play-store
│           │   │   ├── ios-production.yaml      # type: app-store
│           │   │   └── ota.yaml       # type: ota
│           │   ├── realtime/
│           │   │   └── realtime.yaml
│           │   ├── analytics/
│           │   │   └── analytics.yaml
│           │   └── observability/
│           │       └── observability.yaml
│           │
│           └── internal-dashboard/
│               ├── project.yaml
│               ├── .env.example       # Scoped to this project only
│               ├── .env
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
7. Validate all `${VAR}` references are satisfied — fail fast with clear errors
8. Start — or error with the exact file and line that failed

Git conflicts are nearly impossible — two developers adding different tables edit different files. Security policies always reviewed alongside the resources they protect. `.env.example` files are always in sync with the config that generated them. `applad instruct` knows exactly which file to create or edit for any operation.

---

## Core Architecture

Applad's core infrastructure is written in Dart — auth, database engine, storage, realtime, the CLI, and the admin Flutter app. Applad's functions and automation layer is fully polyglot — Dart, Node.js, Python, Go, PHP, Ruby, etc. Each runtime runs in an isolated Docker container. Dart is to Applad what PHP is to Appwrite — the language of the engine, not a constraint on the developer.

---

## Dart First, Pragmatic Where It Matters

Applad is not "Dart only" — it's **Dart first**. For infrastructure-heavy pieces, Applad composes rather than reinvents:

- **Docker** — containerized function runtimes, build environments, and deployment pipelines
- **Caddy** — reverse proxy, SSL, web deployment hosting layer
- **NATS or Redis** — realtime and pub/sub
- **Rclone** — storage adapters across dozens of providers
- **Buildkit** — polyglot function runtime container builds
- **Cloud provider SDKs** — AWS, GCP, Azure adapters for on-demand resource provisioning
- **Go-based tooling** — where performance or ecosystem maturity demands it

Operations teams can inspect and reason about everything Applad provisions using tools they already know. Standard Docker containers. Standard Caddy config. Nothing is a black box.

---

## Bidirectional Infrastructure as Code

In Applad, **the UI and config files are always the same thing** — but only for the things that belong in config. Every developer-driven action — creating a table, defining a permission rule, setting up a deployment pipeline, configuring a web hosting site, adding an auth provider — immediately generates or updates the corresponding `.yaml` file and regenerates the relevant `.env.example` if new `${VAR}` references were introduced.

This works in both directions:

- **UI → Config:** Create a web deployment visually and `deployments/web.yaml` appears in your repo. Set up an Android pipeline and `deployments/android-production.yaml` is written. Add an OAuth provider and `auth/auth.yaml` is updated and `.env.example` gains `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` with annotations.
- **Config → UI:** Edit any config file, merge a PR, and the UI reflects it immediately.
- **`applad instruct` → Config:** Every instruction that changes structure writes config, updates `.env.example` if needed, and records the prompt in the audit trail.

Admin-managed operational data — flag targeting rules, messaging template content, cloud credentials — lives in the database. Runtime data is live data. Config files stay lean and never balloon with operational or runtime state.

---

## Applad as Your Lad — AI-Powered Infrastructure Assistant

Applad is your **lad** — an active collaborator deeply integrated into every layer of the platform. In the CLI this surfaces as `applad instruct`. In the admin UI the lad has a more characterful presence — the same intelligence, a more conversational interface.

The lad reads your config files, your operational database, your live runtime data, your security event log, your logs, your analytics, and your deployment history — and acts on all three layers appropriately. Every instruction is attributed to your SSH key identity in the audit trail, with the exact prompt recorded alongside every change made.

`applad instruct` is:

- **Self-documenting** — the verb tells you exactly what it does. You are instructing Applad to do something.
- **Professional in context** — reads cleanly in documentation, CI logs, and team runbooks
- **Consistent with the CLI's tone** — `applad deploy run`, `applad db migrate`, `applad config push`, `applad instruct`
- **Always `--dry-run` capable** — show the full plan — config files that would change, migrations that would be generated, infrastructure that would be provisioned — without executing anything
- **Always attributed** — the audit trail records the developer's key identity, the exact prompt, and every change made

```bash
# Scaffold and configure
applad instruct "create a users table with email, name, avatar, and soft delete"
applad instruct "add fulltext search to posts"
applad instruct "set up a Play Store deployment pipeline for my Flutter app"
applad instruct "set up a web deployment for myapp.com"
applad instruct "add rate limiting to the payments endpoint"

# Infrastructure
applad instruct "provision a Postgres instance on AWS RDS for production"
applad instruct "set up staging to mirror production on a Hetzner VPS"
applad instruct "spin up a cloud Mac for the iOS build and tear it down when done"

# Debug and diagnose
applad instruct "why is my API error rate high?"
applad instruct "what failed in the last web deployment?"
applad instruct "why was the Play Store submission rejected?"
applad instruct "is this permission rule safe?"

# Context flags and dry run
applad instruct --context logs "what failed in the last hour?"
applad instruct --context deployments "why did the iOS build fail?"
applad instruct --dry-run "set up a Play Store deployment pipeline"
applad instruct --dry-run "provision RDS for production"
```

When `applad instruct` creates a new deployment pipeline, it creates the right file in `deployments/` with the right `type` field, updates the project's `.env.example` if new variables are introduced, and records the exact prompt in the audit trail. When it provisions a cloud resource, it attributes the operation to your SSH key in both Applad's audit log and the cloud provider's access logs.

**AI Provider Agnosticism:** AI features are powered by your own API keys — stored encrypted in the admin-managed operational database, never in config files. Choose OpenAI, Gemini, Claude, Mistral, or any compatible provider. Swap through the admin UI. AI assistance is fully optional — disable it in `applad.yaml`.

---

## Instance → Organization → Project Hierarchy

Every resource belongs to a **project**. Projects belong to **organizations**. Organizations live on an **instance**. Nothing floats free of this hierarchy. Every `.env.example` is scoped to its level in this hierarchy.

```
Instance
└── Organization (e.g. acme-corp)
    ├── Project (e.g. mobile-app)
    │   ├── Tables, Auth, Storage, Functions, Workflows
    │   ├── Messaging, Flags, Deployments (all types)
    │   └── Realtime, Analytics, Observability, Security
    └── Project (e.g. internal-dashboard)
        ├── Tables, Auth, Storage, Functions
        └── Messaging, Security
```

Infrastructure targets — which VPS, which cloud provider, which region — are defined per project and per environment in `project.yaml`. The mobile-app project's production environment can target a Hetzner VPS with AWS cloud adapters for storage and email. Its iOS builds target an AWS Mac instance that is provisioned, used, and torn down per build. Its staging environment targets a smaller VPS. All of this is config. None of it is code.

One Applad instance is viable for:

- **Agencies and freelancers** — all clients as isolated orgs, each with their own infrastructure targets, deployment pipelines, and `.env.example` files
- **Enterprises** — departments as orgs with their own cloud accounts and VPS infrastructure
- **SaaS builders** — customers as organizations
- **Managed cloud** — one fleet, many isolated tenants

---

## Database Agnosticism

No database lock-in. Adapter interface in `database/database.yaml` per project:

- **Relational:** PostgreSQL, MySQL, MariaDB, SQLite
- **NoSQL:** MongoDB, CouchDB, Firestore-compatible
- **Embedded/Edge:** SQLite, libSQL (Turso)
- **Managed cloud:** AWS RDS, Google Cloud SQL, Azure Database
- **Time-series or specialized:** extensible via custom adapters

Connection pooling from day one. Moving from SQLite locally to Postgres on a VPS to RDS in production is a one-line config change. `applad env generate` automatically picks up any new `${VAR}` references introduced by the adapter change and updates `.env.example` with annotations. Application code never changes.

---

## Tables (Not Collections)

`tables` is the universal term for data structures throughout config and CLI, regardless of the underlying adapter. Neutral, understood by everyone — SQL or not. Each table in its own file under `tables/`. Permission rules alongside the schema — always reviewed in the same diff, by the same people, at the same time.

---

## Messaging (Not Just Email)

`messaging` is the umbrella for all communication channels. Provider config in `messaging/messaging.yaml`. Template content in the admin-managed database so non-developers can edit copy without a git commit or deployment.

Channels: Email (Resend, SMTP, SendGrid, SES), SMS (Twilio, Vonage, Africa's Talking), Push (FCM, APNS), In-app, Slack, Discord, Teams.

One unified SDK call. One unified `applad messaging` CLI namespace. One unified admin panel monitoring delivery across all channels. All messaging events logged to the runtime database. All provider API keys referenced via `${VAR}` in `messaging/messaging.yaml` — documented and annotated in the project's `.env.example`.

---

## Deployments — Unified Across All Types

Applad uses a single `deployments/` directory and a single `applad deploy` CLI namespace for all deployment types. There is no separate "hosting" concept. A web deployment to a domain and an Android deployment to the Play Store are both deployments — they both put an artifact somewhere, they both have a pipeline definition in config, they both produce build history in the runtime database, and they are both triggered with `applad deploy run <name>`.

Each deployment file has a `type` field:

- **`web`** — deploys a static or dynamic site to a domain via Caddy. Automatic SSL. Git-connected. Preview environments per PR. Security headers in config. Triggered on git push or manually.
- **`play-store`** — builds, signs, and submits an Android app to the Google Play Store. Build infrastructure defined in config. Signing credentials in the encrypted operational database. Track configurable (internal/alpha/beta/production).
- **`app-store`** — builds, signs, and submits an iOS app to the Apple App Store. iOS builds that require macOS spin up an AWS Mac instance, build, and tear it down — paying only for the build duration. Signing credentials in the operational database.
- **`desktop`** — packages and distributes Windows, macOS, and Linux applications.
- **`ota`** — pushes over-the-air updates to existing Flutter or React Native installs without going through store review. Gradual rollout with adoption tracking in the runtime database. Can be paused mid-rollout.

**The deploy/release distinction is encoded in the type system:**

A `web` deployment puts new code on the server. A `play-store` deployment puts a new build in the store. Neither of these releases new functionality to users — that is the job of feature flags. This is not a convention. It is the architecture. `deployments/` contains pipelines. `flags/` contains release controls. They are separate directories, separate config files, separate CLI namespaces, separate concerns.

OTA updates are a partial exception — they do reach existing users directly — which is why OTA deployments have their own rollout controls:

```bash
applad deploy ota status ota          # Current rollout percentage and adoption
applad deploy ota pause ota           # Pause a gradual rollout mid-flight
applad deploy ota resume ota          # Resume a paused rollout
applad deploy ota rollback ota        # Force all devices to previous version
```

Every deployment of every type is attributed to the SSH key identity that triggered it in the audit trail. `applad audit list --action deployments.run` shows the full history of every deployment across every type.

---

## Feature Flags

Feature flag skeletons live in individual files under `flags/` — version controlled like code. Targeting rules live in the admin-managed database so product managers can adjust rollouts without a deployment. Evaluation logs, per-user flag state, and experiment results live in the runtime database.

**The clean separation:**

- Flag definitions are reviewed and deployed like code — `flags/new-dashboard.yaml` tells you the flag exists and what its environments default to
- Flag targeting is operated by non-developers through the admin UI without git
- Flag runtime behavior is observable as live analytics data
- Rolling back a flag definition is a git revert
- Pausing a rollout is a UI toggle
- Neither touches the other

This is how deploy and release stay separate in practice. A developer deploys new dashboard code continuously. A product manager releases it to 10% of users when ready. Increases to 50% when metrics look good. Rolls back instantly if something is wrong. No deployment required for any of this.

---

## Analytics

Config in `analytics/analytics.yaml`. All raw events and metrics in the runtime database under your control. No data sent anywhere by default. Custom dashboard layouts in the admin-managed database. Deployment events captured across all types — web, mobile, desktop, OTA — in one place.

- API analytics, function analytics, messaging analytics, security analytics
- Cloud resource cost attribution per operation
- OTA update adoption tracking
- Deployment history analytics across all pipeline types
- Export to Mixpanel, Amplitude, BigQuery, etc. when you outgrow the built-in layer

Your lad monitors live analytics and the security event log proactively, surfacing anomalies before they become incidents.

---

## You Scale It Your Way

Single executable to start. The full continuum:

- **Single executable** — local dev, Raspberry Pi, internal tools, offline environments
- **Docker / Docker Compose** — official images, sane defaults
- **VPS** — SSH in, Docker-based, full control, predictable flat-rate costs
- **Kubernetes / Helm charts** — official Helm charts for horizontal scaling
- **Cloud on-demand** — resources when you need them, gone when you don't, billed only for duration
- **Managed cloud** — hosted Applad for teams that don't want to manage infrastructure

Stateless at the core — config holds structure, database holds state, process holds nothing. `applad up` anywhere with the same config tree and database connection produces an identical instance.

---

## No Lock-in, At Any Layer

- **Portable by design** — REST and GraphQL first, SDK second
- **Your config is yours** — `.yaml` files you own, version control, and take anywhere
- **Your data is yours** — all state in your configured database
- **Your infrastructure is yours** — VPS, cloud, local, or mixed. Applad manages it, you own it.
- **Your secrets are yours** — encrypted in your database, documented in `.env.example`, never in third-party vaults
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

Auth config in `auth/auth.yaml`. Runtime preferences in admin-managed database. User records and auth events in runtime database. All OAuth and SAML provider secrets referenced via `${VAR}` and documented in `.env.example`.

- Multi-tenancy first-class
- Complex RBAC in config
- SSO and multiple provider linking
- Custom auth flows
- MFA enforceable per org and project
- Every auth event logged to the security event log

---

## Observability as a Core Feature

Config in `observability/observability.yaml`. Logs, traces, metrics, and security events in runtime database. Custom dashboards in admin-managed database.

- Structured logging and distributed tracing across all modules
- Dedicated observability panel — logs, traces, query performance, function execution, messaging delivery, security events, cloud resource usage, deployment history across all types, `applad instruct` history
- Clear actionable error messages
- Security event log separate from application log
- Cloud resource lifecycle logs with cost attribution
- Deployment event logs across all pipeline types
- Export to OpenTelemetry, Grafana, etc.
- Your lad proactively surfaces issues across all layers

---

## Ship Less, Ship It Complete

Every feature that ships is documented, stable, and production-ready before the next one begins. The first 80% — tables, auth, storage, functions — are table stakes implemented well. The last 20% — production hardening, observability, security depth, `.env.example` generation, deploy/release separation, custom auth flows, scaling, cloud on-demand — gets the same attention and polish. That's where Applad competes.

---

## Radical Configurability — Including Disabling Itself

Every major feature toggleable in `applad.yaml`. Applad is viable as:

- A full-featured BaaS with visual admin, AI assistance, full security suite, and complete deployment platform
- A headless API-only backend configured through code and CLI
- A lightweight single-binary embedded backend
- A complete deployment platform for web, mobile, desktop, and OTA
- A cloud resource orchestrator with on-demand provisioning
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

**Debugging and observability** → Core feature. Clear errors, structured logs, security event log, deployment logs, instruct history.

**RLS complexity** → Applad-native permission rules alongside table definitions

**Security as an afterthought** → Security policies in config alongside the resources they govern, reviewed in the same PR

**No audit trail** → Every change attributed to an SSH key identity with cryptographic proof. Every instruct prompt recorded. Every deployment attributed.

**Deploy/release conflation** → Deploy is technical and goes through `applad deploy`. Release is a business decision and goes through feature flags. Never mixed.

**Environment variable chaos** → `.env.example` auto-generated from config, scoped per org and project, annotated with which file uses each var and whether it's a secret. Always in sync. Always committed.

**Agent overhead** → Agentless like Ansible. SSH in, do the work, leave.

**Cloud complexity** → Cloud as utility. Spin up when needed, tear down when done.

**iOS build infrastructure** → Spin up an AWS Mac instance per build, tear it down when done. Pay only for build duration. No persistent macOS infrastructure to maintain.

**Maturity gaps** → Ship less, ship it complete.

**The 80/20 problem** → The last 20% is where Applad competes.

---

## Key Principles

- **Applad is your lad** — an active AI-powered collaborator for your entire stack
- **`applad instruct`** — the CLI surface of the lad. Self-documenting, professional, `--dry-run` always available, every prompt recorded in the audit trail
- **Applad is the IaC tool for your entire backend** — config files and the UI are always the same thing
- **Deploy ≠ Release** — `applad deploy` puts artifacts somewhere. Feature flags release functionality to users. These are separate concerns, separate commands, separate config directories, separate people.
- **Runs anywhere** — local, VPS, cloud on-demand, or all three at once
- **Cloud as utility, not platform** — draw from cloud providers when they make sense
- **Agentless like Ansible** — SSH in, do the work, leave
- **Internally familiar** — Docker, Caddy, NATS, Redis. No black boxes.
- **SSH keys are identity** — every change cryptographically attributed to a named developer. Every instruct prompt recorded. No anonymous changes, ever.
- **`.env.example` is always generated, always annotated, always in sync** — onboarding a new developer is `cp .env.example .env`, fill in values, `applad up`
- **Three layers, clean separation** — structural intent in config, operational state in admin database, runtime data in runtime database
- **The config tree IS the backend** — one focused file per resource, directory structure encodes the hierarchy
- **Security lives alongside what it protects** — reviewed in the same PR
- **Self-hosting shouldn't require a DevOps PhD** — one binary, sane defaults, point at your config directory
- **Configuration visual and code-friendly simultaneously** — always bidirectionally in sync
- **Admins and marketers are first-class operators** — no deployments for operational changes
- **Instance → Organization → Project → Everything else** — nothing floats free of this hierarchy
- **Tables, not collections** — neutral terminology for everyone
- **Messaging, not email** — one unified channel abstraction
- **Deployments unified** — `deployments/` covers web, mobile, desktop, and OTA. `applad deploy` covers all of them.
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
- **Complete deployment platform** — web to domain, mobile to app stores, desktop distribution, OTA updates, cloud compute jobs
- **Portable by design** — config, data, secrets, AI keys, infrastructure. Never locked in.
- **The last 20% is where we compete**
- **Compose, don't reinvent** — Dart orchestrates, proven open source tools do the heavy lifting
- **Ship less, ship it complete**
- **One source of truth, three layers, one lad** — UI, config, or `applad instruct`. Always coherent, always correct, always attributed.

---
