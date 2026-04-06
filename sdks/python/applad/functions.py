"""Functions service."""


class Functions:
    def __init__(self, client):
        self.client = client

    def create(self, name: str, runtime: str):
        return self.client._call("POST", "/functions", {
            "name": name,
            "runtime": runtime,
            "entrypoint": "index.handler",
            "timeout": 15,
            "vars": {},
            "source": "",
        })

    def list(self):
        return self.client._call("GET", "/functions")

    def get(self, function_id: str):
        return self.client._call("GET", f"/functions/{function_id}")

    def delete(self, function_id: str):
        return self.client._call("DELETE", f"/functions/{function_id}")

    def execute(self, function_id: str, data: dict = None):
        return self.client._call("POST", f"/functions/{function_id}/executions", {
            "data": data or {},
        })

    def list_executions(self, function_id: str):
        return self.client._call("GET", f"/functions/{function_id}/executions")
