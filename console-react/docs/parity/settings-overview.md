# Parity audit: Settings (+ API Key detail) & Overview

Flutter source (baseline):
- `console/lib/features/settings/settings_page.dart` (2540)
- `console/lib/features/settings/api_key_detail_page.dart` (1026)
- `console/lib/features/overview/overview_page.dart` (1157)

React port (under review):
- `console-react/src/features/settings/SettingsPage.tsx`
- `console-react/src/features/settings/ApiKeyDetailPage.tsx`
- `console-react/src/features/settings/scopes.tsx` (shared scope/expiry primitives)
- `console-react/src/features/overview/OverviewPage.tsx`
- `console-react/src/features/overview/UsageChart.tsx`

Legend: ✅ full parity · ⚠️ minor/behavioral/cosmetic divergence · ❌ missing

Overall: the ports are faithful. No feature is missing outright; all gaps below are minor
(toast feedback, one data endpoint, chart rendering style, a few status colors).

---

## Settings — General tab

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| Project name edit | ✔ | ✔ | ✅ | Controlled input, dirty tracking |
| Description edit | ✔ | ✔ | ✅ | |
| Project ID (read-only + copy) | ✔ | ✔ | ✅ | `ReadOnlyValue` + `CopyButton` |
| Save button (appears when dirty) | ✔ | ✔ | ✅ | Loading spinner both |
| Save success/error feedback | SnackBar "Project updated" / "Error: …" | none | ⚠️ | React mutation has no toast on success or error |
| Services card — core toggles (9) | ✔ | ✔ | ✅ | Both are local-only/non-persistent (`onChanged: {}` / local `useState`) |
| Services card — experimental (4, disabled) | ✔ | ✔ | ✅ | Same labels/icons/order |
| Danger zone — delete project | ✔ | ✔ | ✅ | |
| Delete — type-project-ID-to-confirm | field shown but **not enforced** | `submitDisabled={confirmText !== projectId}` | ✅ | React actually enforces the match (improvement) |
| Delete error feedback | SnackBar on error | none | ⚠️ | React navigates on success; no error surface |

## Settings — API Keys tab

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| Keys table (Name/Secret/Scopes/Expires/Created) | ✔ | ✔ | ✅ | Same columns/flex |
| Secret prefix cell (`prefix···`) | ✔ | ✔ | ✅ | |
| Scope summary ("N scopes" / "All scopes") | ✔ | ✔ | ✅ | |
| Expiry cell (Never / colored expired-red vs green) | ✔ | ✔ | ✅ | |
| Create key dialog — name | ✔ | ✔ | ✅ | |
| Create key dialog — expiry presets + custom | showDatePicker | `<input type="date">` | ✅ | Functionally equivalent |
| Create key dialog — expiry preview line | ✔ | ✔ | ✅ | Shared `expiryPreview` |
| Create key dialog — grouped scopes (select/deselect all, tri-state) | ✔ | ✔ | ✅ | Shared `ScopeGroups` / `SCOPE_GROUPS` |
| Secret reveal dialog (eye toggle + copy, once) | ✔ | ✔ | ✅ | |
| Delete key (with confirm) | custom confirm w/ specific text | generic DataTable `ConfirmDialog` ("Delete item") | ⚠️ | Confirm exists; message is generic vs Flutter's "Any applications using this key will lose access immediately." |
| Row tap → key detail page | ✔ | ✔ | ✅ | |
| Empty state | ✔ | ✔ | ✅ | |

## Settings — Webhooks tab

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| Search + count + Create button header | ✔ | ✔ | ✅ | |
| Webhook cards (name, active/disabled badge, URL) | ✔ | ✔ | ✅ | |
| Event tags on card | ✔ | ✔ | ✅ | |
| Create dialog (name, POST URL, event chips) | ✔ | ✔ | ✅ | Same 9 events incl. `credentials.*` |
| Delete webhook (confirm) | ✔ | ✔ | ✅ | Both use confirm dialog |
| Empty state | ✔ | ✔ | ✅ | |
| Client-side search filter (name/url) | ✔ | ✔ | ✅ | |

## Settings — Audit Log tab

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| Table columns (Method/Path/Status/Resource + hidden Action/User/IP + Time) | ✔ | ✔ | ✅ | `defaultVisible:false` on action/userId/ipAddress in both |
| Method chip (color per verb) | ✔ | ✔ | ✅ | |
| Status chip (500/400/2xx colors) | ✔ | ✔ | ✅ | |
| Filters: Method + Resource type | ✔ | ✔ | ✅ | Same option lists |
| Pagination / per-page | ✔ | ✔ | ✅ | |
| Search disabled ("not available server-side") | ✔ | ✔ | ✅ | |
| Empty state | ✔ | ✔ | ✅ | |

## Settings — Platforms tab (not a real tab)

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| Platforms tab | `_buildPlatformsTab` exists but `// ignore: unused_element`, not in tab list | omitted | ✅ | Both hide it — parity preserved. React correctly did not port the dead code. |

### Gaps (actionable) — Settings

1. ⚠️ **General save has no success/error feedback.** Flutter shows a SnackBar ("Project updated" / "Error: …") on save, and on delete failure. React `GeneralTab` `save`/`del` mutations surface nothing. Add a toast/inline error to match.
2. ⚠️ **API key delete confirm text is generic.** DataTable's built-in `ConfirmDialog` shows "Delete item"; Flutter's key delete uses "Any applications using this key will lose access immediately." Consider a key-specific confirm message. (Confirm dialog itself IS present, so not a functional gap.)

