import { BridgeClient, BridgeCancelledError } from "../src/index.js";
import { request } from "./common.js";
const client = await BridgeClient.solver();
const controller = new AbortController();
const promise = client.route(request, { signal: controller.signal });
controller.abort();
try { await promise; } catch (error) {
  if (error instanceof BridgeCancelledError) console.log("キャンセルされました");
  else throw error;
}
