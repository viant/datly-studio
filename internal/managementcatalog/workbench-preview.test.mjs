import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {parse} from '../../ui/node_modules/yaml/dist/index.js';
import {createWorkbenchAPI} from '../../ui/node_modules/forge/src/components/readerWorkbench/apiBindings.js';

test('authored preview binding forwards optional refresh separately from native reader inputs',async()=>{
 const spec=parse(readFileSync(new URL('./windows/reader-workbench/shared/main.yaml',import.meta.url),'utf8')).window.view.content.readerWorkbench;
 const calls=[];
 const context={Context:ref=>({handlers:{dataSource:{fetchRecords:async request=>{calls.push({ref,...request});return {rows:[{data:[]}]};}}}})};
 const api=createWorkbenchAPI(context,spec);
 await api.previewReader('reader',2,{enabled:false,limit:0},50,'/records',false);
 assert.deepEqual(calls.at(-1),{ref:'workbench_preview_reader',parameters:{reportId:'reader',versionNo:2,input:{input:{enabled:false,limit:0},limit:50,cube:false}}});
 for(const refreshSchema of [true,false]){
  await api.previewReader('reader',3,{ids:[1]},25,'/records',false,{refreshSchema});
  assert.deepEqual(calls.at(-1).parameters,{reportId:'reader',versionNo:3,input:{input:{ids:[1]},limit:25,cube:false,refreshSchema}});
 }
});
