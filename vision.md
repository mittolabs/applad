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
Your laptop, a home server, a Raspberry Pi. Applad uses Docker Compose locally — the same Docker Compose it uses on VPS targets — meaning your local environment is a 1:1 mirror of staging and production. You only need Docker installed. The same containers, the same Dart SDK version, the same OS libraries. No "works on my machine" problems. A developer runs `applad up` and gets a full production-equivalent stack running locally with a single command.

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
- **`applad up --dry-run --diff` shows the full plan** — the synthesized Docker Compose, every service that would start or restart, every config change, every migration pending — before touching anything
- **Debugging is standard Docker tooling** — `docker logs`, `docker exec`, `docker compose ps`. No Applad-proprietary runtime primitives. Everything is inspectable with tools your team already knows.

---

## Agentless — Like Ansible, Powered by Familiar Tools

Applad is **agentless**. There is no daemon sitting on your servers waiting for instructions. No agent to install, maintain, update, patch, or secure on every machine you manage. Applad connects over SSH, synthesizes and applies a Docker Compose configuration, manages everything from your config tree, and leaves. The only requirement on the target machine is Docker.

This is the same philosophy that made Ansible win over agent-based tools like Puppet and Chef. The operational overhead of managing agents across a fleet of servers is real, painful, and a source of its own class of bugs and security issues. Applad inherits Ansible's answer to that problem — and deliberately learns from both what Ansible got right and where it fell short.

**What this means in practice:**

- **Zero persistent footprint on target machines** beyond Docker containers running your application
- **No agent version mismatch issues** — a common nightmare with agent-based tools
- **Works on any machine you can SSH into** — cloud VMs, VPS, bare metal, Raspberry Pis, on-premise servers
- **Smaller security surface** — no persistent daemon means no persistent attack surface on managed machines
- **Internally uses familiar technologies** — Docker Compose for service orchestration, Caddy for reverse proxy and SSL, NATS or Redis for messaging. Operations teams can inspect everything using tools they already know. Nothing is a black box.

**The agentless flow for a typical deployment:**

```
Developer runs: applad deploy run android-production

1. Applad reads deployments/android-production.yaml from config tree
2. Applad opens an SSH connection to the build VPS using the developer's SSH key
3. Applad synthesizes the correct docker-compose.yml for the build environment
4. Applad runs the build container, fetches source, executes the build command
5. Applad fetches signing credentials from the encrypted operational database
6. Applad signs the artifact and submits to the Play Store via API
7. Applad logs the full operation to the runtime database, attributed to the developer's SSH key
8. SSH connection closes — only Docker containers running your app remain on the machine
```

---

## Predictability — The Operational Contract

Ansible earned trust not because it was the most powerful tool, but because it was the most predictable. Operators could read a playbook, know what it would do, run it in check mode, verify the plan, run it for real, and read a clear summary of what happened. That predictability is what makes a tool safe to use in production at 2am.

Applad makes the same contract, fully and explicitly:

**`applad up --dry-run --diff` tells you exactly what will change.** Every service that would start or restart. Every config change. Every migration that would run. Every cloud resource that would be provisioned. Every SSH connection that would open. Nothing is a surprise.

**`applad up` changes exactly that and nothing else.** If `--dry-run` showed it, it happens. If `--dry-run` didn't show it, it doesn't happen.

**The run recap tells you what happened.** After every `applad up`, a clean summary:

```
applad up --env production
...

RECAP ─────────────────────────────────────────────
  environment   production
  duration      14.2s
  actor         alice@acme-corp (SHA256:abc123...)

  ok            12    already correct, no changes
  changed        3    database, functions, messaging
  skipped        0
  failed         0

  ✓ 2 pending migrations applied (primary)
  ✓ send-welcome-message redeployed (source updated)
  ✓ messaging config reconciled (provider changed to ses)
─────────────────────────────────────────────────────
```

**The audit trail records who did it.** Every run, every change, every SSH session, attributed to the initiating SSH key identity.

This is the operational contract. It is non-negotiable. Every command in Applad upholds it.

---

## Idempotency — A First-Class Contract, Not an Accident

Applad's reconciliation model is **fully idempotent**. Running `applad up` twice produces the same result as running it once. Running it against an already-reconciled environment is a no-op that produces a clean recap confirming nothing changed. This is guaranteed — not incidental, not aspirational, not "usually true."

This matters in several contexts:

- **CI/CD pipelines** that run `applad up` on every push need to know a no-change run won't cause restarts, downtime, or spurious audit entries
- **Recovery scenarios** where you need to re-run `applad up` after a partial failure without fear of double-applying migrations or restarting healthy services
- **Team environments** where multiple developers might run `applad up` in quick succession

