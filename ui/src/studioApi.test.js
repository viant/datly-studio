import assert from 'node:assert/strict';
import test from 'node:test';
import { StudioAPI } from './studioApi.js';

test('identity-token mode sends bearer ID tokens directly to Datly and never sends cookies', async () => {
  const seen = [];
  let token = 'id-1';
  const identity = { token: async (force) => { if (force) token = 'id-2'; return token; } };
  const config = { mode: 'authenticated', apiBaseURL: 'https://static.example.com', mcpBaseURL: 'https://dynamic.example.com',
    authentication: { mode: 'identity-token' } };
  const api = new StudioAPI(config, { identity, fetcher: async (request) => {
    seen.push({ url: request.url, credentials: request.credentials, bearer: request.headers.get('Authorization') });
    if (seen.length === 1) return response({ message: 'expired' }, 401);
    if (seen.length === 2) return response({ items: [] });
    return response({ jsonrpc: '2.0', id: 1, result: { tools: [] } });
  } });
  assert.deepEqual(await api.listComponents(), { items: [] });
  assert.deepEqual(await api.listMCPTools(), []);
  assert.deepEqual(seen, [
    { url: 'https://static.example.com/v1/studio/sdk/components.list', credentials: 'omit', bearer: 'Bearer id-1' },
    { url: 'https://static.example.com/v1/studio/sdk/components.list', credentials: 'omit', bearer: 'Bearer id-2' },
    { url: 'https://dynamic.example.com/mcp', credentials: 'omit', bearer: 'Bearer id-2' },
  ]);
});

function response(payload, status = 200, requestId = '') {
  return { ok: status >= 200 && status < 300, status, headers: { get: (name) => name === 'X-Request-ID' ? requestId : '' }, json: async () => payload, text: async () => payload == null ? '' : typeof payload === 'string' ? payload : JSON.stringify(payload) };
}

test('selected namespace MCP discovery uses its runtime endpoint', async () => {
  const id = 'a'.repeat(64);
  let seen;
  const api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com', authentication: { mode: 'identity-token' } }, {
    identity: { token: async () => 'test-token' },
    fetcher: async (request) => { seen = request; return response({ jsonrpc: '2.0', id: 1, result: { tools: [] } }); },
  });
  api.setNamespace(id);
  api.getRuntimeStatus = async () => ({ host: { namespaceId: id, mcpUrl: 'http://127.0.0.1:8591/mcp' } });
  assert.deepEqual(await api.listMCPTools(), []);
  assert.equal(seen.url, 'http://127.0.0.1:8591/mcp');
  assert.equal(seen.headers.get('X-Studio-Namespace'), id);
  assert.equal(seen.headers.get('Authorization'), 'Bearer test-token');
  assert.equal(seen.credentials, 'omit');
});

test('selected namespace MCP discovery cannot fall back to another namespace', async () => {
  let calls = 0;
  const api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com', authentication: { mode: 'identity-token' } }, {
    identity: { token: async () => 'test-token' },
    fetcher: async () => { calls++; return response({}); },
  });
  api.setNamespace('a'.repeat(64));
  api.getRuntimeStatus = async () => ({ host: { namespaceId: 'b'.repeat(64), mcpUrl: 'http://127.0.0.1:8592/mcp' } });
  await assert.rejects(api.listMCPSkills(), /no ready MCP endpoint/);
  assert.equal(calls, 0);
});

test('selected namespace MCP never sends a session cookie to its direct endpoint', async () => {
  let seen;
  const api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com' }, {
    fetcher: async (url, init) => { seen = { url, init }; return response({ jsonrpc: '2.0', id: 1, result: { skills: [] } }); },
  });
  api.setNamespace('a'.repeat(64));
  assert.deepEqual(await api.listMCPSkills(), []);
  assert.equal(seen.url, 'https://studio.example.com/v1/studio/mcp/mcp');
  assert.equal(seen.init.headers['X-Studio-Namespace'], 'a'.repeat(64));
  assert.equal(seen.init.credentials, 'include');
});

test('unavailable namespace blocks resources while retaining global discovery', async () => {
  const api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com' }, { fetcher: async (value, init) => {
    assert.equal(new Request(value, init).headers.get('X-Studio-Namespace'), null);
    return response({ items: [] });
  } });
  api.setNamespace('a'.repeat(64));
  api.setNamespaceBlocked(true);
  await assert.rejects(api.listComponents(), { code: 'namespace_unavailable' });
  await assert.rejects(api.listMCPTools(), { code: 'namespace_unavailable' });
  assert.deepEqual(await api.listNamespaces(), { items: [] });
  assert.deepEqual(await api.listConnectors(), { items: [] });
  assert.deepEqual(await api.listAuthorizationPredicateTypes(), []);
});

