<p align="center">
  <img src="assets/logo.jpg" alt="Applad Logo" width="300" />
</p>

<p align="center">
  <b>Config-driven, AI-Native Backend Engine. Visually managed. Open source.</b>
</p>

> More than just a BaaS, Applad is the Infrastructure-as-Code (IaC) tool for your entire backend. It replaces disjointed tools for database, auth, CI/CD, feature flags, and analytics with a single, agentless, coherent system. Your config files are the backend, the admin UI is a friendly lens into them. Built with Dart & Flutter.

---

## Features

- **Config-driven backend** — define tables, auth, storage, functions, workflows, messaging, and hosting in version-controlled YAML.
- **Multi-environment** — dev, staging, prod from a single config tree with per-environment overrides.
- **Dynamic API Gateway** — your configuration is automatically exposed as a live REST API.
- **Zero-Dependency Deploy** — `applad up` provisions local servers or matches production VPS environments natively.
- **Open source** — Apache 2.0, own your data and infrastructure.

## Quick Start

```bash
# Install universally without Dart (macOS / Linux / Windows WSL)
curl -fsSL https://raw.githubusercontent.com/mittolabs/applad/main/scripts/install.sh | bash

# OR build from source (recommended for contributors)
./scripts/build_local.sh

# To remove the local build:
rm ~/.applad/bin/applad

# Scaffold a new project natively using Mason templates
applad init

# Start the Applad server (Local Development)
applad up

# Deploy to VPS (Remote Infrastructure)
# 1. Update project.yaml with your VPS host/user
# 2. Run:
applad up --env staging
```

## CLI Reference

Applad provides a rich CLI mapping directly to your YAML configuration.

### Core Commands

```bash
applad init                           # Scaffold applad.yaml and orgs/ directory structure
applad up                             # Start local API server or provision VPS
applad config validate                # Validate full config tree logic
```

### Discovery & Inspection

```bash
applad orgs list                      # List organizations in the current workspace
applad projects list                  # List projects in the current workspace
applad tables list                    # List all database tables defined in config
```

### Modules

```bash
applad messaging test                 # Test channel connectivity (Email/SMS/Push)
applad db migrate                     # Trigger migrations based on config state
```

## API Gateway

When you run `applad up`, a Dart Frog server boots living at `http://localhost:8080`. This server provides a live lens into your configuration patterns.

- `GET /v1/config` - Full merged configuration JSON.
- `GET /v1/orgs` - Current organization details.
- `GET /v1/projects/database` - Active database adapter and connection specs.
- `GET /v1/projects/auth` - Configured providers and security rules.

## Monorepo Structure

```
applad/
├── packages/
│   ├── applad_core/     # Shared models, config engine, YAML merge
│   ├── applad_cli/      # The `applad` CLI binary
│   ├── applad_server/   # Dart Frog API server
│   ├── applad_console/  # Flutter admin dashboard
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
```

## License

Apache 2.0 — see [LICENSE](LICENSE).
