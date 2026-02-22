/**
 * AppladResponse encapsulates the result returned by a serverless function.
 */
class AppladResponse {
  const AppladResponse({
    this.statusCode = 200,
    required this.data,
    this.headers = const {},
  });

  /// Status code (default: 200).
  final int statusCode;

  /// The JSON data to return.
  final Map<String, dynamic> data;

  /// Optional HTTP response headers.
  final Map<String, String> headers;

  /// Helper for JSON responses.
  factory AppladResponse.json(Map<String, dynamic> data,
      {int statusCode = 200}) {
    return AppladResponse(
      statusCode: statusCode,
      data: data,
    );
  }
}
