"""Users service."""


class Users:
    def __init__(self, client):
        self.client = client

    def create_user(self, email: str, password: str, name: str = ""):
        body = {"userId": "unique()", "email": email, "password": password}
        if name:
            body["name"] = name
        return self.client._call("POST", "/users", body)

    def get_user(self, user_id: str):
        return self.client._call("GET", f"/users/{user_id}")

    def list_users(self):
        return self.client._call("GET", "/users")

    def delete_user(self, user_id: str):
        return self.client._call("DELETE", f"/users/{user_id}")
