# Auth feature — Flutter → React parity audit

Read-only comparison of the Flutter console Auth feature against the React port.

- **Flutter source:** `console/lib/features/auth/auth_page.dart` (2737 lines)
- **React port:** `console-react/src/features/auth/{AuthPage,UsersTab,TeamsTab,SecurityTab,TemplatesTab,UsageTab,SettingsTab}.tsx` + `auth-config.ts` + `format.ts`

Legend: ✅ present · ⚠️ partial · ❌ missing

**Summary:** The port is near-complete. All six tabs, every list/column/form/dialog/toggle/OAuth provider are ported. No ❌ missing sub-features. A handful of ⚠️ cosmetic-only deltas, plus one place where React is *more* complete than Flutter (Templates editor).

---

## Page shell

| Sub-feature | Status | Notes |
|---|---|---|
| Title "Auth" + subtitle | ✅ | Identical copy |
| Tab bar (Users, Teams, Security, Templates, Usage, Settings) | ✅ | `PageTabs`, same order + keys |
| `?tab=` URL sync | ✅ | `useTabIndex(TAB_KEYS)` mirrors Flutter `tabFromQuery` |
| `?page=` URL sync | ⚠️ | Flutter writes/reads `?page=` into the URL for Users/Teams; React drives pagination through `useResourceList` state and does not round-trip `page` into the URL. Deep-linking a page number is lost. Functional pagination is otherwise equivalent. |

## Users tab

| Sub-feature | Status | Notes |
|---|---|---|
| Users table | ✅ | Via `DataTable` / `useResourceList` on `/users` |
| Column: User ID (`$id`) | ✅ | Flutter marks `$id` sortable (default); React leaves it non-sortable — cosmetic |
| Column: Name (Anonymous fallback) | ✅ | Same fallback |
| Column: Email | ✅ | |
| Column: Status | ✅ | Same tri-state chip: disabled / verified / unverified from `status` + `emailVerification` |
| Column: Joined (relative time) | ✅ | `relativeTime` port matches `_relativeTime` |
| Row icon (user) | ✅ | |
| Search (name/email/ID) | ✅ | Same hint text |
| Pagination + per-page | ✅ | |
| Empty state | ✅ | Same icon/title/subtitle |
| Create user dialog (Name, Email, Password) | ✅ | Same fields; posts `userId: 'unique()'`. React adds submit-disable until email+password present (minor enhancement) |
| Delete user + confirm | ⚠️ | Both confirm before deleting (React `DataTable` built-in "Delete item"). Flutter additionally nests a user-specific "Delete user / This action cannot be undone." dialog; React uses only the generic "Delete item" copy. Cosmetic wording delta. |

## Teams tab

| Sub-feature | Status | Notes |
|---|---|---|
| Teams table | ✅ | On `/teams` |
| Columns: Team ID, Name (Unnamed fallback), Members (`total`), Created | ✅ | |
| Row icon (users) | ✅ | |
| Search / pagination / per-page / empty state | ✅ | |
| Create team dialog (Name, Default roles) | ✅ | Comma-split roles, same `parseRoles` logic |
| Delete team | ✅ | |
| Row click → Team detail | ✅ | |
| **Team detail** header (back, name, ID chip, Add member) | ✅ | |
| Memberships table (Email, Roles, Status, Joined) | ✅ | Same 3fr/3fr/2fr/2fr/40px layout |
| Email fallback (`userEmail`→`invitedEmail`→—) | ✅ | |
| Role chips / "No roles" | ✅ | `RoleChip` ported |
| Status chip (active/pending from `joined`) | ✅ | |
| Membership empty state | ✅ | |
| Add member dialog (Email, Roles) | ✅ | |
| Row actions menu (Edit roles / Remove) | ✅ | `DropdownMenu` |
| Edit roles dialog (delete + re-invite workaround) | ✅ | Same "no PATCH endpoint" workaround |
| Remove member + confirm | ✅ | `ConfirmDialog` with same copy |

## Security tab

| Sub-feature | Status | Notes |
|---|---|---|
| Header + subtitle | ✅ | |
| Load `/projects/{id}/auth/security` | ✅ | |
| Save via PUT (merge patch) | ✅ | Same merge-then-PUT behavior |
| Number: Users limit | ✅ | |
| Number: Session length (seconds) | ✅ | default 31536000 |
| Number: Sessions per user | ✅ | default 10 |
| Number: Password minimum length | ✅ | default 8 |
| Number: Password history | ✅ | |
| Toggle: Password dictionary check | ✅ | |
| Toggle: Personal data check | ✅ | |
| Toggle: Require MFA | ✅ | |
| Toggle: Session alerts | ✅ | |
| Toggle: Invalidate sessions on password change | ✅ | default true |
| Digits-only number input | ✅ | React strips non-digits on change |

