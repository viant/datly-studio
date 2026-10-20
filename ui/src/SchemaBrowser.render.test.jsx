import React from 'react';
import {EditorView} from '@codemirror/view';
import {test,expect,vi} from 'vitest';
import {act,render,screen,waitFor,within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ManagementWindowRuntime from './ManagementWindowRuntime.jsx';
import {managementResourceFixture} from './managementWindowFixture.js';
let serial=0;
function mount({denied=false,connector='main',inspectVersion,components}={}){
 const navigate=vi.fn(),dirty=vi.fn();
 const api={readManagementResource:async()=>managementResourceFixture('schema'),fetchMCPDataSource:vi.fn(async(source,input)=>{
  switch(source.backend.method){
   case 'studio.sdk.connectors.list':if(denied)throw new Error('connector catalog unavailable');return {items:[{name:connector,driver:'sqlite'}]};
   case 'studio.sdk.connectors.schemas':return {items:[{catalog:'main',name:'sales'}]};
   case 'studio.sdk.connectors.tables':return {items:[{schema:'sales',name:'VENDOR',type:'TABLE'}]};
   case 'studio.sdk.connectors.table':return {table:{schema:'sales',name:'VENDOR',type:'TABLE'},columns:[{name:'ID',type:'INTEGER',primaryKey:true,nullable:false}]};
   case 'studio.sdk.connectors.test_sql':return {duration:1500000,data:[{ID:1}]};
   case 'studio.sdk.components.list':return {items:components||[{id:'fact',title:'Facts',defaultConnectorName:'facts_db'}]};
   case 'studio.sdk.versions.list':return {items:[{versionNo:2}]};
   case 'studio.sdk.versions.inspect':if(inspectVersion)return inspectVersion(input);return {version:{versionNo:2,sourceRevision:7,state:'draft'},capabilities:{canEdit:true},structure:{views:[{name:'fact'}],component:{rootView:{name:'fact',source:{table:'sales'},relations:[]}}}};
   case 'studio.sdk.components.create':return {id:'created',title:'Vendor',defaultConnectorName:connector};
   case 'studio.sdk.versions.create':return {versionNo:1,sourceRevision:0};
   case 'studio.sdk.versions.builder':return {applied:true};
   default:throw new Error('Unexpected '+source.backend.method);
  }
 })};
 render(<ManagementWindowRuntime api={api} uri="window://datastudio/schema" instanceKey={'forge-schema-'+(++serial)} onNavigate={navigate} onDirtyChange={dirty}/>);return {api,navigate,dirty};
}
async function inspect(user){await user.click(await screen.findByRole('button',{name:'Inspect table',exact:true}));return screen.findByRole('textbox',{name:'SQL source',exact:true});}
test('Forge schema discovery keeps SQL, columns and real SQL results in tabs',async()=>{
 const user=userEvent.setup(),{api}=mount();const editor=await inspect(user);expect(editor.textContent.replaceAll('\n','')).toBe('SELECT *FROM sales.VENDOR');
 await user.click(screen.getByRole('tab',{name:'Columns',exact:true}));expect(await screen.findByText('PK · required')).toBeTruthy();
 await user.click(screen.getByRole('tab',{name:'SQL',exact:true}));await user.click(screen.getByRole('button',{name:'Test SQL',exact:true}));
 expect(await screen.findByRole('heading',{name:'SQL test result'})).toBeTruthy();expect(screen.getByRole('tab',{name:'Result',exact:true}).getAttribute('aria-selected')).toBe('true');
 expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.connectors.test_sql')[1]).toEqual({name:'main',input:{sql:'SELECT *\nFROM sales.VENDOR',limit:50}});
 expect(within(screen.getByRole('tabpanel',{name:'Result'})).getByText('1',{exact:true})).toBeTruthy();
 await user.click(screen.getByRole('tab',{name:'SQL',exact:true}));
 const view=EditorView.findFromDOM(screen.getByRole('textbox',{name:'SQL source',exact:true}).closest('.cm-editor'));
 await act(async()=>view.dispatch({changes:{from:0,to:view.state.doc.length,insert:'SELECT 2 AS ID'}}));
 await user.click(screen.getByRole('tab',{name:'Result',exact:true}));expect(within(screen.getByRole('tabpanel',{name:'Result'})).queryByText('1',{exact:true})).toBeNull();
});
test('Forge schema root creation uses typed MCP commands and opens the new version',async()=>{
 const user=userEvent.setup(),{api,navigate}=mount();await inspect(user);await user.click(screen.getByRole('button',{name:'Add as root view',exact:true}));await user.click(await screen.findByRole('button',{name:'Create component',exact:true}));
 await waitFor(()=>expect(navigate).toHaveBeenCalledWith('builder',{report:expect.objectContaining({id:'created',versionNo:1})}));
 expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.versions.builder')[1]).toEqual({reportId:'created',versionNo:1,command:{expectedSourceRevision:0,operation:{type:'createReader',reader:{route:'/v1/studio/readers/vendor',name:'vendor',sql:'SELECT *\nFROM sales.VENDOR'}}}});
});
test('Forge schema child creation requires keys and preserves optional cross-connector lookup semantics',async()=>{
 const user=userEvent.setup(),{api,navigate}=mount({connector:'dictionary_db'});await inspect(user);await user.click(screen.getByRole('button',{name:'Add as subview',exact:true}));
 const compile=await screen.findByRole('button',{name:'Compile and open',exact:true});await waitFor(()=>expect(compile.disabled).toBe(false));await user.click(compile);
 expect(screen.getByLabelText('Outcome',{exact:true}).value).toContain('complete relation keys');expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.versions.builder')).toBe(false);
 const name=screen.getByLabelText('Child view identifier',{exact:true});await user.clear(name);await user.type(name,'region');await user.click(screen.getByRole('button',{name:'Relationship',exact:true}));await user.click(await screen.findByRole('menuitem',{name:'Optional single lookup',exact:true}));await user.type(screen.getByLabelText('Relation keys',{exact:true}),'region.code=fact.region');await user.click(compile);await waitFor(()=>expect(navigate).toHaveBeenCalled());
 expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.versions.builder')[1]).toEqual({reportId:'fact',versionNo:2,command:{expectedSourceRevision:7,operation:{type:'batch',operations:[{type:'addView',view:{name:'region',kind:'subview',parent:'fact',join:'LEFT JOIN',sql:'SELECT *\nFROM sales.VENDOR',on:'region.code=fact.region AND 1=1'}},{type:'addFunction',function:{name:'use_connector',args:['region','"dictionary_db"']}}]}}});
});
test('Forge schema discovery surfaces rejection without inventing metadata',async()=>{
 const {api}=mount({denied:true});expect((await screen.findByRole('alert')).textContent).toContain('connector catalog unavailable');expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.connectors.schemas')).toBe(false);
});

test('Forge child authoring stays protected until inspection and cancels an outstanding read',async()=>{
 let finish;const user=userEvent.setup(),{navigate}=mount({inspectVersion:()=>new Promise(resolve=>{finish=resolve;})});await inspect(user);await user.click(screen.getByRole('button',{name:'Add as subview',exact:true}));
 await waitFor(()=>expect(finish).toBeTypeOf('function'));expect(screen.getByLabelText('Child view identifier',{exact:true}).readOnly).toBe(true);expect(screen.getByLabelText('Relation keys',{exact:true}).readOnly).toBe(true);expect(screen.getByRole('button',{name:'Compile and open',exact:true}).disabled).toBe(true);
 await user.click(screen.getByRole('button',{name:'Cancel',exact:true}));await waitFor(()=>expect(screen.queryByRole('dialog')).toBeNull());
 await act(async()=>finish({version:{versionNo:2,sourceRevision:7,state:'draft'},structure:{views:[{name:'old'}]}}));expect(navigate).not.toHaveBeenCalled();
});
test('Forge child authoring ignores an older component inspection after a new selection',async()=>{
 let first;const response=parent=>({version:{versionNo:2,sourceRevision:7,state:'draft'},structure:{views:[{name:parent}],component:{rootView:{name:parent,source:{table:parent},relations:[]}}}});
 const user=userEvent.setup();mount({components:[{id:'first',title:'First',defaultConnectorName:'main'},{id:'second',title:'Second',defaultConnectorName:'main'}],inspectVersion:input=>input.reportId==='first'?new Promise(resolve=>{first=resolve;}):Promise.resolve(response('second_root'))});
 await inspect(user);await user.click(screen.getByRole('button',{name:'Add as subview',exact:true}));await waitFor(()=>expect(first).toBeTypeOf('function'));
 await user.click(screen.getByRole('button',{name:'Component',exact:true}));await user.click(await screen.findByRole('menuitem',{name:'Second',exact:true}));await waitFor(()=>expect(screen.getByLabelText('Child view identifier',{exact:true}).readOnly).toBe(false));
 const child=screen.getByLabelText('Child view identifier',{exact:true});await user.clear(child);await user.type(child,'contact');await act(async()=>first(response('first_root')));
 expect(child.value).toBe('contact');expect(screen.getByRole('button',{name:'Parent view',exact:true}).textContent).toContain('second_root');
});


test('schema search locks inspect until the current metadata request completes',async()=>{
 const user=userEvent.setup(),{api}=mount();
 const inspectButton=await screen.findByRole('button',{name:'Inspect table',exact:true});await waitFor(()=>expect(inspectButton.disabled).toBe(false));
 const original=api.fetchMCPDataSource.getMockImplementation();let finish;
 api.fetchMCPDataSource.mockImplementation((source,input)=>source.backend.method==='studio.sdk.connectors.tables'?new Promise(resolve=>{finish=resolve;}):original(source,input));
 await user.click(screen.getByRole('button',{name:'Search tables',exact:true}));await waitFor(()=>expect(inspectButton.disabled).toBe(true));expect(screen.getByRole('button',{name:'Search tables',exact:true}).disabled).toBe(true);
 await user.click(inspectButton);expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.connectors.table')).toBe(false);
 await act(async()=>finish({items:[{schema:'sales',name:'VENDOR',type:'TABLE'}]}));await waitFor(()=>expect(inspectButton.disabled).toBe(false));await user.click(inspectButton);
 await screen.findByRole('textbox',{name:'SQL source',exact:true});expect(api.fetchMCPDataSource.mock.calls.filter(([source])=>source.backend.method==='studio.sdk.connectors.table')).toHaveLength(1);
});
