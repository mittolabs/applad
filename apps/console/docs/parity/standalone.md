# Parity audit: Standalone (login, onboarding, projects, account)

Read-only audit. Flutter is the source of truth; the React port is compared against it.
Status: ✅ present · ⚠️ partial/simplified · ❌ missing.

---

## login
Flutter: `console/lib/features/login/login_page.dart` · React: `console-react/src/features/login/LoginPage.tsx`

| Sub-feature | Status | Notes |
|---|---|---|
| 4 modes: login / signup / forgot / reset | ✅ | Both use a `Mode` union with the same four states. |
| OAuth `console_token` callback → login + redirect | ✅ | Both call `loginWithToken` and strip the query param. Flutter redirects to `/onboarding`; React to `/projects`. |
| `reset_token` deep-link → reset mode | ✅ | Both switch to reset mode and pre-fill the token. |
| `error` param mapping (signup_disabled, oauth_cancelled) | ✅ | Same map in both. |
| Split branding panel (wide) + form panel | ⚠️ | React has a branding panel but a plain radial-gradient background; Flutter paints custom angular `_PanelShapes`. React copy differs ("The self-hosted backend…" vs Flutter's "Go from idea / to production today"), and Flutter swaps the tagline for signup mode — React does not. Cosmetic. |
| Narrow-screen logo row | ✅ | Both render a compact logo on `lg:hidden` / `!isWide`. |
| Social/OAuth buttons | ⚠️ | Flutter **always** renders a GitHub button on the login screen plus any extra configured providers; React renders buttons **only** if `/console/auth-providers` returns a non-empty list (no hardcoded GitHub fallback). Google "G" glyph in Flutter vs generic letter chip in React. |
| Email / password / name fields | ✅ | Same fields; React adds `autoComplete`. |
| Password visibility (eye) toggle on login/signup | ❌ | Flutter password field has a working eye/eyeOff toggle; React login uses a plain `type="password"` input with no reveal. |
| "Password must be at least 8 characters" hint (signup) | ❌ | Present in Flutter signup, absent in React. |
| Client-side reset validation (min 8, passwords match) | ❌ | Flutter validates length and confirm-match before calling; React does neither. |
| Signup policy checkbox gating submit | ✅ | Both disable submit until checked. Link targets differ (Flutter static text, React links to applad.dev/terms|privacy). |
| Forgot form: email + submit | ✅ | Both POST `/console/password-reset/request`. |
| Forgot: SMTP-not-configured surfaced token panel | ⚠️ | Flutter shows a dedicated token card with a "Use this token →" action that jumps into reset mode; React just prints the reset URL inside the success banner text. |
| Reset form: **manual token paste** field | ❌ | Flutter has a "Reset token" text field so a user can paste a token; React reset mode has no token input at all (only works via the `reset_token` URL param). |
| Reset form: **confirm password** field | ❌ | Flutter has New + Confirm password with match check; React reset has only a single new-password field. |
| Success banner ("Password updated…") | ✅ | Both show success; Flutter auto-returns to login after 2s, React returns immediately. |
| Version footer (`v{version}`) | ❌ | Flutter renders app version in the form panel footer; React shows a "© 2026 Mittolabs LTD" line in the branding panel instead — no version. |
| Loading overlay during OAuth ("Signing you in…") | ⚠️ | Flutter shows a full-panel spinner overlay during token exchange; React just navigates without a dedicated overlay. |
| Friendly error mapping | ✅ | Both map HTTP errors to friendly copy (`friendlyError`). |
| Endpoints (`signup-status`, `auth-providers`, `password-reset/*`) | ✅ | Paths match the Flutter providers. |

### Gaps (actionable)
- ❌ Reset mode is missing the **manual token paste field** and the **confirm-password field** — the Flutter reset flow (paste token + confirm) cannot be reproduced in React unless the user arrives via the email deep-link.
- ❌ No **password eye/reveal toggle** on the login/signup password input.
- ❌ No **version footer**; ❌ no signup "min 8 chars" hint; ❌ no client-side reset validation (length/match).
- ⚠️ OAuth buttons: React drops the always-present GitHub button, showing nothing when `auth-providers` is empty.
- ⚠️ Forgot "SMTP not configured" token surfacing is downgraded to plain banner text (no "Use this token →" jump).

---

## onboarding
Flutter: `console/lib/features/onboarding/onboarding_page.dart` · React: `console-react/src/features/onboarding/OnboardingPage.tsx`

**Major mismatch — the two implement different flows.** Current Flutter onboarding is a single-step **organization-creation** screen. React implements an older **3-step project stepper** (project → API key → SDK snippet). They do not correspond.

| Sub-feature (Flutter = org-creation) | Status | Notes |
|---|---|---|
| Personalized welcome (`Welcome, {name}` / `Welcome to Applad`) | ⚠️ | Flutter greets the signed-in user by name; React shows a static "Welcome to Applad". |
| Single **Organization name** field | ❌ | Flutter's whole purpose. React has no org-name field. |
| Org name pre-filled `"{firstName}'s Workspace"` | ❌ | Not in React. |
| "Get Started" → `POST /organizations` → set current org → `/org/{id}/projects` | ❌ | React's primary action creates a **project** (`POST /organizations/{orgId}/projects`), not an org. |
| Guard: redirect to `/login` if no user | ❌ | Flutter guards; React has no auth guard here. |
| Guard: if orgs already exist, skip onboarding → `/org/{id}/projects` | ❌ | Flutter auto-skips; React does not. |
| Error handling on create | ✅ | Both surface errors (snackbar vs inline banner). |
| **React-only** 3-step stepper (project → API key → SDK) | n/a | Not present in current Flutter at all — React implements functionality Flutter no longer has (create project, create key `POST /projects/{id}/keys`, Node SDK snippet, "Skip for now"). |

### Gaps (actionable)
- ❌ The React onboarding **does not port the current Flutter org-creation flow** at all: no org-name field, no `POST /organizations`, no name pre-fill, and none of the redirect guards (no-user → login, orgs-exist → skip).
- ❌ React's action creates a *project*, so a brand-new user with **no organization** cannot create one from onboarding the way Flutter intends (Flutter's project routes require an org). Reconcile which flow onboarding should own (org creation vs project stepper).

