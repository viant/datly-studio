import assert from 'node:assert/strict';
import test from 'node:test';
import { StudioAPI } from './studioApi.js';

function response(payload, status = 200, requestId = '') {
  return { ok: status >= 200 && status < 300, status, headers: { get: (name) => name === 'X-Request-ID' ? requestId : '' }, json: async () => payload, text: async () => payload == null ? '' : typeof payload === 'string' ? payload : JSON.stringify(payload) };
}

test('development API carries an explicit local subject only', async () => {
  let request;
  const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } }, {
    fetcher: async (value) => { request = value; return response({ items: [], limit: 50, offset: 0 }); },
  });
  await api.listReports({ status: 'draft' });
  assert.equal(request.url, 'http://127.0.0.1:8080/v1/studio/sdk/reports.list');
  assert.equal(request.headers.get('X-Studio-Development-Subject'), 'dev-user');
  assert.equal(request.headers.get('Authorization'), null);
  assert.equal(await request.text(), '{"status":"draft"}');
});

test('authenticated API carries only the HttpOnly BFF session cookie', async () => {
  let request;
  const api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com' }, {
    fetcher: async (_, init) => { request = init; return response({ items: [] }); },
  });
  await api.listConnectors();
  assert.equal(request.credentials, 'include');
  assert.equal(request.headers.Authorization, undefined);
  assert.equal(request.headers['X-Studio-Development-Subject'], undefined);
});

test('authenticated API notifies the shell when its opaque session expires', async () => {
  let expired=false;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'},{onUnauthorized:()=>{expired=true;},fetcher:async()=>response({message:'expired'},401)});
  await assert.rejects(()=>api.listReports());
  assert.equal(expired,true);
});

test('MCP catalog uses the Datly tools/list protocol through the BFF path', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'},{fetcher:async(url,init)=>{request={url,init};return response({jsonrpc:'2.0',id:1,result:{tools:[{name:'vendor.read'}]}});}});
  const tools=await api.listMCPTools();
  assert.equal(request.url,'https://studio.example.com/v1/studio/mcp/mcp');
  assert.equal(request.init.headers['Mcp-Method'],'tools/list');
  assert.equal(request.init.headers['Mcp-Protocol-Version'],'2026-07-28');
  assert.equal(request.init.credentials,'include');
  assert.deepEqual(tools,[{name:'vendor.read'}]);
});

test('MCP skills catalog uses the dedicated skills/list protocol', async () => {
  let request;
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async(url,init)=>{request={url,init};return response({jsonrpc:'2.0',id:1,result:{skills:[{uri:'skill://guide/SKILL.md'}]}});}});
  const skills=await api.listMCPSkills();
  assert.equal(request.init.headers['Mcp-Method'],'skills/list');
  assert.match(request.init.body,/"method":"skills\/list"/);
  assert.deepEqual(skills,[{uri:'skill://guide/SKILL.md'}]);
});

test('MCP catalog reports an unavailable runtime instead of a JSON parser error', async () => {
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async()=>response(null,502)});
  await assert.rejects(()=>api.listMCPSkills(),/dedicated Datly MCP server is unavailable/);
});

test('MCP catalog reports invalid successful payloads explicitly', async () => {
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async()=>response('not-json',200)});
  await assert.rejects(()=>api.listMCPSkills(),/invalid response/);
});

test('SDK errors preserve the server request correlation id', async () => {
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async()=>response({message:'failed'},500,'abc123')});
  await assert.rejects(()=>api.listReports(),(error)=>error.requestId==='abc123'&&error.message.includes('request abc123'));
});

test('component identity updates stay behind report SDK operations', async () => {
  const calls=[];
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async(url,init)=>{calls.push({url,body:init.body});return response({id:'vendor',title:'Vendors',etag:3});}});
  await api.getReport('vendor');
  await api.updateReport('vendor',{title:'Vendors',etag:2});
  assert.deepEqual(calls,[
    {url:'http://127.0.0.1:8080/v1/studio/sdk/reports.get',body:'{"id":"vendor"}'},
    {url:'http://127.0.0.1:8080/v1/studio/sdk/reports.update',body:'{"id":"vendor","input":{"title":"Vendors","etag":2}}'},
  ]);
});