test('MCP catalog discards a response after switching namespaces', async () => {
  let resolve;
  const api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com' }, {
    fetcher: () => new Promise((done) => { resolve = done; }),
  });
  const pending = api.listMCPTools();
  const rejected = assert.rejects(pending, { code: 'namespace_changed' });
  api.setNamespace('b'.repeat(64));
  resolve(response({ jsonrpc: '2.0', id: 1, result: { tools: [{ name: 'old-tool' }] } }));
  await rejected;
});

test('development API carries an explicit local subject only', async () => {
  let request;
  const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } }, {
    fetcher: async (value) => { request = value; return response({ items: [], limit: 50, offset: 0 }); },
  });
  const page=await api.listComponents({ status: 'draft' });
  assert.deepEqual(page,{items:[],limit:50,offset:0});
  assert.equal(request.url, 'http://127.0.0.1:8080/v1/studio/sdk/components.list');
  assert.equal(request.headers.get('X-Studio-Development-Subject'), 'dev-user');
  assert.equal(request.headers.get('Authorization'), null);
  assert.equal(await request.text(), '{"status":"draft"}');
});

test('authenticated API carries only the HttpOnly BFF session cookie', async () => {
  let request;
  const api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com' }, {
    fetcher: async (value) => { request = value; return response({ items: [], limit: 50, offset: 0 }); },
  });
  const page=await api.listConnectors();
  assert.deepEqual(page,{items:[],limit:50,offset:0});
  assert.equal(request.credentials, 'include');
  assert.equal(request.headers.get('Authorization'), null);
  assert.equal(request.headers.get('X-Studio-Development-Subject'), null);
  assert.equal(await request.text(), '{}');
});

test('authenticated API notifies the shell when its opaque session expires', async () => {
  let expired=false;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'},{onUnauthorized:()=>{expired=true;},fetcher:async()=>response({message:'expired'},401)});
  await assert.rejects(()=>api.listComponents());
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
  await assert.rejects(()=>api.listComponents(),(error)=>error.requestId==='abc123'&&error.message.includes('request abc123'));
});

test('component identity updates stay behind report SDK operations', async () => {
  const calls=[];
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async(url,init)=>{calls.push(url instanceof Request ? {url:url.url,body:await url.text()} : {url,body:init.body});return response({id:'vendor',title:'Vendors',etag:3});}});
  const report=await api.getComponent('vendor');
  assert.equal(report.id,'vendor');
  await api.updateComponent('vendor',{title:'Vendors',etag:2});
  assert.deepEqual(calls,[
    {url:'http://127.0.0.1:8080/v1/studio/sdk/components.get',body:'{"id":"vendor"}'},
    {url:'http://127.0.0.1:8080/v1/studio/sdk/components.update',body:'{"id":"vendor","input":{"title":"Vendors","etag":2}}'},
  ]);
});

test('generated report get preserves not-found status and request id', async () => {
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async()=>response({message:'report not found'},404,'get-request-1')});
  await assert.rejects(()=>api.getComponent('missing'),(error)=>error.status===404&&error.code==='not_found'&&error.requestId==='get-request-1'&&error.message.includes('report not found'));
});

test('generated version get uses the native route and preserves denial evidence', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{request=value;return response({message:'report version not found'},404,'version-request-1');}});
  await assert.rejects(()=>api.getVersion('reader',2),(error)=>error.status===404&&error.code==='not_found'&&error.requestId==='version-request-1');
  assert.equal(request.url,'https://studio.example.com/v1/studio/sdk/versions.get');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(),'{"reportId":"reader","versionNo":2}');
});

test('generated version create uses the native authenticated route and exact reader identity', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{request=value;return response({reportId:'reader',versionNo:2,state:'draft',sourceRevision:1});}});
  const version=await api.createVersion('reader',{authoringMode:'dql',authoredDql:'SELECT 1'});
  assert.equal(version.versionNo,2);
  assert.equal(request.url,'https://studio.example.com/v1/studio/sdk/versions.create');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(),'{"reportId":"reader","input":{"authoringMode":"dql","authoredDql":"SELECT 1"}}');
});

test('generated version edit carries the optimistic source revision and payload', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{request=value;return response({version:{reportId:'reader',versionNo:2,sourceRevision:4}});}});
  const edited=await api.applyVersionEdit('reader',2,{kind:'set_dql',expectedSourceRevision:3,payload:{authoredDql:'SELECT 1'}});
  assert.equal(edited.version.sourceRevision,4);
  assert.equal(request.url,'https://studio.example.com/v1/studio/sdk/versions.apply');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(),'{"reportId":"reader","versionNo":2,"command":{"kind":"set_dql","expectedSourceRevision":3,"payload":{"authoredDql":"SELECT 1"}}}');
});

test('predicate type catalog uses its native authenticated SDK route', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{request=value;return response({items:[{alias:'studioauthorization',packagePath:'github.com/viant/datly-studio/studio/authorization',typeName:'ReportRead'}]});}});
  const items=await api.listAuthorizationPredicateTypes();
  assert.equal(items[0].typeName,'ReportRead');
  assert.equal(request.url,'https://studio.example.com/v1/studio/sdk/authorization_predicates.types');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(),'{}');
});

