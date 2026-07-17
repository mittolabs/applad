# Parity audit — flags, vault, environments, realtime, health

Read-only comparison of the Flutter console features against their React (`console-react`) ports.

- Flutter source: `console/lib/features/{flags,vault,environments,realtime,health}/*.dart`
- React source: `console-react/src/features/{flags,vault,environments,realtime,health}/*.tsx`

Legend: ✅ full parity · ⚠️ present with a meaningful difference · ❌ missing

Overall the five features are faithful, near-complete ports. Most gaps are small endpoint/behaviour divergences; in several places React is actually *ahead* of Flutter (inline var editing, WS connect URL, auto-polling). One likely functional bug (flags status filter) is worth fixing.

---

## Flags

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| List (DataTable: ID/Key/Name/Type/Enabled) | ✅ | ✅ | ✅ | Columns + labels match |
| Inline enabled toggle | `PATCH /flags/{id}/toggle` (no body) | `PATCH /flags/{id}` `{enabled}` | ⚠️ | React does not use the dedicated `/toggle` endpoint — relies on partial-update PATCH |
| Type badge / row icon+color | ✅ | ✅ | ✅ | `flags-shared.tsx` mirrors icon/color tables |
| Create flag dialog (key/name/desc/type/default) | ✅ | ✅ | ✅ | |
| Delete flag from list | ✅ | ✅ | ✅ | |
| Filters — Type | ✅ (`boolean/string/number/json`) | ✅ | ✅ | |
| Filters — Status | options `enabled`/`disabled` (match cell value) | options `true`/`false` (cell value is `enabled`/`disabled`) | ⚠️ | React filter values don't match `getCellValue` output → status filter likely never matches client-side |
| Detail: breadcrumb + key/type header + StatusChip | ✅ | ✅ | ✅ | |
| Detail tabs (Settings/Rules/Overrides/Stats) | ✅ | ✅ | ✅ | React persists tab in URL via `useTabIndex` |
| Settings: detail rows + Edit dialog | Edit via `PUT /flags/{id}` | Edit via `PATCH /flags/{id}` | ⚠️ | Method differs; React also adds an inline Status **Switch** in Settings (enhancement) |
| Settings: danger zone delete | ✅ | ✅ | ✅ | |
| Rules: list + add (type/attr/operator/condValue/serve/rollout slider) | ✅ | ✅ | ✅ | Same operators + conditions array build |
| Rule card (conditions summary / value / rollout bar) + delete | ✅ | ✅ | ✅ | |
| Overrides: list + add (targetType/targetId/value) + delete | ✅ | ✅ | ✅ | Delete path `/overrides/{type}/{id}` matches |
| Stats: 3 stat cards + value distribution bars | ✅ | ✅ | ✅ | |

### Gaps (actionable)
- **Status filter mismatch (likely bug):** React `FlagsPage` status filter options use `true`/`false`, but `getCellValue('enabled')` returns `'enabled'`/`'disabled'`. Align filter option values with the cell values (`enabled`/`disabled`) as Flutter does, or normalize the cell value.
- **Toggle endpoint divergence:** list toggle and Settings toggle call `PATCH /flags/{id}` instead of Flutter's `PATCH /flags/{id}/toggle`. Confirm the backend accepts partial `{enabled}` updates on the base route, or switch to `/toggle` for parity.
- **Edit method divergence:** Flutter edit uses `PUT`, React uses `PATCH`. Confirm both are supported; pick one for consistency.

---

## Vault

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| Credentials list (ID/Name/Type/Description/Expires) | ✅ | ✅ | ✅ | |
| Type badge + expiry chip | ✅ | ✅ | ✅ | `credentials.ts` / `CredentialBadges.tsx` mirror tables |
| Rotate keys button + confirm (`POST /credentials/rotate`) | ✅ | ✅ | ✅ | React uses toast for result count |
| Add credential button | ✅ | ✅ | ✅ | |
| Search | client-side filter on name/type | server-side via `useResourceList` search | ⚠️ | Different mechanism; results may differ from Flutter's in-memory filter |
| Create/edit modal (name/desc/type/secret+obscure/protected/expiry) | ✅ | ✅ | ✅ | React requires secret re-entry on edit, same as Flutter |
| Expiry input | full `showDatePicker` (date+picker) | `<input type="date">` (date-only) | ⚠️ | Cosmetic; both send ISO. React truncates to day |
| Detail modal — Details tab (type/protected/keyVersion/expires/created) | ✅ | ✅ | ✅ | |
| Reveal secret + copy (`GET /credentials/{id}`) | ✅ | ✅ | ✅ | |
| Detail modal — Access log tab (action badge/ip/time) | ✅ | ✅ | ✅ | `GET /credentials/{id}/accesses?limit=50` |
| Edit-from-detail | ✅ | ✅ | ✅ | |
| Credential types (7: generic/api_key/database/ssh/webhook/tls/oauth2) | ✅ | ✅ | ✅ | |

