from bridge_sdk import BridgeClient, BridgeError, BridgeValidationError
from common import REQUEST

try:
    result = BridgeClient.solver().route(REQUEST).result
    if not result["path_found"]:
        print("経路は見つかりませんでした。これは通信エラーではありません。")
except BridgeValidationError as exc:
    print(f"入力エラー: {exc}")
except BridgeError as exc:
    print(f"BRIDGE SDKエラー: {exc}")
