library;

import '../applad_client.dart';

/// Client for invoking Applad serverless functions.
final class FunctionsClient {
  FunctionsClient({required this.client});

  final ApplAdClient client;

  /// Invoke a named function.
  ///
  /// Example:
  /// ```dart
  /// await client.functions.invoke('send-welcome-email', body: {
  ///   'userId': 'user-123',
  /// });
  /// ```
  Future<Map<String, dynamic>> invoke(
    String functionName, {
    Map<String, dynamic>? body,
    Map<String, String>? headers,
  }) async {
    throw UnimplementedError('Function invocation — available in Phase 3');
  }
}
