"""Databases service."""


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

    def list_rows(self, database_id: str, table_id: str):
        return self.client._call(
            "GET",
            f"/databases/{database_id}/tables/{table_id}/rows",
        )

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