test('named connector SDK operation delegates to the stable transport operation', async () => {
  let request;
  const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } }, {
    fetcher: async (url, init) => { request = { url, init }; return response({ name: 'warehouse', status: 'passed' }); },
  });
  await api.testConnector('warehouse');
  assert.equal(request.url, 'http://127.0.0.1:8080/v1/studio/sdk/connectors.test');
  assert.equal(request.init.body, '{"name":"warehouse"}');
});

test('view SQL context reads connector driver through the named SDK operation', async()=>{
  let request;
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async(url,init)=>{request={url,init};return response({name:'warehouse',driver:'mysql'});}});
  const connector=await api.getConnector('warehouse');
  assert.equal(connector.driver,'mysql');
  assert.equal(request.url,'http://127.0.0.1:8080/v1/studio/sdk/connectors.get');
  assert.equal(request.init.body,'{"name":"warehouse"}');
});

test('connector update delegates the current optimistic revision through the SDK', async () => {
  let request;
  const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } }, {
    fetcher: async (url, init) => { request = { url, init }; return response({ name: 'warehouse', etag: 3 }); },
  });
  await api.updateConnector('warehouse', { description: 'rotated', etag: 2 });
  assert.equal(request.url, 'http://127.0.0.1:8080/v1/studio/sdk/connectors.update');
  assert.equal(request.init.body, '{"name":"warehouse","input":{"description":"rotated","etag":2}}');
});

test('connector deletion stays behind its optimistic SDK operation', async () => {
  let request;
  const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } }, {
    fetcher: async (url, init) => { request = { url, init }; return { ok: true, status: 204, json: async () => null }; },
  });
  await api.deleteConnector('warehouse', 4);
  assert.equal(request.url, 'http://127.0.0.1:8080/v1/studio/sdk/connectors.delete');
  assert.equal(request.init.body, '{"name":"warehouse","etag":4}');
});

test('schema browser uses named connector catalog operations', async () => {
  const calls=[];
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async(url,init)=>{calls.push({url,body:init.body});return response({items:[]});}});
  await api.listTables('warehouse',{schema:'analytics',query:'vendor'});
  await api.getTable('warehouse',{schema:'analytics',table:'vendors'});
  await api.testSQL('warehouse',{sql:'SELECT * FROM vendors',limit:10});
  assert.equal(calls[0].url,'http://127.0.0.1:8080/v1/studio/sdk/connectors.tables');
  assert.equal(calls[0].body,'{"name":"warehouse","input":{"schema":"analytics","query":"vendor"}}');
  assert.equal(calls[1].url,'http://127.0.0.1:8080/v1/studio/sdk/connectors.table');
  assert.equal(calls[2].url,'http://127.0.0.1:8080/v1/studio/sdk/connectors.test_sql');
});

test('named reader SDK operation delegates only author-facing fields', async () => {
  let request;
  const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } }, {
    fetcher: async (url, init) => { request = { url, init }; return response({ id: 'r123', slug: 'vendor-catalog' }); },
  });
  await api.createReport({ title: 'Vendor Catalog', slug: 'vendor-catalog', defaultConnectorName: 'reporting_mysql' });
  assert.equal(request.url, 'http://127.0.0.1:8080/v1/studio/sdk/reports.create');
  assert.equal(request.init.body, '{"title":"Vendor Catalog","slug":"vendor-catalog","defaultConnectorName":"reporting_mysql"}');
});

