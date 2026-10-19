import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

vi.mock('./LazyEditor.jsx', () => ({
  LazyEditor: ({ value, onChange, readOnly, ariaLabel }) => <textarea aria-label={ariaLabel} value={value} readOnly={readOnly} onChange={(event) => onChange(event.target.value)} />,
}));

import { ReaderBuilder } from './ReaderBuilder.jsx';

function readerFixture(capabilities = { canEdit: true, canRun: true, canPublish: true, canUseDql: true }) {
  const version = { versionNo: 1, sourceRevision: 4, compileStatus: 'valid' };
  const rootView = {
    name: 'reader',
    namespace: 'vendor',
    source: { table: 'VENDOR', uri: 'embed:sql/vendor.sql' },
    columns: [{ name: 'ID', source: 'ID', type: { name: 'int' } }],
    relations: [{
      name: 'products', kind: 'subview', cardinality: 'many',
      on: [{ parentColumn: 'ID', childColumn: 'VENDOR_ID' }],
      view: { name: 'products', namespace: 'products', source: { table: 'PRODUCT' }, columns: [], relations: [] },
    }],
  };
  return {
    version,
    dql: 'SELECT * FROM VENDOR',
    capabilities,
    diagnostics: [],
    structure: {
      component: { rootView, routes: [{ path: '/vendors' }] },
      views: [
        { name: 'vendor', sql: 'SELECT * FROM VENDOR', sourceProjectionAll: true },
        { name: 'products', sql: 'SELECT * FROM PRODUCT', sourceProjectionAll: true },
      ],
      declarations: [], predicateExpansions: [], functions: [], columnContracts: [],
    },
  };
}

