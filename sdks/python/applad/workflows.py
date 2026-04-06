"""Workflows service."""


class Workflows:
    def __init__(self, client):
        self.client = client

    def create(self, name: str):
        return self.client._call("POST", "/workflows", {
            "name": name,
            "description": "",
            "triggerType": "manual",
            "triggerConfig": {},
            "nodes": [],
            "edges": [],
        })

    def list(self):
        return self.client._call("GET", "/workflows")

    def get(self, workflow_id: str):
        return self.client._call("GET", f"/workflows/{workflow_id}")

    def delete(self, workflow_id: str):
        return self.client._call("DELETE", f"/workflows/{workflow_id}")

    def execute(self, workflow_id: str, trigger_data: dict = None):
        return self.client._call("POST", f"/workflows/{workflow_id}/execute", {
            "triggerData": trigger_data or {},
        })

    def list_executions(self, workflow_id: str):
        return self.client._call("GET", f"/workflows/{workflow_id}/executions")