test('reader inspection and command methods retain version identity and revision', async () => {
  const calls = [];
  const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } }, {
    fetcher: async (url, init) => { calls.push({ url, body: init.body }); return response({}); },
  });
  await api.inspectVersion('vendor-catalog', 1);
  await api.validateVersion('vendor-catalog', 1, 2);
  await api.publishReader('vendor-catalog', 1, 2, 'release reader');
  await api.applyReaderCommand('vendor-catalog', 1, { expectedSourceRevision: 2, operation: { type: 'inspect' } });
  assert.equal(calls[0].url, 'http://127.0.0.1:8080/v1/studio/sdk/versions.inspect');
  assert.equal(calls[0].body, '{"reportId":"vendor-catalog","versionNo":1}');
  assert.equal(calls[1].url, 'http://127.0.0.1:8080/v1/studio/sdk/versions.validate');
  assert.equal(calls[1].body, '{"reportId":"vendor-catalog","versionNo":1,"expectedSourceRevision":2}');
  assert.equal(calls[2].url, 'http://127.0.0.1:8080/v1/studio/sdk/publications.publish');
  assert.equal(calls[2].body, '{"reportId":"vendor-catalog","versionNo":1,"input":{"expectedSourceRevision":2,"reason":"release reader"}}');
  assert.equal(calls[3].url, 'http://127.0.0.1:8080/v1/studio/sdk/versions.builder');
  assert.equal(calls[3].body, '{"reportId":"vendor-catalog","versionNo":1,"command":{"expectedSourceRevision":2,"operation":{"type":"inspect"}}}');
});

test('isolated view test delegates through the versioned Studio SDK', async () => {
  let request;
  const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } }, {
    fetcher: async (url, init) => { request = { url, body: init.body }; return response({ view: 'vendors' }); },
  });
  await api.testReaderView('vendor-catalog', 1, 'vendors', {}, 3);
  assert.equal(request.url, 'http://127.0.0.1:8080/v1/studio/sdk/versions.test_view');
  assert.equal(request.body, '{"reportId":"vendor-catalog","versionNo":1,"view":"vendors","input":{"input":{},"limit":3}}');
});

test('relation test delegates exact edge identity through the versioned SDK', async () => {
  let request;
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async(url,init)=>{request={url,body:init.body};return response({relation:'products'});}});
  await api.testReaderRelation('reader',2,'vendor->products',{tenant:7},25);
  assert.equal(request.url,'http://127.0.0.1:8080/v1/studio/sdk/versions.test_relation');
  assert.equal(request.body,'{"reportId":"reader","versionNo":2,"relation":"vendor->products","input":{"input":{"tenant":7},"limit":25}}');
});

test('cube composition playground delegates frames and SQL through the versioned SDK', async () => {
  let request;
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async(url,init)=>{request={url,body:init.body};return response({data:{data:[]}});}});
  await api.testCubeCompose('spend',1,{cubes:[{dimensions:{status:true},measures:{productCount:true}}],sql:'SELECT * FROM $CubeSQL1'});
  assert.equal(request.url,'http://127.0.0.1:8080/v1/studio/sdk/versions.test_compose');
  assert.match(request.body,/\$CubeSQL1/);
});

test('server-owned warmup keeps only immutable version identity in the browser', async () => {
  const requests=[];
  const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } }, {
    fetcher: async (url, init) => { requests.push({ url, body: init.body }); return response({ items: [], entries: 1 }); },
  });
  await api.warmupReader('vendor-spend', 1);
  await api.getWarmupRun('vendor-spend','w1');
  await api.listWarmupRuns('vendor-spend',1,{limit:10});
  assert.equal(requests[0].url, 'http://127.0.0.1:8080/v1/studio/sdk/versions.warmup');
  assert.equal(requests[0].body, '{"reportId":"vendor-spend","versionNo":1}');
  assert.equal(requests[1].url, 'http://127.0.0.1:8080/v1/studio/sdk/versions.warmup_get');
  assert.equal(requests[1].body, '{"reportId":"vendor-spend","runId":"w1"}');
  assert.equal(requests[2].url, 'http://127.0.0.1:8080/v1/studio/sdk/versions.warmup_list');
});

