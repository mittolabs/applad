# Parity audit — Platforms & Deploy features (Flutter → React)

Read-only audit comparing the Flutter console (`console/lib/features/**`) to the React
port (`console-react/src/features/**`). Scope: **platforms, sites, containers, mobile,
desktop**, plus the shared **DeployCreateEntry** wizard.

Legend: ✅ full parity · ⚠️ present but differs / partial · ❌ missing in React

Source files:
- Flutter: `console/lib/features/{platforms,sites,containers,mobile,desktop}/*.dart`,
  `console/lib/features/deploy/deploy_page.dart`, `console/lib/core/widgets/deploy_create_entry.dart`
- React: `console-react/src/features/{platforms,sites,containers,mobile,desktop,deploy-shared}/*.tsx`,
  `console-react/src/components/deploy-create-entry.tsx`

---

## 1. Platforms

Flutter: `platforms/platforms_page.dart` · React: `platforms/PlatformsPage.tsx`,
`PlatformDetail.tsx`, `PlatformDeploymentTab.tsx`, `ShellSnippets.tsx`, `platform-utils.ts`

| Sub-feature | Flutter | React | Status |
|---|---|---|---|
| List: DataTable (name/type/identifier/created) | ✓ | ✓ | ✅ |
| List: grid-card view | ✓ | ✓ | ✅ |
| List: search (name/type/identifier) | ✓ | ✓ | ✅ |
| List: Type filter | ✓ | ✓ | ✅ |
| List: pagination + per-page | ✓ | ✓ | ✅ |
| List: delete row | ✓ | ✓ | ✅ |
| Platform types (web/iOS/Android/Desktop/Server) | 5 types | 5 types | ✅ |
| Type icons + identity label/hint per type | ✓ | ✓ | ✅ |
| Add-platform dialog (type chips + name + identifier) | ✓ | ✓ | ✅ |
| Detail: Overview tab (info cards + Platform ID) | ✓ | ✓ | ✅ |
| Detail: Overview SDK-init snippet | ✗ | ✓ | ⚠️ React-only enhancement |
| Detail: Overview CLI-install snippet (ShellSnippets) | ✗ | ✓ | ⚠️ React-only enhancement |
| Detail: Deployment tab — not-connected empty state | ✓ | ✓ | ✅ |
| Detail: Deployment — connect-deployment dialog (target picker) | ✓ | ✓ | ✅ |
| Detail: Deployment — target-not-found state + remove | ✓ | ✓ | ✅ |
| Detail: Deployment — releases table + stats header | ✓ | ✓ | ✅ |
| Detail: Deployment — Disconnect button in releases header | ✗ (param passed, never rendered) | ✓ | ⚠️ React fixed a Flutter dead-end |
| Create deployment menu: Git / CLI / Manual | ✓ | ✓ | ✅ |
| Git deploy dialog (pipeline/branch + activate) | ✓ | ✓ | ✅ |
| CLI deploy dialog (Unix/CMD/PowerShell tabs) | ✓ | ✓ | ✅ |
| Manual deploy dialog (upload placeholder, non-functional) | ✓ (+ dead "Create" btn) | ✓ (no btn) | ✅ both stubs |
| Settings tab: name + identifier + save + danger delete | ✓ | ✓ | ✅ |
| iOS type badge colour (secondary accent) | uniform accent | iOS purple | ⚠️ cosmetic React enhancement |
| Settings identifier field placeholder | `_identityHint(type)` (dynamic) | hardcoded `com.example.myapp` | ⚠️ minor |

**Verdict:** ✅ Full parity, with React actually ahead (SDK/CLI snippets, working Disconnect).

### Gaps (actionable)
- `PlatformDetail.tsx` `SettingsTab`: the identifier `TextField` placeholder is hardcoded
  `"com.example.myapp"`; Flutter uses `_identityHint(type)` so a web platform shows
  `myapp.com` and server shows `192.168.1.1`. Swap to `identityHint(type)`.
- No functional gaps against Flutter. (Manual upload is a non-functional placeholder in
  both; Git-provider OAuth is a TODO in both — see wizard section.)

---

## 2. Sites

Flutter: `sites/sites_page.dart` (2954 lines) · React: `sites/SitesPage.tsx`,
`SiteDetail.tsx`, `CreateSiteDialog.tsx` + `deploy-shared/*`

