# ============================================================

# INSTANCE

# ============================================================

# applad init

# Creates the initial Applad project structure in the current

# directory. Generates applad.yaml, the orgs/ directory tree,

# a .gitignore that excludes all .env files, and an initial

# .env.example at the instance level. Run this once when

# starting a new Applad project.

applad init

# --template lets you start from a pre-built scaffold instead

# of a blank slate. saas includes auth, multi-tenancy, billing

# tables, and common messaging templates. api is a leaner

# REST-only setup. cms adds content tables and hosting config.

# minimal is the absolute bare minimum to get started.

applad init --template saas # saas | api | cms | minimal

# applad up

# The single most important command. Reads your entire config

# tree, compares it to the current state of your infrastructure,

# and makes reality match your config. Connects to VPS targets

# over SSH, provisions Docker containers, configures Caddy,

# connects database adapters, sets up cloud adapters — then

# disconnects. Agentless — nothing persistent runs on your

# servers on Applad's behalf. Think of it like terraform apply

# but for your entire backend: tables, auth, storage, functions,

# deployments, realtime, messaging — all of it.

applad up

# Same as above but only reconciles a single named environment.

# Useful when you want to push changes to staging without

# touching production, or when setting up a new environment

# for the first time.

applad up --env production

# Watches your config files for changes and automatically

# runs applad up whenever a file is saved. Intended for local

# development only — do not use this in production or CI.

applad up --watch

# Shows exactly what applad up would do without actually doing

# anything. Like terraform plan. Shows which SSH connections

# would be opened, which Docker operations would run, which

# source repos would be fetched, which cloud resources would

# be provisioned or torn down, which config has drifted, and

# which migrations are pending. Always run this before applying

# changes to a production environment.

applad up --dry-run

# applad down

# Stops the local running Applad instance. For development use.

applad down

# Tears down all infrastructure for a specific environment —

# stops containers, removes services, disconnects adapters.

# SSHes into the target, cleans up, and leaves. Use with care

# in production. Does not delete your database or stored files.

applad down --env staging

# applad status

# Shows the health and connectivity status of your entire

# Applad instance across all environments. Tells you whether

# each environment's infrastructure is reachable, whether

# services are running, whether adapter connections are healthy,

# and whether any config has drifted from the running state.

applad status

# Same as above but scoped to a single environment. Useful for

# quickly checking whether production is healthy without the

# noise of other environments.

applad status --env production

# applad --version

# Prints the currently installed Applad version.

applad --version
applad -v

# applad upgrade

# Upgrades the Applad binary to the latest stable release.

applad upgrade

# applad audit

# Scans all function dependencies across all runtimes (Node,

# Python, Go, etc.) for known security vulnerabilities. Reports

# findings by severity. By default, critical vulnerabilities

# will also block deployments — this command lets you check

# proactively before triggering a deploy.

applad audit

# ============================================================

# CONFIG

# Manages the config file tree and its relationship to the

# running instance. The config tree is the source of truth —

# these commands help you keep it in sync.

# ============================================================

# Validates the entire config tree for correctness without

# starting or changing anything. Checks syntax, required

# fields, cross-references between files (e.g. a workflow

# referencing a function that doesn't exist), security policy

# completeness, and that all ${VAR} references have

# corresponding entries in your .env. Safe to run at any time.

applad config validate

# Shows the difference between your local config tree and the

# config that the running Applad instance is currently using.

# Useful for seeing what would change before pushing, or for

# understanding why your running instance is behaving

# differently from what's in your files.

applad config diff

# Pushes your local config tree to the running Applad instance,

# making it the new active config. Every push is cryptographically

# signed with your SSH key and recorded in the audit trail with

# a full diff of what changed. The running instance reloads

# affected modules without downtime.

applad config push

# Pulls the config currently running on the instance back to

# your local files. Useful if someone pushed changes through

# the admin UI or another developer pushed directly and you

# want to sync your local files to match.

applad config pull

# Exports the entire config tree as a zip archive. Environment

# variable values are redacted — only the key names are included.

# Useful for sharing a sanitized snapshot of your configuration,

# for backup, or for migrating to a new instance.

applad config export

# Shows how the full config tree merges together in memory

# without actually starting or changing anything. Applad merges

# dozens of yaml files into one resolved config at startup —

# this lets you see the result of that merge and catch conflicts

# or unexpected overrides before they cause issues.

applad config merge --dry-run

# ============================================================

# ENV

# Manages environment variables and .env.example generation.

# Every ${VAR} reference in your yaml files is tracked by

# Applad. .env.example files are auto-generated and scoped

# to each level of the org/project hierarchy.

# ============================================================

# Scans all yaml files in the active project, extracts every

# ${VAR} reference, and generates a .env.example file

# annotated with which config file uses each variable, what

# format it expects, and whether it should be treated as a

# secret. Run this after adding any new ${VAR} reference to

# any yaml file.

applad env generate

# Same as above but generates .env.example files for every

# project within the named org. Each project gets its own

# scoped .env.example containing only the variables that

# project references.

applad env generate --org acme-corp

# Generates .env.example files for every project across the

# entire instance. Useful after a large refactor or when

# onboarding a new instance for the first time.

applad env generate --all

# Generates a .env.example scoped only to the variables needed

# for a specific environment. Useful for production deployments

# where you only want to see the variables that actually matter

# for that environment — skipping development-only overrides.

applad env generate --env production

# Checks that every ${VAR} reference in the config tree has

# a corresponding value set in the environment. Fails with a

# clear error message naming the missing variable and the exact

# config file that references it. applad up runs this

# automatically on startup — use this to check manually before

# a deploy.

applad env validate

# Same as above but only validates variables needed for the

# named environment. Useful for pre-flight checks before

# deploying to production.

applad env validate --env production

# Shows the difference between the variables referenced in

# your config files and what's actually set in your .env.

# Highlights variables that are referenced but missing, and

# variables that are set in .env but not referenced anywhere

# in the config (potential dead config).

applad env diff

# Lists all environment variable keys currently set for the

# active project. Values are never shown — keys only. This is

# intentional: secrets should never appear in terminal output.

applad env list

# Same as above but for a specific project rather than the

# currently active one.

applad env list --project <id>

# Sets an environment variable for the running instance.

# Takes effect immediately without requiring a restart.

applad env set KEY=value

# Removes an environment variable from the running instance.

applad env unset KEY

# Pulls the environment variables from the running instance

