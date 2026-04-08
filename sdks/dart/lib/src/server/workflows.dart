import '../server_client.dart';

/// Workflows service (server-side).
class Workflows {
  final ApplAdServer _client;

  Workflows(this._client);

  Future<Map<String, dynamic>> create({
    required String name,
    String? description,
    String triggerType = 'manual',
    Map<String, dynamic>? triggerConfig,
    List<Map<String, dynamic>>? nodes,
    List<Map<String, dynamic>>? edges,
  }) async {
    return _client.post('/v1/workflows', data: {
      'name': name,
      if (description != null) 'description': description,
      'triggerType': triggerType,
      if (triggerConfig != null) 'triggerConfig': triggerConfig,
      if (nodes != null) 'nodes': nodes,
      if (edges != null) 'edges': edges,
    });
  }

  Future<Map<String, dynamic>> list() async {
    return _client.get('/v1/workflows');
  }

  Future<Map<String, dynamic>> get(String workflowId) async {
    return _client.get('/v1/workflows/$workflowId');
  }

  Future<Map<String, dynamic>> update(
      String workflowId, Map<String, dynamic> data) async {
    return _client.put('/v1/workflows/$workflowId', data: data);
  }

  Future<Map<String, dynamic>> delete(String workflowId) async {
    return _client.delete('/v1/workflows/$workflowId');
  }

  Future<Map<String, dynamic>> execute(
    String workflowId, {
    Map<String, dynamic>? triggerData,
  }) async {
    return _client.post('/v1/workflows/$workflowId/execute', data: {
      if (triggerData != null) 'triggerData': triggerData,
    });
  }

  Future<Map<String, dynamic>> listExecutions(String workflowId) async {
    return _client.get('/v1/workflows/$workflowId/executions');
  }

  Future<Map<String, dynamic>> getExecution(
      String workflowId, String executionId) async {
    return _client.get('/v1/workflows/$workflowId/executions/$executionId');
  }
}