| Sub-feature | Flutter | React | Status |
|---|---|---|---|
| List tabs: Sites / Usage | ✓ | ✓ | ✅ |
| List: DataTable (name/framework/status/updated) | ✓ | ✓ | ✅ |
| List: search + pagination + per-page + delete | ✓ | ✓ | ✅ |
| List Usage tab (aggregate stat cards, `--`) | ✓ | ✓ | ✅ |
| Framework metadata table (8 frameworks) | ✓ | ✓ (`frameworks.ts`) | ✅ |
| Create: 3-option entry wizard | ✓ | ✓ | ✅ (see §6) |
| Create: multi-step form (Configuration/Source/Build) | ✓ | ✓ | ✅ |
| Create: step indicator + framework cards + source chips | ✓ | ✓ | ✅ |
| Create: auto-fill build config from framework | ✓ | ✓ | ✅ |
| Detail tabs: Overview/Deployments/Logs/Domains/Usage/Settings | 6 | 6 | ✅ |
| Overview: preview placeholder + info rows | ✓ | ✓ | ✅ |
| Overview: Visit + Instant rollback buttons | ✓ | ✓ | ✅ |
| Overview: rollback confirmation dialog | ✓ (AlertDialog confirm) | ✗ (rolls back immediately) | ⚠️ React dropped confirm |
| Overview: rollback status match | `status == 'success'` only | `success \|\| ready \|\| active` | ⚠️ behaviour differs (React broader) |
| Overview: summary cards (Domains/Deployments) → tab jump | ✓ | ✓ | ✅ |
| Deployments: metrics row (6 badges) | ✓ | ✓ | ✅ |
| Deployments: Create deployment trigger | ✓ | ✓ | ✅ |
| Deployments: Refresh button | ✗ | ✓ | ⚠️ React-only enhancement |
| Deployments: releases table | ✓ | ✓ | ✅ |
| Logs: header + refresh + table (Log/Path/Method/Status/Duration/Created) | ✓ | ✓ | ✅ |
| Logs: method badge + HTTP status colour badge | ✓ | ✓ | ✅ |
| Domains: list + Add domain | ✓ (DataTable) | ✓ (card list) | ✅ presentation differs, cosmetic |
| Domains: add dialog (active_deployment/git_branch/redirect) | ✓ | ✓ | ✅ |
| Usage: time range 24h/7d/30d + 3 stat cards | ✓ | ✓ | ✅ |
| Settings: General / Build config / Env vars / Save / Danger | ✓ | ✓ | ✅ |
| Settings: env-var add/remove key–value rows | ✓ | ✓ | ✅ |

**Verdict:** ✅ Full structural parity; one behavioural regression (rollback no longer confirms).

### Gaps (actionable)
- `SiteDetail.tsx` `OverviewTab.rollback`: Flutter shows a confirm dialog
  ("Roll back to deployment X…?") before POSTing `/deploy/releases/{id}/rollback`; React
  fires immediately on click. Add a `ConfirmDialog` to match — instant rollback is
  destructive.
- `SiteDetail.tsx` rollback candidate selection differs from Flutter (`success` only vs
  `success|ready|active`). Confirm which status set is authoritative; they can pick
  different releases.

---

## 3. Containers

Flutter: `containers/containers_page.dart` · React: `containers/ContainersPage.tsx`,
`ContainerDetail.tsx` + `deploy-shared/*`

| Sub-feature | Flutter | React | Status |
|---|---|---|---|
| List presentation | custom `ListView` cards | `TargetList` DataTable | ⚠️ different surface |
| List columns | icon/name/registry + tag badge | name/registry/tag/**status**/**updated** | ⚠️ React richer (adds status+updated) |
| List: per-row delete | ✗ (only in Settings) | ✓ | ⚠️ React-only |
| List: search + pagination | ✓ (SearchListHeader) | ✓ | ✅ |
| Create: 3-option entry wizard | ✓ | ✓ | ✅ |
| Create: simple dialog (name/registry/Dockerfile) | ✓ | ✓ | ✅ |
| Detail tabs: Overview/Images/Releases/Settings | 4 | 4 | ✅ |
| Overview: info cards (Registry/Dockerfile/Tag/Runtime/Type) | ✓ (5) | ✓ (5) | ✅ |
| Images tab (repo:tag, platform, size MB) | ✓ | ✓ | ✅ |
| Releases tab | custom list (dot/id/status/durationMs) | `DeploymentsPanel` table (metrics off, trigger off) | ⚠️ different fields/surface |
| Settings: read-only rows + danger delete | ✓ | ✓ | ✅ |

