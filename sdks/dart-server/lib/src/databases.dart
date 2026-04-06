import 'client.dart';

/// Databases service — manage databases, collections, attributes, indexes, and documents.
class Databases {
  final ApplAdServer _client;

  Databases(this._client);

  // --- Databases ---

  /// Create a new database.
  Future<Map<String, dynamic>> createDatabase({
    required String name,
    String? databaseId,
  }) async {
    return _client.post('/v1/databases', data: {
      'name': name,
      'databaseId': databaseId ?? 'unique()',
    });
  }

  /// List all databases.
  Future<Map<String, dynamic>> listDatabases() async {
    return _client.get('/v1/databases');
  }

  /// Get a database by ID.
  Future<Map<String, dynamic>> getDatabase(String databaseId) async {
    return _client.get('/v1/databases/$databaseId');
  }

  /// Update a database.
  Future<Map<String, dynamic>> updateDatabase(
      String databaseId, String name) async {
    return _client.put('/v1/databases/$databaseId', data: {
      'name': name,
    });
  }

  /// Delete a database.
  Future<Map<String, dynamic>> deleteDatabase(String databaseId) async {
    return _client.delete('/v1/databases/$databaseId');
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
    return _client.post('/v1/databases/$databaseId/collections', data: {
      'name': name,
      'collectionId': collectionId ?? 'unique()',
      'permissions': permissions ?? [],
      'documentSecurity': documentSecurity,
    });
  }

  /// List collections in a database.
  Future<Map<String, dynamic>> listCollections(String databaseId) async {
    return _client.get('/v1/databases/$databaseId/collections');
  }

  /// Get a collection.
  Future<Map<String, dynamic>> getCollection(
      String databaseId, String collectionId) async {
    return _client.get('/v1/databases/$databaseId/collections/$collectionId');
  }

  /// Update a collection.
  Future<Map<String, dynamic>> updateCollection({
    required String databaseId,
    required String collectionId,
    required String name,
    List<String>? permissions,
    bool? enabled,
  }) async {
    return _client.put(
      '/v1/databases/$databaseId/collections/$collectionId',
      data: {
        'name': name,
        if (permissions != null) 'permissions': permissions,
        if (enabled != null) 'enabled': enabled,
      },
    );
  }

  /// Delete a collection.
  Future<Map<String, dynamic>> deleteCollection(
      String databaseId, String collectionId) async {
    return _client
        .delete('/v1/databases/$databaseId/collections/$collectionId');
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
    return _client.post(
      '/v1/databases/$databaseId/collections/$collectionId/attributes/string',
      data: {
        'key': key,
        'required': required_,
        'array': array,
        if (size != null) 'size': size,
        if (defaultValue != null) 'default': defaultValue,
      },
    );
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
    return _client.post(
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
    return _client.post(
      '/v1/databases/$databaseId/collections/$collectionId/attributes/boolean',
      data: {
        'key': key,
        'required': required_,
        'array': array,
        if (defaultValue != null) 'default': defaultValue,
      },
    );
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
    return _client.post(
      '/v1/databases/$databaseId/collections/$collectionId/attributes/enum',
      data: {
        'key': key,
        'required': required_,
        'array': array,
        'elements': elements,
        if (defaultValue != null) 'default': defaultValue,
      },
    );
  }

  /// List attributes of a collection.
  Future<Map<String, dynamic>> listAttributes(
      String databaseId, String collectionId) async {
    return _client.get(
        '/v1/databases/$databaseId/collections/$collectionId/attributes');
  }

  /// Delete an attribute.
  Future<Map<String, dynamic>> deleteAttribute(
      String databaseId, String collectionId, String key) async {
    return _client.delete(
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
    return _client.post(
      '/v1/databases/$databaseId/collections/$collectionId/indexes',
      data: {
        'key': key,
        'type': type,
        'attributes': attributes,
        if (orders != null) 'orders': orders,
      },
    );
  }

  /// List indexes of a collection.
  Future<Map<String, dynamic>> listIndexes(
      String databaseId, String collectionId) async {
    return _client.get(
        '/v1/databases/$databaseId/collections/$collectionId/indexes');
  }

  /// Delete an index.
  Future<Map<String, dynamic>> deleteIndex(
      String databaseId, String collectionId, String key) async {
    return _client.delete(
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
    return _client.post(
      '/v1/databases/$databaseId/collections/$collectionId/documents',
      data: {
        'documentId': documentId ?? 'unique()',
        'data': data,
        'permissions': permissions ?? [],
      },
    );
  }

  /// List documents in a collection.
  Future<Map<String, dynamic>> listDocuments({
    required String databaseId,
    required String collectionId,
    int? limit,
    int? offset,
  }) async {
    final query = <String, String>{};
    if (limit != null) query['limit'] = limit.toString();
    if (offset != null) query['offset'] = offset.toString();
    final qs = query.isNotEmpty
        ? '?${query.entries.map((e) => '${e.key}=${Uri.encodeComponent(e.value)}').join('&')}'
        : '';
    return _client.get(
        '/v1/databases/$databaseId/collections/$collectionId/documents$qs');
  }

  /// Get a document by ID.
  Future<Map<String, dynamic>> getDocument({
    required String databaseId,
    required String collectionId,
    required String documentId,
  }) async {
    return _client.get(
        '/v1/databases/$databaseId/collections/$collectionId/documents/$documentId');
  }

  /// Update a document.
  Future<Map<String, dynamic>> updateDocument({
    required String databaseId,
    required String collectionId,
    required String documentId,
    Map<String, dynamic>? data,
    List<String>? permissions,
  }) async {
    return _client.patch(
      '/v1/databases/$databaseId/collections/$collectionId/documents/$documentId',
      data: {
        if (data != null) 'data': data,
        if (permissions != null) 'permissions': permissions,
      },
    );
  }

  /// Delete a document.
  Future<Map<String, dynamic>> deleteDocument({
    required String databaseId,
    required String collectionId,
    required String documentId,
  }) async {
    return _client.delete(
        '/v1/databases/$databaseId/collections/$collectionId/documents/$documentId');
  }
}