test('resources and skills use versioned Studio SDK operations', async () => {
  const calls=[];
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async(url,init)=>{calls.push({url,body:init.body});return response({files:[]});}});
  await api.getResources('reader',2);
  await api.upsertResourceFile({reportId:'reader',versionNo:2,namespace:'dev.docs',resourcePath:'guide/SKILL.md',content:'guide'});
  await api.deleteResourceFile('reader',2,'file');
  await api.upsertResourceFolder({reportId:'reader',versionNo:2,namespace:'dev.docs',rootPath:'guide',uriPrefix:'skill://dev-guide/'});
  await api.deleteResourceFolder('reader',2,'folder');
  await api.upsertSkillRoot({reportId:'reader',versionNo:2,folderId:'folder',skillRoot:'.'});
  await api.deleteSkillRoot('reader',2,'skill');
  assert.equal(calls[0].url,'http://127.0.0.1:8080/v1/studio/sdk/resources.get');
  assert.equal(calls[1].url,'http://127.0.0.1:8080/v1/studio/sdk/resources.upsert_file');
  assert.equal(calls[2].url,'http://127.0.0.1:8080/v1/studio/sdk/resources.delete_file');
  assert.equal(calls[3].url,'http://127.0.0.1:8080/v1/studio/sdk/resources.upsert_folder');
  assert.equal(calls[4].url,'http://127.0.0.1:8080/v1/studio/sdk/resources.delete_folder');
  assert.equal(calls[5].url,'http://127.0.0.1:8080/v1/studio/sdk/resources.upsert_skill');
  assert.equal(calls[6].url,'http://127.0.0.1:8080/v1/studio/sdk/resources.delete_skill');
});

test('governed namespaces use dedicated Studio SDK operations', async () => {
  const calls=[];
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'owner'}},{fetcher:async(url,init)=>{calls.push({url,body:init.body});return response({items:[]});}});
  await api.listNamespaces({status:'active'});
  await api.getNamespace('finance.ops');
  await api.createNamespace({name:'finance.ops',title:'Finance Operations'});
  await api.updateNamespace('finance.ops',{title:'Finance',etag:1});
  await api.deleteNamespace('finance.ops',2);
  assert.equal(calls[0].url,'http://127.0.0.1:8080/v1/studio/sdk/namespaces.list');
  assert.equal(calls[1].url,'http://127.0.0.1:8080/v1/studio/sdk/namespaces.get');
  assert.equal(calls[2].url,'http://127.0.0.1:8080/v1/studio/sdk/namespaces.create');
  assert.equal(calls[3].body,'{"name":"finance.ops","input":{"title":"Finance","etag":1}}');
  assert.equal(calls[4].url,'http://127.0.0.1:8080/v1/studio/sdk/namespaces.delete');
});

test('publication lifecycle stays behind Studio SDK operations', async () => {
  const calls=[];
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async(url,init)=>{calls.push({url,body:init.body});return response({status:'active'});}});
  await api.getPublication('reader');
  await api.unpublishReader('reader',3,'retire reader');
  await api.rollbackReader('reader',1,4,'restore reader');
  await api.getRuntimeStatus();
  assert.equal(calls[0].url,'http://127.0.0.1:8080/v1/studio/sdk/publications.get');
  assert.equal(calls[1].url,'http://127.0.0.1:8080/v1/studio/sdk/publications.unpublish');
  assert.equal(calls[1].body,'{"reportId":"reader","input":{"expectedActiveGeneration":3,"reason":"retire reader"}}');
  assert.equal(calls[2].url,'http://127.0.0.1:8080/v1/studio/sdk/publications.rollback');
  assert.equal(calls[2].body,'{"reportId":"reader","versionNo":1,"input":{"expectedSourceRevision":4,"reason":"restore reader"}}');
  assert.equal(calls[3].url,'http://127.0.0.1:8080/v1/studio/sdk/runtime.status');
});

