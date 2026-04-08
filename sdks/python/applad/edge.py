"""Edge service."""


class Edge:
    def __init__(self, client):
        self.client = client

    def create(self, name: str, code: str, route: str = None, timeout: int = None):
        data = {"name": name, "code": code}
        if route is not None:
            data["route"] = route
        if timeout is not None:
            data["timeout"] = timeout
        return self.client._call("POST", "/edge/functions", data)

    def list(self):
        return self.client._call("GET", "/edge/functions")

    def get(self, function_id: str):
        return self.client._call("GET", f"/edge/functions/{function_id}")

    def update(self, function_id: str, **opts):
        return self.client._call("PUT", f"/edge/functions/{function_id}", opts)

    def delete(self, function_id: str):
        return self.client._call("DELETE", f"/edge/functions/{function_id}")

    def invoke(self, function_id: str, data: dict = None):
        return self.client._call("POST", f"/edge/functions/{function_id}/invoke", data or {})

    def list_executions(self, function_id: str):
        return self.client._call("GET", f"/edge/functions/{function_id}/executions")