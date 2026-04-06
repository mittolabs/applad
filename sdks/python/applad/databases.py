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

    # --- Collections ---

    def create_collection(self, database_id: str, name: str):
        return self.client._call("POST", f"/databases/{database_id}/collections", {
            "name": name,
            "collectionId": "unique()",
            "permissions": [],
            "documentSecurity": False,
        })

    def list_collections(self, database_id: str):
        return self.client._call("GET", f"/databases/{database_id}/collections")

    # --- Documents ---

    def create_document(self, database_id: str, collection_id: str, data: dict):
        return self.client._call(
            "POST",
            f"/databases/{database_id}/collections/{collection_id}/documents",
            {"documentId": "unique()", "data": data, "permissions": []},
        )

    def list_documents(self, database_id: str, collection_id: str):
        return self.client._call(
            "GET",
            f"/databases/{database_id}/collections/{collection_id}/documents",
        )

    def get_document(self, database_id: str, collection_id: str, document_id: str):
        return self.client._call(
            "GET",
            f"/databases/{database_id}/collections/{collection_id}/documents/{document_id}",
        )

    def update_document(self, database_id: str, collection_id: str, document_id: str, data: dict):
        return self.client._call(
            "PATCH",
            f"/databases/{database_id}/collections/{collection_id}/documents/{document_id}",
            {"data": data},
        )

    def delete_document(self, database_id: str, collection_id: str, document_id: str):
        return self.client._call(
            "DELETE",
            f"/databases/{database_id}/collections/{collection_id}/documents/{document_id}",
        )
