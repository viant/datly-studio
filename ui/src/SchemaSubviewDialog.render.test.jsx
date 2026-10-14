import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SchemaSubviewDialog } from './SchemaSubviewDialog.jsx';

describe('SchemaSubviewDialog', () => {
  test('blocks an incomplete relation before sending a Datly command', async () => {
    const user = userEvent.setup();
    const api = {
      listComponents: vi.fn().mockResolvedValue({ items: [{ id: 'reader', title: 'Vendor Catalog' }] }),
      listVersions: vi.fn().mockResolvedValue({ items: [{ versionNo: 2 }] }),
      inspectVersion: vi.fn().mockResolvedValue({
        version: { versionNo: 2, sourceRevision: 7 },
        structure: {
          component: { rootView: { name: 'vendor', source: { table: 'VENDOR' }, relations: [] } },
          views: [{ name: 'vendor' }],
        },
      }),
      applyReaderCommand: vi.fn(),
    };
    render(<SchemaSubviewDialog
      api={api}
      isOpen
      connector="main"
      table={{ name: 'PRODUCT', columns: [] }}
      sql="SELECT * FROM PRODUCT"
      onClose={vi.fn()}
      onApplied={vi.fn()}
    />);

    const compile = await screen.findByRole('button', { name: 'Compile and open' });
    await user.click(compile);
    expect((await screen.findByRole('alert')).textContent).toContain('complete relation keys');
    expect(api.applyReaderCommand).not.toHaveBeenCalled();
  });
  test('adds an optional lookup from another connector to a namespace component', async () => {
    const user = userEvent.setup();
    const api = {
      listComponents: vi.fn().mockResolvedValue({items:[{id:'fact',title:'Facts',defaultConnectorName:'facts_db'}]}),
      listVersions: vi.fn().mockResolvedValue({items:[{versionNo:1}]}),
      inspectVersion: vi.fn().mockResolvedValue({version:{versionNo:1,sourceRevision:3},structure:{views:[{name:'fact'}],component:{rootView:{name:'fact',source:{table:'sales'},relations:[]}}}}),
      applyReaderCommand: vi.fn().mockResolvedValue({applied:true}),
    };
    const onApplied=vi.fn();
    render(<SchemaSubviewDialog isOpen api={api} connector="dictionary_db" table={{name:'regions',columns:[]}} sql="SELECT code,label FROM regions" onClose={vi.fn()} onApplied={onApplied}/>);
    await waitFor(()=>expect(screen.getByRole('button',{name:'Compile and open'}).disabled).toBe(false));
    await user.clear(screen.getByLabelText('Child view identifier'));
    await user.type(screen.getByLabelText('Child view identifier'),'region');
    await user.selectOptions(screen.getByLabelText('Relationship'),'lookup');
    await user.type(screen.getByLabelText('Relation keys'),'region.code=fact.region');
    await user.click(screen.getByRole('button',{name:'Compile and open'}));
    expect(api.listComponents).toHaveBeenCalledWith({limit:100});
    expect(api.applyReaderCommand).toHaveBeenCalledWith('fact',1,{expectedSourceRevision:3,operation:{type:'batch',operations:[
      {type:'addView',view:{name:'region',kind:'subview',parent:'fact',join:'LEFT JOIN',sql:'SELECT code,label FROM regions',on:'region.code=fact.region AND 1=1'}},
      {type:'addFunction',function:{name:'use_connector',args:['region','"dictionary_db"']}},
    ]}});
    expect(onApplied).toHaveBeenCalledWith(expect.objectContaining({id:'fact'}));
  });

});


test('waits for component inspection before accepting child names and relation keys',async()=>{
 let finish;
 const api={
  listComponents:vi.fn().mockResolvedValue({items:[{id:'reader',title:'Records',defaultConnectorName:'main'}]}),
  listVersions:vi.fn().mockResolvedValue({items:[{versionNo:1}]}),
  inspectVersion:vi.fn(()=>new Promise(resolve=>{finish=resolve;})),
  applyReaderCommand:vi.fn().mockResolvedValue({applied:true}),
 };
 const user=userEvent.setup();
 render(<SchemaSubviewDialog isOpen api={api} connector="main" table={{name:'contacts',columns:[]}} sql="SELECT * FROM contacts" onClose={vi.fn()} onApplied={vi.fn()}/>);
 await waitFor(()=>expect(api.inspectVersion).toHaveBeenCalled());
 expect(screen.getByLabelText('Child view identifier').disabled).toBe(true);
 expect(screen.getByLabelText('Relation keys').disabled).toBe(true);
 expect(screen.getByRole('button',{name:'Compile and open'}).disabled).toBe(true);
 await act(async()=>{finish({version:{versionNo:1,sourceRevision:2},structure:{views:[{name:'record'}],component:{rootView:{name:'record',source:{table:'records'},relations:[]}}}});});
 await user.clear(screen.getByLabelText('Child view identifier'));
 await user.type(screen.getByLabelText('Child view identifier'),'contact');
 await user.type(screen.getByLabelText('Relation keys'),'contact.record_id=record.id');
 await user.click(screen.getByRole('button',{name:'Compile and open'}));
 expect(api.applyReaderCommand).toHaveBeenCalledWith('reader',1,{expectedSourceRevision:2,operation:{type:'addView',view:{name:'contact',kind:'subview',parent:'record',join:'JOIN',sql:'SELECT * FROM contacts',on:'contact.record_id=record.id'}}});
});


test('ignores a late inspection after selecting another component', async () => {
  let finishFirst;
  const inspection = (parent) => ({version:{versionNo:1,sourceRevision:2},structure:{views:[{name:parent}],component:{rootView:{name:parent,source:{table:parent},relations:[]}}}});
  const api = {
    listComponents:vi.fn().mockResolvedValue({items:[
      {id:'first',title:'First',defaultConnectorName:'main'},
      {id:'second',title:'Second',defaultConnectorName:'main'},
    ]}),
    listVersions:vi.fn().mockResolvedValue({items:[{versionNo:1}]}),
    inspectVersion:vi.fn((id)=>id==='first'?new Promise(resolve=>{finishFirst=resolve;}):Promise.resolve(inspection('second_root'))),
    applyReaderCommand:vi.fn().mockResolvedValue({applied:true}),
  };
  const user = userEvent.setup();
  render(<SchemaSubviewDialog isOpen api={api} connector="main" table={{name:'contacts',columns:[]}} sql="SELECT * FROM contacts" onClose={vi.fn()} onApplied={vi.fn()}/>);
  await waitFor(()=>expect(api.inspectVersion).toHaveBeenCalledWith('first',1));
  await user.selectOptions(screen.getByLabelText('Component'),'second');
  await waitFor(()=>expect(screen.getByLabelText('Child view identifier').disabled).toBe(false));
  await user.clear(screen.getByLabelText('Child view identifier'));
  await user.type(screen.getByLabelText('Child view identifier'),'contact');
  await user.type(screen.getByLabelText('Relation keys'),'contact.record_id=second_root.id');
  await act(async()=>{finishFirst(inspection('first_root'));});
  expect(screen.getByLabelText('Child view identifier').value).toBe('contact');
  expect(screen.getByLabelText('Parent view').value).toBe('second_root');
  await user.click(screen.getByRole('button',{name:'Compile and open'}));
  expect(api.applyReaderCommand).toHaveBeenCalledWith('second',1,expect.objectContaining({operation:expect.objectContaining({view:expect.objectContaining({name:'contact',parent:'second_root',on:'contact.record_id=second_root.id'})})}));
});
