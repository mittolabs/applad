# Workflows — Flutter → React parity audit

Read-only comparison of the Flutter console Workflows feature against its React port.

- **Flutter source:** `console/lib/features/workflows/workflows_page.dart` (4071 lines, single file)
- **React source:** `console-react/src/features/workflows/` (`WorkflowsPage`, `WorkflowList`, `WorkflowBuilder`, `WorkflowNode`, `NodePalette`, `NodeConfigPanel`, `ExecutionsPanel`, `nodeDefs.ts`) — DAG canvas built on `@xyflow/react`.

Legend: ✅ full parity · ⚠️ partial / behaves differently · ❌ missing in React

---

## Summary

The React port is a **faithful, near-complete** reimplementation. The entire node catalog, config panel, trigger settings, executions history, save/execute/test flows, and undo/redo are ported 1:1. All gaps are **canvas power-user affordances** that the Flutter hand-rolled `CustomPaint` canvas had and that were not rebuilt on top of React Flow: **node duplication, right-click context menu, sticky notes, an inline live-logs panel, and several keyboard shortcuts.**

- Sub-features audited: **28**
- ✅ Full parity: **19**
- ⚠️ Partial / cosmetic difference: **4**
- ❌ Missing: **5**
- **Node types:** 60/60 present in React (exact port) — **zero missing node types.**

---

## Workflow list & CRUD

| Sub-feature | Flutter | React | Status |
|---|---|---|---|
| List (table + grid card) | `AppDataTable`, `_Card` | `DataTable`, `WorkflowGridCard` | ✅ |
| Columns: name/trigger/status/nodes/updated | ✅ | ✅ | ✅ |
| Status chip + trigger icon | ✅ | ✅ (`StatusChip`, `triggerIcon`) | ✅ |
| Relative "updated" time | `_fmtDate` | `relativeTime` | ✅ |
| Create workflow (name dialog) | `_create` | `CreateWorkflowDialog` | ✅ |
| Templates (Welcome Email, Daily Digest, Webhook→DB, Error Alert) | 4 templates | 4 templates (identical nodes/edges) | ✅ |
| Delete row | ✅ | ✅ | ✅ |
| Search + pagination + per-page | ✅ | ✅ (`useResourceList`) | ✅ |

## DAG builder — canvas

| Sub-feature | Flutter | React | Status |
|---|---|---|---|
| Pan / zoom | manual `Listener` + `CustomPaint` | React Flow `Controls` + wheel/drag | ✅ |
| Fit to view | `_fitToView` button | `fitView` prop + Controls button | ✅ |
| Snap to grid (20px) | `_gridSnap` | `snapToGrid` / `snapGrid={[20,20]}` | ✅ |
| Add node (toolbar / palette) | `_AddNodeIntent`, palette | "Add node" button → `NodePalette` | ✅ |
| Drag-to-empty opens palette | `_onPointerUp` dragConn | `onConnectEnd` → palette at drop pos | ✅ |
| Connect nodes (dedupe, no self-loop) | `_addEdge` | `onConnect` (dedupe + self-loop guard) | ✅ |
| Delete node (trigger protected) | `_deleteSelected` (skips trigger) | `onBeforeDelete` / `deleteNode` (skips trigger) | ✅ |
| Toggle node disable | `_toggleDisable` | `toggleDisable` | ✅ |
| Multi-output/-input handles + labels (IF/Switch/Merge/Try-Catch) | `_CanvasPainter` | `WorkflowNode` handles + labels | ✅ |
| Minimap | `_minimap` (toggleable) | React Flow `MiniMap` (always on) | ⚠️ no toggle |
| Zoom % readout / reset-to-100% | `_zoomControls` | React Flow Controls (no % label) | ⚠️ cosmetic |
| Multi-select (shift-click + rubber-band) | `_Mode.selectRect`, shift toggle | React Flow default box-selection | ⚠️ works via RF default, not custom-tuned |
| **Duplicate node(s) + internal edges** | `_duplicateSelected` (⌘D / ctx menu) | — | ❌ |
| **Right-click context menu** | `_showContextMenu` (open/duplicate/disable/delete; add/select-all/fit) | — | ❌ |
| **Sticky notes** | `_addStickyNote` + painter | — | ❌ |

