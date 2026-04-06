import 'package:dio/dio.dart';

/// Databases service — manage databases, collections, attributes, indexes, and documents.
class Databases {
  final Dio _dio;

  Databases(this._dio);

  // --- Databases ---

  /// Create a new database.
  Future<Map<String, dynamic>> createDatabase({
    required String name,
    String? databaseId,
  }) async {
    final res = await _dio.post('/v1/databases', data: {
      'name': name,
      'databaseId': databaseId ?? 'unique()',
    });
    return res.data;
  }

  /// List all databases.
  Future<Map<String, dynamic>> listDatabases() async {
    final res = await _dio.get('/v1/databases');
    return res.data;
  }

  /// Get a database by ID.
  Future<Map<String, dynamic>> getDatabase(String databaseId) async {
    final res = await _dio.get('/v1/databases/$databaseId');
    return res.data;
  }

  /// Update a database.
  Future<Map<String, dynamic>> updateDatabase(
      String databaseId, String name) async {
    final res = await _dio.put('/v1/databases/$databaseId', data: {
      'name': name,
    });
    return res.data;
  }

  /// Delete a database.
  Future<void> deleteDatabase(String databaseId) async {
    await _dio.delete('/v1/databases/$databaseId');
  }

  // --- Collections ---

  /// Create a collection.
  Future<Map<String, dynamic>> createCollection({
    required String databaseId,
    required String name,
    String? collectionId,
    List<String>? permissions,
    bool documentSecurity = false,
  }) async {
    final res = await _dio.post(
      '/v1/databases/$databaseId/collections',
      data: {
        'name': name,
        'collectionId': collectionId ?? 'unique()',
        'permissions': permissions ?? [],
        'documentSecurity': documentSecurity,
      },
    );
    return res.data;
  }

  /// List collections in a database.
  Future<Map<String, dynamic>> listCollections(String databaseId) async {
    final res = await _dio.get('/v1/databases/$databaseId/collections');
    return res.data;
  }

  /// Get a collection.
  Future<Map<String, dynamic>> getCollection(
      String databaseId, String collectionId) async {
    final res = await _dio.get(
        '/v1/databases/$databaseId/collections/$collectionId');
    return res.data;
  }

  /// Update a collection.
  Future<Map<String, dynamic>> updateCollection({
    required String databaseId,
    required String collectionId,
    required String name,
    List<String>? permissions,
    bool? enabled,
  }) async {
    final res = await _dio.put(
      '/v1/databases/$databaseId/collections/$collectionId',
      data: {
        'name': name,
        if (permissions != null) 'permissions': permissions,
        if (enabled != null) 'enabled': enabled,
      },
    );
    return res.data;
  }

  /// Delete a collection.
  Future<void> deleteCollection(
      String databaseId, String collectionId) async {
    await _dio.delete(
        '/v1/databases/$databaseId/collections/$collectionId');
  }

  // --- Attributes ---

  /// Create a string attribute.
  Future<Map<String, dynamic>> createStringAttribute({
    required String databaseId,
    required String collectionId,
    required String key,
    bool required_ = false,
    int? size,
    String? defaultValue,
    bool array = false,
  }) async {
    final res = await _dio.post(
      '/v1/databases/$databaseId/collections/$collectionId/attributes/string',
      data: {
        'key': key,
        'required': required_,
        'array': array,
        if (size != null) 'size': size,
        if (defaultValue != null) 'default': defaultValue,
      },
    );
    return res.data;
  }

  /// Create an integer attribute.
  Future<Map<String, dynamic>> createIntegerAttribute({
    required String databaseId,
    required String collectionId,
    required String key,
    bool required_ = false,
    num? min,
    num? max,
    int? defaultValue,
    bool array = false,
  }) async {
    final res = await _dio.post(
      '/v1/databases/$databaseId/collections/$collectionId/attributes/integer',
      data: {
        'key': key,
        'required': required_,
        'array': array,
        if (min != null) 'min': min,
        if (max != null) 'max': max,
        if (defaultValue != null) 'default': defaultValue,
      },
    );
    return res.data;
  }

