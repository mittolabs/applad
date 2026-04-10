import '../server_client.dart';

/// Functions service (server-side).
class Functions {
  final ApplAdServer _client;

  Functions(this._client);

  Future<Map<String, dynamic>> create({
    required String name,
    required String runtime,
    String entrypoint = 'index.handler',
    int timeout = 15,
    Map<String, String>? vars,
    String? source,
    String cron = '',
  }) async {
    return _client.post('/v1/functions', data: {
      'name': name,
      'runtime': runtime,
      'entrypoint': entrypoint,
      'timeout': timeout,
      if (vars != null) 'vars': vars,
      if (source != null) 'source': source,
      'cron': cron,
    });
  }

  Future<Map<String, dynamic>> list() async {
    return _client.get('/v1/functions');
  }

  Future<Map<String, dynamic>> get(String functionId) async {
    return _client.get('/v1/functions/$functionId');
  }

  Future<Map<String, dynamic>> update(
      String functionId, Map<String, dynamic> data) async {
    return _client.put('/v1/functions/$functionId', data: data);
  }

  Future<Map<String, dynamic>> delete(String functionId) async {
    return _client.delete('/v1/functions/$functionId');
  }

  Future<Map<String, dynamic>> execute(String functionId,
      {Map<String, dynamic>? data}) async {
    return _client.post('/v1/functions/$functionId/executions',
        data: {'data': data ?? {}});
  }

  Future<Map<String, dynamic>> listExecutions(String functionId) async {
    return _client.get('/v1/functions/$functionId/executions');
  }

  Future<Map<String, dynamic>> getExecution(
      String functionId, String executionId) async {
    return _client.get('/v1/functions/$functionId/executions/$executionId');
  }
}
