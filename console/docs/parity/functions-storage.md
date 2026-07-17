# Parity audit — Functions & Storage (Flutter → React)

Read-only comparison of the Flutter console pages against their React ports.

- Flutter: `console/lib/features/functions/functions_page.dart`, `console/lib/features/storage/storage_page.dart`
- React: `console-react/src/features/functions/*.tsx`, `console-react/src/features/storage/*.tsx`

Legend: ✅ full parity · ⚠️ partial / cosmetic drift · ❌ missing

---

## Functions

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| List — DataTable (Name/Runtime/Status/Updated) | yes | yes | ✅ | Same columns/flex. React renders runtime **label** and a `StatusChip` in cells; Flutter shows the raw runtime **id** and a custom badge. React is arguably nicer. |
| List — grid card + view toggle (`persistKey`) | `functions_view` | `functions_view` | ✅ | Icon tile, name, runtime label, status + date footer. |
| List — search / pagination / per-page | yes | yes | ✅ | Both hit `/functions` with search + limit/offset. |
| List — filters (Runtime, Status) | yes | yes | ✅ | Same option sets. |
| List — Usage tab (3 stat cards + "charts coming soon") | yes | yes | ✅ | Placeholder in both. |
| Create dialog (name, runtime, entrypoint, timeout, source toggle, inline/repo/branch) | yes | yes | ✅ | Identical payload. Create dialog uses a plain textarea for inline code in **both**. |
| Status badge colors | custom `_StatusBadge` | shared `StatusChip` | ⚠️ | Color mapping drifts: Flutter `building`→orange, `inactive`→gray; React `building`→info/blue, `inactive`→**danger/red**. Cosmetic only. |
| Detail — back breadcrumb + Execute button | yes | yes | ✅ | |
| Detail — Executions tab (list) | yes | yes | ✅ | Columns Execution ID/Status/Duration/Triggered; relative time. |
| Detail — Executions **row expand → output/errors** | yes (expandable monospace stdout/stderr) | **no** | ❌ | Flutter `_ExecutionRow` expands to show `output`/`errors`. React uses `DataTable` with no expansion, so execution logs are not viewable. |
| Detail — Variables tab (count, add dialog, delete) | yes | yes | ✅ | Both PUT the whole fn with merged `envVars`. |
| Detail — Settings: General (name/runtime/entrypoint/timeout/cron) | yes | yes | ✅ | Cron hint text matches. |
| Detail — Settings: Source editor (inline) | plain multiline `TextField` (monospace) | **Monaco** editor (per-runtime language) | ✅+ | React **exceeds** Flutter — real Monaco with `runtimeById().language`. |
| Detail — Settings: Source (GitHub repo/branch) | yes | yes | ✅ | |
| Detail — Settings: Save + Danger-zone delete (confirm) | yes | yes | ✅ | |
| Runtime picker — all 12 runtimes | yes | yes | ✅ | ids/labels/order identical (`runtimes.ts`). |

### Gaps (actionable)

1. **❌ Execution output/error logs not viewable (React).** Flutter’s `_ExecutionRow` is an expandable row that reveals `output` (stdout) and `errors` (stderr) in a monospace panel. The React `ExecutionsTab` renders a flat `DataTable` with no expand affordance — users cannot inspect an execution’s result. Add a row-expand or a detail drawer that surfaces `output`/`errors`.
2. **⚠️ Status color mapping drift.** `StatusChip` classifies `inactive` as `danger` (red) and `building` as `info` (blue); Flutter uses gray for `inactive` and orange for `building`. Consider a function-scoped variant override if the semantic colors matter.

_(Note: React’s Monaco source editor is an improvement over Flutter’s plain textarea, not a gap.)_

---

