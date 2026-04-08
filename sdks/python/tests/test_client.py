"""Tests for the Applad Python SDK."""

import json
import unittest
from unittest.mock import patch, MagicMock

from applad.client import Client


def _mock_response(data=None, status=200):
    """Create a mock urllib response."""
    resp = MagicMock()
    resp.status = status
    resp.read.return_value = json.dumps(data or {}).encode("utf-8")
    resp.__enter__ = MagicMock(return_value=resp)
    resp.__exit__ = MagicMock(return_value=False)
    return resp


class TestClient(unittest.TestCase):
    def setUp(self):
        self.client = Client("http://localhost:8080", "proj-1", "key-123")

    def test_init(self):
        self.assertEqual(self.client.endpoint, "http://localhost:8080")
        self.assertEqual(self.client.project_id, "proj-1")
        self.assertEqual(self.client.api_key, "key-123")

    def test_strips_trailing_slash(self):
        c = Client("http://localhost:8080/", "p", "k")
        self.assertEqual(c.endpoint, "http://localhost:8080")

    def test_service_accessors(self):
        self.assertIsNotNone(self.client.users)
        self.assertIsNotNone(self.client.databases)
        self.assertIsNotNone(self.client.storage)
        self.assertIsNotNone(self.client.functions)
        self.assertIsNotNone(self.client.teams)
        self.assertIsNotNone(self.client.workflows)
        self.assertIsNotNone(self.client.messaging)
        self.assertIsNotNone(self.client.deploy)
        self.assertIsNotNone(self.client.flags)
        self.assertIsNotNone(self.client.analytics)
        self.assertIsNotNone(self.client.search)
        self.assertIsNotNone(self.client.vectors)
        self.assertIsNotNone(self.client.edge)
        self.assertIsNotNone(self.client.billing)
        self.assertIsNotNone(self.client.regions)

    @patch("urllib.request.urlopen")
    def test_call_sends_headers(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"ok": True})
        self.client._call("GET", "/health")
        req = mock_urlopen.call_args[0][0]
        self.assertEqual(req.get_header("X-applad-project"), "proj-1")
        self.assertEqual(req.get_header("X-applad-key"), "key-123")
        self.assertEqual(req.get_header("Content-type"), "application/json")

    @patch("urllib.request.urlopen")
    def test_call_builds_url(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({})
        self.client._call("GET", "/databases")
        req = mock_urlopen.call_args[0][0]
        self.assertEqual(req.full_url, "http://localhost:8080/v1/databases")

    @patch("urllib.request.urlopen")
    def test_call_returns_none_on_204(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response(status=204)
        result = self.client._call("DELETE", "/resource")
        self.assertIsNone(result)


class TestUsers(unittest.TestCase):
    def setUp(self):
        self.client = Client("http://localhost:8080", "proj-1", "key-123")

    @patch("urllib.request.urlopen")
    def test_list_users(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"users": [], "total": 0})
        result = self.client.users.list_users()
        self.assertEqual(result["users"], [])
        req = mock_urlopen.call_args[0][0]
        self.assertIn("/users", req.full_url)

    @patch("urllib.request.urlopen")
    def test_create_user(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"$id": "u1"})
        result = self.client.users.create_user("a@b.com", "pass123")
        self.assertEqual(result["$id"], "u1")

    @patch("urllib.request.urlopen")
    def test_get_user(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"$id": "u1", "email": "a@b.com"})
        result = self.client.users.get_user("u1")
        self.assertEqual(result["$id"], "u1")

    @patch("urllib.request.urlopen")
    def test_delete_user(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response(status=204)
        self.client.users.delete_user("u1")


class TestDatabases(unittest.TestCase):
    def setUp(self):
        self.client = Client("http://localhost:8080", "proj-1", "key-123")

    @patch("urllib.request.urlopen")
    def test_list_databases(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"databases": []})
        result = self.client.databases.list_databases()
        self.assertEqual(result["databases"], [])

    @patch("urllib.request.urlopen")
    def test_create_database(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"$id": "db1", "name": "mydb"})
        result = self.client.databases.create_database("mydb")
        self.assertEqual(result["name"], "mydb")

    @patch("urllib.request.urlopen")
    def test_create_table(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"$id": "t1"})
        result = self.client.databases.create_table("db1", "users")
        self.assertEqual(result["$id"], "t1")

    @patch("urllib.request.urlopen")
    def test_list_rows(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"documents": [], "total": 0})
        result = self.client.databases.list_rows("db1", "t1")
        self.assertEqual(result["documents"], [])


class TestStorage(unittest.TestCase):
    def setUp(self):
        self.client = Client("http://localhost:8080", "proj-1", "key-123")

    @patch("urllib.request.urlopen")
    def test_list_buckets(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"buckets": []})
        result = self.client.storage.list_buckets()
        self.assertEqual(result["buckets"], [])


class TestFunctions(unittest.TestCase):
    def setUp(self):
        self.client = Client("http://localhost:8080", "proj-1", "key-123")

    @patch("urllib.request.urlopen")
    def test_list(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"functions": []})
        result = self.client.functions.list()
        self.assertEqual(result["functions"], [])

    @patch("urllib.request.urlopen")
    def test_execute(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"output": "hello"})
        result = self.client.functions.execute("fn1", {"input": "world"})
        self.assertEqual(result["output"], "hello")


