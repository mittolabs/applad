"""Vectors service."""


class Vectors:
    def __init__(self, client):
        self.client = client

    def create_index(self, index_id: str, dimensions: int, metric: str = None, name: str = None):
        data = {"indexId": index_id, "dimensions": dimensions}
        if metric is not None:
            data["metric"] = metric
        if name is not None:
            data["name"] = name
        return self.client._call("POST", "/vectors/indexes", data)

    def list_indexes(self):
        return self.client._call("GET", "/vectors/indexes")

    def get_index(self, index_id: str):
        return self.client._call("GET", f"/vectors/indexes/{index_id}")

    def delete_index(self, index_id: str):
        return self.client._call("DELETE", f"/vectors/indexes/{index_id}")

    def upsert(self, index_id: str, vectors: list):
        return self.client._call("POST", f"/vectors/indexes/{index_id}/vectors", {"vectors": vectors})

    def query(self, index_id: str, vector: list, top_k: int = None, filter: dict = None):
        data = {"vector": vector}
        if top_k is not None:
            data["topK"] = top_k
        if filter is not None:
            data["filter"] = filter
        return self.client._call("POST", f"/vectors/indexes/{index_id}/query", data)

    def delete_vectors(self, index_id: str, ids: list):
        return self.client._call("POST", f"/vectors/indexes/{index_id}/delete", {"ids": ids})