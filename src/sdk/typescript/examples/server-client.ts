import { BridgeClient } from "../src/index.js";
import { request } from "./common.js";
console.log((await (await BridgeClient.server("http://127.0.0.1:8080")).route(request)).result);
