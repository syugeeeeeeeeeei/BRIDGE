import { BridgeClient, BridgeError, BridgeValidationError } from "../src/index.js";
import { request } from "./common.js";
try {
  const result = (await (await BridgeClient.solver()).route(request)).result;
  if (!result.path_found) console.log("経路なしは通信エラーではありません");
} catch (error) {
  if (error instanceof BridgeValidationError) console.error(`入力エラー: ${error.message}`);
  else if (error instanceof BridgeError) console.error(`SDKエラー: ${error.message}`);
  else throw error;
}
