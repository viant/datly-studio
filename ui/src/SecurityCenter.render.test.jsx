import React from 'react';
import {test,expect,vi} from 'vitest';
import {render,screen,waitFor,within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ManagementWindowRuntime from './ManagementWindowRuntime.jsx';
import {managementResourceFixture} from './managementWindowFixture.js';
let serial=0;
const resource={kind:'component',id:'opaque-id',tenant:'tenant-a',version:'8'};
const entry={kind:'skill',id:'skill-id',name:'Forecasting guide',ownerName:'Forecasting',resource,inherited:true,explicitAcl:true,public:false};
const predicate={name:'iam.vendor.read',title:'Vendor scope',packagePath:'example.com/first/predicates',typeName:'CustomerScope',columns:['id'],alias:'v',status:'active',etag:7,linked:true};
const types=[{packagePath:'example.com/first/predicates',typeName:'CustomerScope',description:'Limits visibility'},{packagePath:'example.com/second/predicates',typeName:'CustomerScope'}];
function mount({items=[entry],readonly=false,policyConflict=false,kindContract,hasMore=true,parameters}={}){
 const dirty=vi.fn();const api={readManagementResource:async()=>{const value=managementResourceFixture('security');if(kindContract){for(const variant of Object.values(value.variants)){const tab=variant.window.view.content.containers.find(node=>node.id==='security-tabs').containers[0];tab.containers.find(node=>node.id==='security-policy-panel').resourcePolicyEditor.actionsByKind=kindContract;}}return value;},fetchMCPDataSource:vi.fn(async(source,input)=>{
  switch(source.backend.method){
   case 'studio.sdk.access.list':return {items,hasMore};
   case 'authz.sdk.policies.get':return {document:{resource:input.resource,revision:4,policies:{execute:{mode:'protected',rule:{kind:'role',value:'reader'}}}}};
   case 'authz.sdk.policies.context':return {context:{canManage:!readonly,source:'provider-directory',choices:{role:[{id:'reader',label:'Reader'},{id:'analyst',label:'Analyst'}]}}};
   case 'authz.sdk.policies.replace':if(policyConflict)throw Object.assign(new Error('revision mismatch'),{code:'conflict'});return {document:{...input.document,revision:5}};
   case 'authz.sdk.policies.create':return {document:{...input.document,revision:1}};
   case 'authz.sdk.authorization.check':return {decision:{bounded:true,entities:[]}};
   case 'studio.sdk.authorization_predicates.list':return {items:[predicate]};
   case 'studio.sdk.authorization_predicates.types':return {items:types};
   case 'studio.sdk.authorization_predicates.get':return predicate;
   case 'studio.sdk.authorization_predicates.create':return {...predicate,...input};
   case 'studio.sdk.authorization_predicates.update':return {...predicate,...input.input};
   case 'studio.sdk.authorization_predicates.delete':return {};
   default:throw new Error('Unexpected '+source.backend.method);
  }
 })};
 render(<ManagementWindowRuntime api={api} uri="window://datastudio/security" instanceKey={'forge-security-'+(++serial)} onDirtyChange={dirty} parameters={parameters}/>);return {api,dirty};
}
test('metadata Security selects inherited resources and exposes the exact read-only review link',async()=>{
 const user=userEvent.setup(),{api}=mount();await user.click(await screen.findByRole('button',{name:'Open permissions',exact:true}));await screen.findByText('Policy revision 4');
 expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='authz.sdk.policies.get')[1]).toEqual({resource});expect(screen.getByText(/inherits permissions from Forecasting/)).toBeTruthy();
 const link=screen.getByRole('link',{name:'Review current permissions'}),url=new URL(link.href);expect(Object.fromEntries(url.searchParams)).toEqual({mode:'live',kind:'component',id:'opaque-id',tenant:'tenant-a',version:'8'});expect(link.target).toBe('_blank');expect(link.rel).toContain('noopener');
});
test('metadata Security retains reviewed CAS edits and blocks writes when the server denies management',async()=>{
 const user=userEvent.setup();mount({readonly:true});await user.click(await screen.findByRole('button',{name:'Open permissions',exact:true}));await screen.findByText('You have read-only access.');expect(screen.getByLabelText('Access mode').disabled).toBe(true);expect(screen.queryByRole('button',{name:'Review changes'})).toBeNull();
});
test('metadata Security preserves policy edits after a revision conflict',async()=>{
 const user=userEvent.setup(),{api,dirty}=mount({policyConflict:true});await user.click(await screen.findByRole('button',{name:'Open permissions',exact:true}));await screen.findByText('Policy revision 4');await user.click(screen.getByRole('button',{name:/^execute Protected/}));await user.selectOptions(screen.getByLabelText('rule value'),'analyst');await user.click(screen.getByRole('button',{name:'Review changes',exact:true}));
 expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='authz.sdk.policies.replace')).toBe(false);await user.click(screen.getByRole('button',{name:'Save permissions',exact:true}));await screen.findByText(/Your edits are preserved/);expect(screen.getByLabelText('rule value').value).toBe('analyst');expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='authz.sdk.policies.replace')[1].document.revision).toBe(4);expect(dirty).toHaveBeenCalledWith(true);
});
test('metadata Security searches and pages permission resources through MCP',async()=>{
 const user=userEvent.setup(),{api}=mount();await screen.findByText('Forecasting guide');await user.click(screen.getByRole('button',{name:'Next resources',exact:true}));await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source,input])=>source.backend.method==='studio.sdk.access.list'&&input.offset===10)).toBe(true));
 await user.type(screen.getByRole('textbox',{name:'Find a resource',exact:true}),'guide');await user.click(screen.getByRole('button',{name:'Search resources',exact:true}));await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source,input])=>source.backend.method==='studio.sdk.access.list'&&input.offset===0&&input.query==='guide')).toBe(true));
});
test('metadata Security shows resources without an ACL without inventing a policy',async()=>{
 const user=userEvent.setup(),{api}=mount({items:[{...entry,inherited:false,public:true,explicitAcl:false}]});await user.click(await screen.findByRole('button',{name:'Open permissions',exact:true}));expect(await screen.findByText('No explicit ACL is defined for this resource.')).toBeTruthy();expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='authz.sdk.policies.get')).toBe(false);
});
test('metadata Security loads the chosen saved resource version',async()=>{
 const previous={...resource,version:'7'},user=userEvent.setup(),{api}=mount({items:[{...entry,versions:[{resource,published:true,explicitAcl:true},{resource:previous,published:false,explicitAcl:true}]}]});await user.click(await screen.findByRole('button',{name:'Choose resource version',exact:true}));const dialog=await screen.findByRole('dialog',{name:'Resource versions'});const row=within(dialog).getByText('7',{exact:true}).closest('tr');await user.click(within(row).getByRole('button',{name:'Open this version permissions',exact:true}));await screen.findByText('Policy revision 4');expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='authz.sdk.policies.get')[1]).toEqual({resource:previous});
});
test('metadata Security refuses to infer an action contract for an unknown resource kind',async()=>{
 const user=userEvent.setup(),{api}=mount({items:[{...entry,resource:{...resource,kind:'dataset'}}]});await user.click(await screen.findByRole('button',{name:'Open permissions',exact:true}));expect(await screen.findByText(/Configure the permission actions/)).toBeTruthy();expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='authz.sdk.policies.get')).toBe(false);
});
test('metadata Security can use a different declared resource action contract',async()=>{
 const user=userEvent.setup();mount({items:[{...entry,resource:{...resource,kind:'dataset'}}],kindContract:{dataset:['discover','execute']}});await user.click(await screen.findByRole('button',{name:'Open permissions',exact:true}));await screen.findByText('Policy revision 4');expect(screen.getByRole('button',{name:/^execute Protected/})).toBeTruthy();expect(screen.queryByRole('button',{name:/^edit /})).toBeNull();
});
test('metadata authorization predicates use complete discovered Go identities and preserve CAS updates',async()=>{
 const user=userEvent.setup(),{api}=mount();await screen.findByRole('button',{name:'Open permissions',exact:true});await user.click(screen.getByRole('tab',{name:'Authorization predicates',exact:true}));await user.click(await screen.findByRole('button',{name:'Discovered predicates',exact:true}));const discovery=await screen.findByRole('dialog',{name:'Discovered predicates'});expect(within(discovery).getByText(types[0].packagePath,{exact:true})).toBeTruthy();expect(within(discovery).getByText(types[1].packagePath,{exact:true})).toBeTruthy();await user.click(within(discovery).getAllByRole('button',{name:'Close',exact:true}).at(-1));
 await user.click(screen.getByRole('button',{name:'Edit authorization predicate',exact:true}));await user.type(await screen.findByLabelText('Display name',{exact:true}),' updated');await user.click(screen.getByRole('button',{name:'Save predicate',exact:true}));await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.authorization_predicates.update')).toBe(true));expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.authorization_predicates.update')[1]).toEqual({name:predicate.name,input:{title:'Vendor scope updated',description:'',packagePath:predicate.packagePath,typeName:predicate.typeName,alias:'v',columns:['id'],status:'active',etag:7}});
});

