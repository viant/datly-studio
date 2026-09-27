import assert from 'node:assert/strict';
import test from 'node:test';

import { verifyLiveReader } from './verify-live-reader.mjs';

const origin = 'https://studio.example';
const config = {
  token: 'signed-token', uiOrigin: origin,
  sdkURL: 'https://sdk.example/v1/studio/sdk/namespaces.list',
  staticMCPURL: 'https://sdk.example/mcp',
  readerURL: 'https://reader.example/records',
  dynamicMCPURL: 'https://reader.example/mcp',
  readerTool: 'records.list', readerArguments: { limit: 1 }, expectedText: '"id":1',
};

function response(body, status = 200, headers = {}) {
  return new Response(JSON.stringify(body), { status, headers: {
    'Content-Type': 'application/json', 'Access-Control-Allow-Origin': origin, ...headers,
  } });
}

function deployment(change = (original) => original) {
  const calls = [];
  const fetcher = async (url, options) => {
    calls.push({ url: String(url), options });
    const address = String(url);
    if (address === config.sdkURL && !options.headers.Authorization) return response({ code: 'unauthorized' }, 401);
    if (address === config.sdkURL) return change(response({ items: [] }), address, options);
    if (address === config.readerURL && !options.headers.Authorization) return change(response({ code: 'unauthorized' }, 401), address, options);
    if (address === config.readerURL) return change(response({ data: [{ id: 1 }] }), address, options);
    const method = JSON.parse(options.body).method;
    if (address === config.staticMCPURL && method === 'tools/list') return change(response({ jsonrpc: '2.0', id: 1, result: { tools: [{ name: 'studio.sdk.namespaces.list' }] } }), address, options);
    if (address === config.staticMCPURL && method === 'tools/call') return change(response({ jsonrpc: '2.0', id: 1, result: { structuredContent: { items: [] } } }), address, options);
    if (address === config.dynamicMCPURL && method === 'tools/list') return change(response({ jsonrpc: '2.0', id: 1, result: { tools: [{ name: 'records.list' }] } }), address, options);
    if (address === config.dynamicMCPURL && method === 'tools/call') return change(response({ jsonrpc: '2.0', id: 1, result: { structuredContent: { data: [{ id: 1 }] } } }), address, options);
    throw new Error(`unexpected URL ${address}`);
  };
  return { calls, fetcher };
}

test('live reader smoke checks direct SDK, both MCP hosts, CORS and no cookie', async () => {
  const { calls, fetcher } = deployment();
  const result = await verifyLiveReader(config, fetcher);
  assert.deepEqual(result, { sdk: 'ok', staticMCP: 'ok', readerHTTP: 'ok', readerMCP: 'ok', tool: 'records.list' });
  assert.equal(calls.length, 8);
  assert.equal(calls[0].options.headers.Authorization, undefined);
  assert.equal(calls[4].options.headers.Authorization, undefined);
  for (const call of calls.filter((_, index) => index !== 0 && index !== 4)) {
    assert.equal(call.options.headers.Authorization, 'Bearer signed-token');
    assert.equal(call.options.headers.Origin, origin);
    assert.equal(call.options.redirect, 'manual');
  }
  assert.equal(JSON.parse(calls.at(-1).options.body).params.arguments.limit, 1);
  assert.equal(JSON.parse(calls[3].options.body).params.name, 'studio.sdk.namespaces.list');
  assert.ok(!JSON.stringify(result).includes(config.token));
});

test('live reader smoke rejects credentialed CORS and bearer cookies', async () => {
  const cookie = deployment((original, address, options) => address === config.readerURL && options.headers.Authorization
    ? response({ data: [] }, 200, { 'Set-Cookie': 'session=unsafe' }) : original);
  await assert.rejects(verifyLiveReader(config, cookie.fetcher), /set a cookie/);
  const cors = deployment((original, address) => address === config.sdkURL
    ? response({ items: [] }, 200, { 'Access-Control-Allow-Origin': '*' }) : original);
  await assert.rejects(verifyLiveReader(config, cors.fetcher), /exact UI origin/);
  const credentials = deployment((original, address) => address === config.sdkURL
    ? response({ items: [] }, 200, { 'Access-Control-Allow-Credentials': 'true' }) : original);
  await assert.rejects(verifyLiveReader(config, credentials.fetcher), /credentialed CORS/);
});

test('live reader smoke rejects an anonymously exposed reader or unsafe denial headers', async () => {
  const publicReader = deployment((original, address, options) => address === config.readerURL && !options.headers.Authorization
    ? response({ data: [{ id: 1 }] }) : original);
  await assert.rejects(verifyLiveReader(config, publicReader.fetcher), /published reader HTTP without a bearer returned HTTP 200/);
  const denialCookie = deployment((original, address, options) => address === config.readerURL && !options.headers.Authorization
    ? response({ code: 'unauthorized' }, 401, { 'Set-Cookie': 'session=unsafe' }) : original);
  await assert.rejects(verifyLiveReader(config, denialCookie.fetcher), /set a cookie/);
  const denialCors = deployment((original, address, options) => address === config.readerURL && !options.headers.Authorization
    ? response({ code: 'unauthorized' }, 401, { 'Access-Control-Allow-Origin': '*' }) : original);
  await assert.rejects(verifyLiveReader(config, denialCors.fetcher), /exact UI origin/);
});

