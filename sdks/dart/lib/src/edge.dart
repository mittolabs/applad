import 'package:dio/dio.dart';

class Edge {
  final Dio _dio;

  Edge(this._dio);

  Future<Map<String, dynamic>> create({
    required String name,
    required String code,
    String? route,
    int? timeout,
  }) async {
    final res = await _dio.post('/v1/edge/functions', data: {
      'name': name,
      'code': code,
      if (route != null) 'route': route,
      if (timeout != null) 'timeout': timeout,
    });
    return res.data;
  }

  Future<Map<String, dynamic>> list() async {
    final res = await _dio.get('/v1/edge/functions');
    return res.data;
  }

  Future<Map<String, dynamic>> get(String functionId) async {
    final res = await _dio.get('/v1/edge/functions/$functionId');
    return res.data;
  }

  Future<Map<String, dynamic>> update(
    String functionId, {
    String? name,
    String? code,
    String? route,
    int? timeout,
  }) async {
    final res = await _dio.put('/v1/edge/functions/$functionId', data: {
      if (name != null) 'name': name,
      if (code != null) 'code': code,
      if (route != null) 'route': route,
      if (timeout != null) 'timeout': timeout,
    });
    return res.data;
  }

  Future<void> delete(String functionId) async {
    await _dio.delete('/v1/edge/functions/$functionId');
  }

  Future<Map<String, dynamic>> invoke(
    String functionId, {
    Map<String, dynamic>? data,
  }) async {
    final res = await _dio.post('/v1/edge/functions/$functionId/invoke',
        data: data ?? {});
    return res.data;
  }

  Future<Map<String, dynamic>> listExecutions(String functionId) async {
    final res = await _dio.get('/v1/edge/functions/$functionId/executions');
    return res.data;
  }
}