---

## projects
Flutter: `console/lib/features/projects/projects_page.dart` · React: `console-react/src/features/projects/ProjectsPage.tsx`

| Sub-feature | Status | Notes |
|---|---|---|
| Top bar: logo, org switcher (+create org), Support, ⌘K search, user avatar → account | ✅ | Provided by shared `TopNav` via `StandaloneLayout`. |
| 6 tabs: Projects / Members / Roles / Usage / Activity / Settings | ✅ | Same tab set and order. |
| Heading = org name + Invite button | ⚠️ | Present. Flutter's Invite opens the invite dialog directly; React's Invite just switches to the Members tab. |
| Overlapping **member avatars** in heading (`+N` overflow from `totalMembers`) | ❌ | Flutter draws stacked avatar circles; React shows a single org-initial circle. |
| **Projects tab** — search + create + pagination | ✅ | Both filter by name/id, paginate (default 6/page). |
| Project cards: color seed, first-letter icon, name, desc/id, created-at | ✅ | Ported (color hashing, relative time). |
| Card kebab menu: Settings / Delete | ✅ | Both have dropdown with Settings + destructive Delete. |
| Dashed "Create a new project" placeholder card | ✅ | Present in both. |
| Delete project confirm dialog | ✅ | Both confirm then `DELETE /projects/{id}`. |
| Search-empty state ("No projects matching …") | ⚠️ | Flutter shows a dedicated searchX empty message; React only shows a first-run `EmptyState` (when no query) and otherwise renders just the create-card with no "no matches" message. |
| **Members tab** — list + invite + remove | ⚠️ | Present but simplified (see below). |
| Invite dialog: email + **name** + **role (owner/admin/member)** + role-description text | ⚠️ | React invite has email + role only; **no name field**, role limited to **member/admin** (no owner), and no per-role description helper text. |
| Member row **inline role dropdown** → `PATCH /organizations/{org}/members/{id}` | ❌ | Flutter lets you change a member's role inline; React shows a read-only `StatusChip` and never wires the role-update PATCH. |
| Owner rows: dropdown disabled + remove hidden | ⚠️ | Flutter special-cases owners; React shows Remove for everyone. |
| Remove member confirm | ✅ | Both confirm then DELETE. |
| **Roles tab** — permission matrix | ⚠️ | Flutter: heading + subtitle + **11** permission rows with check/minus icons. React: bare table, **5** rows, different permission set, no heading/subtitle. |
| **Usage tab** — stat cards | ⚠️ | Metrics differ. Flutter: Projects / Total users / Storage used (byte-formatted) / Executions, reading `totalProjects,totalUsers,totalStorage,totalExecutions`, plus a heading, subtitle and Members info card. React: Projects / Members / Databases / Storage reading `projects,members,databases,storage` (raw), no byte formatting, no heading/info card. Field keys likely mismatch the API. |
| **Activity tab** — table | ⚠️ | Flutter: full table (Action, Project, Path, colored Status code, relative Time), Refresh button, heading, empty state. React: flat list of action + raw `createdAt` only — no columns, status, path, project, relative time, refresh, or heading. |
| **Settings tab** — org name update | ✅ | Both wired (`PATCH /organizations/{id}`). Flutter uses a dialog; React inline input+Save. |
| Settings: **Organization ID** card (copyable `IdText`) | ❌ | Flutter shows an Org ID card with copy; React omits it entirely. |
| Settings: Danger zone → delete org | ✅ | Both confirm then `DELETE /organizations/{id}`. |
| Create-org dialog (⌘K "Create organization") | ✅ | Ported (React opens via `?create=org`). |

