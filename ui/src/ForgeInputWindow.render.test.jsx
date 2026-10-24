import React from 'react';
import {test,expect,vi} from 'vitest';
import {render,screen,waitFor,fireEvent,within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import WindowContent from 'forge/window';
import {SettingProvider} from 'forge/settings';
import 'forge/packs/blueprint';
import {managementWindowFixture} from './managementWindowFixture.js';
let serial=0;
function mount(inputName='',predicate=null){
 const inspection={version:{versionNo:1,sourceRevision:7,state:'draft'},capabilities:{canEdit:true},structure:{availablePredicates:['equal','in','expr','handler'],component:{rootView:{name:'reader',namespace:'records',columns:[{name:'ID',source:'c.id'}],relations:[]}},declarations:inputName?[{parameter:{name:inputName,typeExpr:'int',source:{kind:'query',name:'limit'},required:false,errorStatusCode:422,errorMessage:'Invalid limit'},...(predicate?{predicates:[{ordinal:0,predicate}]}:{})}]:[],predicateExpansions:predicate?[{group:predicate.group,view:'records',operator:'AND'}]:[]}};
 const calls=[],complete=vi.fn(),close=vi.fn();
 const transport=vi.fn(async request=>{
  calls.push(request);const id=request.dataSourceRef;
  if(id==='input_inspect')return {data:[inspection]};
  if(id==='input_codec_types')return {data:[{items:[{name:'JwtClaim',acceptsArguments:false}]}]};
  if(id==='input_handler_types')return {data:[{items:[{packagePath:'example.com/filters',typeName:'Scope'}]}]};
  if(id==='input_apply')return {data:[{applied:true,inspection:{...inspection,version:{...inspection.version,sourceRevision:8}}}]};
  return {data:[]};
 });
 render(<SettingProvider connectorConfig={{window:{service:{uri:'/window'}}}} services={{dataConnectorTransport:transport,hostWindow:{complete,close}}}><WindowContent window={{windowId:`forge-input-${++serial}`,windowKey:'reader-input',parameters:{reportId:'reader',versionNo:1,inputName}}} selectedMetadata={managementWindowFixture('reader-input')}/></SettingProvider>);
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

const typedArgs=['alias','p',false,0,null,'null'];
async function editHandler(user,args=typedArgs,legacy=[]){
 const result=mount('IDs',{group:1,name:'handler',args:['example.com/filters.Scope',...legacy],...(args===null?{}:{handlerArgs:args})});
 await user.click(await screen.findByText(['example.com/filters.Scope',...legacy].join(', '),{exact:true}));
 await user.click(screen.getByRole('button',{name:'Edit predicate',exact:true}));
 const dialog=await screen.findByRole('dialog',{name:'Predicate settings'});
 const editor=await within(dialog).findByLabelText('Handler arguments (JSON array)',{exact:true});
 return {...result,dialog,editor};
}
test('input predicate editor loads and saves ordered typed positional arguments',async()=>{
 const user=userEvent.setup(),{calls,complete,dialog,editor}=await editHandler(user);
 expect(editor.value).toBe(JSON.stringify(typedArgs));
 await user.click(within(dialog).getByRole('button',{name:'Save predicate',exact:true}));await waitFor(()=>expect(complete).toHaveBeenCalled());
 expect(calls.find(item=>item.dataSourceRef==='input_apply').options.inputParameters.command.operation.predicate).toMatchObject({name:'handler',args:['example.com/filters.Scope'],handlerArgs:typedArgs});
});
test('input predicate editor clears previously authored arguments explicitly',async()=>{
 const user=userEvent.setup(),{calls,dialog,editor}=await editHandler(user);
 fireEvent.change(editor,{target:{value:''}});
 await user.click(within(dialog).getByRole('button',{name:'Save predicate',exact:true}));
 await waitFor(()=>expect(calls.some(item=>item.dataSourceRef==='input_apply')).toBe(true));
 expect(calls.find(item=>item.dataSourceRef==='input_apply').options.inputParameters.command.operation.predicate.handlerArgs).toEqual([]);
});
test('input predicate editor preserves legacy strings before new typed arguments',async()=>{
 const user=userEvent.setup(),{calls,dialog,editor}=await editHandler(user,[false,0,null],['legacy','text']);
 expect(JSON.parse(editor.value)).toEqual(['legacy','text',false,0,null]);
 await user.click(within(dialog).getByRole('button',{name:'Save predicate',exact:true}));
 await waitFor(()=>expect(calls.some(item=>item.dataSourceRef==='input_apply')).toBe(true));
 expect(calls.find(item=>item.dataSourceRef==='input_apply').options.inputParameters.command.operation.predicate).toMatchObject({args:['example.com/filters.Scope'],handlerArgs:['legacy','text',false,0,null]});
});
test('input predicate editor keeps an old handler without arguments unchanged',async()=>{
 const user=userEvent.setup(),{calls,dialog,editor}=await editHandler(user,null);
 expect(editor.value).toBe('');await user.click(within(dialog).getByRole('button',{name:'Save predicate',exact:true}));
 await waitFor(()=>expect(calls.some(item=>item.dataSourceRef==='input_apply')).toBe(true));
 expect(calls.find(item=>item.dataSourceRef==='input_apply').options.inputParameters.command.operation.predicate).not.toHaveProperty('handlerArgs');
});
test('input predicate editor rejects nested values and cancels without changing source arguments',async()=>{
 const user=userEvent.setup(),{calls,dialog,editor}=await editHandler(user);
 fireEvent.change(editor,{target:{value:'[{}]'}});
 await user.click(within(dialog).getByRole('button',{name:'Save predicate',exact:true}));
 expect(calls.some(item=>item.dataSourceRef==='input_apply')).toBe(false);
 await user.click(within(dialog).getByRole('button',{name:'Cancel',exact:true}));
 expect(calls.some(item=>item.dataSourceRef==='input_apply')).toBe(false);
 expect(typedArgs).toEqual(['alias','p',false,0,null,'null']);
});
test.each([{args:undefined},{args:['alias','p','enabled',true,'offset',0,'fallback',null]}])('new input custom predicate preserves optional typed arguments: $args',async({args})=>{
 const user=userEvent.setup(),{calls,complete}=mount();
 await user.type(await screen.findByLabelText('Input name',{exact:true}),'IDs');await user.type(screen.getByLabelText('Kind source',{exact:true}),'ids');
 await user.click(screen.getByText('Predicate',{selector:'summary'}));await choose(user,'Predicate','Custom Go predicate');await choose(user,'Custom predicate','Scope — example.com/filters');
 const editor=screen.getByLabelText('Handler arguments (JSON array)',{exact:true});expect(editor.value).toBe('');if(args)fireEvent.change(editor,{target:{value:JSON.stringify(args)}});
 await user.click(screen.getByRole('button',{name:'Save input',exact:true}));await waitFor(()=>expect(complete).toHaveBeenCalled());
 const mutation=calls.find(item=>item.dataSourceRef==='input_apply').options.inputParameters.command.operation.operations[1].predicate;
 expect(mutation.args).toEqual(['example.com/filters.Scope']);if(args)expect(mutation.handlerArgs).toEqual(args);else expect(mutation).not.toHaveProperty('handlerArgs');
});
