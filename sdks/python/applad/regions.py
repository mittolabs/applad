"""Regions service."""


class Regions:
    def __init__(self, client):
        self.client = client

    def list(self):
        return self.client._call("GET", "/regions")

    def get(self, region_id: str):
        return self.client._call("GET", f"/regions/{region_id}")

    def get_active(self):
        return self.client._call("GET", "/regions/active")

    def set_active(self, region_id: str):
        return self.client._call("PUT", "/regions/active", {"regionId": region_id})

    def get_health(self, region_id: str):
        return self.client._call("GET", f"/regions/{region_id}/health")