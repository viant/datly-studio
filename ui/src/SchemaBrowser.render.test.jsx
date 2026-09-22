import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

vi.mock('./LazyEditor.jsx', () => ({
  LazyEditor: ({ value, onChange }) => <textarea aria-label="SQL source" value={value} onChange={(event) => onChange(event.target.value)} />,
}));
vi.mock('./SchemaRootDialog.jsx', () => ({
  SchemaRootDialog: ({ isOpen, table }) => isOpen ? <div role="dialog" aria-label="Add root view">{table?.name}</div> : null,
}));
vi.mock('./SchemaSubviewDialog.jsx', () => ({
  SchemaSubviewDialog: ({ isOpen, table }) => isOpen ? <div role="dialog" aria-label="Add subview">{table?.name}</div> : null,
}));

import { SchemaBrowser } from './SchemaBrowser.jsx';

function schemaAPI() {
  return {
    listConnectors: vi.fn().mockResolvedValue({ items: [{ name: 'main', driver: 'sqlite' }] }),
    listSchemas: vi.fn().mockResolvedValue({ items: [{ catalog: 'main', name: 'sales' }] }),
    listTables: vi.fn().mockResolvedValue({ items: [{ schema: 'sales', name: 'VENDOR', type: 'TABLE' }] }),
    getTable: vi.fn().mockResolvedValue({
      table: { schema: 'sales', name: 'VENDOR', type: 'TABLE' },
      columns: [{ name: 'ID', type: 'INTEGER', primaryKey: true, nullable: false }],
    }),
    testSQL: vi.fn().mockResolvedValue({ duration: 1_500_000, data: [{ ID: 1 }] }),
  };
}

describe('SchemaBrowser', () => {
  test('keeps discovery, SQL evidence, and root/subview actions in one workspace', async () => {
    const user = userEvent.setup();
    const api = schemaAPI();
    render(<SchemaBrowser api={api} onOpenBuilder={vi.fn()} />);

    await user.click(await screen.findByRole('treeitem', { name: /VENDOR/ }));
    const editor = await screen.findByRole('textbox', { name: 'SQL source' });
    expect(editor.value).toBe('SELECT *\nFROM sales.VENDOR');
    expect(screen.getByText('ID')).toBeTruthy();
    expect(screen.getByText('PK')).toBeTruthy();

    await user.click(screen.getByRole('button', { name: 'Test SQL' }));
    expect(await screen.findByRole('heading', { name: 'SQL test result' })).toBeTruthy();
    expect(api.testSQL).toHaveBeenCalledWith('main', { sql: 'SELECT *\nFROM sales.VENDOR', limit: 50 });

    await user.click(screen.getByRole('button', { name: 'Add as root view' }));
    expect(screen.getByRole('dialog', { name: 'Add root view' }).textContent).toContain('VENDOR');
    await user.click(screen.getByRole('button', { name: 'Add as subview' }));
    expect(screen.getByRole('dialog', { name: 'Add subview' }).textContent).toContain('VENDOR');

    await user.click(screen.getByRole('button', { name: 'Close VENDOR table' }));
    expect(await screen.findByRole('heading', { name: 'Choose a database object' })).toBeTruthy();
    expect(screen.queryByRole('heading', { name: 'SQL test result' })).toBeNull();
    expect(screen.queryByRole('dialog', { name: 'Add root view' })).toBeNull();
    expect(screen.queryByRole('dialog', { name: 'Add subview' })).toBeNull();
  });

  test('renders connector discovery failures without inventing schema data', async () => {
    const api = schemaAPI();
    api.listConnectors.mockRejectedValue(new Error('connector catalog unavailable'));
    render(<SchemaBrowser api={api} onOpenBuilder={vi.fn()} />);

    expect((await screen.findByRole('alert')).textContent).toContain('connector catalog unavailable');
    expect(screen.getByRole('heading', { name: 'Choose a database object' })).toBeTruthy();
    expect(api.listSchemas).not.toHaveBeenCalled();
    expect(api.listTables).not.toHaveBeenCalled();
  });
});
