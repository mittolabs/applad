import 'package:dio/dio.dart';

/// Workflows service — manage workflow definitions and executions.
class Workflows {
  final Dio _dio;

  Workflows(this._dio);

  /// Create a workflow.
  Future<Map<String, dynamic>> create({
    required String name,
    String? description,
    String triggerType = 'manual',
    Map<String, dynamic>? triggerConfig,
    List<Map<String, dynamic>>? nodes,
    List<Map<String, dynamic>>? edges,
  }) async {
    final res = await _dio.post('/v1/workflows', data: {
      'name': name,
      if (description != null) 'description': description,
      'triggerType': triggerType,
      if (triggerConfig != null) 'triggerConfig': triggerConfig,
      if (nodes != null) 'nodes': nodes,
      if (edges != null) 'edges': edges,
    });
    return res.data;
  }

  /// List workflows.
  Future<Map<String, dynamic>> list() async {
    final res = await _dio.get('/v1/workflows');
    return res.data;
  }

  /// Get a workflow by ID.
  Future<Map<String, dynamic>> get(String workflowId) async {
    final res = await _dio.get('/v1/workflows/$workflowId');
    return res.data;
  }

  /// Update a workflow.
  Future<Map<String, dynamic>> update(
    String workflowId,
    Map<String, dynamic> data,
  ) async {
    final res = await _dio.put('/v1/workflows/$workflowId', data: data);
    return res.data;
  }

  /// Delete a workflow.
  Future<void> delete(String workflowId) async {
    await _dio.delete('/v1/workflows/$workflowId');
  }

  /// Execute a workflow.
  Future<Map<String, dynamic>> execute(
    String workflowId, {
    Map<String, dynamic>? triggerData,
  }) async {
    final res = await _dio.post('/v1/workflows/$workflowId/execute', data: {
      if (triggerData != null) 'triggerData': triggerData,
    });
    return res.data;
  }

  /// List executions for a workflow.
  Future<Map<String, dynamic>> listExecutions(String workflowId) async {
    final res = await _dio.get('/v1/workflows/$workflowId/executions');
    return res.data;
  }

  /// Get a single execution.
  Future<Map<String, dynamic>> getExecution(
    String workflowId,
    String executionId,
  ) async {
    final res = await _dio
        .get('/v1/workflows/$workflowId/executions/$executionId');
    return res.data;
  }
}