describe('ReaderBuilder graph-first authoring', () => {
  test('loads the first DQL draft directly from an empty component without choosing another resource', async () => {
    const user = userEvent.setup();
    const api = {listVersions:vi.fn().mockResolvedValue({items:[]}),loadDQL:vi.fn().mockResolvedValue({version:{versionNo:1}})};
    const updated = vi.fn();
    render(<ReaderBuilder api={api} report={{id:'alpha',title:'Alpha marker'}} onReportUpdated={updated} onBack={vi.fn()}/>);
    await screen.findByRole('heading',{name:'Root SQL'});
    await user.click(screen.getByRole('button',{name:'Load DQL'}));
    expect(screen.getByRole('combobox',{name:'Component'}).value).toBe('alpha');
    expect(screen.getByRole('combobox',{name:'Component'}).disabled).toBe(true);
    const file = new File(['SELECT 1'], 'alpha.dql',{type:'text/plain'});
    file.text = async () => 'SELECT 1';
    await user.upload(screen.getByLabelText('DQL or archive'),file);
    await user.click(screen.getByRole('button',{name:'Import as new draft'}));
    await waitFor(()=>expect(api.loadDQL).toHaveBeenCalledWith('alpha',{dql:'SELECT 1'}));
    expect(updated).toHaveBeenCalledWith(expect.objectContaining({id:'alpha',versionNo:1,currentDraftVersion:1}));
  });

  test('opens root SQL immediately and reuses the first draft after a rejected query', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    const api = {
      listVersions: vi.fn().mockResolvedValue({items: []}),
      createVersion: vi.fn().mockResolvedValue({versionNo: 1, sourceRevision: 1, state: 'draft'}),
      inspectVersion: vi.fn().mockResolvedValue(inspection),
      applyReaderCommand: vi.fn()
        .mockResolvedValueOnce({applied: false, inspection: {diagnostics: [{message: 'Invalid root SQL'}]}})
        .mockResolvedValueOnce({applied: true, inspection}),
    };
    render(<ReaderBuilder api={api} report={{id:'new-reader',slug:'new-reader',title:'New reader',defaultConnectorName:'main'}} onBack={vi.fn()}/>);
    expect(await screen.findByRole('heading',{name:'Root SQL'})).toBeTruthy();
    expect(api.createVersion).not.toHaveBeenCalled();
    await user.type(screen.getByRole('textbox',{name:'New reader SQL source'}),'SELECT id FROM customers');
    await user.click(screen.getByRole('button',{name:'Create reader'}));
    expect((await screen.findByRole('alert')).textContent).toContain('Invalid root SQL');
    expect(screen.getByRole('textbox',{name:'New reader SQL source'}).value).toBe('SELECT id FROM customers');
    await user.click(screen.getByRole('button',{name:'Create reader'}));
    expect(await screen.findByRole('heading',{name:'Component graph'})).toBeTruthy();
    expect(api.createVersion).toHaveBeenCalledTimes(1);
    expect(api.applyReaderCommand).toHaveBeenLastCalledWith('new-reader',1,{
      expectedSourceRevision:1,
      operation:{type:'createReader',reader:{name:'record',route:'/v1/studio/readers/new-reader',sql:'SELECT id FROM customers'}},
    });
    expect(screen.getByRole('tab',{name:'Inputs'})).toBeTruthy();
    expect(screen.getByRole('tab',{name:'Outputs'})).toBeTruthy();
  });

  test('opens root SQL for an existing empty draft without creating another version', async () => {
    const user = userEvent.setup();
    const version = {versionNo:2,sourceRevision:3,state:'draft'};
    const api = {
      listVersions: vi.fn().mockResolvedValue({items:[version]}),
      inspectVersion: vi.fn().mockResolvedValueOnce({version,dql:'',structure:{},capabilities:{canEdit:true}}).mockResolvedValue(readerFixture()),
      createVersion: vi.fn(),
      applyReaderCommand: vi.fn().mockResolvedValue({applied:true,inspection:readerFixture()}),
    };
    render(<ReaderBuilder api={api} report={{id:'empty',title:'Empty'}} onBack={vi.fn()}/>);
    await screen.findByRole('heading',{name:'Root SQL'});
    await user.type(screen.getByRole('textbox',{name:'New reader SQL source'}),'SELECT 1 AS id');
    await user.click(screen.getByRole('button',{name:'Create reader'}));
    await screen.findByRole('heading',{name:'Component graph'});
    expect(api.createVersion).not.toHaveBeenCalled();
    expect(api.applyReaderCommand).toHaveBeenCalledWith('empty',2,expect.objectContaining({expectedSourceRevision:3}));
  });

  test('guards the first root query when cancelling or returning to Components', async () => {
    const user = userEvent.setup();
    const onBack = vi.fn();
    render(<ReaderBuilder api={{listVersions:vi.fn().mockResolvedValue({items:[]})}} report={{id:'scratch',title:'Scratch'}} onBack={onBack}/>);
    await screen.findByRole('heading',{name:'Root SQL'});
    await user.type(screen.getByRole('textbox',{name:'New reader SQL source'}),'SELECT 1 AS id');
    await user.click(screen.getByRole('button',{name:'Cancel'}));
    expect(await screen.findByRole('button',{name:'Keep editing'})).toBeTruthy();
    await user.click(screen.getByRole('button',{name:'Keep editing'}));
    expect(screen.getByRole('textbox',{name:'New reader SQL source'}).value).toBe('SELECT 1 AS id');
    await user.click(screen.getByRole('button',{name:'Components'}));
    expect(onBack).not.toHaveBeenCalled();
    await user.click(screen.getByRole('button',{name:'Discard changes'}));
    expect(onBack).toHaveBeenCalledOnce();
  });

  test('opens output settings directly and keeps the catalog after removing a response field', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    inspection.structure.component.rootView.relations = [];
    const hidden = structuredClone(inspection);
    hidden.structure.columnContracts = [{view:'vendor',column:'ID',tags:{json:'-'}}];
    const api = {
      listVersions:vi.fn().mockResolvedValue({items:[inspection.version]}),
      inspectVersion:vi.fn().mockResolvedValueOnce(inspection).mockResolvedValue(hidden),
      getTable:vi.fn().mockResolvedValue({columns:[]}),
      applyReaderCommand:vi.fn().mockResolvedValue({applied:true,inspection:hidden}),
    };
    render(<ReaderBuilder api={api} report={{id:'vendor',title:'Vendor',defaultConnectorName:'main'}} onBack={vi.fn()}/>);
    await screen.findByRole('heading',{name:'Component graph'});
    await user.click(screen.getByRole('tab',{name:'Outputs'}));
    await user.click(screen.getByRole('tab',{name:'Result columns'}));
    await user.click(screen.getByRole('button',{name:/ID ID int vendor Settings/}));
    expect(screen.getByLabelText('Public field name')).toBeTruthy();
    await user.click(screen.getByRole('button',{name:'Remove from output'}));
    await waitFor(()=>expect(screen.queryByRole('dialog')).toBeNull());
    expect(screen.getByRole('heading',{name:'Output columns'})).toBeTruthy();
    expect(screen.getByText('Hidden from response')).toBeTruthy();
    expect(api.applyReaderCommand).toHaveBeenCalledWith('vendor',1,expect.objectContaining({operation:expect.objectContaining({type:'setColumnContract',column:expect.objectContaining({view:'vendor',column:'ID',tags:expect.objectContaining({json:'-'})})})}));
  });

  test('keeps the root reachable and pages 30 child views with searchable relation context', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    inspection.structure.component.rootView.relations = Array.from({ length: 30 }, (_, index) => ({
      name: `child${index + 1}`, kind: 'subview', on: [{ parentColumn: 'ID', childColumn: 'VENDOR_ID' }],
      view: { name: `child${index + 1}`, namespace: `child${index + 1}`, source: { table: `TABLE_${index + 1}` }, columns: [], relations: [] },
    }));
    const api = { listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }), inspectVersion: vi.fn().mockResolvedValue(inspection), getTable: vi.fn().mockResolvedValue({columns:[]}) };
    render(<ReaderBuilder api={api} report={{ id: 'vendor', title: 'Vendor Catalog', defaultConnectorName: 'main' }} onBack={vi.fn()} />);
    await screen.findByRole('heading', { name: 'Component graph' });
    expect(screen.getByRole('button', { name: 'Actions for Vendor' })).toBeTruthy();
    expect(screen.getAllByRole('button', { name: /^Open child view/ })).toHaveLength(10);
    await user.click(screen.getByRole('button', { name: 'Next child views' }));
    expect(screen.getByRole('button', { name: 'Open child view Vendor / Child11' })).toBeTruthy();
    await user.type(screen.getByRole('textbox', { name: 'Find graph view' }), 'TABLE_30');
    expect(screen.getAllByRole('button', { name: /^Open child view/ })).toHaveLength(1);
    await user.click(screen.getByRole('button', { name: 'Open relation Vendor / Child30' }));
    expect(await screen.findByRole('button', { name: 'Edit relation' })).toBeTruthy();
    expect(screen.getByText('ID → VENDOR_ID')).toBeTruthy();
  });

  test('opens cube fields before preview and sends selected fields to the cube executor', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    inspection.structure.component.settings = {report:{enabled:true}};
    inspection.structure.component.rootView.columns = [{name:'Country',groupable:true},{name:'Avails',groupable:false}];
    const api = {listVersions:vi.fn().mockResolvedValue({items:[inspection.version]}),inspectVersion:vi.fn().mockResolvedValue(inspection),previewReader:vi.fn().mockResolvedValue({data:[{avails:3}],duration:1})};
    render(<ReaderBuilder api={api} report={{id:'vendor',title:'Vendor Catalog',defaultConnectorName:'main'}} onBack={vi.fn()}/>);
    await screen.findByRole('heading',{name:'Component graph'});
    expect(screen.queryByRole('region',{name:'Cube preview selection'})).toBeNull();
    const settingsTrigger = screen.getByRole('button',{name:'Run preview',exact:true});
    settingsTrigger.focus();
    await user.keyboard('{Enter}');
    expect(screen.getByRole('region',{name:'Cube preview selection'})).toBeTruthy();
    expect(screen.queryByRole('heading',{name:'Component graph'})).toBeNull();
    screen.getByRole('button',{name:'Close cube preview selection'}).focus();
    await user.keyboard('{Enter}');
    expect(document.activeElement).toBe(settingsTrigger);
    expect(screen.queryByRole('region',{name:'Cube preview selection'})).toBeNull();
    await user.click(screen.getByRole('button',{name:'Run preview',exact:true}));
    expect(api.previewReader).not.toHaveBeenCalled();
    await user.click(screen.getByRole('checkbox',{name:'Country'}));
    await user.click(screen.getByRole('checkbox',{name:'Avails'}));
    await user.click(screen.getByRole('button',{name:'Run cube preview'}));
    await waitFor(()=>expect(api.previewReader).toHaveBeenCalledWith('vendor',1,{dimensions:{country:true},measures:{avails:true},filters:{}},50,'/vendors',true));
    await user.click(screen.getByRole('button',{name:'More actions'}));
    await user.click(screen.getByRole('menuitem',{name:'Refresh component'}));
    await user.click(screen.getByRole('tab',{name:'Graph'}));
    await screen.findByRole('heading',{name:'Component graph'});
    expect(screen.queryByRole('region',{name:'Cube preview selection'})).toBeNull();
  });

  test('uses the child connector for metadata and its settings label',async()=>{
    const user=userEvent.setup();const inspection=readerFixture();
    inspection.structure.component.rootView.relations[0].view.source.bindings={connector:'lookup'};
    const api={listVersions:vi.fn().mockResolvedValue({items:[inspection.version]}),inspectVersion:vi.fn().mockResolvedValue(inspection),getTable:vi.fn().mockResolvedValue({columns:[]})};
    render(<ReaderBuilder api={api} report={{id:'vendor',title:'Vendors',defaultConnectorName:'main'}} onBack={vi.fn()}/>);
    await screen.findByRole('heading',{name:'Component graph'});
    await user.click(screen.getByRole('button',{name:'Products',exact:true}));
    expect(await screen.findByText('lookup',{exact:true})).toBeTruthy();
    await waitFor(()=>expect(api.getTable).toHaveBeenCalledWith('lookup',expect.objectContaining({table:'PRODUCT'})));
  });

  test('keeps an aliased joined view limited to its projected contract', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    const child = inspection.structure.component.rootView.relations[0].view;
    child.columns = [{ name: 'ID', source: 'ID', type: { name: 'int' } }, { name: 'CHANNEL', source: 'c.NAME', type: { name: 'string' } }, { name: 'CHANNEL_LABEL', source: 'c.NAME', type: { name: 'string' } }];
    inspection.structure.views[1] = { name: 'products', sql: 'SELECT ID, NAME AS CHANNEL, NAME AS CHANNEL_LABEL FROM PRODUCT c', sourceProjectionAll: false };
    const api = { listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }), inspectVersion: vi.fn().mockResolvedValue(inspection), getTable: vi.fn().mockResolvedValue({ columns: [{ name: 'ID', type: 'int', primaryKey: true }, { name: 'NAME', type: 'varchar' }, { name: 'UNSELECTED', type: 'varchar' }] }) };
    render(<ReaderBuilder api={api} report={{ id: 'vendor', title: 'Vendors', defaultConnectorName: 'main' }} onBack={vi.fn()} />);
    await screen.findByRole('heading', { name: 'Component graph' });
    await user.click(screen.getByRole('button', { name: 'Products', exact: true }));
    await waitFor(() => expect(screen.getByRole('heading', { name: 'Columns' }).parentElement.textContent).toContain('3 available'));
    await waitFor(() => expect(api.getTable).toHaveBeenCalled());
    expect(screen.getByRole('button', { name: /CHANNEL c.NAME string field Edit/ })).toBeTruthy();
    expect(screen.getByRole('button', { name: /CHANNEL_LABEL c.NAME string field Edit/ })).toBeTruthy();
    expect(screen.queryByRole('button', { name: /UNSELECTED/ })).toBeNull();
    expect(screen.queryByRole('button', { name: /^NAME NAME/ })).toBeNull();
  });

  test('keeps a published reader immutable and creates its edit draft', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    inspection.version = { ...inspection.version, state: 'published' };
    const api = {
      listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }),
      inspectVersion: vi.fn().mockResolvedValue(inspection),
      getVersion: vi.fn().mockResolvedValue({ sourceRevision: 7, authoringMode: 'dql', authoredDql: 'SELECT 1' }),
      getResources: vi.fn().mockResolvedValue({ files: [], folders: [], skills: [] }),
      cloneVersion: vi.fn().mockResolvedValue({ versionNo: 8, sourceRevision: 1 }),
    };
    const onReportUpdated = vi.fn();
    render(<ReaderBuilder api={api} report={{ id: 'vendor', title: 'Vendor Catalog', namespace: 'general', defaultConnectorName: 'main' }} onReportUpdated={onReportUpdated} onBack={vi.fn()} />);
    await screen.findByRole('heading', { name: 'Component graph' });
    expect(screen.queryByRole('button', { name: 'Edit component' })).toBeNull();
    await user.click(screen.getByRole('tab', {name:'Inputs',exact:true}));
    await user.click(screen.getByRole('tab', {name:/Predicates/}));
    expect(screen.queryByRole('button', { name: 'Add predicate' })).toBeNull();
    await user.click(screen.getByRole('button', { name: 'Create editable draft' }));
    await waitFor(() => expect(onReportUpdated).toHaveBeenCalledWith(expect.objectContaining({ id: 'vendor', versionNo: 8 })));
  });

  test('surfaces a server-owned authorization predicate in the large input catalog', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    inspection.structure.declarations = [{
      parameter: { name: 'AuthorizedPublisherIDs', source: { kind: 'param', name: 'Auth.Scope.IDs' }, typeExpr: '[]int' },
      predicates: [{ ordinal: 0, predicate: { name: 'handler', group: 99, args: ['scope.PublisherScope'] } }],
    }];
    const api = { listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }), inspectVersion: vi.fn().mockResolvedValue(inspection) };
    render(<ReaderBuilder api={api} report={{ id: 'vendor', title: 'Vendor Catalog', namespace: 'general', defaultConnectorName: 'main' }} onBack={vi.fn()} />);
    await screen.findByRole('heading', { name: 'Component graph' });
    await user.click(screen.getByRole('tab', {name:'Inputs',exact:true}));
    await user.click(screen.getByRole('tab', {name:/Predicates/}));
    expect(await screen.findByText('Authorization')).toBeTruthy();
    await user.type(screen.getByRole('textbox', { name: 'Search predicates' }), 'authorization');
    expect(screen.getByRole('button', { name: /AuthorizedPublisherIDs/ })).toBeTruthy();
  });

  test('sends scoped preview to the server and surfaces an authorization denial', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    inspection.structure.declarations = [{ parameter: { name: 'AuthorizedPublisherIDs', source: { kind: 'param', name: 'Auth.Scope.IDs' }, typeExpr: '[]int' } }];
    const api = { listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }), inspectVersion: vi.fn().mockResolvedValue(inspection), previewReader: vi.fn().mockRejectedValue(new Error('Verified publisher scope is unavailable')) };
    render(<ReaderBuilder api={api} report={{ id: 'vendor', title: 'Vendor Catalog', defaultConnectorName: 'main' }} onBack={vi.fn()}/>);
    await screen.findByRole('heading', { name: 'Component graph' });
    expect(screen.getByRole('button', { name: 'Run preview', exact: true }).disabled).toBe(false);
    await user.click(screen.getByRole('button', { name: 'Run preview', exact: true }));
    await waitFor(() => expect(api.previewReader).toHaveBeenCalledWith('vendor', 1, {}, 50, '/vendors', false));
    expect(await screen.findByText('Verified publisher scope is unavailable')).toBeTruthy();
    expect(screen.queryByText('UI preview is unavailable: this reader requires a verified runtime entity scope.')).toBeNull();
  });

  test('keeps scoped preview disabled when the server denies execute permission', async () => {
    const inspection = readerFixture({ canEdit: true, canRun: false, canPublish: true, canUseDql: true });
    inspection.structure.declarations = [{ parameter: { name: 'Auth', source: { kind: 'component', name: 'GET:/_studio/access/context/vendor' } } }];
    const api = { listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }), inspectVersion: vi.fn().mockResolvedValue(inspection), previewReader: vi.fn() };
    render(<ReaderBuilder api={api} report={{ id: 'vendor', title: 'Vendor Catalog', defaultConnectorName: 'main' }} onBack={vi.fn()}/>);
    await screen.findByRole('heading', { name: 'Component graph' });
    expect(screen.getByRole('button', { name: 'Run preview', exact: true }).disabled).toBe(true);
    expect(api.previewReader).not.toHaveBeenCalled();
  });

  test('stays on the published version when an older draft remains', async () => {
    const user = userEvent.setup();
    const current = readerFixture();
    current.version = {...current.version,versionNo:13,state:'draft'};
    const old = {...current.version,versionNo:12};
    const outcome = {activeVersionNo:13,activeGeneration:28,status:'active'};
    let published = false;
    const api = {
      listVersions:vi.fn().mockImplementation(async()=>({items:published?[{...current.version,state:'published'},old]:[current.version,old]})),
      inspectVersion:vi.fn().mockImplementation(async(_,no)=>({...current,version:no===13?{...current.version,state:published?'published':'draft'}:old})),
      getPublication:vi.fn().mockResolvedValue({activeVersionNo:11,activeGeneration:25,status:'active'}),
      listPublicationEvents:vi.fn().mockResolvedValue({items:[]}),
      getRuntimeStatus:vi.fn().mockResolvedValue({status:'active',host:{status:'ready'}}),
      publishReader:vi.fn().mockImplementation(async()=>{published=true;return outcome;}),
    };
    render(<ReaderBuilder api={api} report={{id:'vendor',title:'Vendor Catalog',defaultConnectorName:'main'}} onBack={vi.fn()}/>);
    await screen.findByRole('heading',{name:'Component graph'});
    await user.click(screen.getByRole('button',{name:'Release',exact:true}));
    await waitFor(()=>expect(screen.getByRole('button',{name:'Publish selected version'}).disabled).toBe(false));
    await user.click(screen.getByRole('button',{name:'Publish selected version'}));
    await screen.findByText('Published version is now v13 · generation 28.');
    expect(api.inspectVersion.mock.calls.at(-1)).toEqual(['vendor',13,{discoverColumns:false}]);
    expect(screen.getByRole('combobox',{name:'Publish version'}).value).toBe('13');
    expect(screen.getByText('Published v13')).toBeTruthy();
  });

  test('keeps rollback and unpublish reachable while the current draft is invalid', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    inspection.version = { ...inspection.version, versionNo: 2, compileStatus: 'invalid' };
    const api = {
      listVersions: vi.fn().mockResolvedValue({ items: [inspection.version, { versionNo: 1, sourceRevision: 1, compileStatus: 'valid' }] }),
      inspectVersion: vi.fn().mockResolvedValue(inspection),
      getPublication: vi.fn().mockResolvedValue({ activeVersionNo: 1, activeGeneration: 4, desiredGeneration: 4, status: 'active' }),
      listPublicationEvents: vi.fn().mockResolvedValue({ items: [] }),
      getRuntimeStatus: vi.fn().mockResolvedValue({ status: 'active', activeGeneration: 4, reportCount: 1 }),
    };
    render(<ReaderBuilder api={api} report={{ id: 'vendor', title: 'Vendor Catalog', namespace: 'general', defaultConnectorName: 'main' }} onBack={vi.fn()} />);
    await screen.findByRole('heading', { name: 'Component graph' });
    const release = screen.getByRole('button', { name: 'Release' });
    expect(release.disabled).toBe(false);
    await user.click(release);
    expect(await screen.findByRole('dialog', { name: 'Release reader' })).toBeTruthy();
    expect(await screen.findByRole('button', { name: 'Review unpublish' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Publish selected version' }).disabled).toBe(true);
  });

  test('opens preview without stacking workspaces and does not steal navigation on completion', async () => {
    const user=userEvent.setup();
    const inspection=readerFixture();
    let finish;
    const api={listVersions:vi.fn().mockResolvedValue({items:[inspection.version]}),inspectVersion:vi.fn().mockResolvedValue(inspection),previewReader:vi.fn(()=>new Promise(resolve=>{finish=resolve;}))};
    render(<ReaderBuilder api={api} report={{id:'vendor',title:'Vendor'}} onBack={vi.fn()}/>);
    await screen.findByRole('heading',{name:'Component graph'});
    await user.click(screen.getByRole('tab',{name:'Preview',exact:true}));
    expect(api.previewReader).not.toHaveBeenCalled();
    expect(screen.queryByRole('heading',{name:'Component graph'})).toBeNull();
    await user.click(screen.getByRole('button',{name:'Run preview',exact:true}));
    expect(screen.getByRole('status').textContent).toContain('Running preview');
    await user.click(screen.getByRole('tab',{name:'Inputs',exact:true}));
    await user.click(screen.getByRole('button',{name:'Add input'}));
    await user.type(screen.getByLabelText('Input name'),'Search');
    await act(async()=>finish({duration:1,data:{records:[{id:1}]},evidence:{}}));
    expect(screen.getByRole('tab',{name:'Inputs',exact:true}).getAttribute('aria-selected')).toBe('true');
    await user.click(screen.getByRole('tab',{name:'Preview',exact:true}));
    expect(await screen.findByRole('heading',{name:'Reader preview'})).toBeTruthy();
    expect(screen.queryByRole('heading',{name:'Add request input'})).toBeNull();
    await user.click(screen.getByRole('tab',{name:'Inputs',exact:true}));
    expect(screen.getByLabelText('Input name').value).toBe('Search');
  });

  test('clears a prior preview when the next run is denied', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    const api = {
      listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }),
      inspectVersion: vi.fn().mockResolvedValue(inspection),
      previewReader: vi.fn().mockResolvedValueOnce({ duration: 1, data: {}, evidence: {} }).mockRejectedValueOnce(new Error('Scoped preview denied')),
    };
    render(<ReaderBuilder api={api} report={{ id: 'vendor', title: 'Vendor Catalog', namespace: 'general', defaultConnectorName: 'main' }} onBack={vi.fn()} />);
    await screen.findByRole('heading', { name: 'Component graph' });
    await user.click(screen.getByRole('button', { name: 'Run preview' }));
    expect(await screen.findByRole('heading', { name: 'Reader preview' })).toBeTruthy();
    await user.click(screen.getByRole('button', { name: 'Run preview' }));
    await waitFor(() => expect(screen.getByText('Scoped preview denied')).toBeTruthy());
    expect(screen.queryByRole('heading', { name: 'Reader preview' })).toBeNull();
  });

  test('validates the inspected source revision', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    const api = {
      listVersions: vi.fn().mockResolvedValue({items:[inspection.version]}),
      inspectVersion: vi.fn().mockResolvedValue(inspection),
      validateVersion: vi.fn().mockResolvedValue({valid:true,version:{...inspection.version,compileStatus:'valid'}}),
    };
    render(<ReaderBuilder api={api} report={{id:'vendor',title:'Vendor Catalog',namespace:'general',defaultConnectorName:'main'}} onBack={vi.fn()}/>);
    await screen.findByRole('heading',{name:'Component graph'});
    await user.click(screen.getByRole('button',{name:'Validate'}));
    await user.click(screen.getByRole('button',{name:'Validate revision'}));
    expect(api.validateVersion).toHaveBeenCalledWith('vendor',inspection.version.versionNo,inspection.version.sourceRevision);
  });

  test('selects a view inline, opens SQL explicitly, and guards unsaved navigation', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    const api = {
      listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }),
      inspectVersion: vi.fn().mockResolvedValue(inspection),
      getConnector: vi.fn().mockResolvedValue({ name: 'main', driver: 'sqlite' }),
      testReaderView: vi.fn().mockResolvedValue({ view: 'vendor', duration: 1, data: [], evidence: { versionNo: 1, sourceRevision: 4 } }),
      getTable: vi.fn().mockResolvedValue({ columns: [
        { name: 'ID', type: 'INTEGER', primaryKey: true, nullable: false },
        { name: 'NAME', type: 'TEXT', nullable: false },
      ] }),
    };
    render(<ReaderBuilder api={api} report={{ id: 'vendor', title: 'Vendor Catalog', namespace: 'general', defaultConnectorName: 'main' }} onBack={vi.fn()} />);

    const graph = await screen.findByRole('heading', { name: 'Component graph' });
    expect(screen.getByRole('tab', { name: 'Graph' }).getAttribute('aria-controls')).toBe('reader-panel-graph');
    expect(screen.getByRole('tabpanel').getAttribute('aria-labelledby')).toBe('reader-tab-graph');
    const rootNode = graph.closest('.studio-graph-panel').querySelector('.studio-view-select');
    await user.click(rootNode);

    expect(await screen.findByText('Root view')).toBeTruthy();
    expect(screen.getByRole('toolbar', { name: 'Selected view actions' })).toBeTruthy();
    expect(screen.getByText('2 available')).toBeTruthy();
    expect(screen.queryByRole('heading', { name: 'Vendor SQL' })).toBeNull();

    await user.click(screen.getByRole('button', { name: 'Open SQL' }));
    expect(await screen.findByRole('heading', { name: /Vendor SQL/ })).toBeTruthy();
    expect(screen.getByText('Inline view SQL')).toBeTruthy();
    const editor = screen.getByRole('textbox', { name: 'Vendor SQL source' });
    expect(editor.value).toBe('SELECT * FROM VENDOR');
    await user.click(screen.getByRole('button', { name: 'Test view' }));
    expect(api.testReaderView).toHaveBeenCalledWith('vendor', 1, 'vendor');
    await user.click(screen.getByRole('tab',{name:'SQL',exact:true}));
    await user.clear(editor);
    await user.type(editor, 'SELECT ID FROM VENDOR');
    expect(screen.getByText('Unsaved changes')).toBeTruthy();

    await user.click(screen.getByRole('tab', { name: 'Graph' }));
    expect(screen.queryByRole('alertdialog')).toBeNull();
    await user.click(screen.getByRole('tab',{name:'SQL',exact:true}));
    expect(screen.getByRole('textbox',{name:'Vendor SQL source'}).value).toBe('SELECT ID FROM VENDOR');
    await user.click(screen.getByRole('button',{name:'Close Vendor SQL tab'}));
    expect(await screen.findByText(/This SQL tab has unsaved changes/)).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Keep editing' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Discard changes' })).toBeTruthy();
    await user.click(screen.getByRole('button', { name: 'Keep editing' }));
    expect(screen.getByRole('heading', { name: /Vendor SQL/ })).toBeTruthy();
  });

  test('reuses one SQL tab for the runtime view name and its DQL namespace', async () => {
    const user=userEvent.setup();const inspection=readerFixture();
    const api={listVersions:vi.fn().mockResolvedValue({items:[inspection.version]}),inspectVersion:vi.fn().mockResolvedValue(inspection),getTable:vi.fn().mockResolvedValue({columns:[]})};
    render(<ReaderBuilder api={api} report={{id:'vendor',title:'Vendor',defaultConnectorName:'main'}} onBack={vi.fn()}/>);
    await screen.findByRole('heading',{name:'Component graph'});
    await user.click(screen.getByRole('tab',{name:'SQL',exact:true}));
    await user.type(screen.getByRole('textbox',{name:'Vendor SQL source'}),' -- keep this draft');
    await user.click(screen.getByRole('tab',{name:'Graph'}));
    await user.click(screen.getByRole('heading',{name:'Component graph'}).closest('.studio-graph-panel').querySelector('.studio-view-select'));
    await user.click(screen.getByRole('button',{name:'Open SQL'}));
    await user.selectOptions(screen.getByLabelText('Open view SQL'),'vendor');
    expect(screen.getAllByRole('tab',{name:'Vendor SQL'})).toHaveLength(1);
    expect(screen.getByRole('textbox',{name:'Vendor SQL source'}).value).toBe('SELECT * FROM VENDOR -- keep this draft');
  });

  test('uses authored view SQL ahead of a wrapped legacy compiled source', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    inspection.structure.component.rootView.source.sql = 'SELECT * FROM (SELECT * FROM VENDOR t) vendor';
    inspection.structure.views[0].sql = 'SELECT * FROM VENDOR t\n${predicate.Builder().Build("WHERE")}';
    const api = {
      listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }),
      inspectVersion: vi.fn().mockResolvedValue(inspection),
      getTable: vi.fn().mockResolvedValue({ columns: [] }),
    };
    render(<ReaderBuilder api={api} report={{ id: 'vendor', title: 'Vendor Catalog', namespace: 'general', defaultConnectorName: 'main' }} onBack={vi.fn()} />);
    const graph = await screen.findByRole('heading', { name: 'Component graph' });
    await user.click(graph.closest('.studio-graph-panel').querySelector('.studio-view-select'));
    await user.click(screen.getByRole('button', { name: 'Open SQL' }));
    expect((await screen.findByRole('textbox', { name: 'Vendor SQL source' })).value).toBe(inspection.structure.views[0].sql);
  });

  test('enforces a read-only capability projection in the rendered workspace', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture({ canEdit: false, canRun: false, canPublish: false, canUseDql: false });
    const api = {
      listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }),
      inspectVersion: vi.fn().mockResolvedValue(inspection),
      getTable: vi.fn().mockResolvedValue({ columns: [] }),
    };
    render(<ReaderBuilder api={api} report={{ id: 'vendor', title: 'Vendor Catalog', namespace: 'general', defaultConnectorName: 'main' }} onBack={vi.fn()} />);

    expect((await screen.findByRole('button', { name: 'Edit component' })).disabled).toBe(true);
    expect(screen.getByRole('tab', { name: 'Inputs' }).disabled).toBe(false);
    await user.click(screen.getByRole('button',{name:'More actions'}));
    expect(screen.getByRole('menuitem',{name:'Composition lab'}).getAttribute('aria-disabled')).toBe('true');
    await user.keyboard('{Escape}');
    expect(screen.getByRole('button', { name: 'Release' }).disabled).toBe(true);
    expect(screen.queryByRole('tab', { name: 'Advanced component DQL' })).toBeNull();
    await user.click(screen.getByRole('tab', { name: 'Inputs' }));
    expect(await screen.findByRole('heading', { name: 'Inputs' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Add input' })).toBeNull();

    await user.click(screen.getByRole('tab',{name:'Graph'}));
    const graph = screen.getByRole('heading', { name: 'Component graph' });
    await user.click(graph.closest('.studio-graph-panel').querySelector('.studio-view-select'));
    expect((await screen.findByRole('button', { name: 'Test view' })).disabled).toBe(true);
    expect(screen.getByRole('button', { name: 'Add child' }).disabled).toBe(true);
    expect(screen.getByRole('button', { name: 'Manage columns' }).disabled).toBe(true);
  });

  test('uses graph input and output blocks with paged lower catalogs and inline settings', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    inspection.structure.declarations = Array.from({ length: 205 }, (_, index) => ({
      parameter: { name: `Input${index}`, typeExpr: 'string', source: { kind: 'query', name: `input_${index}` } },
      predicates: [{ ordinal: 0, predicate: { group: 1, name: 'equal', args: ['v', `field_${index}`] } }],
    }));
    inspection.structure.predicateExpansions = [{ group: 1, view: 'vendor', operator: 'AND' }];
    const api = { listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }), inspectVersion: vi.fn().mockResolvedValue(inspection), getTable: vi.fn().mockResolvedValue({ columns: [] }), listAuthorizationPredicates: vi.fn().mockResolvedValue({items:[]}), applyReaderCommand: vi.fn() };
    render(<ReaderBuilder api={api} report={{ id: 'vendor', title: 'Vendor Catalog', namespace: 'general', defaultConnectorName: 'main' }} onBack={vi.fn()} />);
    await screen.findByRole('heading', { name: 'Component graph' });
    await user.click(screen.getByRole('button', { name: /Input 205 inputs/ }));
    const parameterTab = screen.getByRole('tab', { name: /Parameters 205/ });
    parameterTab.focus();
    await user.keyboard('{ArrowRight}');
    const predicateTab = screen.getByRole('tab', { name: /Predicates 205/ });
    await waitFor(() => expect(document.activeElement).toBe(predicateTab));
    expect(screen.getByRole('tabpanel', { name: /Predicates 205/ }).id).toBe('reader-input-details-panel');
    await user.keyboard('{ArrowLeft}');
    await waitFor(() => expect(document.activeElement).toBe(parameterTab));
    expect(screen.getByText('1–25 of 205')).toBeTruthy();
    expect(screen.getAllByText('Input0').length).toBeGreaterThan(0);
    expect(screen.queryByRole('dialog', { name: /Inputs/ })).toBeNull();
    await user.click(screen.getByRole('button', { name: 'Next contract page' }));
    expect(screen.getByText('26–50 of 205')).toBeTruthy();
    await user.type(screen.getByRole('textbox', { name: 'Search inputs' }), 'input_204');
    await user.click(screen.getByRole('button', { name: /Input204/ }));
    expect(screen.getByRole('heading', { name: 'Edit Input204' })).toBeTruthy();
    expect(screen.getByLabelText('Input name').value).toBe('Input204');
    expect(screen.queryByRole('dialog')).toBeNull();
    await user.click(screen.getByRole('button', { name: 'Back to inputs' }));
    await user.click(screen.getByRole('tab', { name: /Predicates 205/ }));
    expect(screen.getByText('1–25 of 205')).toBeTruthy();
    await user.click(screen.getByRole('button', { name: /Input0/ }));
    expect(screen.getByRole('heading', { name: 'Edit Input0' })).toBeTruthy();
    expect(screen.queryByRole('dialog')).toBeNull();
    await user.click(screen.getByRole('button', { name: 'Back to predicates' }));
    await user.click(screen.getByRole('tab', {name:'Outputs',exact:true}));
    await user.click(screen.getByRole('tab',{name:'Result columns'}));
    expect(screen.getByRole('heading', { name: 'Output columns' })).toBeTruthy();
    expect(screen.getByRole('button', { name: /ID/ })).toBeTruthy();
  });

  test('keeps a 50-view branched component compact and browsable by lineage', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    const root = inspection.structure.component.rootView;
    root.relations = [];
    for (let branch = 1; branch <= 5; branch += 1) {
      let parent = root;
      for (let depth = 1; depth <= 10; depth += 1) {
        const index = (branch - 1) * 10 + depth;
        const view = { name: `level${index}`, namespace: depth === 1 ? `Branch${branch}` : `Level${index}`, source: { table: `TABLE_${index}` }, columns: [], relations: [] };
        parent.relations.push({ name: view.name, kind: 'subview', view });
        parent = view;
      }
    }
    const api = { listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }), inspectVersion: vi.fn().mockResolvedValue(inspection), getTable: vi.fn().mockResolvedValue({ columns: [] }) };
    render(<ReaderBuilder api={api} report={{ id: 'vendor', title: 'Vendor Catalog', namespace: 'general', defaultConnectorName: 'main' }} onBack={vi.fn()} />);
    await screen.findByRole('heading', { name: 'Component graph' });
    expect(screen.queryByRole('button', { name: 'Level50' })).toBeNull();
    await user.click(screen.getByRole('button', { name: 'Browse views' }));
    expect(screen.getAllByRole('heading', { name: 'Views' })).toHaveLength(1);
    expect(screen.getByText('1–25 of 51')).toBeTruthy();
    await user.type(screen.getByRole('textbox', { name: 'Search views' }), 'TABLE_50');
    expect(screen.getByRole('button', { name: /Open view vendor \/ Branch5 \/ Level42/ })).toBeTruthy();
    await user.click(screen.getByRole('button', { name: /Open view vendor \/ Branch5 \/ Level42/ }));
    expect(await screen.findByRole('heading', { name: 'Level50' })).toBeTruthy();
  });

  test('loads output columns for a 20-view hierarchy from connector metadata', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    const root = inspection.structure.component.rootView;
    root.columns = [];
    root.relations = [];
    const parents = { 2: 1, 3: 1, 4: 1, 5: 1, 6: 2, 7: 2, 8: 3, 9: 4, 10: 5, 11: 6, 12: 7, 13: 8, 14: 9, 15: 10, 16: 11, 17: 12, 18: 13, 19: 14, 20: 15 };
    const nodes = { 1: root };
    for (let index = 2; index <= 20; index += 1) {
      const name = `view_${String(index).padStart(2, '0')}`;
      const view = { name, namespace: name, source: { table: `STUDIO_VIEW_${String(index).padStart(2, '0')}` }, columns: [], relations: [] };
      nodes[parents[index]].relations.push({ name, kind: 'subview', view });
      nodes[index] = view;
      inspection.structure.views.push({ name, sql: `SELECT * FROM ${view.source.table}`, sourceProjectionAll: true });
    }
    const api = {
      listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }),
      inspectVersion: vi.fn().mockResolvedValue(inspection),
      getTable: vi.fn().mockResolvedValue({ columns: ['ID', 'PARENT_ID', 'NAME', 'NODE_KIND', 'LEVEL_NO'].map((name) => ({ name, type: 'varchar' })) }),
    };
    render(<ReaderBuilder api={api} report={{ id: 'hierarchy', title: 'Hierarchy 20', namespace: 'scale', defaultConnectorName: 'scale_mysql' }} onBack={vi.fn()} />);
    await screen.findByRole('heading', { name: 'Component graph' });
    await user.click(screen.getByRole('button', { name: /Output Browse columns/ }));
    await user.click(screen.getByRole('tab',{name:'Result columns'}));
    expect(await screen.findByText('1–25 of 100')).toBeTruthy();
    expect(api.getTable).toHaveBeenCalledTimes(20);
    await user.type(screen.getByRole('textbox', { name: 'Search output columns' }), 'view_20');
    expect(screen.getByText('1–5 of 5')).toBeTruthy();
    expect(screen.getAllByRole('button', { name: /view_20/ })).toHaveLength(5);
  });

  test('shows 60 physical fields with the primary key first and searchable output', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    const root = inspection.structure.component.rootView;
    root.namespace = 'wide_60';
    root.source.table = 'STUDIO_WIDE_60';
    inspection.structure.views[0] = { name: 'wide_60', sql: 'SELECT * FROM STUDIO_WIDE_60', sourceProjectionAll: true };
    root.columns = [];
    root.relations = [];
    const fields = Array.from({ length: 59 }, (_, index) => ({ name: `FIELD_${String(index + 1).padStart(2, '0')}`, type: 'varchar' }));
    fields.push({ name: 'ID', type: 'int', primaryKey: true });
    const api = {
      listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }),
      inspectVersion: vi.fn().mockResolvedValue(inspection),
      getTable: vi.fn().mockResolvedValue({ columns: fields }),
    };
    render(<ReaderBuilder api={api} report={{ id: 'wide', title: 'Wide 60 Fields', namespace: 'scale', defaultConnectorName: 'scale_mysql' }} onBack={vi.fn()} />);
    await screen.findByRole('heading', { name: 'Component graph' });
    await user.click(screen.getByRole('tab',{name:'Graph'}));
    const graph = screen.getByRole('heading', { name: 'Component graph' });
    await user.click(graph.closest('.studio-graph-panel').querySelector('.studio-view-select'));
    await waitFor(() => expect(screen.getByRole('heading', { name: 'Columns' }).parentElement.textContent).toContain('60 available'));
    expect(screen.getByRole('button', { name: /ID ID int field Edit/ })).toBeTruthy();
    await user.click(screen.getByRole('button', { name: /Output Browse columns/ }));
    await user.click(screen.getByRole('tab',{name:'Result columns'}));
    expect(await screen.findByText('1–25 of 60')).toBeTruthy();
    await user.type(screen.getByRole('textbox', { name: 'Search output columns' }), 'FIELD_59');
    expect(screen.getByText('1–1 of 1')).toBeTruthy();
  });

  test('does not present unprojected physical columns as output', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    const root = inspection.structure.component.rootView;
    root.columns = [{ name: 'ID', source: 'ID', type: { name: 'int' } }];
    root.relations = [];
    root.source.table = 'STUDIO_WIDE_60';
    inspection.structure.views[0] = { name: 'vendor', sql: 'SELECT ID FROM STUDIO_WIDE_60', sourceProjectionAll: false };
    const api = {
      listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }),
      inspectVersion: vi.fn().mockResolvedValue(inspection),
      getTable: vi.fn().mockResolvedValue({ columns: [{ name: 'ID', type: 'int', primaryKey: true }, { name: 'FIELD_59', type: 'varchar' }] }),
    };
    render(<ReaderBuilder api={api} report={{ id: 'subset', title: 'Subset', namespace: 'scale', defaultConnectorName: 'scale_mysql' }} onBack={vi.fn()} />);
    await screen.findByRole('heading', { name: 'Component graph' });
    await user.click(screen.getByRole('tab', {name:'Outputs',exact:true}));
    await user.click(screen.getByRole('tab',{name:'Result columns'}));
    expect(screen.getByRole('button', { name: /ID ID int vendor Settings/ })).toBeTruthy();
    expect(screen.queryByText('FIELD_59')).toBeNull();
    expect(api.getTable).not.toHaveBeenCalled();
  });

  test('preserves separate output aliases backed by the same physical field', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    const root = inspection.structure.component.rootView;
    root.relations = [];
    root.columns = [{ name: 'CHANNEL', source: 'v.NAME', type: { name: 'string' } }, { name: 'CHANNEL_LABEL', source: 'v.NAME', type: { name: 'string' } }];
    inspection.structure.views[0] = { name: 'vendor', sql: 'SELECT NAME AS CHANNEL, NAME AS CHANNEL_LABEL FROM VENDOR v', sourceProjectionAll: false };
    const api = { listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }), inspectVersion: vi.fn().mockResolvedValue(inspection), getTable: vi.fn().mockResolvedValue({ columns: [{ name: 'NAME', type: 'varchar' }] }) };
    render(<ReaderBuilder api={api} report={{ id: 'aliases', title: 'Aliases', defaultConnectorName: 'main' }} onBack={vi.fn()} />);
    await screen.findByRole('heading', { name: 'Component graph' });
    await user.click(screen.getByRole('button', { name: /Output 2 columns/ }));
    await user.click(screen.getByRole('tab',{name:'Result columns'}));
    expect(screen.getByRole('button', { name: /^CHANNEL v.NAME string vendor Settings$/ })).toBeTruthy();
    expect(screen.getByRole('button', { name: /^CHANNEL_LABEL v.NAME string vendor Settings$/ })).toBeTruthy();
    expect(screen.queryByRole('button', { name: /^NAME NAME/ })).toBeNull();
    expect(api.getTable).not.toHaveBeenCalled();
  });

  test('names an incomplete output source and retries metadata discovery', async () => {
    const user = userEvent.setup();
    const inspection = readerFixture();
    const root = inspection.structure.component.rootView;
    root.columns = [{ name: 'ID', source: 'ID', type: { name: 'int' } }];
    root.relations = [];
    const api = {
      listVersions: vi.fn().mockResolvedValue({ items: [inspection.version] }),
      inspectVersion: vi.fn().mockResolvedValue(inspection),
      getTable: vi.fn().mockRejectedValueOnce(new Error('offline')).mockResolvedValue({ columns: [{ name: 'ID', type: 'int', primaryKey: true }] }),
    };
    render(<ReaderBuilder api={api} report={{ id: 'retry', title: 'Retry Reader', namespace: 'scale', defaultConnectorName: 'scale_mysql' }} onBack={vi.fn()} />);
    await screen.findByRole('heading', { name: 'Component graph' });
    await user.click(screen.getByRole('tab', {name:'Outputs',exact:true}));
    await user.click(screen.getByRole('tab',{name:'Result columns'}));
    const warning = await screen.findByRole('alert');
    expect(warning.textContent).toContain('VENDOR');
    expect(warning.textContent).toContain('output count is incomplete');
    expect(screen.getByRole('tab',{name:'Outputs',exact:true}).getAttribute('aria-selected')).toBe('true');
    await user.click(screen.getByRole('button', { name: 'Retry metadata' }));
    expect(await screen.findByRole('button', { name: /ID ID int vendor Settings/ })).toBeTruthy();
    expect(api.getTable).toHaveBeenCalledTimes(2);
  });

  test('saves embedded SQL bytes without trimming its closing comment newline', async () => {
    const user=userEvent.setup();
    const inspection=readerFixture();
    inspection.structure.component.rootView.source.sql='SELECT vendor.* FROM (${embed:sql/vendor.sql}) vendor';
    inspection.structure.views[0].sql='${embed:sql/vendor.sql}';
    const file={reportId:'vendor',versionNo:inspection.version.versionNo,resourceId:'sql-id',namespace:'vendor',resourcePath:'sql/vendor.sql',content:'SELECT * FROM VENDOR\n'};
    const edited='SELECT ID FROM VENDOR\n-- comment\n';
    const api={listVersions:vi.fn().mockResolvedValue({items:[inspection.version]}),inspectVersion:vi.fn().mockResolvedValue(inspection),getTable:vi.fn().mockResolvedValue({columns:[]}),getResources:vi.fn().mockResolvedValue({files:[file]}),upsertResourceFile:vi.fn().mockResolvedValue({files:[{...file,content:edited}],version:inspection.version}),applyReaderCommand:vi.fn()};
    render(<ReaderBuilder api={api} report={{id:'vendor',title:'Vendor Catalog',namespace:'general',defaultConnectorName:'main'}} onBack={vi.fn()}/>);
    const graph=await screen.findByRole('heading',{name:'Component graph'});
    await user.click(graph.closest('.studio-graph-panel').querySelector('.studio-view-select'));
    await user.click(screen.getByRole('button',{name:'Open SQL'}));
    const editor=await screen.findByRole('textbox',{name:'Vendor SQL source'});
    await waitFor(()=>expect(editor.value).toBe(file.content));
    await user.clear(editor);await user.type(editor,edited);
    await user.click(screen.getByRole('button',{name:'Save SQL'}));
    await waitFor(()=>expect(api.upsertResourceFile).toHaveBeenCalledWith(expect.objectContaining({resourcePath:'sql/vendor.sql',content:edited,expectedSourceRevision:inspection.version.sourceRevision})));
    expect(api.applyReaderCommand).not.toHaveBeenCalled();
  });

  test('restores focus to the component heading after conflict reload', async () => {
    const user=userEvent.setup();
    const inspection=readerFixture();
    const conflict=Object.assign(new Error('version source revision does not match'),{code:'conflict'});
    const api={
      listVersions:vi.fn().mockResolvedValue({items:[inspection.version]}),
      inspectVersion:vi.fn().mockResolvedValue(inspection),
      getTable:vi.fn().mockResolvedValue({columns:[]}),
      applyReaderCommand:vi.fn().mockRejectedValue(conflict),
    };
    render(<ReaderBuilder api={api} report={{id:'vendor',title:'Vendor Catalog',namespace:'general',defaultConnectorName:'main'}} onBack={vi.fn()}/>);
    const graph=await screen.findByRole('heading',{name:'Component graph'});
    await user.click(graph.closest('.studio-graph-panel').querySelector('.studio-view-select'));
    await user.click(screen.getByRole('button',{name:'Open SQL'}));
    const editor=await screen.findByRole('textbox',{name:'Vendor SQL source'});
    await user.clear(editor);await user.type(editor,'SELECT ID FROM VENDOR');
    await user.click(screen.getByRole('button',{name:'Save SQL'}));
    expect(await screen.findByRole('dialog',{name:'Reader changed elsewhere'})).toBeTruthy();
    await user.click(screen.getByRole('button',{name:'Reload latest'}));
    const heading=await screen.findByRole('heading',{name:'Vendor Catalog'});
    await waitFor(()=>expect(document.activeElement).toBe(heading));
    expect(api.inspectVersion).toHaveBeenCalledTimes(2);
  });
  test('refreshes compiled aggregate columns after inline SQL saves', async () => {
    const user = userEvent.setup();
    const initial = readerFixture();
    delete initial.structure.component.rootView.source.uri;
    const structural = structuredClone(initial);
    structural.version.sourceRevision = 5;
    structural.structure.views[0] = { name: 'vendor', sql: 'SELECT REGION, SUM(AMOUNT) AS TOTAL FROM VENDOR GROUP BY REGION', sourceProjectionAll: false };
    structural.structure.component.rootView.columns = [];
    const resolved = structuredClone(structural);
    resolved.structure.component.rootView.columns = [
      { name: 'Region', source: 'REGION', type: {name:'string'} },
      { name: 'Total', source: 'TOTAL', type: {name:'int'} },
    ];
    const api = {
      listVersions: vi.fn().mockResolvedValue({items:[initial.version]}),
      inspectVersion: vi.fn().mockResolvedValueOnce(initial).mockResolvedValue(resolved),
      getTable: vi.fn().mockResolvedValue({columns:[{name:'ID',type:'int'}]}),
      applyReaderCommand: vi.fn().mockResolvedValue({applied:true,inspection:structural}),
    };
    render(<ReaderBuilder api={api} report={{id:'vendor',title:'Vendor Catalog',defaultConnectorName:'main'}} onBack={vi.fn()}/>);
    const graph = await screen.findByRole('heading',{name:'Component graph'});
    await user.click(graph.closest('.studio-graph-panel').querySelector('.studio-view-select'));
    await user.click(screen.getByRole('button',{name:'Open SQL'}));
    const editor = await screen.findByRole('textbox',{name:'Vendor SQL source'});
    await user.clear(editor); await user.type(editor,structural.structure.views[0].sql);
    await user.click(screen.getByRole('button',{name:'Save SQL'}));
    await waitFor(()=>expect(api.inspectVersion).toHaveBeenLastCalledWith('vendor',1,{discoverColumns:true}));
    await user.click(screen.getByRole('tab',{name:'Graph'}));
    await user.click(screen.getByRole('heading',{name:'Component graph'}).closest('.studio-graph-panel').querySelector('.studio-view-select'));
    await user.click(screen.getByRole('button',{name:'Manage columns'}));
    const dialog = within(screen.getByRole('dialog'));
    expect(dialog.getByText('Total',{selector:'strong'})).toBeTruthy();
    expect(dialog.getByText('Region',{selector:'strong'})).toBeTruthy();
    expect(dialog.queryByText('ID',{selector:'strong'})).toBeNull();
  });

});


