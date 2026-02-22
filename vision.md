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

Applad is designed to run anywhere you can point it — and to move between those places without changing your application code, your config structure, or your mental model. There are three deployment contexts, and they form a natural continuum.

**Local:**
Your laptop, a home server, a Raspberry Pi. Applad uses Docker Compose locally — the same Docker Compose it uses on VPS targets — meaning your local environment is a 1:1 mirror of staging and production. You only need Docker installed. The same containers, the same Dart SDK version, the same OS libraries. No "works on my machine" problems. A developer runs `applad up` and gets a full production-equivalent stack running locally with a single command. When they push to a VPS, nothing about the behavior changes — because the environment never changed.

**VPS:**
A DigitalOcean Droplet, Hetzner server, Linode, bare metal — any machine you own or rent at a flat rate. You have full control, predictable costs, and no vendor dependency at the infrastructure level. Applad connects over SSH, synthesizes and applies a Docker Compose configuration from your project config, manages everything, and leaves. This is the sweet spot for most indie developers, small teams, startups, and anyone who wants full ownership without managing cloud complexity. A single mid-range VPS can comfortably run a production Applad instance serving thousands of users.

**Cloud providers on-demand:**
AWS, GCP, Azure, and others — but used surgically, not as a platform commitment. Applad treats cloud provider resources as adapters you draw from when they make sense for a specific resource. You might run your core app on a Hetzner VPS because it's fast and cheap, use S3 for storage because the pricing is right, use SES for high-volume email at scale, and spin up a cloud compute instance for a heavy data processing job or an iOS build — paying only for the duration, then tearing it down.

**The continuum:**

```
Local dev (Docker Compose, SQLite)
  → VPS staging (Docker Compose, Postgres)
    → VPS production (Docker Compose, Postgres)
      → VPS + cloud storage adapter (S3/R2)
        → VPS + cloud database adapter (RDS)
          → Multi-VPS cluster
            → Kubernetes on cloud
```

You move along this line as your needs grow, and Applad moves with you. Nothing changes about how your application is built or how your config is structured. The only thing that changes is what Applad is pointed at.

**Cloud providers as utility, not platform:**

- Storage can be local filesystem today, S3 tomorrow, R2 next month — application code never changes
- Database can be SQLite locally, Postgres on a VPS in staging, RDS in production — same config structure, different adapter target
- Functions run in Docker containers everywhere — locally, on VPS, or bursting to cloud compute for heavy workloads
- Email can go through SMTP on a VPS, switch to SES at scale — one line change in `messaging/messaging.yaml`
- iOS builds that require macOS spin up an AWS Mac instance, build, and tear it down — paying only for the build duration

---

## Docker Compose Everywhere — Environment Parity by Design

One of Applad's most important architectural decisions is that **Docker Compose is the runtime model at every level** — local development, VPS staging, and VPS production. The CLI synthesizes a `docker-compose.yml` from your project config and orchestrates it. You do not need any Dart tooling installed. You do not need to manage SDK versions. You only need Docker.

This is a deliberate reversal of the usual tradeoff. Many tools make local development feel native and light — run a binary, get instant feedback — but then production looks completely different. Containers, different OS libraries, different runtime behavior. The gap between local and production is where bugs hide.

Applad takes the opposite position: **local is production, from day one**. The same Docker Compose model that runs on your Hetzner VPS runs on your MacBook. The same containers. The same networking. The same service configuration. When something works locally, it works in production — because it's the same thing.

The practical consequences:

- **Onboarding is `applad up`** — any developer on any machine with Docker installed gets a full running stack immediately, no environment setup required beyond filling in `.env`
- **No runtime surprises on deploy** — you've been running production containers locally the entire time
- **The CLI manages Docker Compose synthesis** — Applad reads your project config and generates the correct `docker-compose.yml` for that environment, including service definitions, networking, volume mounts, and environment injection. You never write Docker Compose files by hand.
- **`applad up --dry-run` shows the synthesized compose file** before applying it, so you can inspect exactly what would run
- **Debugging is standard Docker tooling** — `docker logs`, `docker exec`, `docker compose ps`. No Applad-proprietary runtime primitives. Everything is inspectable with tools your team already knows.

This applies equally to the VPS model. When `applad up --env production` runs, it SSHes into the target, synthesizes the correct `docker-compose.yml` for that environment's config, applies it, and leaves. The machine has no Applad agent — just Docker containers running the services defined by your config.

