# Observe suite — Flutter → React parity audit

Read-only comparison of the Flutter console "Observe" observability suite against the React port.

- **Flutter source:** `console/lib/features/observe/{observe_page, observe_overview, observe_errors, observe_releases, observe_logs, observe_replays, observe_uptime, observe_crons, observe_alerts, observe_performance, observe_providers, observe_shared}.dart`
- **React port:** `console-react/src/features/observe/{ObservePage, ObserveOverview, ObserveErrors, ObserveReleases, ObserveLogs, ObserveReplays, ObserveUptime, ObserveCrons, ObserveAlerts}.tsx` + `observe-shared.tsx`

Legend: ✅ present · ⚠️ partial / cosmetic delta · ❌ missing

**Summary:** The port is near-complete and faithful. All 8 routed sub-views (Overview, Errors, Releases, Logs, Replays, Uptime, Crons, Alerts) are ported, including every list, detail view, triage/resolve/ignore action, enable toggle, delete, add-monitor / add-cron / create-rule dialog, log filter/column-picker/live toggle, firing-incidents banner, and the backendless replay player + session-timeline placeholders. Both consoles wire the same `/observe/*` REST endpoints. The one real omission is the **Performance** view (`observe_performance.dart` — 6 stat cards, web-vitals grid, response-time chart, slowest-endpoints table, recent-traces table), which has no React port. Note it is *also dead code in Flutter*: it is not wired into `observe_page.dart`'s section map or `switch`, so it is unreachable in both apps. Remaining deltas are cosmetic (column sortability, pagination behavior).

---

## Page shell (`ObservePage`)

| Sub-feature | Status | Notes |
|---|---|---|
| Section map (8 entries: observe/errors/releases/logs/replays/uptime/crons/alerts) | ✅ | Identical keys + title/subtitle copy |
| Active view derived from last path segment | ✅ | `pathname.split('/').pop()` mirrors Flutter `uri.path.split('/').last` |
| Header title + subtitle | ✅ | Same styling intent |
| Sub-view `switch` / fallback to Overview | ✅ | Same default branch |
| Performance section | n/a | Not in Flutter's section map either — see Performance note below |

## Overview (`ObserveOverview` ← `observe_overview.dart`)

| Sub-feature | Status | Notes |
|---|---|---|
| Loading spinner | ✅ | `Loader2` vs `CircularProgressIndicator`, same accent |
| On-error renders nothing | ✅ | React `return null` mirrors Flutter `SizedBox()` |
| 5 stat cards (Errors 24h, P95, Uptime, Apdex, Log volume) | ✅ | Same labels, colors, icons, value formatting (`obFmtNum`, apdex `toFixed(2)`) |
| Web Vitals row (LCP/FID/CLS/TTFB/FCP) | ✅ | Same 5 vitals, good/poor thresholds, rating chip, color logic. Flutter uses fixed 5-across `Row`; React uses `flex-wrap` (cosmetic) |
| Service Health cards + empty state | ✅ | Same status→color/label map, latency/uptime line, empty-state copy |
| Recent Errors (top 5) + "View all" nav | ✅ | Same `slice(0,5)`, compact row, navigate to `/project/:id/errors`, empty-state copy |

## Errors (`ObserveErrors` ← `observe_errors.dart`)

| Sub-feature | Status | Notes |
|---|---|---|
| Errors table (Level, Title, Status, Events, Users, Last seen) | ✅ | Same columns/values |
| Level chip (dot + colored label) | ✅ | `levelColor` port |
| Status badge (unresolved/resolved/ignored) | ✅ | Same tri-color |
| Events count formatting | ✅ | `obFmtNum` |
| Status + Level filters | ✅ | Same options |
| Search (title) | ✅ | |
| Row delete = ignore (`PATCH /observe/errors/:id/ignore`) | ✅ | React adds toast on error (minor enhancement) |
| Row click → detail | ✅ | |
| "Export" toolbar button | ✅ | Flutter passes `createLabel:'Export'` with **no** `onCreateTap`, so no button renders (confirmed in `AppDataTable`); React omits it. Match — no dead button in either |
| Detail: back link, level dot, title | ✅ | |
| Detail: Resolve / Ignore actions (unresolved) | ✅ | `PATCH …/resolve` / `…/ignore`, then refetch + back |
| Detail: resolved/ignored status badge (non-unresolved) | ✅ | |
| Detail stats (Events, Affected users, First/Last seen) | ✅ | |
| Detail: stack trace block | ✅ | Same mono/dark styling |
| Detail: breadcrumbs (typed icon+color, timeline) | ✅ | Same type→icon/color map |
| Detail: Tags panel | ✅ | |
| Detail: USER / REQUEST / RUNTIME context panels | ✅ | `ObContextPanel` port |
| Detail: Activity feed (typed icon) | ✅ | |
| Column sortability on Title | ⚠️ | Flutter `title` is sortable (default true); React marks only count/users/lastSeen sortable — text columns non-sortable. Cosmetic |

