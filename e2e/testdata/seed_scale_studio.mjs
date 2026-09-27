import assert from 'node:assert/strict';

const apiBase = process.env.STUDIO_SCALE_API_URL || 'http://127.0.0.1:8080';
const endpoint = new URL(apiBase);
assert.ok(['127.0.0.1', 'localhost', '::1'].includes(endpoint.hostname), 'scale seeding is loopback-only');
const subject = 'scale-review';
const connector = 'scale_mysql';
const dsn = process.env.STUDIO_SCALE_MYSQL_DSN || 'root:dev@tcp(127.0.0.1:43306)/reporting_e2e?parseTime=true';

async function sdk(operation, input) {
  const response = await fetch(`${apiBase}/v1/studio/sdk/${operation}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Studio-Development-Subject': subject },
    body: JSON.stringify(input),
  });
  const payload = await response.json();
  assert.ok(response.ok, `${operation} HTTP ${response.status}: ${JSON.stringify(payload)}`);
  return payload;
}

async function apply(reportId, revision, operation) {
  const result = await sdk('versions.builder', {
    reportId, versionNo: 1,
    command: { expectedSourceRevision: revision, operation },
  });
  assert.ok(result.applied, `${operation.type}: ${JSON.stringify(result.inspection?.diagnostics)}`);
  return result.inspection.version.sourceRevision;
}

async function createReader({ slug, title, view, sql, tool }) {
  const report = await sdk('components.create', { slug, title, namespace: 'scale', defaultConnectorName: connector });
  const version = await sdk('versions.create', { reportId: report.id, input: { authoringMode: 'dql', notes: 'Docker scale reader fixture' } });
  let revision = await apply(report.id, version.sourceRevision, {
    type: 'createReader', reader: { route: `/v1/studio/readers/${slug}`, name: view, sql },
  });
  return { report, revision, tool };
}

async function exposeAndPublish(fixture) {
  fixture.revision = await apply(fixture.report.id, fixture.revision, {
    type: 'setSetting', setting: { name: 'mcp', args: [JSON.stringify(fixture.tool), JSON.stringify(`Read ${fixture.report.title}`)] },
  });
  const validation = await sdk('versions.validate', { reportId: fixture.report.id, versionNo: 1, expectedSourceRevision: fixture.revision });
  assert.equal(validation.valid, true, `${fixture.report.title}: ${JSON.stringify(validation.diagnostics)}`);
  const publication = await sdk('publications.publish', {
    reportId: fixture.report.id, versionNo: 1,
    input: { expectedSourceRevision: fixture.revision, reason: 'Docker scale MCP verification' },
  });
  return { id: fixture.report.id, title: fixture.report.title, generation: publication.activeGeneration };
}

await sdk('connectors.create', { name: connector, driver: 'mysql', dsnTemplate: dsn, description: 'Docker scale fixture' });
const probe = await sdk('connectors.test', { name: connector });
assert.equal(probe.status, 'passed', JSON.stringify(probe));
const configured = await sdk('connectors.get', { name: connector });
await sdk('connectors.activate', { name: connector, etag: configured.etag });
await sdk('namespaces.create', { name: 'scale', title: 'Scale Review' });

const hierarchy = await createReader({
  slug: 'hierarchy-20', title: 'Hierarchy 20', view: 'view_01',
  sql: 'SELECT * FROM STUDIO_VIEW_01', tool: 'scalereview.hierarchy20.read',
});
const parents = { 2: 1, 3: 1, 4: 1, 5: 1, 6: 2, 7: 2, 8: 3, 9: 4, 10: 5, 11: 6, 12: 7, 13: 8, 14: 9, 15: 10, 16: 11, 17: 12, 18: 13, 19: 14, 20: 15 };
for (let index = 2; index <= 20; index += 1) {
  const name = `view_${String(index).padStart(2, '0')}`;
  const parent = `view_${String(parents[index]).padStart(2, '0')}`;
  hierarchy.revision = await apply(hierarchy.report.id, hierarchy.revision, {
    type: 'addView', view: {
      name, kind: 'subview', parent, join: 'LEFT JOIN',
      sql: `SELECT * FROM STUDIO_VIEW_${String(index).padStart(2, '0')}`,
      on: `${name}.PARENT_ID=${parent}.ID`,
    },
  });
}
hierarchy.revision = await apply(hierarchy.report.id, hierarchy.revision, {
  type: 'addField', field: { name: 'RootID', type: 'int', sourceKind: 'query', sourceName: 'rootId', required: false },
});
hierarchy.revision = await apply(hierarchy.report.id, hierarchy.revision, {
  type: 'addFieldPredicate', predicate: {
    field: 'RootID', view: 'view_01', group: 1, name: 'equal', args: ['STUDIO_VIEW_01', 'ID'],
  },
});
const hierarchyResult = await exposeAndPublish(hierarchy);

const wide = await createReader({
  slug: 'wide-60', title: 'Wide 60 Fields', view: 'wide_60',
  sql: 'SELECT * FROM STUDIO_WIDE_60', tool: 'scalereview.wide60.read',
});
wide.revision = await apply(wide.report.id, wide.revision, {
  type: 'addField', field: { name: 'RowID', type: 'int', sourceKind: 'query', sourceName: 'rowId', required: false },
});
wide.revision = await apply(wide.report.id, wide.revision, {
  type: 'addFieldPredicate', predicate: {
    field: 'RowID', view: 'wide_60', group: 1, name: 'equal', args: ['STUDIO_WIDE_60', 'ID'],
  },
});
const wideResult = await exposeAndPublish(wide);

const cube = await createReader({
  slug: 'cube-probe', title: 'Cube Probe', view: 'spend',
  sql: 'SELECT ID, COUNT(*) AS TOTAL FROM STUDIO_WIDE_60 GROUP BY ID',
  tool: 'scalereview.cubeprobe.read',
});
cube.revision = await apply(cube.report.id, cube.revision, {
  type: 'setColumnContract', column: { view: 'spend', column: 'ID', tags: { groupable: 'true' } },
});
cube.revision = await apply(cube.report.id, cube.revision, {
  type: 'addFunction', function: { name: 'groupable', args: ['spend'] },
});
for (const setting of [
  { name: 'cube', args: [] },
  { name: 'cubeCompose', args: ['true', 'true', '4', '50', '12000'] },
]) {
  cube.revision = await apply(cube.report.id, cube.revision, { type: 'setSetting', setting });
}
const cubeResult = await exposeAndPublish(cube);
console.log(JSON.stringify({ hierarchy: hierarchyResult, wide: wideResult, cube: cubeResult }));
