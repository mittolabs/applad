# Parity audit — Flutter console vs React port

Notebook tracking feature parity between the Flutter console (`console/lib/features/*`)
and the React port (`console-react/src/features/*`). Per-feature detail lives in
`docs/parity/*.md`; this file is the consolidated index + prioritized gap list.

Legend: ✅ present · ⚠️ partial/simplified · ❌ missing

**Status: audit complete (10/10). P0–P4 all fixed (builds 22–26, live on :3005).** Full parity reached; remaining deltas are intentional (matched stubs, or places React is ahead — see end).

Bottom line: the React port is **broadly at parity**. Most areas are 100% or ahead of
Flutter. The real work concentrates in a handful of spots (Databases table-Settings,
onboarding flow, a few functional field/endpoint mismatches). Several "missing" items are
**dead code or stubs that match on both sides** — those are NOT gaps (listed at the end).

## Feature reports
| Area | Report | ❌ | ⚠️ | Headline |
|---|---|---|---|---|
| Standalone (login/onboarding/projects/account) | `docs/parity/standalone.md` | 16 | 12 | Most real gaps live here |
| Auth | `docs/parity/auth.md` | 0 | 6 | Full parity (ahead on templates) |
| Databases | `docs/parity/databases.md` | 4 | 4 | Table-Settings tab reduced to ID+Delete |
| Functions + Storage | `docs/parity/functions-storage.md` | 1 | 4 | Exec logs not viewable; else ahead |
| Messaging + Content | `docs/parity/messaging-content.md` | 0 | 1 | Near 1:1 |
| Workflows | `docs/parity/workflows.md` | 5 | 4 | All 60 nodes present; missing canvas power-tools |
| Flags/Vault/Env/Realtime/Health | `docs/parity/flags-vault-env-realtime-health.md` | 0 | 10 | Parity/ahead; 1 filter bug |
| Platforms/Sites/Containers/Mobile/Desktop | `docs/parity/platforms-deploy.md` | 2 | 4 | Deploy wizard filters/search missing |
| Settings + Overview | `docs/parity/settings-overview.md` | 0 | 7 | Full parity; endpoint/feedback nits |
| Observe suite | `docs/parity/observe.md` | 1 | 3 | Full parity (the "1" is dead code both sides) |

---

## Consolidated gaps — prioritized

### P0 · Functional bugs (wrong data / broken action) — ✅ ALL FIXED
1. ✅ **Account – password change.** Now sends `oldPassword` (verified vs backend + Flutter). `account/AccountPage.tsx`
2. ✅ **Projects – Usage tab.** Reads `totalProjects/totalUsers/totalStorage/totalExecutions` (+ byte formatting for storage). `projects/ProjectsPage.tsx`
3. ✅ **Projects – member role change.** Role is now a `Select` that `PATCH /organizations/{orgId}/members/{id}` on change (backend `updateMemberRole` confirmed).
4. ✅ **Flags – status filter.** Backend `/flags` returns all rows (no server filter), so Type/Status now filter **client-side** (like Flutter); option values aligned to `enabled/disabled`. `flags/FlagsPage.tsx`
5. ✅ **Overview – Recent Deployments.** Now `GET /deploy/releases?limit=5` (matches Flutter). `overview/OverviewPage.tsx`
6. ✅ **Containers/Sites – Releases columns.** Panel now reads `durationMs`/`triggerType` (with `buildDuration`/`sourceType`/`source` fallbacks). `deploy-shared/DeploymentsPanel.tsx`

_(Fixed in build 22; live on :3005.)_

### P1 · Missing real sub-features — ✅ ALL FIXED (build 23)
7. ✅ **Databases – table Settings tab.** Added Enabled toggle, Name edit, Permissions panel (role/action rows + add/remove via `POST .../permissions`), Row-security toggle + RLS explainer; fetches `GET …/tables/{id}`. `databases/TableDetail.tsx`
8. ✅ **Login – reset mode.** Added token-paste field, confirm-password field (+ mismatch validation), password eye-reveal (both fields), and a version footer. `login/LoginPage.tsx`
9. ✅ **Onboarding.** Replaced the 3-step project stepper with Flutter's **single-step org creation** (`POST /organizations` → drops into its projects). Decision: matched Flutter (also fixes new-user-has-no-org). `onboarding/OnboardingPage.tsx`
10. ✅ **Projects – Settings/header.** Added the **Organization ID** card and a **member-avatar stack** (overlapping avatars + `+N`) in the workspace heading. `projects/ProjectsPage.tsx`
11. ✅ **Functions – execution logs.** Row click opens a detail dialog showing `output` (CodeBlock) + `errors` (red monospace) with status/duration/timestamp. `functions/FunctionDetail.tsx`
12. ✅ **Deploy wizard.** Added template framework-filter chips, name+description search, template card icon/framework badge, and a repository search field. `components/deploy-create-entry.tsx`
13. ✅ **Workflows canvas power-tools.** Node duplication (⌘D + menu), right-click node/canvas context menus, sticky notes (canvas-local), dockable live-logs panel, and shortcuts (⌘A/`d`/Tab/Esc). `features/workflows/*`

