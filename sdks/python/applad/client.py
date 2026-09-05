"""Applad API client."""

import json
import urllib.request
import urllib.error


class Client:
    """Server-side client for the Applad BaaS API.

    Uses only stdlib (urllib) -- no external dependencies required.
    """

    def __init__(self, endpoint: str, project_id: str, api_key: str) -> None:
        self.endpoint = endpoint.rstrip("/")
        self.project_id = project_id
        self.api_key = api_key
        self._headers = {
            "Content-Type": "application/json",
            "X-Applad-Project": project_id,
            "X-Applad-Key": api_key,
        }

        # Lazy-initialised service instances
        self._users = None
        self._databases = None
        self._storage = None
        self._functions = None
        self._teams = None
        self._workflows = None
        self._messaging = None
        self._deploy = None
        self._flags = None
        self._analytics = None
        self._search = None
        self._vectors = None
        self._edge = None
        self._regions = None

    # -----------------------------------------------------------------
    # Internal helpers
    # -----------------------------------------------------------------

    def _call(self, method: str, path: str, data=None):
        """Make an authenticated JSON request to the API."""
        url = f"{self.endpoint}/v1{path}"
        body = json.dumps(data).encode("utf-8") if data is not None else None

        req = urllib.request.Request(url, data=body, headers=self._headers, method=method)

        try:
            with urllib.request.urlopen(req) as resp:
                if resp.status == 204:
                    return None
                return json.loads(resp.read().decode("utf-8"))
        except urllib.error.HTTPError as exc:
            error_body = exc.read().decode("utf-8", errors="replace")
            raise Exception(
                f"Applad API error: {method} {path} returned {exc.code}: {error_body}"
            ) from exc

    # -----------------------------------------------------------------
    # Service properties
    # -----------------------------------------------------------------

    @property
    def users(self):
        if self._users is None:
            from .users import Users
            self._users = Users(self)
        return self._users

    @property
    def databases(self):
        if self._databases is None:
            from .databases import Databases
            self._databases = Databases(self)
        return self._databases

    @property
    def storage(self):
        if self._storage is None:
            from .storage import Storage
            self._storage = Storage(self)
        return self._storage

    @property
    def functions(self):
        if self._functions is None:
            from .functions import Functions
            self._functions = Functions(self)
        return self._functions

    @property
    def teams(self):
        if self._teams is None:
            from .teams import Teams
            self._teams = Teams(self)
        return self._teams

    @property
    def workflows(self):
        if self._workflows is None:
            from .workflows import Workflows
            self._workflows = Workflows(self)
        return self._workflows

    @property
    def messaging(self):
        if self._messaging is None:
            from .messaging import Messaging
            self._messaging = Messaging(self)
        return self._messaging

    @property
    def deploy(self):
        if self._deploy is None:
            from .deploy import Deploy
            self._deploy = Deploy(self)
        return self._deploy

    @property
    def flags(self):
        if self._flags is None:
            from .flags import Flags
            self._flags = Flags(self)
        return self._flags

    @property
    def analytics(self):
        if self._analytics is None:
            from .analytics import Analytics
            self._analytics = Analytics(self)
        return self._analytics

    @property
    def search(self):
        if self._search is None:
            from .search import Search
            self._search = Search(self)
        return self._search

    @property
    def vectors(self):
        if self._vectors is None:
            from .vectors import Vectors
            self._vectors = Vectors(self)
        return self._vectors

    @property
    def edge(self):
        if self._edge is None:
            from .edge import Edge
            self._edge = Edge(self)
        return self._edge

    @property
    def regions(self):
        if self._regions is None:
            from .regions import Regions
            self._regions = Regions(self)
        return self._regions

