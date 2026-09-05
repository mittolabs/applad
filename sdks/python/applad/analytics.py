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

    def get_overview(self):
        """Events, active users, request latency and uptime for the last 24h."""
        return self.client._call("GET", "/analytics/overview")

    def get_performance(self):
        """Per-route request latency measured by the platform over 24h."""
        return self.client._call("GET", "/analytics/performance")

    # ── Uptime monitors ──────────────────────────────────────────────────────

    def list_monitors(self):
        return self.client._call("GET", "/analytics/uptime")

    def create_monitor(self, name: str, url: str, check_type: str = "http",
                       interval_secs: int = 60, keyword: str = None):
        data = {
            "name": name,
            "url": url,
            "checkType": check_type,
            "intervalSecs": interval_secs,
        }
        if keyword is not None:
            data["keyword"] = keyword
        return self.client._call("POST", "/analytics/uptime", data)

    def delete_monitor(self, monitor_id: str):
        return self.client._call("DELETE", f"/analytics/uptime/{monitor_id}")

    # ── Cron monitors ────────────────────────────────────────────────────────

    def list_cron_monitors(self):
        return self.client._call("GET", "/analytics/crons")

    def create_cron_monitor(self, name: str, schedule: str,
                            timezone: str = "UTC", grace_period: int = 5):
        return self.client._call("POST", "/analytics/crons", {
            "name": name,
            "schedule": schedule,
            "timezone": timezone,
            "gracePeriod": grace_period,
        })

    def delete_cron_monitor(self, monitor_id: str):
        return self.client._call("DELETE", f"/analytics/crons/{monitor_id}")

    def cron_checkin(self, monitor_id: str, status: str = "ok",
                     duration_ms: int = None, error_msg: str = None):
        """Report a run of a scheduled job.

        A monitor that hears nothing within its grace period is marked missed.
        """
        data = {"status": status}
        if duration_ms is not None:
            data["durationMs"] = duration_ms
        if error_msg is not None:
            data["errorMsg"] = error_msg
        return self.client._call("POST", f"/analytics/crons/{monitor_id}/checkin", data)