test('publication event history stays behind the owner-scoped SDK operation', async () => {
  const calls=[];
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'owner'}},{fetcher:async(url,init)=>{calls.push({url,body:init.body});return response({items:[],limit:25,offset:0});}});
  await api.listPublicationEvents('reader',{limit:25,offset:0});
  assert.equal(calls[0].url,'http://127.0.0.1:8080/v1/studio/sdk/publications.events.list');
  assert.equal(calls[0].body,'{"reportId":"reader","input":{"limit":25,"offset":0}}');
});

test('report permissions use ACL SDK operations', async () => {
  const calls=[];
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'owner'}},{fetcher:async(url,init)=>{calls.push(url instanceof Request ? {url:url.url,body:await url.text(),subject:url.headers.get('X-Studio-Development-Subject')} : {url,body:init.body});return response({items:[]});}});
  await api.listACL('reader');
  await api.upsertACL({reportId:'reader',subjectType:'user',subjectId:'viewer',canView:true});
  await api.deleteACL('reader','user','viewer',2);
  assert.equal(calls[0].url,'http://127.0.0.1:8080/v1/studio/sdk/acl.list');
  assert.equal(calls[0].body,'{"reportId":"reader"}');
  assert.equal(calls[0].subject,'owner');
  assert.equal(calls[1].url,'http://127.0.0.1:8080/v1/studio/sdk/acl.upsert');
  assert.equal(calls[2].body,'{"reportId":"reader","subjectType":"user","subjectId":"viewer","etag":2}');
  assert.equal(calls[2].url,'http://127.0.0.1:8080/v1/studio/sdk/acl.delete');
});

test('generated ACL client uses the BFF session and preserves denial evidence', async () => {
  let sent;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(request)=>{
    sent=request;
    return response({code:'forbidden',message:'only the report owner can administer access'},403,'acl-request-1');
  }});
  await assert.rejects(()=>api.listACL('reader'),(error)=>error.status===403&&error.code==='forbidden'&&error.requestId==='acl-request-1');
  assert.equal(sent.url,'https://studio.example.com/v1/studio/sdk/acl.list');
  assert.equal(sent.credentials,'include');
  assert.equal(sent.headers.get('Authorization'),null);
  assert.equal(await sent.text(),'{"reportId":"reader"}');
});

test('authenticated preview executes the exact draft version through the SDK', async () => {
  let request;
  const api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com' }, {
    fetcher: async (url, init) => { request = { url, init }; return response({ data: { Summaries: [] } }); },
  });
  const result = await api.previewReader('vendor-spend', 1, {}, 50, '/v1/studio/readers/vendor-product-summary');
  assert.equal(request.url, 'https://studio.example.com/v1/studio/sdk/preview.execute');
  assert.equal(request.init.credentials, 'include');
  assert.equal(request.init.headers.Authorization, undefined);
  assert.equal(request.init.body, '{"reportId":"vendor-spend","versionNo":1,"input":{"input":{},"limit":50}}');
  assert.deepEqual(result.data, { Summaries: [] });
});

test('default browser fetch is invoked with its Window receiver', async () => {
  const original = globalThis.fetch;
  let receiver;
  globalThis.fetch = function () { receiver = this; return Promise.resolve(response({ items: [] })); };
  try {
    const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } });
    await api.listReports();
    assert.equal(receiver, globalThis);
  } finally { globalThis.fetch = original; }
});

test('SDK errors retain conflict metadata for authoring recovery', async () => {
  const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } }, { fetcher: async () => ({
    ok: false,
    status: 409,
    json: async () => ({ code: 'conflict', message: 'version source revision does not match', field: 'sourceRevision' }),
  }) });
  await assert.rejects(api.applyReaderCommand('reader', 1, { expectedSourceRevision: 2, operation: { type: 'inspect' } }), (error) => {
    assert.equal(error.code, 'conflict');
    assert.equal(error.status, 409);
    assert.equal(error.field, 'sourceRevision');
    return true;
  });
});
