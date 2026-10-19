import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {parseDocument} from '../../ui/node_modules/yaml/dist/index.js';

async function fixture(screen) {
 const source=await readFile(new URL(`./windows/${screen}/shared/main.js`,import.meta.url),'utf8');
 let form={}, selected=null,refresh=0,opened=0;const windowForm={};const collections=new Map();const calls=[];const failures=new Map();
 const responses=new Map();
 const dataSource=id=>({
  getWindowFormData:()=>windowForm,getFormData:()=>form,setFormData:({values})=>{form={...values};},setFormField:({item,value})=>{form={...form,[item.dataField]:value};},
  setCollection:rows=>collections.set(id,rows),setInputParameters:()=>{},peekSelection:()=>({selected}),fetchCollection:()=>{refresh++;},
  fetchRecords:async({parameters})=>{calls.push({id,parameters});if(failures.has(id))throw failures.get(id);return {data:[responses.get(id)||{name:parameters.name,etag:8,status:'passed'}]};}
 });
 const context={Context:id=>({handlers:{dataSource:dataSource(id)}}),handlers:{window:{openDialog:()=>{opened++;}}}};
 const utilities={yaml:{parse(text){const doc=parseDocument(text,{uniqueKeys:true});if(doc.errors.length)throw doc.errors[0];return doc.toJS();}}};
 const actions=new Function('context','utilities',`return ${source}`)(context,utilities);
 return {actions,calls,responses,failures,windowForm,collections,get form(){return form;},set form(value){form=value;},set selected(value){selected=value;},get refresh(){return refresh;},get opened(){return opened;}};
}
{
 const f=await fixture('connectors');f.actions.newConnector();assert.equal(f.opened,1);
 f.form={...f.form,name:'warehouse',yamlText:'Driver: mysql\nDSN: "${Username}:${Password}@tcp(127.0.0.1:3306)/example"\nSecret:\n  URL: file:///tmp/mysql.enc\n  Key: blowfish://default'};
 assert.equal(await f.actions.applyYAML(),true);
 assert.equal(await f.actions.saveConnector(),true);
 assert.deepEqual(f.calls.at(-1),{id:'connector_create',parameters:{name:'warehouse',driver:'mysql',dsnTemplate:'${Username}:${Password}@tcp(127.0.0.1:3306)/example',secretRef:'file:///tmp/mysql.enc|blowfish://default',description:'',options:{}}});
 f.selected={name:'warehouse',etag:1};f.responses.set('connector_get',{name:'warehouse',driver:'mysql',etag:8,secretRefDisplay:'file:///tmp/mysql.enc|blowfish://default',options:{pool:2}});
 assert.equal(await f.actions.editConnector(),true);assert.equal(f.form.secretRefDisplay,'file:///tmp/mysql.enc|blowfish://default');
 assert.equal(await f.actions.saveConnector(),true);
 assert.deepEqual(f.calls.at(-1),{id:'connector_update',parameters:{name:'warehouse',input:{driver:'mysql',description:'',options:{pool:2},etag:8}}});
 f.form={...f.form,clearSecret:true};await f.actions.saveConnector();assert.equal(f.calls.at(-1).parameters.input.secretRef,'');
 await f.actions.testConnector();assert.equal(f.calls.at(-1).id,'connector_test');
 await f.actions.activateConnector();assert.deepEqual(f.calls.at(-1),{id:'connector_activate',parameters:{name:'warehouse',etag:8}});
 f.failures.set('connector_update',new Error('Revision conflict'));
 assert.equal(await f.actions.saveConnector(),false);assert.equal(f.form.outcome,'Revision conflict');
 f.form={...f.form,yamlText:'Driver: mysql\nDriver: sqlite'};assert.equal(await f.actions.applyYAML(),false);
 f.form={...f.form,yamlText:'Secret: null'};assert.equal(await f.actions.applyYAML(),true);assert.equal(f.form.clearSecret,true);
}
{
 const f=await fixture('namespaces');f.actions.newNamespace();f.form={...f.form,name:'inventory',title:'Inventory',mcpEnabled:true};
 assert.equal(await f.actions.saveNamespace(),true);
 assert.deepEqual(f.calls.at(-1),{id:'namespace_create',parameters:{name:'inventory',title:'Inventory',description:'',visibility:'private',allowedRoles:[],mcpEnabled:true,mcpPort:0}});
 f.selected={name:'inventory'};f.responses.set('namespace_get',{name:'inventory',title:'Inventory',etag:13,status:'active',visibility:'private',allowedRoles:['editors'],mcpEnabled:true,mcpPort:8097});
 await f.actions.editNamespace();assert.equal(f.form.mcpPort,8097);
 assert.equal(await f.actions.saveNamespace(),true);
 assert.deepEqual(f.calls.at(-1),{id:'namespace_update',parameters:{name:'inventory',input:{title:'Inventory',description:'',visibility:'private',allowedRoles:['editors'],mcpEnabled:true,mcpPort:8097,status:'active',etag:13}}});
 f.failures.set('namespace_update',new Error('Forbidden'));assert.equal(await f.actions.saveNamespace(),false);assert.equal(f.form.outcome,'Forbidden');
}
console.log('functional management actions: exact MCP inputs, secret omission/removal, YAML, etags, test/activate, dynamic port, conflict and denial passed');

