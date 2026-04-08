import 'package:dio/dio.dart';

/// Functions service — manage serverless functions and executions.
class Functions {
  final Dio _dio;

  Functions(this._dio);

  Future<Map<String, dynamic>> create({
    required String name,
    required String runtime,
    String entrypoint = 'index.handler',
    int timeout = 15,
    Map<String, String>? vars,
    String? source,
  }) async {
    final res = await _dio.post('/v1/functions', data: {
      'name': name,
      'runtime': runtime,
      'entrypoint': entrypoint,
      'timeout': timeout,
      if (vars != null) 'vars': vars,
      if (source != null) 'source': source,
    });
    return res.data;
  }

  Future<Map<String, dynamic>> list() async {
    final res = await _dio.get('/v1/functions');
    return res.data;
  }

  Future<Map<String, dynamic>> get(String functionId) async {
    final res = await _dio.get('/v1/functions/$functionId');
    return res.data;
  }

  Future<Map<String, dynamic>> update(
      String functionId, Map<String, dynamic> data) async {
    final res = await _dio.put('/v1/functions/$functionId', data: data);
    return res.data;
  }

  Future<void> delete(String functionId) async {
    await _dio.delete('/v1/functions/$functionId');
  }

  Future<Map<String, dynamic>> execute(String functionId,
      {Map<String, dynamic>? data}) async {
    final res = await _dio.post('/v1/functions/$functionId/executions',
        data: {'data': data ?? {}});
    return res.data;
  }

  Future<Map<String, dynamic>> listExecutions(String functionId) async {
    final res = await _dio.get('/v1/functions/$functionId/executions');
    return res.data;
  }

  Future<Map<String, dynamic>> getExecution(
      String functionId, String executionId) async {
    final res =
        await _dio.get('/v1/functions/$functionId/executions/$executionId');
    return res.data;
  }
}
