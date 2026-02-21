import 'package:dart_frog/dart_frog.dart';

/// Basic CORS middleware — allows cross-origin requests.
Middleware corsMiddleware() {
  return (handler) {
    return (context) async {
      // Handle preflight OPTIONS requests
      if (context.request.method == HttpMethod.options) {
        return Response(
          statusCode: 204,
          headers: _corsHeaders(context.request),
        );
      }

      final response = await handler(context);
      return response.copyWith(headers: {
        ...response.headers,
        ..._corsHeaders(context.request),
      });
    };
  };
}

Map<String, String> _corsHeaders(Request request) {
  final origin = request.headers['origin'] ?? '*';
  return {
    'Access-Control-Allow-Origin': origin,
    'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, PATCH, OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type, Authorization, X-Applad-Key',
    'Access-Control-Allow-Credentials': 'true',
    'Access-Control-Max-Age': '86400',
  };
}