---

## Agentless — Like Ansible, Powered by Familiar Tools

Applad is **agentless**. There is no daemon sitting on your servers waiting for instructions. No agent to install, maintain, update, patch, or secure on every machine you manage. Applad connects over SSH, synthesizes and applies a Docker Compose configuration, manages everything from your config tree, and leaves. The only requirement on the target machine is Docker. When Applad is done, it is gone from that machine until the next operation.

This is the same philosophy that made Ansible win over agent-based tools like Puppet and Chef. The operational overhead of managing agents across a fleet of servers is real, painful, and a source of its own class of bugs and security issues. Applad inherits Ansible's answer to that problem.

**What this means in practice:**

- **Zero persistent footprint on target machines** beyond Docker containers running your application
- **No agent version mismatch issues** — a common nightmare with agent-based tools
- **Works on any machine you can SSH into** — cloud VMs, VPS, bare metal, Raspberry Pis, on-premise servers
- **Smaller security surface** — no persistent daemon means no persistent attack surface on managed machines
- **Internally uses familiar technologies** — Docker Compose for service orchestration, Caddy for reverse proxy and SSL, NATS or Redis for messaging. Operations teams can inspect and reason about everything Applad has provisioned using tools they already know. Nothing is a black box.

**The agentless flow for a typical deployment:**

