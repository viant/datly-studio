import React from 'react';
import {test,expect,vi} from 'vitest';
import {act,render,screen,waitFor,within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {EditorView} from '@codemirror/view';
import ManagementWindowRuntime from './ManagementWindowRuntime.jsx';
import {managementResourceFixture} from './managementWindowFixture.js';
let serial=0;
function mount({execute}={}) {
 const api={readManagementResource:async()=>managementResourceFixture('schema'),fetchMCPDataSource:vi.fn(async(source,input)=>{
  switch(source.backend.method){
   case 'studio.sdk.connectors.list':return {items:[{name:'Bq',driver:'bigquery',status:'active'},{name:'other',driver:'sqlite',status:'active'}]};
   case 'studio.sdk.connectors.discover':return {catalog:'project',catalogs:[{name:'project'}],schemas:[{name:'dataset',catalog:'project'}]};
   case 'studio.sdk.connectors.tables':return {items:[{name:'orders',schema:'dataset',type:'TABLE'},{name:'customers',schema:'dataset',type:'TABLE'}]};
   case 'studio.sdk.connectors.table':return {table:{name:input.input.table,schema:'dataset'},columns:[{name:'id',type:'INTEGER'}]};
   case 'studio.sdk.connectors.test_sql':return execute?execute(input):{data:[{answer:input.input.sql}],duration:1};
   case 'studio.sdk.components.create':return {id:'new-component',title:input.title,defaultConnectorName:input.defaultConnectorName};
   case 'studio.sdk.versions.create':return {versionNo:1,sourceRevision:1};
   case 'studio.sdk.versions.builder':return {applied:true};
   default:throw new Error('Unexpected '+source.backend.method);
  }
 })};const navigate=vi.fn();
 render(<ManagementWindowRuntime api={api} uri="window://datastudio/schema" onNavigate={navigate} instanceKey={'explorer-tabs-'+(++serial)}/>);
 return {api,navigate};
}
async function editSQL(text){const element=await screen.findByRole('textbox',{name:'SQL query',exact:true});const view=EditorView.findFromDOM(element.closest('.cm-editor'));await act(async()=>view.dispatch({changes:{from:0,to:view.state.doc.length,insert:text}}));}

test('each execution gets an immutable result tab and sends the explicit uncapped timeout',async()=>{
 const user=userEvent.setup(),{api}=mount();await editSQL('SELECT 1 AS answer');
 const timeout=screen.getByLabelText('Timeout (seconds)',{exact:true});await user.clear(timeout);await user.type(timeout,'0');
 await user.click(screen.getByRole('button',{name:'Run SQL',exact:true}));
 await screen.findByText('SELECT 1 AS answer',{selector:'td'});
 const first=screen.getByRole('tab',{name:/Run 1/});
 await user.click(screen.getByRole('tab',{name:/SQL 1/}));await editSQL('SELECT 2 AS answer');
 await user.clear(screen.getByLabelText('Timeout (seconds)',{exact:true}));await user.type(screen.getByLabelText('Timeout (seconds)',{exact:true}),'600');
 await user.click(screen.getByRole('button',{name:'Run SQL',exact:true}));await screen.findByText('SELECT 2 AS answer',{selector:'td'});
 const calls=api.fetchMCPDataSource.mock.calls.filter(([source])=>source.backend.method==='studio.sdk.connectors.test_sql');
 expect(calls.map(([,input])=>input.input.timeoutSeconds)).toEqual([0,600]);
 await user.click(first);expect(screen.getByRole('textbox',{name:'Executed SQL',exact:true}).textContent).toContain('SELECT 1 AS answer');
 expect(within(document.getElementById(first.getAttribute('aria-controls'))).getByLabelText('Timeout used (seconds)',{exact:true}).value).toBe('0');
 expect(screen.getAllByRole('tab').filter(tab=>tab.textContent.includes('Run '))).toHaveLength(2);
});

test('SQL tabs retain their connector and table discovery opens separate tabs',async()=>{
 const user=userEvent.setup(),{api}=mount();await editSQL('SELECT 1');
 await user.click(screen.getByRole('button',{name:'Connector',exact:true}));await user.click(await screen.findByRole('menuitem',{name:'other',exact:true}));
 await user.click(screen.getByRole('button',{name:'New SQL tab',exact:true}));await editSQL('SELECT 2');
 await user.click(screen.getByRole('button',{name:'Run SQL',exact:true}));await screen.findByText('SELECT 2',{selector:'td'});
 expect(api.fetchMCPDataSource.mock.calls.filter(([source])=>source.backend.method==='studio.sdk.connectors.test_sql').at(-1)[1].name).toBe('other');
 await user.click(screen.getByRole('button',{name:'Load schemas',exact:true}));await waitFor(()=>expect(screen.getByRole('button',{name:'Load schemas',exact:true}).disabled).toBe(false));
 await user.click(screen.getByRole('button',{name:'Search tables',exact:true}));
 const buttons=await screen.findAllByRole('button',{name:'Inspect table',exact:true});await user.click(buttons[0]);await screen.findByRole('tab',{name:/orders/});
 await user.click(screen.getAllByRole('button',{name:'Inspect table',exact:true})[1]);await screen.findByRole('tab',{name:/customers/});
 expect(screen.getByRole('tab',{name:/orders/})).toBeTruthy();expect(screen.getByRole('tab',{name:/SQL 1.*Bq/})).toBeTruthy();
});

test('a result tab creates a component from that execution SQL and connector',async()=>{
 const user=userEvent.setup(),{api,navigate}=mount();await editSQL('SELECT 7 AS value');
 await user.click(screen.getByRole('button',{name:'Run SQL',exact:true}));await screen.findByText('SELECT 7 AS value',{selector:'td'});
 await user.click(screen.getByRole('button',{name:'Create component',exact:true}));
 const dialog=await screen.findByRole('dialog',{name:'Create component',exact:true});await user.click(within(dialog).getByRole('button',{name:'Create component',exact:true}));
 await waitFor(()=>expect(navigate).toHaveBeenCalled());
 expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.components.create')[1].defaultConnectorName).toBe('Bq');
 expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.versions.builder')[1].command.operation.reader.sql).toBe('SELECT 7 AS value');
});

test('overflowing tabs have clickable left and right scrolling controls',async()=>{
 const user=userEvent.setup();mount();await screen.findByRole('textbox',{name:'SQL query',exact:true});
 for(let index=0;index<5;index++)await user.click(screen.getByRole('button',{name:'New SQL tab',exact:true}));
 const rail=screen.getByRole('tablist',{name:'SQL and table tabs'});
 Object.defineProperties(rail,{scrollWidth:{configurable:true,value:1500},clientWidth:{configurable:true,value:300},scrollLeft:{configurable:true,writable:true,value:0}});
 rail.scrollBy=vi.fn(({left})=>{rail.scrollLeft=Math.max(0,Math.min(1200,rail.scrollLeft+left));rail.dispatchEvent(new Event('scroll'));});
 await act(async()=>rail.dispatchEvent(new Event('scroll')));
 expect(screen.getByRole('button',{name:'Scroll tabs left'}).disabled).toBe(true);
 await user.click(screen.getByRole('button',{name:'Scroll tabs right'}));
 await waitFor(()=>expect(screen.getByRole('button',{name:'Scroll tabs left'}).disabled).toBe(false));
 expect(rail.scrollLeft).toBeGreaterThan(0);
 await user.click(screen.getByRole('button',{name:'Scroll tabs left'}));
 await waitFor(()=>expect(rail.scrollLeft).toBe(0));
 expect(screen.getAllByRole('tab')).toHaveLength(6);
});
