import 'package:dio/dio.dart';

class Vectors {
  final Dio _dio;

  Vectors(this._dio);

  Future<Map<String, dynamic>> createIndex({
    required String indexId,
    required int dimensions,
    String? metric,
    String? name,
  }) async {
    final res = await _dio.post('/v1/vectors/indexes', data: {
      'indexId': indexId,
      'dimensions': dimensions,
      if (metric != null) 'metric': metric,
      if (name != null) 'name': name,
    });
    return res.data;
  }

  Future<Map<String, dynamic>> listIndexes() async {
    final res = await _dio.get('/v1/vectors/indexes');
    return res.data;
  }

  Future<Map<String, dynamic>> getIndex(String indexId) async {
    final res = await _dio.get('/v1/vectors/indexes/$indexId');
    return res.data;
  }

  Future<void> deleteIndex(String indexId) async {
    await _dio.delete('/v1/vectors/indexes/$indexId');
  }

  Future<Map<String, dynamic>> upsert({
    required String indexId,
    required List<Map<String, dynamic>> vectors,
  }) async {
    final res = await _dio.post('/v1/vectors/indexes/$indexId/vectors', data: {
      'vectors': vectors,
    });
    return res.data;
  }

  Future<Map<String, dynamic>> query({
    required String indexId,
    required List<double> vector,
    int? topK,
    Map<String, dynamic>? filter,
  }) async {
    final res = await _dio.post('/v1/vectors/indexes/$indexId/query', data: {
      'vector': vector,
      if (topK != null) 'topK': topK,
      if (filter != null) 'filter': filter,
    });
    return res.data;
  }

  Future<Map<String, dynamic>> deleteVectors({
    required String indexId,
    required List<String> ids,
  }) async {
    final res = await _dio.post('/v1/vectors/indexes/$indexId/delete', data: {
      'ids': ids,
    });
    return res.data;
  }
}