# down to a local .env file. Values are included. Use with

# care — the resulting .env file contains real secrets and

# should never be committed to version control. Applad's

# .gitignore excludes .env files automatically.

applad env pull

# Pushes your local .env file to the running instance, setting

# all variables in one operation. Variables not present in

# the .env file are left unchanged on the instance.

applad env push

# ============================================================

# CLOUD

# A narrow namespace for visibility and explicit management

# of ephemeral on-demand cloud resources — burst compute VMs,

# Mac build instances for iOS, and other short-lived resources

# that Applad provisions and tears down automatically.

#

# This namespace is NOT for provisioning. Provisioning always

# happens through applad up (which reads project.yaml) or

# applad instruct (which updates project.yaml then runs

# applad up). This namespace is only for checking what's

# currently running and cleaning up if something goes wrong.

# ============================================================

# Lists all cloud resources currently provisioned across all

# providers and all projects on this instance. Shows provider

# (AWS, GCP, etc.), resource type, region, hourly cost, how

# long it's been running, and current status. A Mac build

# instance triggered by applad deploy run ios-production will

# appear here while the build is in progress, then disappear

# once it tears down automatically.

applad cloud list

# Shows the total and per-resource cloud costs for the current

# billing period, pulled from the runtime database where all

# cloud resource lifecycle events are logged. Includes burst

# compute, managed databases, storage, and any other cloud

# resources provisioned through Applad.

applad cloud cost

# Same as above but for a specific past month. Useful for

# reviewing costs, reconciling invoices, or understanding

# spending trends over time.

applad cloud cost --month 2026-02

# Explicitly tears down a specific cloud resource by its ID.

# Use this when a resource failed to tear down automatically

# (e.g. a build hung, a burst VM timed out, teardown_after

# failed for any reason). The resource ID comes from

# applad cloud list. This is a safety valve — in normal

# operation, Applad tears down ephemeral resources automatically.

applad cloud tear-down <resource-id>

# ============================================================

# ORGANIZATIONS

# Manages organizations on this Applad instance. Organizations

# are the top level of the hierarchy — every project, every

# developer, every piece of infrastructure belongs to an org.

# One instance can host many completely isolated orgs.

# ============================================================

# Lists all organizations on this instance with their IDs,

# names, member counts, and project counts.

applad orgs list

# Creates a new organization. Scaffolds the org directory at

# orgs/<org-name>/org.yaml with default roles, an empty

# ssh_keys list, and a generated .env.example.

applad orgs create --name "Acme"

# Permanently deletes an organization and all its projects,

# data, and infrastructure configuration. Irreversible.

# Will prompt for confirmation.

applad orgs delete <org-id>

# Sets the active organization context for all subsequent

# commands. Most commands that accept --org default to

# whichever org you've switched to here.

applad orgs switch <org-id>

# Lists all members of an org — their identities, roles,

# and registered SSH key labels. Reads from the runtime

# database where member records are stored.

applad orgs members list <org-id>

# Sends an invitation to the given email address to join

# the org with the specified role. The invited developer

# will need to register their SSH public key when they

# accept the invitation.

applad orgs members invite <org-id> \
 --email user@example.com \
 --role developer

# Removes a member from an org. Does not delete their data

# or audit trail entries — those are preserved. If they have

# an SSH key registered, it is automatically revoked.

applad orgs members remove <org-id> <user-id>

# Changes a member's role within an org. Role changes take

# effect immediately and are recorded in the audit trail.

applad orgs members role <org-id> <user-id> --role admin

# ── SSH KEY MANAGEMENT ────────────────────────────────────────

# SSH keys are the identity system in Applad. Every developer

# registers their public key. Every CLI command, every UI

# session, and every applad instruct action is attributed to

# a key fingerprint in the audit trail. Private keys never

# leave the developer's machine — Applad only stores and uses

# public keys.

# Lists all SSH keys registered to an org — their labels,

# fingerprints, associated identities, roles, and permission

# scopes.

applad orgs keys list <org-id>

# Registers a new SSH public key for an org member. The key

# file should be the .pub file — the public half of an SSH

# keypair. Applad reads the public key, computes its

# fingerprint, and adds it to org.yaml under ssh_keys.

# The developer's private key never leaves their machine.

applad orgs keys add <org-id> \
 --label "alice@macbook-pro" \
 --key "~/.ssh/id_ed25519.pub"

# Revokes a registered SSH key by its fingerprint. Any

# in-progress operations using this key are rejected

# immediately. The key's historical audit trail entries

# are preserved — revocation does not erase history.

# Use this when a developer leaves or a key is compromised.

applad orgs keys revoke <org-id> \
 --fingerprint "SHA256:abc123..."

# Replaces an existing key with a new one while preserving

# the developer's identity and full audit history. The old

# key is revoked and the new key is linked to the same

# identity, so audit trail entries before and after rotation

# are all traceable to the same person.

applad orgs keys rotate <org-id> \
 --old "SHA256:abc123..." \
 --new "~/.ssh/id_ed25519_new.pub"

# Creates a scoped deployment key for use in CI/CD pipelines

# (GitHub Actions, GitLab CI, etc.). Unlike developer keys

# which have broad access, deployment keys have explicitly

# limited permissions defined by --scopes. They appear

# distinctly in the audit trail as automated actions rather

# than human actions, making it easy to distinguish what

# a person did from what a pipeline did.

applad orgs keys create-deployment <org-id> \
 --label "ci-github-actions" \
 --scopes "deployments:run,functions:deploy"

# ============================================================

# PROJECTS

# Projects live inside organizations and own all resources —

# tables, functions, deployments, messaging, flags, etc.

# Every resource belongs to exactly one project.

# ============================================================

# Lists all projects across the entire instance.

applad projects list

# Lists all projects belonging to a specific org.

applad projects list --org <org-id>

# Creates a new project inside an org. Scaffolds the project

# directory at orgs/<org>/projects/<name>/ with a project.yaml,

# empty subdirectories for each resource type, and a generated

# .env.example.

applad projects create \
 --name "Mobile App" \
 --org "acme-corp"

# Permanently deletes a project and all its config files.

# Does not automatically delete runtime data — use

# applad db reset (dev only) or coordinate a manual data

# cleanup. Will prompt for confirmation.

applad projects delete <project-id>

# Sets the active project context for all subsequent commands.

# Most commands that are project-scoped default to whichever

# project you've switched to here, saving you from having to

