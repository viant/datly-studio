import {test} from 'node:test';
import assert from 'node:assert/strict';
import {managementWindowMetadata, managementWindowTransport} from './managementWindow.js';
const actions = {code: '({save:()=>context.Context("draft").handlers.dataSource.getFormData()})'};
const source = {id: 'rows', backend: {kind: 'mcp_tool', method: 'studio.sdk.connectors.list', service: 'mcp://datastudio/admin'}, selectors: {data:'items'}};
const window = {windowKey:'connectors', actions, dataSource:{rows:{cardinality:'collection'}}, view:{content:{id:'canonical-table'}}};
const envelope = {schemaVersion:3,targets:[{target:{platform:'web',formFactor:'desktop'},variant:'desktop'}],variants:{desktop:{window,dataSources:{rows:source}}}};
test('uses the published layout and action objects without a screen-specific transformation',()=>{
 const metadata=managementWindowMetadata(envelope);
 assert.equal(metadata.view,window.view);
 assert.equal(metadata.actions,actions);
 assert.deepEqual(metadata.dataSource.rows,{...source,cardinality:'collection'});
 assert.equal(window.dataSource.rows.backend,undefined);
 assert.throws(()=>managementWindowMetadata(envelope,{platform:'ios',formFactor:'phone'}),/no matching target/);
});
test('forwards exact authored SDK method and paging/filter input to the host SDK',async()=>{
 const calls=[];const page={items:[{name:'warehouse'}]};
 const transport=managementWindowTransport({invoke:async(...args)=>{calls.push(args);return page;}},managementWindowMetadata(envelope));
 assert.equal(await transport({dataSourceRef:'rows',options:{inputParameters:{offset:25,limit:25,query:'warehouse'}}}),page);
 assert.deepEqual(calls,[['connectors.list',{offset:25,limit:25,query:'warehouse'}]]);
 await assert.rejects(()=>transport({dataSourceRef:'other'}),/Unknown/);
});
test('uses the native MCP adapter when configured, with no HTTP fallback',async()=>{
 const calls=[];
 const transport=managementWindowTransport({nativeCallTool:()=>{},nativeRequest:async(...args)=>{calls.push(args);return {items:[]};},invoke:()=>{throw new Error('HTTP fallback');}},managementWindowMetadata(envelope));
 await transport({dataSourceRef:'rows'});
 assert.deepEqual(calls,[[null,'connectors.list',{}]]);
});