## Templates tab

| Sub-feature | Status | Notes |
|---|---|---|
| Header + subtitle | ✅ | |
| 5 template rows (Email verification, Magic URL, Password recovery, Invitation, OTP verification) | ✅ | Same names/types/descriptions |
| Email vs SMS icon | ✅ | Mail / Smartphone |
| Edit button | ✅ (React better) | Flutter's Edit button is a no-op (`onPressed: () {}`). React opens a real editor dialog (Subject for email, Body/Message textarea with placeholder hints). No persistence on either side (React `onSubmit` just closes). |

## Usage tab

| Sub-feature | Status | Notes |
|---|---|---|
| Header + subtitle | ✅ | |
| 3 stat cards (Total users, Active sessions, New signups 30d) — all "—" | ✅ | Static placeholders, identical |
| "Usage charts coming soon" panel | ✅ | |

## Settings tab (auth methods + OAuth providers)

| Sub-feature | Status | Notes |
|---|---|---|
| "Auth methods" section header | ✅ | |
| 7 auth-method toggle rows (email, phone, magic, emailOtp, anonymous, teamInvites, jwt) | ✅ | Same ids/labels/descriptions/icons/defaults in `auth-config.ts` |
| Method icon tint when enabled | ✅ | |
| 1-col / 2-col responsive method grid | ✅ | CSS grid vs Flutter LayoutBuilder; same effect |
| "OAuth2 Providers" section header | ✅ | |
| 15 provider cards (Google, GitHub, Apple, Facebook, Discord, Twitter/X, Microsoft, Slack, Spotify, LinkedIn, GitLab, Bitbucket, Twitch, Notion, Stripe) | ✅ | Full list ported with colors/letters/fields/setup notes |
| Provider badge (letter avatar) | ⚠️ | React always renders the letter/first-char glyph. Flutter additionally swaps in Lucide icons for github/apple/spotify/gitlab (`_iconFor`). Providers with empty `letter` (GitHub, Apple, Spotify, GitLab) fall back to first name char in React (G/A/S/G) instead of an icon. Cosmetic only. |
| Enabled/disabled state ring + label | ✅ | Green border + enabled/disabled text |
| Inline toggle pill on card | ✅ | stopPropagation so card click still opens dialog |
| Card click → configure dialog | ✅ | |
| Responsive card columns (2→5) | ✅ | |
| **OAuth config dialog** title + subtitle | ✅ | |
| Docs link in subtitle | ⚠️ | Flutter renders "visit the docs." as an (inert, non-linking) accent span. React drops the "visit the docs" phrase from the subtitle entirely. Both are non-functional links; React omits the text. |
| Enabled toggle in dialog | ✅ | |
| Per-provider fields (text/secret/multiline) | ✅ | Same field defs; multiline = Textarea rows=5 |
| Secret show/hide (eye toggle) | ✅ | |
| Info box with setup note | ✅ | Same generic + per-provider notes (Google custom note ported) |
| Redirect URI field (read-only) | ✅ | Same URL template |
| Copy URI button (copy→check swap) | ✅ | 2s revert |
| Cancel / Update footer | ✅ | On Update, saves enabled state up to parent |
| Persistence of provider config to backend | ⚠️ (parity) | Neither side persists OAuth method/provider enabled-state or field values to the backend — both keep them in local component state only. So this is at parity (both are UI-only stubs), noted for completeness. |

---

### Gaps (actionable)

- **`?page=` URL round-trip (Users/Teams):** Flutter reflects the current page into the URL query so pages are deep-linkable/refresh-safe; React keeps page in `useResourceList` state only. Add page↔URL sync if deep-linking parity matters.
- **Provider badge icons:** Port `_iconFor` so GitHub/Apple/Spotify/GitLab render their Lucide icons instead of falling back to a first-letter glyph (these four have empty `letter` values).
- **OAuth dialog "visit the docs" text:** React's dialog subtitle omits the trailing "For more info you can visit the docs." phrase present in Flutter. Re-add (even as inert text) for copy parity.
- **Delete-user confirm copy:** React shows the generic "Delete item" confirm; Flutter shows a user-specific "Delete user — This action cannot be undone." dialog. Add specific copy if desired (functionally both confirm).
- **`$id` column sortability (Users):** trivial — Flutter treats `$id` as sortable by default; React leaves it non-sortable.
- **Not a gap (React ahead):** Templates "Edit" opens a working subject/body editor dialog in React vs a no-op button in Flutter. Neither persists.
