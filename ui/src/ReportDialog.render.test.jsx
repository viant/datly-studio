import React from 'react';
import {expect,test,vi} from 'vitest';
import {render,screen,waitFor,within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ManagementWindowRuntime from './ManagementWindowRuntime.jsx';
import {managementResourceFixture} from './managementWindowFixture.js';
let serial=0;
function mount({namespaces=[{name:'general',title:'General'}],create=async()=>({id:'new',title:'Vendor Catalog'})}={}) {
 const navigate=vi.fn(),dirty=vi.fn();
 const api={readManagementResource:async()=>managementResourceFixture('component-catalog'),fetchMCPDataSource:vi.fn(async(source,input)=>{
  const method=source.backend.method;
  if(method==='studio.sdk.components.list')return {items:[]};
  if(method==='studio.sdk.namespaces.list')return {items:namespaces};
  if(method==='studio.sdk.connectors.list')return {items:[{name:'main',driver:'sqlite'}]};
  if(method==='studio.sdk.components.create')return create(input);
  throw new Error('Unexpected '+method);
 })};
 render(<ManagementWindowRuntime api={api} uri="window://datastudio/component-catalog" instanceKey={'forge-create-'+(++serial)} onNavigate={navigate} onDirtyChange={dirty}/>);
 return {api,navigate,dirty};
}
async function open(user){await user.click(await screen.findByRole('button',{name:'New component',exact:true}));return screen.findByRole('dialog',{name:'New component',exact:true});}
async function fill(user){await user.type(await screen.findByLabelText('Component title',{exact:true}),'Vendor Catalog');await user.click(screen.getByRole('button',{name:'Active connector',exact:true}));await user.click(await screen.findByRole('menuitem',{name:'main · sqlite',exact:true}));}
test('Forge component creation requires a governed namespace',async()=>{
 const user=userEvent.setup(),{api}=mount({namespaces:[]});await open(user);
 expect(await screen.findByLabelText('Outcome',{exact:true})).toBeTruthy();
 expect(screen.getByLabelText('Outcome',{exact:true}).value).toBe('Create an active namespace before creating a component.');
 expect(screen.getByRole('button',{name:'Create component',exact:true}).disabled).toBe(true);
 expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.components.create')).toBe(false);
});
test('Forge component creation retains values and error after a rejected MCP mutation',async()=>{
 const user=userEvent.setup(),{navigate}=mount({create:async()=>{throw new Error('component identity already exists');}});await open(user);await fill(user);
 await user.type(screen.getByLabelText('Description',{exact:true}),'Production vendors');await waitFor(()=>expect(screen.getByLabelText('Description',{exact:true}).value).toBe('Production vendors'));await user.click(screen.getByRole('button',{name:'Create component',exact:true}));
 await waitFor(()=>expect(screen.getByLabelText('Outcome',{exact:true}).value).toBe('component identity already exists'));
 expect(screen.getByLabelText('Component title',{exact:true}).value).toBe('Vendor Catalog');expect(screen.getByLabelText('Slug',{exact:true}).value).toBe('vendor-catalog');expect(screen.getByLabelText('Description',{exact:true}).value).toBe('Production vendors');expect(navigate).not.toHaveBeenCalled();
});
test('Forge protects creation during save and opens the created component after success',async()=>{
 let finish;const user=userEvent.setup(),{navigate,dirty}=mount({create:()=>new Promise(resolve=>{finish=resolve;})});await open(user);await fill(user);
 await user.click(screen.getByRole('button',{name:'Create component',exact:true}));await waitFor(()=>expect(finish).toBeTypeOf('function'));
 await user.keyboard('{Escape}');const dialog=screen.getByRole('dialog',{name:'New component'});expect(within(dialog).getByRole('button',{name:'Cancel'}).disabled).toBe(true);
 finish({id:'new',title:'Vendor Catalog'});await waitFor(()=>expect(navigate).toHaveBeenCalledWith('builder',{report:{id:'new',title:'Vendor Catalog'}}));
 await waitFor(()=>expect(screen.queryByRole('dialog',{name:'New component'})).toBeNull());expect(dirty).toHaveBeenLastCalledWith(false);
});
test('Forge dismissing a create dialog releases the workspace edit guard',async()=>{
 const user=userEvent.setup(),{dirty}=mount();await open(user);expect(dirty).toHaveBeenCalledWith(true);await user.keyboard('{Escape}');await waitFor(()=>expect(screen.queryByRole('dialog',{name:'New component'})).toBeNull());expect(dirty).toHaveBeenLastCalledWith(false);
});
