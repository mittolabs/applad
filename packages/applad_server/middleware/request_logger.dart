import 'dart:convert';
import 'dart:io';
import 'package:dart_frog/dart_frog.dart';

/// Structured JSON request logging middleware.
Middleware requestLogger() {
  return (handler) {
    return (context) async {
      final start = DateTime.now();
      final request = context.request;

      Response response;
      try {
        response = await handler(context);
      } catch (e, st) {
        _log({
          'level': 'error',
          'method': request.method.value,
          'path': request.uri.path,
          'error': e.toString(),
          'stack': st.toString(),
          'timestamp': start.toIso8601String(),
        });
        rethrow;
      }

      final duration = DateTime.now().difference(start);
      _log({
        'level': response.statusCode >= 500
            ? 'error'
            : response.statusCode >= 400
                ? 'warn'
                : 'info',
        'method': request.method.value,
        'path': request.uri.path,
        'status': response.statusCode,
        'duration_ms': duration.inMilliseconds,
        'timestamp': start.toIso8601String(),
      });

      return response;
    };
  };
}

void _log(Map<String, dynamic> data) {
  stdout.writeln(jsonEncode(data));
}
