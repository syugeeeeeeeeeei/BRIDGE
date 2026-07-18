import { BridgeClient } from "../src/index.js";
import { request } from "./common.js";
console.log((await (await BridgeClient.solver()).route(request)).result);