## Node palette

| Sub-feature | Flutter | React | Status |
|---|---|---|---|
| Top-level categories (7) | `_categories` | `CATEGORIES` (identical) | ✅ |
| Drill-down sub-sections | `_palContent` groupings | `PALETTE_SECTIONS` (identical) | ✅ |
| Search across label/desc/type/category | `_palSearch` | `SearchResults` | ✅ |
| "Add another trigger" row | ✅ | ✅ | ✅ |
| Node rows (icon/label/desc) | `_palNodeItem` | `NodeRow` | ✅ |

### Node catalog — 60/60 present (exact port)

`nodeDefs.ts` is a line-for-line port of Flutter `_allNodeDefs` (same types, labels, colors, categories, input/output counts, output labels). Categories and counts:

| Category | Count | Types |
|---|---|---|
| Flow | 8 | if_condition, switch, merge, loop, wait, no_operation, execute_sub_workflow, filter |
| Core | 6 | http_request, code, javascript, send_email, set_variable, delay |
| Data transformation | 14 | edit_fields, aggregate, summarize, limit, split_out, remove_duplicates, date_time, convert_to_json, extract_from_json, html_parse, crypto, sort, rename_keys, compare_datasets |
| Error handling (in Flow) | 2 | try_catch, stop_and_error |
| AI | 3 | ai_transform, ai_agent, ai_summarize |
| Integrations | 14 | slack, discord, telegram, github, google_sheets, notion, stripe, twilio_sms, postgres_query, mysql_query, redis_command, s3, sendgrid, jira |
| Applad | 5 | applad_auth, applad_database, applad_storage, applad_functions, applad_messaging |
| Triggers | 7 | trigger_manual, trigger_webhook, trigger_schedule, trigger_database, trigger_auth, trigger_storage, trigger_messaging |
| (base) | 1 | trigger |

**No node types are missing in React.** `TYPE_FIELDS` (per-type config fields) and `PALETTE_SECTIONS` are also identical.

## Node config panel

| Sub-feature | Flutter | React | Status |
|---|---|---|---|
| Header: icon, label, disable/delete/test/close | `_configPanel` | `NodeConfigPanel` header | ✅ |
| Settings / Input / Output tabs | `_cfgTabBtn` | tab bar | ✅ |
| Label field | ✅ | ✅ | ✅ |
| Per-type fields (text / multi-line) | `_tf` (lines param) | `ConfigField` (`TextInput`/`TextArea`) | ✅ |
| Expression suggestion chips (upstream `{{...}}`) | `_upstreamSuggestions` + chips | `upstreamSuggestions` + chips | ✅ |
| On-error select (stop / continue / error_output) | `_buildErrorBranchRow` | `OnErrorSelect` | ✅ |
| Connections list + delete edge | `_connList` / `_connChip` | `ConnectionsList` / `ConnChip` | ✅ |
| Input/Output data preview (JSON) | `_dataPreview` | `DataPreview` | ✅ |
| Pin / unpin output | `_pinnedData` toggle | `onTogglePin` / PINNED badge | ✅ |

### Trigger config

| Sub-feature | Flutter | React | Status |
|---|---|---|---|
| Trigger type chips (Manual / Webhook / Schedule) | `_trigTypeChip` | `TrigChip` | ✅ |
| Webhook URL + copy | `_webhookSettings` | `TriggerSettings` webhook block | ✅ |
| Signing secret + Regenerate (`/webhook-secret`) | `_regenerateSecret` | `onRegenerateSecret` | ✅ |
| HMAC verification hint | ✅ | ✅ | ✅ |
| Cron expression + human description | `_describeCron` | `describeCron` (identical rules) | ✅ |

## Run / execute / history / test

| Sub-feature | Flutter | React | Status |
|---|---|---|---|
| Execute workflow (`/execute`) + per-node status | `_execute` | `execute` (also caches input/output → bonus) | ✅ |
| Executions history dialog | `_showExecs` / `_executionsTab` | `ExecutionsPanel` | ✅ |
| Expandable per-step logs (status/label/type/duration) | ✅ | ✅ | ✅ |
| Load past execution onto canvas | `_loadExecution` | `loadExecution` | ✅ |
| Single-node test (`/nodes/{id}/test`) | `_testNode` | `testNode` | ✅ |
| **Inline live-logs panel (bottom, toggleable)** | `_logsPanel` + Logs toggle | — (only History dialog) | ❌ |
| Edge "N items" count badges | `execCounts` in painter (vestigial — never populated) | — | ⚠️ dead code in Flutter; safe to skip |

