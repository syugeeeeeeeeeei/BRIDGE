import asyncio
from bridge_sdk import BridgeClient
from common import REQUEST

async def main() -> None:
    client = BridgeClient.solver(default_timeout=10)
    print((await client.route_async(REQUEST)).result)

asyncio.run(main())
