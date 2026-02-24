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

- **YAML-Defined Backend** — declare tables, auth, storage, and logic in version-controlled config.
- **Agentless Operation** — connects, provisions, and disconnects. No resident agents on your servers.
- **Environment Parity** — single config tree with native Docker Compose & SSH orchestration.
- **AI-Native** — includes "The Lad," an assistant that understands and modifies your infrastructure.

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

## CLI Reference

Applad mapping directly to your YAML configuration. Use `applad --help` for full details.

### 核心 Core

| Command         | Description                          |
| --------------- | ------------------------------------ |
| `applad init`   | Scaffold a new instance structure    |
| `applad up`     | Reconcile config with infrastructure |
| `applad down`   | Stop the local running instance      |
| `applad status` | Check service health and drift       |

### 智能 Assistance

| Command           | Description                                     |
| ----------------- | ----------------------------------------------- |
| `applad instruct` | Give natural language instructions to "The Lad" |
| `applad config`   | Validate and merge the configuration tree       |

### 运营 Operations

| Command          | Description                                 |
| ---------------- | ------------------------------------------- |
| `applad env`     | Sync `${VAR}` from config to `.env.example` |
| `applad secrets` | Manage encrypted credentials and keys       |
| `applad db`      | Run migrations or open a database shell     |
| `applad tables`  | List and inspect schema definitions         |

---

## Monorepo Structure

```
applad/
├── packages/
│   ├── applad_core/     # Config engine & YAML merge
│   ├── applad_cli/      # Binary CLI tool
│   ├── applad_server/   # Dart Frog API gateway
│   ├── applad_console/  # Flutter admin dashboard
├── examples/            # Working starter projects
└── docker/              # Infrastructure orchestration
```

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