Each resource type in Applad has an explicit idempotency strategy:

- **Containers** — compared by image digest and environment hash. Only restarted if either has changed.
- **Migrations** — tracked by checksum in the migration history table. Never re-applied.
- **Config** — compared against the last-applied config signature. No-op if unchanged.
- **Cloud resources** — checked for existence before provisioning. Existing resources with matching config are left untouched.
- **Caddy config** — generated deterministically from deployment config. Reloaded only if the output differs from the running config.
- **SSH sessions** — opened only if work needs to be done. A fully reconciled environment opens no SSH connections.

**Drift detection** is the companion to idempotency. `applad status --drift` connects to every configured environment, compares the running state against the config tree, and reports what has drifted — without changing anything. This is the continuous monitoring side of the idempotency contract: Applad can tell you at any time whether reality matches your config, and what the delta is if not.

```
$ applad status --drift --env production

DRIFT REPORT ─────────────────────────────────────
  environment   production

  ✓ database         in sync
  ✗ functions        drift detected
      send-welcome-message: running v1.2.0, config specifies v1.3.0
  ✓ storage          in sync
  ✓ messaging        in sync
  ✗ observability    drift detected
      rate_limiting.routes[/auth/*].requests: running 20, config specifies 15
  ✓ deployments      in sync
──────────────────────────────────────────────────
  2 resources drifted. Run applad up --env production to reconcile.
```

---

## `applad up` — One Command, Full Reconciliation

`applad up` is the single most important command in Applad. It is the reconciliation command — the equivalent of `terraform apply` for your entire backend. You describe what you want in your config tree. `applad up` makes reality match it.

Concretely, `applad up`:

1. If the database is uninitialised, runs bootstrap inline before proceeding
2. Reads and merges the entire config tree
3. Validates all `${VAR}` references are satisfied — fails fast with clear, actionable errors
4. Validates all cross-references — functions referenced in workflows exist, tables referenced in realtime channels exist, etc.
5. Checks the invoking SSH key has the required scope for the target environment (skipped for local)
6. Compares desired state against current state to determine what actually needs to change
7. For resources already in sync — records as `ok`, takes no action, opens no SSH connections
8. For resources that have changed — applies changes in dependency order, using the handler pattern to avoid redundant restarts
9. Produces a run recap with counts, durations, and a clear summary of what changed
10. Records the full operation in the audit trail with the initiating SSH key identity

**Flags:**

`--dry-run` shows the full reconciliation plan without executing anything. Always run before applying to a production environment.

`--diff` shows the delta between current state and desired state for every resource. Combines with `--dry-run` for the safest pre-production review: `applad up --env production --dry-run --diff`.

`--only <tag>` reconciles only resources matching the tag. `--only database` runs pending migrations without touching functions or deployments. `--only functions,messaging` reconciles only those two namespaces.

`--skip <tag>` reconciles everything except the tagged resources. `--skip deployments` reconciles infrastructure without triggering any deployment pipelines.

`--watch` is for local development only. Watches the config tree and automatically reconciles on every save.

`-v` through `-vvv` control verbosity. The default is quiet — just the recap. `-v` adds per-resource status lines. `-vv` adds the synthesized Docker Compose and each SSH command. `-vvv` adds full request/response detail.

**Tags** work across the entire config tree. Every resource belongs to an implicit tag matching its directory — `database`, `tables`, `storage`, `buckets`, `functions`, `workflows`, `messaging`, `flags`, `deployments`, `realtime`, `analytics`, `observability`. Explicit custom tags can be defined on any resource.

---

## The Handler Pattern — Efficient Reconciliation

Applad uses a handler pattern for service restarts, modelled on Ansible's handler system. When multiple config changes would each individually trigger a service restart, Applad batches them and restarts the service exactly once at the end of the reconciliation run.

**Without handlers:** Change messaging config + update a messaging template + rotate the messaging provider API key = three restarts of the messaging service.

**With handlers:** All three changes are applied. The messaging service restart handler fires once at the end. One restart. Zero unnecessary downtime.

The handler pattern applies to:

- **Service restarts** — only if one or more dependent configs changed
- **Migration runs** — all pending migrations for a connection run in a single transaction
- **Config reloads** — Caddy config reloaded once after all web deployment config changes are applied
- **Post-deploy health checks** — run once after all functions in a batch are deployed
- **`.env.example` regeneration** — regenerated once after all config changes in a run

The result is that `applad up` is both correct and efficient. It does exactly the work that needs doing, in the right order, and no more.

---

