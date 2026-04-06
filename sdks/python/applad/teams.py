"""Teams service."""


class Teams:
    def __init__(self, client):
        self.client = client

    def create(self, name: str):
        return self.client._call("POST", "/teams", {
            "name": name,
            "teamId": "unique()",
        })

    def list(self):
        return self.client._call("GET", "/teams")

    def get(self, team_id: str):
        return self.client._call("GET", f"/teams/{team_id}")

    def delete(self, team_id: str):
        return self.client._call("DELETE", f"/teams/{team_id}")
