# Platform audit — driven by the Flutter chat app

This app exists to prove Applad has catered for everything a real client needs.
The rule: when a feature needs an API that is missing, we **implement it in core
Applad first, then consume it here** — never work around a gap in the app.

This file is the running ledger of what was missing, what we did about it, and
an honest read on whether each fix has the best achievable **developer
experience (DX)** and **security**. It is updated as the app grows.

Legend: ✅ fixed · 🚧 in progress · 🔎 identified, not yet addressed.

---

## G1 — No cross-platform way to authenticate ✅

**What was missing.** `POST /v1/account/sessions/email` (and the anonymous
variant) returned the session object but delivered the session JWT *only* as an
`HttpOnly` `Set-Cookie`. The API authenticates by JWT (Authorization bearer,
`?token=` for WebSockets, or that cookie). A browser can ride the cookie, but a
mobile app has no cookie jar, and the token was never in the response body, so a
Flutter mobile client had **no path to authenticate at all**. The Dart SDK even
shipped a `setSession()` sending `x-applad-session`, a header the middleware
never reads — dead code implying an auth mode that does not exist.

**Fix.**
- Core: session-creation responses now include `secret` (the JWT), populated
  only on creation (`omitempty`), never when listing/fetching a session. The
  `Set-Cookie` stays for browsers.
- Dart SDK: `createEmailSession` / `createAnonymousSession` apply that secret as
  the bearer token automatically and `deleteSessions` clears it. Persist `secret`
  to stay signed in; restore with `setJWT`.

**DX read.** Good. Login "just works" on every platform with no cookie plumbing,
matching the mental model of Firebase/Appwrite/Supabase mobile SDKs. One fraction
of friction remains: the app must persist `secret` itself (no SDK-managed secure
storage yet) — acceptable, and arguably correct, since token storage is
platform-specific (Keychain/Keystore vs web storage).

**Security read.** Acceptable and standard. Returning a session secret in the
body is the norm for non-browser SDKs; the token is a revocable, expiring
session JWT, not a long-lived credential. Residual risks and the bar we hold:
- On web, prefer the `HttpOnly` cookie over storing the secret in JS-readable
  storage (XSS exfiltration). The SDK sets the bearer for the current process
  but we should document "on web, rely on the cookie; do not persist the secret
  to localStorage." *(Follow-up: guidance + a web-vs-mobile storage note.)*
- `setSession()`/`x-applad-session` is misleading dead code. *(Follow-up:
  remove or make the middleware honour it as a session-secret lookup.)*

---

## G2 — RLS cannot express group access (team/role membership) ✅

**What is missing.** Applad's row RLS supports `any`, `users`, `user:<id>`, and a
*generic role token* (`read("team:X")`, `read("editors")`, …) matched against
`request.jwt.claims -> 'roles'`. The Postgres plumbing is fully built:
`applySessionContext` marshals `roles` into `request.jwt.claims` and the policy
expression checks membership. **But every handler passes `nil` for roles**
(`ListRowsWithAuth(..., userID, nil, ...)`, `ExecuteSQL(..., userID, nil, ...)`),
and nothing derives a user's group memberships. Consequences:
- A row created with `read("team:X")` is readable by **no one** — the role token
  is never present on any request.
- There is also no query for "which teams is this user in" (teams can be listed
  by team id and their members enumerated, but not the reverse), so even a
  resolver has nothing to call.

Group chat depends on exactly this: *members of a channel may read its messages*.
Without it, the only enforceable options are public (`any`/`users`) or
single-owner (`user:<id>`) — neither models a channel.

**Design under consideration (generic, not chat-specific).** Back each channel
with an Applad **Team**; messages carry `read("team:<teamId>")`. On every
authenticated data request, resolve the caller's team memberships and pass them
as `team:<teamId>` roles into the existing pipeline (replacing the `nil`). This
mirrors Appwrite's `Role.team()` and makes group RLS work for *any* app, not
just this one. Requires: (a) a "teams for user" lookup, (b) a shared resolver
the row/SQL handlers call, (c) permission-token vocabulary for `team:<id>`.