## `--dry-run` and `--diff` — Everywhere, Not Just at the Top

`--dry-run` is not just a flag on `applad up`. It is available on every command that produces side effects. This is a design requirement, not a convenience:

```bash
applad up --dry-run
applad db migrate --dry-run
applad functions deploy <n> --dry-run
applad deploy run android-production --dry-run
applad access grant bob@acme-corp --dry-run
applad secrets rotate STRIPE_SECRET --dry-run
applad instruct --dry-run "add fulltext search to posts"
```

Every command that writes anything supports `--dry-run`. The canonical pre-production workflow is always:

```bash
applad up --env production --dry-run --diff
# Review the plan
applad up --env production
# Review the recap
```

---

## Actionable Errors — Never Just "What", Always "Why and How"

Ansible's error messages are notoriously unhelpful — they tell you what failed but not why or how to fix it. Applad invests heavily in the opposite.

Every error in Applad names the file, the line, the exact problem, and whenever possible the fix:

```
ERROR database/tables/users.yaml line 14
  Relation field "org_id" references table "organisations" which does not exist.
  Did you mean "organizations"? (found in database/tables/organizations.yaml)
  Fix: change the table: value on line 14 to "organizations"
```

```
ERROR Missing required variable STRIPE_SECRET
  Referenced by: functions/process-payment.yaml line 8
  This variable is required for the "production" environment.
  Set it with: applad secrets set STRIPE_SECRET
  Or add it to your .env file for local development.
```

```
ERROR applad up --env production requires infrastructure:apply:production scope
  Your key (SHA256:def456... bob@workstation) does not have this scope.
  Ask an admin to run: applad access grant bob@acme-corp \
    --scope "infrastructure:apply:production" --project mobile-app
```

```
ERROR Cross-database relation detected
  database/tables/posts.yaml line 22: "author_id" targets "users" (primary)
  but "posts" targets the "analytics" connection.
  Cross-database relations must be resolved at the application layer.
  Use type: "string" and handle the join in your application code.
```

Error messages are treated as a first-class product surface. Clear, actionable errors are the difference between a tool that teams trust and one that causes support tickets.

---

## Config is Purely Declarative — No YAML Logic

Ansible playbooks start as clean YAML but quickly become a programming language — loops, conditionals, Jinja2 templates embedded in strings. It is one of the most persistent complaints about Ansible at scale. Once you need `when:` conditions and `with_items:` loops in your infrastructure config, you have accidentally written a program in a format that wasn't designed for it, and debugging it is miserable.

Applad's config files are and will always remain **purely declarative**. They describe what exists — schemas, adapters, pipelines, providers, rules. They do not describe logic.

If you need conditional behavior — send this message only if the user has a mobile device, run this migration only if this table exists, deploy to production only if tests pass — that is what functions, workflows, and CI/CD pipeline conditions are for. Config describes structure. Code describes behavior. These are different things and they live in different places.

This is a hard line. Applad will never add templating syntax, conditional blocks, or loop constructs to `.yaml` config files. Any `.yaml` file in an Applad project can be read by anyone — developer, product manager, security auditor, `applad instruct` — and understood immediately, without knowing a template language. Config is always what it appears to be.

---

## Secrets Management — First-Class, Not Bolted On

Ansible Vault was added as an afterthought — encrypted files in the repo, a separate password to manage, awkward integration with external secret managers. The lesson is to design secrets management as a first-class system from the start.

**The fundamental rule:** Secrets never live in config files. Config files contain only `${VAR}` references — pointers. The actual values live in one of three places depending on context:

- **Local development** — `.env` file, filled in by the developer from `.env.example`. Only on the developer's machine. Never committed.
- **Admin database** — for all non-local environments. Set via `applad secrets set`, managed through the admin UI. Encrypted at rest with an application-layer key in addition to database-level encryption.
- **External secret manager** — for teams using AWS Secrets Manager, HashiCorp Vault, GCP Secret Manager, or similar. Applad fetches from there at operation time.

**Scoped secret storage:**

Secrets exist at four scopes, resolved from most specific to least:

```
Environment-level  →  secrets for a specific environment only
Project-level      →  available to all environments in the project
Org-level          →  available to all projects in the org
Instance-level     →  available everywhere
```

The same `${STRIPE_SECRET}` reference in config resolves to the test key in development and the live key in production, automatically, based on scope.

**Safe injection — secrets never written to disk on servers:**

For VPS environments, Applad injects secrets into containers at runtime via Docker's environment injection, passed through the SSH session. The synthesized `docker-compose.yml` written to the server contains references, not values. A developer with SSH access to the production VPS cannot read secret values by inspecting the compose file on disk.

