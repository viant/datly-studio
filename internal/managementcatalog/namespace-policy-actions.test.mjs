import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {parseDocument} from '../../ui/node_modules/yaml/dist/index.js';

async function fixture(screen) {
 let source=await readFile(new URL(`./windows/${screen}/shared/main.js`,import.meta.url),'utf8');
 if(screen==='connectors') {const behavior=await readFile(new URL('./windows/connectors/shared/behavior.js',import.meta.url),'utf8');source=`(() => {${behavior.replace('export default ', '')};return ${source};})()`;}
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
 const f=await fixture('namespaces');
 f.selected={name:'restricted',canManage:true};
 f.responses.set('namespace_settings_get',{etag:17,canEdit:true,settings:{security:{predicatePackages:[],requireSecurityPredicate:false,requireInternalComponents:true}}});
 assert.equal(await f.actions.editNamespacePolicies(),true);
 assert.equal(f.form.requireInternalComponents,true);
 assert.equal(await f.actions.saveNamespacePolicies(),true);
 const saved=f.calls.find(x=>x.id==='namespace_settings_update');
 assert.equal(saved.parameters.name,'restricted');assert.equal(saved.parameters.input.etag,17);
 assert.equal(saved.parameters.input.settings.security.requireInternalComponents,true);
 f.form={...f.form,requireInternalComponents:false};
 assert.equal(await f.actions.saveNamespacePolicies(),true);
 assert.equal(f.calls.at(-1).parameters.input.settings.security.requireInternalComponents,false);
 f.form={...f.form,canEdit:false};const count=f.calls.length;
 assert.equal(await f.actions.saveNamespacePolicies(),false);assert.equal(f.calls.length,count);
 const doc=parseDocument(await readFile(new URL('./windows/namespaces/shared/main.yaml',import.meta.url),'utf8')).toJS();
 const controls=[];const visit=x=>{if(!x||typeof x!=='object')return;if(x.id==='requireInternalComponents')controls.push(x);Object.values(x).forEach(visit);};visit(doc);
 assert.equal(controls.length,1);assert.equal(controls[0].type,'checkbox');
 assert.deepEqual(controls[0].readOnlyWhen,{selector:'canEdit',equals:false});
}
console.log('namespace internal-component setting: readback, true/false save, CAS payload and administrator-only control passed');
