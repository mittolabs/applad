import 'package:dio/dio.dart';

/// Storage service — manage buckets and files.
class Storage {
  final Dio _dio;

  Storage(this._dio);

  // --- Buckets ---

  /// Create a storage bucket.
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
    final res = await _dio.post('/v1/storage/buckets', data: {
      'name': name,
      'bucketId': bucketId ?? 'unique()',
      'permissions': permissions ?? [],
      if (maximumFileSize != null) 'maximumFileSize': maximumFileSize,
      'allowedFileExtensions': allowedFileExtensions ?? [],
      if (compression != null) 'compression': compression,
      'encryption': encryption,
      'antivirus': antivirus,
    });
    return res.data;
  }

  /// List all buckets.
  Future<Map<String, dynamic>> listBuckets() async {
    final res = await _dio.get('/v1/storage/buckets');
    return res.data;
  }

  /// Get a bucket by ID.
  Future<Map<String, dynamic>> getBucket(String bucketId) async {
    final res = await _dio.get('/v1/storage/buckets/$bucketId');
    return res.data;
  }

  /// Update a bucket.
  Future<Map<String, dynamic>> updateBucket({
    required String bucketId,
    required String name,
    List<String>? permissions,
    int? maximumFileSize,
    bool? enabled,
  }) async {
    final res = await _dio.put('/v1/storage/buckets/$bucketId', data: {
      'name': name,
      if (permissions != null) 'permissions': permissions,
      if (maximumFileSize != null) 'maximumFileSize': maximumFileSize,
      if (enabled != null) 'enabled': enabled,
    });
    return res.data;
  }

  /// Delete a bucket.
  Future<void> deleteBucket(String bucketId) async {
    await _dio.delete('/v1/storage/buckets/$bucketId');
  }

  // --- Files ---

  /// Upload a file to a bucket.
  Future<Map<String, dynamic>> createFile({
    required String bucketId,
    required String fileName,
    required List<int> fileBytes,
    String? fileId,
    String? mimeType,
    List<String>? permissions,
  }) async {
    final formData = FormData.fromMap({
      'fileId': fileId ?? 'unique()',
      'file': MultipartFile.fromBytes(
        fileBytes,
        filename: fileName,
        contentType: mimeType != null ? DioMediaType.parse(mimeType) : null,
      ),
      if (permissions != null) 'permissions': permissions,
    });

    final res = await _dio.post(
      '/v1/storage/buckets/$bucketId/files',
      data: formData,
    );
    return res.data;
  }

  /// List files in a bucket.
  Future<Map<String, dynamic>> listFiles({
    required String bucketId,
    int? limit,
    int? offset,
  }) async {
    final res = await _dio.get(
      '/v1/storage/buckets/$bucketId/files',
      queryParameters: {
        if (limit != null) 'limit': limit,
        if (offset != null) 'offset': offset,
      },
    );
    return res.data;
  }

  /// The URL a stored file is served from.
  ///
  /// [downloadFile] and [viewFile] return bytes, which is the wrong shape for
  /// an image in a list: a feed hands a URL to its image cache and lets that
  /// decide what to fetch and when. This builds the URL without a round trip.
  ///
  /// Whether it resolves for an unauthenticated viewer is decided by the bucket
  /// and file permissions — a bucket that grants `read("any")` serves publicly.
  String fileViewUrl({
    required String bucketId,
    required String fileId,
    String? projectId,
  }) =>
      _fileUrl(bucketId, fileId, 'view', projectId);

  /// The URL a stored file downloads from, as an attachment.
  String fileDownloadUrl({
    required String bucketId,
    required String fileId,
    String? projectId,
  }) =>
      _fileUrl(bucketId, fileId, 'download', projectId);

  String _fileUrl(
    String bucketId,
    String fileId,
    String action,
    String? projectId,
  ) {
    var base = _dio.options.baseUrl;
    while (base.endsWith('/')) {
      base = base.substring(0, base.length - 1);
    }
    final project =
        projectId ?? _dio.options.headers['X-Applad-Project']?.toString();
    final path = '/v1/storage/buckets/$bucketId/files/$fileId/$action';
    if (project == null || project.isEmpty) return '$base$path';
    return '$base$path?project=${Uri.encodeQueryComponent(project)}';
  }

  /// Get file metadata.
  Future<Map<String, dynamic>> getFile({
    required String bucketId,
    required String fileId,
  }) async {
    final res = await _dio.get('/v1/storage/buckets/$bucketId/files/$fileId');
    return res.data;
  }

  /// Download a file as bytes.
  Future<List<int>> downloadFile({
    required String bucketId,
    required String fileId,
  }) async {
    final res = await _dio.get<List<int>>(
      '/v1/storage/buckets/$bucketId/files/$fileId/download',
      options: Options(responseType: ResponseType.bytes),
    );
    return res.data!;
  }

  /// Get file content for viewing (inline).
  Future<List<int>> viewFile({
    required String bucketId,
    required String fileId,
  }) async {
    final res = await _dio.get<List<int>>(
      '/v1/storage/buckets/$bucketId/files/$fileId/view',
      options: Options(responseType: ResponseType.bytes),
    );
    return res.data!;
  }

  /// Delete a file.
  Future<void> deleteFile({
    required String bucketId,
    required String fileId,
  }) async {
    await _dio.delete('/v1/storage/buckets/$bucketId/files/$fileId');
  }
}
