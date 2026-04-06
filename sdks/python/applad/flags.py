"""Flags service — manage and evaluate feature flags."""


class Flags:
    def __init__(self, client):
        self.client = client

    # --- CRUD ---

    def list(self):
        """List all flags."""
        return self.client._call("GET", "/flags")

    def create(self, key: str, name: str, *, description: str = None,
               enabled: bool = None, variants: dict = None, rules: list = None):
        """Create a new flag."""
        data = {"key": key, "name": name}
        if description is not None:
            data["description"] = description
        if enabled is not None:
            data["enabled"] = enabled
        if variants is not None:
            data["variants"] = variants
        if rules is not None:
            data["rules"] = rules
        return self.client._call("POST", "/flags", data)

    def get(self, key: str):
        """Get a flag by key."""
        return self.client._call("GET", f"/flags/{key}")

    def update(self, key: str, *, name: str = None, description: str = None,
               variants: dict = None, rules: list = None):
        """Update a flag."""
        data = {}
        if name is not None:
            data["name"] = name
        if description is not None:
            data["description"] = description
        if variants is not None:
            data["variants"] = variants
        if rules is not None:
            data["rules"] = rules
        return self.client._call("PUT", f"/flags/{key}", data)

    def delete(self, key: str):
        """Delete a flag."""
        return self.client._call("DELETE", f"/flags/{key}")

    def toggle(self, key: str, enabled: bool):
        """Toggle a flag on or off."""
        return self.client._call("PATCH", f"/flags/{key}/toggle", {"enabled": enabled})

    # --- Evaluation ---

    def get_flag(self, key: str):
        """Get a single flag evaluation by key."""
        return self.client._call("GET", f"/flags/evaluate/{key}")

    def get_all_flags(self, context: dict = None):
        """Get all flag evaluations with optional context."""
        data = {}
        if context is not None:
            data["context"] = context
        return self.client._call("POST", "/flags/evaluate/all", data)

    def evaluate_flag(self, key: str, context: dict = None):
        """Evaluate a flag with key and context."""
        data = {"key": key}
        if context is not None:
            data["context"] = context
        return self.client._call("POST", "/flags/evaluate", data)
