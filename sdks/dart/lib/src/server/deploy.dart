import '../server_client.dart';

/// Deploy service (server-side) — manage deploy targets.
///
/// Deploy targets are the deployable unit. The API mounts them under
/// /deploy/targets (there is no flat /deploy resource); triggering a deploy
/// runs the target as an execution.
class Deploy {
  final ApplAdServer _client;

  Deploy(this._client);

  Future<Map<String, dynamic>> create({
    required String name,
    required String type,
    Map<String, dynamic>? config,
  }) async {
    return _client.post('/v1/deploy/targets', data: {
      'name': name,
      'type': type,
      if (config != null) ...config,
    });
  }

  Future<Map<String, dynamic>> list() async {
    return _client.get('/v1/deploy/targets');
  }

  Future<Map<String, dynamic>> get(String targetId) async {
    return _client.get('/v1/deploy/targets/$targetId');
  }

  Future<Map<String, dynamic>> update(
      String targetId, Map<String, dynamic> data) async {
    return _client.put('/v1/deploy/targets/$targetId', data: data);
  }

  Future<Map<String, dynamic>> delete(String targetId) async {
    return _client.delete('/v1/deploy/targets/$targetId');
  }

  /// Trigger a deploy of the target. Returns the created execution.
  Future<Map<String, dynamic>> deploy(String targetId,
      {Map<String, dynamic>? options}) async {
    return _client.post('/v1/deploy/targets/$targetId/executions',
        data: options ?? {});
  }
}
