"""Databases service."""

from urllib.parse import urlencode


class QueryResult:
    """The result of a :class:`QueryBuilder` ``get()`` call."""

    def __init__(self, total: int, rows: list):
        self.total = total
        self.rows = rows

    def __repr__(self):
        return f"QueryResult(total={self.total}, rows={len(self.rows)})"


class QueryBuilder:
    """Fluent query builder for listing rows in an Applad table.

    Created via :meth:`Databases.from_table` — do not instantiate directly.

    Example::

        result = client.databases \\
            .from_table('myDb', 'posts') \\
            .equal('published', True) \\
            .order_desc('created_at') \\
            .limit(25) \\
            .get()

        print(result.total)
        print(result.rows)
    """

    def __init__(self, client, database_id: str, table_id: str):
        self._client = client
        self._database_id = database_id
        self._table_id = table_id
        self._queries: list = []
        self._select: str | None = None
        self._order_attr: str | None = None
        self._order_type: str | None = None
        self._limit: int | None = None
        self._offset: int | None = None
        self._cursor: str | None = None

    # ── Column selection ──────────────────────────────────────────────────

    def select(self, columns: str) -> "QueryBuilder":
        """Specify which columns to return."""
        self._select = columns
        return self

    # ── Filters ───────────────────────────────────────────────────────────

    def _scalar(self, method: str, field: str, value) -> "QueryBuilder":
        self._queries.append(f'{method}("{field}","{value}")')
        return self

    def equal(self, field: str, value) -> "QueryBuilder":
        """Matches rows where ``field`` equals ``value``."""
        return self._scalar("equal", field, value)

    def not_equal(self, field: str, value) -> "QueryBuilder":
        """Matches rows where ``field`` does not equal ``value``."""
        return self._scalar("notEqual", field, value)

    def less_than(self, field: str, value) -> "QueryBuilder":
        """Matches rows where ``field`` is less than ``value``."""
        return self._scalar("lessThan", field, value)

    def less_than_or_equal(self, field: str, value) -> "QueryBuilder":
        """Matches rows where ``field`` is less than or equal to ``value``."""
        return self._scalar("lessThanEqual", field, value)

    def greater_than(self, field: str, value) -> "QueryBuilder":
        """Matches rows where ``field`` is greater than ``value``."""
        return self._scalar("greaterThan", field, value)

    def greater_than_or_equal(self, field: str, value) -> "QueryBuilder":
        """Matches rows where ``field`` is greater than or equal to ``value``."""
        return self._scalar("greaterThanEqual", field, value)

    def contains(self, field: str, value: str) -> "QueryBuilder":
        """Matches rows where ``field`` contains ``value`` (case-insensitive)."""
        return self._scalar("contains", field, value)

    def starts_with(self, field: str, value: str) -> "QueryBuilder":
        """Matches rows where ``field`` starts with ``value``."""
        return self._scalar("startsWith", field, value)

    def ends_with(self, field: str, value: str) -> "QueryBuilder":
        """Matches rows where ``field`` ends with ``value``."""
        return self._scalar("endsWith", field, value)

    def search(self, field: str, value: str) -> "QueryBuilder":
        """Full-text search on ``field`` for ``value``."""
        return self._scalar("search", field, value)

    def is_null(self, field: str) -> "QueryBuilder":
        """Matches rows where ``field`` is NULL."""
        self._queries.append(f'isNull("{field}")')
        return self

    def is_not_null(self, field: str) -> "QueryBuilder":
        """Matches rows where ``field`` is NOT NULL."""
        self._queries.append(f'isNotNull("{field}")')
        return self

    def between(self, field: str, min_val, max_val) -> "QueryBuilder":
        """Matches rows where ``field`` is between ``min_val`` and ``max_val``."""
        self._queries.append(f'between("{field}","{min_val}","{max_val}")')
        return self

    # ── Ordering ──────────────────────────────────────────────────────────

    def order_asc(self, field: str) -> "QueryBuilder":
        """Order results by ``field`` ascending."""
        self._order_attr = field
        self._order_type = "ASC"
        return self

    def order_desc(self, field: str) -> "QueryBuilder":
        """Order results by ``field`` descending."""
        self._order_attr = field
        self._order_type = "DESC"
        return self

    # ── Pagination ────────────────────────────────────────────────────────

    def limit(self, n: int) -> "QueryBuilder":
        """Maximum number of rows to return."""
        self._limit = n
        return self

    def offset(self, n: int) -> "QueryBuilder":
        """Number of rows to skip."""
        self._offset = n
        return self

    def cursor_after(self, row_id: str) -> "QueryBuilder":
        """Cursor-based pagination — pass the last seen row ID."""
        self._cursor = row_id
        return self

    # ── Execution ─────────────────────────────────────────────────────────

    def get(self) -> QueryResult:
        """Execute the query and return a :class:`QueryResult`."""
        params: list[tuple] = []
        if self._select:
            params.append(("select", self._select))
        if self._order_attr:
            params.append(("orderAttr", self._order_attr))
            params.append(("orderType", self._order_type or "ASC"))
        if self._limit is not None:
            params.append(("limit", str(self._limit)))
        if self._offset is not None:
            params.append(("offset", str(self._offset)))
        if self._cursor:
            params.append(("cursorAfter", self._cursor))
        for q in self._queries:
            params.append(("queries[]", q))

        path = f"/databases/{self._database_id}/tables/{self._table_id}/rows"
        if params:
            path += "?" + urlencode(params)

        data = self._client._call("GET", path)
        total = data.get("total", 0) if isinstance(data, dict) else 0
        rows = data.get("rows", []) if isinstance(data, dict) else []
        return QueryResult(total=int(total), rows=list(rows))