  /// Create a boolean attribute.
  Future<Map<String, dynamic>> createBooleanAttribute({
    required String databaseId,
    required String collectionId,
    required String key,
    bool required_ = false,
    bool? defaultValue,
    bool array = false,
  }) async {
    final res = await _dio.post(
      '/v1/databases/$databaseId/collections/$collectionId/attributes/boolean',
      data: {
        'key': key,
        'required': required_,
        'array': array,
        if (defaultValue != null) 'default': defaultValue,
      },
    );
    return res.data;
  }

  /// Create an enum attribute.
  Future<Map<String, dynamic>> createEnumAttribute({
    required String databaseId,
    required String collectionId,
    required String key,
    required List<String> elements,
    bool required_ = false,
    String? defaultValue,
    bool array = false,
  }) async {
    final res = await _dio.post(
      '/v1/databases/$databaseId/collections/$collectionId/attributes/enum',
      data: {
        'key': key,
        'required': required_,
        'array': array,
        'elements': elements,
        if (defaultValue != null) 'default': defaultValue,
      },
    );
    return res.data;
  }

  /// List attributes of a collection.
  Future<Map<String, dynamic>> listAttributes(
      String databaseId, String collectionId) async {
    final res = await _dio.get(
        '/v1/databases/$databaseId/collections/$collectionId/attributes');
    return res.data;
  }

  /// Delete an attribute.
  Future<void> deleteAttribute(
      String databaseId, String collectionId, String key) async {
    await _dio.delete(
        '/v1/databases/$databaseId/collections/$collectionId/attributes/$key');
  }

  // --- Indexes ---

  /// Create an index.
  Future<Map<String, dynamic>> createIndex({
    required String databaseId,
    required String collectionId,
    required String key,
    required String type,
    required List<String> attributes,
    List<String>? orders,
  }) async {
    final res = await _dio.post(
      '/v1/databases/$databaseId/collections/$collectionId/indexes',
      data: {
        'key': key,
        'type': type,
        'attributes': attributes,
        if (orders != null) 'orders': orders,
      },
    );
    return res.data;
  }

  /// List indexes of a collection.
  Future<Map<String, dynamic>> listIndexes(
      String databaseId, String collectionId) async {
    final res = await _dio.get(
        '/v1/databases/$databaseId/collections/$collectionId/indexes');
    return res.data;
  }

  /// Delete an index.
  Future<void> deleteIndex(
      String databaseId, String collectionId, String key) async {
    await _dio.delete(
        '/v1/databases/$databaseId/collections/$collectionId/indexes/$key');
  }

  // --- Documents ---

  /// Create a document.
  Future<Map<String, dynamic>> createDocument({
    required String databaseId,
    required String collectionId,
    required Map<String, dynamic> data,
    String? documentId,
    List<String>? permissions,
  }) async {
    final res = await _dio.post(
      '/v1/databases/$databaseId/collections/$collectionId/documents',
      data: {
        'documentId': documentId ?? 'unique()',
        'data': data,
        'permissions': permissions ?? [],
      },
    );
    return res.data;
  }

  /// List documents in a collection.
  Future<Map<String, dynamic>> listDocuments({
    required String databaseId,
    required String collectionId,
    int? limit,
    int? offset,
  }) async {
    final res = await _dio.get(
      '/v1/databases/$databaseId/collections/$collectionId/documents',
      queryParameters: {
        if (limit != null) 'limit': limit,
        if (offset != null) 'offset': offset,
      },
    );
    return res.data;
  }

  /// Get a document by ID.
  Future<Map<String, dynamic>> getDocument({
    required String databaseId,
    required String collectionId,
    required String documentId,
  }) async {
    final res = await _dio.get(
        '/v1/databases/$databaseId/collections/$collectionId/documents/$documentId');
    return res.data;
  }

  /// Update a document.
  Future<Map<String, dynamic>> updateDocument({
    required String databaseId,
    required String collectionId,
    required String documentId,
    Map<String, dynamic>? data,
    List<String>? permissions,
  }) async {
    final res = await _dio.patch(
      '/v1/databases/$databaseId/collections/$collectionId/documents/$documentId',
      data: {
        if (data != null) 'data': data,
        if (permissions != null) 'permissions': permissions,
      },
    );
    return res.data;
  }

  /// Delete a document.
  Future<void> deleteDocument({
    required String databaseId,
    required String collectionId,
    required String documentId,
  }) async {
    await _dio.delete(
        '/v1/databases/$databaseId/collections/$collectionId/documents/$documentId');
  }
}
