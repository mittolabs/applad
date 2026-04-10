/**
 * Fluent query builder for listing rows in an Applad table.
 *
 * Created via {@link Databases.from} — do not instantiate directly.
 *
 * @example
 * ```ts
 * const result = await client.databases
 *   .from('myDb', 'posts')
 *   .select('id, title, author(name)')
 *   .equal('published', true)
 *   .orderDesc('created_at')
 *   .limit(25)
 *   .get();
 *
 * console.log(result.total);  // total matching rows
 * console.log(result.rows);   // the page of rows
 * ```
 */

export interface QueryResult {
  total: number;
  rows: Record<string, unknown>[];
}

type RowFetcher = (params: Record<string, unknown>) => Promise<QueryResult>;

export class QueryBuilder {
  private readonly _fetch: RowFetcher;
  private _queries: string[] = [];
  private _select?: string;
  private _orderAttr?: string;
  private _orderType?: string;
  private _limit?: number;
  private _offset?: number;
  private _cursorAfter?: string;

  /** @internal */
  constructor(fetch: RowFetcher) {
    this._fetch = fetch;
  }

  // ── Column selection ──────────────────────────────────────────────────────

  /** Specify which columns to return, e.g. `'id, title, author(name)'`. */
  select(columns: string): this {
    this._select = columns;
    return this;
  }

  // ── Filters ───────────────────────────────────────────────────────────────

  /** Matches rows where `field` equals `value`. */
  equal(field: string, value: unknown): this {
    return this._scalar('equal', field, value);
  }

  /** Matches rows where `field` does not equal `value`. */
  notEqual(field: string, value: unknown): this {
    return this._scalar('notEqual', field, value);
  }

  /** Matches rows where `field` is less than `value`. */
  lessThan(field: string, value: unknown): this {
    return this._scalar('lessThan', field, value);
  }

  /** Matches rows where `field` is less than or equal to `value`. */
  lessThanOrEqual(field: string, value: unknown): this {
    return this._scalar('lessThanEqual', field, value);
  }

  /** Matches rows where `field` is greater than `value`. */
  greaterThan(field: string, value: unknown): this {
    return this._scalar('greaterThan', field, value);
  }

  /** Matches rows where `field` is greater than or equal to `value`. */
  greaterThanOrEqual(field: string, value: unknown): this {
    return this._scalar('greaterThanEqual', field, value);
  }

  /** Matches rows where `field` contains `value` (case-insensitive). */
  contains(field: string, value: string): this {
    return this._scalar('contains', field, value);
  }

  /** Matches rows where `field` starts with `value`. */
  startsWith(field: string, value: string): this {
    return this._scalar('startsWith', field, value);
  }

  /** Matches rows where `field` ends with `value`. */
  endsWith(field: string, value: string): this {
    return this._scalar('endsWith', field, value);
  }

  /** Full-text search on `field` for `value`. */
  search(field: string, value: string): this {
    return this._scalar('search', field, value);
  }

  /** Matches rows where `field` is NULL. */
  isNull(field: string): this {
    this._queries.push(`isNull("${field}")`);
    return this;
  }

  /** Matches rows where `field` is NOT NULL. */
  isNotNull(field: string): this {
    this._queries.push(`isNotNull("${field}")`);
    return this;
  }

  /** Matches rows where `field` is between `min` and `max` (inclusive). */
  between(field: string, min: unknown, max: unknown): this {
    this._queries.push(`between("${field}","${min}","${max}")`);
    return this;
  }

  // ── Ordering ──────────────────────────────────────────────────────────────

  /** Order results by `field` ascending. */
  orderAsc(field: string): this {
    this._orderAttr = field;
    this._orderType = 'ASC';
    return this;
  }

  /** Order results by `field` descending. */
  orderDesc(field: string): this {
    this._orderAttr = field;
    this._orderType = 'DESC';
    return this;
  }

  // ── Pagination ────────────────────────────────────────────────────────────

  /** Maximum number of rows to return. */
  limit(n: number): this {
    this._limit = n;
    return this;
  }

  /** Number of rows to skip (offset-based pagination). */
  offset(n: number): this {
    this._offset = n;
    return this;
  }

  /**
   * Cursor-based pagination — pass the last seen row ID to fetch the next page.
   *
   * @example
   * ```ts
   * const page1 = await builder.limit(25).get();
   * const page2 = await builder.limit(25).cursorAfter(page1.rows.at(-1)!['$id'] as string).get();
   * ```
   */
  cursorAfter(rowId: string): this {
    this._cursorAfter = rowId;
    return this;
  }

  // ── Execution ─────────────────────────────────────────────────────────────

  /** Execute the query and return a {@link QueryResult}. */
  async get(): Promise<QueryResult> {
    return this._fetch(this._buildParams());
  }

  // ── Internal ──────────────────────────────────────────────────────────────

  private _scalar(method: string, field: string, value: unknown): this {
    this._queries.push(`${method}("${field}","${value}")`);
    return this;
  }

  private _buildParams(): Record<string, unknown> {
    const params: Record<string, unknown> = {};
    if (this._select !== undefined) params['select'] = this._select;
    if (this._orderAttr !== undefined) params['orderAttr'] = this._orderAttr;
    if (this._orderType !== undefined) params['orderType'] = this._orderType;
    if (this._limit !== undefined) params['limit'] = this._limit;
    if (this._offset !== undefined) params['offset'] = this._offset;
    if (this._cursorAfter !== undefined) params['cursorAfter'] = this._cursorAfter;
    if (this._queries.length > 0) params['queries[]'] = this._queries;
    return params;
  }
}
