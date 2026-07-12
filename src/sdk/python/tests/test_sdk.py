import os, sys, unittest
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parents[1] / "src"))
from bridge_sdk import BridgeClient, BridgeValidationError

REQ = {"schema_version":"bridge.route.v1","request_id":"sdk-python","graph":{"type":"inline","nodes":[{"id":0},{"id":1}],"edges":[{"from":0,"to":1,"weight":1.0}]},"route":{"source":0,"target":1,"mode":"balanced"},"observation":{"mode":"off"}}

class SDKTest(unittest.TestCase):
    def test_bundled_route(self):
        client = BridgeClient.local()
        response = client.route(REQ)
        self.assertTrue(response.result["found"])
        self.assertEqual(response.result["distance"], 1)
    def test_validation_error(self):
        client = BridgeClient.local()
        bad = dict(REQ); bad["schema_version"] = "bad"
        with self.assertRaises(BridgeValidationError): client.route(bad)
    def test_environment_override(self):
        client = BridgeClient.local()
        old = os.environ.get("BRIDGE_BINARY")
        os.environ["BRIDGE_BINARY"] = str(client.binary_path)
        try: self.assertEqual(BridgeClient.local().version(), "0.14.0")
        finally:
            if old is None: os.environ.pop("BRIDGE_BINARY", None)
            else: os.environ["BRIDGE_BINARY"] = old

class AsyncSDKTest(unittest.IsolatedAsyncioTestCase):
    async def test_async_route(self):
        client = BridgeClient.local()
        response = await client.route_async(REQ)
        self.assertTrue(response.result["found"])

if __name__ == "__main__": unittest.main()
