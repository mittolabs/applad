# Applad Chat (Flutter)

A small Slack-style chat app built on Applad, running from one Flutter codebase
on **web and mobile**. It exists to exercise the SDKs and a live Applad platform
end to end, and to prove the platform has catered for what a real client needs.

When a feature needed an API that Applad did not have, we added it to Applad
rather than working around it in the app. Those gaps and the reasoning behind
each fix are tracked in **[PLATFORM_AUDIT.md](PLATFORM_AUDIT.md)** — start there
to see what building this surfaced.

## What it demonstrates

| Chat feature | Applad primitive |
|---|---|
| Sign in / sign up, session that survives relaunch | Auth (bearer session, cross-platform) |
| Channels you belong to | Teams (a channel is a team) |
| Invite by email, join by code | Team memberships + accept |
| Only members see a channel's messages | Row RLS scoped `read("team:<id>")` |
| Messages | Databases rows |
| Live delivery | Realtime table subscription |

## Prerequisites

- Flutter via [`fvm`](https://fvm.app) (this repo pins `stable`): `fvm install`
- An Applad project. Use the cloud console (`console.applad.io`) or a local
  stack (`docker compose up`), and note the project id and a server **API key**
  with the databases scope.

## One-time setup: create the schema

The app reads and writes rows but does not define tables. Create the `chat`
database and `messages` table once:

```bash
cd examples/flutter_chat
fvm dart pub get
APPLAD_ENDPOINT=https://api.applad.io \
APPLAD_PROJECT=<project id> \
APPLAD_API_KEY=<server api key> \
fvm dart run tool/bootstrap.dart
```

It is idempotent — re-running just reports what already exists.

## Run

```bash
# Web
fvm flutter run -d chrome \
  --dart-define=APPLAD_ENDPOINT=https://api.applad.io \
  --dart-define=APPLAD_PROJECT=<project id>

# Mobile (a booted simulator/emulator or device)
fvm flutter run \
  --dart-define=APPLAD_ENDPOINT=https://api.applad.io \
  --dart-define=APPLAD_PROJECT=<project id>
```

Sign up, create a channel, and start typing. To bring in a second person: open
a channel, tap the invite icon, send them the code, and they paste it into
"Join a channel". Messages appear live on every signed-in member's screen.

## How it maps to the SDK

- `lib/applad_service.dart` — one `Applad` client, session persistence, current
  user. Login stores the session `secret`; relaunch restores it with `setJWT`.
- `lib/screens/login_screen.dart` — `auth.createAccount` / `createEmailSession`.
- `lib/screens/channels_screen.dart` — `teams.list` / `create` /
  `acceptMembership`.
- `lib/screens/chat_screen.dart` — `databases.from(...).equal(...).get()`,
  `databases.createRow(... permissions: ['read("team:<id>")'])`, and
  `realtime.database(...).onInsert(...)`.
- `tool/bootstrap.dart` — schema creation with an API key.

## Status

Phase 1 (client-only: auth, channels, messages, realtime) is implemented.
Attachments, push, search and an AI assistant are planned next; see the audit.
