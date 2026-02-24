<p align="center">
  <img src="assets/logo.jpg" alt="Applad Logo" width="300" />
</p>

<p align="center">
  <img src="https://github.com/mittolabs/applad/actions/workflows/ci.yml/badge.svg" alt="CI Status" />
  <img src="https://github.com/mittolabs/applad/actions/workflows/release_binaries.yaml/badge.svg" alt="Release Status" />
  <img src="https://img.shields.io/badge/License-BSD_3--Clause-blue.svg" alt="License" />
  <img src="https://img.shields.io/badge/PRs-welcome-brightgreen.svg" alt="PRs Welcome" />
  <img src="https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20WSL-lightgrey.svg" alt="Platform Support" />
</p>

<p align="center">
  <b>Config-driven, AI-Native Backend Engine. Visually managed. Open source.</b>
</p>

> More than just a BaaS, Applad is the Infrastructure-as-Code (IaC) tool for your entire backend. It replaces disjointed tools for database, auth, CI/CD, feature flags, and analytics with a single, agentless, coherent system. Built with Dart & Flutter.

---

## Features

- **YAML-Defined Infrastructure (IaC)** — declare tables, auth, storage, and logic in a single version-controlled config tree.
- **Agentless Operation** — connects, provisions, and disconnects via SSH & Docker. No resident agents or background processes on your servers.
- **Unified BaaS Environment** — auto-generated REST and GraphQL APIs, identity management, and serverless functions in one system.
- **Declarative Access Control** — manage permissions, roles, and access request workflows directly from your admin database.
- **Environment Parity** — maintain exact parity between local development and production with native Docker Compose & VPS orchestration.
- **AI-Native Assistant** — integrated "The Lad" agent that can reason about, safe-check, and modify your infrastructure configuration.

## Quick Start (3 Minutes)

Get a live backend running in three steps:

### 1. Install

```bash
# Install the Applad CLI (macOS / Linux / WSL)
curl -fsSL https://raw.githubusercontent.com/mittolabs/applad/main/scripts/install.sh | bash
```

### 2. Initialize

```bash
# Scaffold a new project starter
applad init --template minimal
```

### 3. Go Live

```bash
# Boot your API server and local infrastructure
applad up
```

_Your API is now live at `http://localhost:8080`_

---

### Core & AI

| Command           | Description                                           |
| ----------------- | ----------------------------------------------------- |
| `applad init`     | Scaffold a new Applad instance structure              |
| `applad up`       | Reconcile local or remote infrastructure              |
| `applad down`     | Stop and remove the local environment                 |
| `applad status`   | Check service health and identify configuration drift |
| `applad instruct` | Give natural language instructions to "The Lad"       |

### API & Access

| Command          | Description                                            |
| ---------------- | ------------------------------------------------------ |
| `applad api`     | Manage REST/GraphQL routes, API Keys, and SDKs         |
| `applad access`  | Manage permissions, roles, and access requests         |
| `applad auth`    | Authenticate and manage identity (login/logout/whoami) |
| `applad secrets` | Manage encrypted vault credentials and keys            |

### Resources & Dev

| Command            | Description                                      |
| ------------------ | ------------------------------------------------ |
| `applad functions` | Manage serverless functions (deploy, logs, test) |
| `applad tables`    | Manage and inspect database schema definitions   |
| `applad config`    | Validate, diff, and snapshot the config tree     |
| `applad env`       | Sync `${VAR}` from config to local `.env` files  |

---

## Monorepo Structure

```
applad/
├── packages/
│   ├── applad_core/     # Core BaaS engine, YAML merge & validation
│   ├── applad_cli/      # Binary CLI tool for infrastructure orchestration
│   ├── applad_server/   # High-performance Dart Frog API Gateway
│   ├── applad_console/  # Flutter-based admin & observability dashboard
│   ├── applad_client/   # Type-safe Dart client SDK for applications
│   ├── applad_function/ # Function runtime & trigger definitions
├── examples/            # Canonical starters (e.g., minimal, auth-flow)
└── docker/              # Base images & orchestration templates
```

---

## Documentation

For a deeper dive into the architecture and operational model:

- 📖 **[Vision & Principles](vision.md)**: The "Why" behind Applad.
- 🛠️ **[Engine Specification](spec.md)**: Details on config formats, triggers, and deployment.

## Contributing

For developers and contributors wanting to build from source:

```bash
# Bootstrap with Melos
dart pub global activate melos
melos bootstrap

# Local Build
./scripts/build_local.sh
```

## License

BSD 3-Clause — see [LICENSE](LICENSE).