# pass --project on every command.

applad projects switch <project-id>

# Shows details about the currently active project — its ID,

# org, configured environments, infrastructure targets, and

# which features are enabled.

applad projects info

# Creates a new project by copying the config structure of an

# existing project. Copies all yaml files (tables, functions,

# deployments, flags, etc.) but does not copy runtime data,

# secrets, or infrastructure state. Useful for creating a

# staging environment that mirrors production's structure.

applad projects clone <project-id> \
 --name "Mobile App Staging"

# ============================================================

# DATABASE & MIGRATIONS

# Manages database connections and schema migrations.

# Migrations are SQL files in database/migrations/ that are

# applied in order to evolve your schema over time. Applad

# tracks which migrations have been applied and which are

# pending.

# ============================================================

# Applies all pending migrations to the active project's

# primary database in the development environment. Migrations

# are applied in filename order (001*, 002*, etc.) and each

# is wrapped in a transaction — if one fails, it rolls back

# and stops. Records the migration in the audit trail with

# the initiating developer's SSH key identity.

applad db migrate

# Runs migrations for a specific project rather than the

# currently active one.

applad db migrate --project <id>

# Runs migrations against a specific environment's database.

# Use this deliberately for staging or production — never

# run migrations against production without reviewing the

# migration content first.

applad db migrate --env production

# Shows which migrations would be applied without actually

# applying them. Prints the filename and first few lines of

# each pending migration so you can review before committing.

applad db migrate --dry-run

# Rolls back the most recently applied migration by running

# its corresponding down migration if one exists. Use this

# to undo a migration that caused problems.

applad db rollback

# Rolls back the last N migrations in reverse order. Useful

# for unwinding a series of related migrations together.

applad db rollback --steps 3

# Shows the full migration history — which migrations have

# been applied, when, by whom (SSH key identity), and which

# are still pending. Reads from the runtime database where

# migration history is stored.

applad db status

# Generates a new empty migration file with the given name,

# prefixed with the next sequential number

# (e.g. 003_add_avatar_to_users.sql). Opens it ready for

# you to write the SQL. Also generates a corresponding down

# migration file for rollback.

applad db generate "add_avatar_to_users"

# Runs seed files to populate the database with initial or

# test data. Seeds are not migrations — they're idempotent

# data insertion scripts intended for development and staging.

applad db seed

# Drops and recreates the entire database, then re-runs all

# migrations from scratch. Only works in the development

# environment — blocked in staging and production. Use this

# when you want a completely clean slate during development.

applad db reset

# Opens an interactive SQL shell connected to the active

# project's primary database. Lets you run ad-hoc queries,

# inspect data, and debug schema issues directly.

applad db shell

# Opens a shell connected to a specific named database

# connection from database.yaml rather than the primary.

# Useful for inspecting the cache or analytics database.

applad db shell --connection cache

# Exports the database contents to a SQL dump file. Useful

# for backups, sharing a snapshot with a colleague, or

# migrating data to a new database.

applad db export --format sql

# Imports a SQL dump file into the active project's database.

# Use with care — this will overwrite existing data for any

# tables included in the dump.

applad db import ./dump.sql

# ============================================================

# TABLES

# Manages table definitions. Each table lives in its own

# yaml file under tables/. "tables" is the universal term

# regardless of whether the underlying adapter is relational

# (PostgreSQL, MySQL) or document-based (MongoDB). Permission

# rules live alongside the schema in the same file.

# ============================================================

# Lists all tables defined in the active project's tables/

# directory, along with their field counts and whether

# they have permissions defined.

applad tables list

# Generates a new empty table definition file at

# tables/<name>.yaml with placeholder fields, indexes, and

# permission rules. Edit the generated file to define your

# actual schema. Running applad db migrate will apply it.

applad tables generate <name>

# Validates all table definition files for correctness —

# checks field types, relation references (does the referenced

# table exist?), permission rule syntax, and index definitions.

# Does not touch the database.

applad tables validate

# Shows the full schema and permission rules for a specific

# table as currently defined in the yaml file.

applad tables show <name>

# Shows the difference between a table's yaml definition and

# what actually exists in the database. Useful for catching

# drift — cases where the database schema has diverged from

# what the config files describe, usually because a migration

# was applied manually or a file was edited without running

# migrations.

applad tables diff <name>

# ============================================================

# AUTH

# Manages authentication configuration and user/session data.

# Auth provider config lives in auth/auth.yaml. User records,

# sessions, and auth events live in the runtime database.

# ============================================================

# Lists all configured auth providers for the active project —

# email, OAuth providers (Google, GitHub), SAML, phone, etc. —

# along with whether each is enabled and any configuration

# issues detected.

applad auth providers list

# Lists all user records in the active project's runtime

# database. Supports filtering and pagination. Shows user ID,

# email, role, creation date, and last sign-in.

applad auth users list

# Shows the full record for a specific user — their profile

# fields, role, MFA status, linked OAuth providers, and

# recent auth events.

applad auth users get <user-id>

# Soft-deletes a user record. The user can no longer sign in

# but their data is preserved. Their active sessions are

# immediately revoked. Recorded in the audit trail.

applad auth users delete <user-id>

# Bans a user, immediately revoking all their sessions and

# preventing any future sign-in attempts. Differs from delete

# in that the ban is explicit and reversible through the

# admin UI.

applad auth users ban <user-id>

# Permanently and irreversibly deletes all data associated

# with a user across all tables in the project. Implements

# the GDPR and CCPA right to erasure. Coordinates deletion

# across the primary database, storage files owned by the

# user, messaging history, analytics events, and auth records.

# Generates a deletion report recorded in the audit trail.

# Cannot be undone.

applad auth users purge <user-id>

# Lists all active sessions across the project — who is

# signed in, from which device/IP, when the session started,

# and when it expires.

applad auth sessions list

# Immediately invalidates a specific session by its ID.

# The user will be signed out on their next request.

applad auth sessions revoke <session-id>

# Immediately invalidates all active sessions for a specific

# user across all devices. Use this if you suspect a user's

# account has been compromised, or when offboarding a user.

applad auth sessions revoke --user <user-id>

# Generates a full data export for a specific user containing

# all their personal data across all tables, storage files,

# messaging history, and auth records. Implements the GDPR

# and CCPA right of data access / subject access request.

# Output is a structured zip archive.

applad auth export --user <user-id>

# ============================================================

# STORAGE

