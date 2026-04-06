import 'package:dio/dio.dart';

/// Deploy service — manage deployments.
class Deploy {
  final Dio _dio;

  Deploy(this._dio);

  /// Create a deployment.
  Future<Map<String, dynamic>> create({
    required String name,
    required String type,
    Map<String, dynamic>? config,
  }) async {
    final res = await _dio.post('/v1/deploy', data: {
      'name': name,
      'type': type,
      if (config != null) 'config': config,
    });
    return res.data;
  }

  /// List deployments.
  Future<Map<String, dynamic>> list() async {
    final res = await _dio.get('/v1/deploy');
    return res.data;
  }

  /// Get a deployment by ID.
  Future<Map<String, dynamic>> get(String deploymentId) async {
    final res = await _dio.get('/v1/deploy/$deploymentId');
    return res.data;
  }

  /// Delete a deployment.
  Future<void> delete(String deploymentId) async {
    await _dio.delete('/v1/deploy/$deploymentId');
  }
}
