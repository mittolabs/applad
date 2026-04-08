import '../server_client.dart';

/// Storage service — manage buckets and files (server-side).
class Storage {
  final ApplAdServer _client;

  Storage(this._client);

  Future<Map<String, dynamic>> createBucket({
    required String name,
    String? bucketId,
    List<String>? permissions,
    int? maximumFileSize,
    List<String>? allowedFileExtensions,
    String? compression,
    bool encryption = false,
    bool antivirus = false,
  }) async {
    return _client.post('/v1/storage/buckets', data: {
      'name': name,
      'bucketId': bucketId ?? 'unique()',
      'permissions': permissions ?? [],
      if (maximumFileSize != null) 'maximumFileSize': maximumFileSize,
      'allowedFileExtensions': allowedFileExtensions ?? [],
      if (compression != null) 'compression': compression,
      'encryption': encryption,
      'antivirus': antivirus,
    });
  }

  Future<Map<String, dynamic>> listBuckets() async {
    return _client.get('/v1/storage/buckets');
  }

  Future<Map<String, dynamic>> getBucket(String bucketId) async {
    return _client.get('/v1/storage/buckets/$bucketId');
  }

  Future<Map<String, dynamic>> updateBucket({
    required String bucketId,
    required String name,
    List<String>? permissions,
    int? maximumFileSize,
    bool? enabled,
  }) async {
    return _client.put('/v1/storage/buckets/$bucketId', data: {
      'name': name,
      if (permissions != null) 'permissions': permissions,
      if (maximumFileSize != null) 'maximumFileSize': maximumFileSize,
      if (enabled != null) 'enabled': enabled,
    });
  }

  Future<Map<String, dynamic>> deleteBucket(String bucketId) async {
    return _client.delete('/v1/storage/buckets/$bucketId');
  }

  Future<Map<String, dynamic>> createFile({
    required String bucketId,
    required String fileName,
    String? fileId,
    List<String>? permissions,
  }) async {
    return _client.post('/v1/storage/buckets/$bucketId/files', data: {
      'fileId': fileId ?? 'unique()',
      'fileName': fileName,
      if (permissions != null) 'permissions': permissions,
    });
  }

  Future<Map<String, dynamic>> listFiles({
    required String bucketId,
    int? limit,
    int? offset,
  }) async {
    final query = <String, String>{};
    if (limit != null) query['limit'] = limit.toString();
    if (offset != null) query['offset'] = offset.toString();
    final qs = query.isNotEmpty
        ? '?${query.entries.map((e) => '${e.key}=${Uri.encodeComponent(e.value)}').join('&')}'
        : '';
    return _client.get('/v1/storage/buckets/$bucketId/files$qs');
  }

  Future<Map<String, dynamic>> getFile({
    required String bucketId,
    required String fileId,
  }) async {
    return _client.get('/v1/storage/buckets/$bucketId/files/$fileId');
  }

  Future<Map<String, dynamic>> deleteFile({
    required String bucketId,
    required String fileId,
  }) async {
    return _client.delete('/v1/storage/buckets/$bucketId/files/$fileId');
  }
}
