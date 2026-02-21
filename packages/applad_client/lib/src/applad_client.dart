library;

import 'auth/auth_client.dart';
import 'tables/table_client.dart';
import 'storage/storage_client.dart';
import 'functions/functions_client.dart';
import 'realtime/realtime_client.dart';

/// The main entry point for the Applad client SDK.
///
/// Example:
/// ```dart
/// final client = ApplAdClient(
///   endpoint: 'https://api.acme-corp.com',
///   projectId: 'mobile-app',
/// );
///
/// await client.auth.signIn(email: 'user@example.com', password: 'password');
/// final posts = await client.from('posts').select().eq('published', true).get();
/// ```
final class ApplAdClient {
  ApplAdClient({
    required this.endpoint,
    required this.projectId,
    this.apiKey,
  }) {
    auth = AuthClient(client: this);
    storage = StorageClient(client: this);
    functions = FunctionsClient(client: this);
    realtime = RealtimeClient(client: this);
  }

  /// The base URL of the Applad server.
  final String endpoint;

  /// The project ID to connect to.
  final String projectId;

  /// Optional API key for anonymous access.
  final String? apiKey;

  /// Current auth token (set after sign-in).
  String? _authToken;

  // ignore: unnecessary_getters_setters
  String? get authToken => _authToken;
  set authToken(String? value) => _authToken = value;

  late final AuthClient auth;
  late final StorageClient storage;
  late final FunctionsClient functions;
  late final RealtimeClient realtime;

  /// Start a table query for the given table name.
  TableClient from(String table) => TableClient(client: this, table: table);

  Map<String, String> get defaultHeaders => {
        'Content-Type': 'application/json',
        'X-Applad-Project': projectId,
        if (apiKey != null) 'X-Applad-Key': apiKey!,
        if (_authToken != null) 'Authorization': 'Bearer $_authToken',
      };
}
