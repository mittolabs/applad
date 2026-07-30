"""Deploy service.

Deploy targets are the deployable unit. The API mounts them under
/deploy/targets (there is no flat /deploy resource); triggering a deploy runs
the target as an execution.
"""


class Deploy:
    def __init__(self, client):
        self.client = client

    def create(self, name: str, deploy_type: str, config: dict = None):
        body = {"name": name, "type": deploy_type}
        if config:
            body.update(config)
        return self.client._call("POST", "/deploy/targets", body)

    def list(self):
        return self.client._call("GET", "/deploy/targets")

    def get(self, target_id: str):
        return self.client._call("GET", f"/deploy/targets/{target_id}")

    def update(self, target_id: str, data: dict):
        return self.client._call("PUT", f"/deploy/targets/{target_id}", data)

    def delete(self, target_id: str):
        return self.client._call("DELETE", f"/deploy/targets/{target_id}")

    def deploy(self, target_id: str, options: dict = None):
        """Trigger a deploy of the target. Returns the created execution."""
        return self.client._call(
            "POST", f"/deploy/targets/{target_id}/executions", options or {}
        )
