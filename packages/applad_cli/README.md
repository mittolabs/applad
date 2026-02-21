# Applad CLI

The official command-line interface for **Applad**, the open-source Backend-as-a-Service (BaaS) and Infrastructure-as-Code (IaC) tool.

Applad replaces disjointed tools for database, auth, CI/CD, feature flags, and analytics with a single, agentless, coherent system where everything is config, everything is visual, and everything is assisted by AI.

## Installation

```bash
dart pub global activate applad_cli
```

## Quick Start

```bash
# Scaffold a new project natively
applad init

# Validate your entire config tree without starting the server
applad config validate

# Use the AI assistant to perform infrastructure changes
applad instruct "add fulltext search to posts"
```

For the complete documentation and configuration specs, please view the [Applad GitHub Repository](https://github.com/mittolabs/applad).