test('metadata Security creates from a discovered absolute handler and confirms predicate deletion',async()=>{
 const user=userEvent.setup(),{api}=mount();await screen.findByRole('button',{name:'Open permissions',exact:true});await user.click(screen.getByRole('tab',{name:'Authorization predicates',exact:true}));await user.click(await screen.findByRole('button',{name:'New predicate',exact:true}));
 await user.type(await screen.findByLabelText('Canonical name',{exact:true}),'iam.customer.read');await user.type(screen.getByLabelText('Display name',{exact:true}),'Customer scope');await user.click(screen.getByRole('button',{name:'Linked handler',exact:true}));await user.click(await screen.findByRole('menuitem',{name:'example.com/second/predicates · CustomerScope',exact:true}));await user.type(screen.getByLabelText('Alias',{exact:true}),'c');await user.type(screen.getByLabelText('Columns',{exact:true}),'id, tenant');await user.click(screen.getByRole('button',{name:'Save predicate',exact:true}));
 await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.authorization_predicates.create')).toBe(true));expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.authorization_predicates.create')[1]).toEqual({name:'iam.customer.read',title:'Customer scope',description:'',packagePath:'example.com/second/predicates',typeName:'CustomerScope',alias:'c',columns:['id','tenant']});
 await user.click(screen.getByRole('button',{name:'Delete authorization predicate',exact:true}));expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.authorization_predicates.delete')).toBe(false);await user.click(await screen.findByRole('button',{name:'Delete predicate',exact:true}));await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.authorization_predicates.delete')).toBe(true));expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.authorization_predicates.delete')[1]).toEqual({name:predicate.name,etag:7});
});
test('metadata Security filters the discovered handler lookup without losing complete identities',async()=>{
 const user=userEvent.setup();mount();await screen.findByRole('button',{name:'Open permissions',exact:true});await user.click(screen.getByRole('tab',{name:'Authorization predicates',exact:true}));await user.click(await screen.findByRole('button',{name:'Discovered predicates',exact:true}));const dialog=await screen.findByRole('dialog',{name:'Discovered predicates'});await user.type(within(dialog).getByLabelText('Search discovered predicates',{exact:true}),'visibility');await waitFor(()=>expect(within(dialog).queryByText(types[1].packagePath,{exact:true})).toBeNull());expect(within(dialog).getByText(types[0].packagePath,{exact:true})).toBeTruthy();
});
test('metadata Security requires explicit discard before closing unsaved policy changes',async()=>{
 const user=userEvent.setup(),{dirty}=mount();await user.click(await screen.findByRole('button',{name:'Open permissions',exact:true}));await screen.findByText('Policy revision 4');await user.click(screen.getByRole('button',{name:/^execute Protected/}));await user.selectOptions(screen.getByLabelText('rule value'),'analyst');await user.click(screen.getByRole('button',{name:'Close permissions',exact:true}));
 await user.click(await screen.findByRole('button',{name:'Keep editing',exact:true}));expect(screen.getByLabelText('rule value').value).toBe('analyst');await user.click(screen.getByRole('button',{name:'Close permissions',exact:true}));await user.click(await screen.findByRole('button',{name:'Discard changes',exact:true}));await waitFor(()=>expect(screen.queryByText('Policy revision 4')).toBeNull());expect(dirty).toHaveBeenLastCalledWith(false);
});