---

## API Key detail page

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| Back breadcrumb + name + "API Secret" badge | ✔ | ✔ | ✅ | |
| Key details card — Name | ✔ | ✔ | ✅ | |
| Key details card — Created | raw ISO string (`createdAt ?? '—'`) | `formatLongDate(createdAt)` "Mon D, YYYY" | ⚠️ | React formats it (improvement); values differ visually |
| Key details card — Expiration ("Never" / date) | ✔ | ✔ | ✅ | Both `formatLongDate`/equivalent |
| Key details card — Secret prefix + "shown once" note | ✔ | ✔ | ✅ | |
| Name editor + Save (dirty-gated) | ✔ | ✔ | ✅ | React also disables on empty name |
| Scopes editor (groups, select/deselect all, tri-state, per-scope) | ✔ | ✔ | ✅ | Shared `ScopeGroups` |
| Scopes Save (dirty-gated) | ✔ | ✔ | ✅ | |
| Expiration editor (preset dropdown + custom + preview) | ✔ | ✔ | ✅ | |
| Expiration Save (dirty-gated) | ✔ | ✔ | ✅ | PATCH `expiresAt` |
| Delete section (key row + delete button + confirm) | ✔ | ✔ | ✅ | "Created …" subline formatted in React |
| Regenerate key | ❌ absent | ❌ absent | ✅ | Task mentioned "regenerate/revoke"; neither implements regenerate. Delete = revoke. Parity (both lack regenerate). |
| Per-key usage stats | ❌ absent | ❌ absent | ✅ | Task mentioned "usage"; neither has it. Parity. |

### Gaps (actionable) — API Key detail

1. ⚠️ **Created-date rendering differs** (raw ISO in Flutter vs formatted in React). React's is nicer; if strict pixel-parity matters, align. Otherwise treat as acceptable improvement.
2. Note (not a React regression): neither app has key **regenerate** or **usage** — those were listed in the audit brief but are absent in the Flutter baseline too, so nothing to port.

---

## Overview — Overview tab

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| Header: project name + folder icon + project ID (`IdText`) | ✔ | ✔ | ✅ | |
| Tabs Overview / Activity | ✔ | ✔ | ✅ | Flutter local `_tabIndex`; React local `useState` (neither URL-persists — parity) |
| Usage graphs: Requests + Bandwidth | ✔ | ✔ | ⚠️ | **Rendering differs: Flutter draws bars (`_GraphPainter`); React draws an area+line (`UsageChart`).** Values/labels identical. |
| Usage graph value formatting (K/M, bytes) | ✔ | ✔ | ✅ | Same `formatNumber`/`formatBytes` |
| Period pill "30d" + chevron | ✔ | ✔ | ✅ | Static in both |
| Stat cards ×4 (Database/Storage/Auth/Functions) + hover + nav | ✔ | ✔ | ✅ | Same icons/labels/routes |
| Info cards: Project ID + API Endpoint + copy | ✔ | ✔ | ✅ | Flutter `Uri.base.origin/v1` ≈ React `window.location.origin/v1` |
| Recent Deployments — list + "View all" | ✔ | ✔ | ✅ | |
| Recent Deployments — data source | GET `/deploy/releases?limit=5`, reads `releases` | GET `/deploy?limit=5`, reads `deployments`/`releases`/`targets` | ⚠️ | **Different endpoint** — may return different/empty data than the Flutter one. |
| Recent Deployments — status dot + pill + time | separate dot + pill (`_statusColor`/`_statusLabel`) | `StatusChip` (dot built into pill) + time | ⚠️ | Dot present in both. Color map differs: `building`/`deploying` amber (Flutter) vs info-blue (React); `rolled_back` purple + "Rolled back" (Flutter) vs neutral + "rolled back" (React) |
| Empty "No deployments yet" | ✔ | ✔ | ✅ | |
| Services grid ×7 (Auth/Databases/Storage/Functions/Deploy/Workflows/Messaging) | ✔ | ✔ | ✅ | Same values/sublabels/routes; hover states both |

## Overview — Activity tab

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| Activity items derived from stat counts (>0) | ✔ | ✔ | ✅ | Same 6 defs (databases/users/buckets/functions/deployments/workflows) |
| Item card (icon tile + title + service) | ✔ | ✔ | ✅ | |
| Empty state ("No activity yet") | ✔ | ✔ | ✅ | |

### Gaps (actionable) — Overview

1. ⚠️ **Recent Deployments endpoint mismatch.** React calls `GET /deploy?limit=5`; Flutter calls `GET /deploy/releases?limit=5`. Confirm which endpoint returns release rows and align React to it, or the list may render empty / different data.
2. ⚠️ **Deployment status colors/labels diverge.** React's shared `StatusChip` maps `building`/`deploying` to info-blue (Flutter uses amber) and has no `rolled_back` variant (Flutter renders purple + "Rolled back"). Add `rolled_back` mapping and/or reconcile the building/deploying color if exact parity is required.
3. ⚠️ **Usage chart is intentionally a different visualization** (area/line vs bars). Cosmetic; flagged for design sign-off. Not a functional gap.
