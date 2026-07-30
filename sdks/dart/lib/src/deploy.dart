import 'package:dio/dio.dart';

/// Deploy service — manage deploy targets.
///
/// Deploy targets are the deployable unit. The API mounts them under
/// /deploy/targets (there is no flat /deploy resource); triggering a deploy
/// runs the target as an execution.
class Deploy {
  final Dio _dio;

  Deploy(this._dio);

  /// Create a deploy target.
  Future<Map<String, dynamic>> create({
    required String name,
    required String type,
    Map<String, dynamic>? config,
  }) async {
    final res = await _dio.post('/v1/deploy/targets', data: {
      'name': name,
      'type': type,
      if (config != null) ...config,
    });
    return res.data;
  }

  /// List deploy targets.
  Future<Map<String, dynamic>> list() async {
    final res = await _dio.get('/v1/deploy/targets');
    return res.data;
  }

  /// Get a deploy target by ID.
  Future<Map<String, dynamic>> get(String targetId) async {
    final res = await _dio.get('/v1/deploy/targets/$targetId');
    return res.data;
  }

  /// Update a deploy target.
  Future<Map<String, dynamic>> update(
    String targetId,
    Map<String, dynamic> data,
  ) async {
    final res = await _dio.put('/v1/deploy/targets/$targetId', data: data);
    return res.data;
  }

  /// Delete a deploy target.
  Future<void> delete(String targetId) async {
    await _dio.delete('/v1/deploy/targets/$targetId');
  }

  /// Trigger a deploy of the target. Returns the created execution.
  Future<Map<String, dynamic>> deploy(
    String targetId, {
    Map<String, dynamic>? options,
  }) async {
    final res = await _dio.post(
      '/v1/deploy/targets/$targetId/executions',
      data: options ?? {},
    );
    return res.data;
  }
}