**External provider adapters:**

```yaml
secrets:
  provider: "vault"
  config:
    address: ${VAULT_ADDR}
    token: ${VAULT_TOKEN}
    path: "secret/mobile-app"
```

The default provider is the admin database. Teams with existing Vault or AWS Secrets Manager infrastructure point Applad at those instead. `${VAR}` references in config files never change — the provider is a resolution detail, not a config concern.

**Rotation with a transition window:**

`applad secrets rotate STRIPE_SECRET` enters a configurable transition window — typically 15 minutes — during which both old and new values are accepted. At the end of the window the old value is permanently invalidated. Dependent containers receive the new value. Services that can hot-reload do so without restart. Those that can't get a rolling restart with zero downtime.

**Emergency revocation:**

`applad secrets revoke STRIPE_SECRET` immediately invalidates the secret, lists every function, workflow, and service that referenced it, and optionally triggers redeployment of affected services. The blast radius is always visible before confirming.

**Full audit trail on all secret operations:**

Every `applad secrets set`, `rotate`, `revoke`, and runtime access of a secret by a function or container produces an audit log entry. Who set it, when it was last rotated, which services have accessed it, and from which environment.

**`.env` files are explicitly development-only:**

`applad up --env staging` and `applad up --env production` warn loudly — and can be configured to refuse — if a `.env` file is detected as the source of secrets rather than the admin database or an external provider.

---

## Composable Config — Shared Roles and Templates

Ansible roles are reusable, self-contained units of configuration that can be shared and composed across playbooks. Applad formalises this with a `shared/` directory at the instance level that any org or project can reference.

A team might define a `postgres-primary` shared config block that includes standard connection settings, migration config, and baseline table permission rules — then reference it across multiple projects. A team that manages many similar projects defines the baseline once in `shared/` and inherits it everywhere, with per-project overrides where needed.

Project templates extend this to whole-project scaffolding. `applad init --template saas` is the entry point, but once a project is running well it can be extracted as a template and reused. Templates are just directories of `.yaml` files with `${VAR}` placeholders — no special syntax.

**Config inheritance** follows the same hierarchy as secret resolution: environment-level overrides beat project-level, which beats org-level, which beats instance-level. Overrides are explicit and always visible in the config tree.

The guiding rule: **compose config, don't copy it**. If the same configuration block appears in more than one project, it belongs in `shared/`. Drift between similar projects is a maintenance burden — shared config eliminates it.

---

## Machine-Readable Output and Exit Codes

Applad's CLI is designed for both human operators and automated pipelines. Every command that produces output supports `--output json`. CI/CD pipelines can parse `applad up --output json` and act on the structured recap.

Exit codes are well-defined and stable across all commands:

| Code | Meaning                                                                  |
| ---- | ------------------------------------------------------------------------ |
| `0`  | Success — no changes needed, or no errors                                |
| `1`  | Error — command failed                                                   |
| `2`  | Success with changes — one or more changes were applied                  |
| `3`  | Validation failure — config is invalid, nothing was attempted            |
| `4`  | Unreachable — one or more infrastructure targets could not be reached    |
| `5`  | Drift detected — returned by `applad status --drift` when drift is found |

Pipelines can differentiate between success-with-no-changes (exit 0) and success-with-changes (exit 2) — which matters for triggering downstream steps like Slack notifications or smoke tests.

## Discovery over Templates — Guided Interaction

Applad avoids the "kitchen sink" template problem. A new project starts minimal, with only the core infrastructure enabled. From there, the CLI guides you through adding exactly what you need.

- **Interactive `applad init`**: You select which namespaces you need (Functions, Storage, Messaging, etc.). Applad enables them in `applad.yaml` and creates the folders, but leaves them empty of clutter.
- **Resource `create` commands**: Instead of copy-pasting YAML, you use commands like `applad functions create`. The CLI prompts you for the name, runtime, trigger, and memory limits, then generates a perfect, spec-compliant file for you.
- **Live Documentation**: The `create` commands act as a guided interface to the spec. No need to look up valid runtime strings or trigger types — the CLI gives you the options inline.

This approach ensures that every file in your project was put there by a deliberate choice, not as part of a boilerplate package you didn't fully understand.

---

## Access Control — Config Describes, Database Enforces

**Config files cannot be the enforcement layer for permissions that govern config files.** That's circular. You cannot guard the safe with a key that's inside the safe. If `require_approval: true` lived in `project.yaml`, any developer with repo access could change it to `false` and push.

Applad's answer is a clean separation: **config files describe intent, the admin database enforces it**.

