import React from 'react';
import {test,expect,vi} from 'vitest';
import {render,screen,waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ManagementWindowRuntime from './ManagementWindowRuntime.jsx';
import {managementResourceFixture} from './managementWindowFixture.js';
let serial=0;
const event={id:10,toolName:'example.read',subject:'caller',status:'OK',durationMs:4,startedAt:'2026-10-21T10:00:00Z'};
function mount({canView=true,canManage=false}={}){
 const api={readManagementResource:async()=>managementResourceFixture('audit'),fetchMCPDataSource:vi.fn(async(source,input)=>{
  if(source.backend.method==='studio.sdk.audit.status')return {enabled:true,canView,canManage,viewerRole:'E2E_AUDIT_VIEWER'};
  if(source.backend.method==='studio.sdk.audit.export')return {filename:'audit.csv',mediaType:'text/csv',archive:'aWQKMQo=',count:1};
  if(source.backend.method==='studio.sdk.audit.configure')return {enabled:input.enabled,canView,canManage};
  return {items:[{...event,...(input?.filter?.id?{request:'{"input":1}',response:'{"output":2}',responseTruncated:true}:{})}],nextId:0};
 })};
 render(<ManagementWindowRuntime api={api} uri="window://datastudio/audit" instanceKey={'audit-'+(++serial)}/>);return api;
}
test('audit viewer can filter and inspect capped event without changing capture',async()=>{
 const user=userEvent.setup(),api=mount();await screen.findByText('example.read');
 expect(screen.getByLabelText('Namespace audit enabled').disabled).toBe(true);
 await user.type(screen.getByLabelText('Tool name'),'example.read');await user.click(screen.getByRole('button',{name:'Filter calls'}));
 await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source,input])=>source.backend.method==='studio.sdk.audit.list'&&input.filter?.toolName==='example.read')).toBe(true));
 await user.click(screen.getByRole('button',{name:'Inspect audit event'}));await screen.findByLabelText('Response');
 expect(screen.getByLabelText('Response').value).toBe('{"output":2}');expect(screen.getByLabelText('Capture').value).toContain('truncated');
});
test('namespace manager can toggle without receiving audit payloads',async()=>{
 const user=userEvent.setup(),api=mount({canView:false,canManage:true});await screen.findByText('Reading audit events requires the E2E_AUDIT_VIEWER role.');
 expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.audit.list')).toBe(false);
 await user.click(screen.getByLabelText('Namespace audit enabled'));await user.click(screen.getByRole('button',{name:'Save audit setting'}));
 await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.audit.configure')?.[1]).toMatchObject({enabled:false}));
});

test('CSV export sends the active filters to the backend download tool',async()=>{
 const oldCreate=URL.createObjectURL,oldRevoke=URL.revokeObjectURL;
 URL.createObjectURL=vi.fn(()=> 'blob:audit-test');URL.revokeObjectURL=vi.fn();
 const click=vi.spyOn(HTMLAnchorElement.prototype,'click').mockImplementation(()=>{});
 try{
  const user=userEvent.setup(),api=mount();await screen.findByText('example.read');
  await user.type(screen.getByLabelText('Tool name'),'example.read');
  await user.type(screen.getByLabelText('Caller subject'),'caller');
  await user.click(screen.getByRole('button',{name:'Export CSV'}));
  await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.audit.export')?.[1]).toMatchObject({filter:{toolName:'example.read',subject:'caller'}}));
  await screen.findByText('Exported 1 audit events.');expect(click).toHaveBeenCalled();
 }finally{click.mockRestore();URL.createObjectURL=oldCreate;URL.revokeObjectURL=oldRevoke;}
});
