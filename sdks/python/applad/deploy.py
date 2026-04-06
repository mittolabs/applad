"""Deploy service."""


class Deploy:
    def __init__(self, client):
        self.client = client

    def create(self, name: str, deploy_type: str, config: dict = None):
        return self.client._call("POST", "/deploy", {
            "name": name,
            "type": deploy_type,
            "config": config or {},
        })

    def list(self):
        return self.client._call("GET", "/deploy")

    def get(self, deployment_id: str):
        return self.client._call("GET", f"/deploy/{deployment_id}")

    def delete(self, deployment_id: str):
        return self.client._call("DELETE", f"/deploy/{deployment_id}")
