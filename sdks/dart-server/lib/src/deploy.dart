import 'client.dart';

/// Deploy service — manage deployments.
class Deploy {
  final ApplAdServer _client;

  Deploy(this._client);

  /// Create a deployment.
  Future<Map<String, dynamic>> create({
    required String name,
    required String type,
    Map<String, dynamic>? config,
  }) async {
    return _client.post('/v1/deploy', data: {
      'name': name,
      'type': type,
      if (config != null) 'config': config,
    });
  }

  /// List deployments.
  Future<Map<String, dynamic>> list() async {
    return _client.get('/v1/deploy');
  }

  /// Get a deployment by ID.
  Future<Map<String, dynamic>> get(String deploymentId) async {
    return _client.get('/v1/deploy/$deploymentId');
  }

  /// Update a deployment.
  Future<Map<String, dynamic>> update(
      String deploymentId, Map<String, dynamic> data) async {
    return _client.put('/v1/deploy/$deploymentId', data: data);
  }

  /// Delete a deployment.
  Future<Map<String, dynamic>> delete(String deploymentId) async {
    return _client.delete('/v1/deploy/$deploymentId');
  }
}
