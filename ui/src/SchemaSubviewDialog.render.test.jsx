import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SchemaSubviewDialog } from './SchemaSubviewDialog.jsx';

describe('SchemaSubviewDialog', () => {
  test('blocks an incomplete relation before sending a Datly command', async () => {
    const user = userEvent.setup();
    const api = {
      listReports: vi.fn().mockResolvedValue({ items: [{ id: 'reader', title: 'Vendor Catalog' }] }),
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
});