test('governed predicate get and list use native authenticated SDK routes', async () => {
  const requests=[];
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{
    requests.push({url:value.url,body:await value.text(),credentials:value.credentials});
    return response(value.url.endsWith('.get')?{name:'studio.alpha.read',linked:true}:{items:[{name:'studio.alpha.read'}],limit:1,offset:0});
  }});
  assert.equal((await api.getAuthorizationPredicate('studio.alpha.read')).linked,true);
  assert.equal((await api.listAuthorizationPredicates({query:'Alpha',limit:1})).items.length,1);
  assert.deepEqual(requests,[
    {url:'https://studio.example.com/v1/studio/sdk/authorization_predicates.get',body:'{"name":"studio.alpha.read"}',credentials:'include'},
    {url:'https://studio.example.com/v1/studio/sdk/authorization_predicates.list',body:'{"query":"Alpha","limit":1}',credentials:'include'},
  ]);
});

test('governed predicate create uses the native publisher route', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{request=value;return response({name:'studio.gamma.read',ownerId:'alice',linked:true,etag:1});}});
  const created=await api.createAuthorizationPredicate({name:'studio.gamma.read',title:'Gamma',packagePath:'github.com/viant/datly-studio/studio/authorization',typeName:'ReportEdit'});
  assert.equal(created.linked,true);
  assert.equal(request.url,'https://studio.example.com/v1/studio/sdk/authorization_predicates.create');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(),'{"name":"studio.gamma.read","title":"Gamma","packagePath":"github.com/viant/datly-studio/studio/authorization","typeName":"ReportEdit"}');
});

test('governed predicate update and delete preserve native ETag and 204 contracts', async () => {
  const requests=[];
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{
    requests.push({url:value.url,body:await value.text(),credentials:value.credentials});
    return value.url.endsWith('.delete')?response(null,204):response({name:'studio.gamma.read',title:'Revised',etag:2});
  }});
  const updated=await api.updateAuthorizationPredicate('studio.gamma.read',{title:'Revised',etag:1});
  assert.equal(updated.etag,2);
  await api.deleteAuthorizationPredicate('studio.gamma.read',2);
  assert.deepEqual(requests,[
    {url:'https://studio.example.com/v1/studio/sdk/authorization_predicates.update',body:'{"name":"studio.gamma.read","input":{"title":"Revised","etag":1}}',credentials:'include'},
    {url:'https://studio.example.com/v1/studio/sdk/authorization_predicates.delete',body:'{"name":"studio.gamma.read","etag":2}',credentials:'include'},
  ]);
});

test('generated DQL import uses the native route with its exact source payload', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{request=value;return response({version:{reportId:'reader',versionNo:2},entryDql:'main.dql',entries:['main.dql'],files:['main.dql']});}});
  const imported=await api.loadDQL('reader',{dql:'SELECT 1',notes:'first import'});
  assert.equal(imported.version.versionNo,2);
  assert.equal(request.url,'https://studio.example.com/v1/studio/sdk/versions.load_dql');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(),'{"reportId":"reader","input":{"dql":"SELECT 1","notes":"first import"}}');
});

test('generated archive import preserves base64 bytes and selected root DQL', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{request=value;return response({version:{reportId:'reader',versionNo:3},entryDql:'main.dql',entries:['main.dql','other.dql'],files:['main.dql','other.dql']});}});
  const imported=await api.loadArchive('reader',{archive:'eA==',format:'zip',entryDql:'main.dql'});
  assert.equal(imported.version.versionNo,3);
  assert.equal(request.url,'https://studio.example.com/v1/studio/sdk/versions.load_archive');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(),'{"reportId":"reader","input":{"archive":"eA==","format":"zip","entryDql":"main.dql"}}');
});

test('generated version list sends nested filters through the native route', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{request=value;return response({items:[],limit:25,offset:2});}});
  const page=await api.listVersions('reader',{state:'draft',limit:25,offset:2});
  assert.deepEqual(page,{items:[],limit:25,offset:2});
  assert.equal(request.url,'https://studio.example.com/v1/studio/sdk/versions.list');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(),'{"reportId":"reader","input":{"state":"draft","limit":25,"offset":2}}');
});

test('generated DQL export uses the native route and preserves authorization denial', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{request=value;return response({message:'report version not found'},404,'export-request-1');}});
  await assert.rejects(()=>api.exportDQL('reader',2),(error)=>error.status===404&&error.code==='not_found'&&error.requestId==='export-request-1');
  assert.equal(request.url,'https://studio.example.com/v1/studio/sdk/versions.export_dql');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(),'{"reportId":"reader","versionNo":2}');
});

