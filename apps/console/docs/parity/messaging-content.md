# Parity audit: Messaging & Content

Read-only comparison of the Flutter console features against their React ports.

- **Flutter**: `console/lib/features/messaging/messaging_page.dart` (2151 lines), `console/lib/features/content/content_page.dart` (2029 lines)
- **React**: `console-react/src/features/messaging/*.tsx`, `console-react/src/features/content/*.tsx`

Legend: ✅ full parity · ⚠️ present but with a gap · ❌ missing

---

## Messaging

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| Page shell + tabs (Messages/Topics/Templates/Providers) | ✓ | `MessagingPage.tsx` | ✅ | Same 4 tabs, tab synced to URL (`useTabIndex`). |
| Messages list — DataTable (ID/Type/Status/Created) | ✓ | `MessagesTab.tsx` | ✅ | Same columns, cell values, row type-icon. |
| Messages — status badge (sent/failed/draft/processing) | ✓ | `shared.tsx` `MessageStatusChip` | ✅ | Variant mapping matches `_StatusBadge`. |
| Messages — type + status filters | ✓ | `MessagesTab.tsx` | ✅ | Both filters present (React adds proper option labels). |
| Messages — search + pagination | ✓ (page synced to `?page`) | `useResourceList` | ✅ | React also syncs `?page` to URL. |
| Messages — empty state / loading / error+retry | ✓ | ✓ | ✅ | |
| Create-message menu (Email/SMS/Push) | Popup menu | `CreateMenu` dropdown | ✅ | Same 3 options + icons. |
| Create Email (subject, message, HTML toggle, To, schedule) | ✓ | `CreateMessageView.tsx` | ✅ | All fields + schedule hint. |
| Create SMS (message + `/900` counter, To, schedule, preview) | ✓ | ✓ | ✅ | Char counter + live SMS preview. |
| Create Push (title, message + `/1000`, media upload, FCM token, schedule, preview) | ✓ | ✓ | ✅ | Media dropzone is display-only in both. |
| Create — Save as draft / Create / Cancel actions | ✓ | ✓ | ✅ | Same POST bodies (`/messaging/messages/{email,sms,push}`, `draft` flag). |
| Live SMS phone preview | ✓ | `PhonePreview.tsx` `SmsPhonePreview` | ✅ | Status bar, avatar, timestamp, bubble/empty hint. |
| Live Push phone preview | ✓ | `PushPhonePreview` | ✅ | App row, title, body, empty hint. |
| Message detail (type/message/targets cards + ID tag) | ✓ | `MessageDetail.tsx` | ✅ | Same 3 cards, created date, status chip. |
| Topics — search + list (hash icon, subscriber count) | ✓ | `TopicsTab.tsx` | ✅ | |
| Topics — create dialog | ✓ | ✓ | ✅ | POST `/messaging/topics`. |
| Topics — empty state + no-match state | ✓ | ✓ | ✅ | (No edit/delete of topics in either.) |
| Templates — list (type icon/color, subject, type badge) | ✓ | `TemplatesTab.tsx` | ✅ | |
| Templates — create (name, type, subject, body) | ✓ | ✓ | ✅ | Both send `variables: []` (no variable-editing UI in either). |
| Templates — delete + confirm | ✓ | ✓ | ✅ | |
| Templates — empty state | ✓ | ✓ | ✅ | |
| Providers — 8 static config cards grouped Email/SMS/Push | ✓ | `ProvidersTab.tsx` | ✅ | Identical provider set + env vars. |

### Gaps (actionable)

- None material. Messaging is a faithful, essentially 1:1 port.
- (Nit, not a regression) The **media upload** on the Push form is a non-functional dropzone in *both* Flutter and React — no file is actually attached. Parity holds; flag only if real upload is ever wanted.
- (Nit) The **template "variables"** feature is stubbed in both (`variables: []`, no editor UI). Parity holds.

---

## Content

| Sub-feature | Flutter | React | Status | Notes |
|---|---|---|---|---|
| 3-screen state machine (Types → Entries → Entry editor) | ✓ | `ContentPage.tsx` | ✅ | Same selection-driven navigation + back behavior. |
| Types — search | ✓ | `TypesView.tsx` | ✅ | name + slug match. |
| Types — filter chips (All/Versioned/Localized) | ✓ | ✓ | ✅ | |
| Types — grid/list view toggle | ✓ | ✓ | ✅ | |
| Types — count + New type button | ✓ | ✓ | ✅ | |
| Types — grid cards (icon, name, slug, badges, edit/delete) | ✓ | `TypeCard` | ✅ | Badges: field count, versioned, i18n. |
| Types — list rows (hover actions) | ✓ | `TypeListRow` | ✅ | |
| Types — empty state (+ create action) | ✓ | ✓ | ✅ | |
| Type create/edit dialog (name, slug, versioning, localization) | ✓ | `TypeFormDialog.tsx` | ✅ | Edit sends only name+fields in both; React additionally disables slug/toggles on edit (clearer, same effect). |
| Custom fields editor (key, label, type dropdown, required, add/remove) | ✓ | ✓ | ✅ | Same 9 field types (`text…seo`). |
| Type delete + confirm | ✓ | ✓ | ✅ | |
| Entries — status filter chips (All/Draft/Published/Archived) | ✓ | `EntriesView.tsx` | ✅ | |
| Entries — table (Slug/Status/Locale/Ver./Updated) | ✓ | ✓ | ✅ | |
| Entries — row actions (publish/unpublish/delete) | ✓ | ✓ | ✅ | PATCH publish/unpublish, DELETE. |
| Entries — empty state | ✓ | ✓ | ✅ | |
| Entry editor — header (back, title, Save draft, Publish) | ✓ | `EntryEditor.tsx` | ✅ | "Update & publish" when already published; green publish button. |
| Entry editor — locale field (localized types) | ✓ | ✓ | ✅ | |
| Entry editor — dynamic field inputs | ✓ | `FieldInput` | ✅ | boolean/richtext/number/date/seo/text all covered. |
| Entry editor — rich text editor | ✓ | `RichTextEditor` | ⚠️ | React placeholder is abbreviated ("Write in Markdown…") vs Flutter's version with formatting tips (`**bold**, *italic*, \`code\`, ## headings`). Cosmetic. |
| Entry editor — draft info card (new entry) | ✓ | ✓ | ✅ | |
| Entry editor — sidebar Status + Version | ✓ | ✓ | ✅ | |
| Entry editor — version history (last 5) | ✓ | `VersionHistory` | ✅ | React adds a "No versions" empty label (minor improvement). |
| Slug derivation from first text field | ✓ | `slugify` | ✅ | Same regex/logic. |

### Gaps (actionable)

- **RichTextEditor placeholder** (`EntryEditor.tsx`, `richtext` case): React drops the Markdown formatting hint that Flutter shows (`Tip: **bold**, *italic*, \`code\`, ## headings`). Restore the fuller hint text for parity. Cosmetic only.
- No functional gaps otherwise. Content is a faithful, essentially 1:1 port.

---

## Summary

Both features are near-complete ports with no missing sub-features. The audit found **1 cosmetic gap** (Content rich-text placeholder hint) and **2 stubbed-in-both nits** (Push media upload, template variables) that are not regressions from Flutter. No ❌ items.
