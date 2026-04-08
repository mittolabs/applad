"""Analytics service."""


class Analytics:
    def __init__(self, client):
        self.client = client

    def track_event(self, event: str, properties: dict = None):
        data = {"event": event}
        if properties is not None:
            data["properties"] = properties
        return self.client._call("POST", "/analytics/events", data)

    def list_events(self, params: dict = None):
        query = ""
        if params:
            parts = [f"{key}={value}" for key, value in params.items() if value is not None]
            if parts:
                query = "?" + "&".join(parts)
        return self.client._call("GET", f"/analytics/events{query}")

    def get_stats(self, params: dict = None):
        query = ""
        if params:
            parts = [f"{key}={value}" for key, value in params.items() if value is not None]
            if parts:
                query = "?" + "&".join(parts)
        return self.client._call("GET", f"/analytics/stats{query}")

    def get_realtime_count(self):
        return self.client._call("GET", "/analytics/realtime")