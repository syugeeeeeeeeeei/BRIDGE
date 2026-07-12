import test from "node:test";
import assert from "node:assert/strict";
import { BridgeClient, BridgeValidationError } from "../dist/index.js";
const request = {schema_version:"bridge.route.request.v2",request_id:"sdk-ts",graph:{type:"inline",nodes:[{id:0},{id:1}],edges:[{from:0,to:1,weight:1}]},route:{source:0,target:1,route_mode:"balanced"},observation_config:{level:"off"}};
test("bundled route", async () => { const c = await BridgeClient.local(); const r = await c.route(request); assert.equal(r.result.found, true); assert.equal(r.result.distance, 1); });
test("validation error", async () => { const c = await BridgeClient.local(); await assert.rejects(() => c.route({...request, schema_version:"bad"}), BridgeValidationError); });
test("environment override", async () => { const c = await BridgeClient.local(); const old = process.env.BRIDGE_BINARY; process.env.BRIDGE_BINARY = c.binaryPath; try { assert.equal(await (await BridgeClient.local()).version(), "0.14.0"); } finally { if (old === undefined) delete process.env.BRIDGE_BINARY; else process.env.BRIDGE_BINARY = old; } });

test("already aborted signal", async () => { const c = await BridgeClient.local(); const controller = new AbortController(); controller.abort(); await assert.rejects(() => c.route(request, {signal: controller.signal}), /cancelled/); });