## Releases (`ObserveReleases` ← `observe_releases.dart`)

| Sub-feature | Status | Notes |
|---|---|---|
| Releases table (Version, Crash-free, New/Regressed/Fixed, Commits, Created) | ✅ | Same columns/values |
| Version cell (tag icon, mono, env badge) | ✅ | |
| Crash-free % color thresholds (99/95) | ✅ | |
| Colored issue counts (new=orange, regressed=red, fixed=green) | ✅ | `CountCell` |
| Search (version/environment) | ✅ | |
| Row click → detail (by `$id` or version) | ✅ | |
| Detail: back link + version header | ✅ | |
| Detail: Commits list (sha7, message, author) + empty card | ✅ | |
| Detail: Issues list (typed icon/color) + empty card | ✅ | |
| Column sortability on Version | ⚠️ | Flutter Version sortable (default); React marks it non-sortable. Cosmetic |

## Logs (`ObserveLogs` ← `observe_logs.dart`)

| Sub-feature | Status | Notes |
|---|---|---|
| Toolbar search (message) | ✅ | `ObSearchField` port |
| Filter popover: Level | ✅ | Same 5 levels, "Any" option, active count badge |
| Filter popover: Source (dynamic from data) | ✅ | Distinct sorted sources |
| "Clear all" filters | ✅ | |
| Column picker popover (Time/Level/Source/Message; Message locked) | ✅ | Same lock + min-1-visible rule; `VerticalBars` glyph + count |
| Live / Paused toggle | ✅ | **Visual only in both** — neither polls or tails; `live` state is decorative. Backendless placeholder, matched |
| Refresh button | ✅ | `query.refetch()` vs `invalidate` |
| Log line (ts / level / source / message, mono, level colors) | ✅ | Same color map incl. debug slate / info `#94A3B8` |
| Expandable per-line `meta` block | ✅ | Chevron toggle, same indented mono block |
| Empty state (filters) | ✅ | Same copy |

## Replays (`ObserveReplays` ← `observe_replays.dart`)

| Sub-feature | Status | Notes |
|---|---|---|
| Replays table (User, Page, Duration, Errors, Flags, Browser, Started) | ✅ | Same columns |
| Duration formatting (`Ns` / `Nm Ns`) | ✅ | |
| Errors badge (hidden when 0) | ✅ | |
| Flags badges (Rage / Dead) | ✅ | |
| Flags filter (has_errors / rage_click) | ✅ | React applies filter explicitly in `useMemo`; Flutter routes it through `AppDataTable` filter on the joined flags string — functionally equivalent |
| Search (user/URL) | ✅ | |
| Row click → detail | ✅ | |
| Detail: back link + "Session — user" + duration | ✅ | |
| **Placeholder replay player** (dark box, play icon, "Replay player") | ✅ | Backendless placeholder in both — no real player. Matched |
| Detail: Events timeline (first 20, typed icon) | ✅ | |
| Detail: Network requests (first 20, method/status/url/dur) | ✅ | |
| Detail: Console lines (when present) | ✅ | |

## Uptime (`ObserveUptime` ← `observe_uptime.dart`)

| Sub-feature | Status | Notes |
|---|---|---|
| Monitors table (Status, Name, URL, Uptime, Latency, Checked) | ✅ | Same columns |
| Status cell (dot + up/down/degraded color) | ✅ | |
| Uptime % thresholds (99.9 / 99.0) | ✅ | |
| Status filter (up/down/degraded/paused) | ✅ | |
| Search (name/URL) | ✅ | |
| Row delete (`DELETE /observe/uptime/:id`) | ✅ | React adds toast |
| Add monitor button + dialog | ✅ | |
| Dialog: Name, URL/host, Check type (http/tcp/ping/keyword), Interval (30s–30m) | ✅ | Same fields, options, labels; posts `{name,url,checkType,intervalSecs}` |
| Dialog submit-disable until name+url | ✅ | React `submitDisabled` (Flutter has none) — minor enhancement |

## Crons (`ObserveCrons` ← `observe_crons.dart`)