```
Developer runs: applad deploy run android-production

1. Applad reads deployments/android-production.yaml from config tree
2. Applad opens an SSH connection to the build VPS using the developer's SSH key
3. Applad synthesizes the correct docker-compose.yml for the build environment
4. Applad runs the build container, mounts the source, executes the build command
5. Applad fetches signing credentials from the encrypted operational database
6. Applad signs the artifact and submits to the Play Store via API
7. Applad logs the full operation to the runtime database, attributed to the developer's SSH key
8. SSH connection closes — only Docker containers running your app remain on the machine
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
- **Agentless remote operations** — when Applad SSHes into a remote machine, it does so using the initiating developer's key or a scoped deployment key. The remote machine's auth logs show exactly which key performed which operation and when.
- **Cloud provider operations** — cloud API calls are attributed to the developer identity in both Applad's audit log and the cloud provider's own access logs.
- **Deployment operations** — every `applad deploy run` is attributed to the initiating SSH key identity. The audit trail records which pipeline ran, which artifact was produced, which infrastructure was used, and who triggered it.

**What the audit trail captures:**

```
{
  "timestamp": "2026-02-22T10:32:14Z",
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

## `applad up` — One Command to Rule Them All

`applad up` is the single most important command in Applad. It is the reconciliation command — the equivalent of `terraform apply` for your entire backend. You describe what you want in your config tree. `applad up` makes reality match it.

Concretely, `applad up`:

1. Reads and merges the entire config tree
2. Validates all `${VAR}` references are satisfied — fails fast with clear errors pointing to the exact file and variable
3. Validates all cross-references — functions referenced in workflows exist, tables referenced in realtime channels exist, etc.
4. For local environments: synthesizes a `docker-compose.yml` and applies it via Docker Compose
5. For VPS environments: SSHes in, synthesizes the correct `docker-compose.yml` for that environment, applies it, and disconnects
6. For cloud adapters: provisions any configured adapters that aren't yet active
7. Starts or restarts affected services without downtime
8. Records the operation in the audit trail with the initiating SSH key identity

`applad up --dry-run` is the equivalent of `terraform plan`. It shows exactly what would change — which SSH connections would open, which Docker Compose services would start or restart, which cloud adapters would be provisioned, which config has drifted, which migrations are pending — without doing any of it. **Always run `--dry-run` before applying changes to a production environment.**

`applad up --watch` is for local development only. It watches your config files and automatically reconciles on every save. Combined with Docker Compose's fast container restart, you get an instant feedback loop without leaving the terminal.

Infrastructure is not a separate concern requiring separate commands. It is just config. You define your environments and infrastructure targets. `applad up` handles the rest.

---

## .env.example — Auto-Generated, Always In Sync

Applad eliminates one of the most persistent sources of developer friction — figuring out which environment variables are needed and where. Every `${VAR_NAME}` reference across the entire config tree is automatically extracted and placed into a scoped `.env.example` file that mirrors the config tree structure.

**How it works:**

Applad scans every `.yaml` file in the tree and extracts every `${VAR}` reference. It places each variable into the `.env.example` at the scope where it is first meaningfully referenced — instance-level vars at the root, org-level vars in `orgs/<org>/.env.example`, project-level vars in `orgs/<org>/projects/<project>/.env.example`. Each `.env.example` is annotated — not just a list of empty keys but a documented file that tells you what each variable is for, which config file uses it, what format it expects, and whether it should go through `applad secrets set` rather than a `.env` file in production.

**Key behaviors:**

- **Auto-generated, never manually edited** — `applad env generate` regenerates from the config tree. Adding a new `${VAR}` to any yaml file means the next generation picks it up and places it at the right scope with annotations.
- **Always gitignored for `.env`** — `applad init` writes `.gitignore` entries for all `.env` files across the entire tree automatically. `.env.example` files are always committed.
- **Validated on startup** — `applad up` checks that all referenced `${VAR}` values are present before starting. Missing variables produce a clear error: `Missing required variable STRIPE_SECRET — used by functions/process-payment.yaml`
- **Environment-aware** — `applad env generate --env production` generates a `.env.example` containing only the variables needed for that environment
- **Secret classification** — variables that reference credentials are annotated with a note pointing to `applad secrets set`
- **Across organizations** — each org and project has its own scoped `.env.example`. A developer onboarding to `mobile-app` only sees that project's variables.

**Onboarding a new developer becomes:**

```bash
git clone github.com/myorg/myapp-infra
cp orgs/acme-corp/projects/mobile-app/.env.example \
   orgs/acme-corp/projects/mobile-app/.env
# Fill in values — every variable is annotated with what it's for
applad up
```

Everything they need is documented in the `.env.example`. Nothing is missing. Nothing is a mystery. And because local runs the exact same Docker Compose model as production, they're already testing against a production-equivalent environment from their first `applad up`.

---

## What Lives Where — The Three-Way Separation

One of Applad's core architectural decisions is a clean, three-way separation between what belongs in config files, what belongs in the database as admin-managed operational data, and what belongs in the database as application runtime data.

**Config files — structural decisions requiring developer review, rarely changing:**

- Database schema definitions and migrations
- Table, field, and index definitions
- Permission and security rules
- Auth provider configuration
- Feature flag skeletons — that a flag exists, its variants, its default state, its environments
- Function definitions and source block pointers — local path, GitHub repo, or container registry
- Workflow and automation pipeline structure
- Storage bucket definitions and access rules
- Deployment pipeline definitions — web, mobile, desktop, OTA — with source blocks
- Environment definitions and infrastructure targets
- Organization structure and member role definitions
- Docker Compose synthesis rules — what services each environment needs, how they're networked
- Plugin and adapter configuration
- Enabled/disabled feature toggles at instance and project level
- API and webhook endpoint definitions
- Service integration configuration
- Secret references — `${VAR}` pointers to environment-injected secrets, never the secrets themselves
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
- Full audit log — every config change, every SSH operation, every Docker Compose synthesis and apply, every cloud API call, every deployment of every type, every admin action, every auth event, every `applad instruct` prompt and its changes, with key fingerprint, identity, timestamp, diff, and cryptographic signature
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

Security in Applad is woven through every architectural decision from the ground up. The agentless model, the SSH key identity system, the three-way data separation, the Docker-everywhere approach, and the config-as-code approach all have direct security benefits.

**Encryption:**

- All databases encrypted at rest. Secrets, signing certificates, cloud credentials, and AI API keys in the operational database are encrypted with an additional application-layer key derived from the instance secret — database-level access alone is insufficient to read sensitive values.
- TLS everywhere, automatic via Caddy. SSH connections use key-based auth only — password auth is disabled by design.
- Every config push signed with the initiating developer's SSH key — cryptographic proof of the config state at every point in time.
- Cloud provider API calls made over TLS with credentials fetched at operation time from the encrypted operational database. Never in environment variables or config files.

**Authentication and Identity:**

- SSH key-based identity for all developer and CI/CD interactions. Password-based access disabled.
- Admin UI MFA — TOTP and WebAuthn/FIDO2. Configurable per org.
- Brute force protection — automatic lockout and exponential backoff.
- Argon2id password hashing with configurable cost parameters.
- Short-lived SSH sessions — ephemeral, opened for the duration of an operation and immediately closed.

**Container Security:**

- Every function runs in an isolated Docker container with a read-only filesystem, no-new-privileges enforcement, and restricted network access with an explicit allowlist of permitted outbound hosts.
- Container images are scanned for vulnerabilities before deployment. Critical vulnerabilities block deployment by default.
- The Docker Compose model means container security is consistent across local, VPS, and cloud — the same container configuration runs everywhere.
- No host filesystem access from function containers. No inter-container networking except through Applad's controlled invocation interface.

**Permissions and Isolation:**

- Applad-native permission rules in `tables/*.yaml`, translated to the underlying database's enforcement mechanism.
- Row-level filtering — permissions support filter expressions.
- Strict project and organization isolation by default.
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

**Security in the config tree:**
Security policies live alongside the resources they protect. Permission rules in `tables/*.yaml`. Auth security in `auth/auth.yaml`. Rate limits and CORS in `observability/observability.yaml`. A developer cannot add a table without its permission rules being visible to reviewers in the same diff.

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
│       ├── .env
│       │
│       └── projects/
│           ├── mobile-app/
│           │   ├── project.yaml       # Environments and infrastructure targets
│           │   ├── .env.example       # All project vars — auto-generated, annotated
│           │   ├── .env
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
│           │   ├── functions/         # Flat — one file per function
│           │   │   ├── send-welcome-message.yaml  # source: local | github | registry
│           │   │   ├── process-payment.yaml
│           │   │   └── daily-report.yaml
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
│           │   ├── deployments/       # Flat — one file per deployment, all types
│           │   │   ├── web.yaml       # type: web,        source: github
│           │   │   ├── docs.yaml      # type: web,        source: github
│           │   │   ├── android-production.yaml  # type: play-store, source: github
│           │   │   ├── ios-production.yaml      # type: app-store,  source: github
│           │   │   └── ota.yaml       # type: ota,        source: github
│           │   ├── realtime/
│           │   │   └── realtime.yaml
│           │   ├── analytics/
│           │   │   └── analytics.yaml
│           │   └── observability/
│           │       └── observability.yaml
│           │
│           └── internal-dashboard/
│               ├── project.yaml
│               ├── .env.example
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
6. Validate the full merged config including security policy completeness and cross-references
7. Validate all `${VAR}` references are satisfied — fail fast with clear errors
8. Synthesize Docker Compose configuration for the target environment
9. Start — or error with the exact file and line that failed

---

## Core Architecture

Applad's core infrastructure is written in Dart — auth, database engine, storage, realtime, the CLI, and the admin Flutter app. Applad's functions and automation layer is fully polyglot — Dart, Node.js, Python, Go, PHP, Ruby, etc. Each runtime runs in an isolated Docker container.

The CLI synthesizes Docker Compose configurations from your project config and orchestrates them via Docker. You never write Docker Compose files by hand. Applad generates the correct service definitions, networking, volume mounts, and environment injection for each environment target automatically.

---

## Dart First, Pragmatic Where It Matters

Applad is not "Dart only" — it's **Dart first**. For infrastructure-heavy pieces, Applad composes rather than reinvents:

- **Docker and Docker Compose** — the universal runtime model at every level, synthesized by Applad's CLI from project config
- **Caddy** — reverse proxy, SSL, web deployment hosting layer
- **NATS or Redis** — realtime and pub/sub
- **Rclone** — storage adapters across dozens of providers
- **Buildkit** — polyglot function runtime container builds
- **Cloud provider SDKs** — AWS, GCP, Azure adapters for on-demand resource provisioning

Operations teams can inspect and reason about everything Applad provisions using tools they already know. Standard Docker containers. Standard Caddy config. Standard Docker Compose files on the target machines. Nothing is a black box.

---

## Bidirectional Infrastructure as Code

In Applad, **the UI and config files are always the same thing**. Every developer-driven action — creating a table, defining a permission rule, setting up a deployment pipeline, adding an auth provider — immediately generates or updates the corresponding `.yaml` file and regenerates the relevant `.env.example` if new `${VAR}` references were introduced.

- **UI → Config:** Create a web deployment visually and `deployments/web.yaml` appears in your repo with the correct source block. Set up an Android pipeline and `deployments/android-production.yaml` is written. Add an OAuth provider and `auth/auth.yaml` is updated and `.env.example` gains the credential references with annotations.
- **Config → UI:** Edit any config file, merge a PR, and the UI reflects it immediately.
- **`applad instruct` → Config:** Every instruction that changes structure writes config, updates `.env.example` if needed, updates the synthesized Docker Compose configuration, and records the prompt in the audit trail.
- **`applad up`** reconciles everything — reads the config, synthesizes Docker Compose, makes reality match.

---

## Applad as Your Lad — AI-Powered Infrastructure Assistant

Applad is your **lad** — an active collaborator deeply integrated into every layer of the platform. In the CLI this surfaces as `applad instruct`. In the admin UI the lad has a more characterful presence — the same intelligence, a more conversational interface.

The lad reads your config files, your operational database, your live runtime data, your security event log, your logs, your analytics, and your deployment history — and acts on all three layers appropriately. Every instruction is attributed to your SSH key identity in the audit trail, with the exact prompt recorded alongside every change made.

`applad instruct` is:

- **Self-documenting** — the verb tells you exactly what it does
- **Professional in context** — reads cleanly in documentation, CI logs, and team runbooks
- **Consistent with the CLI's tone** — `applad deploy run`, `applad db migrate`, `applad config push`, `applad instruct`
- **Always `--dry-run` capable** — show the full plan — config files that would change, migrations that would be generated, Docker Compose changes, infrastructure that would be provisioned — without executing anything
- **Always attributed** — the audit trail records the developer's key identity, the exact prompt, and every change made

When `applad instruct` creates a new function, it creates `functions/<name>.yaml` with a source block. When it creates a deployment pipeline, it creates `deployments/<name>.yaml` with the right type and source block. When it provisions infrastructure, it updates `project.yaml`, regenerates `.env.example`, synthesizes the updated Docker Compose config, and runs `applad up --dry-run` so you can review before applying.

**AI Provider Agnosticism:** AI features are powered by your own API keys — stored encrypted in the admin-managed operational database, never in config files. Choose OpenAI, Gemini, Claude, Mistral, or any compatible provider. Swap through the admin UI. Disable entirely in `applad.yaml`.

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

Infrastructure targets are defined per project and per environment in `project.yaml`. Each environment specifies whether it runs locally via Docker Compose, on a VPS via SSH + Docker Compose, or with cloud adapters layered on top. The same project config works at every level of the continuum.

One Applad instance is viable for:

- **Agencies and freelancers** — all clients as isolated orgs, each with their own infrastructure targets and `.env.example` files
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

Connection pooling from day one. Moving from SQLite locally to Postgres on a VPS to RDS in production is a one-line config change. `applad env generate` automatically picks up any new `${VAR}` references and updates `.env.example`. `applad up` synthesizes the correct Docker Compose service configuration for the new adapter. Application code never changes.

---

## Functions — Flat Files, Flexible Source

Each function is a single `.yaml` file in `functions/`. No nested folders. The file defines the function's runtime, resource limits, container security settings, and triggers — and points to the function code via a `source` block.

The source block supports three patterns:

```yaml
# Local — relative to project root
source:
  type: "local"
  path: "./src/functions/process-payment/index.js"

# GitHub — fetched at deploy time
source:
  type: "github"
  repo: "myorg/myapp"
  branch: "main"
  path: "src/functions/process-payment/index.js"
  ssh_key: "ci-github-actions"

# Registry — pre-built container image
source:
  type: "registry"
  image: "ghcr.io/myorg/process-payment:latest"
  credentials: "ghcr-credentials"
```

Teams with a monorepo point all their function source blocks at different paths within the same repo. Teams with microservices point each function at a different repo entirely. Teams that pre-build containers point at a registry. The Applad config tree is identical in structure regardless. The function code can live anywhere.

The same `source` block pattern applies consistently to deployment pipelines — `deployments/web.yaml`, `deployments/android-production.yaml`, and every other deployment type all use the same `source` block shape.

---

## Tables (Not Collections)

`tables` is the universal term for data structures throughout config and CLI, regardless of the underlying adapter. Neutral, understood by everyone. Each table in its own file under `tables/`. Permission rules alongside the schema — always reviewed in the same diff, by the same people, at the same time.

---

## Messaging (Not Just Email)

`messaging` is the umbrella for all communication channels. Provider config in `messaging/messaging.yaml`. Template content in the admin-managed database so non-developers can edit copy without a git commit or deployment.

Channels: Email (Resend, SMTP, SendGrid, SES), SMS (Twilio, Vonage, Africa's Talking), Push (FCM, APNS), In-app, Slack, Discord, Teams.

One unified SDK call. One unified `applad messaging` CLI namespace. One unified admin panel. All provider API keys referenced via `${VAR}` and documented in the project's `.env.example`.

---

## Deployments — Unified Across All Types

Applad uses a single `deployments/` directory and a single `applad deploy` CLI namespace for all deployment types. A web deployment to a domain and an Android deployment to the Play Store are both deployments. Each file has a `type` field and a `source` block.

- **`web`** — deploys a static or dynamic site to a domain via Caddy. Applad synthesizes the Caddy configuration from the deployment yaml. Automatic SSL. Git-connected via source block. Preview environments per PR.
- **`play-store`** — builds an Android app and submits it to the Google Play Store. Build runs in Docker on the configured build VPS.
- **`app-store`** — builds and submits an iOS app. Spins up an AWS Mac instance via cloud-on-demand, builds in Docker, submits, tears down.
- **`desktop`** — packages and distributes Windows, macOS, and Linux applications.
- **`ota`** — pushes over-the-air updates to existing installs. Gradual rollout with adoption tracking.

**The deploy/release distinction is encoded in the type system.** `deployments/` contains pipelines. `flags/` contains release controls. They are separate directories, separate config files, separate CLI namespaces, separate concerns.

Every deployment is attributed to the SSH key identity that triggered it in the audit trail.

---

## Feature Flags

Feature flag skeletons live in individual files under `flags/` — version controlled like code. Targeting rules live in the admin-managed database. Evaluation logs in the runtime database.

Deploy puts the code on the server. The flag releases it to users. A developer deploys new code continuously. A product manager releases it to 10% of users when ready. Increases to 50% when metrics look good. Rolls back instantly if something is wrong. No deployment required for any of this.

---

## You Scale It Your Way

Single Docker Compose stack to start. The full continuum:

- **Local** — Docker Compose on your laptop, SQLite, full production-equivalent environment
- **VPS** — SSH in, Docker Compose applied, full control, predictable flat-rate costs
- **Docker / Docker Compose** — the same model running anywhere
- **Kubernetes / Helm charts** — official Helm charts for horizontal scaling
- **Cloud on-demand** — resources when you need them, gone when you don't, billed only for duration
- **Managed cloud** — hosted Applad for teams that don't want to manage infrastructure

Stateless at the core — config holds structure, database holds state, Docker containers hold nothing beyond what they need to run. `applad up` anywhere with the same config tree and database connection produces an identical stack.

---

## No Lock-in, At Any Layer

- **Portable by design** — REST and GraphQL first, SDK second
- **Your config is yours** — `.yaml` files you own, version control, and take anywhere
- **Your runtime is yours** — standard Docker containers, standard Docker Compose, inspectable with standard tools
- **Your data is yours** — all state in your configured database
- **Your infrastructure is yours** — VPS, cloud, local, or mixed
- **Your secrets are yours** — encrypted in your database, documented in `.env.example`
- **No database lock-in** — swap adapters with one config line
- **No runtime lock-in** — any language for functions, any container for execution
- **No AI lock-in** — your keys, your provider
- **No cloud lock-in** — cloud as utility, not platform
- **No feature lock-in** — enable only what you need
- **No scaling lock-in** — local Docker Compose to Kubernetes cluster
- **No pricing lock-in** — self-hosting is predictable. Cloud billed only when used.
- **No tool lock-in** — replaces BaaS, IaC, CI/CD, feature flags, analytics, AI assistant

---

## Problems We're Solving

**Vendor lock-in** → Portable config, database agnosticism, cloud as utility, standard Docker everywhere

**Environment parity** → Docker Compose at every level — local, VPS, cloud. The same containers run everywhere. "Works on my machine" is eliminated by design.

**Scaling surprises** → Stateless core, connection pooling from day one, clear paths from local to cloud

**Pricing cliffs** → Docker Compose self-hosting on a VPS, cloud billed only when used, transparent managed cloud pricing

**Limited customization** → Everything opt-in, plugins and adapters first-class, last 20% gets same polish

**Auth edge cases** → Multi-tenancy, RBAC, SSO, MFA, custom flows — first-class from day one

**Debugging and observability** → Core feature. Clear errors, structured logs, security event log, deployment logs, instruct history. Standard Docker tooling for inspecting containers.

**Security as an afterthought** → Security policies in config alongside the resources they govern, reviewed in the same PR. Container security enforced uniformly at every level.

**No audit trail** → Every change attributed to an SSH key identity with cryptographic proof. Every instruct prompt recorded. Every deployment attributed.

**Deploy/release conflation** → Deploy is technical and goes through `applad deploy`. Release is a business decision and goes through feature flags. Never mixed.

**Environment variable chaos** → `.env.example` auto-generated from config, scoped per org and project, annotated, always in sync, always committed.

**Agent overhead** → Agentless like Ansible. SSH in, synthesize Docker Compose, apply, leave.

**Cloud complexity** → Cloud as utility. Spin up when needed, tear down when done.

**iOS build infrastructure** → Spin up an AWS Mac instance per build, tear it down when done. Pay only for build duration. No persistent macOS infrastructure to maintain.

**Function code location assumptions** → Source blocks point anywhere — local path, GitHub repo, container registry. The config tree is independent of where code lives.

**Onboarding friction** → `applad init`, fill in `.env`, `applad up`. Three steps. Every variable documented. Full production-equivalent stack running locally.

**The 80/20 problem** → The last 20% is where Applad competes.

---

## Architectural Decisions

### AD-001: Docker Compose as the Universal Runtime Model

**Status:** Accepted

Applad uses Docker Compose for service orchestration at every level — local development, VPS staging, and VPS production. The CLI synthesizes `docker-compose.yml` from project config and applies it via Docker. Users only need Docker installed.

This was a deliberate choice over native binary execution (e.g. running `dart_frog dev` directly), which would have required users to install specific Dart CLI tools and would have created a gap between local and production environments.

**Why it matters:**

- Local is production — the same containers, SDK versions, OS libraries, and service configuration run everywhere
- Onboarding requires only Docker — no language toolchains, no version management, no path resolution
- The CLI manages complexity — Applad synthesizes correct Docker Compose configurations from project config rather than requiring users to write or understand them
- Standard tooling for debugging — `docker logs`, `docker exec`, `docker compose ps` work everywhere. Nothing is Applad-proprietary.
- `applad up --dry-run` shows the synthesized Docker Compose configuration before applying it

---

## Key Principles

- **Applad is your lad** — an active AI-powered collaborator for your entire stack
- **`applad instruct`** — the CLI surface of the lad. Self-documenting, professional, `--dry-run` always available, every prompt recorded in the audit trail
- **Applad is the IaC tool for your entire backend** — config files and the UI are always the same thing
- **Deploy ≠ Release** — `applad deploy` puts artifacts somewhere. Feature flags release functionality to users. Separate concerns, separate commands, separate config directories, separate people.
- **`applad up` is the single reconciliation command** — like terraform apply, for your entire backend. Reads config, synthesizes Docker Compose, makes reality match.
- **Docker Compose everywhere** — local, VPS, cloud. The same runtime model at every level. No environment parity surprises. Ever.
- **Runs anywhere** — local, VPS, cloud on-demand, or all three at once
- **Cloud as utility, not platform** — draw from cloud providers when they make sense
- **Agentless like Ansible** — SSH in, synthesize Docker Compose, apply, leave
- **Internally familiar** — Docker, Docker Compose, Caddy, NATS, Redis. No black boxes.
- **SSH keys are identity** — every change cryptographically attributed to a named developer. Every instruct prompt recorded. No anonymous changes, ever.
- **`.env.example` is always generated, always annotated, always in sync** — onboarding is `cp .env.example .env`, fill in values, `applad up`
- **Three layers, clean separation** — structural intent in config, operational state in admin database, runtime data in runtime database
- **The config tree IS the backend** — one focused file per resource, directory structure encodes the hierarchy
- **Flat files, flexible source** — functions and deployments are flat yaml files with source blocks that point anywhere. No nested folders. No code location assumptions.
- **Security lives alongside what it protects** — reviewed in the same PR
- **Self-hosting shouldn't require a DevOps PhD** — Docker, fill in `.env`, `applad up`
- **Configuration visual and code-friendly simultaneously** — always bidirectionally in sync
- **Admins and marketers are first-class operators** — no deployments for operational changes
- **Instance → Organization → Project → Everything else** — nothing floats free of this hierarchy
- **Tables, not collections** — neutral terminology for everyone
- **Messaging, not email** — one unified channel abstraction
- **Deployments unified** — `deployments/` covers web, mobile, desktop, and OTA. `applad deploy` covers all of them.
- **Extending it should feel native** — plugins, adapters, functions, workflows all first-class
- **Scaling is the developer's choice** — local Docker Compose to Kubernetes cluster
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
