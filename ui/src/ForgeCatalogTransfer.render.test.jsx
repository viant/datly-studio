import React from 'react';
import {test,expect,vi} from 'vitest';
import {render,screen,waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import WindowContent from 'forge/window';
import {SettingProvider} from 'forge/settings';
import 'forge/packs/blueprint';
import {managementWindowFixture} from './managementWindowFixture.js';
let serial=0;
function mount({paging=false}={}){
 const navigate=vi.fn(),download=vi.fn(),dirty=vi.fn();
 const transport=vi.fn(async request=>{
  const id=request.dataSourceRef,input=request.options.inputParameters||{};
  if(id==='catalog_list')return {items:paging?Array.from({length:input.offset?1:26},(_,index)=>({id:'r'+(index+(input.offset||0)),title:'Component '+(index+(input.offset||0)),currentDraftVersion:3,status:'draft'})):[{id:'r1',title:'Vendor Catalog',currentDraftVersion:3,status:'draft'}]};
  if(id==='catalog_namespace_list')return {items:[{name:'general',title:'General'}]};
  if(id==='catalog_download')return {archive:[1,2,3],filename:'vendor.zip',mediaType:'application/zip'};
  if(id==='catalog_load_dql')return {version:{versionNo:4}};
  return {data:[]};
 });
 render(<SettingProvider connectorConfig={{window:{service:{uri:'/window'}}}} services={{dataConnectorTransport:transport,hostWindow:{parameters:{},navigate,download,setDirty:dirty}}}><WindowContent window={{windowId:'forge-catalog-transfer-'+(++serial),windowKey:'component-catalog'}} selectedMetadata={managementWindowFixture('component-catalog')}/></SettingProvider>);
 return {transport,navigate,download,dirty};
}
test('metadata catalog downloads the row exact draft through MCP and the host download capability',async()=>{
 const user=userEvent.setup(),{download,transport}=mount();await screen.findByText('Vendor Catalog');
 await user.click(screen.getByRole('button',{name:'Download component',exact:true}));
 await waitFor(()=>expect(download).toHaveBeenCalledWith(expect.objectContaining({filename:'vendor.zip',archive:[1,2,3]})));
 expect(transport.mock.calls.find(([request])=>request.dataSourceRef==='catalog_download')[0].options.inputParameters).toEqual({reportId:'r1',versionNo:3});
});
test('metadata import uploads UTF-8 DQL through the file widget and opens the returned draft',async()=>{
 const user=userEvent.setup(),{transport,navigate}=mount();await screen.findByText('Vendor Catalog');
 await user.click(screen.getByRole('button',{name:'Import component',exact:true}));
 await user.click(await screen.findByRole('button',{name:'Component',exact:true}));await user.click(await screen.findByRole('menuitem',{name:'Vendor Catalog',exact:true}));
 const source='SELECT 1 AS id -- café';const file=new File([source],'vendor.dql',{type:'text/plain'});file.arrayBuffer=async()=>new TextEncoder().encode(source).buffer;await user.upload(screen.getByLabelText('DQL or archive',{exact:true}),file);
 await user.click(screen.getByRole('button',{name:'Import as new draft',exact:true}));
 await waitFor(()=>expect(navigate).toHaveBeenCalledWith('builder',{report:expect.objectContaining({id:'r1',versionNo:4,currentDraftVersion:4})}));
 expect(transport.mock.calls.find(([request])=>request.dataSourceRef==='catalog_load_dql')[0].options.inputParameters).toEqual({reportId:'r1',input:{dql:source}});
});
test('metadata catalog sends server paging and query parameters',async()=>{
 const user=userEvent.setup(),{transport}=mount({paging:true});await screen.findByText('Component 0');
 await user.click(await screen.findByRole('button',{name:'Next',exact:true}));expect(await screen.findByText('Component 25')).toBeTruthy();expect(screen.queryByText('Component 0')).toBeNull();
 await user.type(screen.getByRole('textbox',{name:'Search components',exact:true}),'vendor');await user.click(screen.getByRole('button',{name:'Search components',exact:true}));
 await waitFor(()=>expect(transport.mock.calls.some(([request])=>request.dataSourceRef==='catalog_list'&&request.options.inputParameters.query==='vendor'&&request.options.inputParameters.offset===0&&request.options.inputParameters.limit===26)).toBe(true));
});