{
 const f=await fixture('components'), namespaceId='a'.repeat(64);
 f.selected={id:'reader'};assert.equal(await f.actions.openReader(),false);
 f.windowForm.namespaceId=namespaceId;f.actions.selectNamespace();
 const file={reportId:'reader',versionNo:2,resourceId:'sql-file',namespace:'reader_sql',resourcePath:'sql/main.sql',content:'SELECT 1',mediaType:'text/plain'};
 f.responses.set('component_components_get',{id:'reader',currentDraftVersion:2});
 f.responses.set('component_versions_inspect',{version:{versionNo:2,sourceRevision:7,state:'draft'},dql:'SELECT 1',diagnostics:[]});
 f.responses.set('component_resources_get',{files:[file],version:{sourceRevision:7}});
 assert.equal(await f.actions.openReader(),true);assert.equal(f.form.sourceRevision,7);
 f.form={...f.form,source:'SELECT 2'};assert.equal(await f.actions.saveDQL(),true);
 assert.deepEqual(f.calls.find(call=>call.id==='component_versions_apply').parameters,{namespaceId,reportId:'reader',versionNo:2,command:{kind:'set_dql',expectedSourceRevision:7,payload:{authoredDql:'SELECT 2'}}});
 await f.actions.openFile({item:{...file,treePath:'reader_sql/sql/main.sql',label:'main.sql'}});f.form={...f.form,source:'SELECT 3'};
 f.responses.set('component_resources_upsert_file',{files:[{...file,content:'SELECT 3'}],version:{sourceRevision:8}});
 assert.equal(await f.actions.saveFile(),true);
 assert.deepEqual(f.calls.find(call=>call.id==='component_resources_upsert_file').parameters,{namespaceId,reportId:'reader',versionNo:2,resourceId:'sql-file',namespace:'reader_sql',resourcePath:'sql/main.sql',mediaType:'text/plain',content:'SELECT 3',expectedSourceRevision:7});
 assert.equal(f.form.sourceRevision,8);assert.equal(await f.actions.saveDQL(),false);
 f.responses.set('component_versions_validate',{valid:true,diagnostics:[],version:{sourceRevision:8}});await f.actions.validateReader();
 assert.equal(f.calls.at(-1).parameters.expectedSourceRevision,8);
 f.form={...f.form,previewInput:'{"ProductID":[4]}',limit:25,reason:'release'};
 await f.actions.previewReader();assert.deepEqual(f.calls.at(-1).parameters.input,{input:{ProductID:[4]},limit:25,cube:false});
 await f.actions.publishReader();assert.deepEqual(f.calls.at(-1).parameters.input,{expectedSourceRevision:8,reason:'release'});
 f.responses.set('component_publications_get',{activeGeneration:11});await f.actions.unpublishReader();assert.deepEqual(f.calls.at(-1).parameters.input,{expectedActiveGeneration:11,reason:'release'});
 f.form={...f.form,structureCommand:'{"kind":"set_view","view":{"name":"main","sql":"SELECT 4"}}'};
 f.responses.set('component_versions_builder',{applied:true});assert.equal(await f.actions.applyStructureCommand(),true);
 const command=f.calls.find(call=>call.id==='component_versions_builder').parameters.command;
 assert.equal(command.expectedSourceRevision,8);assert.equal(command.operation.view.sql,'SELECT 4');
 f.failures.set('component_resources_upsert_file',new Error('Source revision conflict'));
 await f.actions.openFile({item:file});assert.equal(await f.actions.saveFile(),false);assert.equal(f.form.outcome,'Source revision conflict');
}
console.log('component controller: selected namespace, source/file CAS, typed operation inputs, validation, preview, publication and conflict passed');