Role definitions in `org.yaml` are declarative documentation. Editing any config file and pushing to git does not change what anyone can actually do. Only `applad access` commands and the admin UI change what anyone can do — both require an SSH key with `access:manage` scope, both write to the admin database, and both are recorded in the audit trail.

**The three-layer permission model:**

Every operation is checked against three things simultaneously. The effective permission is the intersection of all three.

```
Role grants        — what your role allows, stored in admin database
Project grants     — your role in this specific project, stored in admin database
SSH key scopes     — the maximum your key can ever exercise, stored in org.yaml
```

Key scopes in `org.yaml` are a hard ceiling that role grants cannot exceed. This is the one security function config files legitimately serve — constraining maximum capability, not granting it.

**What `applad up` actually checks:**

When `applad up --env production` runs, Applad checks the admin database — not `project.yaml` — to verify the invoking SSH key has `infrastructure:apply:production`. A developer can push config changes all day. Nothing touches production until an authorized key explicitly runs the command, and that gate checks the database.

**Destructive operations require explicit elevation:** `schema:destructive`, `permissions:write`, `infrastructure:apply:production`, `access:manage` — none granted by default, all audited.

**Local development is exempt:** `applad up` against a local environment runs immediately without database grant checks.

---

## SSH Keys and Traceability — Every Change Has an Author

Every CLI operation, every config push, every admin UI session, every `applad instruct` action, every `applad access` operation, every agentless remote operation, and every deployment of every type is attributed to a named SSH key identity with cryptographic proof.

The audit trail captures actor, action, target, change, and — for `applad instruct` — the exact prompt that produced the change. Every access grant and revocation records both who received the change and who made it. Key rotation preserves historical attribution. Revocation blocks new operations immediately while preserving the historical record.

---

## First Run and Developer Onboarding

**`applad init` does one thing: scaffold a new project.** It fails immediately if `applad.yaml` already exists.

**`applad up` handles bootstrap on first run.** When it detects an uninitialised database, it prompts for the instance URL, first owner's email, SSH public key path, and organisation name, then seeds the database and permanently closes the bootstrap path.

**Developers joining an existing project run `applad login`.** Reads the instance URL from `applad.yaml`, registers their SSH key, sends an access request to administrators. Local environments run immediately — no approval needed. Shared environments require an administrator to run `applad access approve`.

| Command        | Intent                               | Fails if                                           |
| -------------- | ------------------------------------ | -------------------------------------------------- |
| `applad init`  | Scaffold a new project               | `applad.yaml` already exists                       |
| `applad up`    | Reconcile. Bootstrap on first run.   | Validation fails, vars missing, scope insufficient |
| `applad login` | Authenticate to an existing instance | Instance unreachable, key already registered       |

---

## .env.example — Auto-Generated, Always In Sync

Every `${VAR_NAME}` reference across the entire config tree is automatically extracted and placed into a scoped `.env.example`. Each is annotated — what the variable is for, which config file uses it, what format it expects, whether it should go through `applad secrets set` in production.

Auto-generated on every config change. Always gitignored for `.env`. Validated on startup with actionable errors for missing variables. Environment-aware. Scoped per project.

Onboarding: `cp .env.example .env`, fill in values, `applad login`, `applad up`. Every variable documented. Nothing is a mystery.

---

## What Lives Where — The Four-Way Separation

**Config files** — structural decisions requiring developer review: schemas, pipelines, rules, adapters, role definitions (intent only), SSH key scopes (hard ceilings), `${VAR}` references (never values), `.env.example` files.

**Database (access control)** — managed via `applad access` only: actual role grants, project-level overrides, scope grants, environment-level apply grants, time-limited access, pending requests, access change history.

**Database (admin-managed operational data)** — feature flag targeting rules, messaging template content, signing certificates (encrypted), AI and cloud provider credentials (encrypted), secret values for non-local environments (encrypted), IP allowlists, MFA enrollment.

**Database (application runtime data)** — user records, application data, analytics, full audit log, deployment history, function logs, messaging send history, real-time state, storage metadata, security event logs.

---

## Security

Security is woven through every architectural decision. Config files cannot enforce security over themselves — access controls live in the admin database. Secrets are injected at runtime via SSH session, never written to disk on servers. Every function runs in an isolated container with read-only filesystem, no-new-privileges, and a restricted network allowlist. Container images scanned before deployment. TLS everywhere via Caddy. SSH key-based auth only. Bootstrap permanently closeable. Full secret access audit trail. Security policies live alongside the resources they protect, reviewed in the same PR.

---

## Config File Structure — The Directory IS the UI

