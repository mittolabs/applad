/// A fluent query builder for listing rows in an Applad table.
///
/// Created via [Databases.from] — do not instantiate directly.
///
/// ```dart
/// final result = await client.databases
///     .from('myDb', 'posts')
///     .select('id, title, author(name)')
///     .equal('published', true)
///     .orderDesc('created_at')
///     .limit(25)
///     .get();
///
/// print(result.total);   // total matching rows
/// print(result.rows);    // List<Map<String, dynamic>>
/// ```
library;

/// The fetch function type used internally by [QueryBuilder].
typedef RowFetcher = Future<Map<String, dynamic>> Function(
    Map<String, dynamic> params);

class QueryBuilder {
  final RowFetcher _fetch;
  final List<String> _queries = [];

  String? _select;
  String? _orderAttr;
  String? _orderType;
  int? _limit;
  int? _offset;
  String? _cursorAfter;

  /// Creates a [QueryBuilder]. Prefer [Databases.from] over calling this directly.
  QueryBuilder(RowFetcher fetch) : _fetch = fetch;

  // ── Column selection ────────────────────────────────────────────────────────

  /// Specify which columns to return, e.g. `'id, title, author(name)'`.
  /// Use `author(name)` syntax to select a related column via a relationship.
  QueryBuilder select(String columns) {
    _select = columns;
    return this;
  }

  // ── Filters ─────────────────────────────────────────────────────────────────

  /// Matches rows where [field] equals [value].
  QueryBuilder equal(String field, dynamic value) =>
      _scalar('equal', field, value);

  /// Matches rows where [field] does not equal [value].
  QueryBuilder notEqual(String field, dynamic value) =>
      _scalar('notEqual', field, value);

  /// Matches rows where [field] is less than [value].
  QueryBuilder lessThan(String field, dynamic value) =>
      _scalar('lessThan', field, value);

  /// Matches rows where [field] is less than or equal to [value].
  QueryBuilder lessThanOrEqual(String field, dynamic value) =>
      _scalar('lessThanEqual', field, value);

  /// Matches rows where [field] is greater than [value].
  QueryBuilder greaterThan(String field, dynamic value) =>
      _scalar('greaterThan', field, value);

  /// Matches rows where [field] is greater than or equal to [value].
  QueryBuilder greaterThanOrEqual(String field, dynamic value) =>
      _scalar('greaterThanEqual', field, value);

  /// Matches rows where [field] contains [value] (case-insensitive LIKE).
  QueryBuilder contains(String field, String value) =>
      _scalar('contains', field, value);

  /// Matches rows where [field] starts with [value].
  QueryBuilder startsWith(String field, String value) =>
      _scalar('startsWith', field, value);

  /// Matches rows where [field] ends with [value].
  QueryBuilder endsWith(String field, String value) =>
      _scalar('endsWith', field, value);

  /// Full-text search on [field] for [value].
  QueryBuilder search(String field, String value) =>
      _scalar('search', field, value);

  /// Matches rows where [field] is NULL.
  QueryBuilder isNull(String field) {
    _queries.add('isNull("$field")');
    return this;
  }

  /// Matches rows where [field] is NOT NULL.
  QueryBuilder isNotNull(String field) {
    _queries.add('isNotNull("$field")');
    return this;
  }

  /// Matches rows where [field] is between [min] and [max] (inclusive).
  QueryBuilder between(String field, dynamic min, dynamic max) {
    _queries.add('between("$field","$min","$max")');
    return this;
  }

  // ── Ordering ─────────────────────────────────────────────────────────────────

  /// Order results by [field] ascending.
  QueryBuilder orderAsc(String field) {
    _orderAttr = field;
    _orderType = 'ASC';
    return this;
  }

  /// Order results by [field] descending.
  QueryBuilder orderDesc(String field) {
    _orderAttr = field;
    _orderType = 'DESC';
    return this;
  }

  // ── Pagination ───────────────────────────────────────────────────────────────

  /// Maximum number of rows to return. Defaults to the server default (25).
  QueryBuilder limit(int n) {
    _limit = n;
    return this;
  }

  /// Number of rows to skip (offset-based pagination).
  /// Prefer [cursorAfter] for large datasets.
  QueryBuilder offset(int n) {
    _offset = n;
    return this;
  }

  /// Cursor-based pagination — pass the last seen row ID to fetch the next page.
  ///
  /// ```dart
  /// // First page
  /// final page1 = await builder.limit(25).get();
  ///
  /// // Next page
  /// final page2 = await builder
  ///     .limit(25)
  ///     .cursorAfter(page1.rows.last['$id'])
  ///     .get();
  /// ```
  QueryBuilder cursorAfter(String rowId) {
    _cursorAfter = rowId;
    return this;
  }

  // ── Execution ────────────────────────────────────────────────────────────────

  /// Execute the query and return a [QueryResult].
  Future<QueryResult> get() async {
    final data = await _fetch(_buildParams());
    return QueryResult._fromJson(data);
  }

  // ── Internal ─────────────────────────────────────────────────────────────────

  QueryBuilder _scalar(String method, String field, dynamic value) {
    _queries.add('$method("$field","$value")');
    return this;
  }

  Map<String, dynamic> _buildParams() {
    final params = <String, dynamic>{};
    if (_select != null) params['select'] = _select;
    if (_orderAttr != null) params['orderAttr'] = _orderAttr;
    if (_orderType != null) params['orderType'] = _orderType;
    if (_limit != null) params['limit'] = _limit;
    if (_offset != null) params['offset'] = _offset;
    if (_cursorAfter != null) params['cursorAfter'] = _cursorAfter;
    if (_queries.isNotEmpty) params['queries[]'] = _queries;
    return params;
  }
}

/// The result of a [QueryBuilder.get] call.
class QueryResult {
  /// Total number of rows matching the query (before limit/offset).
  final int total;

  /// The rows returned for this page.
  final List<Map<String, dynamic>> rows;

  const QueryResult({required this.total, required this.rows});

  factory QueryResult._fromJson(Map<String, dynamic> json) {
    final rawRows = json['rows'];
    final rows = rawRows is List
        ? rawRows
            .map((r) => Map<String, dynamic>.from(r as Map))
            .toList()
        : <Map<String, dynamic>>[];
    return QueryResult(
      total: (json['total'] as num?)?.toInt() ?? rows.length,
      rows: rows,
    );
  }

  @override
  String toString() => 'QueryResult(total: $total, rows: ${rows.length})';
}
