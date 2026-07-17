# Databases — Flutter → React parity audit

Read-only comparison of the Flutter console `Databases` feature
(`console/lib/features/databases/databases_page.dart`, 4332 lines) against the
React port (`console-react/src/features/databases/*.tsx`).

Legend: ✅ full parity · ⚠️ partial / behaves differently · ❌ missing in React

React file map:
- `DatabasesPage.tsx` — 3-level selection shell
- `DatabaseList.tsx` — Level 1 (list + Usage)
- `DatabaseDetail.tsx` — Level 2 (Tables/SQL/Usage/Settings)
- `TablesPanel.tsx`, `SqlConsole.tsx` — Level 2 tab bodies
- `TableDetail.tsx` — Level 3 (Rows/Columns/Indexes/Relationships/Settings)
- `RowsPanel.tsx`, `ColumnsPanel.tsx`, `IndexesPanel.tsx`, `RelationshipsPanel.tsx` — Level 3 tab bodies
- `shared.tsx` — helpers (BackHeader, MetricPill, ChipGroup, type icons, fmtDate)

---

## Level 1 — Database list (+ Usage)

| Sub-feature | Status | Notes |
|---|---|---|
| Page header + subtitle | ⚠️ | Rendered **twice**: `DatabasesPage` renders `<h1>Databases</h1>` + `PageTabs`, and the child `DatabaseList` renders its **own** `<h1>Databases</h1>` + subtitle + a second `PageTabs`. On the list screen you get duplicate headers and two tab bars. Flutter renders one. |
| Databases / Usage top tabs | ⚠️ | Present but duplicated (see above). Two independent tab states drive two Usage variants. |
| Server-side search (name/ID) | ✅ | `useResourceList` passes `search` param; parity with Flutter `_dbSearchProvider`. |
| Pagination (limit/offset, per-page dropdown, prev/next) | ✅ | `useResourceList` + `DataTable` footer; server offset/limit like Flutter `SearchListFooter`. |
| URL `?page` sync | ✅ | `useResourceList` syncs `?page` (Flutter does too via `pageFromQuery`/`withQuery`). |
| Create database | ✅ | `POST /databases {name}`. |
| Delete database (row action) | ✅ | `DELETE /databases/{id}` with confirm. |
| Open database (row click) | ✅ | Drills into Level 2. |
| Created-date column | ✅ | `fmtDate(createdAt / $createdAt)`. |
| Usage tab — 3 stat cards (Total databases/tables/rows) | ⚠️ | Present only in `DatabaseList`'s internal `UsageTab` (behind the duplicated inner tab). The `DatabasesPage`-level Usage tab shows a bare "Usage coming soon" box with **no stat cards**. Flutter always shows the 3 `—` stat cards + placeholder. |
| Usage tab — "charts coming soon" placeholder | ✅ | Both stub. |

---

## Level 2 — Database detail (Tables / SQL / Usage / Settings)

| Sub-feature | Status | Notes |
|---|---|---|
| Back header (name + ID) | ✅ | `BackHeader` in `shared.tsx`. |
| Tab bar Tables/SQL/Usage/Settings | ✅ | `DatabaseDetail` `TABS`. |
| **Tables** — list (ID/Name/Created) | ✅ | `TablesPanel`. |
| **Tables** — create table | ✅ | `POST /databases/{db}/tables {name}`. |
| **Tables** — delete table | ✅ | `DELETE .../tables/{id}` with confirm. |
| **Tables** — open table (row click) | ✅ | Drills into Level 3. |
| **Tables** — search box | ✅ (React better) | React `localFilter` actually filters. Flutter's Tables search box is a decorative `TextEditingController()` with **no `onChanged`** — a no-op. |
| **SQL** — Monaco editor | ⚠️ | React `@monaco-editor/react` hardcodes `theme="vs-dark"`; Flutter switches vs/vsDark with the console light/dark theme. Not theme-aware in React. |
| **SQL** — Run query | ✅ | `POST /databases/{db}/sql {statement, writeAllowed}`. |
| **SQL** — Cmd/Ctrl+Enter to run | ✅ | Bound in `onMount`. |
| **SQL** — Copy SQL | ✅ | `navigator.clipboard`. |
| **SQL** — Allow-writes toggle + confirm dialog | ✅ | Same guard + "forced read-only / INSERT,UPDATE,DELETE allowed" copy. |
| **SQL** — Results grid (columns × rows) | ✅ | `ResultsGrid`, monospace, horizontal scroll. |
| **SQL** — Empty-result summary ("Statement completed", rows affected + ms) | ✅ | `SummaryView`. |
| **SQL** — Affected-count / exec-time metric pills | ✅ | `MetricPill` rows / ms. |
| **SQL** — Error view | ✅ | Styled error block (React uses `friendlyError`, Flutter raw). |
| **SQL** — Export JSON / CSV | ✅ | `downloadResults` blob download, same filename scheme. |
| **SQL** — Schema browser (expand tables, list columns) | ✅ | `SchemaBrowser`/`SchemaTableRow` from `information_schema` query. |
| **SQL** — Insert table / Insert column | ✅ | `onInsert` appends identifier. |
| **SQL** — Open table from schema browser | ✅ | `onOpenTable`. |
| **SQL** — Recent queries history (session, max 12) | ✅ | Same cap + click-to-restore. |
| **SQL** — inline autocomplete suggestions | ✅ | Both rely on Monaco `quickSuggestions`. (Flutter also computes a custom `_sqlSuggestions` list but never renders it — dead code; no visible gap.) |
| **Usage** tab | ✅ | Both placeholder ("Usage coming soon" / `_PlaceholderTab`). |
| **Settings** — Database ID info | ✅ | `DatabaseSettings`. |
| **Settings** — Delete database (danger card) | ✅ | `DELETE /databases/{id}` with confirm. |