The directory structure mirrors UI navigation exactly. `database/tables/users.yaml` is Database > Tables > users. `storage/buckets/avatars.yaml` is Storage > Buckets > avatars. The path in the filesystem is the breadcrumb in the UI. They are the same thing.

```
my-project/
├── applad.yaml
├── .env.example
├── .env
├── .gitignore
│
├── orgs/
│   └── acme-corp/
│       ├── org.yaml
│       ├── .env.example
│       └── mobile-app/
│           ├── project.yaml
│           ├── .env.example
│           ├── auth/
│           ├── database/
│           │   ├── database.yaml
│           │   ├── migrations/
│           │   └── tables/
│           ├── storage/
│           │   ├── storage.yaml
│           │   └── buckets/
│           ├── functions/
│           ├── workflows/
│           ├── messaging/
│           │   └── templates/
│           ├── flags/
│           ├── deployments/
│           ├── realtime/
│           ├── analytics/
│           └── observability/
│
└── shared/
```

`org.yaml` marks an org. `project.yaml` marks a project. No explicit listing anywhere. No `projects/` wrapper folder — one level shorter everywhere. Applad discovers everything by scanning for these marker files.

---

## Core Architecture

Dart for the core — auth, database engine, storage, realtime, CLI, admin Flutter app. Polyglot for functions — Dart, Node.js, Python, Go, PHP, Ruby, etc., each in an isolated Docker container.

Composing proven open source tools: Docker Compose for orchestration, Caddy for reverse proxy and SSL, NATS or Redis for realtime and pub/sub, Rclone for storage adapters, Buildkit for container builds, cloud provider SDKs for on-demand resources. Nothing is a black box. Everything inspectable with standard tools.

---

## Bidirectional Infrastructure as Code

Every developer-driven action in the UI immediately generates or updates the corresponding `.yaml` file. Every config file change reflects in the UI immediately. `applad instruct` writes config, updates `.env.example`, updates Docker Compose synthesis, records the prompt. `applad up` reconciles everything. No gap, no drift, no manual translation.

---

## Applad as Your Lad — AI-Powered Infrastructure Assistant

`applad instruct` is the CLI surface of the lad. Self-documenting, professional, always `--dry-run` capable, access-aware, every prompt recorded in the audit trail attributed to the invoking SSH key identity. The lad tells you when an instruction would require elevated scope before attempting it.

When it creates a function, it creates `functions/<n>.yaml` with a source block. When it creates a table, it creates `database/tables/<n>.yaml` with schema and permission rules together. When it provisions infrastructure, it updates `project.yaml`, regenerates `.env.example`, synthesizes updated Docker Compose, and runs `--dry-run` for review before applying.

**AI Provider Agnosticism:** Your own API keys, stored encrypted in the admin database. Choose any compatible provider. Swap through the admin UI. Disable entirely in `applad.yaml`.

---

## Instance → Organization → Project Hierarchy

```
Instance
└── Organisation (e.g. acme-corp)
    ├── Project (e.g. mobile-app)
    └── Project (e.g. internal-dashboard)
```

Nothing floats free of this hierarchy. `org.yaml` marks an org. `project.yaml` marks a project. Infrastructure targets defined per project and per environment. The same config works at every level of the continuum.

---

## Database Agnosticism, Multi-Tenancy, Functions, Deployments

Multiple named connections per project, each with its own `migrations.dir`. Tables reference connections via `database:` field. Cross-database relations flagged at validation time with actionable errors. Adapters: PostgreSQL, MySQL, SQLite, MongoDB, Redis, libSQL, RDS, Cloud SQL, extensible via custom adapters.

Three multi-tenancy models in `auth/auth.yaml`: row (shared schema, tenant field, row-level filters), schema (separate Postgres schemas per tenant), database (separate connection per tenant). Choose once, configured in one place.

Functions are flat yaml files with source blocks pointing anywhere — local path, GitHub repo, container registry. Deployments are the same: flat yaml files in `deployments/`, one per pipeline, covering web, Play Store, App Store, desktop, and OTA. `applad deploy` covers all of them. `flags/` controls release. They never touch.

---

## You Scale It Your Way

Single Docker Compose stack to start. Full continuum to Kubernetes. Stateless at the core. `applad up` anywhere with the same config tree and database connection produces an identical stack.

---

## No Lock-in, At Any Layer

Config, data, secrets, AI keys, infrastructure, runtime, database, cloud, features, tools. The `.yaml` files you own, version control, and take anywhere. Standard Docker containers inspectable with standard tools. Your data in your database. Your secrets in your encrypted database or your existing secret manager. Never locked in at any layer.

