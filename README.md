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
# Install Applad CLI
dart pub global activate applad_cli

# Scaffold a new project
applad init

# Validate your config
applad config validate

# Start the server
applad up
```

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
