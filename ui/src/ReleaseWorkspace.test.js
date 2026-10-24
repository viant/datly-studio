import assert from 'node:assert/strict';
import test from 'node:test';
import {StudioAPI} from './studioApi.js';

const namespaceId='a'.repeat(64);
const namespace='sales';
const methods=['releases/list','releases/candidates','releases/publish','releases/promote','releases/history'];

test('Data Studio advertises Releases only when every release tool is present and caches the selected namespace check',async()=>{
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://studio.test',development:{subject:'fixture'}});
  api.setNamespace(namespaceId);
  let reads=0;
  api.listMCPTools=async()=>{reads++;return methods.map(name=>({name}));};
  assert.equal(await api.supportsReleases(),true);
  assert.equal(await api.supportsReleases(),true);
  assert.equal(reads,1);
  api.setNamespace('b'.repeat(64));
  api.listMCPTools=async()=>{reads++;return methods.slice(0,4).map(name=>({name}));};
  assert.equal(await api.supportsReleases(),false);
  assert.equal(reads,2);
});

test('Data Studio release requests use authenticated MCP with selected namespace ID header and DTO name only',async()=>{
  const sent=[];
  const api=new StudioAPI({mode:'development',apiBaseURL:'http://studio.test',development:{subject:'fixture'}},{nativeCallTool:async(method,input,options)=>{sent.push({method,input,headers:options.headers});return {channels:[]};}});
  api.setNamespace(namespaceId);
  const result=await api.callReleaseTool('releases/list',{namespace});
  assert.deepEqual(result,{channels:[]});
  assert.equal(sent[0].method,'releases/list');
  assert.deepEqual(sent[0].input,{namespace});
  assert.equal(sent[0].headers['X-Studio-Namespace'],namespaceId);
  await assert.rejects(()=>api.callReleaseTool('unrelated/delete',{namespace}),/Unsupported release operation/);
});

test('Data Studio cookie-mode release request uses the existing MCP tools/call endpoint',async()=>{
  const calls=[];
  const api=new StudioAPI({mode:'authenticated',apiBaseURL:'http://studio.test'},{fetcher:async(url,init)=>{
    calls.push({url,init});
    return Response.json({jsonrpc:'2.0',id:1,result:{structuredContent:{channels:[]}}});
  }});
  api.setNamespace(namespaceId);
  assert.deepEqual(await api.callReleaseTool('releases/list',{namespace}),{channels:[]});
  assert.equal(calls[0].url,'http://studio.test/v1/studio/sdk-mcp/mcp');
  assert.equal(calls[0].init.headers['Mcp-Name'],'releases/list');
  assert.equal(calls[0].init.headers['X-Studio-Namespace'],namespaceId);
  assert.equal(calls[0].init.credentials,'include');
  assert.deepEqual(JSON.parse(calls[0].init.body).params.arguments,{namespace});
});