**Verdict:** ✅ Feature-complete; React uses a shared table surface (arguably richer) where
Flutter used bespoke lists. No missing capability.

### Gaps (actionable)
- Releases tab field mismatch: Flutter renders `durationMs` (ms) per release; React's shared
  `DeploymentsPanel` renders `buildDuration` (seconds) + `totalSize`. If the container
  release payload uses `durationMs`/`triggerType` (as Flutter reads) rather than
  `buildDuration`/`source`, the React columns will show `--`. Verify the container release
  schema against `DeploymentsPanel` field names.
- Cosmetic only: Flutter container list is a card list without status/updated; React shows a
  full DataTable. No action needed unless design parity with Flutter is required.

---

## 4. Mobile

Flutter: `mobile/mobile_page.dart` · React: `mobile/MobilePage.tsx` + `deploy-shared/shared.tsx`

| Sub-feature | Flutter | React | Status |
|---|---|---|---|
| List: DataTable (name/type/status/updated) | ✓ | ✓ | ✅ |
| List: buildType row icon (Tablet for ipa, else Smartphone) | ✓ | ✓ | ✅ |
| List: search + pagination + delete | ✓ | ✓ | ✅ |
| Page subtitle/description | ✗ | ✓ | ⚠️ React-only |
| Create: 3-option entry wizard | ✓ | ✓ | ✅ |
| Create: dialog with Android/iOS platform chips | ✓ | ✓ | ✅ |
| Detail tabs: Overview/Builds/Signing/Settings | 4 | 4 | ✅ |
| Overview: Platform/Build Type/Runtime cards + distribution note | ✓ | ✓ | ✅ |
| Builds tab (release list: dot/id/trigger•duration/status) | ✓ | ✓ (shared `BuildsTab`) | ✅ |
| Signing: Android Keystore + iOS Provisioning upload cards (stub) | ✓ | ✓ (`SigningUploadCard`) | ✅ both non-functional |
| Settings: Name/Platform/Build Type + danger delete | ✓ | ✓ | ✅ |

**Verdict:** ✅ Full parity.

### Gaps (actionable)
- None. (Upload buttons in Signing are non-functional placeholders in both.)

---

## 5. Desktop

Flutter: `desktop/desktop_page.dart` · React: `desktop/DesktopPage.tsx` + `deploy-shared/shared.tsx`

| Sub-feature | Flutter | React | Status |
|---|---|---|---|
| List: DataTable (name/platforms/status/updated) | ✓ | ✓ | ✅ |
| List: `platformsLabel` from mac/win/linux flags | ✓ | ✓ | ✅ |
| List: search + pagination + delete | ✓ | ✓ | ✅ |
| Create: 3-option entry wizard | ✓ | ✓ | ✅ |
| Create: platform chips (macOS/Win/Linux/Cross) + framework (Flutter/Electron/Tauri) | ✓ | ✓ | ✅ |
| Detail tabs: Overview/Builds/Signing/Distribution/Settings | 5 | 5 | ✅ |
| Overview: 6 info cards (platform/framework/buildType/source/repo/branch) | ✓ | ✓ | ✅ |
| Builds tab | ✓ | ✓ (shared `BuildsTab`) | ✅ |
| Signing: per-platform (macOS cert + Team ID + Notarization; Win pfx; Linux GPG) | ✓ | ✓ | ✅ |
| Distribution: macOS (DMG/PKG/Homebrew) | ✓ | ✓ | ✅ |
| Distribution: Windows (MSIX/NSIS/MS Store) | ✓ | ✓ | ✅ |
| Distribution: Linux (DEB/RPM/AppImage/Flatpak/Snap) | ✓ | ✓ | ✅ |
| Distribution: enabled/Configure state per item | ✓ | ✓ | ✅ |
| Settings: 6 read-only rows + danger delete | ✓ | ✓ | ✅ |

