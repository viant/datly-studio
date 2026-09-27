import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
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
    expect(screen.getByRole('tab', { name: 'Component Vendor Catalog' }).getAttribute('aria-controls')).toBe('component-workspace-panel-component');
    expect(screen.getByRole('tabpanel').getAttribute('aria-labelledby')).toBe('component-workspace-tab-component');
    const rootNode = graph.closest('.studio-graph-panel').querySelector('.studio-view-select');
    await user.click(rootNode);

    expect(await screen.findByText('Root view')).toBeTruthy();
    expect(screen.getByRole('toolbar', { name: 'Selected view actions' })).toBeTruthy();
    expect(screen.getByText('2 available')).toBeTruthy();
    expect(screen.queryByRole('heading', { name: 'Vendor SQL' })).toBeNull();

    await user.click(screen.getByRole('button', { name: 'Open SQL' }));
    expect(await screen.findByRole('heading', { name: /Vendor SQL/ })).toBeTruthy();
    expect(screen.getByText('embed:sql/vendor.sql')).toBeTruthy();
    const editor = screen.getByRole('textbox', { name: 'Vendor SQL source' });
    expect(editor.value).toBe('SELECT * FROM VENDOR');
    await user.click(screen.getByRole('button', { name: 'Minimize SQL editor' }));
    expect(screen.getByLabelText('Resizable SQL pane').className).toContain('compact');
    await user.click(screen.getByRole('button', { name: 'Test view' }));
    expect(api.testReaderView).toHaveBeenCalledWith('vendor', 1, 'vendor');
    await user.clear(editor);
    await user.type(editor, 'SELECT ID FROM VENDOR');
    expect(screen.getByText('Unsaved changes')).toBeTruthy();

    await user.click(screen.getByRole('tab', { name: 'Component Vendor Catalog' }));
    expect(await screen.findByText(/This SQL tab has unsaved changes/)).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Keep editing' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Discard changes' })).toBeTruthy();
    await user.click(screen.getByRole('button', { name: 'Keep editing' }));
    expect(screen.getByRole('heading', { name: /Vendor SQL/ })).toBeTruthy();
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
    expect(screen.getByRole('button', { name: 'Inputs' }).disabled).toBe(false);
    expect(screen.getByRole('button', { name: 'Predicates' }).disabled).toBe(false);
    expect(screen.getByRole('button', { name: 'Composition lab' }).disabled).toBe(true);
    expect(screen.getByRole('button', { name: 'Publish' }).disabled).toBe(true);
    expect(screen.queryByRole('tab', { name: 'Advanced component DQL' })).toBeNull();
    await user.click(screen.getByRole('button', { name: 'Inputs' }));
    expect(await screen.findByRole('heading', { name: 'Inputs' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Add input' })).toBeNull();

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
    await user.click(screen.getByRole('button', { name: /Output 1 columns/ }));
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
    await user.click(screen.getByRole('button', { name: /Views 51 views/ }));
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
    const graph = screen.getByRole('heading', { name: 'Component graph' });
    await user.click(graph.closest('.studio-graph-panel').querySelector('.studio-view-select'));
    await waitFor(() => expect(screen.getByRole('heading', { name: 'Columns' }).parentElement.textContent).toContain('60 available'));
    expect(screen.getByRole('button', { name: /ID ID int field Edit/ })).toBeTruthy();
    await user.click(screen.getByRole('button', { name: /Output Browse columns/ }));
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
    await user.click(screen.getByRole('button', { name: /Output 1 columns/ }));
    expect(screen.getByRole('button', { name: /ID ID int vendor View/ })).toBeTruthy();
    expect(screen.queryByText('FIELD_59')).toBeNull();
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
    await user.click(screen.getByRole('button', { name: /Output 1 columns/ }));
    const warning = await screen.findByRole('alert');
    expect(warning.textContent).toContain('VENDOR');
    expect(warning.textContent).toContain('output count is incomplete');
    expect(screen.getByRole('button', { name: /Output Browse columns/ })).toBeTruthy();
    await user.click(screen.getByRole('button', { name: 'Retry metadata' }));
    expect(await screen.findByRole('button', { name: /ID ID int vendor View/ })).toBeTruthy();
    expect(api.getTable).toHaveBeenCalledTimes(2);
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
});