test('keeps component editing unavailable until inspection has loaded its route', async () => {
  const inspection = readerFixture();
  let finishInspection;
  const api = {
    listVersions: vi.fn().mockResolvedValue({items: [inspection.version]}),
    inspectVersion: vi.fn(() => new Promise(resolve => { finishInspection = resolve; })),
  };
  render(<ReaderBuilder api={api} report={{id: 'vendor', title: 'Vendors'}} onBack={vi.fn()}/>);
  await waitFor(() => expect(api.inspectVersion).toHaveBeenCalled());
  expect(screen.queryByRole('button', {name: 'Edit component'})?.disabled ?? true).toBe(true);
  expect(screen.queryByRole('dialog')).toBeNull();
  await act(async () => { finishInspection(inspection); });
  expect((await screen.findByRole('button', {name: 'Edit component'})).disabled).toBe(false);
});


test('ignores an old inspection after switching components', async () => {
  const user = userEvent.setup();
  let finishFirst;
  const first = readerFixture();
  first.structure.component.routes = [{path: '/first-component'}];
  const second = readerFixture();
  second.structure.component.routes = [{path: '/second-component'}];
  const api = {
    inspectVersion: vi.fn(id => id === 'first' ? new Promise(resolve => { finishFirst = resolve; }) : Promise.resolve(second)),
    listConnectors: vi.fn().mockResolvedValue({items: []}),
  };
  const view = render(<ReaderBuilder api={api} report={{id: 'first', title: 'First', versionNo: 1}} onBack={vi.fn()}/>);
  await waitFor(() => expect(api.inspectVersion).toHaveBeenCalledWith('first', 1, {discoverColumns:false}));
  view.rerender(<ReaderBuilder api={api} report={{id: 'second', title: 'Second', versionNo: 1}} onBack={vi.fn()}/>);
  await user.click(await screen.findByRole('button', {name: 'Edit component'}));
  expect(screen.getByText('/second-component', {exact: false})).toBeTruthy();
  await act(async () => { finishFirst(first); });
  expect(screen.getByText('/second-component', {exact: false})).toBeTruthy();
  expect(screen.queryByText('/first-component', {exact: false})).toBeNull();
});
