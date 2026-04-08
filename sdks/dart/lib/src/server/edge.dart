import '../server_client.dart';

class Edge {
  final ApplAdServer _client;

  Edge(this._client);

  Future<Map<String, dynamic>> create({
    required String name,
    required String code,
    String? route,
    int? timeout,
  }) async {
    return _client.post('/v1/edge/functions', data: {
      'name': name,
      'code': code,
      if (route != null) 'route': route,
      if (timeout != null) 'timeout': timeout,
    });
  }

  Future<Map<String, dynamic>> list() async {
    return _client.get('/v1/edge/functions');
  }

  Future<Map<String, dynamic>> get(String functionId) async {
    return _client.get('/v1/edge/functions/$functionId');
  }

  Future<Map<String, dynamic>> update(
    String functionId, {
    String? name,
    String? code,
    String? route,
    int? timeout,
  }) async {
    return _client.put('/v1/edge/functions/$functionId', data: {
      if (name != null) 'name': name,
      if (code != null) 'code': code,
      if (route != null) 'route': route,
      if (timeout != null) 'timeout': timeout,
    });
  }

  Future<Map<String, dynamic>> delete(String functionId) async {
    return _client.delete('/v1/edge/functions/$functionId');
  }

  Future<Map<String, dynamic>> invoke(
    String functionId, {
    Map<String, dynamic>? data,
  }) async {
    return _client.post('/v1/edge/functions/$functionId/invoke',
        data: data ?? {});
  }

  Future<Map<String, dynamic>> listExecutions(String functionId) async {
    return _client.get('/v1/edge/functions/$functionId/executions');
  }
}
