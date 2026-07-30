/// Applad server-side Dart SDK.
///
/// Uses `package:http` and API key authentication.
/// No Flutter dependency.
///
/// ```dart
/// import 'package:applad/applad_server.dart';
///
/// final server = ApplAdServer(
///   endpoint: 'https://applad.example.com',
///   projectId: 'my-project',
///   apiKey: 'applad_key_abc123',
/// );
/// ```
library applad_server;

export 'src/server/analytics.dart';
export 'src/server_client.dart';
export 'src/server/users.dart';
export 'src/server/databases.dart';
export 'src/query_builder.dart' show QueryBuilder, QueryResult;
export 'src/server/storage.dart';
export 'src/server/edge.dart';
export 'src/server/functions.dart';
export 'src/server/teams.dart';
export 'src/server/workflows.dart';
export 'src/server/messaging.dart';
export 'src/server/deploy.dart';
export 'src/server/flags.dart';
export 'src/server/regions.dart';
export 'src/server/search.dart';
export 'src/server/vectors.dart';
