import '../server_client.dart';

class Search {
  final ApplAdServer _client;

  Search(this._client);

  Future<Map<String, dynamic>> createIndex({
    required String indexId,
    String? name,
    List<String>? attributes,
  }) async {
    return _client.post('/v1/search/indexes', data: {
      'indexId': indexId,
      if (name != null) 'name': name,
      if (attributes != null) 'attributes': attributes,
    });
  }

  Future<Map<String, dynamic>> listIndexes() async {
    return _client.get('/v1/search/indexes');
  }

  Future<Map<String, dynamic>> getIndex(String indexId) async {
    return _client.get('/v1/search/indexes/$indexId');
  }

  Future<Map<String, dynamic>> deleteIndex(String indexId) async {
    return _client.delete('/v1/search/indexes/$indexId');
  }

  Future<Map<String, dynamic>> indexDocument({
    required String indexId,
    required String documentId,
    required Map<String, dynamic> data,
  }) async {
    return _client.post('/v1/search/indexes/$indexId/documents', data: {
      'documentId': documentId,
      'data': data,
    });
  }

  Future<Map<String, dynamic>> query({
    required String indexId,
    required String query,
    int? limit,
    int? offset,
    Map<String, dynamic>? filters,
  }) async {
    return _client.post('/v1/search/indexes/$indexId/search', data: {
      'query': query,
      if (limit != null) 'limit': limit,
      if (offset != null) 'offset': offset,
      if (filters != null) 'filters': filters,
    });
  }

  Future<Map<String, dynamic>> deleteDocument({
    required String indexId,
    required String documentId,
  }) async {
    return _client.delete('/v1/search/indexes/$indexId/documents/$documentId');
  }
}