### Gaps (actionable)
- ❌ Member **inline role change is not wired** — React never calls `PATCH …/members/{id}`, so roles are display-only.
- ❌ Settings tab is **missing the Organization ID card** (copyable ID).
- ❌ Heading **member-avatar stack** (with `+N` overflow) not ported.
- ⚠️ Invite dialog drops the **name field** and the **owner** role option (+ role descriptions).
- ⚠️ Usage tab reads **different stat keys** than Flutter (`projects/members/databases/storage` vs `totalProjects/totalUsers/totalStorage/totalExecutions`) and drops byte formatting — likely shows blanks against the real API.
- ⚠️ Activity and Roles tabs are heavily simplified (list vs table; 5 vs 11 permission rows).

---

## account
Flutter: `console/lib/features/account/account_page.dart` · React: `console-react/src/features/account/AccountPage.tsx`
(Audited against the recently-updated Flutter: sign-out, two-column sections, MFA switch, password fields.)

| Sub-feature | Status | Notes |
|---|---|---|
| Top nav (no project switcher) | ✅ | `StandaloneLayout showOrg={false}` → shared TopNav. |
| Title = user name, with **Sign out** button + confirm dialog | ✅ | Both confirm, then logout → `/login`. |
| 4 tabs: Overview / Sessions / Activity / Organizations | ✅ | Same set. |
| Two-column section cards (title/subtitle left, controls right) | ✅ | React `Card` ports `_AccountSection` layout. |
| **Name** update (separate section, own Update button) | ⚠️ | Flutter has separate Name and Email sections, each `PATCH /console/me` with a single field and its own Update button. React merges both into one "Profile" card with one "Save changes" that PATCHes `{name, email}` together. |
| **Email** update (separate section) | ⚠️ | Merged into the Profile card in React (see above). |
| **Password** update (old + new) | ⚠️ | **Field-name bug:** backend + Flutter send `oldPassword`; React sends `currentPassword` (`{ currentPassword, password }`) — the old-password check will not receive the value, so the change likely fails. |
| Password **eye/reveal** toggle | ✅ | React `PasswordField` has a working Eye/EyeOff toggle. (Note: Flutter's eye icon is actually static/non-toggling — React is better here.) |
| MFA section with switch | ✅ | Both are non-functional placeholders (local state only, no backend call). Parity. |
| Delete-account section with avatar chip + confirm | ✅ | Both show the avatar chip and confirm then `DELETE /console/me` → logout. |
| Sessions tab (placeholder) | ✅ | Both placeholders. React uses the Monitor icon (Flutter uses monitor too). |
| Activity tab (placeholder) | ⚠️ | Both placeholders; React reuses the Monitor icon where Flutter uses an Activity icon. Cosmetic. |
| Organizations tab | ✅+ | Flutter is a placeholder with a "View organizations" button. React actually **lists the orgs** (functional) and links each to `/org/{id}/projects` — exceeds Flutter. |

### Gaps (actionable)
- ❌ **Password change is likely broken**: React posts `currentPassword` but the backend (`console/handler.go`) and Flutter expect `oldPassword`. Rename the field to `oldPassword`.
- ⚠️ Name and Email are **merged** into one Profile card (one PATCH) instead of Flutter's two independent sections/buttons — acceptable but a behavioral divergence.
- (No functional regressions otherwise; React's Organizations tab and password eye-toggle are improvements over Flutter.)
