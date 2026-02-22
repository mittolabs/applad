/**
 * AppladRequest encapsulates the data passed to a serverless function.
 */
class AppladRequest {
  const AppladRequest({
    required this.body,
    this.headers = const {},
    this.context = const {},
  });

  /// The JSON decoded body of the request (or event payload).
  final Map<String, dynamic> body;

  /// HTTP headers if the function was triggered via HTTP/Webhook.
  final Map<String, String> headers;

  /// Execution context provided by Applad (e.g. auth user info).
  final Map<String, dynamic> context;
}