# Manages files in the configured storage adapter (local

# filesystem, S3, R2, GCS, etc.). The adapter is transparent

# — these commands work identically regardless of which

# storage backend is configured.

# ============================================================

# Lists all storage buckets defined in the active project's

# storage/ directory, along with their visibility (public

# or private), size limits, and allowed file types.

applad storage list

# Lists all files in a specific bucket. Shows filename, size,

# content type, upload date, and owner.

applad storage ls <bucket>

# Uploads a local file to a bucket. Applies the bucket's

# configured virus scan, file type validation, and size limit

# before accepting the file.

applad storage upload <bucket> <file>

# Downloads a file from a bucket to the current local

# directory. Works with both public and private buckets —

# for private buckets, your SSH key identity is used to

# authorize the download.

applad storage download <bucket> <file>

# Permanently deletes a file from a bucket. Cannot be undone.

applad storage delete <bucket> <file>

# Moves or renames a file within or between buckets. The

# source file is removed after the copy is confirmed.

applad storage move <bucket> <file> <destination>

# Runs a virus scan on all files currently stored in a bucket.

# Reports any threats found. By default Applad scans files at

# upload time — use this to scan existing files retrospectively

# or to audit a bucket's contents.

applad storage scan <bucket>

# ============================================================

# FUNCTIONS

# Manages serverless functions. Each function is defined by a

# single yaml file in functions/. The yaml file points to the

# function code via a source block — local path, GitHub repo,

# or container registry. Applad fetches from source at deploy

# time. Functions run in isolated Docker containers.

# ============================================================

# Lists all functions defined in the active project's

# functions/ directory, along with their runtime, source

# type, trigger type, and deployment status.

applad functions list

# Deploys a specific function. Fetches the code from the

# source defined in the function's yaml file (clones the

# repo, copies the local path, or pulls the registry image),

# builds the container, scans it for vulnerabilities, and

# deploys it to the configured infrastructure. The previous

# version keeps running until the new one is healthy.

applad functions deploy <name>

# Deploys all functions defined in the active project's

# functions/ directory in parallel. Useful after a large

# change or when setting up a project for the first time.

applad functions deploy --all

# Tails the execution logs for a specific function — every

# invocation, its duration, its output, and any errors.

# Streams new logs in real time as the function executes.

applad functions logs <name>

# Invokes a function immediately, outside of its normal

# trigger (HTTP, event, schedule). Useful for testing a

# function manually without waiting for its trigger condition.

# Output and any errors are printed to the terminal.

applad functions invoke <name>

# Same as above but passes a JSON payload as the function's

# input. The function receives this as its event/request body.

applad functions invoke <name> --data '{"key":"value"}'

# Fetches the function source and builds the container image

# without deploying it. Useful for catching build errors

# before a real deployment, or for pre-building images in

# a CI step that precedes the actual deploy.

applad functions build <name>

# Scans the built container image for a function against

# known vulnerability databases. Reports findings by severity.

# This runs automatically as part of applad functions deploy

# — use this to scan a built image without deploying it.

applad functions scan <name>

# Removes a function from the active project. Stops the

# running container, removes it from the infrastructure, and

# deletes the function's yaml file. Does not delete the

# function's source code — only the Applad config and

# running container.

applad functions delete <name>

# ============================================================

# WORKFLOWS

# Manages automation workflows — multi-step sequences of

# messages, function invocations, delays, and conditions

# triggered by events or schedules. Each workflow is defined

# by a single yaml file in workflows/.

# ============================================================

# Lists all workflows defined in the active project's

# workflows/ directory, along with their trigger type and

# whether they are currently active.

applad workflows list

# Manually triggers a workflow immediately, outside of its

# normal trigger condition. Useful for testing a workflow

# end-to-end without waiting for the triggering event.

applad workflows trigger <name>

# Shows the execution history for a workflow — every run,

# its steps, which steps succeeded, which failed, and the

# output of each step. Useful for debugging a workflow that

# isn't behaving as expected.

applad workflows logs <name>

# Pauses a workflow so it stops accepting new trigger events.

# Any currently running executions complete normally. New

# trigger events are dropped while paused. Use this to

# temporarily disable a workflow without deleting it.

applad workflows pause <name>

# Resumes a paused workflow. It starts accepting new trigger

# events again immediately.

applad workflows resume <name>

# ============================================================

# MESSAGING

# Manages unified messaging across all channels — email, SMS,

# push notifications, in-app, Slack, Discord, Teams. Provider

# config lives in messaging/messaging.yaml. Template content

# lives in the admin database (editable through the UI without

# a deployment). Template references live in

# messaging/templates/\*.yaml.

# ============================================================

# Lists all configured messaging channels for the active

# project — which channels are enabled, which providers are

# configured for each, and whether each channel is healthy.

applad messaging channels list

# Sends a test message through the email channel to the

# specified address using the named template. Useful for

# verifying that your email provider credentials are correct

# and that a template renders as expected before it goes

# to real users.

applad messaging test email \
 --to user@example.com \
 --template welcome

# Sends a test SMS to the specified phone number using the

# named template. Number must include country code.

applad messaging test sms \
 --to +1234567890 \
 --template password-reset

# Sends a test push notification to a specific device

# identified by its push token. Useful for testing FCM or

# APNS configuration and verifying token registration.

applad messaging test push \
 --token <device-token> \
 --template welcome

# Sends a test message to the configured Slack integration

# using the named template. Useful for verifying webhook

# configuration and template formatting.

applad messaging test slack \
 --template new-user-alert

# Shows the delivery log for all messaging channels — every

# message sent, its channel, template, recipient, timestamp,

# and delivery status (delivered, bounced, failed, etc.).

applad messaging logs

# Same as above but filtered to a specific channel.

# Useful for debugging a specific channel in isolation.

applad messaging logs --channel email

# Same as above but filtered to messages sent using a

# specific template. Useful for checking whether a particular

# message type (e.g. welcome emails) is being delivered.

applad messaging logs --template welcome

# Lists all messaging template references defined in the

# active project's messaging/templates/ directory. Shows

# each template's key, which channels it supports, and

# its default subject/title/body values.

applad messaging templates list

# Validates that all template references in the yaml files

# have corresponding content entries in the admin database.

# Catches cases where a template was defined in yaml but

# its content was never created in the admin UI.

applad messaging templates validate

# ============================================================

# FEATURE FLAGS

# Manages feature flags. Flag skeletons — that a flag exists,

