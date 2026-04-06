import 'client.dart';

/// Databases service — manage databases, tables, columns, indexes, and rows.
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

  // --- Tables ---

  /// Create a table.
  Future<Map<String, dynamic>> createTable({
    required String databaseId,
    required String name,
    String? tableId,
    List<String>? permissions,
    bool documentSecurity = false,
  }) async {
    return _client.post('/v1/databases/$databaseId/tables', data: {
      'name': name,
      'tableId': tableId ?? 'unique()',
      'permissions': permissions ?? [],
      'documentSecurity': documentSecurity,
    });
  }

  /// List tables in a database.
  Future<Map<String, dynamic>> listTables(String databaseId) async {
    return _client.get('/v1/databases/$databaseId/tables');
  }

  /// Get a table.
  Future<Map<String, dynamic>> getTable(
      String databaseId, String tableId) async {
    return _client.get('/v1/databases/$databaseId/tables/$tableId');
  }

  /// Update a table.
  Future<Map<String, dynamic>> updateTable({
    required String databaseId,
    required String tableId,
    required String name,
    List<String>? permissions,
    bool? enabled,
  }) async {
    return _client.put(
      '/v1/databases/$databaseId/tables/$tableId',
      data: {
        'name': name,
        if (permissions != null) 'permissions': permissions,
        if (enabled != null) 'enabled': enabled,
      },
    );
  }

  /// Delete a table.
  Future<Map<String, dynamic>> deleteTable(
      String databaseId, String tableId) async {
    return _client
        .delete('/v1/databases/$databaseId/tables/$tableId');
  }

  // --- Columns ---

  /// Create a string column.
  Future<Map<String, dynamic>> createStringColumn({
    required String databaseId,
    required String tableId,
    required String key,
    bool required_ = false,
    int? size,
    String? defaultValue,
    bool array = false,
  }) async {
    return _client.post(
      '/v1/databases/$databaseId/tables/$tableId/columns/string',
      data: {
        'key': key,
        'required': required_,
        'array': array,
        if (size != null) 'size': size,
        if (defaultValue != null) 'default': defaultValue,
      },
    );
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
  }) async {
    return _client.post(
      '/v1/databases/$databaseId/tables/$tableId/columns/integer',
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

  /// Create a boolean column.
  Future<Map<String, dynamic>> createBooleanColumn({
    required String databaseId,
    required String tableId,
    required String key,
    bool required_ = false,
    bool? defaultValue,
    bool array = false,
  }) async {
    return _client.post(
      '/v1/databases/$databaseId/tables/$tableId/columns/boolean',
      data: {
        'key': key,
        'required': required_,
        'array': array,
        if (defaultValue != null) 'default': defaultValue,
      },
    );
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
  }) async {
    return _client.post(
      '/v1/databases/$databaseId/tables/$tableId/columns/enum',
      data: {
        'key': key,
        'required': required_,
        'array': array,
        'elements': elements,
        if (defaultValue != null) 'default': defaultValue,
      },
    );
  }

  /// List columns of a table.
  Future<Map<String, dynamic>> listColumns(
      String databaseId, String tableId) async {
    return _client.get(
        '/v1/databases/$databaseId/tables/$tableId/columns');
  }

  /// Delete a column.
  Future<Map<String, dynamic>> deleteColumn(
      String databaseId, String tableId, String key) async {
    return _client.delete(
        '/v1/databases/$databaseId/tables/$tableId/columns/$key');
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
    return _client.post(
      '/v1/databases/$databaseId/tables/$tableId/indexes',
      data: {
        'key': key,
        'type': type,
        'columns': columns,
        if (orders != null) 'orders': orders,
      },
    );
  }

  /// List indexes of a table.
  Future<Map<String, dynamic>> listIndexes(
      String databaseId, String tableId) async {
    return _client.get(
        '/v1/databases/$databaseId/tables/$tableId/indexes');
  }

  /// Delete an index.
  Future<Map<String, dynamic>> deleteIndex(
      String databaseId, String tableId, String key) async {
    return _client.delete(
        '/v1/databases/$databaseId/tables/$tableId/indexes/$key');
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
    return _client.post(
      '/v1/databases/$databaseId/tables/$tableId/rows',
      data: {
        'rowId': rowId ?? 'unique()',
        'data': data,
        'permissions': permissions ?? [],
      },
    );
  }

  /// List rows in a table.
  Future<Map<String, dynamic>> listRows({
    required String databaseId,
    required String tableId,
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
        '/v1/databases/$databaseId/tables/$tableId/rows$qs');
  }

  /// Get a row by ID.
  Future<Map<String, dynamic>> getRow({
    required String databaseId,
    required String tableId,
    required String rowId,
  }) async {
    return _client.get(
        '/v1/databases/$databaseId/tables/$tableId/rows/$rowId');
  }

  /// Update a row.
  Future<Map<String, dynamic>> updateRow({
    required String databaseId,
    required String tableId,
    required String rowId,
    Map<String, dynamic>? data,
    List<String>? permissions,
  }) async {
    return _client.patch(
      '/v1/databases/$databaseId/tables/$tableId/rows/$rowId',
      data: {
        if (data != null) 'data': data,
        if (permissions != null) 'permissions': permissions,
      },
    );
  }

  /// Delete a row.
  Future<Map<String, dynamic>> deleteRow({
    required String databaseId,
    required String tableId,
    required String rowId,
  }) async {
    return _client.delete(
        '/v1/databases/$databaseId/tables/$tableId/rows/$rowId');
  }
}
