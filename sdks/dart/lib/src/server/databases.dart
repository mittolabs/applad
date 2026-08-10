import '../query_builder.dart';
import '../server_client.dart';

export '../query_builder.dart' show QueryBuilder, QueryResult;

/// Databases service — manage databases, tables, columns, indexes, and rows.
class Databases {
  final ApplAdServer _client;

  Databases(this._client);

  // --- Query builder ---

  /// Returns a [QueryBuilder] for fluent row queries on [tableId] inside [databaseId].
  ///
  /// ```dart
  /// final result = await server.databases
  ///     .from('myDb', 'posts')
  ///     .equal('status', 'active')
  ///     .orderDesc('created_at')
  ///     .limit(100)
  ///     .get();
  /// ```
  QueryBuilder from(String databaseId, String tableId) {
    return QueryBuilder((params) async {
      final qs = _buildQueryString(params);
      return _client.get(
        '/v1/databases/$databaseId/tables/$tableId/rows$qs',
      );
    });
  }

  static String _buildQueryString(Map<String, dynamic> params) {
    if (params.isEmpty) return '';
    final parts = <String>[];
    params.forEach((key, value) {
      if (value is List) {
        for (final v in value) {
          parts.add('${Uri.encodeQueryComponent(key)}=${Uri.encodeQueryComponent(v.toString())}');
        }
      } else {
        parts.add('${Uri.encodeQueryComponent(key)}=${Uri.encodeQueryComponent(value.toString())}');
      }
    });
    return '?${parts.join('&')}';
  }

  // --- Databases ---

  Future<Map<String, dynamic>> createDatabase({
    required String name,
    String? databaseId,
  }) async {
    return _client.post('/v1/databases', data: {
      'name': name,
      'databaseId': databaseId ?? 'unique()',
    });
  }

  Future<Map<String, dynamic>> listDatabases() async {
    return _client.get('/v1/databases');
  }

  Future<Map<String, dynamic>> getDatabase(String databaseId) async {
    return _client.get('/v1/databases/$databaseId');
  }

  Future<Map<String, dynamic>> updateDatabase(
      String databaseId, String name) async {
    return _client.put('/v1/databases/$databaseId', data: {'name': name});
  }

  Future<Map<String, dynamic>> deleteDatabase(String databaseId) async {
    return _client.delete('/v1/databases/$databaseId');
  }

  // --- Tables ---

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

  Future<Map<String, dynamic>> listTables(String databaseId) async {
    return _client.get('/v1/databases/$databaseId/tables');
  }

  Future<Map<String, dynamic>> getTable(
      String databaseId, String tableId) async {
    return _client.get('/v1/databases/$databaseId/tables/$tableId');
  }

  Future<Map<String, dynamic>> updateTable({
    required String databaseId,
    required String tableId,
    required String name,
    List<String>? permissions,
    bool? enabled,
  }) async {
    return _client.put('/v1/databases/$databaseId/tables/$tableId', data: {
      'name': name,
      if (permissions != null) 'permissions': permissions,
      if (enabled != null) 'enabled': enabled,
    });
  }

  Future<Map<String, dynamic>> deleteTable(
      String databaseId, String tableId) async {
    return _client.delete('/v1/databases/$databaseId/tables/$tableId');
  }

  // --- Columns ---

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
    return _client.post(
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
  }

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
    return _client.post(
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
  }

  Future<Map<String, dynamic>> createBooleanColumn({
    required String databaseId,
    required String tableId,
    required String key,
    bool required_ = false,
    bool? defaultValue,
    bool array = false,
    bool encrypted = false,
  }) async {
    return _client.post(
      '/v1/databases/$databaseId/tables/$tableId/columns/boolean',
      data: {
        'key': key,
        'required': required_,
        'array': array,
        'encrypted': encrypted,
        if (defaultValue != null) 'default': defaultValue,
      },
    );
  }

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
    return _client.post(
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
  }