test('live reader smoke fails closed on missing MCP tool and unsafe URL', async () => {
  const missing = deployment((original, address, options) => address === config.dynamicMCPURL && JSON.parse(options.body).method === 'tools/list'
    ? response({ jsonrpc: '2.0', id: 1, result: { tools: [] } }) : original);
  await assert.rejects(verifyLiveReader(config, missing.fetcher), /published reader tool/);
  await assert.rejects(verifyLiveReader({ ...config, sdkURL: 'http://remote.example/sdk' }, deployment().fetcher), /requires HTTPS/);
  await assert.rejects(verifyLiveReader({ ...config, sdkURL: 'https://sdk.example/v1/studio/sdk/access.replace' }, deployment().fetcher), /read-only namespaces.list/);
});

test('live reader smoke requires an executable native SDK MCP tool', async () => {
  const missingResult = deployment((original, address, options) => address === config.staticMCPURL && JSON.parse(options.body).method === 'tools/call'
    ? response({ jsonrpc: '2.0', id: 1 }) : original);
  await assert.rejects(verifyLiveReader(config, missingResult.fetcher), /no MCP result/);
  const failedTool = deployment((original, address, options) => address === config.staticMCPURL && JSON.parse(options.body).method === 'tools/call'
    ? response({ jsonrpc: '2.0', id: 1, result: { isError: true } }) : original);
  await assert.rejects(verifyLiveReader(config, failedTool.fetcher), /returned a tool error/);
});

test('live reader smoke accepts streamable MCP JSON-RPC events', async () => {
  const sse = deployment((original, address, options) => {
    if (address !== config.staticMCPURL && address !== config.dynamicMCPURL) return original;
    const method = JSON.parse(options.body).method;
    return new Response(`event: message\ndata: ${JSON.stringify({ jsonrpc: '2.0', id: 1, result: address === config.staticMCPURL
      ? method === 'tools/call' ? { structuredContent: { items: [] } } : { tools: [{ name: 'studio.sdk.namespaces.list' }] }
      : { tools: [{ name: 'records.list' }] } })}\n\n`, { status: 200, headers: {
      'Content-Type': 'text/event-stream', 'Access-Control-Allow-Origin': origin,
    } });
  });
  // Only catalog calls are replaced here; the tool call retains its JSON body.
  const fetcher = async (url, options) => {
    if (String(url) === config.dynamicMCPURL && JSON.parse(options.body).method === 'tools/call') {
      return response({ jsonrpc: '2.0', id: 1, result: { structuredContent: { data: [{ id: 1 }] } } });
    }
    return sse.fetcher(url, options);
  };
  assert.equal((await verifyLiveReader(config, fetcher)).readerMCP, 'ok');
});

test('live reader smoke selects its MCP response after notifications and rejects mismatched IDs', async () => {
  const event = (payload) => `event: message\ndata: ${JSON.stringify(payload)}\n\n`;
  const notification = event({ jsonrpc: '2.0', method: 'notifications/progress', params: { progress: 1 } });
  const unrelated = event({ jsonrpc: '2.0', id: 2, result: { tools: [] } });
  const matching = event({ jsonrpc: '2.0', id: 1, result: { tools: [{ name: 'studio.sdk.namespaces.list' }] } });
  const stream = (body) => new Response(body, { status: 200, headers: {
    'Content-Type': 'text/event-stream', 'Access-Control-Allow-Origin': origin,
  } });
  const ordered = deployment((original, address, options) => address === config.staticMCPURL && JSON.parse(options.body).method === 'tools/list'
    ? stream(notification + unrelated + matching) : original);
  assert.equal((await verifyLiveReader(config, ordered.fetcher)).staticMCP, 'ok');
  const wrongID = deployment((original, address, options) => address === config.staticMCPURL && JSON.parse(options.body).method === 'tools/list'
    ? stream(notification + unrelated) : original);
  await assert.rejects(verifyLiveReader(config, wrongID.fetcher), /no matching JSON-RPC response/);
  const malformed = deployment((original, address, options) => address === config.staticMCPURL && JSON.parse(options.body).method === 'tools/list'
    ? stream('event: message\ndata: {secret:signed-token}\n\n') : original);
  await assert.rejects(verifyLiveReader(config, malformed.fetcher), (error) => {
    assert.match(error.message, /invalid JSON-RPC payload/);
    assert.ok(!error.message.includes(config.token));
    return true;
  });
});