**DX read (target).** Strong if the resolver is automatic: an app writes
`permissions: ['read("team:$id")']` and membership is enforced with zero
per-request wiring. The risk to DX is making developers hand-assemble roles;
the resolver must be implicit and server-derived.

**What we did.** Added a `RoleResolver` seam to the databases service
(`SetRoleResolver`), a `resolveRoles` helper that runs when a caller passes no
explicit roles, and a teams-backed implementation `Teams.RolesForUser` that
yields `team:<id>` (and `team:<id>/<role>`) for every team the user has *joined*,
scoped to the project. Wired in `router.go`. All row/SQL handlers keep passing
`nil`, so the authoritative server path always runs; internal callers with a
vetted list pass through untouched.

**DX read.** Strong: an app writes `permissions: ['read("team:$id")']` and
membership enforcement is automatic, no per-request role wiring.

**Security read.** Meets the bar: roles are resolved server-side from the caller's
own identity, never from the request body; the resolver reads only *joined*
memberships (an unaccepted invite grants nothing); a resolver error fails closed
to the built-in roles rather than opening or hard-denying. **Residual — membership
freshness:** roles are resolved live per request (correct; revocation is
immediate; costs one indexed query). We chose this over baking roles into the JWT
precisely so a removed member loses access at once. *(Follow-up: add an index on
`memberships(user_id) WHERE joined` and measure before any caching.)* **Residual —
realtime bypass:** see G5; RLS on REST does not yet cover the WebSocket fan-out.

---

## G3 — Team membership had no lifecycle (no creator, no join) ✅

**What was missing.** Two holes made G2 inert in practice:
- `teams.Create` inserted only the team row. The creator got **no membership**,
  so they held no `team:<id>` role and were locked out of their own team's rows.
- `CreateMembership` wrote `joined=false` with a one-time secret, but **no
  endpoint ever set `joined=true`** — the API had create/list/delete only. So a
  membership could never activate, and every `team:<id>` role was unreachable.
  The invite `secret` was also never returned, so an invite could not be sent
  (self-hosted often has no SMTP).

**Fix.**
- `teams.Create` now enrols the authenticated creator as a joined `owner`
  member. Server-side (API-key) creation with no user still makes an unowned
  team, unchanged.
- New `PATCH /v1/teams/{teamId}/memberships/{membershipId}/status` accepts an
  invite: it binds the *session's* user id, marks the row joined, and clears the
  secret. `CreateMembership` now returns the `secret` (omitempty) so an inviter
  can build a join link.
- Dart client gains `teams.acceptMembership(...)`.