## Storage

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| Buckets list — DataTable (Bucket ID/Name/Created/Updated) | yes | yes | ✅ | |
| Buckets list — grid card + view toggle | no `persistKey` | `persistKey="storage-buckets"` | ✅ | React adds view persistence; folder icon + `IdText` chip in both. |
| Buckets list — search / pagination / per-page | yes | yes | ✅ | |
| Buckets — top-level Usage tab (3 cards + placeholder) | yes | yes | ✅ | |
| Create bucket dialog | yes | yes | ✅ | Same payload (`bucketId: 'unique()'`, empty perms/exts). |
| Delete bucket (from list, confirm) | yes | yes | ✅ | |
| Bucket detail — back + name + id header | yes | yes | ✅ | |
| Bucket detail — tabs (Files/Usage/Settings) | yes | yes | ✅ | |
| Files tab — table (icon/name/type/size/created) + mime icon tint | custom `_FilesTable` | shared `DataTable` | ✅ | React drops the custom header styling but keeps mime icon + color and gains wired search/pagination. |
| Files tab — search | present but **not wired** (local `TextField`, no effect) | wired to `useResourceList` | ✅+ | React **exceeds** — Flutter’s file search box does nothing. |
| **File upload** | **no-op** (`onPressed: () {}` on both button and empty-state) | **works** (hidden input + multipart POST, `Uploading…` state, error banner) | ✅+ | React **exceeds**; Flutter upload is a stub. |
| **File download** | **no-op** (`onPressed: () {}`) | **works** (`<a download>` to view URL) | ✅+ | React **exceeds**; Flutter download is a stub. |
| File delete (row menu / detail, confirm) | yes | yes | ✅ | |
| File detail — image preview (`<img>` / `Image.network`) | yes | yes | ✅ | Non-image → file icon in both. |
| File detail — metadata rows (filename/mime/size/created/updated) | yes | yes | ✅ | |
| File detail — File URL + copy to clipboard | yes | yes (+ Check confirmation icon) | ✅ | React adds transient "copied" feedback. |
| File detail — Permissions card (static informational) | yes | yes | ✅ | Static in both. |
| **Image transforms (resize / format / quality)** | **none** | **none** | ⚠️ | Listed sub-feature, but neither UI exposes transform controls (backend supports them). Absent in both → port is at parity but feature unimplemented. |
| Bucket settings — status/Enabled toggle + Update | yes | yes | ✅ | Toggle is display-only; Update button PUTs `{enabled}`. |
| Bucket settings — Name field + Update | yes | yes | ✅ | |
| Bucket settings — Permissions table (roles × CRUD, Add role) | static/disabled, Add-role no-op | static/disabled, Add-role no-op | ✅ | Non-functional in both (matches). |
| Bucket settings — File security toggle + descriptive text | yes | yes | ✅ | Conditional helper text matches. |
| Bucket settings — Security (Encryption, Antivirus) toggles + Update | yes | yes | ✅ | Update PUTs `{encryption}` only in both. |
| Bucket settings — Compression dropdown (none/gzip/zstd) | display-only, no persist | display-only, disabled, no persist | ✅ | Read-only in both. |
| Bucket settings — Maximum file size (MB) | read-only text | read-only text | ✅ | No editor in either. |
| Bucket settings — Allowed extensions (chips / "all allowed") | read-only | read-only | ✅ | No add/remove in either. |
| Bucket settings — Delete bucket card (confirm) | yes | yes | ✅ | |
| Bucket detail — Usage tab (3 cards) | yes | yes | ✅ | Placeholder in both. |

### Gaps (actionable)

1. **⚠️ Image transforms unimplemented (both platforms).** The storage feature description calls for resize / format-conversion preview controls; neither the Flutter page nor the React port renders any transform UI (the file detail shows a plain preview only). Not a port regression, but the sub-feature is missing end-to-end. If desired, add transform query params to `fileViewUrl` + preview controls in `FileDetailView`.
2. **⚠️ Settings write surface is partial (both platforms, faithful port).** Permissions (roles × CRUD), Compression, Maximum file size, and Allowed extensions are display-only in both Flutter and React — only Enabled / Name / File security / Encryption actually persist. React faithfully mirrors this, so it is **not** a port gap, but these controls are inert versus what the settings imply.

_(Note: React’s working file upload, working download, wired file search, and copy-confirmation are improvements over Flutter, not gaps.)_

---

## Summary

- **Functions:** 1 real port gap (❌ execution output/error logs not viewable in React) + 1 cosmetic (⚠️ status colors). React additionally **exceeds** Flutter with a Monaco source editor.
- **Storage:** No React port regressions. React **exceeds** Flutter on file upload, download, and wired search (all no-ops/stubs in Flutter). Two shared-limitation ⚠️s (image transforms; inert settings controls) exist in **both** platforms, so they are feature gaps rather than parity gaps.