---

## Problems We're Solving

**Vendor lock-in** → Portable config, database agnosticism, cloud as utility, standard Docker everywhere

**Environment parity** → Docker Compose at every level. The same containers run everywhere. "Works on my machine" eliminated by design.

**No idempotency guarantee** → Every operation idempotent. Running `applad up` twice is the same as running it once. Guaranteed.

**No drift detection** → `applad status --drift` shows exactly what has drifted in every environment, without changing anything.

**Unpredictable reconciliation** → `--dry-run --diff` first, `applad up` second, recap third. No surprises. Ever.

**Redundant restarts** → Handler pattern. Services restart exactly once per reconciliation run.

**No dry-run on subcommands** → `--dry-run` on every command that produces side effects. Always.

**Unhelpful error messages** → Every error: file, line, problem, fix. Never just "what."

**YAML as a programming language** → Config is purely declarative. Always. Logic goes in functions and workflows.

**Secrets bolted on** → First-class secrets: scoped hierarchy, safe injection, external provider adapters, rotation with transition windows, emergency revocation, full audit trail.

**No machine-readable output** → `--output json` everywhere. Well-defined exit codes 0–5. Pipelines can act on the difference between success-no-changes and success-with-changes.

**Config files as security enforcement** → Access controls in the admin database. A developer cannot edit their way to elevated privileges.

**No audit trail** → Every change attributed to an SSH key identity. Every instruct prompt recorded. Every secret access logged.

**Agent overhead** → Agentless. SSH in, synthesize Docker Compose, apply, leave.

**Deploy/release conflation** → Deploy is technical. Release is a business decision. Separate commands, separate config, separate people.

**Onboarding friction** → `cp .env.example .env`, fill in values, `applad login`, `applad up`.

**The 80/20 problem** → The last 20% is where Applad competes.

---

## Architectural Decisions

### AD-001: Docker Compose as the Universal Runtime Model

**Status:** Accepted — Same containers at every level. Local is production from day one. Only Docker required.

### AD-002: `org.yaml` and `project.yaml` as Discovery Markers

**Status:** Accepted — File presence is the signal. No explicit listing. No `projects/` wrapper. Every path shorter.

### AD-003: Directory Structure Mirrors UI Navigation

**Status:** Accepted — `database/tables/users.yaml` is Database > Tables > users. Always. Error messages, audit entries, and `applad instruct` references are self-navigable.

### AD-004: Config Files Describe, Admin Database Enforces

**Status:** Accepted — Role definitions are declarative documentation. Actual grants live in the admin database, managed only via `applad access`. Editing config files cannot change what anyone can do.

### AD-005: Bootstrap is Inline in `applad up`

**Status:** Accepted — First-run setup is a special case of reconciliation. No separate command. `applad init` keeps its single meaning.

### AD-006: `applad init`, `applad up`, and `applad login` Have Non-Overlapping Intent

**Status:** Accepted — Three commands, three jobs, no overlap, no double duty.

### AD-007: Tables Inside `database/`, Buckets Inside `storage/`

**Status:** Accepted — Mirrors UI navigation. Directory path equals UI breadcrumb.

### AD-008: Multiple Database Connections with Per-Connection Migration Dirs

**Status:** Accepted — Named connections, per-connection `migrations.dir`, table-level routing via `database:` field, cross-database relations flagged at validation time.

### AD-009: Idempotency is a Guaranteed Contract

**Status:** Accepted — Every operation idempotent by explicit strategy per resource type. CI/CD pipelines can rely on it absolutely.

### AD-010: `--dry-run` is Available on Every Side-Effecting Command

**Status:** Accepted — Design requirement, not convenience. `--diff` available alongside. Canonical pre-production workflow is always `--dry-run --diff` first.

### AD-011: Config is Purely Declarative — No YAML Logic

**Status:** Accepted — Hard line, never crossed. No templating, no conditionals, no loops. Logic goes in functions and workflows. Config is always readable by anyone.

### AD-012: Handler Pattern for Efficient Reconciliation

**Status:** Accepted — Services restart exactly once per run regardless of how many changes triggered them. Applied to restarts, migrations, config reloads, health checks, `.env.example` regeneration.

### AD-013: Secrets are First-Class, Not Bolted On

**Status:** Accepted — Scoped hierarchy. Safe injection via SSH session. External provider adapters. Rotation with transition windows. Emergency revocation with visible blast radius. Full audit trail.

### AD-014: Actionable Errors — File, Line, Problem, Fix

**Status:** Accepted — Every error names the file, line, problem, and fix. Error messages are a first-class product surface.

