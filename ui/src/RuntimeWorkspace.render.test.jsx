import React from 'react';
import {expect,test,vi} from 'vitest';
import {render,screen,waitFor,within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ManagementWindowRuntime from './ManagementWindowRuntime.jsx';
import {managementResourceFixture} from './managementWindowFixture.js';
let serial=0;
const reader={reportId:'vendor',title:'Vendor Catalog',namespace:'general',ownerPackage:'alice',connectorName:'reporting',versionNo:2,status:'active',runtimeRevision:'rev12',mcpExposures:[{name:'alice.vendor.read'}],mcpResources:[{namespace:'alice.docs',rootPath:'guide',uriPrefix:'skill://alice-guide/'}]};
function mount({failedCatalog=false,failedStatus=false}={}){
 const navigate=vi.fn();const api={readManagementResource:async()=>managementResourceFixture('runtime'),fetchMCPDataSource:vi.fn(async(source)=>{if(failedStatus)throw new Error('Status denied');return {data:[{status:'active',activeGeneration:12,reportCount:1,host:{status:'ready',revision:12},readers:[reader]}]};}),listMCPTools:vi.fn(async()=>{if(failedCatalog)throw new Error('Discovery denied');return [{name:'alice.vendor.read',description:'Read vendors',inputSchema:{type:'object',required:['limit'],properties:{limit:{type:'integer',description:'Maximum rows'}}},outputSchema:{type:'object',properties:{vendors:{type:'array'}}}}];})};
 render(<ManagementWindowRuntime api={api} uri="window://datastudio/runtime" instanceKey={'forge-runtime-'+(++serial)} onNavigate={navigate}/>);return {api,navigate};
}
test('metadata Runtime retains component navigation, live MCP contracts and resources',async()=>{
 const user=userEvent.setup(),{api,navigate}=mount();
 expect(await screen.findByRole('heading',{name:'Runtime',exact:true})).toBeTruthy();
 await screen.findByText('Vendor Catalog');await screen.findByText('Generation 12');
 await user.click(screen.getByRole('button',{name:'Open component',exact:true}));
 expect(navigate).toHaveBeenCalledWith('builder',{report:expect.objectContaining({id:'vendor',ownerPackage:'alice',versionNo:2})});
 await user.click(screen.getByRole('tab',{name:'MCP tools',exact:true}));
 await screen.findByText('alice.vendor.read');
 expect(screen.getAllByText('1 fields')).toHaveLength(2);
 await user.click(screen.getByRole('button',{name:'Inspect schema',exact:true}));
 const dialog=await screen.findByRole('dialog',{name:'MCP tool contract'});
 expect(within(dialog).getByText('limit',{exact:true})).toBeTruthy();expect(within(dialog).getByText('required',{exact:true})).toBeTruthy();
 await user.click(within(dialog).getByRole('tab',{name:'Output',exact:true}));
 expect(within(dialog).getByText('vendors',{exact:true})).toBeTruthy();
 await user.click(within(dialog).getAllByRole('button',{name:'Close',exact:true}).at(-1));
 await user.click(screen.getByRole('tab',{name:'Resources',exact:true}));
 expect(await screen.findByText('skill://alice-guide/')).toBeTruthy();
 await user.click(screen.getByRole('button',{name:'Refresh',exact:true}));
 await waitFor(()=>expect(api.listMCPTools).toHaveBeenCalledTimes(2));
});
test('metadata Runtime searches the live tool catalog and keeps healthy status on discovery failure',async()=>{
 const user=userEvent.setup();mount();await screen.findByText('Vendor Catalog');
 await user.click(screen.getByRole('tab',{name:'MCP tools',exact:true}));
 const query=screen.getByLabelText('Find MCP tool name or description',{exact:true});
 await user.type(query,'missing');await waitFor(()=>expect(screen.queryByText('alice.vendor.read',{exact:true})).toBeNull());
 await user.clear(query);expect(await screen.findByText('alice.vendor.read',{exact:true})).toBeTruthy();
});
test('metadata Runtime reports partial MCP failure while retaining deployed components',async()=>{
 mount({failedCatalog:true});expect(await screen.findByText('Vendor Catalog')).toBeTruthy();
 expect(await screen.findByRole('alert')).toBeTruthy();expect(screen.getByText(/Datly MCP catalog unavailable: Discovery denied/)).toBeTruthy();
});
test('metadata Runtime keeps live discovery visible when status retrieval fails',async()=>{
 const user=userEvent.setup();mount({failedStatus:true});await screen.findByText(/Runtime status unavailable: Status denied/);
 await user.click(screen.getByRole('tab',{name:'MCP tools',exact:true}));expect(await screen.findByText('alice.vendor.read')).toBeTruthy();
});
