import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const protocol = '2026-07-28';
const meta = {
  'io.modelcontextprotocol/protocolVersion': protocol,
  'io.modelcontextprotocol/clientCapabilities': {},
};

function secureURL(value, name) {
  const url = new URL(value);
  const local = url.hostname === 'localhost' || url.hostname === '127.0.0.1' || url.hostname === '[::1]';
  assert.ok(url.protocol === 'https:' || (url.protocol === 'http:' && local), `${name} requires HTTPS or loopback HTTP`);
  assert.equal(url.username, '', `${name} must not contain credentials`);
  assert.equal(url.password, '', `${name} must not contain credentials`);
  assert.equal(url.hash, '', `${name} must not contain a fragment`);
  return url;
}

function mustBeDirect(response, label, origin, status = 200) {
  assert.equal(response.status, status, `${label} returned HTTP ${response.status}`);
  assert.equal(response.headers.get('access-control-allow-origin'), origin, `${label} did not return the exact UI origin`);
  assert.notEqual(response.headers.get('access-control-allow-credentials'), 'true', `${label} enabled credentialed CORS`);
  assert.equal(response.headers.get('set-cookie'), null, `${label} set a cookie`);
}

async function jsonResponse(response, label) {
  try { return await response.json(); }
  catch { throw new Error(`${label} returned invalid JSON`); }
}

function mcpPayload(text, label) {
  const trimmed = text.trim();
  const parse = (value) => {
    try { return JSON.parse(value); }
    catch { throw new Error(`${label} returned invalid JSON-RPC payload`); }
  };
  if (trimmed.startsWith('{')) {
    const payload = parse(trimmed);
    if (payload?.jsonrpc === '2.0' && payload.id === 1) return payload;
  } else {
    for (const event of trimmed.split(/\r?\n\r?\n/)) {
      const data = event.split(/\r?\n/).filter((line) => line.startsWith('data:'))
        .map((line) => line.slice(5).trimStart()).join('\n');
      if (!data || data === '[DONE]') continue;
      const payload = parse(data);
      if (payload?.jsonrpc === '2.0' && payload.id === 1) return payload;
    }
  }
  throw new Error(`${label} returned no matching JSON-RPC response`);
}

async function mcpCall(fetcher, endpoint, token, origin, method, params, name = '') {
  const response = await fetcher(endpoint, {
    method: 'POST', redirect: 'manual', cache: 'no-store',
    headers: {
      Accept: 'application/json, text/event-stream',
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      Origin: origin,
      'Mcp-Protocol-Version': protocol,
      'Mcp-Method': method,
      ...(name ? { 'Mcp-Name': name } : {}),
    },
    body: JSON.stringify({ jsonrpc: '2.0', id: 1, method, params: { ...params, _meta: meta } }),
  });
  mustBeDirect(response, `${method} ${name}`.trim(), origin);
  const payload = mcpPayload(await response.text(), `${method} ${name}`.trim());
  assert.equal(payload.error, undefined, `${method} ${name} returned a JSON-RPC error`);
  assert.ok(payload.result && typeof payload.result === 'object', `${method} ${name} returned no MCP result`);
  assert.notEqual(payload.result?.isError, true, `${method} ${name} returned a tool error`);
  return payload.result;
}

