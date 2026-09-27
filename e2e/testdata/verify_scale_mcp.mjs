import assert from 'node:assert/strict';

const endpoint = process.env.STUDIO_SCALE_MCP_URL || 'http://127.0.0.1:8091/mcp';
const cookie = process.env.STUDIO_SCALE_MCP_COOKIE || '';
const protocol = '2026-07-28';
const meta = {
  'io.modelcontextprotocol/protocolVersion': protocol,
  'io.modelcontextprotocol/clientCapabilities': {},
};

async function request(method, params = {}, name = '') {
  const response = await fetch(endpoint, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      'Mcp-Protocol-Version': protocol,
      'Mcp-Method': method,
      ...(name ? { 'Mcp-Name': name } : {}),
      ...(cookie ? { Cookie: cookie } : {}),
    },
    body: JSON.stringify({ jsonrpc: '2.0', id: 1, method, params: { ...params, _meta: meta } }),
  });
  const payload = await response.json();
  assert.equal(response.status, 200, `${method} HTTP ${response.status}: ${JSON.stringify(payload)}`);
  assert.equal(payload.error, undefined, `${method}: ${JSON.stringify(payload.error)}`);
  assert.notEqual(payload.result?.isError, true, `${method}: ${JSON.stringify(payload.result)}`);
  return payload.result;
}

const base = 'scalereview.cubeprobe.read';
const names = [base, `${base}Cube`, `${base}CubeCompose`, 'scalereview.hierarchy20.read', 'scalereview.wide60.read'];
const catalog = await request('tools/list');
const available = new Set((catalog.tools || []).map((tool) => tool.name));
for (const name of names) assert.ok(available.has(name), `missing live MCP tool ${name}`);
for (const [name, parameter] of [
  ['scalereview.hierarchy20.read', 'RootID'],
  ['scalereview.wide60.read', 'RowID'],
]) {
  const schema = catalog.tools.find((tool) => tool.name === name)?.inputSchema;
  assert.equal(schema?.properties?.[parameter]?.type, 'integer', `${name} must publish its integer filter`);
}

async function call(name, args) {
  const result = await request('tools/call', { name, arguments: args }, name);
  assert.ok(result.structuredContent, `${name} returned no structured content`);
  return result.structuredContent;
}

const hierarchy = await call('scalereview.hierarchy20.read', {});
const hierarchyRows = Object.values(hierarchy).find(Array.isArray);
assert.equal(hierarchyRows?.length, 2, 'hierarchy reader should return two roots');
assert.ok(JSON.stringify(hierarchy).includes('view_20'), 'deepest hierarchy view is missing');
const filteredHierarchy = await call('scalereview.hierarchy20.read', { RootID: 101 });
const filteredHierarchyRows = Object.values(filteredHierarchy).find(Array.isArray);
assert.equal(filteredHierarchyRows?.length, 1, 'rootId should select one hierarchy root');
assert.equal(filteredHierarchyRows[0].id, 101, 'rootId selected the wrong hierarchy root');
assert.ok(JSON.stringify(filteredHierarchy).includes('View 20 row 1'), 'filtered hierarchy lost its deepest descendant');
assert.ok(!JSON.stringify(filteredHierarchy).includes('View 20 row 2'), 'filtered hierarchy leaked the other root');

const wide = await call('scalereview.wide60.read', {});
const wideRows = Object.values(wide).find(Array.isArray);
assert.equal(wideRows?.length, 2, 'wide reader should return two rows');
assert.equal(Object.keys(wideRows[0]).length, 60, 'wide reader should return exactly 60 columns');
const filteredWide = await call('scalereview.wide60.read', { RowID: 1 });
const filteredWideRows = Object.values(filteredWide).find(Array.isArray);
assert.equal(filteredWideRows?.length, 1, 'rowId should select one 60-column row');
assert.equal(Object.keys(filteredWideRows[0]).length, 60, 'filtered wide row lost columns');
assert.equal(filteredWideRows[0].id, 1, 'rowId selected the wrong wide row');

const cube = await call(`${base}Cube`, { dimensions: { iD: true }, limit: 10 });
assert.equal(Object.values(cube).find(Array.isArray)?.length, 2, 'Cube should return two grouped rows');

const compose = await call(`${base}CubeCompose`, {
  cubes: [{ filters: {} }],
  sql: 'SELECT t1.ID FROM $CubeSQL1 AS t1 LIMIT 10',
});
assert.equal(compose.data?.length, 2, 'Cube Compose should return two composed rows');

console.log(JSON.stringify({ tools: names, hierarchyRoots: hierarchyRows.length, filteredHierarchyRoots: filteredHierarchyRows.length, wideColumns: Object.keys(wideRows[0]).length, filteredWideRows: filteredWideRows.length, cubeRows: Object.values(cube).find(Array.isArray).length, composedRows: compose.data.length }));
