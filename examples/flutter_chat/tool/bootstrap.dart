// Creates the chat schema once. Run it against a project before the app can
// talk to it:
//
//   APPLAD_ENDPOINT=https://api.applad.io \
//   APPLAD_PROJECT=<project id> \
//   APPLAD_API_KEY=<a server API key with databases scope> \
//   dart run tool/bootstrap.dart
//
// It is idempotent: re-running skips anything that already exists.
//
// Why an API key and not a user session: defining tables is an operator action,
// not something an end user should do. The app itself only reads and writes
// rows as the signed-in user.
import 'dart:io';

import 'package:applad/applad.dart';

Future<void> main() async {
  final endpoint = Platform.environment['APPLAD_ENDPOINT'] ?? 'http://localhost:8080';
  final projectId = Platform.environment['APPLAD_PROJECT'];
  final apiKey = Platform.environment['APPLAD_API_KEY'];

  if (projectId == null || projectId.isEmpty || apiKey == null || apiKey.isEmpty) {
    stderr.writeln('Set APPLAD_PROJECT and APPLAD_API_KEY (and optionally APPLAD_ENDPOINT).');
    exit(1);
  }

  final client = Applad(endpoint: endpoint, projectId: projectId)..setKey(apiKey);
  final db = client.databases;

  await _step('database "chat"', () => db.createDatabase(name: 'chat', databaseId: 'chat'));

  await _step(
    'table "messages"',
    () => db.createTable(
      databaseId: 'chat',
      name: 'messages',
      tableId: 'messages',
      // Any signed-in user may post; who can READ a given message is decided by
      // that row's own read("team:<channelId>") permission, so document
      // security must be on.
      permissions: ['create("users")'],
      documentSecurity: true,
    ),
  );

  Future<void> col(String key, int size, {bool required = true}) => _step(
        'column "$key"',
        () => db.createStringColumn(
          databaseId: 'chat',
          tableId: 'messages',
          key: key,
          size: size,
          required_: required,
        ),
      );

  await col('channel_id', 128);
  await col('user_id', 128);
  await col('author_name', 256);
  await col('body', 8000);

  await _step(
    'index on channel_id',
    () => db.createIndex(
      databaseId: 'chat',
      tableId: 'messages',
      key: 'idx_channel',
      type: 'key',
      columns: ['channel_id'],
    ),
  );

  stdout.writeln('\nDone. The chat schema is ready.');
  exit(0);
}

Future<void> _step(String what, Future<void> Function() run) async {
  try {
    await run();
    stdout.writeln('created  $what');
  } catch (e) {
    final s = e.toString();
    if (s.contains('already') || s.contains('409') || s.contains('exists')) {
      stdout.writeln('exists   $what');
    } else {
      stdout.writeln('FAILED   $what -> $e');
    }
  }
}
