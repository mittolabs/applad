"""Teams service."""


class Teams:
    def __init__(self, client):
        self.client = client

    def create(self, name: str, roles=None):
        body = {"name": name, "teamId": "unique()"}
        if roles:
            body["roles"] = roles
        return self.client._call("POST", "/teams", body)

    def list(self):
        return self.client._call("GET", "/teams")

    def get(self, team_id: str):
        return self.client._call("GET", f"/teams/{team_id}")

    def update(self, team_id: str, name: str):
        return self.client._call("PUT", f"/teams/{team_id}", {"name": name})

    def delete(self, team_id: str):
        return self.client._call("DELETE", f"/teams/{team_id}")

    # Memberships

    def create_membership(self, team_id: str, email: str, roles=None):
        body = {"email": email}
        if roles:
            body["roles"] = roles
        return self.client._call("POST", f"/teams/{team_id}/memberships", body)

    def list_memberships(self, team_id: str):
        return self.client._call("GET", f"/teams/{team_id}/memberships")

    def get_membership(self, team_id: str, membership_id: str):
        return self.client._call(
            "GET", f"/teams/{team_id}/memberships/{membership_id}"
        )

    def update_membership(self, team_id: str, membership_id: str, roles):
        return self.client._call(
            "PATCH",
            f"/teams/{team_id}/memberships/{membership_id}",
            {"roles": roles},
        )

    def delete_membership(self, team_id: str, membership_id: str):
        return self.client._call(
            "DELETE", f"/teams/{team_id}/memberships/{membership_id}"
        )

    def accept_membership(self, team_id: str, membership_id: str, secret: str):
        return self.client._call(
            "PATCH",
            f"/teams/{team_id}/memberships/{membership_id}/status",
            {"secret": secret},
        )