**Verdict:** ✅ Full parity — near line-for-line, including all distribution items and
signing config fields.

### Gaps (actionable)
- None. (Signing uploads and Distribution "Configure" buttons are non-functional in both.)

---

## 6. DeployCreateEntry wizard

Flutter: `core/widgets/deploy_create_entry.dart` · React: `components/deploy-create-entry.tsx`

| Sub-feature | Flutter | React | Status |
|---|---|---|---|
| Entry view: 3 options (template / repo / upload) | ✓ | ✓ | ✅ |
| Back navigation between views | ✓ | ✓ | ✅ |
| Templates: load by `category` | ✓ | ✓ | ✅ |
| Templates: name search | ✓ | ✓ | ✅ |
| Templates: search also matches description | ✓ | ✗ (name only) | ⚠️ narrower |
| Templates: framework filter chips (All + per-fw) | ✓ | ✗ | ❌ missing |
| Templates: card shows icon + framework badge | ✓ | ✗ (name + desc only) | ⚠️ cosmetic |
| Repo: connection selector | ✓ (dropdown) | ✓ (chips) | ✅ cosmetic diff |
| Repo: repository search field | ✓ | ✗ | ❌ missing |
| Repo: empty state with "Connect to GitHub" button | ✓ (button is TODO) | ✗ (plain text) | ⚠️ minor |
| Repo: pick repo → returns repoConfig | ✓ | ✓ | ✅ |
| Upload: dedicated drag/drop view before finishing | ✓ | ✗ (finishes immediately) | ⚠️ per-feature dialogs cover upload UI |
| Returns `{choice, templateConfig?, repoConfig?}` to caller | ✓ | ✓ | ✅ |

**Verdict:** ⚠️ Functional but with real feature drops in the templates and repo pickers.

### Gaps (actionable)
- **Templates framework filter (❌):** Flutter renders an "All" + per-framework chip row that
  filters the template grid; React has only the name search box. Add the framework filter
  chips to `TemplatesView`.
- **Repo search (❌):** Flutter has a "Search repositories…" field over the repo list; React
  lists repos with no filter. For accounts with many repos this is a usability regression.
  Add a search input to `RepoView`.
- **Templates search scope (⚠️):** Flutter matches `name` **and** `description`; React matches
  `name` only. Widen the React filter predicate.
- **Template card content (⚠️):** Flutter card shows an icon + framework badge; React shows
  name + description only. Add framework badge if visual parity matters.
- **Repo empty state (⚠️):** Flutter shows a styled empty state with a "Connect to GitHub"
  CTA (currently a TODO/no-op); React shows plain text. Low priority since the button is
  non-functional in Flutter too.

---

## 7. Note — standalone Deploy page (out of the 5 core features)

`console/lib/features/deploy/deploy_page.dart` (a generic `/deploy` deployments list +
detail) has **no React port** — `console-react/src/features/` contains only `deploy-shared`
(reusable primitives: `DeploymentsPanel`, `TargetList`, `TargetDetailScaffold`, formatters),
and the React router registers no `deploy` route (only platforms/sites/containers/mobile/
desktop). This appears intentional: the deploy surface was split into the five typed target
pages, and `deploy-shared` factors out the common table/detail plumbing they reuse. Flagged
here for completeness; classify as **❌ not ported** if a 1:1 `/deploy` list is still required,
otherwise **out of scope / superseded**.

---

## Summary

| Feature | Overall | Notable items |
|---|---|---|
| Platforms | ✅ | React ahead (SDK/CLI snippets, Disconnect btn); 1 minor placeholder gap |
| Sites | ✅ | Rollback confirm dialog dropped (⚠️ regression); Refresh btn added |
| Containers | ✅ | Richer React list; verify releases field names (durationMs vs buildDuration) |
| Mobile | ✅ | Full parity |
| Desktop | ✅ | Full parity (line-for-line) |
| DeployCreateEntry | ⚠️ | ❌ framework filter + ❌ repo search missing; narrower template search |

**Counts:** 5 core features at ✅ structural parity, 1 shared wizard at ⚠️.
Hard gaps (❌): 2 (both in the wizard — template framework filter, repo search).
Behavioural regressions (⚠️): rollback confirmation (Sites), template search scope (wizard).
Everything else is cosmetic difference or React enhancement.
