import React from 'react';
import {test,expect,vi} from 'vitest';
import {render,screen,waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ManagementWindowRuntime from './ManagementWindowRuntime.jsx';
import {managementResourceFixture} from './managementWindowFixture.js';
let serial=0;
const original={name:'inventory.forecasting',title:'Forecasting',description:'Planning',status:'active',etag:3,canManage:true,visibility:'private',allowedRoles:[],mcpEnabled:false,mcpPort:0};
function mount({saved=original,conflict=false,update,canCreate=true,settings={settings:{},etag:3,canEdit:true},saveSettings}={}){let reads=0;const api={readManagementResource:async()=>managementResourceFixture('namespaces'),fetchMCPDataSource:vi.fn(async(source,input)=>{const method=source.backend.method;if(method==='studio.sdk.namespaces.list')return {items:[saved],canCreate};if(method==='studio.sdk.namespaces.get')return ++reads>1?{...saved,title:'Authoritative title',etag:4,canManage:false,ownerId:'new-owner'}:saved;if(method==='studio.sdk.namespaces.settings_get')return settings;if(method==='studio.sdk.namespaces.settings_update')return saveSettings?saveSettings(input):{settings:input.input.settings,etag:input.input.etag+1};if(method==='studio.sdk.namespaces.update'){if(conflict)throw Object.assign(new Error('namespace etag does not match'),{code:'conflict'});return update?update(input):saved;}if(method==='studio.sdk.namespaces.create')throw new Error('namespace already exists');return {};})};render(<ManagementWindowRuntime api={api} uri="window://datastudio/namespaces" instanceKey={'namespace-dialog-'+(++serial)}/>);return api;}
async function edit(user){await user.click(await screen.findByRole('button',{name:'Edit namespace',exact:true}));return screen.findByRole('dialog',{name:'Namespace settings'});}
test('metadata namespace conflict preserves edits and reloads authoritative management capability',async()=>{const user=userEvent.setup(),api=mount({conflict:true});await edit(user);const title=screen.getByLabelText('Display title',{exact:true});await user.clear(title);await user.type(title,'Unsaved');await user.click(screen.getByRole('button',{name:'Save',exact:true}));await waitFor(()=>expect(screen.getByLabelText('Outcome',{exact:true}).value).toBe('namespace etag does not match'));expect(title.value).toBe('Unsaved');await user.click(screen.getByRole('button',{name:'Reload saved revision',exact:true}));await waitFor(()=>expect(title.value).toBe('Authoritative title'));expect(screen.queryByRole('button',{name:'Save',exact:true})).toBeNull();expect(title.readOnly).toBe(true);expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.namespaces.update')[1].input.etag).toBe(3);});
test('metadata namespace validates MCP port and retains rejected create fields',async()=>{const user=userEvent.setup();mount();await user.click(await screen.findByRole('button',{name:'New namespace',exact:true}));await user.type(screen.getByLabelText('Namespace',{exact:true}),'finance.reporting');await user.type(screen.getByLabelText('Display title',{exact:true}),'Finance');await user.click(screen.getByRole('button',{name:'Save',exact:true}));await waitFor(()=>expect(screen.getByLabelText('Outcome',{exact:true}).value).toBe('namespace already exists'));expect(screen.getByLabelText('Namespace',{exact:true}).value).toBe('finance.reporting');});
test('metadata namespace cannot be dismissed while its CAS save is pending',async()=>{let finish;const user=userEvent.setup();mount({update:()=>new Promise(resolve=>{finish=resolve;})});await edit(user);await user.click(screen.getByRole('button',{name:'Save',exact:true}));await waitFor(()=>expect(finish).toBeTypeOf('function'));await user.keyboard('{Escape}');expect(screen.getByRole('dialog',{name:'Namespace settings'})).toBeTruthy();expect(screen.getByRole('button',{name:'Cancel',exact:true}).disabled).toBe(true);finish(original);await waitFor(()=>expect(screen.queryByRole('dialog')).toBeNull());});
test('metadata namespace preserves host-owned immutable names and exact viewer roles',async()=>{const user=userEvent.setup(),api=mount({saved:{...original,name:'ui-component-policy-role'}});await edit(user);expect(screen.getByLabelText('Namespace',{exact:true}).readOnly).toBe(true);await user.click(screen.getByRole('button',{name:'Add viewer role',exact:true}));const roles=screen.getByRole('textbox',{name:'Viewer role 1',exact:true});await user.type(roles,'XYZ');await user.click(screen.getByRole('button',{name:'Save',exact:true}));await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.namespaces.update')).toBe(true));expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.namespaces.update')[1]).toMatchObject({name:'ui-component-policy-role',input:{etag:3,allowedRoles:['XYZ']}});});

 test('namespace manager roles are saved separately from viewer roles',async()=>{
 const user=userEvent.setup(),api=mount();await edit(user);
 await user.click(screen.getByRole('button',{name:'Add manager role',exact:true}));
 const managers=screen.getByRole('textbox',{name:'Manager role 1',exact:true});
 await user.type(managers,'TEAM_MANAGER');
 await user.click(screen.getByRole('button',{name:'Save',exact:true}));
 await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.namespaces.update')).toBe(true));
 expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.namespaces.update')[1]).toMatchObject({name:original.name,input:{managerRoles:['TEAM_MANAGER'],allowedRoles:[],etag:3}});
 });

test('role rows load, add, remove and save plain string arrays',async()=>{
 const user=userEvent.setup(),api=mount({saved:{...original,allowedRoles:['READER_A','READER_B'],managerRoles:['MANAGER_A']}});
 await edit(user);
 expect(screen.getByRole('textbox',{name:'Viewer role 1',exact:true}).value).toBe('READER_A');
 expect(screen.getByRole('textbox',{name:'Manager role 1',exact:true}).value).toBe('MANAGER_A');
 await user.click(screen.getByRole('button',{name:'Remove viewer role 1',exact:true}));
 await user.click(screen.getByRole('button',{name:'Remove manager role 1',exact:true}));
 await user.click(screen.getByRole('button',{name:'Add viewer role',exact:true}));
 await user.type(screen.getByRole('textbox',{name:'Viewer role 2',exact:true}),'READER_C');
 await user.click(screen.getByRole('button',{name:'Save',exact:true}));
 await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.namespaces.update')).toBe(true));
 expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.namespaces.update')[1].input).toMatchObject({allowedRoles:['READER_B','READER_C'],managerRoles:[]});
});
test('viewer cannot add, remove or edit role rows',async()=>{
 const user=userEvent.setup();mount({saved:{...original,canManage:false,allowedRoles:['READER'],managerRoles:['MANAGER']}});await edit(user);
 expect(screen.queryByRole('button',{name:'Add viewer role',exact:true})).toBeNull();
 expect(screen.queryByRole('button',{name:'Add manager role',exact:true})).toBeNull();
 expect(screen.queryByRole('button',{name:'Remove manager role 1',exact:true})).toBeNull();
 expect(screen.getByRole('textbox',{name:'Manager role 1',exact:true}).disabled).toBe(true);
});
test('empty role rows are explained without saving',async()=>{
 const user=userEvent.setup(),api=mount();await edit(user);
 await user.click(screen.getByRole('button',{name:'Add manager role',exact:true}));
 await user.click(screen.getByRole('button',{name:'Save',exact:true}));
 await waitFor(()=>expect(screen.getByLabelText('Outcome',{exact:true}).value).toBe('Enter each manager role or remove its empty row.'));
 expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.namespaces.update')).toBe(false);
});

test('namespace creation action follows server capability while existing management remains available',async()=>{
 const api=mount({canCreate:false});
 await waitFor(()=>expect(screen.getByRole('button',{name:'New namespace',exact:true}).disabled).toBe(true));
 expect(screen.getByRole('button',{name:'Edit namespace',exact:true})).toBeTruthy();
 expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.namespaces.create')).toBe(false);
});


test('namespace policies save typed thresholds using the namespace ETag',async()=>{
 const user=userEvent.setup(),api=mount({settings:{etag:7,canEdit:true,settings:{security:{predicatePackages:[],requireSecurityPredicate:false},resourceLimits:{maxBytesProcessed:100}}}});
 await user.click(await screen.findByRole('button',{name:'Security and resource limits',exact:true}));
 await screen.findByRole('dialog',{name:'Namespace security and resource limits'});
 const bytes=screen.getByLabelText('Maximum bytes processed (blank means unlimited)',{exact:true});
 await user.clear(bytes);await user.type(bytes,'200');
 await user.click(screen.getByRole('button',{name:'Save limits and security',exact:true}));
 await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.namespaces.settings_update')).toBe(true));
 const payload=api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.namespaces.settings_update')[1];
 expect(payload).toMatchObject({name:original.name,input:{etag:7,settings:{security:{predicatePackages:[],requireSecurityPredicate:false},resourceLimits:{maxBytesProcessed:200}}}});
 expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.namespaces.update')).toBe(false);
});

test('namespace policy conflict retains thresholds and keeps the dialog open',async()=>{
 const user=userEvent.setup();mount({saveSettings:()=>{throw new Error('namespace etag does not match');}});
 await user.click(await screen.findByRole('button',{name:'Security and resource limits',exact:true}));
 await screen.findByRole('dialog',{name:'Namespace security and resource limits'});
 await user.type(screen.getByLabelText('Maximum referenced tables (blank means unlimited)',{exact:true}),'4');
 await user.click(screen.getByRole('button',{name:'Save limits and security',exact:true}));
 await waitFor(()=>expect(screen.getByLabelText('Outcome',{exact:true}).value).toBe('namespace etag does not match'));
 expect(screen.getByRole('dialog',{name:'Namespace security and resource limits'})).toBeTruthy();
 expect(screen.getByLabelText('Maximum referenced tables (blank means unlimited)',{exact:true}).value).toBe('4');
});

test('namespace package list adds and removes full string values',async()=>{
 const user=userEvent.setup(),api=mount();await user.click(await screen.findByRole('button',{name:'Security and resource limits',exact:true}));await screen.findByRole('dialog',{name:'Namespace security and resource limits'});
 expect(screen.queryByRole('checkbox',{name:'Use host predicate packages'})).toBeNull();
 await user.click(screen.getByRole('button',{name:'Add package',exact:true}));await user.type(screen.getByRole('textbox',{name:'Package path 1',exact:true}),'example.com/app/security');
 await user.click(screen.getByRole('button',{name:'Add package',exact:true}));await user.type(screen.getByRole('textbox',{name:'Package path 2',exact:true}),'example.com/unused');
 await user.click(screen.getByRole('button',{name:'Remove package 2',exact:true}));
 await user.click(screen.getByRole('button',{name:'Save limits and security',exact:true}));
 await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.namespaces.settings_update')).toBe(true));
 expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.namespaces.settings_update')[1].input.settings.security.predicatePackages).toEqual(['example.com/app/security']);
});
test('ordinary namespace owner cannot mutate policy settings in the UI',async()=>{
 const user=userEvent.setup();mount({settings:{settings:{},etag:3,canEdit:false}});await user.click(await screen.findByRole('button',{name:'Security and resource limits',exact:true}));await screen.findByRole('dialog',{name:'Namespace security and resource limits'});
 expect(screen.queryByRole('button',{name:'Save limits and security',exact:true})).toBeNull();expect(screen.queryByRole('button',{name:'Add package',exact:true})).toBeNull();
});