---

## Level 3 — Table detail (Rows / Columns / Indexes / Relationships / Settings)

| Sub-feature | Status | Notes |
|---|---|---|
| Back header + 5-tab bar | ✅ | `TableDetail`. |
| **Rows** — grid with dynamic columns from schema | ✅ | `$id` key icon + per-column type icons. |
| **Rows** — search | ✅ (React better) | React `localFilter`; Flutter search box is a no-op `TextEditingController()`. |
| **Rows** — Create row (per-column fields, or JSON when no columns) | ✅ | `POST .../rows {data}`. |
| **Rows** — Edit row | ✅ | `PATCH .../rows/{id} {data}` via row dropdown. |
| **Rows** — Delete row (confirm) | ✅ | `DELETE .../rows/{id}`. |
| **Rows** — empty states (no columns / no rows) | ✅ | Both variants present. |
| **Rows** — Import CSV | ✅ (both stubbed) | Neither implements it. React button is `disabled`; Flutter button is `onTap: () {}` (no-op). Not a regression, but the feature is absent in both. |
| **Columns** — list (name/required/type/default/permissions) | ✅ | `ColumnsPanel` table. |
| **Columns** — Create column side panel | ✅ | Slide-in panel, matches Flutter `_CreateColumnPanel`. |
| **Columns** — type picker (all 10 types) | ✅ | string, integer, float, boolean, datetime, email, url, enum, point, relationship. |
| **Columns** — string `size` field | ✅ | Shown for `string`. |
| **Columns** — enum `elements` field | ✅ | Comma-split. |
| **Columns** — default value | ✅ | Optional. |
| **Columns** — required / array toggles | ✅ | Both. |
| **Columns** — validation (min/max length, min/max value, regex pattern, custom message) | ✅ | Same per-type gating (`showValidation` excludes boolean/datetime/point/relationship). |
| **Columns** — `POST .../columns/{type}` payload shape | ✅ | key/required/array/default/size/elements/validation identical. |
| **Columns** — Delete column | ✅ | `DELETE .../columns/{key}` with confirm. |
| **Columns** — Column permissions dialog (allow read/write) | ✅ | `POST .../columns/{key}/permissions`. |
| **Indexes** — list (key/type/columns/orders) | ✅ | `IndexesPanel`. |
| **Indexes** — Create (key, columns, type chips btree/unique/fulltext, orders=ASC) | ✅ | Same payload. |
| **Indexes** — Delete | ✅ | With confirm. |
| **Relationships** — list (key/type/related table/on delete) | ✅ | `RelationshipsPanel`. |
| **Relationships** — Create (key, related table ID, type chips 1:1/1:N/N:1/N:N) | ✅ | `POST .../columns/relationship`. |
| **Relationships** — Delete | ✅ | With confirm. |
| **Settings** — Table ID info | ✅ | `TableSettings`. |
| **Settings** — Delete table (danger card) | ✅ | `DELETE .../tables/{id}` with confirm. |
| **Settings** — Enabled toggle | ❌ | Flutter has an Enabled switch → `PUT {enabled}`. Missing in React. |
| **Settings** — Name edit field | ❌ | Flutter has an inline name field → `PUT {name}`. Missing in React. |
| **Settings** — Permissions (list role+action, Add-permission dialog, remove) | ❌ | Flutter fetches `.../permissions`, lists role/action rows, adds via dialog (role text + action chips read/create/update/delete), removes per-row → `POST .../permissions`. Entirely missing in React. |
| **Settings** — Row security toggle + explanation | ❌ | Flutter has a Row-security switch → `PUT {rowSecurity}` with the RLS explainer text. Missing in React. |
| **Settings** — reads table details (`GET .../tables/{id}`) & permissions | ❌ | React never fetches table details or the permissions endpoint. |

---

### Gaps (actionable)

**❌ Missing (all in the table-level Settings tab — `TableDetail.tsx` `TableSettings`):**
- Add the **Enabled** toggle → `PUT /databases/{db}/tables/{id} {enabled}`.
- Add the inline **Name** edit field → `PUT /databases/{db}/tables/{id} {name}` (and invalidate the tables list).
- Add the **Permissions** panel: fetch `GET .../tables/{id}/permissions`, render role/action rows, an "Add permission" dialog (role input + action chips read/create/update/delete), and per-row remove — all persisting via `POST .../tables/{id}/permissions`.
- Add the **Row security** toggle → `PUT /databases/{db}/tables/{id} {rowSecurity}` with the RLS explainer copy.
- The above requires wiring a `GET /databases/{db}/tables/{id}` (table details) query that React currently never issues.

**⚠️ Fix / align:**
- Remove the **duplicate header + tab bar** on the Level-1 list: `DatabaseList` re-renders `<h1>Databases</h1>` + subtitle + `PageTabs` that `DatabasesPage` already renders. Collapse to one.
- Level-1 **Usage** tab (from `DatabasesPage`) shows only a bare "Usage coming soon" box; it lacks the 3 stat cards (Total databases/tables/rows) Flutter shows. The stat-card version lives in `DatabaseList`'s orphaned inner Usage tab — consolidate so the top-level Usage tab shows the stat cards.
- Make the SQL Monaco editor **theme-aware** (`vs` vs `vs-dark`) instead of hardcoded `vs-dark`, to match Flutter's light/dark switch.

**Notes (parity holds, flagged for awareness):**
- **CSV import** for rows is unimplemented in both (React `disabled`, Flutter no-op). If it's on the roadmap, neither side has it.
- Flutter's Tables and Rows search boxes are non-functional stubs; React's actually filter — React is ahead here, no action needed.
