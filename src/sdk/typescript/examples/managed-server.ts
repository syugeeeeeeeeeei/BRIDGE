import { BridgeServer } from "../src/index.js";
import { request } from "./common.js";
const server = await BridgeServer.start();
try { console.log((await (await server.client()).route(request)).result); }
finally { await server.stop(); }
