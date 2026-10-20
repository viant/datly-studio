import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
const forms=new Map(),collections=new Map(),calls=[];
const parameters={permissionsOnly:true,kinds:['window'],actionsByKind:{window:['read']},choices:{roles:['reader']}};
const context={windowState:{parameters},Context:id=>({handlers:{dataSource:{
 getFormData:()=>forms.get(id)||{},
 setFormData:({values})=>forms.set(id,values),
 getCollection:()=>collections.get(id)||[],
 setCollection:values=>collections.set(id,values),
 fetchRecords:async({parameters})=>{calls.push({id,parameters});return {data:[{items:[],hasMore:false}]};},
}}})};
const source=readFileSync(new URL('./windows/security/shared/main.js',import.meta.url),'utf8');
const actions=new Function('context',`return ${source}`)(context);
assert.equal(await actions.initialize(),true);
assert.equal(forms.get('security_state').heading,'Permissions');
assert.deepEqual(forms.get('security_state').actionsByKind,parameters.actionsByKind);
assert.deepEqual(collections.get('security_kinds'),[{id:'window',label:'window'}]);
assert.equal(calls[0].id,'security_access');
assert.equal(forms.get('security_state').outcome,'');
// No native host adapter is required to initialize the shared definition.
assert.equal(context.services,undefined);
console.log('Shared Security uses standard Forge parameters without a native host adapter.');