test('embedded permissions metadata preserves host kinds and custom action contracts',async()=>{
 const user=userEvent.setup();const custom={kind:'dashboard',id:'dashboard-a',tenant:'tenant-a',version:'2'};
 const {api}=mount({items:[{id:'dashboard-a',name:'Sales dashboard',kind:'dashboard',resource:custom,explicitAcl:true}],parameters:{permissionsOnly:true,kinds:['dashboard'],actionsByKind:{dashboard:['inspect','viewAccess']}}});
 await screen.findByRole('heading',{name:'Permissions',exact:true});
 expect(screen.queryByRole('tab',{name:'Authorization predicates',exact:true})).toBeNull();
 await user.click(screen.getByRole('button',{name:'Resource type',exact:true}));
 expect(await screen.findByRole('menuitem',{name:'dashboard',exact:true})).toBeTruthy();
 expect(screen.queryByRole('menuitem',{name:'Components',exact:true})).toBeNull();
 await user.click(screen.getByRole('menuitem',{name:'dashboard',exact:true}));
 await user.click(screen.getByRole('button',{name:'Open permissions',exact:true}));
 await screen.findByText('Policy revision 4');
 expect(screen.getByRole('button',{name:/^inspect /})).toBeTruthy();
 expect(screen.queryByRole('button',{name:/^publish /})).toBeNull();
 expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='authz.sdk.policies.get')[1]).toEqual({resource:custom});
});
test('namespace default opens first and saves one policy for all tools',async()=>{
 const user=userEvent.setup(),{api}=mount({items:[],parameters:{currentNamespace:{namespaceId:'ns-a',name:'analytics',canManage:true}}});
 await screen.findByText('Namespace default permissions');
 expect(screen.getByRole('button',{name:'Edit namespace default',exact:true})).toBeTruthy();
 expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='authz.sdk.policies.get')[1]).toEqual({resource:{kind:'namespace',id:'ns-a',tenant:'',version:'1'}});
 await screen.findByText('Policy revision 4');
 await user.click(screen.getByRole('button',{name:/^execute Protected/}));
 await user.selectOptions(screen.getByLabelText('rule value'),'analyst');
 await user.click(screen.getByRole('button',{name:'Apply to exposure and execution',exact:true}));
 await user.click(screen.getByRole('button',{name:'Review changes',exact:true}));
 await user.click(screen.getByRole('button',{name:'Save permissions',exact:true}));
 await screen.findByText('Permissions saved.');
 const calls=api.fetchMCPDataSource.mock.calls.filter(([source])=>source.backend.method==='authz.sdk.policies.replace');
 expect(calls).toHaveLength(1);
 expect(calls[0][1].document.resource.kind).toBe('namespace');
 for(const action of ['discover','describe','execute'])expect(calls[0][1].document.policies[action].rule.value).toBe('analyst');
 expect(screen.queryByRole('button',{name:/^manageAccess /})).toBeNull();
});
test('tools without an override show namespace inheritance instead of public access',async()=>{
 const user=userEvent.setup(),{api}=mount({items:[{...entry,kind:'component',inherited:false,public:false,explicitAcl:false,defaultResource:{kind:'namespace',id:'ns-a',tenant:'tenant-a',version:'1'}}]});
 await user.click(await screen.findByRole('button',{name:'Open permissions',exact:true}));
 expect(await screen.findByText('Uses the namespace default policy. An optional tool override takes precedence.')).toBeTruthy();
 expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='authz.sdk.policies.get')).toBe(false);
});