  Future<Map<String, dynamic>> listColumns(
      String databaseId, String tableId) async {
    return _client.get('/v1/databases/$databaseId/tables/$tableId/columns');
  }

  Future<Map<String, dynamic>> deleteColumn(
      String databaseId, String tableId, String key) async {
    return _client
        .delete('/v1/databases/$databaseId/tables/$tableId/columns/$key');
  }

  Future<Map<String, dynamic>> getColumnPermissions(
      String databaseId, String tableId, String key) async {
    return _client.get(
        '/v1/databases/$databaseId/tables/$tableId/columns/$key/permissions');
  }

  Future<Map<String, dynamic>> setColumnPermissions(
      String databaseId, String tableId, String key, List<String> permissions) async {
    return _client.post(
      '/v1/databases/$databaseId/tables/$tableId/columns/$key/permissions',
      data: {'permissions': permissions},
    );
  }

  // --- Indexes ---

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

  Future<Map<String, dynamic>> listIndexes(
      String databaseId, String tableId) async {
    return _client.get('/v1/databases/$databaseId/tables/$tableId/indexes');
  }

  Future<Map<String, dynamic>> deleteIndex(
      String databaseId, String tableId, String key) async {
    return _client
        .delete('/v1/databases/$databaseId/tables/$tableId/indexes/$key');
  }

  // --- Rows ---

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

  Future<Map<String, dynamic>> listRows({
    required String databaseId,
    required String tableId,
    int? limit,
    int? offset,
    String? status,
    String? locale,
  }) async {
    final query = <String, String>{};
    if (limit != null) query['limit'] = limit.toString();
    if (offset != null) query['offset'] = offset.toString();
    // Content mode: narrow to a publish state and/or a locale.
    if (status != null) query['status'] = status;
    if (locale != null) query['locale'] = locale;
    final qs = query.isNotEmpty
        ? '?${query.entries.map((e) => '${e.key}=${Uri.encodeComponent(e.value)}').join('&')}'
        : '';
    return _client.get('/v1/databases/$databaseId/tables/$tableId/rows$qs');
  }

  Future<Map<String, dynamic>> getRow({
    required String databaseId,
    required String tableId,
    required String rowId,
  }) async {
    return _client.get('/v1/databases/$databaseId/tables/$tableId/rows/$rowId');
  }

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

  Future<Map<String, dynamic>> deleteRow({
    required String databaseId,
    required String tableId,
    required String rowId,
  }) async {
    return _client
        .delete('/v1/databases/$databaseId/tables/$tableId/rows/$rowId');
  }

  // --- Content mode ---
  // A table can act as an editorial collection: rows gain a draft/published
  // workflow, a slug, a locale and version history. Same table, same rows API.

  /// Turns a table into an editorial collection.
  Future<Map<String, dynamic>> enableContentMode({
    required String databaseId,
    required String tableId,
  }) async {
    return _client.post('/v1/databases/$databaseId/tables/$tableId/content');
  }

  /// Hides the editorial tools again. Nothing is deleted.
  Future<Map<String, dynamic>> disableContentMode({
    required String databaseId,
    required String tableId,
  }) async {
    return _client.delete('/v1/databases/$databaseId/tables/$tableId/content');
  }

  /// Publishes an entry.
  Future<Map<String, dynamic>> publishRow({
    required String databaseId,
    required String tableId,
    required String rowId,
  }) async {
    return _client
        .post('/v1/databases/$databaseId/tables/$tableId/rows/$rowId/publish');
  }

  /// Returns an entry to draft.
  Future<Map<String, dynamic>> unpublishRow({
    required String databaseId,
    required String tableId,
    required String rowId,
  }) async {
    return _client
        .post('/v1/databases/$databaseId/tables/$tableId/rows/$rowId/unpublish');
  }

  /// Version snapshots for an entry, newest first.
  Future<Map<String, dynamic>> listRowVersions({
    required String databaseId,
    required String tableId,
    required String rowId,
  }) async {
    return _client
        .get('/v1/databases/$databaseId/tables/$tableId/rows/$rowId/versions');
  }
}
