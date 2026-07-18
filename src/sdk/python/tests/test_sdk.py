import os, sys, unittest
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parents[1] / "src"))
from bridge_sdk import BridgeClient, BridgeServer, BridgeValidationError

REQ = {"schema_version":"bridge.route.request.v1","request_id":"sdk-python","graph":{"type":"inline","nodes":[{"id":0},{"id":1}],"edges":[{"from":0,"to":1,"weight":1.0}]},"route":{"source":0,"target":1,"route_mode":"balanced"},"observation_config":{"level":"minimum"}}

class SDKTest(unittest.TestCase):
    def test_bundled_route(self):
        client = BridgeClient.solver()
        response = client.route(REQ)
        self.assertTrue(response.result["path_found"])
        self.assertEqual(response.result["path_cost"], 1)
    def test_validation_error(self):
        client = BridgeClient.solver()
        bad = dict(REQ); bad["schema_version"] = "bad"
        with self.assertRaises(BridgeValidationError) as raised: client.route(bad)
        self.assertEqual(raised.exception.code, "INVALID_SCHEMA_VERSION")
        self.assertEqual(raised.exception.category, "validation")
        self.assertFalse(raised.exception.retryable)
    def test_environment_override(self):
        client = BridgeClient.solver()
        old = os.environ.get("BRIDGE_BINARY")
        os.environ["BRIDGE_BINARY"] = str(client.binary_path)
        try: self.assertEqual(BridgeClient.solver().version(), "0.15.3")
        finally:
            if old is None: os.environ.pop("BRIDGE_BINARY", None)
            else: os.environ["BRIDGE_BINARY"] = old

class ServerSDKTest(unittest.TestCase):
    def test_managed_server_route(self):
        with BridgeServer.start() as server:
            client = server.client()
            response = client.route(REQ)
            self.assertTrue(response.result["path_found"])
            self.assertEqual(response.result["path_cost"], 1)
            self.assertEqual(client.base_url, server.base_url)

    def test_server_client(self):
        with BridgeServer.start() as server:
            client = BridgeClient.server(server.base_url)
            self.assertEqual(client.version(), "0.15.3")

class AsyncSDKTest(unittest.IsolatedAsyncioTestCase):
    async def test_async_route(self):
        client = BridgeClient.solver()
        response = await client.route_async(REQ)
        self.assertTrue(response.result["path_found"])

if __name__ == "__main__": unittest.main()
