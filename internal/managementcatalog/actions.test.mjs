import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {parseDocument} from '../../ui/node_modules/yaml/dist/index.js';

async function fixture(screen) {
 const source=await readFile(new URL(`./windows/${screen}/shared/main.js`,import.meta.url),'utf8');
 let form={}, selected=null,refresh=0,opened=0;const calls=[];const failures=new Map();
 const responses=new Map();
 const dataSource=id=>({
  getFormData:()=>form,setFormData:({values})=>{form={...values};},setFormField:({item,value})=>{form={...form,[item.dataField]:value};},
  peekSelection:()=>({selected}),fetchCollection:()=>{refresh++;},
  fetchRecords:async({parameters})=>{calls.push({id,parameters});if(failures.has(id))throw failures.get(id);return {data:[responses.get(id)||{name:parameters.name,etag:8,status:'passed'}]};}
 });
 const context={Context:id=>({handlers:{dataSource:dataSource(id)}}),handlers:{window:{openDialog:()=>{opened++;}}}};
 const utilities={yaml:{parse(text){const doc=parseDocument(text,{uniqueKeys:true});if(doc.errors.length)throw doc.errors[0];return doc.toJS();}}};
 const actions=new Function('context','utilities',`return ${source}`)(context,utilities);
 return {actions,calls,responses,failures,get form(){return form;},set form(value){form=value;},set selected(value){selected=value;},get refresh(){return refresh;},get opened(){return opened;}};
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
