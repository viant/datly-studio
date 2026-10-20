import React from 'react';
import {test,expect,vi} from 'vitest';
import {render,screen,waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import WindowContent from 'forge/window';
import {SettingProvider} from 'forge/settings';
import 'forge/packs/blueprint';
import {managementWindowFixture} from './managementWindowFixture.js';
let serial=0;
function mount(inputName=''){
 const inspection={version:{versionNo:1,sourceRevision:7,state:'draft'},capabilities:{canEdit:true},structure:{availablePredicates:['equal','in','expr','handler'],component:{rootView:{name:'reader',namespace:'records',columns:[{name:'ID',source:'c.id'}],relations:[]}},declarations:inputName?[{parameter:{name:inputName,typeExpr:'int',source:{kind:'query',name:'limit'},required:false,errorStatusCode:422,errorMessage:'Invalid limit'}}]:[],predicateExpansions:[]}};
 const calls=[],complete=vi.fn(),close=vi.fn();
 const transport=vi.fn(async request=>{
  calls.push(request);const id=request.dataSourceRef;
  if(id==='input_inspect')return {data:[inspection]};
  if(id==='input_codec_types')return {data:[{items:[{name:'JwtClaim',acceptsArguments:false}]}]};
  if(id==='input_handler_types')return {data:[{items:[{packagePath:'example.com/filters',typeName:'Scope'}]}]};
  if(id==='input_apply')return {data:[{applied:true,inspection:{...inspection,version:{...inspection.version,sourceRevision:8}}}]};
  return {data:[]};
 });
 render(<SettingProvider connectorConfig={{window:{service:{uri:'/window'}}}} services={{dataConnectorTransport:transport,hostWindow:{parameters:{reportId:'reader',versionNo:1,inputName},complete,close}}}><WindowContent window={{windowId:`forge-input-${++serial}`,windowKey:'reader-input'}} selectedMetadata={managementWindowFixture('reader-input')}/></SettingProvider>);
 return {calls,complete,close};
}
async function choose(user,label,option){await user.click(screen.getByLabelText(label,{exact:true}));await user.click(await screen.findByRole('menuitem',{name:option,exact:true}));}
test('Forge parameter editor creates an input and predicate through the metadata MCP command',async()=>{
 const user=userEvent.setup(),{calls,complete}=mount();
 await user.type(await screen.findByLabelText('Input name',{exact:true}),'IDs');
 const type=screen.getByLabelText('Type expression',{exact:true});await user.clear(type);await user.type(type,'int');
 await user.type(screen.getByLabelText('Kind source',{exact:true}),'ids');
 await user.click(screen.getByText('Predicate',{selector:'summary'}));
 await choose(user,'Predicate','in');
 await user.type(screen.getByLabelText('SQL alias',{exact:true}),'c');
 await choose(user,'Filter dimension','ID (c.id)');
 await user.click(screen.getByRole('button',{name:'Save input',exact:true}));
 await waitFor(()=>expect(complete).toHaveBeenCalled());
 const request=calls.find(item=>item.dataSourceRef==='input_apply').options.inputParameters;
 expect(request).toMatchObject({reportId:'reader',versionNo:1,command:{expectedSourceRevision:7,operation:{type:'batch',operations:[{type:'addField',field:{name:'IDs',sourceKind:'query',sourceName:'ids'}},{type:'addFieldPredicate',predicate:{name:'in',field:'IDs',view:'records',args:['c','id']}}]}}});
 expect(complete).toHaveBeenCalledWith(expect.objectContaining({close:true}));
});
test('Forge codec lookup authors the exact JWT binding and error contract',async()=>{
 const user=userEvent.setup(),{calls,complete}=mount();
 await user.type(await screen.findByLabelText('Input name',{exact:true}),'JWT');
 await user.click(screen.getByText('Binding options',{selector:'summary'}));
 await choose(user,'Codec','JwtClaim');
 expect(screen.getByLabelText('Type expression',{exact:true}).value).toBe('string,*github.com/viant/scy/auth/jwt.Claims');
 expect(screen.getByLabelText('Kind source',{exact:true}).value).toBe('Authorization');
 await user.click(screen.getByRole('button',{name:'Save input',exact:true}));
 await waitFor(()=>expect(complete).toHaveBeenCalled());
 expect(calls.find(item=>item.dataSourceRef==='input_apply').options.inputParameters.command.operation.field).toMatchObject({required:true,errorStatusCode:401,errorMessage:'Unauthorized',codec:{name:'JwtClaim',args:[]}});
});
test('Forge removal requires the exact input name and preserves the source revision guard',async()=>{
 const user=userEvent.setup(),{calls,complete}=mount('Limit');
 await screen.findByDisplayValue('Limit');
 await user.click(screen.getByRole('button',{name:'Remove input',exact:true}));
 await user.type(await screen.findByLabelText('Confirm input name',{exact:true}),'Limit');
 await user.click(screen.getByRole('button',{name:'Confirm removal',exact:true}));
 await waitFor(()=>expect(complete).toHaveBeenCalled());
 expect(calls.find(item=>item.dataSourceRef==='input_apply').options.inputParameters.command).toEqual({expectedSourceRevision:7,operation:{type:'removeField',field:{existingName:'Limit'}}});
});