## Save / status / undo-redo / shortcuts

| Sub-feature | Flutter | React | Status |
|---|---|---|---|
| Save (`PUT /workflows/{id}`) | `_save` | `save` | ✅ |
| Trigger → `set_variable{_trigger}` save transform | ✅ | `toSaveNodes` (+ reverse in `toRfNodes`) | ✅ |
| Capture `webhookSecret` from save response | `widget.workflow[...]` | `setWebhookSecret` | ✅ |
| Status select (draft / active / paused) | `_statusChip` | `StatusSelect` | ✅ |
| Dirty flag (`Save*` / `Saved`) | `_dirty` | `dirty` | ✅ |
| Undo / redo (toolbar buttons) | `_UndoStack` | `undoStack`/`redoStack` refs | ✅ |
| Undo/redo shortcuts (⌘Z / ⌘⇧Z) | ✅ | ✅ | ✅ |
| Save shortcut (⌘S) | ✅ | ✅ | ✅ |
| Delete / Backspace | ✅ | ✅ (`deleteKeyCode`) | ✅ |
| **Duplicate (⌘D)** | ✅ | — | ❌ |
| **Select-all (⌘A)** | ✅ | — | ❌ |
| **Toggle-disable (`d`)** | ✅ | — | ❌ |
| **Add-node (Tab)** | ✅ | — | ❌ |
| **Clear/close (Esc)** | ✅ | — | ❌ |
| Copy/Paste (⌘C / ⌘V) | present but no-op | — | ✅ (no behavior lost) |

---

## Gaps (actionable)

1. **Node duplication — missing entirely.** Flutter offers `_duplicateSelected` (via ⌘D and the context menu) which clones selected non-trigger nodes plus their internal edges, offset by (40,40). React has no duplicate path at all. *Add a `duplicateNode`/`duplicateSelection` action wired to ⌘D and the config panel.*

2. **Right-click context menu — missing.** Flutter's `_showContextMenu` gives: on a node → Open settings / Duplicate / Toggle disable / Delete; on empty canvas → Add node / Select all / Fit to view. React relies on React Flow defaults (no menu). *Add an `onNodeContextMenu` / `onPaneContextMenu` menu.*

3. **Sticky notes — missing.** Flutter can add draggable sticky notes to the canvas (`_addStickyNote`, painted behind nodes). No equivalent in React. Note these are **canvas-local only** in Flutter (not persisted in the save payload), so it is a UI-only affordance.

4. **Inline live-logs panel — missing.** Flutter has a toggleable bottom "Logs" panel (`_logsPanel`) that lists each node's live execution status after Execute. React only surfaces logs through the History dialog (`ExecutionsPanel`). *Consider a collapsible logs strip, or accept the History dialog as the substitute.*

5. **Keyboard shortcuts — partial.** React implements only ⌘S, ⌘Z, ⌘⇧Z, Delete/Backspace. Missing vs Flutter: **⌘D duplicate, ⌘A select-all, `d` toggle-disable, Tab open-palette, Esc clear-selection/close-panels.** (⌘C/⌘V are intentional no-ops in Flutter — no parity loss.)

6. **Cosmetic / minor (⚠️, low priority):**
   - Minimap is always visible in React; Flutter lets the user toggle it.
   - Zoom controls: Flutter shows a live zoom % with click-to-reset-100%; React uses React Flow's default Controls (no % readout).
   - Multi-select works via React Flow's default box-selection but is not explicitly tuned (`selectionOnDrag`/`panOnDrag`) to match Flutter's shift-drag rubber-band + shift-click toggle exactly.
   - Double-click empty canvas to open the palette (Flutter) is not wired in React (Add-node button + drag-to-empty cover the use case).
   - Edge "N items" count badges exist in the Flutter painter but are **dead code** (`_execCounts` is never populated) — no action needed.

**No node types are present in Flutter but missing in React.** The node catalog, per-type config fields, palette sections, and trigger settings are at full parity.
