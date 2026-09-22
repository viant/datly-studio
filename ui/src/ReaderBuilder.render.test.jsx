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
        { name: 'vendor', sql: 'SELECT * FROM VENDOR' },
        { name: 'products', sql: 'SELECT * FROM PRODUCT' },
      ],
      declarations: [], predicateExpansions: [], functions: [], columnContracts: [],
    },
  };
}

describe('ReaderBuilder graph-first authoring', () => {
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
    expect(screen.getByRole('button', { name: 'Inputs' }).disabled).toBe(true);
    expect(screen.getByRole('button', { name: 'Predicates' }).disabled).toBe(true);
    expect(screen.getByRole('button', { name: 'Composition lab' }).disabled).toBe(true);
    expect(screen.getByRole('button', { name: 'Publish' }).disabled).toBe(true);
    expect(screen.queryByRole('tab', { name: 'Advanced component DQL' })).toBeNull();

    const graph = screen.getByRole('heading', { name: 'Component graph' });
    await user.click(graph.closest('.studio-graph-panel').querySelector('.studio-view-select'));
    expect((await screen.findByRole('button', { name: 'Test view' })).disabled).toBe(true);
    expect(screen.getByRole('button', { name: 'Add child' }).disabled).toBe(true);
    expect(screen.getByRole('button', { name: 'Manage columns' }).disabled).toBe(true);
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
