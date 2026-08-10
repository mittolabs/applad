import 'package:dio/dio.dart';

import 'query_builder.dart';

export 'query_builder.dart' show QueryBuilder, QueryResult;

/// Databases service — manage databases, tables, columns, indexes, and rows.
class Databases {
  final Dio _dio;

  Databases(this._dio);

  // --- Query builder ---

  /// Returns a [QueryBuilder] for fluent row queries on [tableId] inside [databaseId].
  ///
  /// ```dart
  /// final result = await client.databases
  ///     .from('myDb', 'posts')
  ///     .select('id, title, author(name)')
  ///     .equal('published', true)
  ///     .orderDesc('created_at')
  ///     .limit(25)
  ///     .get();
  /// ```
  QueryBuilder from(String databaseId, String tableId) {
    return QueryBuilder((params) async {
      final res = await _dio.get(
        '/v1/databases/$databaseId/tables/$tableId/rows',
        queryParameters: params,
      );
      return Map<String, dynamic>.from(res.data as Map);
    });
  }

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

  // --- Tables ---

  /// Create a table.
  Future<Map<String, dynamic>> createTable({
    required String databaseId,
    required String name,
    String? tableId,
    List<String>? permissions,
    bool documentSecurity = false,
  }) async {
    final res = await _dio.post(
      '/v1/databases/$databaseId/tables',
      data: {
        'name': name,
        'tableId': tableId ?? 'unique()',
        'permissions': permissions ?? [],
        'documentSecurity': documentSecurity,
      },
    );
    return res.data;
  }

  /// List tables in a database.
  Future<Map<String, dynamic>> listTables(String databaseId) async {
    final res = await _dio.get('/v1/databases/$databaseId/tables');
    return res.data;
  }

  /// Get a table.
  Future<Map<String, dynamic>> getTable(
      String databaseId, String tableId) async {
    final res = await _dio.get('/v1/databases/$databaseId/tables/$tableId');
    return res.data;
  }

  /// Update a table.
  Future<Map<String, dynamic>> updateTable({
    required String databaseId,
    required String tableId,
    required String name,
    List<String>? permissions,
    bool? enabled,
  }) async {
    final res = await _dio.put(
      '/v1/databases/$databaseId/tables/$tableId',
      data: {
        'name': name,
        if (permissions != null) 'permissions': permissions,
        if (enabled != null) 'enabled': enabled,
      },
    );
    return res.data;
  }

  /// Delete a table.
  Future<void> deleteTable(String databaseId, String tableId) async {
    await _dio.delete('/v1/databases/$databaseId/tables/$tableId');
  }

  // --- Columns ---

  /// Create a string column.
  ///
  /// Set [encrypted] to store this column's values as opaque ciphertext at
  /// rest (see field-level encryption docs). Cannot be combined with [array],
  /// and requires the instance to have MASTER_ENCRYPTION_KEY configured.
  Future<Map<String, dynamic>> createStringColumn({
    required String databaseId,
    required String tableId,
    required String key,
    bool required_ = false,
    int? size,
    String? defaultValue,
    bool array = false,
    bool encrypted = false,
  }) async {
    final res = await _dio.post(
      '/v1/databases/$databaseId/tables/$tableId/columns/string',
      data: {
        'key': key,
        'required': required_,
        'array': array,
        'encrypted': encrypted,
        if (size != null) 'size': size,
        if (defaultValue != null) 'default': defaultValue,
      },
    );
    return res.data;
  }

  /// Create an integer column.
  Future<Map<String, dynamic>> createIntegerColumn({
    required String databaseId,
    required String tableId,
    required String key,
    bool required_ = false,
    num? min,
    num? max,
    int? defaultValue,
    bool array = false,
    bool encrypted = false,
  }) async {
    final res = await _dio.post(
      '/v1/databases/$databaseId/tables/$tableId/columns/integer',
      data: {
        'key': key,
        'required': required_,
        'array': array,
        'encrypted': encrypted,
        if (min != null) 'min': min,
        if (max != null) 'max': max,
        if (defaultValue != null) 'default': defaultValue,
      },
    );
    return res.data;
  }

  /// Create a boolean column.
  Future<Map<String, dynamic>> createBooleanColumn({
    required String databaseId,
    required String tableId,
    required String key,
    bool required_ = false,
    bool? defaultValue,
    bool array = false,
    bool encrypted = false,
  }) async {
    final res = await _dio.post(
      '/v1/databases/$databaseId/tables/$tableId/columns/boolean',
      data: {
        'key': key,
        'required': required_,
        'array': array,
        'encrypted': encrypted,
        if (defaultValue != null) 'default': defaultValue,
      },
    );
    return res.data;
  }

