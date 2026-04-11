"""Observe service — error tracking, logs, performance, releases, replays,
uptime monitors, cron monitors, and alert rules."""


class Observe:
    def __init__(self, client):
        self.client = client

    # ── Overview ──────────────────────────────────────────────────────────────

    def get_overview(self):
        """Get observability overview (stats, health, recent errors)."""
        return self.client._call("GET", "/observe/overview")

    # ── Errors ────────────────────────────────────────────────────────────────

    def capture_error(self, title: str, *, error_type: str = None, level: str = "error",
                      stack_trace: str = None, breadcrumbs: list = None,
                      user_context: dict = None, request_context: dict = None,
                      runtime_context: dict = None, tags: dict = None,
                      environment: str = "production", release: str = None):
        """Capture an error event."""
        data = {"title": title, "level": level, "environment": environment}
        if error_type is not None:
            data["errorType"] = error_type
        if stack_trace is not None:
            data["stackTrace"] = stack_trace
        if breadcrumbs is not None:
            data["breadcrumbs"] = breadcrumbs
        if user_context is not None:
            data["userContext"] = user_context
        if request_context is not None:
            data["requestContext"] = request_context
        if runtime_context is not None:
            data["runtimeContext"] = runtime_context
        if tags is not None:
            data["tags"] = tags
        if release is not None:
            data["release"] = release
        return self.client._call("POST", "/observe/errors", data)

    def list_errors(self, *, status: str = None, level: str = None, limit: int = None):
        """List errors with optional filters."""
        params = {}
        if status is not None:
            params["status"] = status
        if level is not None:
            params["level"] = level
        if limit is not None:
            params["limit"] = str(limit)
        path = "/observe/errors"
        if params:
            from urllib.parse import urlencode
            path += "?" + urlencode(params)
        return self.client._call("GET", path)

    def get_error(self, error_id: str):
        """Get a single error with full detail."""
        return self.client._call("GET", f"/observe/errors/{error_id}")

    def resolve_error(self, error_id: str):
        """Mark an error as resolved."""
        return self.client._call("PATCH", f"/observe/errors/{error_id}/resolve")

    def ignore_error(self, error_id: str):
        """Mark an error as ignored."""
        return self.client._call("PATCH", f"/observe/errors/{error_id}/ignore")

    def unresolve_error(self, error_id: str):
        """Reopen a resolved or ignored error."""
        return self.client._call("PATCH", f"/observe/errors/{error_id}/unresolve")

    def set_priority(self, error_id: str, priority: str):
        """Set error priority (P1–P4)."""
        return self.client._call("PATCH", f"/observe/errors/{error_id}/priority",
                                  {"priority": priority})

    def assign_error(self, error_id: str, assignee: str):
        """Assign an error to a team member."""
        return self.client._call("PATCH", f"/observe/errors/{error_id}/assign",
                                  {"assignee": assignee})

    def add_note(self, error_id: str, text: str):
        """Add a note to an error's activity feed."""
        return self.client._call("POST", f"/observe/errors/{error_id}/activity",
                                  {"text": text})

    # ── Logs ──────────────────────────────────────────────────────────────────

    def capture_log(self, message: str, *, level: str = "info", source: str = None,
                    environment: str = None, release: str = None, meta: dict = None,
                    trace_id: str = None, span_id: str = None):
        """Capture a log entry."""
        data = {"message": message, "level": level}
        if source is not None:
            data["source"] = source
        if environment is not None:
            data["environment"] = environment
        if release is not None:
            data["release"] = release
        if meta is not None:
            data["meta"] = meta
        if trace_id is not None:
            data["traceId"] = trace_id
        if span_id is not None:
            data["spanId"] = span_id
        return self.client._call("POST", "/observe/logs", data)

    def list_logs(self, *, level: str = None, source: str = None, limit: int = None):
        """List log entries with optional filters."""
        params = {}
        if level is not None:
            params["level"] = level
        if source is not None:
            params["source"] = source
        if limit is not None:
            params["limit"] = str(limit)
        path = "/observe/logs"
        if params:
            from urllib.parse import urlencode
            path += "?" + urlencode(params)
        return self.client._call("GET", path)

    # ── Performance ───────────────────────────────────────────────────────────

    def get_performance(self):
        """Get aggregated performance metrics."""
        return self.client._call("GET", "/observe/performance")

    def record_perf(self, path: str, *, method: str = "GET",
                    p50_ms: float = 0, p75_ms: float = 0,
                    p95_ms: float = 0, p99_ms: float = 0,
                    rps: float = 0, error_pct: float = 0, req_count: int = 1):
        """Record a performance snapshot for an endpoint."""
        return self.client._call("POST", "/observe/performance", {
            "path": path, "method": method,
            "p50Ms": p50_ms, "p75Ms": p75_ms,
            "p95Ms": p95_ms, "p99Ms": p99_ms,
            "rps": rps, "errorPct": error_pct, "reqCount": req_count,
        })

    def report_web_vitals(self, *, page_url: str = None, lcp: float = None,
                          fid: float = None, cls: float = None, ttfb: float = None,
                          fcp: float = None, inp: float = None):
        """Report Core Web Vitals."""
        data = {}
        if page_url is not None:
            data["pageUrl"] = page_url
        if lcp is not None:
            data["lcp"] = lcp
        if fid is not None:
            data["fid"] = fid
        if cls is not None:
            data["cls"] = cls
        if ttfb is not None:
            data["ttfb"] = ttfb
        if fcp is not None:
            data["fcp"] = fcp
        if inp is not None:
            data["inp"] = inp
        return self.client._call("POST", "/observe/performance/vitals", data)

    # ── Releases ──────────────────────────────────────────────────────────────

    def list_releases(self):
        """List all releases."""
        return self.client._call("GET", "/observe/releases")

    def create_release(self, version: str, *, environment: str = "production",
                       commits: list = None):
        """Create a release."""
        data = {"version": version, "environment": environment}
        if commits is not None:
            data["commits"] = commits
        return self.client._call("POST", "/observe/releases", data)

    def get_release(self, release_id: str):
        """Get a release by ID."""
        return self.client._call("GET", f"/observe/releases/{release_id}")

    # ── Cron checkin ──────────────────────────────────────────────────────────

    def cron_checkin(self, monitor_id: str, *, status: str = "ok",
                     duration_ms: int = None, error_msg: str = None):
        """Send a cron job check-in."""
        data = {"status": status}
        if duration_ms is not None:
            data["durationMs"] = duration_ms
        if error_msg is not None:
            data["errorMsg"] = error_msg
        return self.client._call("POST", f"/observe/crons/{monitor_id}/checkin", data)