### AD-015: Well-Defined Exit Codes and Machine-Readable Output

**Status:** Accepted — Exit codes 0–5, stable and well-defined. `--output json` on every command. Pipelines differentiate success-no-changes from success-with-changes.

---

## Key Principles

- **Applad is your lad** — an active AI-powered collaborator for your entire stack
- **`applad instruct`** — self-documenting, professional, `--dry-run` always available, every prompt recorded, access-aware
- **Applad is the IaC tool for your entire backend** — config files and the UI are always the same thing
- **Deploy ≠ Release** — separate commands, separate config, separate people
- **`applad up` is the single reconciliation command** — reads config, synthesizes Docker Compose, makes reality match. Bootstrap on first run. Idempotent always.
- **`applad init` scaffolds. `applad up` runs. `applad login` connects.** Three commands, three jobs, no overlap.
- **Predictability is the contract** — `--dry-run --diff` first, `applad up` second, recap third. No surprises, ever.
- **Idempotency is guaranteed, not incidental** — running `applad up` twice is the same as running it once, always
- **`--dry-run` everywhere** — every command that writes anything supports it
- **Drift detection is built in** — `applad status --drift` shows what has drifted in every environment
- **Handler pattern** — services restart exactly once per reconciliation run
- **Errors are actionable** — file, line, problem, fix. Never just "what."
- **Config is purely declarative** — no templating, no conditionals, no loops. Logic goes in functions and workflows.
- **Composable config** — shared roles and templates. Compose, don't copy.
- **Well-defined exit codes** — pipelines differentiate success-no-changes from success-with-changes
- **`--output json` everywhere** — machine-readable output on every command
- **Docker Compose everywhere** — local, VPS, cloud. Same runtime model. No environment parity surprises. Ever.
- **Runs anywhere** — local, VPS, cloud on-demand, or all three at once
- **Cloud as utility, not platform** — draw from cloud providers when they make sense
- **Agentless like Ansible** — SSH in, synthesize Docker Compose, apply, leave. No persistent footprint.
- **Internally familiar** — Docker, Caddy, NATS, Redis. No black boxes.
- **SSH keys are identity** — every change cryptographically attributed. Every prompt recorded. No anonymous changes, ever.
- **Config files describe. The admin database enforces.** A developer cannot edit their way to elevated privileges. Ever.
- **`applad access` is the only way to change access control** — authenticated, privileged, fully audited
- **Secrets are first-class** — scoped hierarchy, safe injection, external adapters, rotation, emergency revocation, full audit trail
- **`.env.example` is always generated, always annotated, always in sync**
- **Four layers, clean separation** — structural intent in config, access control in access database, operational state in admin database, runtime data in runtime database
- **The directory path is the UI breadcrumb** — always
- **The config tree IS the backend** — one focused file per resource, directory structure encodes the hierarchy
- **`org.yaml` marks an org. `project.yaml` marks a project.** No redundant wrapper folders.
- **Flat files, flexible source** — functions and deployments are flat yaml files with source blocks pointing anywhere
- **Security lives alongside what it protects** — reviewed in the same PR
- **Self-hosting shouldn't require a DevOps PhD** — Docker, fill in `.env`, `applad login`, `applad up`
- **Configuration visual and code-friendly simultaneously** — always bidirectionally in sync
- **Admins and marketers are first-class operators** — no deployments for operational changes
- **Instance → Organisation → Project → Everything else** — nothing floats free of this hierarchy
- **Tables, not collections** — neutral terminology
- **Messaging, not email** — one unified channel abstraction
- **Deployments unified** — web, mobile, desktop, OTA. One directory, one CLI namespace.
- **Multi-tenancy is a first-class choice** — row, schema, or database isolation
- **Multiple databases per project** — named connections, per-connection migrations, table-level routing
- **Extending it should feel native** — plugins, adapters, functions, workflows all first-class
- **Scaling is the developer's choice** — local Docker Compose to Kubernetes cluster
- **The admin UI is a Flutter app** — desktop, mobile, and web, but optional
- **AI provider agnosticism** — your keys, your provider, encrypted in your database
- **Feature flags without a third-party tool** — skeletons in config, targeting in admin database
- **Analytics without leaving the platform** — with escape hatches when you need them
- **Portable by design** — config, data, secrets, AI keys, infrastructure. Never locked in.
- **The last 20% is where we compete**
- **Compose, don't reinvent** — Dart orchestrates, proven open source tools do the heavy lifting
- **Ship less, ship it complete**
- **One source of truth, four layers, one lad** — UI, config, or `applad instruct`. Always coherent, always correct, always attributed.

---