**DX read.** Good and conventional (mirrors Appwrite's team lifecycle): create a
team and you are in it; invite returns a link you can send; the invitee redeems
with just the secret.

**Security read.** Sound. The joining identity is taken from the authenticated
session, never the body, so an invite cannot be redeemed *as someone else*. The
secret is single-use (cleared on accept) and only matches an un-joined row.
**Residual:** the invite secret is a bearer token with no expiry yet.
*(Follow-up: add an `expires_at` to memberships and reject stale invites; today
delete-and-reissue is the only revocation.)*

---

## G4 — Client SDK could not use Teams ✅

**What was missing.** Teams are `Project + Auth`, so a signed-in user can manage
them, but the Dart **client** SDK exposed no Teams service (only the server SDK
did). The app could authenticate and read/write rows but could not create the
teams that back its channels.

**Fix.** Added a client `Teams` service (`client.teams`) covering create / list /
get / update / delete, plus membership create / list / accept / delete, wired
into the `Applad` client and exported.

**DX read.** Good: `client.teams.create(name: ...)` then
`client.teams.createMembership(...)` reads the way the equivalent Firebase/
Appwrite calls do. **Security read.** Neutral — it is a thin transport over
endpoints already authorised server-side; no new surface.

---

## G5 — Realtime fan-out is not RLS-filtered 🔎

**What is missing (identified, not yet fixed).** Row RLS protects REST reads, but
the realtime hub broadcasts table-change events to every subscriber of a
table's channel (`databases.<project>.<db>.<table>`). A user who can subscribe to
the `messages` table could receive change events for rows they are **not**
permitted to read — the group boundary G2/G3 enforce on REST does not yet apply
to the live stream. For a single-workspace demo this is latent, but "cater for
everything" means closing it.

**Design under consideration.** Evaluate each event against the subscriber's
resolved roles before delivery (reuse the G2 resolver + `checkRowPermission` on
the row's `permissions`), or scope subscriptions to a permission the server
verifies at subscribe time. The WebSocket already carries the user via `?token=`,
so the identity is available. **Security is the whole point of this item**;
until it lands, treat realtime as broadcasting within a table to any subscriber.

---

## G6 — Team listing leaked every team in the project ✅

**What was missing.** `GET /v1/teams` returned *all* teams in the project to any
caller. For a chat app that means every user could enumerate the names of every
other user's channels/workspaces — a confidentiality leak, not just untidy.

**Fix.** A plain signed-in user now gets a membership-scoped listing
(`ListForUser`: only teams they have joined); an admin (console JWT) or API key
still gets the unscoped project-wide view for management.

**DX read.** Good and expected — "list teams" returns *my* teams, matching every
comparable SDK. **Security read.** Closes the leak with the right actor model:
scope keys off the server-resolved identity, and only privileged callers see all.

---

## G7 — Per-row (document-level) permissions are accepted but not enforced 🔎 (decision needed)

**What is missing — the big one.** The API takes a `permissions` array on
`createRow`/`updateRow` (e.g. `read("team:X")`), and the docs imply per-document
security. In reality:
- `CreateRowWithAuth` **drops the `permissions` argument** — it builds the INSERT
  from data columns only and stores the permissions nowhere. The physical
  row-security table is created with just `id, created_at, updated_at` plus the
  user's columns; there is **no per-row permissions column**.
- The RLS SELECT policy (`applad_read_access`) is derived from the **table's**
  permissions, not the row's. So list queries are filtered only table-wide.

Consequences for the two obvious chat models:
- *Per-row* `read("team:X")` on a shared `messages` table: the permission is
  discarded, and because a document-security table has no table-level read grant,
  the read policy is skipped and FORCE RLS returns **no rows to anyone**. The
  channel would always look empty.
- The feature that would make a single shared table correct — per-row read
  scoped to a channel's team — does not exist yet.

**What works today instead.** *Table-level* RLS **is** enforced on list, and with
the G2 resolver a table permissioned `read("team:X")` shows its rows to exactly
that team's members. So a channel modelled as **its own table** (created with
`read/write/create("team:<channelId>")`) is correctly and efficiently secured
right now, no core change required. The cost is a physical table per channel.

**The fork.** Two honest paths, and they change the app's data model:
1. **Model channels as per-channel tables** — correct and enforced today; ships
   the app immediately; exercises dynamic table creation. Downside: table
   proliferation, and it leaves the per-row feature unbuilt.
2. **Implement per-row document security in core (the proper BaaS feature)** — add
   a normalised per-row permissions column on document-security tables, persist
   permissions on write, and extend the read/update/delete policies to admit a
   row when its own permissions grant the action to one of the caller's resolved
   roles (`permissions->'read' ?| session_roles`). Bigger, security-critical, and
   must be verified against a real Postgres. Then a single `messages` table with
   `read("team:X")` per row is correct, and the platform gains a headline
   capability it currently only pretends to have.

**Security read.** The current state is not a leak — it fails **closed** (no rows
rather than too many), which is the safe direction. But it is a correctness gap
and a truthfulness gap: an API that accepts per-row permissions must enforce
them or reject them. Path 2 is the real fix; path 1 is a legitimate model that
sidesteps the gap without hiding it. **Chosen direction: pending owner decision**
(this is a core-feature investment, so it is being surfaced rather than assumed).

---

## Conventions this audit holds itself to
- Fix in core, then consume. No app-side shims that hide a platform gap.
- Server-derived authority only; never trust client-supplied identity/roles.
- Each entry states DX and security honestly, including residual risk and
  follow-ups, not just "done".