test('generated version descriptor uses the native route for view-capable subjects', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{request=value;return response({component:{route:'/read'},types:{}});}});
  const descriptor=await api.getVersionDescriptor('reader',2);
  assert.deepEqual(descriptor,{component:{route:'/read'},types:{}});
  assert.equal(request.url,'https://studio.example.com/v1/studio/sdk/versions.descriptor');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(),'{"reportId":"reader","versionNo":2}');
});

test('generated connector create sends connection material only to the authenticated BFF route', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{request=value;return response({name:'warehouse',driver:'mysql',dsnConfigured:true,secretConfigured:true,ownerId:'owner',status:'draft',etag:1});}});
  const value=await api.createConnector({name:'warehouse',driver:'mysql',dsnTemplate:'private-dsn',secretRef:'private-ref'});
  assert.equal(value.name,'warehouse');
  assert.equal(request.url,'https://studio.example.com/v1/studio/sdk/connectors.create');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(),'{"name":"warehouse","driver":"mysql","dsnTemplate":"private-dsn","secretRef":"private-ref"}');
});

test('component download uses the native route without exposing a bearer to the browser', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{request=value;return response({filename:'component-v2.zip',mediaType:'application/zip',archive:'eA==',entryDql:'component.dql',files:['component.dql']});}});
  const archive=await api.downloadComponent('reader',2);
  assert.equal(archive.filename,'component-v2.zip');
  assert.equal(request.url,'https://studio.example.com/v1/studio/sdk/versions.download');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(),'{"reportId":"reader","versionNo":2}');
});

test('resource snapshot uses the native route and preserves the exact version identity', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{request=value;return response({files:[],folders:[],skills:[],version:{reportId:'reader',versionNo:2}});}});
  const snapshot=await api.getResources('reader',2);
  assert.equal(snapshot.version.versionNo,2);
  assert.equal(request.url,'https://studio.example.com/v1/studio/sdk/resources.get');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(),'{"reportId":"reader","versionNo":2}');
});

test('connector probe uses the native SDK route with a scoped development subject', async () => {
  let request;
  const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } }, {
    fetcher: async (value) => { request = value; return response({ name: 'warehouse', status: 'passed' }); },
  });
  await api.testConnector('warehouse');
  assert.equal(request.url, 'http://127.0.0.1:8080/v1/studio/sdk/connectors.test');
  assert.equal(request.headers.get('X-Studio-Development-Subject'), 'dev-user');
  assert.equal(await request.text(), '{"name":"warehouse"}');
});

test('view SQL context reads connector driver through the named SDK operation', async()=>{
  let request;
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async(value)=>{request=value;return response({name:'warehouse',driver:'mysql'});}});
  const connector=await api.getConnector('warehouse');
  assert.equal(connector.driver,'mysql');
  assert.equal(request.url,'http://127.0.0.1:8080/v1/studio/sdk/connectors.get');
  assert.equal(await request.text(),'{"name":"warehouse"}');
});

test('generated connector update delegates the current optimistic revision through the native route', async () => {
  let request;
  const api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com' }, {
    fetcher: async (value) => { request = value; return response({ name: 'warehouse', etag: 3 }); },
  });
  await api.updateConnector('warehouse', { description: 'rotated', etag: 2 });
  assert.equal(request.url, 'https://studio.example.com/v1/studio/sdk/connectors.update');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(), '{"name":"warehouse","input":{"description":"rotated","etag":2}}');
});

test('generated connector disable uses the native authenticated route', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{request=value;return response({name:'warehouse',status:'disabled',dsnConfigured:true,secretConfigured:true,etag:3});}});
  const connector=await api.disableConnector('warehouse',2);
  assert.equal(connector.status,'disabled');
  assert.equal(request.url,'https://studio.example.com/v1/studio/sdk/connectors.disable');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(),'{"name":"warehouse","etag":2}');
});

test('generated connector activation uses the native authenticated route', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{request=value;return response({name:'warehouse',status:'active',dsnConfigured:true,secretConfigured:true,etag:3});}});
  const connector=await api.activateConnector('warehouse',2);
  assert.equal(connector.status,'active');
  assert.equal(request.url,'https://studio.example.com/v1/studio/sdk/connectors.activate');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(),'{"name":"warehouse","etag":2}');
});

test('generated connector deletion accepts the native no-content response', async () => {
  let request;
  const api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com' }, {
    fetcher: async (value) => { request = value; return new Response(null,{status:204}); },
  });
  assert.equal(await api.deleteConnector('warehouse', 4), undefined);
  assert.equal(request.url, 'https://studio.example.com/v1/studio/sdk/connectors.delete');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(), '{"name":"warehouse","etag":4}');
});