# its variants, its per-environment defaults — live in

# flags/\*.yaml and are version controlled. Targeting rules

# (which users see which variant) live in the admin database

# and are managed through the UI without a deployment.

# This is how deploy and release are kept separate: deploy

# puts code on the server, flags release it to users.

# ============================================================

# Lists all feature flags defined in the active project's

# flags/ directory, along with their type (boolean or

# multivariate), their per-environment defaults, and

# whether targeting rules have been configured in the

# admin database.

applad flags list

# Generates a new empty flag skeleton file at

# flags/<key>.yaml. Edit it to set the flag type, variants,

# and per-environment defaults. Targeting rules are then

# configured through the admin UI.

applad flags generate <key>

# Validates all flag definition files — checks syntax, that

# multivariate flags have at least two variants, and that

# the default value is one of the defined variants.

applad flags validate

# Shows the full definition for a specific flag — its type,

# variants, per-environment defaults, and the targeting rules

# currently configured in the admin database.

applad flags show <key>

# Enables a flag for a specific environment by setting its

# default to true (for boolean flags) or to the first

# non-control variant (for multivariate flags). This updates

# the admin database — no deployment required. Product

# managers can do this through the admin UI as well.

applad flags enable <key> --env production

# Disables a flag for a specific environment, reverting it

# to its default-off state. Takes effect immediately without

# a deployment.

applad flags disable <key> --env production

# Shows the evaluation history for a flag — every time it

# was evaluated, which variant was returned, and for which

# user. Useful for verifying that a flag is being evaluated

# correctly and that targeting rules are working as expected.

applad flags logs <key>

# ============================================================

# DEPLOYMENTS

# Manages all deployment pipelines — web sites to domains,

# Android apps to the Play Store, iOS apps to the App Store,

# desktop app distribution, and OTA updates to existing

# installs. All are defined as yaml files in deployments/

# with a type field and a source block pointing to the code.

#

# IMPORTANT: Deploy and release are separate concerns.

# applad deploy puts an artifact somewhere (technical, often

# invisible to users). Feature flags release functionality

# to users (a business decision). Never conflate the two.

# ============================================================

# Lists all deployment pipelines defined in the active

# project's deployments/ directory across all types.

applad deploy list

# Lists only web deployment pipelines (type: web).

applad deploy list --type web

# Lists only Android / Play Store pipelines (type: play-store).

applad deploy list --type play-store

# Lists only iOS / App Store pipelines (type: app-store).

applad deploy list --type app-store

# Lists only OTA update pipelines (type: ota).

applad deploy list --type ota

# Triggers a deployment pipeline by name. Fetches the source

# code from the repo or path defined in the deployment's yaml

# source block, runs the configured build command, and deploys

# the artifact to its target. For web: pushes to the domain

# via Caddy. For play-store: submits to the Play Store. For

# app-store: spins up an AWS Mac instance, builds, submits,

# tears down. For ota: pushes the update to the OTA channel.

# Every deployment is attributed to your SSH key identity in

# the audit trail.

applad deploy run <name>

# Deploys the web deployment named "web" to its configured

# domain. Fetches the source, builds, pushes via Caddy.

applad deploy run web

# Triggers the android-production pipeline. Fetches source

# from the configured GitHub repo, builds the AAB on the

# build VPS, signs it with credentials from the admin database,

# and submits to the Play Store on the configured track.

applad deploy run android-production

# Triggers the ios-production pipeline. Spins up an AWS Mac

# instance, SSHes in, fetches source, builds the IPA, signs

# it, submits to the App Store, and tears the Mac instance

# down. The instance appears in applad cloud list while

# the build is in progress.

applad deploy run ios-production

# Triggers an OTA update deployment to the configured channel.

# Starts the gradual rollout at the percentage configured in

# the yaml file.

applad deploy run ota

# Triggers a deployment against a specific environment rather

# than the default one. Useful for deploying to staging

# without affecting production.

applad deploy run <name> --env staging

# Shows the deployment log for a specific pipeline — every

# run, its build output, its submission result, any errors,

# and the duration of each step. Streams recent history and

# can tail live logs for an in-progress deployment.

applad deploy logs <name>

# Shows the current status of the most recent deployment for

# a pipeline — whether it's in progress, succeeded, or failed,

# and at which step it currently is or stopped.

applad deploy status <name>

# Rolls back a deployment to the previous successful version.

# For web: reverts the Caddy config to serve the previous

# build. For play-store/app-store: triggers a rollout of the

# previously submitted build on the respective store. For ota:

# see applad deploy ota rollback.

applad deploy rollback <name>

# Opens the deployed artifact in a browser or store page.

# For web deployments: opens the site URL. For play-store:

# opens the Play Store listing. For app-store: opens the

# App Store listing.

applad deploy open <name>

# ── DOMAIN MANAGEMENT (WEB DEPLOYMENTS) ───────────────────────

# Lists all custom domains configured across all web

# deployment pipelines, along with their verification status

# and SSL certificate status.

applad deploy domains list

# Verifies that a domain's DNS is correctly pointed at the

# Applad instance. Checks that the A record or CNAME points

# to the right IP/hostname. Caddy will not serve a domain

# until DNS is verified.

applad deploy domains verify <domain>

# Adds a custom domain to an existing web deployment pipeline.

# Updates the deployment's yaml file and triggers Caddy to

# request an SSL certificate for the new domain.

applad deploy domains add <name> --domain "newdomain.com"

# ── OTA ROLLOUT MANAGEMENT ────────────────────────────────────

# OTA updates roll out gradually to existing installs. These

# commands let you manage a rollout that's in progress —

# checking adoption, pausing if something is wrong, resuming

# when ready, or rolling back if the update causes problems.

# Shows the current state of an OTA rollout — what percentage

# of devices have received the update, how many are still on

# the previous version, and the adoption rate over time.

applad deploy ota status <name>

# Pauses an in-progress gradual rollout. Devices that have

# already received the update keep it. New devices stop

# receiving it until the rollout is resumed. Use this if you

# notice a problem after a rollout has started.

applad deploy ota pause <name>

# Resumes a paused OTA rollout. Picks up where it left off —

# continuing to roll out to the next increment of devices.

applad deploy ota resume <name>

# Forces all devices on the new OTA version back to the

# previous version. Use this if the OTA update introduced

# a critical bug and needs to be fully reversed.

applad deploy ota rollback <name>

