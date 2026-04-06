import 'package:dio/dio.dart';

/// Flags service — evaluate feature flags.
class Flags {
  final Dio _dio;

  Flags(this._dio);

  /// Get a single flag evaluation by key.
  Future<Map<String, dynamic>> getFlag(String key) async {
    final res = await _dio.get('/v1/flags/evaluate/$key');
    return res.data;
  }

  /// Get all flag evaluations with optional context.
  Future<Map<String, dynamic>> getAllFlags({
    Map<String, dynamic>? context,
  }) async {
    final res = await _dio.post('/v1/flags/evaluate/all', data: {
      if (context != null) 'context': context,
    });
    return res.data;
  }

  /// Evaluate a flag with key and context.
  Future<Map<String, dynamic>> evaluateFlag({
    required String key,
    Map<String, dynamic>? context,
  }) async {
    final res = await _dio.post('/v1/flags/evaluate', data: {
      'key': key,
      if (context != null) 'context': context,
    });
    return res.data;
  }
}