class Databases:
    def __init__(self, client):
        self.client = client

    # --- Databases ---

    def create_database(self, name: str):
        return self.client._call("POST", "/databases", {
            "name": name,
            "databaseId": "unique()",
        })

    def list_databases(self):
        return self.client._call("GET", "/databases")

    def get_database(self, database_id: str):
        return self.client._call("GET", f"/databases/{database_id}")

    def delete_database(self, database_id: str):
        return self.client._call("DELETE", f"/databases/{database_id}")

    # --- Tables ---

    def create_table(self, database_id: str, name: str):
        return self.client._call("POST", f"/databases/{database_id}/tables", {
            "name": name,
            "tableId": "unique()",
            "permissions": [],
            "documentSecurity": False,
        })

    def list_tables(self, database_id: str):
        return self.client._call("GET", f"/databases/{database_id}/tables")

    # --- Rows ---

    def create_row(self, database_id: str, table_id: str, data: dict):
        return self.client._call(
            "POST",
            f"/databases/{database_id}/tables/{table_id}/rows",
            {"rowId": "unique()", "data": data, "permissions": []},
        )

    def list_rows(self, database_id: str, table_id: str, status: str = "", locale: str = ""):
        # Content mode: narrow to a publish state and/or a locale.
        query = "&".join(
            f"{k}={v}" for k, v in (("status", status), ("locale", locale)) if v
        )
        path = f"/databases/{database_id}/tables/{table_id}/rows"
        return self.client._call("GET", f"{path}?{query}" if query else path)

    def get_row(self, database_id: str, table_id: str, row_id: str):
        return self.client._call(
            "GET",
            f"/databases/{database_id}/tables/{table_id}/rows/{row_id}",
        )

    def update_row(self, database_id: str, table_id: str, row_id: str, data: dict):
        return self.client._call(
            "PATCH",
            f"/databases/{database_id}/tables/{table_id}/rows/{row_id}",
            {"data": data},
        )

    def delete_row(self, database_id: str, table_id: str, row_id: str):
        return self.client._call(
            "DELETE",
            f"/databases/{database_id}/tables/{table_id}/rows/{row_id}",
        )

    # --- Content mode ---
    # A table can act as an editorial collection: rows gain a draft/published
    # workflow, a slug, a locale and version history. Same table, same rows API.

    def enable_content_mode(self, database_id: str, table_id: str):
        """Turn a table into an editorial collection."""
        return self.client._call(
            "POST", f"/databases/{database_id}/tables/{table_id}/content"
        )

    def disable_content_mode(self, database_id: str, table_id: str):
        """Hide the editorial tools again. Nothing is deleted."""
        return self.client._call(
            "DELETE", f"/databases/{database_id}/tables/{table_id}/content"
        )

    def publish_row(self, database_id: str, table_id: str, row_id: str):
        """Publish an entry."""
        return self.client._call(
            "POST", f"/databases/{database_id}/tables/{table_id}/rows/{row_id}/publish"
        )

    def unpublish_row(self, database_id: str, table_id: str, row_id: str):
        """Return an entry to draft."""
        return self.client._call(
            "POST", f"/databases/{database_id}/tables/{table_id}/rows/{row_id}/unpublish"
        )

    def list_row_versions(self, database_id: str, table_id: str, row_id: str):
        """Version snapshots for an entry, newest first."""
        return self.client._call(
            "GET", f"/databases/{database_id}/tables/{table_id}/rows/{row_id}/versions"
        )

    # --- Columns ---

    def create_column(
        self,
        database_id: str,
        table_id: str,
        column_type: str,
        key: str,
        *,
        required: bool = False,
        array: bool = False,
        encrypted: bool = False,
        default=None,
        size: int | None = None,
        min_value=None,
        max_value=None,
        elements: list | None = None,
        validation: dict | None = None,
    ):
        """Create a column of the given type ("string", "integer", "boolean",
        "double", "datetime", "email", "url", "enum", ...) on a table.

        Set ``encrypted=True`` to store this column's values as opaque
        ciphertext at rest (see the field-level encryption docs). Cannot be
        combined with ``array=True``, and requires the instance to have
        MASTER_ENCRYPTION_KEY configured.
        """
        body = {
            "key": key,
            "required": required,
            "array": array,
            "encrypted": encrypted,
        }
        if default is not None:
            body["default"] = default
        if size is not None:
            body["size"] = size
        if min_value is not None:
            body["min"] = min_value
        if max_value is not None:
            body["max"] = max_value
        if elements is not None:
            body["elements"] = elements
        if validation is not None:
            body["validation"] = validation
        return self.client._call(
            "POST",
            f"/databases/{database_id}/tables/{table_id}/columns/{column_type}",
            body,
        )

    # --- Column permissions ---

    def get_column_permissions(self, database_id: str, table_id: str, key: str):
        return self.client._call(
            "GET",
            f"/databases/{database_id}/tables/{table_id}/columns/{key}/permissions",
        )

    def set_column_permissions(self, database_id: str, table_id: str, key: str, permissions: list):
        return self.client._call(
            "POST",
            f"/databases/{database_id}/tables/{table_id}/columns/{key}/permissions",
            {"permissions": permissions},
        )

    # --- Query builder ---

    def from_table(self, database_id: str, table_id: str) -> QueryBuilder:
        """Return a fluent :class:`QueryBuilder` for the given table.

        Example::

            result = client.databases \\
                .from_table('myDb', 'orders') \\
                .equal('status', 'pending') \\
                .order_asc('created_at') \\
                .limit(50) \\
                .get()
        """
        return QueryBuilder(self.client, database_id, table_id)
