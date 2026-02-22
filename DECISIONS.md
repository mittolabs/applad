# Architectural Decisions

## AD-001: Dockerized Local Development Environment

**Date**: 2026-02-22
**Status**: Accepted

### Context

Applad uses Docker Compose for VPS deployments to ensure a consistent, reproducible runtime environment. Initially, the local development server was orchestrated natively via the CLI using `dart_frog dev`, which required the user to have `dart_frog_cli` installed and handled complex pathing fallbacks.

### Decision

We will use **Docker Compose** for local server orchestration instead of native binary execution. This effectively makes the local environment a 1:1 mirror of the production/staging VPS environments.

### Consequences

- **Environment Parity**: Eliminates "works on my machine" issues by standardizing the runtime (Dart SDK version, OS libraries).
- **Dependency Management**: Users only need Docker installed, rather than specific Dart CLI tools.
- **Complexity Shift**: The CLI now manages `docker-compose.yml` synthesis and orchestration instead of complex path resolution and process spawning.
- **Native Debugging**: While standard `dart_frog dev` is more "native", the containerized approach ensures higher reliability for the combined backend (Gateway + Project Modules).
