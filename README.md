# Applad

**Open-source Backend-as-a-Service (BaaS) + Infrastructure-as-Code (IaC) + AI-native assistant — built with Dart & Flutter.**

> Deploy your entire backend with a single YAML config tree. Scale from SQLite to Postgres to distributed clusters without changing your app code.

---

## Features

- **Config-driven backend** — define tables, auth, storage, functions, workflows, messaging, and hosting in version-controlled YAML
- **Multi-environment** — dev, staging, prod from a single config tree with per-environment overrides
- **AI assistant** — `applad instruct` understands your config and applies infrastructure changes safely, with full audit trails
- **Self-hostable** — `docker compose up` and you're running
- **Open source** — Apache 2.0, own your data and infrastructure

## Quick Start

```bash
# Install Applad CLI globally
dart pub global activate applad_cli

# Scaffold a new project natively using Mason templates (creates applad.yaml + orgs/ config tree)
applad init

# Validate your entire config tree without starting the server
applad config validate

# Start the Applad server (merges config and listens for requests)
applad up
```

## CLI Reference

Applad provides a rich CLI mapping directly to your YAML configuration. Here are some of the most common commands:

### Scaffolding & Config

```bash
applad init                           # Scaffold applad.yaml and orgs/ directory structure
applad config validate                # Validate full config tree without starting
applad config diff                    # Diff between local config tree and running instance
```

### AI Assistant (Instruct)

```bash
applad instruct "create a users table with email, name, avatar, and soft delete"
applad instruct "add fulltext search to posts"
applad instruct "set up a deployment pipeline for my Flutter app to the Play Store"
applad instruct --dry-run "provision RDS for production"  # Preview changes
```

### Organizations & Projects

```bash
applad orgs list                      # List all organizations on this instance
applad orgs create --name "Acme"      # Create a new org — scaffolds orgs/acme/org.yaml
applad projects list                  # List all projects on this instance
applad projects switch <project-id>   # Set active project context
```

### Database & Tables

```bash
applad db migrate                     # Run pending migrations
applad db generate "add_avatar_to_users"  # Generate a new migration file
applad tables list                    # List all tables in active project
applad tables generate <name>         # Scaffold a new table file in tables/<name>.yaml
```

### Functions & Workflows

```bash
applad functions list
applad functions deploy <name>
applad workflows trigger <name>
applad workflows logs <name>
```

### Messaging & Realtime

```bash
applad messaging channels list        # List configured channels
applad messaging test email --to user@example.com --template welcome
applad realtime channels list
applad realtime status
```

_For a full list of commands, run `applad --help`._

## Monorepo Structure

```
applad/
├── packages/
│   ├── applad_core/     # Shared models, config engine, YAML merge
│   ├── applad_cli/      # The `applad` CLI binary
│   ├── applad_server/   # Dart Frog API server
│   ├── applad_admin/    # Flutter admin dashboard
│   └── applad_client/   # Dart/Flutter client SDK
├── examples/
│   └── starter-project/ # Full working example
└── docker/              # Docker + Compose for self-hosting
```

## Development

```bash
# Install Melos
dart pub global activate melos

# Bootstrap the workspace
melos bootstrap

# Analyze all packages
melos run analyze

# Run all tests
melos run test
```

## License

Apache 2.0 — see [LICENSE](LICENSE).
