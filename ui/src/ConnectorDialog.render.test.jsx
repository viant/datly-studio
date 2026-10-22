import React from 'react';
import {test,expect,vi} from 'vitest';
import {render,screen,waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ManagementWindowRuntime from './ManagementWindowRuntime.jsx';
import {managementResourceFixture} from './managementWindowFixture.js';
let serial=0;
const saved={name:'warehouse',driver:'mysql',etag:4,status:'draft',description:'',dsnDisplay:'${Username}:${Password}@tcp(db:3306)/reporting',secretRefDisplay:'file:///config/old.enc|blowfish://env/OLD',options:{}};
function mount(){const api={readManagementResource:async()=>managementResourceFixture('connectors'),fetchMCPDataSource:vi.fn(async(source,input)=>source.backend.method==='studio.sdk.connectors.list'?{items:[saved]}:source.backend.method==='studio.sdk.connectors.get'?saved:{...saved,etag:5})};render(<ManagementWindowRuntime api={api} uri="window://datastudio/connectors" instanceKey={'connector-dialog-'+(++serial)}/>);return api;}
async function edit(user){await user.click(await screen.findByRole('button',{name:'Edit',exact:true}));await screen.findByLabelText('Description',{exact:true});}
test('metadata connector saves unrelated edits without reusing displayed secrets or DSN',async()=>{const user=userEvent.setup(),api=mount();await edit(user);expect(screen.getByLabelText('DSN',{exact:true}).value).toBe(saved.dsnDisplay);expect(screen.getByLabelText('DSN',{exact:true}).readOnly).toBe(false);expect(screen.getByLabelText('Secret reference',{exact:true}).value).toBe(saved.secretRefDisplay);await user.type(screen.getByLabelText('Description',{exact:true}),'Updated');await user.click(screen.getByRole('button',{name:'Save',exact:true}));await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.connectors.update')).toBe(true));const input=api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.connectors.update')[1];expect(input).toMatchObject({name:'warehouse',input:{description:'Updated',etag:4}});expect(input.input).not.toHaveProperty('dsnTemplate');expect(input.input).not.toHaveProperty('secretRef');});
test('metadata connector replaces DSN and SCY reference only when explicitly entered',async()=>{const user=userEvent.setup(),api=mount();await edit(user);await user.clear(screen.getByLabelText('DSN',{exact:true}));await user.type(screen.getByLabelText('DSN',{exact:true}),'mysql://new');await user.type(screen.getByLabelText('Secret URL',{exact:true}),'file:///config/new.enc');await user.type(screen.getByLabelText('Secret key',{exact:true}),'blowfish://default');await user.click(screen.getByRole('button',{name:'Save',exact:true}));await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.connectors.update')).toBe(true));expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.connectors.update')[1].input).toMatchObject({dsnTemplate:'mysql://new',secretRef:'file:///config/new.enc|blowfish://default'});});
test('metadata connector rejects a replacement key without URL and preserves the entered draft',async()=>{const user=userEvent.setup(),api=mount();await edit(user);await user.type(screen.getByLabelText('Secret key',{exact:true}),'blowfish://default');await user.click(screen.getByRole('button',{name:'Save',exact:true}));await waitFor(()=>expect(screen.getByLabelText('Outcome',{exact:true}).value).toContain('secret URL'));expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.connectors.update')).toBe(false);expect(screen.getByLabelText('Secret key',{exact:true}).value).toBe('blowfish://default');});

test('connector lookup entries save independently of the editable driver',async()=>{
 const user=userEvent.setup(),api=mount();await edit(user);
 await user.click(screen.getByRole('button',{name:'Add project / catalog',exact:true}));
 await user.type(screen.getByRole('textbox',{name:'Project / catalog 1',exact:true}),'my-project');
 await user.click(screen.getByRole('button',{name:'Add schema / dataset',exact:true}));
 await user.type(screen.getByRole('textbox',{name:'Project / catalog (optional) 1',exact:true}),'my-project');
 await user.type(screen.getByRole('textbox',{name:'Schema / dataset 1',exact:true}),'analytics');
 await user.click(screen.getByRole('button',{name:'Save',exact:true}));
 await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.connectors.update')).toBe(true));
 const input=api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.connectors.update')[1].input;
 expect(input.options.discovery).toEqual({catalogs:['my-project'],schemas:[{catalog:'my-project',name:'analytics'}]});
 expect(input).not.toHaveProperty('dsnTemplate');expect(input).not.toHaveProperty('secretRef');
});
test('connector lifecycle button shows green On and red Off after server refresh',async()=>{
 const user=userEvent.setup();let status='active',etag=4;
 const api={readManagementResource:async()=>managementResourceFixture('connectors'),fetchMCPDataSource:vi.fn(async(source,input)=>{
  if(source.backend.method==='studio.sdk.connectors.list')return {items:[{...saved,status,etag}]};
  if(source.backend.method==='studio.sdk.connectors.disable'){status='disabled';etag++;return {...saved,status,etag};}
  if(source.backend.method==='studio.sdk.connectors.activate'){status='active';etag++;return {...saved,status,etag};}
  throw new Error('Unexpected '+source.backend.method);
 })};
 render(<ManagementWindowRuntime api={api} uri="window://datastudio/connectors" instanceKey={'connector-state-'+(++serial)}/>);
 const toggle=await screen.findByRole('button',{name:'Activate or disable',exact:true});
 expect(toggle.getAttribute('aria-pressed')).toBe('true');expect(toggle.textContent).toContain('On');expect(toggle.className).toContain('intent-success');
 await user.click(toggle);
 await waitFor(()=>expect(screen.getByRole('button',{name:'Activate or disable',exact:true}).textContent).toContain('Off'));
 expect(screen.getByRole('button',{name:'Activate or disable',exact:true}).className).toContain('intent-danger');expect(screen.getByRole('button',{name:'Activate or disable',exact:true}).getAttribute('aria-pressed')).toBe('false');
 await user.click(screen.getByRole('button',{name:'Activate or disable',exact:true}));
 await waitFor(()=>expect(screen.getByRole('button',{name:'Activate or disable',exact:true}).textContent).toContain('On'));
 expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.connectors.activate')[1]).toEqual({name:saved.name,etag:5});
});

test('Save & test probes the newly saved configuration and keeps detailed errors in the dialog',async()=>{
 const user=userEvent.setup();let current={...saved};
 const api={readManagementResource:async()=>managementResourceFixture('connectors'),fetchMCPDataSource:vi.fn(async(source,input)=>{
  switch(source.backend.method){
   case 'studio.sdk.connectors.list':return {items:[current]};
   case 'studio.sdk.connectors.get':return current;
   case 'studio.sdk.connectors.update':current={...current,...input.input,dsnDisplay:input.input.dsnTemplate??current.dsnDisplay,etag:5};return current;
   case 'studio.sdk.connectors.test':current={...current,lastTestStatus:'failed',etag:6};return {name:current.name,status:'failed',errorCode:'secret_not_found',message:'Connector secret file was not found on this server. Check the secret URL and deployment volume mount.',testedAt:'2026-10-22T17:00:00Z'};
   default:throw new Error('Unexpected '+source.backend.method);
  }
 })};
 render(<ManagementWindowRuntime api={api} uri="window://datastudio/connectors" instanceKey={'connector-probe-'+(++serial)}/>);
 await edit(user);
 await user.clear(screen.getByLabelText('DSN',{exact:true}));await user.type(screen.getByLabelText('DSN',{exact:true}),'new-dsn');
 await user.click(screen.getByRole('button',{name:'Save & test',exact:true}));
 await waitFor(()=>expect(screen.getByLabelText('Outcome',{exact:true}).value).toContain('[secret_not_found]'));
 expect(screen.getByLabelText('Outcome',{exact:true}).value).toContain('deployment volume mount');
 expect(screen.getByRole('button',{name:'Save & test',exact:true})).toBeTruthy();
 expect(screen.getByLabelText('DSN',{exact:true}).value).toBe('new-dsn');
 const calls=api.fetchMCPDataSource.mock.calls;
 const update=calls.findIndex(([source])=>source.backend.method==='studio.sdk.connectors.update');
 const probe=calls.findIndex(([source])=>source.backend.method==='studio.sdk.connectors.test');
 expect(update).toBeGreaterThan(-1);expect(probe).toBeGreaterThan(update);
 expect(calls[update][1].input.dsnTemplate).toBe('new-dsn');
 expect(calls[probe][1]).toEqual({name:'warehouse'});
 await user.click(screen.getByRole('button',{name:'Save',exact:true}));
 await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.filter(([source])=>source.backend.method==='studio.sdk.connectors.update').at(-1)[1].input.etag).toBe(6));
});


test('saved secret reference can be edited directly and saved with its key',async()=>{
 const user=userEvent.setup(),api=mount();await edit(user);
 const reference=screen.getByLabelText('Secret reference',{exact:true});
 expect(reference.readOnly).toBe(false);
 await user.clear(reference);await user.type(reference,'/opt/studio/secrets/steward-mysql.enc|blowfish://default');
 await user.click(screen.getByRole('button',{name:'Save',exact:true}));
 await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.connectors.update')).toBe(true));
 expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.connectors.update')[1].input).toMatchObject({secretRef:'/opt/studio/secrets/steward-mysql.enc|blowfish://default',etag:4});
});

test('clearing the editable secret reference removes it explicitly',async()=>{
 const user=userEvent.setup(),api=mount();await edit(user);
 await user.clear(screen.getByLabelText('Secret reference',{exact:true}));
 await user.click(screen.getByRole('button',{name:'Save',exact:true}));
 await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.connectors.update')).toBe(true));
 expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.connectors.update')[1].input.secretRef).toBe('');
});
