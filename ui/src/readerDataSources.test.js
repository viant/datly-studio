import test from 'node:test';
import assert from 'node:assert/strict';
import {readerDataSources,fetchReaderDataSource} from './readerDataSources.js';
import {readerDimensions,readerViews} from './readerViews.js';
import {predicateOptions} from './predicateDraft.js';

test('reader lookups preserve server view, column, and predicate identities',()=>{
 const structure={availablePredicates:['expr','host_filter'],component:{rootView:{name:'Records',namespace:'rows',columns:[{name:'PublicName',source:'c.name'}],relations:[{view:{name:'Detail',namespace:'detail',columns:[{name:'ChildID',source:'d.id'}]}}]}}};
 assert.deepEqual(readerViews(structure),[{name:'rows',label:'Records (rows)'},{name:'detail',label:'Detail (detail)'}]);
 assert.deepEqual(readerDimensions(structure,'detail'),[{name:'d.id',label:'ChildID (d.id)'}]);
 assert.deepEqual(predicateOptions(structure),['expr','host_filter']);
 assert.deepEqual(predicateOptions({}),[]);
});

test('every lookup uses an MCP datasource, including discovered and saved predicates',async()=>{
 const seen=[];const api={fetchMCPDataSource:async(source,input)=>{seen.push({source,input});return []}};
 for(const name of Object.keys(readerDataSources))await fetchReaderDataSource(api,name,{reportId:'reader',versionNo:3});
 assert.equal(seen.length,4);
 assert.ok(seen.every(({source})=>source.backend.kind==='mcp_tool'&&source.cache.enabled===false));
 assert.equal(seen.find(({source})=>source.id==='reader_definition').source.backend.method,'studio.sdk.versions.inspect');
 await assert.rejects(fetchReaderDataSource(api,'unknown'),/Unknown reader lookup/);
});
