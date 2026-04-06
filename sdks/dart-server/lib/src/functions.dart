import 'client.dart';

/// Functions service — manage serverless functions and executions.
class Functions {
  final ApplAdServer _client;

  Functions(this._client);

  /// Create a function.
  Future<Map<String, dynamic>> create({
    required String name,
    required String runtime,
    String entrypoint = 'index.handler',
    int timeout = 15,
    Map<String, String>? vars,
    String? source,
  }) async {
    return _client.post('/v1/functions', data: {
      'name': name,
      'runtime': runtime,
      'entrypoint': entrypoint,
      'timeout': timeout,
      if (vars != null) 'vars': vars,
      if (source != null) 'source': source,
    });
  }

  /// List all functions.
  Future<Map<String, dynamic>> list() async {
    return _client.get('/v1/functions');
  }

  /// Get a function by ID.
  Future<Map<String, dynamic>> get(String functionId) async {
    return _client.get('/v1/functions/$functionId');
  }

  /// Update a function.
  Future<Map<String, dynamic>> update(
      String functionId, Map<String, dynamic> data) async {
    return _client.put('/v1/functions/$functionId', data: data);
  }

  /// Delete a function.
  Future<Map<String, dynamic>> delete(String functionId) async {
    return _client.delete('/v1/functions/$functionId');
  }

  /// Execute a function.
  Future<Map<String, dynamic>> execute(String functionId,
      {Map<String, dynamic>? data}) async {
    return _client.post('/v1/functions/$functionId/executions',
        data: {'data': data ?? {}});
  }

  /// List executions for a function.
  Future<Map<String, dynamic>> listExecutions(String functionId) async {
    return _client.get('/v1/functions/$functionId/executions');
  }

  /// Get a single execution.
  Future<Map<String, dynamic>> getExecution(
      String functionId, String executionId) async {
    return _client.get('/v1/functions/$functionId/executions/$executionId');
  }
}
