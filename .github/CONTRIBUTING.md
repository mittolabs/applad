# Contributing to Applad

Thank you for helping make Applad better. This document covers everything you need to get a change merged.

## Code of Conduct

Be respectful. We follow the [Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct/).

## Before You Start

- For **bug fixes and small improvements**, open a PR directly.
- For **new features or breaking changes**, open an issue first to discuss scope before writing code.
- Check [open issues](https://github.com/mittolabs/applad/issues) and [open PRs](https://github.com/mittolabs/applad/pulls) to avoid duplicating effort.

## Development Setup

**Requirements**: Docker with the Compose plugin.

```bash
# Clone and start the dev stack (hot-reload on all services)
git clone https://github.com/mittolabs/applad.git
cd applad
docker compose -f docker-compose.dev.yml up
```

- **API** reloads on every `.go` file save (Air)
- **Console** runs Flutter web dev server on port 3000
- **Postgres** on port 5432, **Redis** on port 6379 (exposed for direct access)

## Tests

All PRs must pass CI. Run the same checks locally before pushing:

```bash
# Backend
cd apps/backend
go build ./...
go vet ./...
go test ./...

# Console + Dart SDK
make bootstrap   # first time only
melos analyze
melos test

# TypeScript client SDK
cd sdks/js && npm install && npm run build && npm test
```

## Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add magic link expiry config
fix: correct cursor pagination offset
docs: update self-host instructions
chore: bump Go to 1.23
```

Keep the subject line under 72 characters. No trailing period.

## Pull Request Checklist

- [ ] Tests pass locally (`go test ./...` / `melos test`)
- [ ] No new `go vet` or `dart analyze` warnings
- [ ] New behaviour is covered by tests
- [ ] CLAUDE.md updated if architecture changes
- [ ] PR description explains *why*, not just *what*

## Database Migrations

If your change requires a schema change, add a new migration file to `apps/backend/internal/db/migrations/` following the existing naming convention. Document the change in `DATABASE_CHANGE.md`.

## Style

- **Go**: `gofmt -w .` before committing. Follow standard Go idioms.
- **Dart/Flutter**: `dart format .`. Follow existing widget patterns.
- **TypeScript**: `npm run lint` in the affected SDK directory.

## Need Help?

Open a [GitHub Discussion](https://github.com/mittolabs/applad/discussions) or ask in an issue. We're happy to guide you.