# ── PREVIEW ENVIRONMENTS (WEB DEPLOYMENTS) ────────────────────

# Preview environments are temporary deployments automatically

# created for each pull request against the source branch.

# They let you test and share changes before merging.

# Lists all currently active preview environments for a web

# deployment pipeline — which PR each belongs to, its URL,

# and when it was created.

applad deploy preview list <name>

# Opens the preview environment for a specific pull request

# number in a browser. The URL follows the pattern configured

# in the deployment yaml (e.g. pr-42.preview.myapp.com).

applad deploy preview open <name> --pr 42

# ============================================================

# ANALYTICS

# Manages the built-in analytics system. All events are

# stored in the project's analytics database under your

# control. No data is sent to third parties by default.

# ============================================================

# Shows the current state of the analytics system — whether

# it's running, how much data has been collected, storage

# used, and which event types are being captured.

applad analytics status

# Lists recent analytics events captured by the system.

# Supports filtering by event type, user, time range, etc.

applad analytics events list

# Exports analytics events to a file for external analysis.

# Supports csv and json formats. --from and --to define the

# time range of events to include.

applad analytics export \
 --format csv \
 --from 2026-01-01 \
 --to 2026-02-01

# Permanently deletes all analytics events older than the

# given date. Use this to enforce data retention policies

# or to free up storage space.

applad analytics purge --before 2025-01-01

# Shows a breakdown of cloud resource costs for the active

# project — which resources were provisioned, for how long,

# and what they cost. Same data as applad cloud cost but

# scoped to the active project and grouped by resource type.

applad analytics cost

# ============================================================

# REALTIME

# Manages the realtime pub/sub layer powered by NATS or Redis.

# Realtime channels let clients subscribe to database changes

# and receive updates as they happen.

# ============================================================

# Lists all realtime channels defined in realtime/realtime.yaml

# along with which table each is subscribed to, which events

# it broadcasts, and its permission rules.

applad realtime channels list

# Shows whether the realtime adapter (NATS or Redis) is

# connected and healthy, and how many active subscriptions

# are currently open.

applad realtime status

# Publishes a test event to a channel and confirms that it

# is received by the broker. Useful for verifying that the

# realtime adapter is correctly configured and that a channel

# is set up properly.

applad realtime test <channel>

# ============================================================

# SECURITY

# Security tooling and visibility. Security policies live in

# config files alongside the resources they protect and are

# reviewed in the same PRs. Runtime security events live in

# a dedicated security event log in the runtime database,

# separate from the general application log.

# ============================================================

# Runs a full security audit of the active project. Checks

# table permission rules for common mistakes (e.g. missing

# auth filters), auth configuration for weak settings,

# function containers for vulnerabilities, hosting headers

# for missing security directives, and secrets for any that

# appear to be hardcoded in config rather than injected

# via ${VAR} references.

applad security audit

# Scans all function container images for the active project

# against known vulnerability databases. Reports findings

# by severity. Runs automatically as part of function

# deployment — use this to scan outside of a deployment.

applad security scan

# Lists recent security events from the dedicated security

# event log — failed auth attempts, rate limit hits, blocked

# IPs, permission denials, anomalous access patterns, and

# applad instruct actions. Separate from general application

# logs to make security monitoring focused and actionable.

applad security events list

# Same as above but filtered to a specific event type.

# Common types: failed_auth, rate_limit_hit, blocked_ip,

# permission_denied, anomalous_access, instruct_action.

applad security events list --type failed_auth

# Same as above but filtered to events after a specific date.

# Useful for reviewing events following an incident or

# suspicious activity report.

applad security events list --since "2026-02-01"

# Lists all SSH keys registered across all orgs on this

# instance. Shows key labels, fingerprints, identities,

# roles, and scopes. Useful for a periodic access review.

applad security keys list

# Audits all registered SSH keys and flags any that haven't

# been rotated within the period configured in applad.yaml

# (key_rotation_reminder_days). A practical prompt for

# regular key hygiene without being prescriptive.

applad security keys audit

# Lists all secret keys currently set for the active project.

# Keys only — values are never shown in any output. Use this

# to audit which secrets are configured without exposing them.

applad security secrets list

# Interactively sets a secret value. Prompts you to enter the

# value without echoing it to the terminal, so it never

# appears in shell history or logs. Stores it encrypted in

# the admin database.

applad security secrets set <key>

# Rotates a secret — sets a new value, updates any references

# to it, and records the rotation event in the audit trail.

# The old value is immediately invalidated.

applad security secrets rotate <key>

# Permanently deletes a secret. Any config that references

# it via ${VAR} will fail to validate until a new value is

# set or the reference is removed.

applad security secrets delete <key>

# ============================================================

# OBSERVABILITY

# Logs, traces, and alerts across all modules. Structured

# logs and distributed traces are stored in the runtime

# database and can be exported to external systems via the

# config in observability/observability.yaml.

# ============================================================

# Tails all logs from the running Applad instance across

# all modules — API requests, function executions, database

# queries, messaging sends, deployment events, auth events,

# and more. Streams new log lines in real time.

applad logs

# Same as above but scoped to a specific project. Useful

# when one instance runs multiple projects and you only

# want to see logs for one of them.

applad logs --project <id>

# Same as above but scoped to a specific environment.

applad logs --env production

# Filters logs to a specific module. Common values: api,

# functions, db, storage, messaging, deployments, auth,

# realtime, workflows. Useful for focusing on one subsystem

# when debugging.

applad logs --module functions
applad logs --module deployments

# Filters logs to a specific severity level. Shows only

# logs at that level and above — error shows errors and

# above, warn shows warnings and errors, etc.

applad logs --level error

# Lists recent distributed traces. A trace represents a

# single request or operation as it flows through multiple

# services or functions. Useful for understanding latency

# and finding bottlenecks in complex operations.

applad traces list

# Shows the full detail of a specific trace — every span,

# its duration, its parent, and any attributes or errors

# attached to it. The trace ID comes from applad traces list

# or from a log line that includes a trace_id field.

applad traces show <trace-id>

# Lists all configured alerts from observability/observability.yaml

# — their names, the metric they watch, their thresholds,

# and which notification channels they use.

applad alerts list

# Shows the current firing status of all alerts — which are

# currently triggered, which have recently recovered, and

# which have never fired.

applad alerts status

# ============================================================

# INSTRUCT

# The CLI surface of the lad — Applad's AI-powered assistant.

