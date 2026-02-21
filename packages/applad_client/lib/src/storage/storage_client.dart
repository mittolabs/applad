library;

import 'dart:typed_data';
import '../applad_client.dart';

/// Client for Applad storage buckets.
final class StorageClient {
  StorageClient({required this.client});

  final ApplAdClient client;

  /// Access a specific bucket.
  BucketClient from(String bucket) => BucketClient(client: client, bucket: bucket);
}

/// Client for a specific storage bucket.
final class BucketClient {
  BucketClient({required this.client, required this.bucket});

  final ApplAdClient client;
  final String bucket;

  /// Upload a file to this bucket.
  Future<Map<String, dynamic>> upload(
    String path,
    Uint8List data, {
    String? contentType,
  }) async {
    throw UnimplementedError('Storage upload — available in Phase 2');
  }

  /// Download a file from this bucket.
  Future<Uint8List> download(String path) async {
    throw UnimplementedError('Storage download — available in Phase 2');
  }

  /// Get the public URL for a file.
  String getPublicUrl(String path) {
    return '${client.endpoint}/v1/storage/buckets/$bucket/objects/$path';
  }

  /// Delete a file from this bucket.
  Future<void> delete(String path) async {
    throw UnimplementedError('Storage delete — available in Phase 2');
  }

  /// List files in a directory.
  Future<List<Map<String, dynamic>>> list([String? prefix]) async {
    throw UnimplementedError('Storage list — available in Phase 2');
  }
}