| Sub-feature | Status | Notes |
|---|---|---|
| Monitors table (Status, Name, Schedule, Timezone, Last/Next run, Enabled) | ✅ | Same columns |
| Status cell (ok/missed/failed/running/waiting icon+color) | ✅ | `statusMeta` port |
| Schedule cell (mono accent) | ✅ | |
| Enabled toggle (`PATCH …/toggle`) | ✅ | React stops row-click propagation; both refetch |
| Status filter | ✅ | Same options |
| Search (name/schedule) | ✅ | |
| Row delete (`DELETE /observe/crons/:id`) | ✅ | React adds toast |
| Add monitor button + dialog | ✅ | |
| Dialog: Name, Schedule (cron) + hint, Timezone (8 zones), Grace period (1–60m) | ✅ | Same fields/options; posts `{name,schedule,timezone,gracePeriod}` |

## Alerts (`ObserveAlerts` ← `observe_alerts.dart`)

| Sub-feature | Status | Notes |
|---|---|---|
| Firing-incidents banner ("Firing Now" + count badge) | ✅ | Shown only when incidents present |
| Incident card (severity icon/color, ruleName, value, fired-ago, severity chip) | ✅ | Same layout |
| Rules table (Severity, Name, Condition, Window, Channel, Enabled, Last fired) | ✅ | Same columns |
| Severity cell (dot + color) | ✅ | |
| Condition string (`metric op threshold`, op symbol map) | ✅ | `conditionStr` port |
| Window value (`time_window` ?? `window`) | ✅ | Same fallback |
| Enabled toggle (`PATCH …/toggle`) | ✅ | |
| Severity filter | ✅ | |
| Search (name/metric) | ✅ | |
| Row delete (`DELETE /observe/alerts/:id`) | ✅ | React adds toast |
| Create rule button + dialog | ✅ | |
| Dialog: Name, Metric (10 opts), Condition op + Threshold, Window (6 opts), Severity, Notify-via (email/slack/webhook/pagerduty/opsgenie) | ✅ | Same fields/options; posts `{name,metric,operator,threshold,window,severity,channel,enabled:true}` |

## Performance (`observe_performance.dart` — NOT one of the 8 sub-views)

| Sub-feature | Flutter | React | Status |
|---|---|---|---|
| Whole view (perf cards P50–P99 + Req/s + Apdex, Web-Vitals grid incl. INP, response-time line chart, slowest-endpoints table, recent-traces table, empty state) | Fully built as `ObPerformanceTab`, but **unrouted** — absent from `observe_page.dart` section map and `switch`, so unreachable | No port exists | ❌ (see note) |
| `performanceProvider` / `GET /observe/performance` | Present (only consumed by the unrouted tab) | No hook uses `/observe/performance` | ❌ |
| `ObLineChart` chart primitive | In `observe_shared.dart` | Ported to `observe-shared.tsx` but **unused** (only Performance consumed it) | ⚠️ dead port |

Because Performance is dead in Flutter too, this is not a user-visible regression — but if the Flutter view is ever wired up, the React side has no equivalent to route to.

---

### Gaps (actionable)

1. **❌ Performance view has no React port.** `observe_performance.dart` (perf percentile cards, 6-vital grid with INP, 24h response-time `ObLineChart`, slowest-endpoints table, recent-traces table, empty state) is fully implemented in Flutter but has no React counterpart, and `GET /observe/performance` is never called from React. *Caveat:* it is also unreachable in Flutter (not in the section map/switch), so porting it requires first deciding whether Performance should be a routed sub-view at all. `ObLineChart` is already ported to `observe-shared.tsx` (currently unused) and would be reused.

2. **⚠️ Text-column sortability differs.** Flutter's `AppTableColumn` defaults to `sortable: true`, so Title (Errors), Version (Releases), Name/URL (Uptime/Crons/Alerts) are click-sortable. React's `DataTableColumn` defaults to `sortable: false` and only the numeric/time columns are explicitly opted in, so those text columns are not sortable. Cosmetic; to match, add `sortable: true` to the text columns.

3. **⚠️ Pagination behavior differs.** Flutter's Observe tables pass `perPage = <all rows>` with no-op prev/next, effectively disabling pagination (single scrolling page). React's `DataTable` defaults to `perPage = 12` and none of the Observe views override it, so React paginates client-side at 12/page. Not a functional loss (arguably an improvement) but a visible behavior delta from Flutter. To mirror Flutter, pass a large `perPage`.

4. **Live-logs tailing is a placeholder in both (not a gap).** The Logs "Live/Paused" toggle is decorative in both consoles — neither polls nor streams. Noted only so it is not mistaken for a React regression.
