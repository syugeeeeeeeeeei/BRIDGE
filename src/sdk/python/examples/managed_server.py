from bridge_sdk import BridgeServer
from common import REQUEST

with BridgeServer.start() as server:
    print(server.client().route(REQUEST).result)