test('schema browser uses named connector catalog operations', async () => {
  const calls=[];
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async(url,init)=>{calls.push(url instanceof Request?{url:url.url,body:await url.text(),subject:url.headers.get('X-Studio-Development-Subject')}:{url,body:init.body});return response({items:[]});}});
  await api.listSchemas('warehouse',{query:'analytics'});
  await api.listTables('warehouse',{schema:'analytics',query:'vendor'});
  await api.getTable('warehouse',{schema:'analytics',table:'vendors'});
  await api.testSQL('warehouse',{sql:'SELECT * FROM vendors',limit:10});
  assert.equal(calls[0].url,'http://127.0.0.1:8080/v1/studio/sdk/connectors.schemas');
  assert.equal(calls[0].body,'{"name":"warehouse","input":{"query":"analytics"}}');
  assert.equal(calls[1].url,'http://127.0.0.1:8080/v1/studio/sdk/connectors.tables');
  assert.equal(calls[1].body,'{"name":"warehouse","input":{"schema":"analytics","query":"vendor"}}');
  assert.equal(calls[2].url,'http://127.0.0.1:8080/v1/studio/sdk/connectors.table');
  assert.equal(calls[2].body,'{"name":"warehouse","input":{"schema":"analytics","table":"vendors"}}');
  assert.equal(calls[2].subject,'dev-user');
  assert.equal(calls[3].url,'http://127.0.0.1:8080/v1/studio/sdk/connectors.test_sql');
  assert.equal(calls[3].body,'{"name":"warehouse","input":{"sql":"SELECT * FROM vendors","limit":10}}');
});

test('generated report create sends only author-facing fields to the native route', async () => {
  let request;
  const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } }, {
    fetcher: async (value) => { request = value; return response({ id: 'r123', slug: 'vendor-catalog' }); },
  });
  await api.createComponent({ title: 'Vendor Catalog', slug: 'vendor-catalog', defaultConnectorName: 'reporting_mysql' });
  assert.equal(request.url, 'http://127.0.0.1:8080/v1/studio/sdk/components.create');
  assert.equal(request.headers.get('Authorization'), null);
  assert.equal(await request.text(), '{"title":"Vendor Catalog","slug":"vendor-catalog","defaultConnectorName":"reporting_mysql"}');
});

test('reader inspection and command methods retain version identity and revision', async () => {
  const calls = [];
  const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } }, {
    fetcher: async (url, init) => { calls.push(url instanceof Request ? {url:url.url,body:await url.text()} : {url,body:init.body}); return response({}); },
  });
  await api.inspectVersion('vendor-catalog', 1);
  await api.validateVersion('vendor-catalog', 1, 2);
  await api.publishReader('vendor-catalog', 1, 2, 'release reader');
  await api.applyReaderCommand('vendor-catalog', 1, { expectedSourceRevision: 2, operation: { type: 'inspect' } });
  assert.equal(calls[0].url, 'http://127.0.0.1:8080/v1/studio/sdk/versions.inspect');
  assert.equal(calls[0].body, '{"reportId":"vendor-catalog","versionNo":1}');
  await api.inspectVersion('vendor-catalog', 1, { discoverColumns: false });
  assert.equal(calls.at(-1).body, '{"reportId":"vendor-catalog","versionNo":1,"discoverColumns":false}');
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
    fetcher: async (value) => { request = value; return response({ view: 'vendors' }); },
  });
  await api.testReaderView('vendor-catalog', 1, 'vendors', {}, 3);
  assert.equal(request.url, 'http://127.0.0.1:8080/v1/studio/sdk/versions.test_view');
  assert.equal(await request.text(), '{"reportId":"vendor-catalog","versionNo":1,"view":"vendors","input":{"input":{},"limit":3}}');
});

test('relation test delegates exact edge identity through the versioned SDK', async () => {
  let request;
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async(value)=>{request=value;return response({relation:'products'});}});
  await api.testReaderRelation('reader',2,'vendor->products',{tenant:7},25);
  assert.equal(request.url,'http://127.0.0.1:8080/v1/studio/sdk/versions.test_relation');
  assert.equal(await request.text(),'{"reportId":"reader","versionNo":2,"relation":"vendor->products","input":{"input":{"tenant":7},"limit":25}}');
});

test('cube composition playground delegates frames and SQL through the versioned SDK', async () => {
  let request;
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async(value)=>{request=value;return response({data:{data:[]}});}});
  await api.testCubeCompose('spend',1,{cubes:[{dimensions:{status:true},measures:{productCount:true}}],sql:'SELECT * FROM $CubeSQL1'});
  assert.equal(request.url,'http://127.0.0.1:8080/v1/studio/sdk/versions.test_compose');
  assert.match(await request.text(),/\$CubeSQL1/);
});

