import '../server_client.dart';

class Vectors {
  final ApplAdServer _client;

  Vectors(this._client);

  Future<Map<String, dynamic>> createIndex({
    required String indexId,
    required int dimensions,
    String? metric,
    String? name,
  }) async {
    return _client.post('/v1/vectors/indexes', data: {
      'indexId': indexId,
      'dimensions': dimensions,
      if (metric != null) 'metric': metric,
      if (name != null) 'name': name,
    });
  }

  Future<Map<String, dynamic>> listIndexes() async {
    return _client.get('/v1/vectors/indexes');
  }

  Future<Map<String, dynamic>> getIndex(String indexId) async {
    return _client.get('/v1/vectors/indexes/$indexId');
  }

  Future<Map<String, dynamic>> deleteIndex(String indexId) async {
    return _client.delete('/v1/vectors/indexes/$indexId');
  }

  Future<Map<String, dynamic>> upsert({
    required String indexId,
    required List<Map<String, dynamic>> vectors,
  }) async {
    return _client.post('/v1/vectors/indexes/$indexId/vectors', data: {
      'vectors': vectors,
    });
  }

  Future<Map<String, dynamic>> query({
    required String indexId,
    required List<double> vector,
    int? topK,
    Map<String, dynamic>? filter,
  }) async {
    return _client.post('/v1/vectors/indexes/$indexId/query', data: {
      'vector': vector,
      if (topK != null) 'topK': topK,
      if (filter != null) 'filter': filter,
    });
  }

  Future<Map<String, dynamic>> deleteVectors({
    required String indexId,
    required List<String> ids,
  }) async {
    return _client.post('/v1/vectors/indexes/$indexId/delete', data: {
      'ids': ids,
    });
  }
}