# You give instructions in plain language and the lad

# translates them into config changes, migrations, and

# applad up operations. Every instruction is attributed to

# your SSH key identity in the audit trail. The exact prompt

# is recorded alongside every change made — AI-assisted

# changes are always traceable to the human who authorized

# them.

#

# --dry-run is always available and always safe. It shows

# exactly what the lad would do — which files it would create

# or edit, which migrations it would generate, which applad

# up operations it would trigger — without doing any of it.

# Use --dry-run liberally, especially for infrastructure

# changes.

# ============================================================

# ── SCAFFOLD AND CONFIGURE ────────────────────────────────────

# Creates a new table definition file at tables/users.yaml

# with fields for email, name, avatar_url, and soft delete

# support (deleted_at). Also generates the corresponding

# migration SQL and updates .env.example if needed.

applad instruct "create a users table with email, name, avatar, and soft delete"

# Adds a fulltext index to the posts table's title and body

# fields. Edits tables/posts.yaml to add the index definition

# and generates a migration that creates the fulltext index

# in the database.

applad instruct "add fulltext search to posts"

# Creates a new function definition file at

# functions/send-welcome-message.yaml with a source block

# pointing to a local path, a trigger on auth.user.created,

# and appropriate container security settings. Also scaffolds

# the function source file at the configured path.

applad instruct "create a function that sends a welcome message on user signup"

# Creates a deployment pipeline definition at

# deployments/android-production.yaml with type: play-store,

# a source block pointing to the project's configured GitHub

# repo, and a build config for Flutter. Prompts for signing

# credential setup if not already configured.

applad instruct "set up a Play Store deployment pipeline for my Flutter app"

# Creates a web deployment definition at deployments/web.yaml

# with type: web, the configured domain, SSL, sensible security

# headers, and a source block pointing to the web app repo.

applad instruct "set up a web deployment for myapp.com"

# Adds a rate limiting rule to the payments endpoint in

# observability/observability.yaml — sets a per-user limit

# appropriate for payment operations.

applad instruct "add rate limiting to the payments endpoint"

# Creates a workflow definition at

# workflows/post-published.yaml that triggers on a post

# status change to published, sends a push notification

# via the push channel and an email via the email channel.

applad instruct "create a workflow that sends push and email when a post is published"

# Creates three messaging template reference files:

# messaging/templates/order-confirmation-email.yaml,

# messaging/templates/order-confirmation-sms.yaml,

# messaging/templates/order-confirmation-push.yaml —

# and prompts you to add the template content through

# the admin UI.

applad instruct "add a messaging template for order confirmation across email, sms and push"

# ── INFRASTRUCTURE ────────────────────────────────────────────

# Adds an AWS RDS Postgres instance as a cloud adapter to

# the production environment in project.yaml. Updates

# database/database.yaml with a production environment

# override pointing to the RDS connection. Updates

# .env.example with the new required credentials. Then

# runs applad up --env production --dry-run so you can

# review before applying.

applad instruct "provision a Postgres instance on AWS RDS for production"

# Creates or updates the staging environment in project.yaml

# to target a Hetzner VPS, mirroring the infrastructure

# configuration of the production environment. Runs

# applad up --env staging --dry-run for review.

applad instruct "set up staging to mirror production on a Hetzner VPS"

# Updates storage/storage.yaml to use the S3 adapter,

# adds the required ${VAR} references, updates .env.example

# with S3_BUCKET, S3_REGION, S3_ACCESS_KEY, S3_SECRET_KEY,

# and runs applad up --dry-run so you can review the change.

applad instruct "migrate our storage adapter from local to S3"

# Adds a production environment override to

# messaging/messaging.yaml switching the email provider

# from the current one to SES, adds the required credential

# references, and updates .env.example.

applad instruct "switch email provider to SES for production"

# Provisions an on-demand cloud VM, SSHes into it, runs

# the specified processing job in a Docker container, then

# tears the VM down when the job completes. The VM appears

# in applad cloud list while running. Cost and duration are

# logged to the runtime database.

applad instruct "spin up a cloud VM for this data processing job and tear it down when done"

# Reads the current list of cloud resources from

# applad cloud list, cross-references their usage patterns

# from the analytics database, and suggests which resources

# appear idle or underutilised and could be torn down.

applad instruct "what cloud resources can we safely tear down?"

# ── DEBUG AND DIAGNOSE ────────────────────────────────────────

# Reads the current API error rate from analytics, finds the

# requests contributing most to the error rate, cross-

# references function logs and database query traces, and

# gives a diagnosis with suggested fixes.

applad instruct "why is my API error rate high?"

# Reads the deployment log for the most recent web deployment,

# identifies the failing step, and explains what went wrong

# and how to fix it.

applad instruct "what failed in the last web deployment?"

# Reads the Play Store submission log for the most recent

# iOS build, identifies the rejection reason from the store

# API response, and explains what needs to change.

applad instruct "why was the Play Store submission rejected?"

# Reads the slow query log from the analytics database,

# identifies the query, cross-references the table definition

# in tables/\*.yaml, and suggests index changes that would

# improve performance. Generates the index definition and

# migration if you confirm.

applad instruct "why is this query slow?"

# Reviews the permission rules for a specific table or set

# of tables in tables/\*.yaml and identifies any rules that

# could allow unintended data access — missing auth filters,

# overly broad role permissions, or missing row-level filters.

applad instruct "is this permission rule safe?"

# Reads the security event log for the past 24 hours and

# surfaces any patterns that look anomalous — unusual volumes

# of failed auth, access from unexpected locations,

# permission denial spikes, or unusual instruct activity.

applad instruct "are there any security anomalies in the last 24 hours?"

# Reads cloud resource lifecycle logs from the runtime

# database and produces a cost breakdown by resource type,

# project, and environment for the current month.

applad instruct "how much are we spending on cloud resources this month?"

# ── CONTEXT FLAGS ─────────────────────────────────────────────

# --context narrows the lad's focus to a specific data source.

# Without --context the lad draws on all available sources.

# With --context it focuses specifically on the named source,

# giving faster and more precise answers for targeted questions.

# Focuses the lad on recent log output. Useful for diagnosing

# a specific failure that just happened.

applad instruct --context logs "what failed in the last hour?"

# Focuses the lad on the tables/ directory. Useful for

# performance questions that relate specifically to schema

# and indexing decisions.

applad instruct --context tables "suggest indexes for better query performance"

