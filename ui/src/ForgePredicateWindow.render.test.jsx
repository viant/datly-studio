import React from 'react';
import {test,expect,vi} from 'vitest';
import {render,screen,waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import WindowContent from 'forge/window';
import {SettingProvider} from 'forge/settings';
import 'forge/packs/blueprint';
import {managementWindowFixture} from './managementWindowFixture.js';
let serial=0;
function mount(group=1){
 const inspection={version:{versionNo:1,sourceRevision:9,state:'draft'},capabilities:{canEdit:true},structure:{availablePredicates:['in','not_in','expr','handler'],component:{rootView:{name:'reader',namespace:'records',columns:[{name:'ID',source:'c.id'}],relations:[]}},declarations:[{parameter:{name:'IDs',typeExpr:'[]int',source:{kind:'query',name:'ids'}},predicates:[{ordinal:0,predicate:{group,name:'in',args:['c','id']}}]}],predicateExpansions:[{group,view:'records',operator:'AND'}]}};
 const calls=[],complete=vi.fn();
 const transport=vi.fn(async request=>{calls.push(request);if(request.dataSourceRef==='predicate_inspect')return {data:[inspection]};if(request.dataSourceRef==='predicate_types')return {data:[{items:[]}]};if(request.dataSourceRef==='predicate_apply')return {data:[{applied:true,inspection:{...inspection,version:{...inspection.version,sourceRevision:10}}}]};return {data:[]};});
 render(<SettingProvider connectorConfig={{window:{service:{uri:'/window'}}}} services={{dataConnectorTransport:transport,hostWindow:{parameters:{reportId:'reader',versionNo:1,predicateID:'IDs:0'},complete,close:vi.fn()}}}><WindowContent window={{windowId:`forge-predicate-${++serial}`,windowKey:'reader-predicate'}} selectedMetadata={managementWindowFixture('reader-predicate')}/></SettingProvider>);
 return {calls,complete};
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