test('server-owned warmup keeps only immutable version identity in the browser', async () => {
  const requests=[];
  const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } }, {
    fetcher: async (url, init) => { requests.push(url instanceof Request ? {url:url.url,body:await url.text()} : { url, body: init.body }); return response({ items: [], entries: 1 }); },
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
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async(url,init)=>{calls.push(url instanceof Request ? {url:url.url,body:await url.text(),generated:true} : {url,body:init.body,generated:false});return response({files:[]});}});
  await api.getResources('reader',2);
  await api.upsertResourceFile({reportId:'reader',versionNo:2,namespace:'dev.docs',resourcePath:'guide/SKILL.md',content:'guide'});
  await api.deleteResourceFile('reader',2,'file');
  await api.upsertResourceFolder({reportId:'reader',versionNo:2,namespace:'dev.docs',rootPath:'guide',uriPrefix:'skill://dev-guide/'});
  await api.deleteResourceFolder('reader',2,'folder');
  await api.upsertSkillRoot({reportId:'reader',versionNo:2,folderId:'folder',skillRoot:'.'});
  await api.deleteSkillRoot('reader',2,'skill');
  assert.equal(calls.every((call)=>call.generated),true);
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
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'owner'}},{fetcher:async(url,init)=>{calls.push(url instanceof Request ? {url:url.url,body:await url.text()} : {url,body:init.body});return response({items:[],limit:50,offset:0});}});
  await api.listNamespaces({status:'active'});
  await api.getNamespace('finance.ops');
  await api.createNamespace({name:'finance.ops',title:'Finance Operations'});
  await api.updateNamespace('finance.ops',{title:'Finance',etag:1});
  await api.deleteNamespace('finance.ops',2);
  assert.equal(calls[0].url,'http://127.0.0.1:8080/v1/studio/sdk/namespaces.list');
  assert.equal(calls[0].body,'{"status":"active"}');
  assert.equal(calls[1].url,'http://127.0.0.1:8080/v1/studio/sdk/namespaces.get');
  assert.equal(calls[1].body,'{"name":"finance.ops"}');
  assert.equal(calls[2].url,'http://127.0.0.1:8080/v1/studio/sdk/namespaces.create');
  assert.equal(calls[2].body,'{"name":"finance.ops","title":"Finance Operations"}');
  assert.equal(calls[3].url,'http://127.0.0.1:8080/v1/studio/sdk/namespaces.update');
  assert.equal(calls[3].body,'{"name":"finance.ops","input":{"title":"Finance","etag":1}}');
  assert.equal(calls[4].url,'http://127.0.0.1:8080/v1/studio/sdk/namespaces.delete');
});

test('generated namespace delete accepts the native 204 response without parsing JSON', async () => {
  let request;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(value)=>{request=value;return new Response(null,{status:204});}});
  assert.equal(await api.deleteNamespace('finance.ops',3), undefined);
  assert.equal(request.url,'https://studio.example.com/v1/studio/sdk/namespaces.delete');
  assert.equal(request.credentials,'include');
  assert.equal(request.headers.get('Authorization'),null);
  assert.equal(await request.text(),'{"name":"finance.ops","etag":3}');
});

test('publication lifecycle stays behind Studio SDK operations', async () => {
  const calls=[];
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'dev-user'}},{fetcher:async(url,init)=>{calls.push(url instanceof Request ? {url:url.url,body:await url.text(),subject:url.headers.get('X-Studio-Development-Subject')} : {url,body:init.body});return response({status:'active'});}});
  await api.getPublication('reader');
  await api.unpublishReader('reader',3,'retire reader');
  await api.rollbackReader('reader',1,4,'restore reader');
  await api.getRuntimeStatus();
  assert.equal(calls[0].url,'http://127.0.0.1:8080/v1/studio/sdk/publications.get');
  assert.equal(calls[0].body,'{"reportId":"reader"}');
  assert.equal(calls[0].subject,'dev-user');
  assert.equal(calls[1].url,'http://127.0.0.1:8080/v1/studio/sdk/publications.unpublish');
  assert.equal(calls[1].body,'{"reportId":"reader","input":{"expectedActiveGeneration":3,"reason":"retire reader"}}');
  assert.equal(calls[1].subject,'dev-user');
  assert.equal(calls[2].url,'http://127.0.0.1:8080/v1/studio/sdk/publications.rollback');
  assert.equal(calls[2].body,'{"reportId":"reader","versionNo":1,"input":{"expectedSourceRevision":4,"reason":"restore reader"}}');
  assert.equal(calls[2].subject,'dev-user');
  assert.equal(calls[3].url,'http://127.0.0.1:8080/v1/studio/sdk/runtime.status');
  assert.equal(calls[3].body,'{}');
  assert.equal(calls[3].subject,'dev-user');
});

test('generated publication get preserves not-found evidence', async () => {
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async()=>response({message:'publication not found'},404,'publication-request-1')});
  await assert.rejects(()=>api.getPublication('missing'),(error)=>error.status===404&&error.code==='not_found'&&error.requestId==='publication-request-1');
});

