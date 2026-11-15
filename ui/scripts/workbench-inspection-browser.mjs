import assert from 'node:assert/strict';
import {mkdtemp, rm, writeFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {chromium} from 'playwright';

// Run only against a fresh instance of e2e/cmd/fixture. This uses its synthetic
// owner identity, real compiled window resources, and native SDK/MCP operations.
const origin = process.env.STUDIO_WORKBENCH_FIXTURE_URL;
assert.ok(origin, 'STUDIO_WORKBENCH_FIXTURE_URL must identify an owned E2E fixture');
const cacheRoot = await mkdtemp(path.join(tmpdir(), 'studio-workbench-binding-'));
const browser = await chromium.launch({headless:true, channel:process.env.PLAYWRIGHT_CHANNEL || 'chrome'});
try {
  const context = await browser.newContext({viewport:{width:1440, height:1050}});
  const health = await (await context.request.get(`${origin}/health`)).json();
  assert.equal(health.status, 'ready');
  assert.ok(health.predicatePackage, 'Expected the owned Studio E2E fixture');
  let namespaceId;
  const post = async (operation, data, scoped = true) => {
    const response = await context.request.post(`${origin}/v1/studio/sdk/${operation}`, {
      headers:scoped && namespaceId ? {'X-Studio-Namespace':namespaceId} : {}, data,
    });
    const body = await response.json();
    assert.ok(response.ok(), `${operation}: ${response.status()} ${JSON.stringify(body)}`);
    return body;
  };
  const suffix = Date.now().toString(36);
  const namespace = await post('namespaces.create', {name:`binding.${suffix}`, title:'Workbench binding proof'}, false);
  namespaceId = namespace.namespaceId;
  assert.ok(namespaceId);
  const connector = await post('connectors.create', {name:`binding_${suffix}`, driver:'sqlite', dsnTemplate:':memory:'}, false);
  await post('connectors.test', {name:connector.name}, false);
  const tested = await post('connectors.get', {name:connector.name}, false);
  await post('connectors.activate', {name:connector.name, etag:tested.etag}, false);
  const component = await post('components.create', {slug:`binding-${suffix}`, title:'Workbench binding proof', defaultConnectorName:connector.name});
  const versions = [];
  for (const number of [1, 2]) {
    const dql = `#package('${component.componentScope}')
#setting($_ = $connector('${connector.name}'))
#setting($_ = $route('/binding-${suffix}/${number}','GET'))
#setting($_ = $cache('records','1m').WithLocation('${cacheRoot}/${number}'))
#setting($_ = $cache_warmup('ID','Limit=5','MaxCases=2'))
#define($_ = $ID<int>(query/id).Optional())
#define($_ = $Rows<[]*Record>(output/view))
SELECT records.*, type(records,'Record') FROM (SELECT ${number} AS ID) records`;
    versions.push(await post('versions.create', {reportId:component.id, input:{authoringMode:'dql', authoredDql:dql}}));
  }
  const stored = () => Promise.all(versions.map(version => post('versions.get', {reportId:component.id, versionNo:version.versionNo})));
  const before = await stored();
  const page = await context.newPage();
  const operations = [], pending = new Map(), timings = [], errors = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('request', request => {
    if (request.method() !== 'POST') return;
    let body;
    try { body = request.postDataJSON(); } catch { return; }
    const tool = body?.method === 'tools/call' ? body.params?.name : '';
    if (!tool?.startsWith('studio.sdk.')) return;
    const operation = tool.slice('studio.sdk.'.length);
    operations.push({operation, input:body.params.arguments});
    if (operation === 'versions.inspect') pending.set(request, {input:body.params.arguments, started:performance.now()});
  });
  page.on('requestfinished', request => {
    const item = pending.get(request);
    if (item) {timings.push({input:item.input, milliseconds:Math.round(performance.now()-item.started)});pending.delete(request);}
  });
  const inspections = () => operations.filter(item => item.operation === 'versions.inspect');
  const graph = {discoverColumns:false, includePreviewInput:false, includeEffectiveWarmup:false};
  const expectInspection = (number, options) => {
    assert.deepEqual(inspections().at(-1)?.input, {reportId:component.id, versionNo:number, ...options, namespaceId});
  };
  const graphReady = () => page.getByRole('heading', {name:'Component graph', exact:true}).waitFor();
  await page.goto(`${origin}/?section=components`);
  const namespaces = page.getByRole('combobox', {name:'Current namespace', exact:true});
  await namespaces.waitFor();
  await namespaces.selectOption(namespaceId);
  const open = page.getByRole('button', {name:'Open Reader Builder', exact:true});
  await open.waitFor();
  const navigation = [];
  for (const action of ['open', 'reopen']) {
    const started = performance.now();
    await open.click(); await graphReady();
    navigation.push({action, milliseconds:Math.round(performance.now()-started)});
    expectInspection(2, graph);
    assert.equal(await page.getByRole('combobox', {name:'Component version'}).inputValue(), '2');
    if (action === 'open') await page.getByRole('main', {name:'Component editor', exact:true}).getByRole('button', {name:'Components', exact:true}).click();
  }
  assert.equal(inspections().length, 2);
  await page.getByRole('combobox', {name:'Component version'}).selectOption('1');
  await page.waitForFunction(() => document.querySelector('[aria-label="Component version"]')?.value === '1');
  await graphReady();
  expectInspection(1, graph);
  await page.getByRole('button', {name:'Run preview', exact:true}).click();
  await page.getByRole('button', {name:'Run with parameters', exact:true}).waitFor();
  expectInspection(1, {...graph, includePreviewInput:true});
  await page.getByRole('button', {name:'Close preview parameters'}).click();
  await page.getByRole('button', {name:'More actions'}).click();
  await page.getByRole('menuitem', {name:'Cache & warmup', exact:true}).click();
  await page.getByText('No effective warmup plans for this saved version.', {exact:true}).waitFor({state:'hidden'});
  await page.getByRole('dialog', {name:'Cache & warmup', exact:true}).getByText('records', {exact:false}).first().waitFor();
  expectInspection(1, {...graph, includeEffectiveWarmup:true});
  assert.equal(inspections().length, 5);
  assert.deepEqual(operations.filter(item => ['preview.execute','versions.warmup','versions.builder','versions.apply'].includes(item.operation)), []);
  const after = await stored();
  for (let index=0; index<before.length; index++) {
    for (const key of ['authoredDql','generatedDql','sourceRevision']) assert.equal(after[index][key], before[index][key], `Stored ${key} changed`);
  }
  assert.deepEqual(errors, []);
  const result = {fixture:'owned SQLite E2E fixture', navigation, inspections:timings};
  if (process.env.STUDIO_WORKBENCH_TIMINGS) await writeFile(process.env.STUDIO_WORKBENCH_TIMINGS, JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result, null, 2));
  console.log('PASS authored workbench: open/reopen, version switch, deferred preview/warmup, unchanged source, no execution');
} finally {
  await browser.close();
  await rm(cacheRoot, {recursive:true, force:true});
}