# Focuses the lad on the security event log. Useful for

# targeted security investigations.

applad instruct --context security "are there any anomalous access patterns?"

# Focuses the lad on currently provisioned cloud resources.

# Useful for cost and utilisation questions.

applad instruct --context cloud "which resources are idle?"

# Focuses the lad on deployment logs and history. Useful for

# debugging a specific deployment failure.

applad instruct --context deployments "why did the iOS build fail?"

# Focuses the lad on function execution logs and error rates.

applad instruct --context functions "which functions have the highest error rate?"

# Focuses the lad on the audit trail of previous instruct

# actions. Useful for reviewing what the lad (and who)

# changed recently, or for understanding the history of

# a specific piece of config.

applad instruct --context instruct-history "what did alice change yesterday?"

# ── DRY RUN ───────────────────────────────────────────────────

# --dry-run shows the complete plan the lad would execute

# without making any changes. Shows every file that would be

# created or modified, every migration that would be generated,

# and every applad up operation that would run. Use this to

# review and understand what an instruction would do before

# committing to it. Especially important for infrastructure

# changes and anything touching production.

applad instruct --dry-run "add fulltext search to posts"
applad instruct --dry-run "provision RDS for production"
applad instruct --dry-run "set up staging to mirror production"
applad instruct --dry-run "create a function that processes payments"

# ── SCOPING ───────────────────────────────────────────────────

# By default instruct operates in the context of the active

# project and environment (set by applad projects switch and

# the current environment). Use these flags to override.

# Runs the instruction in the context of a specific project

# regardless of which project is currently active.

applad instruct --project mobile-app "why is staging slow?"

# Runs the instruction in the context of a specific

# environment. Useful for asking environment-specific

# questions without switching your active environment.

applad instruct --env production "show me anomalous access patterns"

# ============================================================

# AUDIT TRAIL

# Every change in Applad is recorded in the audit trail —

# config changes, SSH operations, cloud API calls, deployments,

# migrations, flag toggles, and instruct actions. Every entry

# is cryptographically signed with the SSH key of the person

# or system that initiated it. The audit trail cannot be

# modified or deleted — it is append-only and retained for

# the period configured in applad.yaml (default: 7 years).

# ============================================================

# Lists recent audit trail entries across all orgs and

# projects on this instance. Each entry shows: timestamp,

# actor identity and key fingerprint, action type, target

# (org/project/environment), and a summary of what changed.

applad audit list

# Filters the audit trail to entries initiated by a specific

# developer identity. Useful for reviewing what a particular

# person has changed, or for an access review.

applad audit list --actor "alice@acme-corp"

# Filters to entries where the action was initiated by

# applad instruct — AI-assisted changes. Each entry includes

# the exact prompt that was given to the lad. Useful for

# reviewing and auditing AI-assisted changes specifically.

applad audit list --via instruct

# Filters to entries where the action was initiated by a

# CI/CD pipeline using a scoped deployment key. Useful for

# auditing automated pipeline activity separately from

# human activity.

applad audit list --via ci

# Filters to a specific action type. Common values:

# db.migrate, db.rollback, deployments.run, functions.deploy,

# config.push, flags.enable, secrets.rotate, users.purge.

applad audit list --action db.migrate
applad audit list --action deployments.run
applad audit list --action functions.deploy

# Filters to entries created after a specific date.

applad audit list --since "2026-02-01"

# Shows the full detail of a specific audit trail entry —

# the complete config diff, the full list of files changed,

# the infrastructure operations performed, the instruction

# prompt if it was an instruct action, and the cryptographic

# signature. The entry ID comes from applad audit list.

applad audit show <entry-id>

# Cryptographically verifies that a specific audit trail

# entry has not been tampered with since it was created.

# Checks the entry's signature against the SSH public key

# of the actor recorded in the entry. Returns verified or

# tampered with a clear explanation.

applad audit verify <entry-id>

# ============================================================

# MULTI-INSTANCE / CLUSTER

# For running multiple Applad nodes together as a cluster —

# useful when you need horizontal scaling or high availability.

# The core Applad process is stateless, so clustering is a

# matter of pointing multiple nodes at the same database

# and coordinating them through these commands.

# ============================================================

# Shows the status of the current cluster — how many nodes

# are running, which is the current node, and whether all

# nodes are in sync.

applad cluster status

# Adds the current node to an existing cluster by connecting

# it to the node at the given URL. The new node begins

# receiving traffic immediately after joining.

applad cluster join <node-url>

# Gracefully removes the current node from the cluster.

# In-flight requests complete before the node disconnects.

# Other nodes continue serving traffic without interruption.

applad cluster leave

# Lists all nodes currently in the cluster — their URLs,

# statuses, versions, and when they last checked in.

applad cluster nodes list

# ============================================================

# IMPORT / EXPORT / PORTABILITY

# Applad is designed to be portable. Your config tree, your

# data, and your infrastructure targets are all yours. These

# commands let you move between instances, back up your

# setup, or migrate from another platform.

# ============================================================

# Exports a complete snapshot of the active instance —

# config tree, schema, and data — as a zip archive. Does

# not include secret values, only their key references.

# Use this to back up an instance or migrate it to new

# infrastructure.

applad export

# Same as above but scoped to a single project.

applad export --project <id>

# Exports only the config tree and migrations — no runtime

# data. Useful for sharing the structural definition of a

# project without any user data.

applad export --config-only

# Imports a previously exported zip archive, restoring the

# config tree and data to a running instance. Secret values

# will need to be re-set via applad secrets set after import.

applad import ./applad-export.zip

# Migrates an existing Appwrite project to Applad. Connects

# to your Appwrite instance, reads your collections, auth

# config, and storage config, and generates equivalent

# Applad config files. Data migration runs separately.

applad migrate-from appwrite --project <id>

# Migrates an existing Supabase project to Applad. Reads

# your Supabase schema, RLS policies, auth config, and

# storage buckets and generates equivalent Applad config.

applad migrate-from supabase --project <id>

# Migrates an existing Firebase project to Applad. Reads

# your Firestore schema, Firebase Auth config, and Storage

# config and generates equivalent Applad config files.

applad migrate-from firebase --project <id>

# Migrates an existing PocketBase instance to Applad. Reads

# the PocketBase data directory directly — no running

# PocketBase instance required. Generates Applad table

# definitions from PocketBase collections and migrates data.

applad migrate-from pocketbase --data-dir ./pb_data