### Gaps (actionable)
- **Search behaviour:** Flutter filters the loaded page client-side (name/type); React delegates to `useResourceList` server search. Low risk, but list results won't match Flutter for partial/paged data. Verify backend `/credentials` honours the `search` param.
- **Expiry granularity:** React date input is day-only vs Flutter's date picker — acceptable, note if time-of-day expiry matters.

---

## Environments

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| Env list sidebar (color dot / name / default label / delete) | ✅ | ✅ | ✅ | 220px vs w-56; delete reveals on hover in React |
| Env color mapping (prod/staging/dev) | ✅ | ✅ | ✅ | Identical hex values |
| New environment dialog (name/slug/branch) | ✅ | ✅ | ✅ | Slug auto-derived from name when blank |
| Delete environment + confirm | ✅ | ✅ | ✅ | |
| Selected-env URL param | `?envId=` | `?env=` | ⚠️ | Query-param name differs → deep links not cross-compatible |
| Detail: name header + tabs (Overview/Variables/Settings) | ✅ | ✅ | ✅ | React persists tab in URL |
| Overview: 4 info cards (slug/branch/domain/variables) | ✅ | ✅ | ✅ | |
| Variables: list + obscure toggle + add dialog + delete + dirty Save | ✅ | ✅ | ✅ | |
| Variables: edit existing var in place | ❌ (delete + re-add only) | ✅ (pencil edit + duplicate-key validation) | ⚠️ | React is ahead — enhancement over Flutter |
| Settings: name/branch/domain + Save | ✅ | ✅ | ✅ | Both `PUT /deploy/environments/{id}` |

### Gaps (actionable)
- **URL param name:** React uses `?env=`, Flutter uses `?envId=`. Harmless within React, but pick one convention if deep-link parity across consoles matters.
- No functional deficits; React variable editor is a superset of Flutter.

---

## Realtime

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| Header + description | ✅ | ✅ | ✅ | |
| Tabs (Overview/Channels) | ✅ | ✅ | ✅ | React persists tab in URL |
| Stats poll (`GET /realtime/stats`) | Timer 5s | `refetchInterval: 5000` | ✅ | Same cadence; React dedupes across tabs via shared query key |
| Overview: stat cards (connections/channels) | ✅ | ✅ | ✅ | |
| Overview: "How it works" (3 info rows) | ✅ | ✅ | ✅ | Identical copy |
| Overview: channel patterns (3 rows + copy) | ✅ | ✅ | ✅ | React shows copied ✓ feedback |
| Overview: WebSocket connect URL / WS info | ❌ | ✅ (`CodeBlock` with `ws(s)://.../v1/realtime?project=`) | ⚠️ | React is ahead — extra "Connect" section |
| Channels tab: count + channel rows w/ subscriber badge | ✅ | ✅ | ✅ | |
| Empty state | ✅ | ✅ | ✅ | |

### Gaps (actionable)
- No missing Flutter functionality. React adds a WS connect-URL block Flutter lacks; consider back-porting to Flutter if cross-console parity is desired (not a React gap).

---

## Health

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| Header + subtitle (project-aware) | ✅ | ✅ | ✅ | |
| Refresh button | ✅ (invalidate) | ✅ (`refetch`, shows loading) | ✅ | |
| Fan-out to `/health`, `/health/db`, `/health/cache` | ✅ | ✅ | ✅ | Same 3 endpoints + labels (Gateway/PostgreSQL/Redis) |
| Overall status derivation (pass/warn/fail) | ✅ | ✅ | ✅ | Identical logic |
| Overview card (icon/headline/pass count/timestamp) | ✅ | ✅ | ✅ | React formats timestamp via `toLocaleTimeString`; Flutter shows raw ISO |
| Per-service cards (status dot/pill, endpoint, ping, error) | ✅ | ✅ | ✅ | |
| Auto-refresh | ❌ (manual only) | ✅ (`refetchInterval: 10000`) | ⚠️ | React polls every 10s — enhancement |

### Gaps (actionable)
- No missing Flutter functionality. React adds 10s auto-poll (enhancement) and friendlier timestamp formatting. Optional: back-port auto-poll to Flutter for consistency.

---

## Summary

| Feature | Verdict | Notable gaps |
|---|---|---|
| Flags | ✅ parity, 3 minor diffs | status-filter value mismatch (likely bug); toggle uses base PATCH not `/toggle`; edit PATCH vs PUT |
| Vault | ✅ parity | client vs server search; date-only expiry |
| Environments | ✅ parity (React ahead) | `?env=` vs `?envId=` param name; React adds inline var edit |
| Realtime | ✅ parity (React ahead) | React adds WS connect URL |
| Health | ✅ parity (React ahead) | React adds 10s auto-poll |

No feature is missing sub-functionality in React. The only item that could be a real defect is the **Flags status filter** value mismatch.
