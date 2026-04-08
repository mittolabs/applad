import 'package:dio/dio.dart';

class Search {
  final Dio _dio;

  Search(this._dio);

  Future<Map<String, dynamic>> createIndex({
    required String indexId,
    String? name,
    List<String>? attributes,
  }) async {
    final res = await _dio.post('/v1/search/indexes', data: {
      'indexId': indexId,
      if (name != null) 'name': name,
      if (attributes != null) 'attributes': attributes,
    });
    return res.data;
  }

  Future<Map<String, dynamic>> listIndexes() async {
    final res = await _dio.get('/v1/search/indexes');
    return res.data;
  }

  Future<Map<String, dynamic>> getIndex(String indexId) async {
    final res = await _dio.get('/v1/search/indexes/$indexId');
    return res.data;
  }

  Future<void> deleteIndex(String indexId) async {
    await _dio.delete('/v1/search/indexes/$indexId');
  }

  Future<Map<String, dynamic>> indexDocument({
    required String indexId,
    required String documentId,
    required Map<String, dynamic> data,
  }) async {
    final res = await _dio.post('/v1/search/indexes/$indexId/documents', data: {
      'documentId': documentId,
      'data': data,
    });
    return res.data;
  }

  Future<Map<String, dynamic>> query({
    required String indexId,
    required String query,
    int? limit,
    int? offset,
    Map<String, dynamic>? filters,
  }) async {
    final res = await _dio.post('/v1/search/indexes/$indexId/search', data: {
      'query': query,
      if (limit != null) 'limit': limit,
      if (offset != null) 'offset': offset,
      if (filters != null) 'filters': filters,
    });
    return res.data;
  }

  Future<void> deleteDocument({
    required String indexId,
    required String documentId,
  }) async {
    await _dio.delete('/v1/search/indexes/$indexId/documents/$documentId');
  }
}
