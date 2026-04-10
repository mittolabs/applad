# DATABASE_CHANGE.md — MariaDB → PostgreSQL + PostgREST Migration

> **Decision locked**: April 9, 2026
> **Scope**: Replace MariaDB with PostgreSQL, replace JSON document model with real tables, integrate PostgREST as the database CRUD engine behind Applad's unified gateway, switch realtime to PostgreSQL-driven events.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Current State](#current-state)
3. [Target State](#target-state)
4. [Phase 1 — PostgreSQL Driver Swap](#phase-1--postgresql-driver-swap)
5. [Phase 2 — Migrations Rewrite](#phase-2--migrations-rewrite)
6. [Phase 3 — Service SQL Conversion (Non-Database Services)](#phase-3--service-sql-conversion-non-database-services)
7. [Phase 4 — Database Service Redesign (Real Tables + PostgREST)](#phase-4--database-service-redesign-real-tables--postgrest)
8. [Phase 5 — RLS Policy Engine](#phase-5--rls-policy-engine)
9. [Phase 6 — Realtime Rewrite (PostgreSQL LISTEN/NOTIFY)](#phase-6--realtime-rewrite-postgresql-listennotify)
10. [Phase 7 — PostgREST Integration](#phase-7--postgrest-integration)
11. [Phase 8 — Docker & Infrastructure](#phase-8--docker--infrastructure)
12. [Phase 9 — Test Updates](#phase-9--test-updates)
13. [Phase 10 — SDK, Console & SQL Editor](#phase-10--sdk-console--sql-editor)
14. [Phase 11 — Terminology Cleanup](#phase-11--terminology-cleanup)
15. [Phase 12 — Documentation](#phase-12--documentation)
16. [Constraints & Rules](#constraints--rules)
17. [Progress Checklist](#progress-checklist)

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────────┐
│               Applad Gateway (Go API server)             │
│                                                          │
│  /v1/account/*      → Auth service (Go)                  │
│  /v1/users/*        → Users service (Go)                 │
│  /v1/teams/*        → Teams service (Go)                 │
│  /v1/storage/*      → Storage service (Go)               │
│  /v1/functions/*    → Functions service (Go)              │
│  /v1/workflows/*    → Workflows service (Go)             │
│  /v1/messaging/*    → Messaging service (Go)             │
│  /v1/deploy/*       → Deploy service (Go)                │
│  /v1/databases/*    → DDL: Go | CRUD: PostgREST proxy    │
│  /v1/realtime       → WebSocket ← PG LISTEN/NOTIFY       │
│                                                          │
│  Auth injection, rate limiting, API key scoping,         │
│  project isolation — all enforced at gateway level       │
└──────────────────────────┬───────────────────────────────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
        ┌─────┴──┐  ┌─────┴──┐  ┌─────┴──┐
        │ Postgres│  │PostgREST│  │ Redis  │
        │  Real   │  │  CRUD   │  │ Cache  │
        │  Tables │  │  Engine │  │ Queue  │
        │  RLS    │  │         │  │ PubSub │
        └─────────┘  └─────────┘  └────────┘
```

### Key Principles

- **Applad Go server is the single gateway** — NOT a Kong/mesh-style sidecar assembly
- **PostgREST handles database row CRUD only** — proxied through the gateway transparently
- **Schema DDL (CREATE/ALTER/DROP TABLE) stays in Go** — Applad orchestrates all schema changes
- **RLS policies are auto-generated** — Applad writes PostgreSQL policies, PostgREST enforces them
- **Realtime is PostgreSQL-native** — WAL/LISTEN/NOTIFY replaces manual `events.Publish()` calls
- **One PostgreSQL schema per user-database** — naming: `p_{projectId}_{databaseId}`
- **No MariaDB/MySQL code remains** — this is a full replacement

---

## Current State

### Driver & Connection
- **File**: `backend/internal/db/db.go`
- **Driver**: `github.com/go-sql-driver/mysql v1.7.1`
- **Connection**: `sql.Open("mysql", dsn)`
- **Pool**: 25 max open, 10 max idle, 5min lifetime

### Config
- **File**: `backend/internal/config/config.go`
- **Field**: `DatabaseDSN`
- **Default**: `applad:applad@tcp(mariadb:3306)/applad?parseTime=true`
- No driver selection field exists

### Entry Point
- **File**: `backend/cmd/api/main.go`
- Flow: `config.Load()` → `db.Connect(dsn)` → `database.Migrate()` → `router.New()` → serve

### Migration System
- **File**: `backend/internal/db/migrate.go`
- Embeds `backend/internal/db/migrations/*.sql` via `go:embed`
- Bootstraps `schema_migrations` table with MySQL DDL (`ENGINE=InnoDB`)
- Uses `?` placeholders for version tracking queries
- Splits SQL on `;` (not dialect-aware)

### Migration Files (20 files)
| File | Summary |
|---|---|
| `001_init.sql` | Core schema: projects, api_keys, users, sessions, teams, memberships, _databases, collections, attributes, _indexes, documents, buckets, files |
| `002_deployments.sql` | Deployments table |
| `003_workflows.sql` | Workflows and workflow_executions |
| `004_console_users.sql` | Console admin users |
| `005_oauth.sql` | OAuth provider/id columns on users + index |
| `006_auth_extras.sql` | MFA (TOTP) + auth_tokens (magic link, verification, reset) |
| `007_functions.sql` | Functions and function_executions |
| `008_project_oauth.sql` | Per-project OAuth provider config |
| `009_relationships.sql` | Collection relationships |
| `010_organizations.sql` | Organizations, organization_members, project org_id |
| `011_workflow_features.sql` | Workflow credentials, folders, tags, versioning, templates |
| `012_p0_p1_features.sql` | Permissions, webhooks, platforms, project config, usage, deploy model |
| `013_webhooks_platforms_usage.sql` | Webhooks, webhook_deliveries, platforms, usage_metrics |
| `014_feature_flags.sql` | Feature flags, rules, overrides, evaluations |
| `015_p3_features.sql` | Custom domains, build agents, registry images |
| `016_migrations.sql` | External platform import tracking |
| `017_deploy_extras.sql` | Deploy templates, git connections, environments |
| `018_future_features.sql` | Audit, analytics, search, jobs, billing, vectors, content, regions, edge |
| `019_bucket_settings.sql` | Bucket file_security field |
| `020_messaging.sql` | Messages, topics, topic subscribers |

### MySQL-Specific Syntax in Migrations
- `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4` — every CREATE TABLE
- `DATETIME(3)` with `DEFAULT CURRENT_TIMESTAMP(3)` — all timestamp columns
- `ON UPDATE CURRENT_TIMESTAMP(3)` — all `updated_at` columns
- `TINYINT(1)` — all boolean columns
- `AUTO_INCREMENT` — messaging tables
- `LONGTEXT` — function source fields
- `VARCHAR(n)` with prefix-length indexes — `target(255)`
- Backtick quoting — reserved column names like `` `key` ``, table names like `` `_databases` ``

### Database CRUD (JSON Document Model)
- **File**: `backend/internal/databases/service.go` (1170 lines)
- User data stored as JSON in a generic `documents` table: `id, collection_id, database_id, data JSON, permissions, created_at, updated_at`
- Schema tracked in `collections` (tables) and `attributes` (columns) metadata tables
- Query builder (400+ lines) uses `JSON_EXTRACT`, `JSON_UNQUOTE`, `JSON_TYPE`, `CAST`, `LIKE` on JSON paths
- 15 query operators: equal, notEqual, lessThan, greaterThan, lessThanEqual, greaterThanEqual, contains, startsWith, endsWith, search, isNull, isNotNull, between, geo_near, geo_within
- Indexes stored in `_indexes` metadata table — not real database indexes
- Relationships stored in `collection_relationships` — not real foreign keys

### Files With Direct SQL (33 total)

**Service files (27):**
| File | MySQL-Specific Patterns |
|---|---|
| `internal/analytics/service.go` | `?` placeholders |
| `internal/appcache/service.go` | `?` placeholders |
| `internal/audit/service.go` | `?` placeholders |
| `internal/auth/service.go` | `?` placeholders |
| `internal/billing/service.go` | `ON DUPLICATE KEY UPDATE`, `?` |
| `internal/console/service.go` | `?` placeholders |
| `internal/content/service.go` | `?` placeholders |
| `internal/credentials/service.go` | `?` placeholders |
| `internal/databases/service.go` | `JSON_EXTRACT`, `JSON_UNQUOTE`, `JSON_TYPE`, backtick identifiers, `?` |
| `internal/deploy/service.go` | `DATE_FORMAT`, `?` |
| `internal/edge/service.go` | `?` placeholders |
| `internal/flags/service.go` | `ON DUPLICATE KEY UPDATE`, `?` |
| `internal/functions/service.go` | `?` placeholders |
| `internal/jobs/service.go` | `FOR UPDATE SKIP LOCKED`, `?` |
| `internal/messaging/service.go` | `DATE_FORMAT`, `GROUP_CONCAT`, `NOW()`, `?` |
| `internal/migrations/service.go` | `?` placeholders |
| `internal/oauth/project.go` | `ON DUPLICATE KEY UPDATE`, `?` |
| `internal/organizations/service.go` | `?` placeholders |
| `internal/projects/service.go` | `?` placeholders |
| `internal/regions/service.go` | `ON DUPLICATE KEY UPDATE`, `?` |
| `internal/search/service.go` | `ON DUPLICATE KEY UPDATE`, `?` |
| `internal/storage/service.go` | `?` placeholders |
| `internal/teams/service.go` | `?` placeholders |
| `internal/usage/service.go` | `DATE_FORMAT`, `?` |
| `internal/vectors/service.go` | `ON DUPLICATE KEY UPDATE`, `?` |
| `internal/webhooks/service.go` | `?` placeholders |
| `internal/workflows/service.go` | `ON DUPLICATE KEY UPDATE`, `?` |

**Worker files (6):**
| File | MySQL-Specific Patterns |
|---|---|
| `internal/worker/builds.go` | `?` placeholders |
| `internal/worker/cron.go` | `?` placeholders |
| `internal/worker/databases.go` | Backtick identifiers, `?` |
| `internal/worker/deletes.go` | `?` placeholders |
| `internal/worker/migrations.go` | `OPTIMIZE TABLE`, `NOW()` |
| `internal/worker/usage.go` | Backtick identifiers, `?` |

**Total `?` placeholder occurrences**: ~1550 across all backend Go files.

### Docker Services
- `mariadb` service in both `docker-compose.yml` (root) and `docker/docker-compose.yml`
- Image: `mariadb:11.2`
- Init script: `docker/mariadb/init.sql` (empty — just a comment)
- Volume: `db_data:/var/lib/mysql`
- Healthcheck: `healthcheck.sh --connect --innodb_initialized`

### Realtime
- **File**: `backend/internal/realtime/`
- WebSocket hub with manual event publishing
- Each service calls `s.events.Publish()` after writes
- Fragile: miss one call and clients don't get notified

---

## Target State

### PostgreSQL with Real Tables
- User-created tables are REAL PostgreSQL tables with real columns, real indexes, real foreign keys
- One PostgreSQL schema per user-database: `p_{projectId}_{databaseId}`
- Example: Project "acme", Database "main" → schema `p_acme_main` → table `p_acme_main.users`

### PostgREST as CRUD Engine
- All row-level CRUD (SELECT, INSERT, UPDATE, DELETE) on user tables proxied to PostgREST
- PostgREST auto-generates REST endpoints from the PostgreSQL schema
- Applad gateway injects auth context (JWT claims, schema selection) before proxying
- Filtering, embedding (joins), ordering, pagination — all handled by PostgREST's operator set
- PostgREST reloads schema cache when Applad creates/alters/drops tables (`NOTIFY pgrst, 'reload schema'`)

### Applad Go Gateway
- Single unified entry point — all requests go through Go server
- Auth, storage, functions, workflows, messaging, deploy — SQL converted from MySQL → PostgreSQL
- Database DDL — Go orchestrator creates/alters/drops real tables + generates RLS policies
- Database CRUD — proxied to PostgREST with auth context injection
- Schema switching via `Accept-Profile` / `Content-Profile` PostgREST headers

### Realtime from PostgreSQL
- PostgreSQL triggers on user tables fire `NOTIFY` on data changes
- Applad realtime service uses `LISTEN` to receive change events
- WebSocket hub broadcasts to subscribed clients
- All `events.Publish()` calls in service code are deleted
- Works for both Applad API writes AND direct PostgREST writes

### RLS (Row-Level Security)
- Auto-generated by Applad when tables are created
- Project isolation: `current_setting('applad.project_id') = '{projectId}'`
- User-level policies tied to Applad auth JWT claims
- Updated when permissions change via the Applad API

---

## Phase 1 — PostgreSQL Driver Swap

### 1.1 Replace Go driver dependency

**File**: `backend/go.mod`

Remove:
```
github.com/go-sql-driver/mysql v1.7.1
```

Add:
```
github.com/jackc/pgx/v5
```

Run: `go get github.com/jackc/pgx/v5 && go mod tidy`

### 1.2 Update database connection

**File**: `backend/internal/db/db.go`

Changes:
- Replace import `_ "github.com/go-sql-driver/mysql"` with `_ "github.com/jackc/pgx/v5/stdlib"`
- Change `sql.Open("mysql", dsn)` to `sql.Open("pgx", dsn)`
- Keep pool settings as-is (they work the same)

Target state:
```go
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Connect(dsn string) (*DB, error) {
	d, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	d.SetMaxOpenConns(25)
	d.SetMaxIdleConns(10)
	d.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return &DB{d}, nil
}
```

### 1.3 Update config default

**File**: `backend/internal/config/config.go`

Change default DSN from:
```
applad:applad@tcp(mariadb:3306)/applad?parseTime=true
```
To:
```
postgres://applad:applad@postgres:5432/applad?sslmode=disable
```

### 1.4 Update migration bootstrap

**File**: `backend/internal/db/migrate.go`

Change the `schema_migrations` bootstrap from:
```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    VARCHAR(32) NOT NULL PRIMARY KEY,
    applied_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
```
To:
```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    VARCHAR(32) NOT NULL PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)
```

Change placeholder from `?` to `$1` in:
- `SELECT COUNT(*) FROM schema_migrations WHERE version = ?` → `... version = $1`
- `INSERT INTO schema_migrations (version) VALUES (?)` → `... VALUES ($1)`

---

## Phase 2 — Migrations Rewrite

### Strategy
Consolidate all 20 MySQL migration files into a **single new PostgreSQL migration** (`001_init.sql`). Since this is a full replacement (not a live migration), there is no need to port each migration individually. The previous `schema_migrations` table will be empty in a fresh PostgreSQL database.

Delete all existing migration files and create one comprehensive PostgreSQL schema.

### MySQL → PostgreSQL Type Mapping

| MySQL | PostgreSQL |
|---|---|
| `VARCHAR(n)` | `VARCHAR(n)` (same) |
| `TEXT` | `TEXT` (same) |
| `LONGTEXT` | `TEXT` |
| `TINYINT(1)` | `BOOLEAN` |
| `INT AUTO_INCREMENT` | `SERIAL` or `GENERATED ALWAYS AS IDENTITY` |
| `BIGINT` | `BIGINT` (same) |
| `FLOAT` / `DOUBLE` | `DOUBLE PRECISION` |
| `DATETIME(3)` | `TIMESTAMPTZ` |
| `DEFAULT CURRENT_TIMESTAMP(3)` | `DEFAULT NOW()` |
| `ON UPDATE CURRENT_TIMESTAMP(3)` | Trigger function (see below) |
| `JSON` | `JSONB` |
| `BLOB` / `LONGBLOB` | `BYTEA` |
| `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4` | (remove entirely) |

### Updated timestamp trigger

MySQL's `ON UPDATE CURRENT_TIMESTAMP` has no direct PostgreSQL equivalent. Create a reusable trigger function:

```sql
CREATE OR REPLACE FUNCTION applad_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

Apply to every table with an `updated_at` column:
```sql
CREATE TRIGGER set_updated_at BEFORE UPDATE ON {table_name}
FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();
```

### Identifier quoting
- Replace all backtick-quoted identifiers (`` `key` ``, `` `_databases` ``) with double-quoted identifiers (`"key"`, `"_databases"`)
- Or better: rename reserved-word columns to avoid quoting (`key` → `api_key` or `secret`)

### Single consolidated migration file
- **New file**: `backend/internal/db/migrations/001_init.sql`
- Must create ALL tables from migrations 001–020 in PostgreSQL syntax
- Must include the `applad_set_updated_at()` trigger function
- Must attach the trigger to every table with `updated_at`
- Must include all indexes, foreign keys, and constraints
- Must include seed data (workflow templates from 011, deploy templates from 017)

### Also delete
- `backend/migrations/001_init.sql` (duplicate file at repo root)
- All files `002_*.sql` through `020_*.sql` in `backend/internal/db/migrations/`

---

## Phase 3 — Service SQL Conversion (Non-Database Services)

### Placeholder conversion

The biggest mechanical change: ~1550 `?` placeholders → numbered `$1, $2, $3...` placeholders.

**Recommended approach**: Create a helper function to avoid manual placeholder counting:

```go
// backend/internal/db/placeholder.go
package db

import "fmt"

// PH returns a PostgreSQL numbered placeholder: $1, $2, etc.
func PH(n int) string {
	return fmt.Sprintf("$%d", n)
}
```

However, for clarity and to avoid widespread refactoring of query-building patterns, the simpler approach is direct replacement: manually number each `?` in each query string. Most queries have 2–10 placeholders. For queries built dynamically (e.g., IN clauses), use a small helper to generate `$N` sequences.

### Conversion table for each affected service file

#### Files with ONLY `?` placeholder changes (20 files)
These need only `?` → `$N` conversion. No other MySQL-specific syntax.

| File | Approx. `?` count |
|---|---|
| `internal/analytics/service.go` | ~20 |
| `internal/appcache/service.go` | ~10 |
| `internal/audit/service.go` | ~15 |
| `internal/auth/service.go` | ~60 |
| `internal/console/service.go` | ~30 |
| `internal/content/service.go` | ~20 |
| `internal/credentials/service.go` | ~15 |
| `internal/edge/service.go` | ~15 |
| `internal/functions/service.go` | ~25 |
| `internal/migrations/service.go` | ~10 |
| `internal/organizations/service.go` | ~30 |
| `internal/projects/service.go` | ~40 |
| `internal/storage/service.go` | ~35 |
| `internal/teams/service.go` | ~25 |
| `internal/webhooks/service.go` | ~15 |
| `internal/worker/builds.go` | ~10 |
| `internal/worker/cron.go` | ~5 |
| `internal/worker/deletes.go` | ~10 |

#### Files with `ON DUPLICATE KEY UPDATE` → `ON CONFLICT ... DO UPDATE SET` (7 files)

| File | Context |
|---|---|
| `internal/billing/service.go` | Subscription/invoice upsert |
| `internal/flags/service.go` | Flag rule/override upsert |
| `internal/oauth/project.go` | OAuth provider config upsert |
| `internal/regions/service.go` | Region config upsert |
| `internal/search/service.go` | Document index upsert |
| `internal/vectors/service.go` | Vector upsert |
| `internal/workflows/service.go` | Workflow state upsert |

Pattern:
```sql
-- MySQL
INSERT INTO table (a, b, c) VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE b = VALUES(b), c = VALUES(c)

-- PostgreSQL
INSERT INTO table (a, b, c) VALUES ($1, $2, $3)
ON CONFLICT (a) DO UPDATE SET b = EXCLUDED.b, c = EXCLUDED.c
```

Note: You must identify the conflict target column(s) — usually the PRIMARY KEY or a UNIQUE constraint.

#### Files with `DATE_FORMAT` → `to_char` (3 files)

| File | MySQL | PostgreSQL |
|---|---|---|
| `internal/deploy/service.go` | `DATE_FORMAT(col, '%Y-%m-%d')` | `to_char(col, 'YYYY-MM-DD')` |
| `internal/messaging/service.go` | `DATE_FORMAT(col, '%Y-%m-%d')` | `to_char(col, 'YYYY-MM-DD')` |
| `internal/usage/service.go` | `DATE_FORMAT(col, '%Y-%m-%d')` | `to_char(col, 'YYYY-MM-DD')` |

#### File with `GROUP_CONCAT` → `string_agg` (1 file)

| File | MySQL | PostgreSQL |
|---|---|---|
| `internal/messaging/service.go` | `GROUP_CONCAT(col)` | `string_agg(col, ',')` |

#### File with `NOW()` (1 file — compatible)
`NOW()` works identically in PostgreSQL. No change needed in `internal/messaging/service.go`.

#### File with `FOR UPDATE SKIP LOCKED` (1 file — compatible)
PostgreSQL supports `FOR UPDATE SKIP LOCKED` natively. No change needed in `internal/jobs/service.go`.

#### File with `OPTIMIZE TABLE` → `VACUUM ANALYZE` (1 file)

| File | MySQL | PostgreSQL |
|---|---|---|
| `internal/worker/migrations.go` | `OPTIMIZE TABLE table_name` | Run `VACUUM ANALYZE table_name` (note: cannot run inside a transaction) |

#### Files with backtick identifiers → double quotes (3 files)

| File | MySQL | PostgreSQL |
|---|---|---|
| `internal/databases/service.go` | `` `key` `` | `"key"` |
| `internal/worker/databases.go` | `` `_databases` `` etc. | `"_databases"` |
| `internal/worker/usage.go` | backtick identifiers | double-quoted identifiers |

---

## Phase 4 — Database Service Redesign (Real Tables + PostgREST)

This is the largest single change. The databases service transforms from a JSON document CRUD engine into a **DDL orchestrator + PostgREST proxy**.

### 4.1 Schema-per-database model

When a user creates a database via the Applad API:
```go
func (s *Service) CreateDatabase(ctx context.Context, projectID, dbID, name string) (*model.Database, error) {
    schemaName := fmt.Sprintf("p_%s_%s", projectID, dbID)
    _, err := s.db.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA %s", pq.QuoteIdentifier(schemaName)))
    // Record in _databases metadata table
    // NOTIFY pgrst, 'reload schema'
}
```

### 4.2 Real table creation

When a user creates a table (collection):
```go
func (s *Service) CreateCollection(ctx context.Context, ...) (*model.Collection, error) {
    // Build CREATE TABLE DDL
    // Create in the correct schema: p_{projectId}_{databaseId}.{tableName}
    // Add default columns: id (UUID PK), created_at, updated_at
    // Create updated_at trigger
    // Generate RLS policies (see Phase 5)
    // NOTIFY pgrst, 'reload schema'
    // Record metadata in collections table
}
```

### 4.3 Real column management

When a user adds a column (attribute):
```go
func (s *Service) CreateAttribute(ctx context.Context, ...) (*model.Attribute, error) {
    // ALTER TABLE schema.table ADD COLUMN name TYPE [NOT NULL] [DEFAULT value]
    // NOTIFY pgrst, 'reload schema'
    // Record metadata in attributes table
}
```

Column type mapping from Applad types to PostgreSQL:
| Applad Type | PostgreSQL Type |
|---|---|
| `string` | `TEXT` |
| `integer` | `BIGINT` |
| `float` | `DOUBLE PRECISION` |
| `boolean` | `BOOLEAN` |
| `datetime` | `TIMESTAMPTZ` |
| `email` | `TEXT` + CHECK constraint |
| `url` | `TEXT` + CHECK constraint |
| `ip` | `INET` |
| `enum` | Custom `ENUM` type or `TEXT` + CHECK |
| `relationship` | Foreign key reference |

### 4.4 Real indexes

When a user creates an index:
```go
func (s *Service) CreateIndex(ctx context.Context, ...) (*model.Index, error) {
    // CREATE INDEX name ON schema.table (columns...)
    // For fulltext: CREATE INDEX name ON schema.table USING GIN (to_tsvector('english', column))
    // For unique: CREATE UNIQUE INDEX ...
    // NOTIFY pgrst, 'reload schema'
}
```

### 4.5 Real relationships (foreign keys)

```go
func (s *Service) CreateRelationship(ctx context.Context, ...) (*Relationship, error) {
    // ALTER TABLE schema.child ADD CONSTRAINT fk_name
    //   FOREIGN KEY (column) REFERENCES schema.parent(id)
    //   ON DELETE {CASCADE|SET NULL|RESTRICT}
    // PostgREST auto-detects FK relationships for embedding
    // NOTIFY pgrst, 'reload schema'
}
```

### 4.6 Row CRUD → PostgREST proxy

All document/row operations proxy to PostgREST:

```go
func (s *Service) CreateDocument(ctx context.Context, projectID, dbID, collID, docID string, data map[string]interface{}) (*model.Document, error) {
    schema := fmt.Sprintf("p_%s_%s", projectID, dbID)
    // Build PostgREST request:
    //   POST http://postgrest:3000/{tableName}
    //   Headers: Content-Profile: {schema}, Authorization: Bearer {jwt}
    //   Body: data (with id injected if provided)
    // Forward response back to client
}

func (s *Service) ListDocumentsWithQuery(ctx context.Context, ...) ([]*model.Document, int, error) {
    // Translate Applad query operators to PostgREST query params:
    //   equal       → ?column=eq.value
    //   notEqual    → ?column=neq.value
    //   lessThan    → ?column=lt.value
    //   greaterThan → ?column=gt.value
    //   contains    → ?column=like.*value*
    //   search      → ?column=fts.value  (full-text search)
    //   isNull      → ?column=is.null
    //   between     → ?column=gte.low&column=lte.high
    //   geo_near    → PostGIS: ?location=st_dwithin(geography, {lat},{lng}, {distance})
    //
    // Proxy to: GET http://postgrest:3000/{tableName}?{queryParams}
    //   Headers: Content-Profile: {schema}, Range: {offset}-{offset+limit-1}
}
```

### 4.7 Transaction support

PostgREST does not support multi-statement transactions. For `ExecuteTransaction`, either:
- Execute directly against PostgreSQL (not proxied) using a transaction, OR
- Use PostgREST's bulk insert endpoint + database-level constraints for atomicity

Recommended: Direct PostgreSQL execution for transactions, PostgREST for simple CRUD.

### 4.8 Code to delete

The following will be completely removed from `databases/service.go`:
- The entire JSON query builder (~400 lines of `JSON_EXTRACT`/`JSON_UNQUOTE`/`JSON_TYPE` logic)
- `validateDocData()` — real columns + PostgreSQL constraints replace this
- All direct SQL for document CRUD (`INSERT INTO documents ...`, `SELECT ... FROM documents ...`)
- The raw `documents` table concept entirely

### 4.9 CSV Import

`ImportCSV` will use PostgreSQL's native `COPY` command for bulk import instead of looping INSERT statements:
```sql
COPY schema.table (col1, col2, ...) FROM STDIN WITH (FORMAT csv, HEADER true)
```

---

## Phase 5 — RLS Policy Engine

### 5.1 New package or addition to databases service

Create RLS policy management. When a table is created, Applad generates PostgreSQL policies:

```sql
-- Enable RLS on the table
ALTER TABLE p_acme_main.users ENABLE ROW LEVEL SECURITY;

-- Force RLS even for table owners
ALTER TABLE p_acme_main.users FORCE ROW LEVEL SECURITY;

-- Project isolation policy
CREATE POLICY project_isolation ON p_acme_main.users
    USING (current_setting('applad.project_id', true) = '{projectId}');
```

### 5.2 Session variable injection

Before every request (both direct and PostgREST-proxied), Applad sets PostgreSQL session variables:

```sql
SET LOCAL applad.project_id = '{projectId}';
SET LOCAL applad.user_id = '{userId}';
SET LOCAL role = 'applad_user';  -- PostgREST uses this for RLS
```

For PostgREST, these are passed via JWT claims that PostgREST extracts automatically.

### 5.3 Permission-based policies

When users set permissions on a table via the API:
```sql
-- Read access for authenticated users
CREATE POLICY read_auth ON p_acme_main.users FOR SELECT
    USING (current_setting('applad.user_id', true) IS NOT NULL);

-- Write access for specific roles
CREATE POLICY write_admin ON p_acme_main.users FOR INSERT
    USING (current_setting('applad.role', true) = 'admin');

-- Row-level: user can only see their own documents
CREATE POLICY own_rows ON p_acme_main.users FOR ALL
    USING (owner_id = current_setting('applad.user_id', true));
```

### 5.4 PostgREST JWT configuration

PostgREST reads JWT claims to set PostgreSQL roles and session variables. Configure PostgREST with:
```
PGRST_JWT_SECRET={same as APPLAD_JWT_SECRET}
PGRST_DB_SCHEMAS=p_*                    # expose all project schemas
PGRST_DB_ANON_ROLE=applad_anon          # unauthenticated role
PGRST_DB_PRE_REQUEST=applad.check_jwt   # pre-request function for auth
```

---

## Phase 6 — Realtime Rewrite (PostgreSQL LISTEN/NOTIFY)

### 6.1 Change notification trigger

Create a reusable trigger function for change notification:

```sql
CREATE OR REPLACE FUNCTION applad_notify_change()
RETURNS TRIGGER AS $$
DECLARE
    payload JSONB;
BEGIN
    payload = jsonb_build_object(
        'table', TG_TABLE_NAME,
        'schema', TG_TABLE_SCHEMA,
        'action', TG_OP,
        'old', CASE WHEN TG_OP IN ('UPDATE', 'DELETE') THEN row_to_json(OLD)::jsonb ELSE NULL END,
        'new', CASE WHEN TG_OP IN ('INSERT', 'UPDATE') THEN row_to_json(NEW)::jsonb ELSE NULL END,
        'timestamp', NOW()
    );
    PERFORM pg_notify('applad_changes', payload::text);
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;
```

### 6.2 Auto-attach trigger on table creation

When Applad creates a user table, also attach the notification trigger:

```sql
CREATE TRIGGER notify_changes
    AFTER INSERT OR UPDATE OR DELETE ON p_acme_main.users
    FOR EACH ROW EXECUTE FUNCTION applad_notify_change();
```

### 6.3 Rewrite realtime service

**File**: `backend/internal/realtime/`

Replace the manual event publishing hub with a PostgreSQL listener:

```go
func (s *Service) Start(ctx context.Context) {
    // Use pgx directly (not database/sql) for LISTEN:
    conn, _ := pgx.Connect(ctx, dsn)
    conn.Exec(ctx, "LISTEN applad_changes")

    for {
        notification, _ := conn.WaitForNotification(ctx)
        // Parse payload JSON
        // Route to subscribed WebSocket clients based on schema + table
    }
}
```

### 6.4 Delete manual event publishing

Remove all `s.events.Publish(...)` calls from every service file. The trigger handles it automatically for all database writes — including writes that come through PostgREST.

Non-database events (storage uploads, function executions, workflow completions) can still use `pg_notify` directly or keep using the Redis pub/sub channel for non-table events.

---

## Phase 7 — PostgREST Integration

### 7.1 PostgREST proxy in Go

Add a reverse proxy in the databases handler that forwards row CRUD to PostgREST:

```go
// backend/internal/databases/proxy.go
func (h *Handler) proxyToPostgREST(w http.ResponseWriter, r *http.Request, schema, table string) {
    // Build PostgREST URL: http://postgrest:3000/{table}
    // Set headers:
    //   Content-Profile: {schema}
    //   Accept-Profile: {schema}
    //   Authorization: Bearer {signed JWT with project_id, user_id, role claims}
    // Forward query params (PostgREST filter syntax)
    // Proxy response back to client
}
```

### 7.2 Applad query operator → PostgREST translation

| Applad Operator | PostgREST Syntax |
|---|---|
| `equal(col, val)` | `?col=eq.val` |
| `notEqual(col, val)` | `?col=neq.val` |
| `lessThan(col, val)` | `?col=lt.val` |
| `greaterThan(col, val)` | `?col=gt.val` |
| `lessThanEqual(col, val)` | `?col=lte.val` |
| `greaterThanEqual(col, val)` | `?col=gte.val` |
| `contains(col, val)` | `?col=like.*val*` |
| `startsWith(col, val)` | `?col=like.val*` |
| `endsWith(col, val)` | `?col=like.*val` |
| `search(col, val)` | `?col=fts.val` |
| `isNull(col)` | `?col=is.null` |
| `isNotNull(col)` | `?col=not.is.null` |
| `between(col, a, b)` | `?col=gte.a&col=lte.b` |
| `orderAsc(col)` | `?order=col.asc` |
| `orderDesc(col)` | `?order=col.desc` |
| `limit(n)` | `?limit=n` |
| `offset(n)` | `?offset=n` |
| `select(cols)` | `?select=col1,col2` |

### 7.3 PostgREST schema reload

After any DDL operation (create/alter/drop table, column, index, relationship):

```go
_, err := s.db.Exec("NOTIFY pgrst, 'reload schema'")
```

### 7.4 PostgREST health check

Add PostgREST to the health check endpoint (`/v1/health`):

```go
func (h *Handler) checkPostgREST() error {
    resp, err := http.Get("http://postgrest:3000/")
    // Check for 200 OK
}
```

---

## Phase 8 — Docker & Infrastructure

### 8.1 Replace MariaDB with PostgreSQL

In both `docker-compose.yml` (root) and `docker/docker-compose.yml`:

Remove the `mariadb` service. Add:

```yaml
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: "applad"
      POSTGRES_USER: "applad"
      POSTGRES_PASSWORD: "${DB_PASSWORD:-applad}"
    volumes:
      - db_data:/var/lib/postgresql/data
      - ./docker/postgres/init.sql:/docker-entrypoint-initdb.d/init.sql:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U applad"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped
```

### 8.2 Add PostgREST service

```yaml
  postgrest:
    image: postgrest/postgrest:v12.2.3
    environment:
      PGRST_DB_URI: "postgres://applad:${DB_PASSWORD:-applad}@postgres:5432/applad"
      PGRST_DB_SCHEMAS: "public"
      PGRST_DB_ANON_ROLE: "applad_anon"
      PGRST_JWT_SECRET: "${JWT_SECRET:-change-me-in-production}"
      PGRST_DB_PRE_REQUEST: "applad.check_jwt"
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped
```

Note: `PGRST_DB_SCHEMAS` will be dynamically updated or use a wildcard pattern as project schemas are created.

### 8.3 PostgreSQL init script

Create `docker/postgres/init.sql`:

```sql
-- Create roles for PostgREST
CREATE ROLE applad_anon NOLOGIN;
CREATE ROLE applad_user NOLOGIN;

-- Grant applad_user to the applad role so PostgREST can switch
GRANT applad_anon TO applad;
GRANT applad_user TO applad;

-- Internal schema for Applad helper functions
CREATE SCHEMA IF NOT EXISTS applad;

-- JWT pre-request function for PostgREST
CREATE OR REPLACE FUNCTION applad.check_jwt() RETURNS void AS $$
DECLARE
    proj_id text;
    usr_id text;
BEGIN
    proj_id := current_setting('request.jwt.claims', true)::json->>'project_id';
    usr_id  := current_setting('request.jwt.claims', true)::json->>'user_id';

    IF proj_id IS NOT NULL THEN
        PERFORM set_config('applad.project_id', proj_id, true);
    END IF;
    IF usr_id IS NOT NULL THEN
        PERFORM set_config('applad.user_id', usr_id, true);
    END IF;
END;
$$ LANGUAGE plpgsql;
```

### 8.4 Delete MariaDB artifacts

- Delete `docker/mariadb/` directory
- Remove `mariadb` references from nginx config if applicable

### 8.5 Update environment variables

| Variable | Old Default | New Default |
|---|---|---|
| `DATABASE_DSN` | `applad:applad@tcp(mariadb:3306)/applad?parseTime=true` | `postgres://applad:applad@postgres:5432/applad?sslmode=disable` |
| `POSTGREST_URL` | (new) | `http://postgrest:3000` |

### 8.6 Update all worker service DSNs

Every worker binary in `backend/cmd/workers/` connects to the database. Update their docker-compose environment to use the new PostgreSQL DSN.

### 8.7 Update proxy/nginx config

In `docker/nginx/nginx.conf`, no changes should be needed — the proxy routes to the `api` service, which is unchanged. PostgREST is internal-only (not directly exposed).

---

## Phase 9 — Test Updates

### 9.1 Placeholder updates in all test files

Every `go-sqlmock` expectation that matches a SQL query string must be updated:
- `?` → `\$1`, `\$2`, etc. in regex expectations
- `ON DUPLICATE KEY UPDATE` → `ON CONFLICT ... DO UPDATE SET`
- `DATE_FORMAT` → `to_char`
- `GROUP_CONCAT` → `string_agg`
- `OPTIMIZE TABLE` → `VACUUM ANALYZE`

### 9.2 Database service tests — full rewrite

`backend/internal/databases/service_test.go` and `handler_test.go` need complete rewriting:
- Test DDL generation (CREATE TABLE, ALTER TABLE, DROP TABLE)
- Test PostgREST proxy request construction
- Test RLS policy generation
- Test schema creation/deletion
- Mock PostgREST responses for CRUD operations using `httptest.NewServer`

### 9.3 Realtime tests — rewrite

Test the PostgreSQL LISTEN/NOTIFY flow instead of manual event publishing.

### 9.4 Integration tests

`backend/tests/integration_test.go` — update to use PostgreSQL container (e.g., via testcontainers-go or a test docker-compose).

---

## Phase 10 — SDK, Console & SQL Editor

### 10.1 SDKs

The SDK interface should remain **mostly unchanged** — the Applad API contract stays the same. The backend implementation changes but the HTTP API shape is preserved.

Changes needed if exposing PostgREST-style direct queries as an advanced feature:
- Add `directQuery()` method to database services in each SDK
- Document PostgREST filter syntax in SDK docs

### 10.2 Console (Flutter Web)

- Update database table views to show real column types (not JSON-stored attribute metadata)
- Update table creation forms to reflect real PostgreSQL types
- Update index creation to show actual PostgreSQL index types (B-tree, GIN, GiST)
- Health page: show PostgreSQL + PostgREST status
- Remove any references to MariaDB in settings/UI

### 10.3 SQL Editor (Console — new feature)

Add an interactive SQL editor to the console at `/databases/{dbId}/sql`. This is a key differentiator — Supabase has one, Appwrite and Directus do not.

**Backend endpoint**: `POST /v1/databases/{databaseId}/sql`
- Accepts raw SQL query string in request body
- Executes against the user's project schema (`p_{projectId}_{databaseId}`)
- Returns rows as JSON (column names + typed values)
- Read-only by default (`SET TRANSACTION READ ONLY`), opt-in write mode via request flag
- Query timeout enforced (e.g., 30 seconds via `statement_timeout`)
- DDL statements (`CREATE`, `ALTER`, `DROP`) are blocked — schema changes must go through the Applad API to maintain RLS policies and PostgREST schema cache
- Must set `applad.project_id` and `applad.user_id` session variables before execution so RLS is enforced

**Console UI features**:
- **Editor**: Monaco-based SQL editor with syntax highlighting, autocomplete (table/column names from schema), and keyboard shortcuts (Cmd+Enter to run)
- **Results table**: Paginated data grid with sortable columns, row count, and execution time display
- **Export**: Download results as CSV or JSON
- **Query history**: Persisted per-session, with re-run and copy buttons
- **Schema sidebar**: Collapsible tree showing tables → columns (with types) in the current database, pulled from PostgreSQL `information_schema`
- **Safety**: Write mode requires explicit toggle + confirmation dialog. Destructive queries show a warning.
- **Error display**: PostgreSQL error messages shown inline with line/position highlighting in the editor

---

## Phase 11 — Documentation

### 11.1 CLAUDE.md

Update all references:
- MariaDB → PostgreSQL
- `mariadb` service → `postgres` service
- Add `postgrest` to service table
- Update DSN format
- Update migration count
- Document schema-per-database model
- Document RLS policy approach
- Update env var table

### 11.2 README.md

Update setup instructions, Docker service list, architecture diagram.

### 11.3 OpenAPI spec

`backend/api/openapi.yaml` — add PostgREST-style direct query parameters if exposing them. Otherwise, no changes (API shape is preserved).

---

## Constraints & Rules

1. **No MariaDB/MySQL code may remain** — zero backward compatibility. This is a clean break.
2. **PostgREST is a core service, not optional** — it must start with `docker compose up` and be health-checked.
3. **All DDL operations go through Applad Go code** — PostgREST never creates/alters/drops tables.
4. **RLS must be enabled on every user-created table** — no exceptions.
5. **The `applad` schema is reserved** for internal helper functions, metadata tables, and the migration tracking table.
6. **User-created schemas** follow the pattern `p_{projectId}_{databaseId}` — sanitize project and database IDs to be valid PostgreSQL identifiers.
7. **Realtime triggers must be attached to every user-created table** automatically.
8. **`NOTIFY pgrst, 'reload schema'`** must be called after every DDL change.
9. **The Applad API contract (HTTP routes, request/response shapes) must not break** — SDKs should work with no or minimal changes.
10. **Security**: All user-provided identifiers (table names, column names, schema names) must be sanitized using `pq.QuoteIdentifier()` or equivalent — never string-interpolated directly into SQL.
11. **Console must stay Flutter Web** — permanent decision, no stack replacement.
12. **No `collections`/`documents`/`attributes` terminology in new code** — use `tables`/`rows`/`columns` exclusively. Delete all backward-compatibility aliases. The MongoDB/Appwrite-style naming is gone.

---

## Progress Checklist

### Phase 1 — PostgreSQL Driver Swap
- [x] Replace `go-sql-driver/mysql` with `pgx/v5` in `go.mod`
- [x] Update `db.go`: driver import + `sql.Open("pgx", dsn)`
- [x] Update `config.go`: default DSN to PostgreSQL format
- [x] Update `migrate.go`: bootstrap SQL + placeholder conversion
- [x] Run `go build ./...` — compiles cleanly

### Phase 2 — Migrations Rewrite
- [x] Delete all 20 existing MySQL migration files
- [x] Delete `backend/migrations/001_init.sql` (duplicate at repo root)
- [x] Create single consolidated `001_init.sql` in PostgreSQL syntax
- [x] Include `applad_set_updated_at()` trigger function
- [x] Include triggers on all tables with `updated_at`
- [x] Include all indexes, foreign keys, constraints
- [x] Include seed data (workflow templates, deploy templates)
- [x] Verify migration runs on fresh PostgreSQL database

### Phase 3 — Service SQL Conversion
- [x] Convert `internal/analytics/service.go` (`?` → `$N`)
- [x] Convert `internal/appcache/service.go` (`?` → `$N`)
- [x] Convert `internal/audit/service.go` (`?` → `$N`)
- [x] Convert `internal/auth/service.go` (`?` → `$N`)
- [x] Convert `internal/billing/service.go` (`?` → `$N`, upsert)
- [x] Convert `internal/console/service.go` (`?` → `$N`)
- [x] Convert `internal/content/service.go` (`?` → `$N`)
- [x] Convert `internal/credentials/service.go` (`?` → `$N`)
- [x] Convert `internal/deploy/service.go` (`?` → `$N`, `DATE_FORMAT` → `to_char`)
- [x] Convert `internal/edge/service.go` (`?` → `$N`)
- [x] Convert `internal/flags/service.go` (`?` → `$N`, upsert)
- [x] Convert `internal/functions/service.go` (`?` → `$N`)
- [x] Convert `internal/jobs/service.go` (`?` → `$N`)
- [x] Convert `internal/messaging/service.go` (`?` → `$N`, `DATE_FORMAT`, `GROUP_CONCAT`, `NOW()`)
- [x] Convert `internal/migrations/service.go` (`?` → `$N`)
- [x] Convert `internal/oauth/project.go` (`?` → `$N`, upsert)
- [x] Convert `internal/organizations/service.go` (`?` → `$N`)
- [x] Convert `internal/projects/service.go` (`?` → `$N`)
- [x] Convert `internal/regions/service.go` (`?` → `$N`, upsert)
- [x] Convert `internal/search/service.go` (`?` → `$N`, upsert)
- [x] Convert `internal/storage/service.go` (`?` → `$N`)
- [x] Convert `internal/teams/service.go` (`?` → `$N`)
- [x] Convert `internal/usage/service.go` (`?` → `$N`, `DATE_FORMAT`)
- [x] Convert `internal/vectors/service.go` (`?` → `$N`, upsert)
- [x] Convert `internal/webhooks/service.go` (`?` → `$N`)
- [x] Convert `internal/workflows/service.go` (`?` → `$N`, upsert)
- [x] Convert `internal/worker/builds.go` (`?` → `$N`)
- [x] Convert `internal/worker/cron.go` (`?` → `$N`)
- [x] Convert `internal/worker/databases.go` (`?` → `$N`, backtick → double-quote)
- [x] Convert `internal/worker/deletes.go` (`?` → `$N`)
- [x] Convert `internal/worker/migrations.go` (`OPTIMIZE TABLE` → `VACUUM ANALYZE`, `NOW()`)
- [x] Convert `internal/worker/usage.go` (`?` → `$N`, backtick → double-quote)
- [x] Run `go build ./...` — compiles cleanly

### Phase 4 — Database Service Redesign
- [x] Implement schema-per-database: `CREATE SCHEMA p_{projectId}_{databaseId}`
- [x] Rewrite `CreateCollection` → real `CREATE TABLE` DDL
- [x] Rewrite `CreateAttribute` → real `ALTER TABLE ADD COLUMN`
- [x] Rewrite `DeleteAttribute` → `ALTER TABLE DROP COLUMN`
- [x] Rewrite `CreateIndex` → real `CREATE INDEX` (B-tree, GIN, UNIQUE)
- [x] Rewrite `CreateRelationship` → real `FOREIGN KEY` constraints
- [x] Implement PostgREST proxy for `CreateDocument`
- [x] Implement PostgREST proxy for `GetDocument`
- [x] Implement PostgREST proxy for `ListDocuments` / `ListDocumentsWithQuery`
- [x] Implement Applad query → PostgREST query param translation
- [x] Implement PostgREST proxy for `UpdateDocument`
- [x] Implement PostgREST proxy for `DeleteDocument`
- [x] Implement `ExecuteTransaction` via direct PostgreSQL (not PostgREST)
- [x] Rewrite `ImportCSV` to use PostgreSQL `COPY`
- [x] Delete the entire JSON query builder code
- [x] Delete `validateDocData` (replaced by real column constraints)
- [x] Implement `NOTIFY pgrst, 'reload schema'` after all DDL ops
- [x] Rewrite `databases/handler.go` routes for new service methods

### Phase 5 — RLS Policy Engine
- [x] Create RLS policy generator (project isolation policy)
- [x] Auto-enable RLS on every user-created table
- [x] Implement `SET LOCAL applad.project_id` session variable injection
- [x] Implement `SET LOCAL applad.user_id` session variable injection
- [x] Implement permission-based policies (read/write per role)
- [x] Implement row-level ownership policies
- [x] Generate JWT for PostgREST with project_id + user_id claims
- [ ] Test RLS enforcement via PostgREST

### Phase 6 — Realtime Rewrite
- [x] Create `applad_notify_change()` trigger function in PostgreSQL
- [x] Auto-attach notification trigger on user-table creation
- [x] Rewrite realtime service to use `pgx` LISTEN/NOTIFY
- [x] Parse notification payload and route to WebSocket subscribers
- [x] Remove all `s.events.Publish()` calls from service files
- [ ] Keep Redis pub/sub for non-database events (storage, functions, etc.)
- [ ] Test: insert via API → WebSocket client receives event
- [ ] Test: insert via PostgREST → WebSocket client receives event

### Phase 7 — PostgREST Integration
- [x] Create `databases/proxy.go` — reverse proxy to PostgREST
- [x] Implement schema selection via `Content-Profile` / `Accept-Profile` headers
- [x] Implement auth context injection (JWT with claims)
- [x] Implement Applad query operator → PostgREST syntax translation
- [x] Implement pagination translation (Range header)
- [x] Implement embedding (relationship joins via PostgREST)
- [x] Add PostgREST to health check endpoint
- [x] Implement `NOTIFY pgrst, 'reload schema'` signal helper

### Phase 8 — Docker & Infrastructure
- [x] Delete `docker/mariadb/` directory
- [x] Create `docker/postgres/init.sql` with roles + helper functions
- [x] Replace `mariadb` service with `postgres` in root `docker-compose.yml`
- [x] Replace `mariadb` service with `postgres` in `docker/docker-compose.yml`
- [x] Add `postgrest` service to root `docker-compose.yml`
- [x] Add `postgrest` service to `docker/docker-compose.yml`
- [x] Update all worker service environment variables (DSN)
- [x] Update API service environment variable (DSN + POSTGREST_URL)
- [x] Update volume names (`db_data` mount path)
- [x] Verify `docker compose up` starts all services healthy
- [x] Verify migrations run on fresh start

### Phase 9 — Test Updates
- [ ] Update all sqlmock expectations in test files (`?` → `$N` regex)
- [ ] Update upsert expectations (`ON CONFLICT`)
- [ ] Update `DATE_FORMAT` expectations → `to_char`
- [ ] Update `GROUP_CONCAT` expectations → `string_agg`
- [x] Rewrite `databases/service_test.go` (DDL + PostgREST mock)
- [ ] Rewrite `databases/handler_test.go`
- [ ] Rewrite realtime tests (LISTEN/NOTIFY instead of manual publish)
- [ ] Update integration tests for PostgreSQL
- [x] Run `go test ./...` — all tests pass
- [x] Run `go vet ./...` — clean

### Phase 10 — SDK, Console & SQL Editor
- [x] Verify all 5 SDKs work against new API (no contract changes for basic CRUD)
- [x] Update Dart SDK tests if any API response shapes changed
- [x] Update JS SDK tests if any API response shapes changed
- [x] Update Node SDK tests if any API response shapes changed
- [x] Update Go SDK tests if any API response shapes changed
- [x] Update Python SDK tests if any API response shapes changed
- [x] Update Flutter console: database table views for real column types
- [x] Update Flutter console: table creation forms with PostgreSQL types
- [x] Update Flutter console: index creation with PostgreSQL index types
- [x] Update Flutter console: health page (PostgreSQL + PostgREST status)
- [x] Remove any MariaDB references from console UI
- [x] Add backend SQL execution endpoint: `POST /v1/databases/{databaseId}/sql`
- [x] Enforce read-only default + opt-in write mode with `SET TRANSACTION READ ONLY`
- [x] Block DDL statements in SQL editor endpoint
- [x] Enforce query timeout (`statement_timeout`)
- [x] Set RLS session variables before SQL execution
- [x] Add SQL editor page to console: `/databases/{dbId}/sql`
- [x] Integrate Monaco editor with SQL syntax highlighting
- [x] Implement SQL autocomplete from schema metadata (tables, columns, types)
- [x] Implement results data grid with pagination, sorting, row count, execution time
- [x] Implement export (CSV + JSON download)
- [x] Implement query history (per-session, re-run, copy)
- [x] Implement schema sidebar (tables → columns tree from `information_schema`)
- [x] Implement write mode toggle with confirmation dialog
- [x] Implement PostgreSQL error display with line/position highlighting

---

## Phase 11 — Terminology Cleanup

With real PostgreSQL tables, the MongoDB/Appwrite-style naming is eliminated. The codebase unifies on database terminology everywhere.

### 11.1 What changes

| Old (MongoDB-style) | New (Database-style) | Scope |
|---|---|---|
| `collections` (MySQL table) | `tables` (PostgreSQL table) | Metadata table in Applad's internal schema |
| `documents` (MySQL table) | **Deleted entirely** | User data now lives in real PostgreSQL tables |
| `attributes` (MySQL table) | `columns` (PostgreSQL table) | Metadata table in Applad's internal schema |
| `_indexes` (MySQL table) | `indexes` (PostgreSQL table) | Metadata table — underscore prefix no longer needed |
| `collection_relationships` (MySQL table) | `table_relationships` | Metadata table |
| `collection_id` (column) | `table_id` | Foreign key columns across metadata tables |
| `model.Collection` (Go alias) | **Delete alias** — use `model.Table` only | `backend/internal/model/model.go` |
| `model.Document` (Go alias) | **Delete alias** — use `model.Row` only | `backend/internal/model/model.go` |
| `model.Attribute` (Go alias) | **Delete alias** — use `model.Column` only | `backend/internal/model/model.go` |
| `CreateCollection()` | `CreateTable()` | `databases/service.go` + `handler.go` |
| `CreateDocument()` | `CreateRow()` | `databases/service.go` + `handler.go` |
| `CreateAttribute()` | `CreateColumn()` | `databases/service.go` + `handler.go` |
| `createCollection` (handler) | `createTable` | `databases/handler.go` |
| `createDocument` (handler) | `createRow` | `databases/handler.go` |
| `collID` (variable) | `tableID` | Throughout databases service |
| `docID` (variable) | `rowID` | Throughout databases service |

### 11.2 Current state

The model types are already correct — `Table`, `Column`, `Row` are the primary types in `backend/internal/model/model.go`. The aliases (`Collection = Table`, `Attribute = Column`, `Document = Row`) exist for backward compatibility. With this migration, delete the aliases entirely.

The API routes already use `/tables/` and `/rows/` externally. The internal code still uses `collection`/`document` in:
- Function names: `CreateCollection`, `CreateDocument`, `CreateAttribute`
- Variable names: `collID`, `docID`
- SQL table names: `collections`, `documents`, `attributes`
- Handler names: `createCollection`, `createDocument`
- Test names: `TestCreateCollection`, `TestCreateDocument`

### 11.3 Approach

Since Phase 4 rewrites the databases service anyway, the new code should use the new names from the start. For other packages that reference `model.Collection`/`model.Document`/`model.Attribute`, do a codebase-wide rename after deleting the aliases.

The consolidated PostgreSQL migration (Phase 2) should use the new table names (`tables`, `columns`, `indexes`, `table_relationships`) from day one — there is no backward compatibility concern since this is a fresh schema.

---

## Phase 11 — Terminology Cleanup
- [x] Delete `model.Collection` alias (use `model.Table` only)
- [x] Delete `model.Document` alias (use `model.Row` only)
- [x] Delete `model.Attribute` alias (use `model.Column` only)
- [x] Rename metadata table `collections` → `tables` in PostgreSQL migration
- [x] Delete `documents` metadata table entirely (data in real tables)
- [x] Rename metadata table `attributes` → `columns` in PostgreSQL migration
- [x] Rename metadata table `_indexes` → `indexes` in PostgreSQL migration
- [x] Rename metadata table `collection_relationships` → `table_relationships`
- [x] Rename all `collection_id` foreign key columns → `table_id`
- [x] Rename `CreateCollection` → `CreateTable` in databases service
- [x] Rename `CreateDocument` → `CreateRow` in databases service
- [x] Rename `CreateAttribute` → `CreateColumn` in databases service
- [x] Rename all `collID` variables → `tableID` in databases service
- [x] Rename all `docID` variables → `rowID` in databases service
- [x] Rename handler functions: `createCollection` → `createTable`, etc.
- [x] Update all test names and references
- [x] Search codebase for any remaining `collection`/`document`/`attribute` references in database context
- [x] Update `worker/databases.go` references
- [x] Run `go build ./...` — compiles cleanly
- [x] Run `go test ./...` — all tests pass

### Phase 12 — Documentation
- [x] Update `CLAUDE.md` — all MariaDB → PostgreSQL references
- [x] Update `CLAUDE.md` — add PostgREST to service table
- [x] Update `CLAUDE.md` — document schema-per-database model
- [x] Update `CLAUDE.md` — remove collections/documents/attributes terminology
- [x] Update `CLAUDE.md` — update env var table
- [x] Update `CLAUDE.md` — update migration documentation
- [x] Update `README.md` — setup instructions
- [x] Update `README.md` — architecture diagram
- [x] Update `backend/api/openapi.yaml` if any API changes
- [x] Final `docker compose up` — full stack runs end to end
