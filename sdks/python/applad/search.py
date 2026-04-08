"""Search service."""


class Search:
    def __init__(self, client):
        self.client = client

    def create_index(self, index_id: str, name: str = None, attributes: list = None):
        data = {"indexId": index_id}
        if name is not None:
            data["name"] = name
        if attributes is not None:
            data["attributes"] = attributes
        return self.client._call("POST", "/search/indexes", data)

    def list_indexes(self):
        return self.client._call("GET", "/search/indexes")

    def get_index(self, index_id: str):
        return self.client._call("GET", f"/search/indexes/{index_id}")

    def delete_index(self, index_id: str):
        return self.client._call("DELETE", f"/search/indexes/{index_id}")

    def index_document(self, index_id: str, document_id: str, data: dict):
        return self.client._call(
            "POST",
            f"/search/indexes/{index_id}/documents",
            {"documentId": document_id, "data": data},
        )

    def query(self, index_id: str, query: str, limit: int = None, offset: int = None, filters: dict = None):
        data = {"query": query}
        if limit is not None:
            data["limit"] = limit
        if offset is not None:
            data["offset"] = offset
        if filters is not None:
            data["filters"] = filters
        return self.client._call("POST", f"/search/indexes/{index_id}/search", data)

    def delete_document(self, index_id: str, document_id: str):
        return self.client._call("DELETE", f"/search/indexes/{index_id}/documents/{document_id}")