test('publication event history stays behind the owner-scoped SDK operation', async () => {
  const calls=[];
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://127.0.0.1:8080',development:{subject:'owner'}},{fetcher:async(url,init)=>{calls.push(url instanceof Request ? {url:url.url,body:await url.text(),subject:url.headers.get('X-Studio-Development-Subject')} : {url,body:init.body});return response({items:[],limit:25,offset:0});}});
  await api.listPublicationEvents('reader',{limit:25,offset:0});
  assert.equal(calls[0].url,'http://127.0.0.1:8080/v1/studio/sdk/publications.events.list');
  assert.equal(calls[0].body,'{"reportId":"reader","input":{"limit":25,"offset":0}}');
  assert.equal(calls[0].subject,'owner');
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

test('shared resource access client uses canonical routes and BFF session', async () => {
  const calls=[];
  const resource={kind:'component',id:'reader',tenant:'one',version:'1'};
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(request)=>{
    calls.push({url:request.url,body:await request.text(),credentials:request.credentials,authorization:request.headers.get('Authorization')});
    return response(request.url.endsWith('policies.context')?{context:{canManage:true,choices:{},source:'verified-principal'}}:{document:{resource,revision:request.url.endsWith('policies.replace')?2:1,policies:{}}});
  }});
  assert.equal((await api.getResourceAccess(resource)).revision,1);
  assert.equal((await api.getResourceAccessContext(resource)).canManage,true);
  assert.equal((await api.replaceResourceAccess({resource,revision:1,policies:{}})).revision,2);
  assert.deepEqual(calls.map(({url})=>url),['policies.get','policies.context','policies.replace'].map(operation=>`https://studio.example.com/v1/authz/sdk/${operation}`));
  assert.deepEqual(calls.map(({body})=>JSON.parse(body)),[{resource},{resource},{document:{resource,revision:1,policies:{}}}]);
  assert.ok(calls.every(({credentials,authorization})=>credentials==='include'&&authorization===null));
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

test('generated ACL delete accepts 204 and preserves optimistic conflict metadata', async () => {
  let sent;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(request)=>{sent=request;return new Response(null,{status:204});}});
  assert.equal(await api.deleteACL('reader','user','viewer',2),undefined);
  assert.equal(sent.url,'https://studio.example.com/v1/studio/sdk/acl.delete');
  assert.equal(sent.credentials,'include');
  assert.equal(sent.headers.get('Authorization'),null);
  assert.equal(await sent.text(),'{"reportId":"reader","subjectType":"user","subjectId":"viewer","etag":2}');
  const conflictAPI=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async()=>response({message:'ACL etag does not match',expectedEtag:2,currentEtag:3},409,'acl-conflict-1')});
  await assert.rejects(()=>conflictAPI.deleteACL('reader','user','viewer',2),(error)=>error.status===409&&error.expectedEtag===2&&error.currentEtag===3&&error.requestId==='acl-conflict-1');
});

test('generated ACL upsert preserves the native grant and conflict contract', async () => {
  let sent;
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async(request)=>{sent=request;return response({reportId:'reader',subjectType:'user',subjectId:'viewer',canView:true,etag:1});}});
  const grant=await api.upsertACL({reportId:'reader',subjectType:'user',subjectId:'viewer',canView:true});
  assert.equal(grant.etag,1);
  assert.equal(sent.url,'https://studio.example.com/v1/studio/sdk/acl.upsert');
  assert.equal(sent.credentials,'include');
  assert.equal(sent.headers.get('Authorization'),null);
  assert.equal(await sent.text(),'{"reportId":"reader","subjectType":"user","subjectId":"viewer","canView":true}');
  const conflictAPI=new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async()=>response({message:'ACL etag does not match',expectedEtag:1,currentEtag:2},409,'acl-upsert-conflict')});
  await assert.rejects(()=>conflictAPI.upsertACL({reportId:'reader',subjectType:'user',subjectId:'viewer',canView:true,etag:1}),(error)=>error.status===409&&error.expectedEtag===1&&error.currentEtag===2&&error.requestId==='acl-upsert-conflict');
});

test('authenticated preview executes the exact draft version through the SDK', async () => {
  let request;
  const api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com' }, {
    fetcher: async (value) => { request = value; return response({ data: { Summaries: [] } }); },
  });
  const result = await api.previewReader('vendor-spend', 1, {}, 50, '/v1/studio/readers/vendor-product-summary');
  assert.equal(request.url, 'https://studio.example.com/v1/studio/sdk/preview.execute');
  assert.equal(request.credentials, 'include');
  assert.equal(request.headers.get('Authorization'), null);
  assert.equal(await request.text(), '{"reportId":"vendor-spend","versionNo":1,"input":{"input":{},"limit":50,"cube":false}}');
  assert.deepEqual(result.data, { Summaries: [] });
});

test('default browser fetch is invoked with its Window receiver', async () => {
  const original = globalThis.fetch;
  let receiver;
  globalThis.fetch = function () { receiver = this; return Promise.resolve(response({ items: [] })); };
  try {
    const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } });
    await api.listComponents();
    assert.equal(receiver, globalThis);
  } finally { globalThis.fetch = original; }
});