  /// Create an enum column.
  Future<Map<String, dynamic>> createEnumColumn({
    required String databaseId,
    required String tableId,
    required String key,
    required List<String> elements,
    bool required_ = false,
    String? defaultValue,
    bool array = false,
    bool encrypted = false,
  }) async {
    final res = await _dio.post(
      '/v1/databases/$databaseId/tables/$tableId/columns/enum',
      data: {
        'key': key,
        'required': required_,
        'array': array,
        'encrypted': encrypted,
        'elements': elements,
        if (defaultValue != null) 'default': defaultValue,
      },
    );
    return res.data;
  }

  /// List columns of a table.
  Future<Map<String, dynamic>> listColumns(
      String databaseId, String tableId) async {
    final res =
        await _dio.get('/v1/databases/$databaseId/tables/$tableId/columns');
    return res.data;
  }

  /// Delete a column.
  Future<void> deleteColumn(
      String databaseId, String tableId, String key) async {
    await _dio.delete('/v1/databases/$databaseId/tables/$tableId/columns/$key');
  }

  /// Get field-level permissions for a column.
  Future<Map<String, dynamic>> getColumnPermissions(
      String databaseId, String tableId, String key) async {
    final res = await _dio.get(
        '/v1/databases/$databaseId/tables/$tableId/columns/$key/permissions');
    return res.data;
  }

  /// Set field-level permissions for a column.
  Future<Map<String, dynamic>> setColumnPermissions(
      String databaseId, String tableId, String key, List<String> permissions) async {
    final res = await _dio.post(
      '/v1/databases/$databaseId/tables/$tableId/columns/$key/permissions',
      data: {'permissions': permissions},
    );
    return res.data;
  }

  // --- Indexes ---

  /// Create an index.
  Future<Map<String, dynamic>> createIndex({
    required String databaseId,
    required String tableId,
    required String key,
    required String type,
    required List<String> columns,
    List<String>? orders,
  }) async {
    final res = await _dio.post(
      '/v1/databases/$databaseId/tables/$tableId/indexes',
      data: {
        'key': key,
        'type': type,
        'columns': columns,
        if (orders != null) 'orders': orders,
      },
    );
    return res.data;
  }

  /// List indexes of a table.
  Future<Map<String, dynamic>> listIndexes(
      String databaseId, String tableId) async {
    final res =
        await _dio.get('/v1/databases/$databaseId/tables/$tableId/indexes');
    return res.data;
  }

  /// Delete an index.
  Future<void> deleteIndex(
      String databaseId, String tableId, String key) async {
    await _dio.delete('/v1/databases/$databaseId/tables/$tableId/indexes/$key');
  }

  // --- Rows ---

  /// Create a row.
  Future<Map<String, dynamic>> createRow({
    required String databaseId,
    required String tableId,
    required Map<String, dynamic> data,
    String? rowId,
    List<String>? permissions,
  }) async {
    final res = await _dio.post(
      '/v1/databases/$databaseId/tables/$tableId/rows',
      data: {
        'rowId': rowId ?? 'unique()',
        'data': data,
        'permissions': permissions ?? [],
      },
    );
    return res.data;
  }

  /// List rows in a table.
  Future<Map<String, dynamic>> listRows({
    required String databaseId,
    required String tableId,
    int? limit,
    int? offset,
  }) async {
    final res = await _dio.get(
      '/v1/databases/$databaseId/tables/$tableId/rows',
      queryParameters: {
        if (limit != null) 'limit': limit,
        if (offset != null) 'offset': offset,
      },
    );
    return res.data;
  }

  /// Get a row by ID.
  Future<Map<String, dynamic>> getRow({
    required String databaseId,
    required String tableId,
    required String rowId,
  }) async {
    final res =
        await _dio.get('/v1/databases/$databaseId/tables/$tableId/rows/$rowId');
    return res.data;
  }

  /// Update a row.
  Future<Map<String, dynamic>> updateRow({
    required String databaseId,
    required String tableId,
    required String rowId,
    Map<String, dynamic>? data,
    List<String>? permissions,
  }) async {
    final res = await _dio.patch(
      '/v1/databases/$databaseId/tables/$tableId/rows/$rowId',
      data: {
        if (data != null) 'data': data,
        if (permissions != null) 'permissions': permissions,
      },
    );
    return res.data;
  }

  /// Delete a row.
  Future<void> deleteRow({
    required String databaseId,
    required String tableId,
    required String rowId,
  }) async {
    await _dio.delete('/v1/databases/$databaseId/tables/$tableId/rows/$rowId');
  }
}
