import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReportDialog } from './ReportDialog.jsx';

describe('ReportDialog', () => {
  test('blocks creation until a governed namespace exists', async () => {
    const api = {
      listConnectors: vi.fn().mockResolvedValue({ items: [{ name: 'main', driver: 'sqlite' }] }),
      listNamespaces: vi.fn().mockResolvedValue({ items: [] }),
      createReport: vi.fn(),
    };
    render(<ReportDialog api={api} isOpen onClose={vi.fn()} onCreated={vi.fn()} />);

    expect(await screen.findByText('Create an active namespace before creating a component.')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Create component' }).disabled).toBe(true);
    expect(api.listConnectors).toHaveBeenCalledWith({ status: 'active', limit: 100 });
    expect(api.listNamespaces).toHaveBeenCalledWith({ status: 'active', limit: 100 });
  });

  test('preserves entered values and shows the server rejection', async () => {
    const user = userEvent.setup();
    const api = {
      listConnectors: vi.fn().mockResolvedValue({ items: [{ name: 'main', driver: 'sqlite' }] }),
      listNamespaces: vi.fn().mockResolvedValue({ items: [{ name: 'general', title: 'General', ownerId: 'owner' }] }),
      createReport: vi.fn().mockRejectedValue(new Error('component identity already exists')),
    };
    const onClose = vi.fn();
    const onCreated = vi.fn();
    render(<ReportDialog api={api} isOpen onClose={onClose} onCreated={onCreated} />);

    const title = await screen.findByLabelText('Component title');
    await user.type(title, 'Vendor Catalog');
    await user.selectOptions(screen.getByLabelText('Active connector'), 'main');
    await user.type(screen.getByLabelText('Description'), 'Production vendors');
    await user.click(screen.getByRole('button', { name: 'Create component' }));

    expect((await screen.findByRole('alert')).textContent).toContain('component identity already exists');
    expect(title.value).toBe('Vendor Catalog');
    expect(screen.getByLabelText('Slug').value).toBe('vendor-catalog');
    expect(screen.getByLabelText('Description').value).toBe('Production vendors');
    expect(onCreated).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });
});