// This probes an existing deployment without mutating Studio or publishing a
// component. The token must represent a user allowed to list SDK namespaces
// and execute the specified published reader.
export async function verifyLiveReader(config, fetcher = globalThis.fetch) {
  const token = String(config.token || '').trim();
  assert.ok(token && !/\s/.test(token), 'one raw ID token is required');
  const origin = secureURL(config.uiOrigin, 'UI origin').origin;
  const sdk = secureURL(config.sdkURL, 'SDK URL');
  assert.ok(sdk.pathname.endsWith('/v1/studio/sdk/namespaces.list'), 'SDK URL must be the read-only namespaces.list route');
  const staticMCP = secureURL(config.staticMCPURL, 'static MCP URL');
  const reader = secureURL(config.readerURL, 'reader URL');
  const dynamicMCP = secureURL(config.dynamicMCPURL, 'dynamic MCP URL');
  const tool = String(config.readerTool || '').trim();
  assert.ok(tool, 'published reader tool name is required');
  const expectedText = String(config.expectedText || '').trim();
  assert.ok(expectedText, 'one non-sensitive expected reader value is required');
  assert.equal(sdk.search, '', 'SDK URL must not contain a query');
  assert.equal(staticMCP.search, '', 'static MCP URL must not contain a query');
  assert.equal(dynamicMCP.search, '', 'dynamic MCP URL must not contain a query');

  const sdkRequest = {
    method: 'POST', redirect: 'manual', cache: 'no-store',
    headers: { Accept: 'application/json', 'Content-Type': 'application/json', Authorization: `Bearer ${token}`, Origin: origin },
    body: '{}',
  };
  const deniedHeaders = { ...sdkRequest.headers };
  delete deniedHeaders.Authorization;
  const denied = await fetcher(sdk, { ...sdkRequest, headers: deniedHeaders });
  mustBeDirect(denied, 'native SDK without a bearer', origin, 401);
  const sdkResponse = await fetcher(sdk, sdkRequest);
  mustBeDirect(sdkResponse, 'native SDK', origin);
  const sdkBody = await jsonResponse(sdkResponse, 'native SDK');
  assert.notEqual(sdkBody?.status, 'error', 'native SDK returned an error body');

  const sdkTools = await mcpCall(fetcher, staticMCP, token, origin, 'tools/list', {});
  assert.ok((sdkTools?.tools || []).some((item) => item.name === 'studio.sdk.namespaces.list'), 'native SDK tool is absent');
  const sdkToolResult = await mcpCall(fetcher, staticMCP, token, origin, 'tools/call', { name: 'studio.sdk.namespaces.list', arguments: {} }, 'studio.sdk.namespaces.list');
  assert.ok(sdkToolResult.structuredContent && typeof sdkToolResult.structuredContent === 'object', 'native SDK MCP returned no structured content');
  const readerRequest = {
    method: 'GET', redirect: 'manual', cache: 'no-store',
    headers: { Accept: 'application/json', Authorization: `Bearer ${token}`, Origin: origin },
  };
  const deniedReaderHeaders = { ...readerRequest.headers };
  delete deniedReaderHeaders.Authorization;
  const deniedReader = await fetcher(reader, { ...readerRequest, headers: deniedReaderHeaders });
  mustBeDirect(deniedReader, 'published reader HTTP without a bearer', origin, 401);
  assert.ok(!(await deniedReader.text()).includes(expectedText), 'published reader exposed the expected value without a bearer');
  const readerResponse = await fetcher(reader, readerRequest);
  mustBeDirect(readerResponse, 'published reader HTTP', origin);
  const readerBody = await jsonResponse(readerResponse, 'published reader HTTP');
  assert.ok(readerBody && typeof readerBody === 'object', 'published reader returned no JSON object');
  assert.notEqual(readerBody.status, 'error', 'published reader HTTP returned an error body');
  assert.ok(JSON.stringify(readerBody).includes(expectedText), 'published reader HTTP did not return the expected value');

  const tools = await mcpCall(fetcher, dynamicMCP, token, origin, 'tools/list', {});
  assert.ok((tools?.tools || []).some((item) => item.name === tool), `published reader tool ${tool} is absent`);
  const result = await mcpCall(fetcher, dynamicMCP, token, origin, 'tools/call', { name: tool, arguments: config.readerArguments || {} }, tool);
  assert.ok(result?.structuredContent && typeof result.structuredContent === 'object', 'published reader MCP returned no structured content');
  assert.ok(JSON.stringify(result.structuredContent).includes(expectedText), 'published reader MCP did not return the expected value');
  return { sdk: 'ok', staticMCP: 'ok', readerHTTP: 'ok', readerMCP: 'ok', tool };
}

if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
  try {
    const token = readFileSync(0, 'utf8').trim();
    const result = await verifyLiveReader({
      token,
      uiOrigin: process.env.STUDIO_LIVE_UI_ORIGIN,
      sdkURL: process.env.STUDIO_LIVE_SDK_URL,
      staticMCPURL: process.env.STUDIO_LIVE_STATIC_MCP_URL,
      readerURL: process.env.STUDIO_LIVE_READER_URL,
      dynamicMCPURL: process.env.STUDIO_LIVE_DYNAMIC_MCP_URL,
      readerTool: process.env.STUDIO_LIVE_READER_TOOL,
      expectedText: process.env.STUDIO_LIVE_EXPECTED_TEXT,
      readerArguments: process.env.STUDIO_LIVE_READER_ARGS ? JSON.parse(process.env.STUDIO_LIVE_READER_ARGS) : {},
    });
    process.stdout.write(`${JSON.stringify(result)}\n`);
  } catch (error) {
    process.stderr.write(`Live reader check failed: ${error.message}\n`);
    process.exitCode = 1;
  }
}