class TestWorkflows(unittest.TestCase):
    def setUp(self):
        self.client = Client("http://localhost:8080", "proj-1", "key-123")

    @patch("urllib.request.urlopen")
    def test_list(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"workflows": []})
        result = self.client.workflows.list()
        self.assertEqual(result["workflows"], [])

    @patch("urllib.request.urlopen")
    def test_execute(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"executionId": "ex1"})
        result = self.client.workflows.execute("wf1")
        self.assertEqual(result["executionId"], "ex1")


class TestFlags(unittest.TestCase):
    def setUp(self):
        self.client = Client("http://localhost:8080", "proj-1", "key-123")

    @patch("urllib.request.urlopen")
    def test_list(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"flags": []})
        result = self.client.flags.list()
        self.assertEqual(result["flags"], [])

    @patch("urllib.request.urlopen")
    def test_create(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"key": "dark-mode"})
        result = self.client.flags.create("dark-mode", "Dark Mode")
        self.assertEqual(result["key"], "dark-mode")

    @patch("urllib.request.urlopen")
    def test_evaluate(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"value": True})
        result = self.client.flags.get_flag("dark-mode")
        self.assertTrue(result["value"])

    @patch("urllib.request.urlopen")
    def test_toggle(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"enabled": False})
        result = self.client.flags.toggle("dark-mode", False)
        self.assertFalse(result["enabled"])


class TestMessaging(unittest.TestCase):
    def setUp(self):
        self.client = Client("http://localhost:8080", "proj-1", "key-123")

    @patch("urllib.request.urlopen")
    def test_send_email(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"success": True})
        result = self.client.messaging.send_email(["a@b.com"], "Test")
        self.assertTrue(result["success"])

    @patch("urllib.request.urlopen")
    def test_send_sms(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"success": True})
        result = self.client.messaging.send_sms(["+1234567890"], "Hello")
        self.assertTrue(result["success"])


class TestDeploy(unittest.TestCase):
    def setUp(self):
        self.client = Client("http://localhost:8080", "proj-1", "key-123")

    @patch("urllib.request.urlopen")
    def test_list(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"deployments": []})
        result = self.client.deploy.list()
        self.assertEqual(result["deployments"], [])

    @patch("urllib.request.urlopen")
    def test_create(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"$id": "d1"})
        result = self.client.deploy.create("staging", "docker")
        self.assertEqual(result["$id"], "d1")


class TestAnalytics(unittest.TestCase):
    def setUp(self):
        self.client = Client("http://localhost:8080", "proj-1", "key-123")

    @patch("urllib.request.urlopen")
    def test_track_event(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"success": True})
        result = self.client.analytics.track_event("signup", {"source": "web"})
        self.assertTrue(result["success"])

    @patch("urllib.request.urlopen")
    def test_get_stats(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"events": 12})
        result = self.client.analytics.get_stats({"event": "signup"})
        self.assertEqual(result["events"], 12)


class TestSearch(unittest.TestCase):
    def setUp(self):
        self.client = Client("http://localhost:8080", "proj-1", "key-123")

    @patch("urllib.request.urlopen")
    def test_create_index(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"$id": "idx1"})
        result = self.client.search.create_index("idx1", name="Posts")
        self.assertEqual(result["$id"], "idx1")

    @patch("urllib.request.urlopen")
    def test_query(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"documents": []})
        result = self.client.search.query("idx1", "hello")
        self.assertEqual(result["documents"], [])


class TestVectors(unittest.TestCase):
    def setUp(self):
        self.client = Client("http://localhost:8080", "proj-1", "key-123")

    @patch("urllib.request.urlopen")
    def test_create_index(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"$id": "vec1"})
        result = self.client.vectors.create_index("vec1", 1536)
        self.assertEqual(result["$id"], "vec1")

    @patch("urllib.request.urlopen")
    def test_query(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"matches": []})
        result = self.client.vectors.query("vec1", [0.1, 0.2])
        self.assertEqual(result["matches"], [])


class TestEdge(unittest.TestCase):
    def setUp(self):
        self.client = Client("http://localhost:8080", "proj-1", "key-123")

    @patch("urllib.request.urlopen")
    def test_create(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"$id": "edge1"})
        result = self.client.edge.create("proxy", "export default {}")
        self.assertEqual(result["$id"], "edge1")

    @patch("urllib.request.urlopen")
    def test_invoke(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"ok": True})
        result = self.client.edge.invoke("edge1", {"name": "test"})
        self.assertTrue(result["ok"])


class TestBilling(unittest.TestCase):
    def setUp(self):
        self.client = Client("http://localhost:8080", "proj-1", "key-123")

    @patch("urllib.request.urlopen")
    def test_list_plans(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"plans": []})
        result = self.client.billing.list_plans()
        self.assertEqual(result["plans"], [])

    @patch("urllib.request.urlopen")
    def test_get_usage(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"requests": 100})
        result = self.client.billing.get_usage()
        self.assertEqual(result["requests"], 100)


class TestRegions(unittest.TestCase):
    def setUp(self):
        self.client = Client("http://localhost:8080", "proj-1", "key-123")

    @patch("urllib.request.urlopen")
    def test_list(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"regions": []})
        result = self.client.regions.list()
        self.assertEqual(result["regions"], [])

    @patch("urllib.request.urlopen")
    def test_set_active(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"regionId": "fra1"})
        result = self.client.regions.set_active("fra1")
        self.assertEqual(result["regionId"], "fra1")


if __name__ == "__main__":
    unittest.main()
