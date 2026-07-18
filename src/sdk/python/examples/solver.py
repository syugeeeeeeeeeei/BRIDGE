from bridge_sdk import BridgeClient
from common import REQUEST

response = BridgeClient.solver(default_timeout=10).route(REQUEST)
print(response.result)