### P2 · Behavioral regressions — ✅ ALL FIXED (build 24)
14. ✅ **Sites – Instant Rollback** now shows a confirm dialog before rolling back, and matches the last `success` release only (was `success|ready|active`). `sites/SiteDetail.tsx`
15. ✅ **Databases – Level-1 duplication removed.** `DatabaseList` no longer renders its own `<h1>`/tab bar; the page owns them, and the real **Usage** tab (3 stat cards + chart) now shows under the page's Usage tab. `databases/DatabaseList.tsx`, `DatabasesPage.tsx`

### P3 · Cross-cutting polish — ✅ ALL FIXED (build 25)
16. ✅ **Mutation feedback (toasts).** Added a global TanStack `MutationCache.onError` that surfaces `friendlyError()` as a toast for any mutation without its own `onError` — no create/save/delete fails silently anymore. `main.tsx`
17. ✅ **StatusChip color/label — investigated; shared component was already correct.** The Flutter shared `status_chip.dart` maps `inactive`→danger and `building/deploying`→info, **identical** to React (the audit note's premise was inverted). The real divergence was overview-local: `overview_page.dart` colors in-progress builds amber and `rolled_back` purple via its own `_statusColor`/`_statusLabel`. Ported that as a local `DeployStatusChip` in the overview (shared StatusChip left untouched). `overview/OverviewPage.tsx`
18. ✅ **Monaco theme-aware.** Added `useMonacoTheme()` (`stores/theme.ts`); SQL console + function editor now follow light/dark. `databases/SqlConsole.tsx`, `functions/FunctionDetail.tsx`
19. ✅ **URL-param divergence.** Environments now uses `?envId=` (was `?env=`) to match Flutter deep-links. `environments/EnvironmentsPage.tsx`
20. ✅ **Observe sortability — aligned to Flutter (audit note was inverted).** Flutter sets **all** Observe columns `sortable: false` (backendless stub data, 0 sortable / 10 non-sortable); React had made all 22 sortable. Flipped React's Observe columns to `sortable: false` for faithful parity. `features/observe/*`

### Login page — ✅ REBUILT to match Flutter (build 30)
Full rewrite of `features/login/LoginPage.tsx` against `login_page.dart` (previously a loose approximation):
- **5:4 branding/form split** (`min-[900px]` breakpoint, matching Flutter's 900px) with a 1px white/6% divider.
- **Branding panel**: mascot head (`applad-mascot-head.png`) with the exact two-layer blue glow + lowercase "applad" wordmark, the "Go from idea / to production today." tagline (52px), subtitle, and a faithful SVG port of the `_PanelShapes` CustomPaint (4 swept shapes, userSpaceOnUse gradients over a 0–100 viewBox). Signup mode swaps the tagline to "Your backend, / your rules."
- **Form panel**: "Sign in" heading (no subtitle on login/signup, per Flutter), **Continue with GitHub** always shown on login + any extra providers, **or** divider, Email/Password fields, accent submit, a centered links row (`Forgot password?` · `Sign up`), the "By signing in…" policy line, and a `v{version}` footer.
- Kept all four modes (login/signup/forgot/reset) incl. the SMTP-not-configured **surfaced-token box** + "Use this token →", 8-char password hint, and signup policy checkbox.
- Verified live: wide + narrow (no overflow), login submit → API → "Invalid email or password", forgot flow, and signup correctly hidden when `signup-status` is disabled.

### Get Started page — ✅ REDESIGNED (build 29)
Was a placeholder ("coming from the React port"). Rebuilt as a working project onboarding hub (`features/get-started/GetStartedPage.tsx`, wired in `router.tsx`):
- Live **setup progress** banner (real resource counts) + a 4-step **checklist** (database/auth/storage/function) that ticks off as resources are created and deep-links to each feature.
- **Connect your app** card with a 5-language SDK switcher (JS/Node/Dart/Go/Python) — real package names + endpoint/projectId injected into install + init snippets via `CodeBlock`.
- **Project credentials** card (copyable endpoint, `IdText` project ID, Manage-API-keys CTA).
- **Keep exploring** resource links. Fully responsive (verified 390/1400px, no overflow; code blocks scroll internally).

### Column-visibility menu — ✅ RESTYLED (build 28)
`components/data-table.tsx` — replaced the Radix dropdown checkmark items with the Flutter design: a "Columns" header + labelled blue-filled `Checkbox` rows in a popover; preserves the min-1-visible rule (last checked box disabled).

### Responsive layout — ✅ FIXED (build 27)
Ported shell.dart's responsive breakpoints (was desktop-only):
- **< 650px (mobile):** icon rail + detail panel are replaced by a fixed **bottom nav** (Overview/Build/Platforms/Observe/Settings, capped at 5) + a **group bottom sheet** for groups with children (Build/Observe); direct-nav groups route straight through. Console footer hidden (matches Flutter). `shell/MobileNav.tsx`, `shell/Shell.tsx`
- **< 780px:** top-nav Feedback + Support collapse into a **⋯ overflow menu** (`NavOverflowMenu`, ports `_NavOverflowMenu`); the Feedback/Support panel bodies were extracted so both the inline buttons and the overflow reuse them. `shell/navbar-popovers.tsx`, `shell/TopNav.tsx`
- Breadcrumb (org/project switchers) now truncates instead of forcing horizontal overflow; panel popovers cap at `100vw`.
- New `hooks/use-media-query.ts` (`useIsMobile` <650, `useIsNavCompact` <780) mirrors `_isMobile`.
- Verified via headless CDP at 390/700/1200px: no horizontal overflow (scrollWidth==clientWidth) at any width; bottom-nav, group sheet, tablet overflow, and desktop rail all confirmed.

### P4 · Cosmetic / minor — ✅ FIXED (build 26)
- ✅ **Provider badge icons.** Ported `_iconFor`: GitHub/Apple/Spotify/GitLab now render Lucide icons (GitBranch/Apple/Music/GitBranch) instead of a first-letter glyph. `auth/SettingsTab.tsx`
- ✅ **Delete-confirm wording.** Added optional `deleteTitle`/`deleteMessage` to `<DataTable>` (defaults unchanged). Wired entity-specific copy: API-key list → "Any applications using this key will lose access immediately." (matches the full-page detail, which already had it); user delete → "Delete user / …This action cannot be undone." `components/data-table.tsx`, `settings/SettingsPage.tsx`, `auth/UsersTab.tsx`
- ✅ **Content rich-text placeholder tip.** Restored the fuller hint: `Write in Markdown…\n\nTip: **bold**, *italic*, \`code\`, ## headings`. `content/EntryEditor.tsx`
- ✅ **Platform identifier placeholder.** `PlatformDetail` Settings now uses dynamic `identityHint(type)` (was hardcoded `com.example.myapp`). `platforms/PlatformDetail.tsx`
- ✅ **Workflow minimap toggle + zoom-% readout.** Added a bottom-right `ZoomPanel`: live zoom % (click to reset to 100%) + a minimap show/hide toggle (minimap was always-on). `workflows/WorkflowBuilder.tsx`
- ✅ **Invite dialog.** Added the optional **Name** field, the **Owner** role option, and per-role description helper text (owner/admin/member) — matching `projects_page.dart`. `projects/ProjectsPage.tsx`
- ✅ **Account Activity tab icon.** Uses the `Activity` icon (was reusing `Monitor`). `account/AccountPage.tsx`

_Not changed (deliberate):_ OAuth dialog "visit docs" copy and API-key date formatting were re-checked and already match; the login Google-glyph / always-render-GitHub-button behavior is a functional login-screen difference tracked under standalone, not a P4 cosmetic.

---

## NOT gaps (matched stubs / dead code on both sides — do not "fix")
- **MFA** toggle (stub both), **OAuth provider/method enable + field values** (UI-only both), **image transforms** (no UI either side), **CSV row import** (stub both), **template variables** (stub both), **log tailing / session replays** (backendless both), **Observe Performance view** (dead/unreachable code in Flutter too), **manual upload / signing / distribution / git-OAuth** deploy placeholders (both), **edge exec-count badges** (dead code in Flutter).
- Several places React is **ahead** of Flutter: Storage upload/download/search, Functions Monaco editor, Environments inline var editing, Realtime WS-URL block, Health auto-poll, type-to-confirm project delete, working Templates editor, wired search boxes.
