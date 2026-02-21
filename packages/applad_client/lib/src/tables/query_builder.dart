library;

import 'table_client.dart';

/// Fluent query builder for Applad table queries.
///
/// Example:
/// ```dart
/// final posts = await client
///   .from('posts')
///   .select('id, title, published_at')
///   .eq('published', true)
///   .order('published_at', ascending: false)
///   .limit(10)
///   .get();
/// ```
final class QueryBuilder {
  QueryBuilder._(this._tableClient, this._columns);

  // ignore: unused_field
  final TableClient _tableClient;
  // ignore: unused_field
  final String _columns;
  final Map<String, dynamic> _filters = {};
  // ignore: unused_field
  String? _orderBy;
  // ignore: unused_field
  bool _ascending = true;
  // ignore: unused_field
  int? _limit;
  // ignore: unused_field
  int? _offset;

  /// Filter rows where [column] equals [value].
  QueryBuilder eq(String column, dynamic value) {
    _filters['$column.eq'] = value;
    return this;
  }

  /// Filter rows where [column] does not equal [value].
  QueryBuilder neq(String column, dynamic value) {
    _filters['$column.neq'] = value;
    return this;
  }

  /// Filter rows where [column] is greater than [value].
  QueryBuilder gt(String column, dynamic value) {
    _filters['$column.gt'] = value;
    return this;
  }

  /// Filter rows where [column] is less than [value].
  QueryBuilder lt(String column, dynamic value) {
    _filters['$column.lt'] = value;
    return this;
  }

  /// Filter rows where [column] contains [value].
  QueryBuilder like(String column, String value) {
    _filters['$column.like'] = value;
    return this;
  }

  /// Filter rows where [column] is in [values].
  QueryBuilder inFilter(String column, List<dynamic> values) {
    _filters['$column.in'] = values;
    return this;
  }

  /// Order results by [column].
  QueryBuilder order(String column, {bool ascending = true}) {
    _orderBy = column;
    _ascending = ascending;
    return this;
  }

  /// Limit results to [count] rows.
  QueryBuilder limit(int count) {
    _limit = count;
    return this;
  }

  /// Skip the first [count] rows.
  QueryBuilder offset(int count) {
    _offset = count;
    return this;
  }

  /// Execute the query and return results.
  Future<List<Map<String, dynamic>>> get() async {
    // Phase 2 implementation
    throw UnimplementedError('Table queries — available in Phase 2');
  }

  /// Execute the query and return a single row.
  Future<Map<String, dynamic>?> getSingle() async {
    throw UnimplementedError('getSingle — available in Phase 2');
  }
}

extension QueryBuilderFactory on TableClient {
  QueryBuilder select([String columns = '*']) => QueryBuilder._(this, columns);
}
