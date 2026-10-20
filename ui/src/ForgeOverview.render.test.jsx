import React from 'react';
import {test,expect,vi} from 'vitest';
import {render,screen,waitFor,within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import WindowContent from 'forge/window';
import {SettingProvider} from 'forge/settings';
import 'forge/packs/blueprint';
import {managementWindowFixture} from './managementWindowFixture.js';
let serial=0;
function mount(partial=false){
 const navigate=vi.fn();
 const transport=vi.fn(async request=>{
  if(request.dataSourceRef==='overview_reports')return {items:[{id:'r1',title:'Revenue',status:'draft',defaultConnectorName:'main'}]};
  if(request.dataSourceRef==='overview_connectors'){if(partial)throw new Error('Connector service unavailable');return {items:[{name:'main',status:'active',lastTestStatus:'passed'}]};}
  if(request.dataSourceRef==='overview_namespaces')return {items:[{name:'general'}]};
  if(request.dataSourceRef==='overview_runtime')return {data:[{status:'active',host:{status:'ready',revision:12}}]};
  return {data:[]};
 });
 render(<SettingProvider connectorConfig={{window:{service:{uri:'/window'}}}} services={{dataConnectorTransport:transport,hostWindow:{navigate}}}><WindowContent window={{windowId:`overview-${++serial}`,windowKey:'overview'}} selectedMetadata={managementWindowFixture('overview')}/></SettingProvider>);
 return {transport,navigate};
}
test('Forge overview loads MCP-backed summaries and opens the selected component',async()=>{
 const user=userEvent.setup(),{transport,navigate}=mount();
 await screen.findByText('Revenue');await screen.findByText('Ready · revision 12');
 expect(screen.getByText('1/1 active connectors tested')).toBeTruthy();
 await user.click(within(screen.getByRole('region',{name:'Continue authoring'})).getByRole('button',{name:/Revenue/}));
 expect(screen.queryByRole('table')).toBeNull();
 expect(screen.queryByLabelText('Outcome')).toBeNull();
 expect(screen.getByRole('button',{name:'Refresh',exact:true}).textContent).toBe('Refresh');
 expect(navigate).toHaveBeenCalledWith('builder',{report:expect.objectContaining({id:'r1',title:'Revenue'})});
 expect(transport.mock.calls.filter(([request])=>request.dataSourceRef==='overview_reports')[0][0].options.inputParameters).toMatchObject({limit:6,offset:0});
});
test('Forge overview preserves healthy sections when one MCP datasource fails',async()=>{
 mount(true);await screen.findByText('Revenue');
 await waitFor(()=>expect(screen.getByText('Connector readiness is unavailable')).toBeTruthy());
 expect(screen.getByText('Ready · revision 12')).toBeTruthy();
});
