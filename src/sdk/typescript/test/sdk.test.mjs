import test from "node:test";
import assert from "node:assert/strict";
import { BridgeClient, BridgeServer, BridgeValidationError } from "../dist/index.js";
const request = {schema_version:"bridge.route.request.v1",request_id:"sdk-ts",graph:{type:"inline",nodes:[{id:0},{id:1}],edges:[{from:0,to:1,weight:1}]},route:{source:0,target:1,route_mode:"balanced"},observation_config:{level:"minimum"}};
test("bundled route", async () => { const c = await BridgeClient.solver(); const r = await c.route(request); assert.equal(r.result.path_found, true); assert.equal(r.result.path_cost, 1); });
test("validation error", async () => { const c = await BridgeClient.solver(); await assert.rejects(() => c.route({...request, schema_version:"bad"}), (error) => { assert.ok(error instanceof BridgeValidationError); assert.equal(error.code, "INVALID_SCHEMA_VERSION"); assert.equal(error.category, "validation"); assert.equal(error.retryable, false); return true; }); });
test("environment override", async () => { const c = await BridgeClient.solver(); const old = process.env.BRIDGE_BINARY; process.env.BRIDGE_BINARY = c.binaryPath; try { assert.equal(await (await BridgeClient.solver()).version(), "0.15.3"); } finally { if (old === undefined) delete process.env.BRIDGE_BINARY; else process.env.BRIDGE_BINARY = old; } });

test("already aborted signal", async () => { const c = await BridgeClient.solver(); const controller = new AbortController(); controller.abort(); await assert.rejects(() => c.route(request, {signal: controller.signal}), /cancelled/); });


test("managed server route", async () => {
  const server = await BridgeServer.start();
  try {
    const client = await server.client();
    const response = await client.route(request);
    assert.equal(response.result.path_found, true);
    assert.equal(response.result.path_cost, 1);
    assert.equal(client.baseUrl, server.baseUrl);
  } finally {
    await server.stop();
  }
});

test("server client", async () => {
  const server = await BridgeServer.start();
  try {
    const client = await BridgeClient.server(server.baseUrl);
    assert.equal(await client.version(), "0.15.3");
  } finally {
    await server.stop();
  }
});
