import React from 'react';
import {test,expect,vi} from 'vitest';
import {render,screen,waitFor,fireEvent} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import WindowContent from 'forge/window';
import {SettingProvider} from 'forge/settings';
import 'forge/packs/blueprint';
import {managementWindowFixture} from './managementWindowFixture.js';
let serial=0;
function mount(group=1,predicate={group,name:'in',args:['c','id']}){
 const inspection={version:{versionNo:1,sourceRevision:9,state:'draft'},capabilities:{canEdit:true},structure:{availablePredicates:['in','not_in','expr','handler'],component:{rootView:{name:'reader',namespace:'records',columns:[{name:'ID',source:'c.id'}],relations:[]}},declarations:[{parameter:{name:'IDs',typeExpr:'[]int',source:{kind:'query',name:'ids'}},predicates:[{ordinal:0,predicate}]}],predicateExpansions:[{group,view:'records',operator:'AND'}]}};
 const calls=[],complete=vi.fn(),close=vi.fn();
 const transport=vi.fn(async request=>{calls.push(request);if(request.dataSourceRef==='predicate_inspect')return {data:[inspection]};if(request.dataSourceRef==='predicate_types')return {data:[{items:[]}]};if(request.dataSourceRef==='predicate_apply')return {data:[{applied:true,inspection:{...inspection,version:{...inspection.version,sourceRevision:10}}}]};return {data:[]};});
 render(<SettingProvider connectorConfig={{window:{service:{uri:'/window'}}}} services={{dataConnectorTransport:transport,hostWindow:{complete,close}}}><WindowContent window={{windowId:`forge-predicate-${++serial}`,windowKey:'reader-predicate',parameters:{reportId:'reader',versionNo:1,predicateID:'IDs:0'}}} selectedMetadata={managementWindowFixture('reader-predicate')}/></SettingProvider>);
 return {calls,complete,close};
}
test('Forge predicate metadata edits the exact occurrence and preserves its view scope',async()=>{
 const user=userEvent.setup(),{calls,complete}=mount();
 const picker=await screen.findByLabelText('Predicate',{exact:true});await user.click(picker);await user.click(await screen.findByRole('menuitem',{name:'not_in',exact:true}));
 await user.click(screen.getByRole('button',{name:'Save predicate',exact:true}));await waitFor(()=>expect(complete).toHaveBeenCalled());
 expect(calls.find(item=>item.dataSourceRef==='predicate_apply').options.inputParameters.command).toEqual({expectedSourceRevision:9,operation:{type:'updateFieldPredicate',predicate:{field:'IDs',occurrence:0,view:'records',expansionViews:['records'],name:'not_in',args:['c','id'],group:1,groupOperator:'AND',applyWhenAbsent:false}}});
});
test('Forge predicate metadata keeps required authorization predicates read-only',async()=>{
 const {calls}=mount(99);await screen.findByLabelText('Predicate',{exact:true});
 expect(screen.getByRole('button',{name:'Save predicate',exact:true}).disabled).toBe(true);
 expect(screen.getByRole('button',{name:'Remove predicate',exact:true}).disabled).toBe(true);
 expect(calls.some(item=>item.dataSourceRef==='predicate_apply')).toBe(false);
});

const handlerArgs=['alias','p','enabled',false,'limit',0,null,'null'];
const handlerPredicate=args=>({group:1,name:'handler',args:['example.com/filters.Scope'],...(args===undefined?{}:{handlerArgs:args})});
test.each([{args:handlerArgs},{args:undefined}])('custom predicate retains ordered typed arguments and legacy omission: $args',async ({args})=>{
 const user=userEvent.setup(),{calls,complete}=mount(1,handlerPredicate(args));
 const editor=await screen.findByLabelText('Handler arguments (JSON array)',{exact:true});
 expect(editor.value).toBe(args?JSON.stringify(args):'');
 await user.click(screen.getByRole('button',{name:'Save predicate',exact:true}));await waitFor(()=>expect(complete).toHaveBeenCalled());
 const mutation=calls.find(item=>item.dataSourceRef==='predicate_apply').options.inputParameters.command.operation.predicate;
 expect(mutation.args).toEqual(['example.com/filters.Scope']);
 if(args)expect(mutation.handlerArgs).toEqual(args);else expect(mutation).not.toHaveProperty('handlerArgs');
});
test('custom predicate explicitly clears existing arguments',async()=>{
 const user=userEvent.setup(),{calls}=mount(1,handlerPredicate(handlerArgs));
 const editor=await screen.findByLabelText('Handler arguments (JSON array)',{exact:true});
 fireEvent.change(editor,{target:{value:'[]'}});
 await user.click(screen.getByRole('button',{name:'Save predicate',exact:true}));
 await waitFor(()=>expect(calls.some(item=>item.dataSourceRef==='predicate_apply')).toBe(true));
 expect(calls.find(item=>item.dataSourceRef==='predicate_apply').options.inputParameters.command.operation.predicate.handlerArgs).toEqual([]);
});
test('custom predicate migrates legacy string extras before typed arguments and accepts positional edits',async()=>{
 const user=userEvent.setup(),{calls}=mount(1,{...handlerPredicate([false,0,null]),args:['example.com/filters.Scope','legacy','text']});
 const editor=await screen.findByLabelText('Handler arguments (JSON array)',{exact:true});
 expect(JSON.parse(editor.value)).toEqual(['legacy','text',false,0,null]);
 fireEvent.change(editor,{target:{value:'["positional",false,0,null]'}});
 await user.click(screen.getByRole('button',{name:'Save predicate',exact:true}));
 await waitFor(()=>expect(calls.some(item=>item.dataSourceRef==='predicate_apply')).toBe(true));
 expect(calls.find(item=>item.dataSourceRef==='predicate_apply').options.inputParameters.command.operation.predicate).toMatchObject({args:['example.com/filters.Scope'],handlerArgs:['positional',false,0,null]});
});
test.each(['{bad','{}','[[1]]','[{}]','[1.5]','[9007199254740992]'])('custom predicate rejects unsupported argument input %s before dispatch',async value=>{
 const user=userEvent.setup(),{calls}=mount(1,handlerPredicate(handlerArgs));
 fireEvent.change(await screen.findByLabelText('Handler arguments (JSON array)',{exact:true}),{target:{value}});
 await user.click(screen.getByRole('button',{name:'Save predicate',exact:true}));
 expect(calls.some(item=>item.dataSourceRef==='predicate_apply')).toBe(false);
 expect((screen.getByLabelText('Outcome').value)).toMatch(/handler arguments/i);
});
test('leaving the predicate editor cancels edited arguments without a mutation',async()=>{
 const user=userEvent.setup(),{calls,close}=mount(1,handlerPredicate(handlerArgs));
 fireEvent.change(await screen.findByLabelText('Handler arguments (JSON array)',{exact:true}),{target:{value:'["changed",true]'}});
 await user.click(screen.getByRole('button',{name:'Back to predicates',exact:true}));
 expect(close).toHaveBeenCalled();expect(calls.some(item=>item.dataSourceRef==='predicate_apply')).toBe(false);
 expect(handlerArgs).toEqual(['alias','p','enabled',false,'limit',0,null,'null']);
});
