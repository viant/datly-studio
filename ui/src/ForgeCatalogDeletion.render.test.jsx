import React from 'react';
import {test,expect,vi} from 'vitest';
import {render,screen,waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import WindowContent from 'forge/window';
import {SettingProvider} from 'forge/settings';
import 'forge/packs/blueprint';
import {managementWindowFixture} from './managementWindowFixture.js';
let serial=0;
function mount({subject='owner',remove=async()=>({componentId:'component-a',deletedVersions:1}),refreshFails=false}={}) {
 let deleted=false;
 const transport=vi.fn(async request=>{
  if(request.dataSourceRef==='catalog_list'){if(deleted&&refreshFails)throw new Error('Catalog unavailable');return {items:deleted?[]:[{id:'component-a',title:'Synthetic reader',ownerId:'owner',etag:3,status:'draft'}]};}
  if(request.dataSourceRef==='catalog_delete_component'){const result=await remove();deleted=true;return result;}
  return {items:[]};
 });
 render(<SettingProvider connectorConfig={{window:{service:{uri:'/window'}}}} services={{dataConnectorTransport:transport,hostWindow:{parameters:{subject},setDirty:vi.fn()}}}><WindowContent window={{windowId:'deletion-'+(++serial),windowKey:'component-catalog',parameters:{subject}}} selectedMetadata={managementWindowFixture('component-catalog')}/></SettingProvider>);
 return transport;
}
const requests=transport=>transport.mock.calls.filter(([r])=>r.dataSourceRef==='catalog_delete_component');
test('catalog confirms the displayed component, cancels without mutation, submits one exact request, and refreshes',async()=>{
 const user=userEvent.setup();let finish;const transport=mount({remove:()=>new Promise(resolve=>{finish=resolve;})});
 const action=await screen.findByRole('button',{name:'Permanently delete component'});await user.click(action);
 expect(screen.getByDisplayValue('Synthetic reader')).toBeTruthy();expect(screen.getByDisplayValue('component-a')).toBeTruthy();
 expect(screen.getByDisplayValue(/Retained publication and audit history survives/)).toBeTruthy();
 await user.click(screen.getByRole('button',{name:'Cancel',exact:true}));expect(requests(transport)).toHaveLength(0);
 await user.click(action);const confirm=screen.getAllByRole('button',{name:'Permanently delete component'}).at(-1);await user.dblClick(confirm);
 expect(requests(transport)).toHaveLength(1);expect(requests(transport)[0][0].options.inputParameters).toEqual({id:'component-a',input:{etag:3}});
 expect(screen.getByRole('button',{name:'Cancel',exact:true}).disabled).toBe(true);
 finish({componentId:'component-a',deletedVersions:1});await screen.findByText('No components yet');
 await waitFor(()=>expect(screen.queryByRole('dialog')).toBeNull());
});
test.each(['delegate',''])('catalog omits deletion for non-owner subject %s',async subject=>{
 mount({subject});await screen.findByText('Synthetic reader');expect(screen.queryByRole('button',{name:'Permanently delete component'})).toBeNull();
});
test('backend conflict remains meaningful and leaves the catalog intact',async()=>{
 const user=userEvent.setup(),transport=mount({remove:async()=>{throw new Error('Retained release binding prevents deletion');}});
 await user.click(await screen.findByRole('button',{name:'Permanently delete component'}));await user.click(screen.getAllByRole('button',{name:'Permanently delete component'}).at(-1));
 await screen.findByDisplayValue('Retained release binding prevents deletion');expect(requests(transport)).toHaveLength(1);expect(screen.queryByText('Reader changed elsewhere')).toBeNull();
});
test('refresh failure after deletion cannot submit deletion again',async()=>{
 const user=userEvent.setup(),transport=mount({refreshFails:true});await user.click(await screen.findByRole('button',{name:'Permanently delete component'}));await user.click(screen.getAllByRole('button',{name:'Permanently delete component'}).at(-1));
 await screen.findByDisplayValue('Component deleted. Catalog refresh failed: Catalog unavailable');expect(screen.getAllByRole('button',{name:'Permanently delete component'}).at(-1).disabled).toBe(true);expect(requests(transport)).toHaveLength(1);
});

test('retained catalog refreshes after deleting the inspected last version through the host and returns without selecting a draft',async()=>{
 const {StudioApp}=await import('./StudioApp.jsx');
 const {managementResourceFixture}=await import('./managementWindowFixture.js');
 let deleted=false;
 const component={id:'component-a',title:'Synthetic reader',ownerId:'owner',etag:3,currentDraftVersion:7};
 const api={readManagementResource:vi.fn(async uri=>managementResourceFixture(uri.split('/').at(-1))),invoke:vi.fn(async(operation,input)=>{
  if(operation==='components.list')return {items:[{...component,currentDraftVersion:deleted?0:7}]};
  if(operation==='namespaces.list'||operation==='connectors.list')return {items:[]};
  if(operation==='versions.inspect')return {version:{versionNo:7,sourceRevision:14,state:'draft'},dql:'SELECT 1',structure:{component:{views:[]}},capabilities:{}};
  if(operation==='versions.list')return {items:[]};
  if(operation==='versions.delete'){deleted=true;return {componentId:input.reportId,versionNo:input.versionNo,deletedVersions:1};}
  if(operation==='runtime.status')return {};
  throw new Error('Unexpected operation '+operation);
 })};
 window.history.replaceState({},'','/?section=components');
 const user=userEvent.setup();
 try{
  render(<StudioApp api={api} subject="owner" mode="development"/>);
  await user.click(await screen.findByRole('button',{name:'Open Reader Builder',exact:true}));
  await user.click(await screen.findByRole('button',{name:'Delete version v7',exact:true}));
  await user.click(screen.getByRole('button',{name:'Permanently delete version',exact:true}));
  await waitFor(()=>expect(api.invoke).toHaveBeenCalledWith('versions.delete',{reportId:'component-a',versionNo:7,expectedSourceRevision:14}));
  await screen.findByRole('button',{name:'Open Reader Builder',exact:true});
  await waitFor(()=>expect(api.invoke.mock.calls.filter(([operation])=>operation==='components.list').length).toBeGreaterThan(1));
  expect(api.invoke.mock.calls.filter(([operation])=>operation==='versions.inspect')).toHaveLength(1);
  expect(api.invoke.mock.calls.filter(([operation])=>operation==='versions.list')).toHaveLength(0);
  expect(screen.getByText('Synthetic reader')).toBeTruthy();
  await user.click(screen.getByRole('button',{name:'Open Reader Builder',exact:true}));
  await screen.findByRole('heading',{name:'Root SQL',exact:true});
  expect(screen.queryByRole('button',{name:'Delete version v7',exact:true})).toBeNull();
  expect(api.invoke.mock.calls.filter(([operation])=>operation==='versions.list')).toHaveLength(1);
 }finally{window.history.replaceState({},'','/');}
});