test('SDK errors retain conflict metadata for authoring recovery', async () => {
  const api = new StudioAPI({ mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } }, { fetcher: async () =>
    response({ code: 'conflict', message: 'version source revision does not match', field: 'sourceRevision' }, 409) });
  await assert.rejects(api.applyReaderCommand('reader', 1, { expectedSourceRevision: 2, operation: { type: 'inspect' } }), (error) => {
    assert.equal(error.code, 'conflict');
    assert.equal(error.status, 409);
    assert.equal(error.field, 'sourceRevision');
    return true;
  });
});

const namespaceA = 'a'.repeat(64);
const namespaceB = 'b'.repeat(64);

function deferred() {
  let resolve;
  const promise = new Promise((done) => { resolve = done; });
  return { promise, resolve };
}

test('resource requests pin namespace headers while global directories stay unscoped', async () => {
  const seen = [];
  const api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com' }, {
    fetcher: async (value, init) => { const request = new Request(value, init); seen.push(request.headers.get('X-Studio-Namespace')); return response({ items: [] }); },
  });
  api.setNamespace(namespaceA);
  await api.listComponents();
  await api.listNamespaces();
  await api.listConnectors();
  await api.invoke('resources.get', {});
  assert.deepEqual(seen, [namespaceA, null, null, namespaceA]);
  assert.throws(() => api.setNamespace('forecasting'), TypeError);
  assert.equal(api.namespaceId, namespaceA);
});

test('a response from an earlier namespace is discarded even after A to B to A', async () => {
  const started = deferred();
  const release = deferred();
  const api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com' }, {
    fetcher: async (request) => { assert.equal(request.headers.get('X-Studio-Namespace'), namespaceA); started.resolve(); await release.promise; return response({ items: [{ id: 'old-a' }] }); },
  });
  api.setNamespace(namespaceA);
  const pending = api.listComponents();
  await started.promise;
  api.setNamespace(namespaceB);
  api.setNamespace(namespaceA);
  const rejected = assert.rejects(pending, (error) => error.name === 'AbortError' && error.code === 'namespace_changed');
  release.resolve();
  await rejected;
});

test('late unauthorized responses do not expire the new namespace view', async () => {
  const started = deferred();
  const release = deferred();
  let expired = false;
  const api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com' }, {
    onUnauthorized: () => { expired = true; },
    fetcher: async () => { started.resolve(); await release.promise; return response({ message: 'expired' }, 401); },
  });
  api.setNamespace(namespaceA);
  const pending = api.listComponents();
  await started.promise;
  api.setNamespace(namespaceB);
  const rejected = assert.rejects(pending, { code: 'namespace_changed' });
  release.resolve();
  await rejected;
  assert.equal(expired, false);
});

test('token refresh cannot retry a resource request into a different namespace', async () => {
  const seen = [];
  let expired = false;
  let api;
  const identity = { token: async (force) => { if (force) api.setNamespace(namespaceB); return 'signed-token'; } };
  api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com', authentication: { mode: 'identity-token' } }, {
    identity, onUnauthorized: () => { expired = true; },
    fetcher: async (request) => { seen.push(request.headers.get('X-Studio-Namespace')); return response({ message: 'expired' }, 401); },
  });
  api.setNamespace(namespaceA);
  await assert.rejects(api.listComponents(), { code: 'namespace_changed' });
  assert.deepEqual(seen, [namespaceA]);
  assert.equal(expired, false);
});

test('shared authorization check retains namespace and distinguishes denial from outage', async () => {
  const resource = {kind:'component',id:'reader',tenant:'one',version:'1'};
  const namespace = 'a'.repeat(64);
  let status = 200;
  const api = new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:async request => {
    assert.equal(request.url, 'https://studio.example.com/v1/authz/sdk/authorization.check');
    assert.equal(request.headers.get('X-Studio-Namespace'), namespace);
    assert.deepEqual(JSON.parse(await request.text()), {resource, action:'execute'});
    return response(status === 200 ? {decision:{bounded:false,entities:[]}} : {message:'access unavailable'}, status);
  }});
  api.setNamespace(namespace);
  assert.equal((await api.checkCurrentAccess(resource, 'execute')).effect, 'allow');
  status = 403;
  assert.equal((await api.checkCurrentAccess(resource, 'execute')).effect, 'deny');
  status = 503;
  await assert.rejects(() => api.checkCurrentAccess(resource, 'execute'), error => error.status === 503 && error.code === 'unavailable');
});

test('shared policy read rejects a response after namespace switch', async () => {
  let release;
  const resource = {kind:'component',id:'reader',tenant:'one',version:'1'};
  const api = new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'}, {fetcher:() => new Promise(resolve => { release = resolve; })});
  api.setNamespace('a'.repeat(64));
  const pending = api.getResourceAccess(resource);
  while (!release) await new Promise(resolve => setImmediate(resolve));
  api.setNamespace('b'.repeat(64));
  release(response({document:{resource,revision:1,policies:{}}}));
  await assert.rejects(() => pending, error => error.code === 'namespace_changed');
});
